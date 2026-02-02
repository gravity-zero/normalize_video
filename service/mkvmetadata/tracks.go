package mkvmetadata

import (
	"normalize_video/types"
	"strings"
	
	"golang.org/x/text/language"
)

func findTrackByISO(tracks []types.Track, isoCode string) *types.Track {
	if isoCode == "" {
		return nil
	}

	targetLang, err := language.Parse(isoCode)
	if err != nil {
		return nil
	}
	targetBase, _ := targetLang.Base()

	for _, track := range tracks {
		langStr := track.Properties.LanguageIetf
		if langStr == "" {
			langStr = track.Properties.Language
		}
		if langStr == "" {
			continue
		}

		trackLang, err := language.Parse(langStr)
		if err != nil {
			if strings.EqualFold(langStr, isoCode) {
				return &track
			}
			continue
		}

		trackBase, _ := trackLang.Base()
		if trackBase == targetBase {
			return &track
		}
	}

	return nil
}

func filterTracksByISO(tracks []types.Track, isoCode string) []types.Track {
	if isoCode == "" {
		return nil
	}

	targetLang, err := language.Parse(isoCode)
	if err != nil {
		return nil
	}
	targetBase, _ := targetLang.Base()

	var result []types.Track
	for _, track := range tracks {
		langStr := track.Properties.LanguageIetf
		if langStr == "" {
			langStr = track.Properties.Language
		}
		if langStr == "" {
			continue
		}

		trackLang, err := language.Parse(langStr)
		if err != nil {
			if strings.EqualFold(langStr, isoCode) {
				result = append(result, track)
			}
			continue
		}

		trackBase, _ := trackLang.Base()
		if trackBase == targetBase {
			result = append(result, track)
		}
	}

	return result
}