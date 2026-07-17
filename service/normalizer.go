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
	} else if indexCut == 0 && len(firstStep) > 0 {
		cleanedParts = firstStep[0]
	} else if len(firstStep) > 0 {
		cleanedParts = strings.Join(firstStep, " ")
	}

	cleanedParts = strings.TrimSpace(cleanedParts)
	if strings.HasSuffix(cleanedParts, "-") {
		cleanedParts = strings.TrimSuffix(cleanedParts, "-")
		cleanedParts = strings.TrimSpace(cleanedParts)
	}

	// The year is what the title is cut on, so it is dropped by construction.
	// Remember it here: -keep-year puts it back in the normalized name. The
	// scan starts at 1 because a year at index 0 IS the title ("2012") - the
	// release year of such a film is the NEXT year token ("2012.2009")
	infos.GetNormalizer().Year = findYear(firstStep)

	infos.SetTitle(UcFirst(cleanedParts))
}

// findYear returns the release year of a source filename, "" when it carries
// none. It skips index 0, where a 4-digit token is the TITLE ("2012", "1917")
// rather than a release year - the release year of such a film is the next
// year token along.
func findYear(parts []string) string {
	for i := 1; i < len(parts); i++ {
		if isYear(parts[i]) {
			return parts[i]
		}
	}
	return ""
}

// isYear reports whether s is a plausible release year. getDateIndex accepts
// any 4-digit token as a cut point (it always has), but only a real year is
// worth writing back into a filename.
func isYear(s string) bool {
	y, err := strconv.Atoi(s)
	return err == nil && len(s) == 4 && y >= 1900 && y <= 2099
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
	return -1
}

func NormalizeFilename(infos types.Normalizable) {

	video := infos.GetVideo()
	ext := video.Extension
	title := infos.GetNormalizer().Title
	year := YearSuffix(infos.GetNormalizer())
	var normalized string

	switch infos.GetVideo().Type {
		case "Serie":
			// the year sits after SxxEyy, not glued to the title, so the
			// unflagged format ("Title SxxEyy - LANG - QUALITY") is untouched
			normalized = title + " " + infos.GetSE() + year

			if video.LanguageTag != "" {
				normalized += " - " + strings.ToUpper(video.LanguageTag)
			}

			if video.Quality != "" {
				normalized += " - " + strings.ToUpper(video.Quality)
			}

		case "Movie":
			normalized = title + year

			if video.Quality != "" {
				normalized += " - " + strings.ToUpper(video.Quality)
			}

		default:
			normalized = title + year
	}
	normalized += "." + ext
	infos.SetNormalizeFilename(normalized)
}

// YearSuffix is " - <year>" under -keep-year when the source filename carried
// one, "" otherwise - "Dune - 1984" vs "Dune - 2021" tells two films of the
// same name apart. For series it goes after SxxEyy, not glued to the title:
// "The sentinel S04E02 - 2008 - VF - 1080P.mkv".
func YearSuffix(n *types.Normalizer) string {
	if !config.KEEP_YEAR || n.Year == "" {
		return ""
	}
	return " - " + n.Year
}
