package main

import (
	"fmt"
	"normalize_video/config"
	"normalize_video/files"
	"normalize_video/service"
	"normalize_video/service/mkvmetadata"
	"normalize_video/types"
	"path/filepath"
	"strings"
	"sync"

	"github.com/k0kubun/pp"
)

func main() {
	stats := processWithStreaming()
	printStats(stats)
}

func processWithStreaming() ProcessStats {
	videoChan := make(chan string, 100)
	var scanErr error
	var scanWg sync.WaitGroup

	scanWg.Add(1)
	go func() {
		defer scanWg.Done()
		scanErr = service.ScanVideoFilesStream(
			config.ORIGIN_PATH,
			config.RECURSIVE_SCAN,
			config.Extensions,
			videoChan,
		)
	}()

	stats := ProcessStats{}
	var processWg sync.WaitGroup
	var mu sync.Mutex

	workerJobs := make(chan *types.Video, config.MAX_WORKERS)

	for i := 0; i < config.MAX_WORKERS; i++ {
		processWg.Add(1)
		go func() {
			defer processWg.Done()
			for video := range workerJobs {
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

	go func() {
		for path := range videoChan {
			filename := strings.ToLower(filepath.Base(path))
			filenameParts := service.SplitFilename(filename)
			extension := filenameParts[len(filenameParts)-1]

			video := files.NewVideo(filename, filenameParts, path, extension)
			workerJobs <- video
		}
		close(workerJobs)
	}()

	scanWg.Wait()
	if scanErr != nil {
		pp.Printf("Scan error: %v\n", scanErr)
	}

	processWg.Wait()

	return stats
}

type ProcessStats struct {
	MoviesCount int
	SeriesCount int
	Errors      []error
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