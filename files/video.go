package files

import (
	"normalize_video/config"
	"normalize_video/types"
	"regexp"
	"slices"
	"strings"
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

// getLanguage recognises a language ONLY from the known release tags
// (config.LanguageTags: vf, vostfr, multi, truefrench, en, ...).
//
// It used to fall back on x/text's language.Parse for any 2-3 letter token,
// which is not a language check but a well-formedness check: "big", "the",
// "in", "age", "sun" and "vol" are all valid ISO 639-3 codes, so any short
// word in a title was taken for a language - and since the title is cut at
// the first language token, it was cut mid-title. "Men in Black" normalised
// to "Men", "War of the Worlds" to "War of", "Ice Age" to "Ice".
func getLanguage(filenamePart string) (isoCode string, originalTag string) {
	part := strings.ToLower(filenamePart)

	if iso, found := config.LanguageTags[part]; found {
		return iso, part
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