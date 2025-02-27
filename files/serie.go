package files

import (
	"regexp"
	"normalize_video/config"
	"normalize_video/service"
	"normalize_video/types"
	"strings"
)

func NewSerie(video *types.Video) *types.Serie {

	serie := &types.Serie{
		Video: video,
	}

	extractSE(serie)
	extractSaisonEpisode(serie)
	service.NormalizeTitle(serie)
	constructSaisonPath(serie)
	service.NormalizeFilename(serie)
	service.NormalizePath(serie.SaisonPath, serie)
	service.NormalizeEscapedNewPath(serie)

	return serie
}

func extractSE(serie *types.Serie) {
	re, _ := regexp.Compile(config.REGEXSERIES)

	filename := service.SplitStringFromLastCharacter(service.FormatFilename(serie.Video.Filename), ".")
	splits := strings.Fields(filename[0])

	for _, split := range splits {
		if re.MatchString(split) {
			serie.SE = strings.ToUpper(split)
		}
	}
}

func extractSaisonEpisode(serie *types.Serie) {
	splittedSE := strings.Split(serie.SE, `E`)
	serie.Saison = splittedSE[0]
	serie.Episode = `E` + splittedSE[1]
}

func constructSaisonPath(serie *types.Serie) {
	serie.SaisonPath = config.DEST_PATH + serie.Video.Type + "/" + serie.Normalizer.Title + "/" + serie.Saison
}