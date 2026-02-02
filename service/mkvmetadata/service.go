package mkvmetadata

import (
	"normalize_video/types"
	"regexp"
	"strings"
)

type MkvConfig struct {
	PreferredAudioLang    string
	FallbackAudioLang     string
	PreferredSubtitleLang string
	FallbackSubtitleLang  string
	SubtitleForcedOnly    bool
}

type MkvService struct {
	config MkvConfig
}

func NewMkvService(cfg MkvConfig) *MkvService {
	return &MkvService{config: cfg}
}

func (s *MkvService) GetBestAudioTrack(tracks []types.Track) *types.Track {
	if len(tracks) == 0 {
		return nil
	}

	if track := findTrackByISO(tracks, s.config.PreferredAudioLang); track != nil {
		return track
	}

	if s.config.FallbackAudioLang != "" {
		if track := findTrackByISO(tracks, s.config.FallbackAudioLang); track != nil {
			return track
		}
	}

	return nil
}

func (s *MkvService) GetBestSubtitleTrack(tracks []types.Track) *types.Track {
	if len(tracks) == 0 {
		return nil
	}

	forcedRegex := regexp.MustCompile(`\b(force[ds]?|forc)\b`)

	preferredTracks := filterTracksByISO(tracks, s.config.PreferredSubtitleLang)

	for _, track := range preferredTracks {
		if track.Properties.TrackName != "" {
			trackName := RemoveAccent(strings.ToLower(track.Properties.TrackName))
			if forcedRegex.MatchString(trackName) {
				return &track
			}
		}
	}

	if s.config.SubtitleForcedOnly {
		if s.config.FallbackSubtitleLang != "" {
			fallbackTracks := filterTracksByISO(tracks, s.config.FallbackSubtitleLang)
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

	if s.config.FallbackSubtitleLang != "" {
		if track := findTrackByISO(tracks, s.config.FallbackSubtitleLang); track != nil {
			return track
		}
	}

	return nil
}
