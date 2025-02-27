package mkvmetadata

import (
	"regexp"
	"normalize_video/types"
	"strings"
)

func getFrenchTracks(tracks []types.Track) []types.Track {
	var frenchTracks []types.Track
	frenchLangs := []string{"fr_fr", "fr-fr", "fr", "fre", "french"}
	for _, track := range tracks {
		lang := strings.ToLower(track.Properties.LanguageIetf)
		if lang == "" {
			lang = strings.ToLower(track.Properties.Language)
		}
		for _, f := range frenchLangs {
			if lang == f {
				frenchTracks = append(frenchTracks, track)
				break
			}
		}
	}
	return frenchTracks
}

func GetBestAudioFrenchTrack(tracks []types.Track) *types.Track {
	frenchTracks := getFrenchTracks(tracks)
	if len(frenchTracks) == 0 {
		return nil
	}
	if len(frenchTracks) == 1 {
		return &frenchTracks[0]
	}

	audioRegex := regexp.MustCompile(`\bvf?(f|i|q)\b`)
	var bestTracks []types.Track
	for _, track := range frenchTracks {
		if track.Properties.TrackName != "" {
			trackNameLower := strings.ToLower(track.Properties.TrackName)
			if audioRegex.MatchString(trackNameLower) {
				bestTracks = append(bestTracks, track)
			}
		}
	}

	if len(bestTracks) == 0 {
		for _, track := range frenchTracks {
			if track.Properties.TrackName != "" {
				trackNameLower := RemoveAccent(strings.ToLower(track.Properties.TrackName))
				if audioRegex.MatchString(trackNameLower) {
					return &track
				}
			}
		}
	}

	if len(bestTracks) > 0 {
		return &bestTracks[0]
	}
	return nil
}

func GetBestSubtitleFrenchTrack(tracks []types.Track) *types.Track {
	frenchTracks := getFrenchTracks(tracks)
	if len(frenchTracks) == 0 {
		return nil
	}

	forcedRegex := regexp.MustCompile(`\b(forces|forced|force|forc)\b`)
	for _, track := range frenchTracks {
		if track.Properties.TrackName != "" {
			trackName := RemoveAccent(strings.ToLower(track.Properties.TrackName))
			trackName = strings.ReplaceAll(trackName, "[", "")
			trackName = strings.ReplaceAll(trackName, "]", "")
			if forcedRegex.MatchString(trackName) {
				return &track
			}
		}
	}
	return nil
}
