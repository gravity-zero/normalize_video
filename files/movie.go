package files

import (
	"normalize_video/config"
	"normalize_video/service"
	"normalize_video/types"
)

func NewMovie(video *types.Video) *types.Movie {

	movie := &types.Movie{
		Video: video,
	}

	movie.Normalizer = *service.NewNormalizer(movie)
	service.NormalizePath(config.DEST_PATH+movie.Video.Type, movie)
	service.NormalizeEscapedNewPath(movie)

	return movie
}
