package mkvmetadata

import (
	"context"
	"os"
	"path/filepath"

	"github.com/gravity-zero/mkvgo/mp4"
	"github.com/k0kubun/pp"
)

// ConvertToMkv remuxes an MP4-family file (mp4, m4v, mov) to a Matroska file
// at dstPath, no re-encode, and removes the source on success. This replaces
// the plain move for convertible files: the media lands at its normalized
// destination directly as MKV.
func ConvertToMkv(ctx context.Context, srcPath, dstPath string) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), os.ModePerm); err != nil {
		return err
	}

	name := filepath.Base(dstPath)
	pp.Printf("   %s: converting to MKV (remux, no re-encode)...\n", name)

	opts := mp4.Options{Progress: progressLogger(name, "convert")}
	if err := mp4.RemuxFromMP4(ctx, srcPath, dstPath, opts); err != nil {
		os.Remove(dstPath)
		return err
	}

	return os.Remove(srcPath)
}
