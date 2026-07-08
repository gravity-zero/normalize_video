package mkvmetadata

import (
	"context"
	"os"
	"path/filepath"

	"github.com/gravity-zero/mkvgo/matroska"
	"github.com/k0kubun/pp"
)

// WriteHashes stores each track's content SHA-256 as a CONTENT_SHA256 tag, so
// the file self-verifies later (matroska.VerifyContentHashes, or any tool
// reading the tag). One full read of the file; the tag write itself is
// in-place and instant, with a full-rewrite fallback when the metadata region
// has no room. Idempotent: existing hashes are replaced.
func WriteHashes(ctx context.Context, path string) error {
	name := filepath.Base(path)
	pp.Printf("   %s: hashing tracks (CONTENT_SHA256)...\n", name)

	opts := matroska.Options{Progress: progressLogger(name, "hash")}
	if err := matroska.WriteContentHashes(ctx, path, "", opts); err == nil {
		return nil
	}

	// No room for an in-place tag write: full rewrite into a temp + swap
	tmpPath := path + ".hash.tmp"
	if err := matroska.WriteContentHashes(ctx, path, tmpPath, opts); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}
