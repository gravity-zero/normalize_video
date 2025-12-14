package service

import (
	"normalize_video/config"
	"normalize_video/types"
	"regexp"
	"strconv"
	"strings"
	"slices"
)

type NormalizerFunc func(types.Normalizable)

var normalizerPipeline = []NormalizerFunc{
	NormalizeTitle,
	NormalizeFilename,
}

func NewNormalizer(infos types.Normalizable) *types.Normalizer {
	for _, fn := range normalizerPipeline {
		fn(infos)
	}
	return infos.GetNormalizer()
}

func NormalizePath(destinationPath string, infos types.Normalizable) {
	infos.SetNewPath(destinationPath + "/" + infos.GetNormalizer().NormalizeFilename)
}

func NormalizeEscapedNewPath(infos types.Normalizable) {
	newPath := infos.GetNormalizer().NewPath
	re := regexp.MustCompile(`(\s+)`)
	escaped := re.ReplaceAllString(newPath, "\\$1")
	infos.SetEscapedNewPath(escaped)
}

func NormalizeTitle(infos types.Normalizable) {
	parts := infos.GetVideo().SplittedFilename

	firstStep := parts
	if len(parts) > 1 {
		firstStep = parts[:len(parts)-1]
	}

	firstIndex := getDateIndex(firstStep)
	secondIndex := getFirstParamIndex(firstStep, infos)

	var indexCut int
	if firstIndex > -1 && (firstIndex < secondIndex || secondIndex <= 0) {
		indexCut = firstIndex
	} else {
		indexCut = secondIndex
	}

	var cleanedParts string
	if indexCut > 0 && indexCut <= len(firstStep) {
		cleanedParts = strings.Join(firstStep[:indexCut], " ")
	} else if len(firstStep) > 0 {
		cleanedParts = firstStep[0]
	}

	cleanedParts = strings.TrimSpace(cleanedParts)
	if strings.HasSuffix(cleanedParts, "-") {
		cleanedParts = strings.TrimSuffix(cleanedParts, "-")
		cleanedParts = strings.TrimSpace(cleanedParts)
	}

	infos.SetTitle(UcFirst(cleanedParts))
}

func UcFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

func getDateIndex(arr []string) int {
	for i, s := range arr {
		if len(s) == 4 {
			if _, err := strconv.Atoi(s); err == nil {
				return i
			}
		}
	}
	return -1
}

func getFirstParamIndex(arr []string, infos types.Normalizable) int {
	knownParams := []string{infos.GetVideo().Quality, infos.GetVideo().Language}

	if infos.GetVideo().Type == "Serie" {
		
	}

	for i, s := range arr {
		if infos.GetVideo().Type == "Serie" {
			knownParams = append(knownParams, strings.ToLower(infos.GetSE()))
			var reSeries = regexp.MustCompile(config.REGEXSERIES)
			var reSeriesExtend = regexp.MustCompile(config.REGEXSERIESEXTEND)
			if reSeries.MatchString(s) || reSeriesExtend.MatchString(s) {
				knownParams = append(knownParams, strings.ToLower(s))
			}
		}
		if slices.Contains(knownParams, s) {
			return i
		}
	}
	return 0
}

func NormalizeFilename(infos types.Normalizable) {

	video := infos.GetVideo()
	ext := video.Extension
	var normalized string

	switch infos.GetVideo().Type {
		case "Serie":
			normalized = infos.GetNormalizer().Title + " " + infos.GetSE()

			if video.Language != "" {
				normalized += " - " + strings.ToUpper(video.Language)
			}

			if video.Quality != "" {
				normalized += " - " + strings.ToUpper(video.Quality)
			}

		case "Movie":
			normalized = infos.GetNormalizer().Title

			if video.Quality != "" {
				normalized += " - " + strings.ToUpper(video.Quality)
			}

		default:
			normalized = infos.GetNormalizer().Title
	}
	normalized += "." + ext
	infos.SetNormalizeFilename(normalized)
}
