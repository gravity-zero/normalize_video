package files

import (
	"normalize_video/config"
	"normalize_video/types"
	"regexp"
	"slices"
)

func NewVideo(filename string, fileNameParts []string, filepath string, extension string) *types.Video {
	newVideo := &types.Video{
		Filename:         filename,
		SplittedFilename: fileNameParts,
		OriginPath:       filepath,
		Extension:        extension,
	}

	extractInfos(newVideo)

	return newVideo
}

func isSerie(filenamePart string) bool {
	re, _ := regexp.Compile(config.REGEXSERIES)

	return re.MatchString(filenamePart)
}

func getLanguage(filenamePart string) bool {

	return slices.Contains(config.Languages, filenamePart)
}

func getQuality(filenamePart string) bool {

	return slices.Contains(config.Qualities, filenamePart)
}

func extractInfos(video *types.Video) {
	video.Type = "Movie" //Default
	isVideoSerie := false

	for _, split := range video.SplittedFilename {
		if !isVideoSerie {
			isVideoSerie = isSerie(split)
		}

		if getLanguage(split) {
			video.Language = split
		}

		if getQuality(split) {
			video.Quality = split
		}
	}

	if isVideoSerie {
		video.Type = "Serie"
	}
}
