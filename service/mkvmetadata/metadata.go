package mkvmetadata

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"normalize_video/types"
	"strings"

	"github.com/k0kubun/pp"
)

func UpdateMkvMetadataTitle(info types.FileInfos) error {
	cmd := exec.Command("mkvpropedit", info.MkvFilePath, "--edit", "info", "--set", fmt.Sprintf("title=%s", info.MkvTitle))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return pp.Errorf("mkvpropedit error: %v, output: %s", err, string(output))
	}
	return nil
}

func UpdateMkvMetadataTrack(info types.FileInfos, tracks []types.Track, defaultTrack *types.Track) error {
	for _, track := range tracks {
		flag := "0"
		if track.Properties.Number == defaultTrack.Properties.Number {
			flag = "1"
		}
		cmd := exec.Command("mkvpropedit", info.MkvFilePath, "--edit", fmt.Sprintf("track:%d", track.Properties.Number), "--set", "flag-default="+flag)
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

func UpdateMkvMetadata(m interface{}) (types.FileInfos, error) {
	var info types.FileInfos

	switch v := m.(type) {
	case *types.Serie:
		info.MkvFilePath = v.Normalizer.NewPath
		info.MkvTitle = v.Normalizer.Title
	case *types.Movie:
		info.MkvFilePath = v.Normalizer.NewPath
		info.MkvTitle = v.Normalizer.Title
	default:
		return info, errors.New("UpdateMkvMetadata: unknown type")
	}

	installed, err := IsMkvToolInstalled()
	if err != nil {
		return info, err
	}
	if !installed {
		return info, errors.New("mkvtoolnix is not installed")
	}

	if err := UpdateMkvMetadataTitle(info); err != nil {
		return info, err
	}

	cmd := exec.Command("mkvmerge", "-J", info.MkvFilePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return info, err
	}

	var metadata types.Metadata
	if err := json.Unmarshal(out.Bytes(), &metadata); err != nil {
		return info, err
	}

	var audioTracks, subtitleTracks []types.Track
	for _, track := range metadata.Tracks {
		if track.Type == "audio" {
			audioTracks = append(audioTracks, track)
		} else if track.Type == "subtitles" {
			subtitleTracks = append(subtitleTracks, track)
		}
	}

	var frenchAudioTrack *types.Track = GetBestAudioFrenchTrack(audioTracks)
	var frenchSubTrack *types.Track = GetBestSubtitleFrenchTrack(subtitleTracks)

	if frenchAudioTrack != nil {
		if frenchAudioTrack.Properties.TrackName != "" {
			info.MkvAudioTrack = strings.ToLower(frenchAudioTrack.Properties.TrackName)
		} else {
			info.MkvAudioTrack = strings.ToLower(frenchAudioTrack.Properties.LanguageIetf)
		}
	}

	if frenchSubTrack != nil && frenchSubTrack.Properties.TrackName != "" {
		info.MkvSubTrack = strings.ToLower(frenchSubTrack.Properties.TrackName)
	}

	if frenchAudioTrack != nil {
		if err := UpdateMkvMetadataTrack(info, audioTracks, frenchAudioTrack); err != nil {
			pp.Println("MKV UpdateMkvMetadataTrack AUDIO ERROR")
			return info, err
		}
	}
	if frenchSubTrack != nil {
		if err := UpdateMkvMetadataTrack(info, subtitleTracks, frenchSubTrack); err != nil {
			pp.Println("MKV UpdateMkvMetadataTrack SUBTITLE ERROR")
			return info, err
		}
	}

	return info, nil
}
