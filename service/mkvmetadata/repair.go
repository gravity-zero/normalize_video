package mkvmetadata

import (
	"context"
	"fmt"
	"normalize_video/config"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gravity-zero/mkvgo"
	"github.com/gravity-zero/mkvgo/matroska"
	"github.com/k0kubun/pp"
)

// resyncRepair repairs a damaged Matroska cluster stream in one verified
// pass: ReindexReplace with Resync tolerates the damage (bounded forward
// scan, block-level recovery inside broken clusters, lying sizes corrected
// losslessly), CleanCut resumes video only at the next keyframe so nothing
// decodes with artifacts, and DeepVerify re-reads the result before the
// atomic swap - only a defect the repair would ADD can refuse it, defects
// the file already carried are logged with their remedy. Refused when the
// source is mostly damaged (the caller falls back to the uncapped salvage).
func resyncRepair(ctx context.Context, path string) error {
	name := filepath.Base(path)
	pp.Printf("   %s: damaged cluster stream, surgical repair...\n", name)

	opts := matroska.Options{
		Progress:   progressLogger(name, "repair"),
		Resync:     true,
		CleanCut:   true,
		DeepVerify: true,
		OnSkip: func(r matroska.DamagedRange) {
			pp.Println(fmt.Sprintf("   %s: lost %d byte(s) at offset %d (~%s-%s)",
				name, r.EndOffset-r.StartOffset, r.StartOffset,
				msToClock(r.ApproxStartMs), msToClock(r.ApproxEndMs)))
		},
		OnRepair: func(r matroska.RepairedRange) {
			pp.Println(fmt.Sprintf("   %s: cluster framing rebuilt at offset %d, %d media byte(s) saved",
				name, r.StartOffset, r.BytesKept))
		},
		OnPreexisting: func(is matroska.Issue) {
			pp.Println(fmt.Sprintf("   %s: preexisting defect (not added by the repair): %s", name, is.Message))
		},
	}
	return matroska.ReindexReplace(ctx, path, opts)
}

// describeDamageRepair summarises a completed surgical repair for the
// per-file table and the journal, from the damage map the triage took
// (identical to what the repair walk reports). A truncated tail stays named
// even after the repair: the playable prefix was kept, the missing bytes
// only exist in a fresh download.
func describeDamageRepair(dmg *matroska.SalvageReport, f *matroska.Finding) string {
	if dmg == nil {
		return "repaired (was: " + f.Detail + ")"
	}
	s := fmt.Sprintf("repaired: %d damaged range(s), %d byte(s) unrecoverable, %d region(s) rebuilt losslessly",
		len(dmg.DamagedRanges), dmg.BytesSkipped, len(dmg.RepairedRanges))
	if dmg.CleanCutBytes > 0 {
		s += fmt.Sprintf(", %d video byte(s) clean-cut to the next keyframe", dmg.CleanCutBytes)
	}
	if dmg.TruncatedTail {
		s += " - tail truncated, re-download advised"
	}
	return s
}

// retimeTracks cancels a measured A/V start desync on a Matroska file by
// shifting the audio blocks (negative shift = earlier). mkvgo picks the
// cheaper engine - 2-byte in-place patches under the crash-safe journal, or
// a verified rewrite that also rebuilds a healthy index - and DeepVerify
// checks every shifted track's first block moved by exactly the requested
// amount.
func retimeTracks(ctx context.Context, path string, shifts map[uint64]int64) error {
	name := filepath.Base(path)
	pp.Printf("   %s: cancelling the A/V start delay...\n", name)
	return matroska.RetimeTracks(ctx, path, shifts, matroska.Options{
		Progress:   progressLogger(name, "retime"),
		DeepVerify: true,
	})
}

// DiagnoseMedia triages a media file whatever its container - the first
// bytes decide the engine (Matroska: index health + damage walk when sizes
// disagree; MP4/MOV: head-only box layout + edit-list delays), never the
// extension, so a mislabeled file routes correctly. It returns the damage
// and audio-sync verdicts the per-file table shows. With retime=true a
// diagnosed audio start delay is cancelled through the container's own
// repair (Matroska: block timecodes; MP4: the moov edit list, a few bytes
// whatever the file size).
func DiagnoseMedia(ctx context.Context, path string, retime bool) (damage, audioSync string, retimed bool, err error) {
	diag, err := mkvgo.Diagnose(ctx, path)
	if err != nil {
		return "", "", false, err
	}

	damage = damageText(diag)

	if shifts := audioShiftsOf(diag); len(shifts) > 0 {
		delay := audioDelayTextOf(diag)
		switch {
		case !retime:
			audioSync = delay + " (-retime to fix)"
		default:
			name := filepath.Base(path)
			pp.Printf("   %s: cancelling the A/V start delay...\n", name)
			if rerr := mkvgo.RetimeTracks(ctx, path, shifts, matroska.Options{
				Progress: progressLogger(name, "retime"),
			}); rerr != nil {
				pp.Printf("Warning: retime failed for %s: %v\n", name, rerr)
				audioSync = delay + " (retime failed: " + rerr.Error() + ")"
			} else {
				audioSync = "retimed (was: " + delay + ")"
				retimed = true
			}
		}
	}

	return damage, audioSync, retimed, nil
}

// PlanMediaTriage is the dry-run counterpart of DiagnoseMedia: the same
// container-agnostic triage, read-only, phrased as a plan ("would retime").
func PlanMediaTriage(ctx context.Context, path string) (damage, audioSync string, err error) {
	diag, err := mkvgo.Diagnose(ctx, path)
	if err != nil {
		return "", "", err
	}
	damage = damageText(diag)
	if shifts := audioShiftsOf(diag); len(shifts) > 0 {
		if config.RETIME {
			audioSync = "would retime (" + audioDelayTextOf(diag) + ")"
		} else {
			audioSync = audioDelayTextOf(diag) + " (-retime to fix)"
		}
	}
	return damage, audioSync, nil
}

// ConvertBlocker reports why remuxing srcPath to MKV would be pointless
// ("" when nothing blocks it): a truncated source or one without a moov has
// nothing a remux could carry over cleanly - the honest outcome is a plain
// move with the re-download verdict on the table.
func ConvertBlocker(ctx context.Context, srcPath string) string {
	diag, err := mkvgo.Diagnose(ctx, srcPath)
	if err != nil {
		// let the remux itself decide: its refusal is more specific
		return ""
	}
	for _, f := range diag.Findings {
		if f.Kind == "truncated" || f.Kind == "no-moov" {
			return f.Kind + ": " + f.Detail
		}
	}
	return ""
}

// damageText renders the structural findings of a diagnosis (both
// containers) for the per-file table; "" when the structure is sound.
// Index and audio-delay findings have their own columns and repairs.
func damageText(d *matroska.Diagnosis) string {
	var parts []string
	for _, f := range d.Findings {
		switch f.Kind {
		case "truncated", "no-moov":
			parts = append(parts, f.Kind+": "+f.Detail+" - re-download advised")
		case "damaged":
			parts = append(parts, f.Kind+": "+f.Detail)
		case "trailing-junk":
			parts = append(parts, f.Kind+": "+f.Detail)
		}
	}
	return strings.Join(parts, "; ")
}

// findingOf returns the first finding of the given kind, nil when absent.
func findingOf(d *matroska.Diagnosis, kind string) *matroska.Finding {
	if d == nil {
		return nil
	}
	for i := range d.Findings {
		if d.Findings[i].Kind == kind {
			return &d.Findings[i]
		}
	}
	return nil
}

// audioShiftsOf returns the retime map cancelling every diagnosed audio
// start delay (track -> negative shift in ns), empty when A/V start together.
func audioShiftsOf(d *matroska.Diagnosis) map[uint64]int64 {
	if d == nil {
		return nil
	}
	shifts := map[uint64]int64{}
	for _, f := range d.Findings {
		if f.Kind == "audio-delay" {
			shifts[f.Track] = -f.DelayNs
		}
	}
	return shifts
}

// audioDelayTextOf describes the diagnosed start delays, deterministically
// ordered ("track 2 starts 350ms late").
func audioDelayTextOf(d *matroska.Diagnosis) string {
	var parts []string
	for _, f := range d.Findings {
		if f.Kind == "audio-delay" {
			parts = append(parts, fmt.Sprintf("track %d starts %dms late", f.Track, f.DelayNs/1_000_000))
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

// msToClock renders a millisecond timestamp as mm:ss for the damage logs.
func msToClock(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	return fmt.Sprintf("%02d:%02d", ms/60000, (ms/1000)%60)
}
