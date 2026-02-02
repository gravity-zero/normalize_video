package mkvmetadata

import (
	"normalize_video/config"
	"normalize_video/types"
	"regexp"
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

		if trackLang.String() == targetLang.String() {
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

		if trackLang.String() == targetLang.String() {
			result = append(result, track)
		}
	}

	return result
}

func GetBestAudioTrack(tracks []types.Track) *types.Track {
	if len(tracks) == 0 {
		return nil
	}

	if track := findTrackByISO(tracks, config.PREFERRED_AUDIO_LANG); track != nil {
		return track
	}

	if config.FALLBACK_AUDIO_LANG != "" {
		if track := findTrackByISO(tracks, config.FALLBACK_AUDIO_LANG); track != nil {
			return track
		}
	}

	return nil
}

func GetBestSubtitleTrack(tracks []types.Track) *types.Track {
	if len(tracks) == 0 {
		return nil
	}

	forcedRegex := regexp.MustCompile(`\b(force[ds]?|forc)\b`)

	preferredTracks := filterTracksByISO(tracks, config.PREFERRED_SUBTITLE_LANG)

	for _, track := range preferredTracks {
		if track.Properties.TrackName != "" {
			trackName := RemoveAccent(strings.ToLower(track.Properties.TrackName))
			if forcedRegex.MatchString(trackName) {
				return &track
			}
		}
	}

	if config.SUBTITLE_FORCED_ONLY {
		if config.FALLBACK_SUBTITLE_LANG != "" {
			fallbackTracks := filterTracksByISO(tracks, config.FALLBACK_SUBTITLE_LANG)
			for _, track := range fallbackTracks {
				if track.Properties.TrackName != "" {
					trackName := RemoveAccent(strings.ToLower(track.Properties.TrackName))
					if forcedRegex.MatchString(trackName) {
						return &track
					}
				}
			}
		}
		return nil
	}

	if len(preferredTracks) > 0 {
		return &preferredTracks[0]
	}

	if config.FALLBACK_SUBTITLE_LANG != "" {
		if track := findTrackByISO(tracks, config.FALLBACK_SUBTITLE_LANG); track != nil {
			return track
		}
	}

	return nil
}
