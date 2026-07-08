package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveCollision decides what to do when destination already exists, per
// mode: "skip" keeps the existing file (returns skip=true), "replace"
// overwrites it, "suffix" finds a free "name (N).ext" variant. An empty mode
// keeps the historical behavior: overwrite. When the destination is free the
// path comes back unchanged whatever the mode.
func ResolveCollision(destination, mode string) (string, bool) {
	if mode == "" {
		return destination, false
	}
	if _, err := os.Stat(destination); os.IsNotExist(err) {
		return destination, false
	}

	switch mode {
	case "skip":
		return destination, true
	case "replace":
		return destination, false
	case "suffix":
		ext := filepath.Ext(destination)
		base := strings.TrimSuffix(destination, ext)
		for i := 1; i <= 999; i++ {
			candidate := fmt.Sprintf("%s (%d)%s", base, i, ext)
			if _, err := os.Stat(candidate); os.IsNotExist(err) {
				return candidate, false
			}
		}
		// 999 duplicates: give up rather than overwrite
		return destination, true
	}

	return destination, false
}
