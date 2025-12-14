package files

import (
	"normalize_video/config"
	"normalize_video/service"
	"normalize_video/types"
	"regexp"
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
	re2, _ := regexp.Compile(config.REGEXSERIESEXTEND)
	for _, split := range serie.Video.SplittedFilename {
		if re.MatchString(split) {
			serie.SE = strings.ToUpper(split)
			return
		}

		if m := re2.FindStringSubmatch(split); len(m) >= 3 {
			season := service.Normalize2digits(m[1])
			episode := service.Normalize2digits(m[2])
			serie.SE = "S" + season + "E" + episode
			return
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
