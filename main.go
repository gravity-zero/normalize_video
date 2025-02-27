package main

import (
	"os"
	"path/filepath"
	"normalize_video/config"
	"normalize_video/files"
	"normalize_video/service"
	"normalize_video/service/mkvmetadata"
	"normalize_video/types"
	"slices"
	"strings"

	"github.com/k0kubun/pp"
)

func main() {

	originFiles, err := os.ReadDir(config.ORIGIN_PATH)

	if err != nil {
		pp.Fatal(err)
	}

	videos := []*types.Video{}

	for _, file := range originFiles {
		filename := strings.ToLower(file.Name())
		fileNameParts := strings.Split(filename, ".")

		extension := fileNameParts[len(fileNameParts)-1]
		path := filepath.Join(config.ORIGIN_PATH, file.Name())

		if slices.Contains(config.Extensions, extension) {
			videos = append(videos, files.NewVideo(filename, path, extension))
		}
	}

	var countMovies = 0
	var countSeries = 0

	for _, video := range videos {
		if video.Type == "Serie" {
			serie := files.NewSerie(video)
			applyTreatment(serie.ActualPath, serie.Normalizer.NewPath, serie.Video.Filename, serie.Video.Extension, serie)
			countSeries++
		} else {
			movie := files.NewMovie(video)
			applyTreatment(movie.ActualPath, movie.Normalizer.NewPath, movie.Video.Filename, movie.Video.Extension, movie)
			countMovies++
		}
	}

	pp.Println("Movies Treated:", countMovies)
	pp.Println("Series Treated:", countSeries)
	pp.Println("Total Video Treated:", countMovies+countSeries)
}

func applyTreatment(actualPath, newPath, filename, extension string, media interface{}){

	if err := service.MoveFile(actualPath, newPath); err != nil {
		pp.Printf("Erreur lors du renommage %s : %v", filename, err)
	}

	if strings.ToLower(extension) != "mkv" {
		return
	}

	result, err := mkvmetadata.UpdateMkvMetadata(media)

	if err != nil {
		pp.Fatalf("Error: %v", err)
	}

	switch v := media.(type) {
		case *types.Serie:
			v.MkvMetadata = result
		case *types.Movie:
			v.MkvMetadata = result
	}

	service.PrintStructTable(media)
	pp.Println("________________________________________________________________________________")
}


