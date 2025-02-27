package mkvmetadata

import (
	"os/exec"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func RemoveAccent(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, err := transform.String(t, s)
	if err != nil {
		return s
	}
	return result
}

func IsMkvToolInstalled() (bool, error) {
	cmd := exec.Command("sh", "-c", "dpkg -l | grep mkvtoolnix | wc -l")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	countStr := strings.TrimSpace(string(output))
	if countStr != "0" {
		return true, nil
	}
	return false, nil
}
