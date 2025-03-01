package main

import (
	"normalize_video/config"
	"normalize_video/files"
	"normalize_video/service"
	"normalize_video/service/mkvmetadata"
	"normalize_video/types"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

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

	var wg sync.WaitGroup
	var moviesCount, seriesCount int
	var mu sync.Mutex

	for _, video := range videos {
		wg.Add(1)
		go func(video *types.Video) {
			defer wg.Done()
			if video.Type == "Serie" {
				serie := files.NewSerie(video)
				processVideos(serie.OriginPath, serie.Normalizer.NewPath, serie.Video.Filename, serie.Video.Extension, serie)
				mu.Lock()
				seriesCount++
				mu.Unlock()
			} else {
				movie := files.NewMovie(video)
				processVideos(movie.OriginPath, movie.Normalizer.NewPath, movie.Video.Filename, movie.Video.Extension, movie)
				mu.Lock()
				moviesCount++
				mu.Unlock()
			}
		}(video)
	}

	wg.Wait()

	totalCount := moviesCount + seriesCount

	if totalCount > 0 {
		pp.Println("Movies processed:", moviesCount)
		pp.Println("Series processed:", seriesCount)
		pp.Println("Total Video processed:", totalCount)
		pp.Println("________________________________________________________________________________")
	}
}

func processVideos(originPath, newPath, filename, extension string, media any) {

	if err := service.MoveFile(originPath, newPath); err != nil {
		pp.Printf("An error occurred while renaming %s : %v", filename, err)
	}

	if strings.ToLower(extension) == "mkv" {
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
	}

	service.PrintStructTable(media)
	pp.Println("________________________________________________________________________________")
}
