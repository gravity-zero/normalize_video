package mkvmetadata

import (
	"context"
	"errors"
	"fmt"
	"normalize_video/config"
	"normalize_video/types"
	"os"
	"path/filepath"
	"strings"

	"github.com/gravity-zero/mkvgo/matroska"
	"github.com/k0kubun/pp"
)

// mkvAnalysis is the read-only part of the pipeline: parsed container, chosen
// tracks and seek index health. Shared by the real update and the dry-run plan.
type mkvAnalysis struct {
	container *matroska.Container
	bestAudio *types.Track
	bestSub   *types.Track
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

	return &mkvAnalysis{
		container: c,
		bestAudio: service.GetBestAudioTrack(audioTracks),
		bestSub:   service.GetBestSubtitleTrack(subtitleTracks),
		seekIssue: SeekIndexIssue(c),
	}, nil
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

	if a.seekIssue != "" {
		if !config.REPAIR_SEEK_INDEX {
			seekStatus = a.seekIssue + " (repair disabled)"
		} else if err := reindexInPlace(ctx, info.MkvFilePath, a.seekIssue); err == nil {
			// Surgical repair: Cues appended, SeekHead repointed, cluster
			// bytes untouched - one read pass, no file copy. Crash-safe
			// (in-file journal, verified before commit)
			seekStatus = "rebuilt in place (was: " + a.seekIssue + ")"
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
			seekStatus = "rebuilt (was: " + a.seekIssue + ")"
			needEdit = false
		}
	}

	if needEdit {
		if err := matroska.EditInPlace(ctx, info.MkvFilePath, applyEdit); err != nil {
			pp.Printf("Warning: in-place edit failed, trying full rewrite: %v\n", err)
			if err := fullRewrite(ctx, info.MkvFilePath, applyEdit); err != nil {
				return info, err
			}
		}
	}
	info.MkvSeekIndex = seekStatus

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

	a.fillTrackInfo(&info)

	return info, nil
}

// IsNotMatroska reports whether err means the file is not a real Matroska
// container (e.g. an MP4 mislabeled as .mkv) - a case salvage/retry logic
// must leave alone.
func IsNotMatroska(err error) bool {
	return errors.Is(err, matroska.ErrNotMatroska)
}

// SeekIndexIssue reports why a file's Cues index hurts seeking (evey scrubbing,
// VLC arrow keys, HLS segmenting): "" when healthy, otherwise a short reason.
// Cheap: works from the already-parsed metadata, no cluster scan.
func SeekIndexIssue(c *matroska.Container) string {
	videoIDs := map[uint64]bool{}
	for _, t := range c.Tracks {
		if t.Type == matroska.VideoTrack {
			videoIDs[t.ID] = true
		}
	}
	if len(videoIDs) == 0 {
		return ""
	}

	if len(c.Cues) == 0 {
		return "missing Cues"
	}

	for _, cue := range c.Cues {
		if !videoIDs[cue.Track] {
			return "cues keyed on non-video track"
		}
	}

	return ""
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
