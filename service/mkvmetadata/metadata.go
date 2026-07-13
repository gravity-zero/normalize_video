package mkvmetadata

import (
	"context"
	"errors"
	"fmt"
	"normalize_video/config"
	"normalize_video/types"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gravity-zero/mkvgo/matroska"
	"github.com/k0kubun/pp"
)

// mkvAnalysis is the read-only part of the pipeline: parsed container, chosen
// tracks and the mkvgo triage (seek index health, cluster-stream damage,
// audio start delays). Shared by the real update and the dry-run plan.
type mkvAnalysis struct {
	container *matroska.Container
	bestAudio *types.Track
	bestSub   *types.Track
	diag      *matroska.Diagnosis
	seekIssue string
}

func analyzeMkv(ctx context.Context, path string) (*mkvAnalysis, error) {
	c, err := matroska.Open(ctx, path)
	if err != nil {
		if errors.Is(err, matroska.ErrNotMatroska) {
			return nil, fmt.Errorf("%s is not a real Matroska file (MP4-family container mislabeled as .mkv), left untouched: %w", filepath.Base(path), err)
		}
		return nil, err
	}

	var audioTracks, subtitleTracks []types.Track
	for _, t := range c.Tracks {
		track := mkvTrackToType(t)
		switch t.Type {
		case matroska.AudioTrack:
			audioTracks = append(audioTracks, track)
		case matroska.SubtitleTrack:
			subtitleTracks = append(subtitleTracks, track)
		}
	}

	service := NewMkvService(MkvConfig{
		PreferredAudioLang:    config.PREFERRED_AUDIO_LANG,
		FallbackAudioLang:     config.FALLBACK_AUDIO_LANG,
		PreferredSubtitleLang: config.PREFERRED_SUBTITLE_LANG,
		FallbackSubtitleLang:  config.FALLBACK_SUBTITLE_LANG,
		SubtitleForcedOnly:    config.SUBTITLE_FORCED_ONLY,
	})

	a := &mkvAnalysis{
		container: c,
		bestAudio: service.GetBestAudioTrack(audioTracks),
		bestSub:   service.GetBestSubtitleTrack(subtitleTracks),
	}

	// The index verdict comes from the container we just parsed, NOT from the
	// triage's head-only view. Both answer the same question, but we already
	// paid for the full read, and it sees what a player sees: Cues sitting at
	// the tail with no SeekHead pointing at them are found by the bounded
	// scan back from EOF, and are perfectly seekable. A head-only check that
	// only follows SeekHead pointers calls those files unindexed and has us
	// rewrite an index they already have.
	a.seekIssue = SeekIndexIssue(c)

	// The triage is what the full parse cannot give: cluster-stream damage
	// (declared-size coherence, tolerant walk when the sizes disagree) and
	// per-track A/V start delays. Every finding names its remedy, which is
	// what routes the repairs below.
	diag, derr := matroska.Diagnose(ctx, path)
	if derr != nil {
		// triage must never block the pipeline: the index verdict above still
		// stands; damage stays undetected until an operation trips on it
		pp.Printf("Warning: diagnose failed for %s, damage and A/V sync not checked: %v\n", filepath.Base(path), derr)
		return a, nil
	}
	a.diag = diag
	return a, nil
}

// cueHealthBefore is the head-only index view the triage took before anything
// wrote to the file, nil when the triage did not run.
func cueHealthBefore(a *mkvAnalysis) *matroska.CueHealthReport {
	if a.diag == nil {
		return nil
	}
	return a.diag.CueHealth
}

// finding returns the first diagnosis finding of the given kind, nil when the
// triage did not run or found nothing of that kind.
func (a *mkvAnalysis) finding(kind string) *matroska.Finding {
	return findingOf(a.diag, kind)
}

// damageFinding returns the finding that calls for a cluster-stream repair
// (a truncated source or mid-file damage), nil when the stream is intact.
func (a *mkvAnalysis) damageFinding() *matroska.Finding {
	if f := a.finding("truncated"); f != nil {
		return f
	}
	return a.finding("damaged")
}

// audioShifts returns the retime map cancelling every diagnosed audio start
// delay (track -> negative shift in ns), empty when A/V start together.
func (a *mkvAnalysis) audioShifts() map[uint64]int64 {
	return audioShiftsOf(a.diag)
}

// audioDelayText describes the diagnosed start delays, deterministically
// ordered ("track 2 starts 350ms late").
func (a *mkvAnalysis) audioDelayText() string {
	return audioDelayTextOf(a.diag)
}

// fillTrackInfo reports the chosen tracks in the FileInfos.
func (a *mkvAnalysis) fillTrackInfo(info *types.FileInfos) {
	if a.bestAudio != nil {
		if a.bestAudio.Properties.TrackName != "" {
			info.MkvAudioTrack = strings.ToLower(a.bestAudio.Properties.TrackName)
		} else {
			info.MkvAudioTrack = strings.ToLower(a.bestAudio.Properties.LanguageIetf)
		}
	}
	if a.bestSub != nil && a.bestSub.Properties.TrackName != "" {
		info.MkvSubTrack = strings.ToLower(a.bestSub.Properties.TrackName)
	}
}

func mediaInfo(m interface{}) (types.FileInfos, error) {
	var info types.FileInfos
	switch v := m.(type) {
	case *types.Serie:
		info.MkvFilePath = v.Normalizer.NewPath
		info.MkvTitle = v.Normalizer.Title + " " + v.SE
	case *types.Movie:
		info.MkvFilePath = v.Normalizer.NewPath
		info.MkvTitle = v.Normalizer.Title
	default:
		return info, errors.New("mkvmetadata: unknown media type")
	}
	return info, nil
}

func UpdateMkvMetadata(m interface{}) (types.FileInfos, error) {
	info, err := mediaInfo(m)
	if err != nil {
		return info, err
	}

	ctx := context.Background()

	a, err := analyzeMkv(ctx, info.MkvFilePath)
	if err != nil {
		return info, err
	}

	// Trailing bytes past the declared Segment end can be the crash journal
	// of an interrupted in-place repair (a previous run killed mid-reindex):
	// roll the file back to its pre-repair bytes, then look at it again.
	// Plain junk carries no journal and RecoverInPlace reports false.
	if a.finding("trailing-junk") != nil {
		if recovered, rerr := matroska.RecoverInPlace(ctx, info.MkvFilePath); rerr == nil && recovered {
			pp.Printf("   %s: interrupted in-place repair rolled back\n", filepath.Base(info.MkvFilePath))
			if fresh, aerr := analyzeMkv(ctx, info.MkvFilePath); aerr == nil {
				a = fresh
			}
		}
	}

	applyEdit := func(c *matroska.Container) {
		c.Info.Title = info.MkvTitle

		for i := range c.Tracks {
			switch c.Tracks[i].Type {
			case matroska.AudioTrack:
				c.Tracks[i].IsDefault = a.bestAudio != nil && c.Tracks[i].ID == uint64(a.bestAudio.Properties.Number)
			case matroska.SubtitleTrack:
				isBest := a.bestSub != nil && c.Tracks[i].ID == uint64(a.bestSub.Properties.Number)
				c.Tracks[i].IsDefault = isBest
				// A sub selected as forced (by name or flag) gets the real
				// FlagForced, so players honor it without reading track names
				if isBest && a.bestSub.Properties.Forced {
					c.Tracks[i].IsForced = true
				}
			}
		}
	}

	seekStatus := "ok"
	needEdit := true
	seekIssue := a.seekIssue

	// Damaged cluster stream: one verified resync rewrite fixes lying sizes
	// losslessly, cuts around the unrecoverable bytes block by block (video
	// clean-cut to the next keyframe), seals the Segment size and rebuilds
	// SeekHead + Cues in the same pass. Refusals (mostly-damaged source)
	// surface as an error so the caller's uncapped salvage takes over.
	//
	// The metadata edit runs FIRST, on the still-damaged file: its head
	// region is intact (the damage lives in the cluster stream), and the
	// rewrite then carries the edited metadata over and stays the LAST pass
	// on the file - so the index it builds is the one the file keeps.
	if f := a.damageFinding(); f != nil {
		if !config.SALVAGE {
			info.MkvDamage = f.Kind + " (-salvage to repair): " + f.Detail
		} else {
			if eerr := matroska.EditInPlace(ctx, info.MkvFilePath, applyEdit); eerr != nil {
				pp.Printf("   %s: metadata edit deferred to after the repair: %v\n", filepath.Base(info.MkvFilePath), eerr)
			} else {
				needEdit = false
			}
			if rerr := resyncRepair(ctx, info.MkvFilePath); rerr != nil {
				return info, fmt.Errorf("surgical repair refused for %s: %w", filepath.Base(info.MkvFilePath), rerr)
			}
			info.MkvDamage = describeDamageRepair(a.diag.Damage, f)
			if seekIssue != "" {
				seekStatus = "rebuilt (was: " + seekIssue + ")"
			}
			// the rewrite rebuilt the seek index, skip the dedicated repair
			seekIssue = ""
		}
	}

	if seekIssue != "" {
		if !config.REPAIR_SEEK_INDEX {
			seekStatus = seekIssue + " (repair disabled)"
		} else if err := reindexInPlace(ctx, info.MkvFilePath, seekIssue); err == nil {
			// Surgical repair: Cues appended, SeekHead repointed, cluster
			// bytes untouched - one read pass, no file copy. Crash-safe
			// (in-file journal, verified before commit)
			seekStatus = "rebuilt in place (was: " + seekIssue + ")"
		} else {
			// Fallback: full rewrite. EditMetadata rebuilds SeekHead + Cues
			// while rewriting, so metadata edit and repair share the pass
			if errors.Is(err, matroska.ErrIndexNotHeadDiscoverable) {
				// expected for some layouts, not an error: the file was
				// rolled back byte-identical and the copy path handles it
				pp.Printf("   %s: layout cannot hold a head-discoverable index, full rewrite instead\n", filepath.Base(info.MkvFilePath))
			} else {
				pp.Printf("Warning: in-place reindex failed, falling back to full rewrite: %v\n", err)
			}
			if err := fullRewrite(ctx, info.MkvFilePath, applyEdit); err != nil {
				return info, err
			}
			seekStatus = "rebuilt (was: " + seekIssue + ")"
			needEdit = false
		}
	}

	if needEdit {
		if err := matroska.EditInPlace(ctx, info.MkvFilePath, applyEdit); err != nil {
			pp.Printf("Warning: in-place edit failed, trying full rewrite: %v\n", err)
			if err := fullRewrite(ctx, info.MkvFilePath, applyEdit); err != nil {
				return info, err
			}
		} else if config.REPAIR_SEEK_INDEX && seekIndexLost(ctx, info.MkvFilePath, cueHealthBefore(a)) {
			// Post-condition, not a workaround: the edit must not cost the
			// file the index it had. mkvgo 0.21.1 refuses the edit rather
			// than overwrite the SeekHead (the branch above then rewrites),
			// so this should never fire; it costs one head-only read to be
			// sure, on an operation that reported success while corrupting
			// the file once. A head with no SeekHead left cannot grow one
			// back in place, hence the rewrite.
			const lost = "index dropped by the metadata edit"
			if err := reindexInPlace(ctx, info.MkvFilePath, "index left unreachable by the metadata edit"); err == nil {
				if !strings.HasPrefix(seekStatus, "rebuilt") {
					seekStatus = "rebuilt in place (was: " + lost + ")"
				}
			} else if err := fullRewrite(ctx, info.MkvFilePath, applyEdit); err != nil {
				return info, err
			} else if !strings.HasPrefix(seekStatus, "rebuilt") {
				seekStatus = "rebuilt (was: " + lost + ")"
			}
		}
	}
	info.MkvSeekIndex = seekStatus

	// A/V start desync, measured natively by the triage (first clusters).
	// -retime cancels it by shifting the audio blocks; otherwise the delay
	// is only reported, with the flag that would fix it.
	if shifts := a.audioShifts(); len(shifts) > 0 {
		switch {
		case !config.RETIME:
			info.MkvAudioSync = a.audioDelayText() + " (-retime to fix)"
		default:
			if rerr := retimeTracks(ctx, info.MkvFilePath, shifts); rerr != nil {
				pp.Printf("Warning: retime failed for %s: %v\n", filepath.Base(info.MkvFilePath), rerr)
				info.MkvAudioSync = a.audioDelayText() + " (retime failed: " + rerr.Error() + ")"
			} else {
				info.MkvAudioSync = "retimed (was: " + a.audioDelayText() + ")"
			}
		}
	}

	a.fillTrackInfo(&info)

	return info, nil
}

// PlanMkvMetadata is the dry-run counterpart of UpdateMkvMetadata: it analyzes
// the file at its CURRENT location (path) read-only and reports what the real
// run would do, without writing anything.
func PlanMkvMetadata(m interface{}, path string) (types.FileInfos, error) {
	info, err := mediaInfo(m)
	if err != nil {
		return info, err
	}

	a, err := analyzeMkv(context.Background(), path)
	if err != nil {
		return info, err
	}

	switch {
	case a.seekIssue == "":
		info.MkvSeekIndex = "ok"
	case config.REPAIR_SEEK_INDEX:
		info.MkvSeekIndex = "would rebuild (" + a.seekIssue + ")"
	default:
		info.MkvSeekIndex = a.seekIssue + " (repair disabled)"
	}

	if f := a.damageFinding(); f != nil {
		if config.SALVAGE {
			info.MkvDamage = "would repair (resync): " + f.Detail
			if f.Kind == "truncated" {
				info.MkvDamage += " - re-download advised"
			}
		} else {
			info.MkvDamage = f.Kind + " (-salvage to repair): " + f.Detail
		}
	}

	if shifts := a.audioShifts(); len(shifts) > 0 {
		if config.RETIME {
			info.MkvAudioSync = "would retime (" + a.audioDelayText() + ")"
		} else {
			info.MkvAudioSync = a.audioDelayText() + " (-retime to fix)"
		}
	}

	a.fillTrackInfo(&info)

	return info, nil
}

// IsNotMatroska reports whether err means the file is not a real Matroska
// container (e.g. an MP4 mislabeled as .mkv) - a case salvage/retry logic
// must leave alone.
func IsNotMatroska(err error) bool {
	return errors.Is(err, matroska.ErrNotMatroska)
}

// maxSeekGapMs is the widest hole tolerated in a file's video cue coverage.
// Past it, a seek into the hole lands more than this far from its target,
// which is no longer seeking - and a reindex fixes it, writing one cue per
// cluster holding a video keyframe (a few seconds apart on any real mux).
const maxSeekGapMs = 30_000

// SeekIndexIssue reports why a file's Cues index hurts seeking (evey scrubbing,
// VLC arrow keys, HLS segmenting): "" when healthy, otherwise a short reason.
// Cheap: works from the already-parsed metadata, no cluster scan.
//
// The verdict is the VIDEO index's own ability to serve a seek. A cue keyed on
// an audio track is inert - the keyframe index a player seeks with is built
// from the video-keyed cues and drops the rest - and real muxers routinely cue
// every track, so condemning a file over a single audio cue (which this used
// to do, and mkvgo with it until 0.22) rewrites the index of files whose video
// coverage is dense and perfectly seekable.
func SeekIndexIssue(c *matroska.Container) string {
	videoIDs := map[uint64]bool{}
	knownIDs := map[uint64]bool{}
	for _, t := range c.Tracks {
		knownIDs[t.ID] = true
		if t.Type == matroska.VideoTrack {
			videoIDs[t.ID] = true
		}
	}
	// an audio-only file legitimately cues audio
	if len(videoIDs) == 0 {
		return ""
	}

	if len(c.Cues) == 0 {
		return "missing Cues"
	}

	var videoTimes []int64
	for _, cue := range c.Cues {
		switch {
		case videoIDs[cue.Track]:
			videoTimes = append(videoTimes, cue.TimeMs)
		case !knownIDs[cue.Track]:
			// a cue pointing at a track the file does not have: a stale or
			// foreign index, whatever the rest of it looks like
			return "cues referencing stale tracks"
		}
	}

	if len(videoTimes) == 0 {
		return "cues keyed on non-video track"
	}

	if gap := maxVideoGapMs(videoTimes, c.DurationMs); gap > maxSeekGapMs {
		return fmt.Sprintf("video cues leave a %ds hole", gap/1000)
	}

	return ""
}

// maxVideoGapMs is the widest hole in the video cue coverage - the worst
// distance a seek can land from its target - measured between consecutive
// video cues, and from 0 to the first and from the last to the duration when
// it is known (so a half-indexed file is caught too).
func maxVideoGapMs(videoTimes []int64, durationMs int64) int64 {
	sorted := append([]int64(nil), videoTimes...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	max := sorted[0] // from the start of the file to the first video cue
	for i := 1; i < len(sorted); i++ {
		if gap := sorted[i] - sorted[i-1]; gap > max {
			max = gap
		}
	}
	if durationMs > 0 {
		if tail := durationMs - sorted[len(sorted)-1]; tail > max {
			max = tail
		}
	}
	return max
}

// seekIndexLost reports whether the metadata edit destroyed the seek index the
// file had: the in-place edit rewrites the head region where the SeekHead
// lives, and mkvgo <= 0.21.0 wrote the metadata straight over it when its slot
// no longer fit - Cues still in the file, nothing pointing at them, and nil
// returned. It compares the head-only view BEFORE the edit (from the triage)
// against the one after: a file whose index was never head-discoverable to
// begin with (Cues at the tail, no SeekHead) is not a file the edit broke.
func seekIndexLost(ctx context.Context, path string, before *matroska.CueHealthReport) bool {
	if before == nil || before.TotalCues == 0 {
		return false
	}
	after, err := matroska.CueHealth(ctx, path)
	if err != nil {
		return false
	}
	return after.HasVideoTrack && after.TotalCues == 0
}

// reindexInPlace patches the file itself through matroska.ReindexInPlace: the
// new Cues element is appended inside the Segment and the SeekHead repointed,
// without moving cluster bytes - one read pass, no file copy, crash-safe
// (in-file journal + verification, rollback on any failure).
func reindexInPlace(ctx context.Context, path, issue string) error {
	name := filepath.Base(path)
	pp.Printf("   %s: %s, rebuilding seek index in place...\n", name, issue)
	return matroska.ReindexInPlace(ctx, path, matroska.Options{Progress: progressLogger(name, "reindex")})
}

// fullRewrite rewrites path through EditMetadata (which also rebuilds
// SeekHead + Cues) into a sibling temp file, then swaps it in atomically.
// Progress is logged every 25% so long copies stay visible.
func fullRewrite(ctx context.Context, path string, edit func(*matroska.Container)) error {
	name := filepath.Base(path)
	pp.Printf("   %s: rewriting (metadata + seek index)...\n", name)

	tmpPath := path + ".rewrite.tmp"
	opts := matroska.Options{Progress: progressLogger(name, "rewrite")}
	if err := matroska.EditMetadata(ctx, path, tmpPath, edit, opts); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

// progressLogger returns a ProgressFunc logging "<name>: <verb> N%" every 25%.
func progressLogger(name, verb string) func(processed, total int64) {
	var lastStep int64
	return func(processed, total int64) {
		if total <= 0 {
			return
		}
		if step := processed * 4 / total; step > lastStep {
			lastStep = step
			// fmt.Sprintf first: pp.Printf colorizes its args to strings,
			// which breaks numeric verbs like %d
			pp.Println(fmt.Sprintf("   %s: %s %d%%", name, verb, step*25))
		}
	}
}

func mkvTrackToType(t matroska.Track) types.Track {
	return types.Track{
		Type: string(t.Type),
		Properties: types.TrackProperties{
			Number:       int(t.ID),
			Language:     t.Language,
			LanguageIetf: t.LanguageBCP47,
			TrackName:    t.Name,
			Forced:       t.IsForced,
		},
	}
}
