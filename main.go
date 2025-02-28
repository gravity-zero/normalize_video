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

		fileInfos, err := os.Stat(path)

		if err != nil {
			pp.Println("Erreur :", err)
			continue
		}

		if slices.Contains(config.Extensions, extension) && fileInfos.Size() > 0 {
			videos = append(videos, files.NewVideo(filename, path, extension))
		}
	}

	var moviesCount = 0
	var seriesCount = 0

	for _, video := range videos {
		if video.Type == "Serie" {
			serie := files.NewSerie(video)
			applyTreatment(serie.ActualPath, serie.Normalizer.NewPath, serie.Video.Filename, serie.Video.Extension, serie)
			seriesCount++
		} else {
			movie := files.NewMovie(video)
			applyTreatment(movie.ActualPath, movie.Normalizer.NewPath, movie.Video.Filename, movie.Video.Extension, movie)
			moviesCount++
		}
	}

	totalCount := moviesCount+seriesCount

	if totalCount > 0 {
		pp.Println("Movies Treated:", moviesCount)
		pp.Println("Series Treated:", seriesCount)
		pp.Println("Total Video Treated:", totalCount)
		pp.Println("________________________________________________________________________________")
	}
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


