package mkvmetadata

import (
	"context"
	"normalize_video/config"
	"os"
	"path/filepath"
	"strings"

	"github.com/gravity-zero/mkvgo/matroska"
	"github.com/k0kubun/pp"
	"golang.org/x/text/language"
)

// SidecarSub describes an external subtitle file sitting next to a video.
type SidecarSub struct {
	Path   string
	Lang   string // 2-letter code, resolved from the filename tokens
	Forced bool
	ASS    bool // .ass/.ssa, merged via MergeASS instead of MergeSubtitle
}

// TrackName is the display name written for the merged track. Forced sidecars
// get an explicit "(forced)" so the forced-subtitle selection recognizes them.
func (s SidecarSub) TrackName() string {
	name := strings.ToUpper(s.Lang)
	if s.Forced {
		name += " (forced)"
	}
	return name
}

// FindSubtitleSidecars looks next to originPath for subtitle files sharing
// its basename: "movie.srt", "movie.fr.srt", "movie.fr.forced.srt",
// "movie.ass"... Language and forced markers are read from the extra
// filename tokens; a bare "movie.srt" falls back to the preferred language.
func FindSubtitleSidecars(originPath string) []SidecarSub {
	dir := filepath.Dir(originPath)
	base := strings.TrimSuffix(filepath.Base(originPath), filepath.Ext(originPath))
	baseLower := strings.ToLower(base)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var subs []SidecarSub
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		nameLower := strings.ToLower(e.Name())
		ext := filepath.Ext(nameLower)
		if ext != ".srt" && ext != ".ass" && ext != ".ssa" {
			continue
		}

		stem := strings.TrimSuffix(nameLower, ext)
		if stem != baseLower && !strings.HasPrefix(stem, baseLower+".") {
			continue
		}

		sub := SidecarSub{
			Path: filepath.Join(dir, e.Name()),
			ASS:  ext == ".ass" || ext == ".ssa",
		}
		if len(stem) > len(baseLower) {
			for _, tok := range strings.Split(stem[len(baseLower)+1:], ".") {
				switch {
				case tok == "forced" || tok == "force":
					sub.Forced = true
				default:
					if iso, ok := config.LanguageTags[tok]; ok && iso != "multi" {
						sub.Lang = iso
					}
				}
			}
		}
		if sub.Lang == "" {
			sub.Lang = config.PREFERRED_SUBTITLE_LANG
		}

		subs = append(subs, sub)
	}

	return subs
}

// MergeSubtitleSidecars embeds every subtitle sidecar of originPath into the
// MKV at mkvPath (one full rewrite per sidecar, progress logged) and deletes
// the sidecar on success. Returns the number of merged subtitles.
func MergeSubtitleSidecars(ctx context.Context, originPath, mkvPath string) int {
	merged := 0
	for _, sub := range FindSubtitleSidecars(originPath) {
		name := filepath.Base(mkvPath)
		pp.Printf("   %s: merging subtitle %s...\n", name, filepath.Base(sub.Path))

		tmpPath := mkvPath + ".sub.tmp"
		opts := matroska.Options{Progress: progressLogger(name, "subtitle merge")}
		var err error
		if sub.ASS {
			err = matroska.MergeASS(ctx, mkvPath, sub.Path, tmpPath, toISO3(sub.Lang), sub.TrackName(), opts)
		} else {
			err = matroska.MergeSubtitle(ctx, mkvPath, sub.Path, tmpPath, toISO3(sub.Lang), sub.TrackName(), opts)
		}
		if err != nil {
			os.Remove(tmpPath)
			pp.Printf("Warning: subtitle merge failed for %s: %v\n", filepath.Base(sub.Path), err)
			continue
		}
		if err := os.Rename(tmpPath, mkvPath); err != nil {
			os.Remove(tmpPath)
			pp.Printf("Warning: subtitle merge failed for %s: %v\n", filepath.Base(sub.Path), err)
			continue
		}

		if err := os.Remove(sub.Path); err != nil {
			pp.Printf("Warning: merged sidecar could not be removed: %s\n", sub.Path)
		}
		merged++
	}
	return merged
}

// toISO3 converts a 2-letter language code to the 3-letter ISO 639 code the
// Matroska legacy Language field expects ("fr" -> "fra")
func toISO3(code string) string {
	tag, err := language.Parse(code)
	if err != nil {
		return code
	}
	base, _ := tag.Base()
	return base.ISO3()
}
