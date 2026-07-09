package mkvmetadata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"github.com/gravity-zero/mkvgo/matroska"
)

// PresentationFromContainer rebuilds the mkvgo Fingerprint "Presentation"
// content-identity hash from the CONTENT_SHA256 tags WriteHashes stored -
// head-only, no media read. The recipe is mkvgo's documented one: per track
// build "type|codec|sha256hex", sort ascending, concatenate the raw 32-byte
// sums in that order, SHA-256 the concatenation. Two files with the same
// track payloads (a re-mux, a renamed duplicate) produce the same hash
// whatever their metadata, filename or track order.
//
// Returns "" when any track lacks its CONTENT_SHA256 tag (file not hashed).
func PresentationFromContainer(c *matroska.Container) string {
	hashes := map[uint64]string{}
	for _, tag := range c.Tags {
		for _, st := range tag.SimpleTags {
			if st.Name == "CONTENT_SHA256" && st.Value != "" {
				hashes[tag.TargetID] = st.Value
			}
		}
	}
	if len(hashes) == 0 || len(c.Tracks) == 0 {
		return ""
	}

	type entry struct {
		key string
		sum []byte
	}
	entries := make([]entry, 0, len(c.Tracks))
	for _, t := range c.Tracks {
		uid := t.UID
		if uid == 0 {
			uid = t.ID
		}
		hexSum, ok := hashes[uid]
		if !ok {
			return ""
		}
		raw, err := hex.DecodeString(hexSum)
		if err != nil || len(raw) != sha256.Size {
			return ""
		}
		entries = append(entries, entry{key: string(t.Type) + "|" + t.Codec + "|" + hexSum, sum: raw})
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].key < entries[j].key })
	h := sha256.New()
	for _, e := range entries {
		h.Write(e.sum)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// FingerprintFile returns the content-identity hash of an already-hashed file
// (one metadata read, clusters untouched). "" when the file carries no
// CONTENT_SHA256 tags - run WriteHashes first.
func FingerprintFile(ctx context.Context, path string) (string, error) {
	c, err := matroska.Open(ctx, path)
	if err != nil {
		return "", err
	}
	return PresentationFromContainer(c), nil
}
