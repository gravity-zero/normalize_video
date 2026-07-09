package mkvmetadata

import (
	"context"
	"fmt"
	"strings"

	"github.com/gravity-zero/mkvgo/matroska"
)

// CheckPlayability evaluates the file against a named capability profile
// (chrome, safari, firefox, chromecast-gen3, ...) and returns a compact
// verdict: "direct-play", "remux -> mp4", or "transcode (track 1: hevc not
// supported; ...)". Head-only metadata read, no decode.
func CheckPlayability(ctx context.Context, path, targetName string) (string, error) {
	target, ok := matroska.TargetByName(targetName)
	if !ok {
		return "", fmt.Errorf("unknown playability target %q", targetName)
	}

	report, err := matroska.Playability(ctx, path, target)
	if err != nil {
		return "", err
	}

	verdict := report.OverallVerdict
	if verdict == "remux" && report.RemuxContainer != "" {
		verdict += " -> " + report.RemuxContainer
	}

	var reasons []string
	for _, tv := range report.Tracks {
		if tv.Verdict == "direct-play" {
			continue
		}
		for _, r := range tv.Reasons {
			reasons = append(reasons, fmt.Sprintf("track %d: %s", tv.TrackID, r))
		}
	}
	// keep the table readable: the first reasons tell the story
	if len(reasons) > 3 {
		reasons = append(reasons[:3], "...")
	}
	if len(reasons) > 0 {
		verdict += " (" + strings.Join(reasons, "; ") + ")"
	}

	return verdict, nil
}
