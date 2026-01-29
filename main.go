package main

import (
	"fmt"
	"normalize_video/config"
	"normalize_video/files"
	"normalize_video/service"
	"normalize_video/service/mkvmetadata"
	"normalize_video/types"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/k0kubun/pp"
)

func main() {
	videos, err := loadVideos()
	if err != nil {
		pp.Printf("Error loading videos: %v\n", err)
		os.Exit(1)
	}

	if len(videos) == 0 {
		pp.Println("No videos found")
		return
	}

	stats := processVideosWithWorkerPool(videos, config.MAX_WORKERS)
	printStats(stats)
}

func loadVideos() ([]*types.Video, error) {
	videoPaths, err := service.ScanVideoFiles(
		config.ORIGIN_PATH,
		config.RECURSIVE_SCAN,
		config.Extensions,
	)
	if err != nil {
		return nil, err
	}

	var videos []*types.Video
	for _, path := range videoPaths {
		filename := strings.ToLower(filepath.Base(path))
		filenameParts := service.SplitFilename(filename)
		extension := filenameParts[len(filenameParts)-1]

		video := files.NewVideo(filename, filenameParts, path, extension)
		videos = append(videos, video)
	}

	return videos, nil
}

type ProcessStats struct {
	MoviesCount int
	SeriesCount int
	Errors      []error
}

func processVideosWithWorkerPool(videos []*types.Video, numWorkers int) ProcessStats {
	jobs := make(chan *types.Video, len(videos))
	var wg sync.WaitGroup
	var mu sync.Mutex
	stats := ProcessStats{}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for video := range jobs {
				err := processVideo(video)
				mu.Lock()
				if err != nil {
					stats.Errors = append(stats.Errors, err)
				} else {
					if video.Type == "Serie" {
						stats.SeriesCount++
					} else {
						stats.MoviesCount++
					}
				}
				mu.Unlock()
			}
		}()
	}

	for _, video := range videos {
		jobs <- video
	}
	close(jobs)
	wg.Wait()

	return stats
}

func processVideo(video *types.Video) error {
	if video.Type == "Serie" {
		serie := files.NewSerie(video)
		return processMedia(serie, serie.OriginPath, serie.Normalizer.NewPath, serie.Video.Extension)
	}

	movie := files.NewMovie(video)
	return processMedia(movie, movie.OriginPath, movie.Normalizer.NewPath, movie.Video.Extension)
}

func processMedia(media any, originPath, newPath, extension string) error {
	if err := service.MoveFile(originPath, newPath); err != nil {
		return fmt.Errorf("move file error for %s: %w", filepath.Base(originPath), err)
	}

	if strings.ToLower(extension) == "mkv" {
		result, err := mkvmetadata.UpdateMkvMetadata(media)
		if err != nil {
			pp.Printf("Warning: MKV metadata update failed for %s: %v\n", filepath.Base(newPath), err)
		} else {
			switch v := media.(type) {
			case *types.Serie:
				v.MkvMetadata = result
			case *types.Movie:
				v.MkvMetadata = result
			}
		}
	}

	service.PrintStructTable(media)
	pp.Println("________________________________________________________________________________")

	return nil
}

func printStats(stats ProcessStats) {
	if len(stats.Errors) > 0 {
		pp.Println("\nErrors encountered:")
		for _, err := range stats.Errors {
			pp.Printf("  - %v\n", err)
		}
	}

	totalCount := stats.MoviesCount + stats.SeriesCount
	if totalCount > 0 {
		pp.Println("")
		pp.Println("Movies processed:", stats.MoviesCount)
		pp.Println("Series processed:", stats.SeriesCount)
		pp.Println("Total videos processed:", totalCount)
		pp.Println("________________________________________________________________________________")
	}
}