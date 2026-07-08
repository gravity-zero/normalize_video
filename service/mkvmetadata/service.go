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

var forcedRegex = regexp.MustCompile(`\b(force[ds]?|forc)\b`)

// isForcedTrack reports whether a subtitle track is forced: either the
// container FlagForced is set, or the track name says so
func isForcedTrack(track types.Track) bool {
	if track.Properties.Forced {
		return true
	}
	if track.Properties.TrackName == "" {
		return false
	}
	trackName := RemoveAccent(strings.ToLower(track.Properties.TrackName))
	return forcedRegex.MatchString(trackName)
}

func (s *MkvService) GetBestSubtitleTrack(tracks []types.Track) *types.Track {
	if len(tracks) == 0 {
		return nil
	}

	preferredTracks := filterTracksByISO(tracks, s.config.PreferredSubtitleLang)

	for _, track := range preferredTracks {
		if isForcedTrack(track) {
			track.Properties.Forced = true
			return &track
		}
	}

	if s.config.SubtitleForcedOnly {
		if s.config.FallbackSubtitleLang != "" {
			fallbackTracks := filterTracksByISO(tracks, s.config.FallbackSubtitleLang)
			for _, track := range fallbackTracks {
				if isForcedTrack(track) {
					track.Properties.Forced = true
					return &track
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
