package mkvmetadata

import (
	"context"
	"errors"
	"normalize_video/config"
	"normalize_video/types"
	"os"
	"strings"

	"github.com/gravity-zero/mkvgo/matroska"
	"github.com/k0kubun/pp"
)

func UpdateMkvMetadata(m interface{}) (types.FileInfos, error) {
	var info types.FileInfos

	switch v := m.(type) {
	case *types.Serie:
		info.MkvFilePath = v.Normalizer.NewPath
		info.MkvTitle = v.Normalizer.Title + " " + v.SE
	case *types.Movie:
		info.MkvFilePath = v.Normalizer.NewPath
		info.MkvTitle = v.Normalizer.Title
	default:
		return info, errors.New("UpdateMkvMetadata: unknown type")
	}

	c, err := matroska.Open(context.Background(), info.MkvFilePath)
	if err != nil {
		return info, err
	}

	var audioTracks, subtitleTracks []types.Track
	for _, t := range c.Tracks {
		track := mkvTrackToType(t)
		switch t.Type {
		case matroska.AudioTrack:
			audioTracks = append(audioTracks, track)
		case matroska.SubtitleTrack:
			subtitleTracks = append(subtitleTracks, track)
		}
	}

	service := NewMkvService(MkvConfig{
		PreferredAudioLang:    config.PREFERRED_AUDIO_LANG,
		FallbackAudioLang:     config.FALLBACK_AUDIO_LANG,
		PreferredSubtitleLang: config.PREFERRED_SUBTITLE_LANG,
		FallbackSubtitleLang:  config.FALLBACK_SUBTITLE_LANG,
		SubtitleForcedOnly:    config.SUBTITLE_FORCED_ONLY,
	})

	bestAudioTrack := service.GetBestAudioTrack(audioTracks)
	bestSubTrack := service.GetBestSubtitleTrack(subtitleTracks)

	err = matroska.EditInPlace(context.Background(), info.MkvFilePath, func(c *matroska.Container) {
		c.Info.Title = info.MkvTitle

		for i := range c.Tracks {
			switch c.Tracks[i].Type {
			case matroska.AudioTrack:
				c.Tracks[i].IsDefault = bestAudioTrack != nil && c.Tracks[i].ID == uint64(bestAudioTrack.Properties.Number)
			case matroska.SubtitleTrack:
				c.Tracks[i].IsDefault = bestSubTrack != nil && c.Tracks[i].ID == uint64(bestSubTrack.Properties.Number)
			}
		}
	})
	if err != nil {
		pp.Printf("Warning: in-place edit failed, trying full rewrite: %v\n", err)
		err = matroska.EditMetadata(context.Background(), info.MkvFilePath, info.MkvFilePath+".tmp", func(c *matroska.Container) {
			c.Info.Title = info.MkvTitle
			for i := range c.Tracks {
				switch c.Tracks[i].Type {
				case matroska.AudioTrack:
					c.Tracks[i].IsDefault = bestAudioTrack != nil && c.Tracks[i].ID == uint64(bestAudioTrack.Properties.Number)
				case matroska.SubtitleTrack:
					c.Tracks[i].IsDefault = bestSubTrack != nil && c.Tracks[i].ID == uint64(bestSubTrack.Properties.Number)
				}
			}
		})
		if err != nil {
			return info, err
		}
		// replace original with temp
		if renameErr := rename(info.MkvFilePath+".tmp", info.MkvFilePath); renameErr != nil {
			return info, renameErr
		}
	}

	if bestAudioTrack != nil {
		if bestAudioTrack.Properties.TrackName != "" {
			info.MkvAudioTrack = strings.ToLower(bestAudioTrack.Properties.TrackName)
		} else {
			info.MkvAudioTrack = strings.ToLower(bestAudioTrack.Properties.LanguageIetf)
		}
	}

	if bestSubTrack != nil {
		if bestSubTrack.Properties.TrackName != "" {
			info.MkvSubTrack = strings.ToLower(bestSubTrack.Properties.TrackName)
		}
	}

	return info, nil
}

func mkvTrackToType(t matroska.Track) types.Track {
	return types.Track{
		Type: string(t.Type),
		Properties: types.TrackProperties{
			Number:       int(t.ID),
			Language:     t.Language,
			LanguageIetf: t.Language,
			TrackName:    t.Name,
		},
	}
}

func rename(src, dst string) error {
	return os.Rename(src, dst)
}
