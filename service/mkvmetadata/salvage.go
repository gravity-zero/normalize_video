package mkvmetadata

import (
	"context"
	"os"
	"path/filepath"

	"github.com/gravity-zero/mkvgo/matroska"
	"github.com/k0kubun/pp"
)

// SalvageFile replaces a structurally damaged file with its best-effort
// salvaged copy: intact metadata and clusters are carried over verbatim,
// cluster framing is re-derived from the bytes inside damaged regions (lying
// sizes fixed losslessly), the Cues index is rebuilt, video resumes at the
// next keyframe after each damage gap (CleanCut: a short jump instead of
// reference-broken artifacts) and the unrecoverable spans are skipped
// (listed in the report, with TruncatedTail naming an incomplete download).
// Lossy by design - only call it once the surgical resync repair has refused
// the file. The original is only replaced after the salvage completed.
func SalvageFile(ctx context.Context, path string) (*matroska.SalvageReport, error) {
	name := filepath.Base(path)
	pp.Printf("   %s: damaged file, salvaging what is intact...\n", name)

	tmpPath := path + ".salvage.tmp"
	report, err := matroska.Salvage(ctx, path, tmpPath, matroska.Options{
		Progress: progressLogger(name, "salvage"),
		CleanCut: true,
	})
	if err != nil {
		os.Remove(tmpPath)
		return nil, err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return nil, err
	}
	return report, nil
}
