package files

import (
	"normalize_video/config"
	"normalize_video/types"
	"regexp"
	"slices"
	"strings"
	
	"golang.org/x/text/language"
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
	re := regexp.MustCompile(config.REGEXSERIES)
	re2 := regexp.MustCompile(config.REGEXSERIESEXTEND)

	return re.MatchString(filenamePart) || re2.MatchString(filenamePart)
}

func getLanguage(filenamePart string) (isoCode string, originalTag string) {
	part := strings.ToLower(filenamePart)
	
	if iso, found := config.LanguageTags[part]; found {
		return iso, part
	}
	
	if len(part) >= 2 && len(part) <= 3 {
		if _, err := language.Parse(part); err == nil {
			return part, part
		}
	}
	
	return "", ""
}

func getQuality(filenamePart string) bool {
	return slices.Contains(config.Qualities, filenamePart)
}

func extractInfos(video *types.Video) {
	video.Type = "Movie"
	isVideoSerie := false

	partsToCheck := video.SplittedFilename
	if len(partsToCheck) > 0 {
		partsToCheck = partsToCheck[:len(partsToCheck)-1]
	}

	for _, split := range partsToCheck {
		if !isVideoSerie {
			isVideoSerie = isSerie(split)
		}

		if isoCode, originalTag := getLanguage(split); isoCode != "" {
			video.Language = isoCode
			video.LanguageTag = originalTag
		}

		if video.Quality == "" && getQuality(split) {
			video.Quality = split
		}
	}

	if isVideoSerie {
		video.Type = "Serie"
	}
}