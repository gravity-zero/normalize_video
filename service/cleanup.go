package service

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// junkExtensions is the ONLY set of files cleanup may delete. Anything else
// (video files, archives, subtitles, unknown types) protects its whole
// directory from removal.
var junkExtensions = map[string]bool{
	".nfo":     true,
	".txt":     true,
	".jpg":     true,
	".jpeg":    true,
	".png":     true,
	".gif":     true,
	".sfv":     true,
	".md5":     true,
	".url":     true,
	".website": true,
	".torrent": true,
}

// CleanupResult reports what a cleanup pass removed (or would remove).
type CleanupResult struct {
	JunkRemoved []string
	DirsRemoved []string
}

// CleanupSourceDir removes release junk from dir after its video(s) were moved
// out, then removes the emptied directories. Safety rules, in order:
//
//   - dir must be strictly inside root; root itself is never cleaned
//   - if ANY video file remains anywhere under dir, nothing is touched
//   - if any file is not on the junk whitelist, nothing is touched either -
//     unknown content (archives, subtitles, executables...) keeps the dir alive
//   - only then: junk files are deleted and empty dirs removed bottom-up
//
// With dryRun the result lists what would be removed, disk untouched. The
// ignore set lists files to treat as already gone (dry-run: the video and
// sidecars whose move is planned but not executed).
func CleanupSourceDir(dir, root string, videoExtensions []string, dryRun bool, ignore map[string]bool) (CleanupResult, error) {
	var res CleanupResult

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return res, err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return res, err
	}
	rel, err := filepath.Rel(absRoot, absDir)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		// outside root, or root itself: refuse
		return res, nil
	}

	var junk []string
	var dirs []string
	blocked := false

	walkErr := filepath.WalkDir(absDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			blocked = true
			return filepath.SkipAll
		}
		if d.Type()&os.ModeSymlink != 0 {
			blocked = true
			return filepath.SkipAll
		}
		if d.IsDir() {
			dirs = append(dirs, path)
			return nil
		}
		if ignore[path] {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if slices.Contains(videoExtensions, strings.TrimPrefix(ext, ".")) {
			blocked = true
			return filepath.SkipAll
		}
		if !junkExtensions[ext] {
			blocked = true
			return filepath.SkipAll
		}
		junk = append(junk, path)
		return nil
	})
	if walkErr != nil || blocked {
		return res, nil
	}

	if dryRun {
		res.JunkRemoved = junk
		res.DirsRemoved = dirs
		return res, nil
	}

	for _, path := range junk {
		if err := os.Remove(path); err == nil {
			res.JunkRemoved = append(res.JunkRemoved, path)
		}
	}
	// deepest first so parents empty out before their own removal
	slices.SortFunc(dirs, func(a, b string) int { return len(b) - len(a) })
	removedSelf := false
	for _, d := range dirs {
		// os.Remove refuses non-empty directories, an extra safety net: a file
		// that appeared meanwhile keeps its directory alive
		if err := os.Remove(d); err == nil {
			res.DirsRemoved = append(res.DirsRemoved, d)
			if d == absDir {
				removedSelf = true
			}
		}
	}

	// Climb towards root removing parents that just emptied out (a serie
	// season dir leaves its show dir behind otherwise). os.Remove keeps
	// refusing non-empty dirs, and the loop stops strictly before root.
	if removedSelf {
		for parent := filepath.Dir(absDir); parent != absRoot && strings.HasPrefix(parent, absRoot+string(filepath.Separator)); parent = filepath.Dir(parent) {
			if err := os.Remove(parent); err != nil {
				break
			}
			res.DirsRemoved = append(res.DirsRemoved, parent)
		}
	}

	return res, nil
}
