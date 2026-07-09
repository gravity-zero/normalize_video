package main

import (
	"context"
	"fmt"
	"normalize_video/config"
	"normalize_video/files"
	"normalize_video/service"
	"normalize_video/service/mkvmetadata"
	"normalize_video/types"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/k0kubun/pp"
)

// printMu serializes the per-file result blocks so parallel workers do not
// interleave their tables. Single-line progress logs stay unserialized.
var printMu sync.Mutex

// dedupIndex maps content fingerprints to library paths; nil unless -dedup.
var dedupIndex *service.DedupIndex

func main() {
	config.ParseFlags()

	if err := service.OpenJournal(config.JOURNAL_PATH); err != nil {
		pp.Printf("Error opening journal %s: %v\n", config.JOURNAL_PATH, err)
		os.Exit(1)
	}
	defer service.CloseJournal()

	if config.DEDUP {
		idx, err := service.LoadDedupIndex(filepath.Join(config.DEST_PATH, ".normalize_fingerprints.jsonl"))
		if err != nil {
			pp.Printf("Error loading dedup index: %v\n", err)
			os.Exit(1)
		}
		dedupIndex = idx
	}

	if config.DRY_RUN {
		pp.Println("DRY RUN: showing the plan, nothing will be modified")
		pp.Println("________________________________________________________________________________")
	}

	start := time.Now()
	var stats ProcessStats
	if config.WATCH {
		stats = watchLoop()
	} else {
		stats = processWithStreaming()
		runCleanup(&stats)
	}
	printStats(stats, time.Since(start))
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

	stats := newProcessStats()
	var processWg sync.WaitGroup
	var mu sync.Mutex

	workerJobs := make(chan *types.Video, config.MAX_WORKERS)

	for i := 0; i < config.MAX_WORKERS; i++ {
		processWg.Add(1)
		go func() {
			defer processWg.Done()
			for video := range workerJobs {
				res, err := processVideo(video)
				collectResult(&stats, &mu, video, res, err)
			}
		}()
	}

	go func() {
		for path := range videoChan {
			workerJobs <- newVideoFromPath(path)
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

// watchLoop rescans ORIGIN_PATH periodically and processes a file once its
// size and mtime are stable between two scans (download finished). Polling on
// purpose: inotify does not fire for Windows-side writes on WSL /mnt/* drvfs
// mounts. Ctrl+C stops cleanly and prints the totals.
func watchLoop() ProcessStats {
	stats := newProcessStats()
	var mu sync.Mutex

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	jobs := make(chan *types.Video)
	var wg sync.WaitGroup
	for i := 0; i < config.MAX_WORKERS; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for video := range jobs {
				res, err := processVideo(video)
				collectResult(&stats, &mu, video, res, err)
			}
		}()
	}

	type fileState struct {
		size int64
		mod  time.Time
	}
	pending := map[string]fileState{}
	handled := map[string]bool{}
	interval := time.Duration(config.WATCH_INTERVAL_SECONDS) * time.Second

	pp.Println(fmt.Sprintf("Watching %s every %ds (Ctrl+C to stop)", config.ORIGIN_PATH, config.WATCH_INTERVAL_SECONDS))

	for {
		paths, err := service.ScanVideoFiles(config.ORIGIN_PATH, config.RECURSIVE_SCAN, config.Extensions)
		if err != nil {
			pp.Printf("Scan error: %v\n", err)
		}

		current := map[string]bool{}
		for _, path := range paths {
			current[path] = true
			if handled[path] {
				continue
			}
			st, err := os.Stat(path)
			if err != nil {
				continue
			}

			prev, seen := pending[path]
			if seen && prev.size == st.Size() && prev.mod.Equal(st.ModTime()) {
				handled[path] = true
				delete(pending, path)
				select {
				case jobs <- newVideoFromPath(path):
				case <-ctx.Done():
				}
			} else {
				pending[path] = fileState{st.Size(), st.ModTime()}
				if !seen {
					pp.Printf("   %s: waiting for a stable size...\n", filepath.Base(path))
				}
			}
		}

		// files gone from the scan were moved away: forget them
		for p := range handled {
			if !current[p] {
				delete(handled, p)
			}
		}
		for p := range pending {
			if !current[p] {
				delete(pending, p)
			}
		}

		mu.Lock()
		runCleanup(&stats)
		mu.Unlock()

		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			mu.Lock()
			runCleanup(&stats)
			mu.Unlock()
			return stats
		case <-time.After(interval):
		}
	}
}

func newVideoFromPath(path string) *types.Video {
	filename := strings.ToLower(filepath.Base(path))
	filenameParts := service.SplitFilename(filename)
	extension := filenameParts[len(filenameParts)-1]
	return files.NewVideo(filename, filenameParts, path, extension)
}

type ProcessStats struct {
	MoviesCount       int
	SeriesCount       int
	SeekIndexRepaired int
	SubtitlesMerged   int
	Converted         int
	Skipped           int
	Hashed            int
	Salvaged          int
	Duplicates        int
	JunkRemoved       int
	DirsRemoved       int
	Errors            []error
	sourceDirs        map[string]bool
	plannedGone       map[string]bool
}

func newProcessStats() ProcessStats {
	return ProcessStats{sourceDirs: map[string]bool{}, plannedGone: map[string]bool{}}
}

type mediaResult struct {
	SeekIndexRepaired bool
	SubtitlesMerged   int
	Converted         bool
	Skipped           bool
	Hashed            bool
	Salvaged          bool
	Duplicate         bool
	SourceDir         string
	// PlannedGone lists files a dry-run pretends were moved/consumed, so the
	// cleanup preview matches what a real run would leave behind
	PlannedGone []string
}

func collectResult(stats *ProcessStats, mu *sync.Mutex, video *types.Video, res mediaResult, err error) {
	mu.Lock()
	defer mu.Unlock()

	switch {
	case err != nil:
		stats.Errors = append(stats.Errors, err)
		service.Journal("move", video.OriginPath, "", "failed", err.Error())
	case res.Skipped:
		stats.Skipped++
	default:
		if video.Type == "Serie" {
			stats.SeriesCount++
		} else {
			stats.MoviesCount++
		}
		if res.SourceDir != "" {
			stats.sourceDirs[res.SourceDir] = true
		}
		for _, p := range res.PlannedGone {
			stats.plannedGone[p] = true
		}
	}

	if res.SeekIndexRepaired {
		stats.SeekIndexRepaired++
	}
	if res.Converted {
		stats.Converted++
	}
	if res.Hashed {
		stats.Hashed++
	}
	if res.Salvaged {
		stats.Salvaged++
	}
	if res.Duplicate {
		stats.Duplicates++
	}
	stats.SubtitlesMerged += res.SubtitlesMerged
}

// runCleanup sweeps the source dirs of processed videos. Caller holds the
// stats lock in watch mode; batch mode calls it after all workers stopped.
func runCleanup(stats *ProcessStats) {
	if !config.CLEANUP_SOURCE {
		return
	}
	for dir := range stats.sourceDirs {
		res, err := service.CleanupSourceDir(dir, config.ORIGIN_PATH, config.Extensions, config.DRY_RUN, stats.plannedGone)
		if err != nil {
			pp.Printf("Warning: cleanup failed for %s: %v\n", dir, err)
			continue
		}
		delete(stats.sourceDirs, dir)
		if len(res.JunkRemoved) == 0 && len(res.DirsRemoved) == 0 {
			continue
		}
		stats.JunkRemoved += len(res.JunkRemoved)
		stats.DirsRemoved += len(res.DirsRemoved)
		status := "done"
		if config.DRY_RUN {
			status = "planned"
		}
		service.Journal("cleanup", dir, "", status,
			fmt.Sprintf("%d junk file(s), %d dir(s)", len(res.JunkRemoved), len(res.DirsRemoved)))
		for _, f := range res.JunkRemoved {
			pp.Printf("   cleanup: %s\n", f)
		}
	}
}

func processVideo(video *types.Video) (mediaResult, error) {
	var media types.Normalizable
	if video.Type == "Serie" {
		media = files.NewSerie(video)
	} else {
		media = files.NewMovie(video)
	}
	return processMedia(media)
}

func processMedia(media types.Normalizable) (mediaResult, error) {
	var res mediaResult

	video := media.GetVideo()
	originPath := video.OriginPath
	newPath := media.GetNormalizer().NewPath
	extension := strings.ToLower(video.Extension)
	res.SourceDir = filepath.Dir(originPath)

	pp.Printf("-> %s\n", filepath.Base(originPath))

	convert := config.CONVERT_MP4 && config.MP4Convertible[extension]
	plainNewPath := newPath
	if convert {
		newPath = strings.TrimSuffix(newPath, filepath.Ext(newPath)) + ".mkv"
		media.SetNewPath(newPath)
		extension = "mkv"
	}

	if config.ON_COLLISION != "" {
		resolved, skip := service.ResolveCollision(newPath, config.ON_COLLISION)
		if skip {
			pp.Printf("   %s: destination exists, skipped\n", filepath.Base(newPath))
			service.Journal("collision", originPath, newPath, "skipped", "destination exists")
			res.Skipped = true
			return res, nil
		}
		if resolved != newPath {
			service.Journal("collision", originPath, newPath, "suffixed", resolved)
			newPath = resolved
			media.SetNewPath(newPath)
		}
	}

	if config.DRY_RUN {
		return dryRunMedia(media, originPath, newPath, extension, convert)
	}

	ctx := context.Background()

	if convert {
		if err := mkvmetadata.ConvertToMkv(ctx, originPath, newPath); err != nil {
			pp.Printf("Warning: MP4 conversion failed for %s, moving as-is: %v\n", filepath.Base(originPath), err)
			service.Journal("convert", originPath, newPath, "failed", err.Error())
			// fall back to a plain move under the original extension
			convert = false
			extension = strings.ToLower(video.Extension)
			newPath, res.Skipped = service.ResolveCollision(plainNewPath, config.ON_COLLISION)
			if res.Skipped {
				return res, nil
			}
			media.SetNewPath(newPath)
			if err := service.MoveFile(originPath, newPath); err != nil {
				return res, fmt.Errorf("move file error for %s: %w", filepath.Base(originPath), err)
			}
			service.Journal("move", originPath, newPath, "done", "")
		} else {
			res.Converted = true
			service.Journal("convert", originPath, newPath, "done", "remux to mkv")
		}
	} else {
		if err := service.MoveFile(originPath, newPath); err != nil {
			return res, fmt.Errorf("move file error for %s: %w", filepath.Base(originPath), err)
		}
		service.Journal("move", originPath, newPath, "done", "")
	}

	if extension == "mkv" {
		if config.MERGE_SUBTITLE_SIDECARS {
			res.SubtitlesMerged = mkvmetadata.MergeSubtitleSidecars(ctx, originPath, newPath)
			if res.SubtitlesMerged > 0 {
				service.Journal("merge_sub", originPath, newPath, "done",
					fmt.Sprintf("%d subtitle(s) embedded", res.SubtitlesMerged))
			}
		}

		result, err := mkvmetadata.UpdateMkvMetadata(media)
		if err != nil && config.SALVAGE && !mkvmetadata.IsNotMatroska(err) {
			// Last resort: the normal path refused the file, keep what is
			// playable and retry once
			pp.Printf("Warning: MKV processing failed for %s: %v\n", filepath.Base(newPath), err)
			report, serr := mkvmetadata.SalvageFile(ctx, newPath)
			if serr != nil {
				service.Journal("salvage", newPath, "", "failed", serr.Error())
			} else {
				res.Salvaged = true
				service.Journal("salvage", newPath, "", "done",
					fmt.Sprintf("%d cluster(s) kept, %d byte(s) skipped, %d damaged range(s)",
						report.ClustersCopied, report.BytesSkipped, len(report.DamagedRanges)))
				result, err = mkvmetadata.UpdateMkvMetadata(media)
			}
		}
		if err != nil {
			pp.Printf("Warning: MKV metadata update failed for %s: %v\n", filepath.Base(newPath), err)
		} else {
			res.SeekIndexRepaired = strings.HasPrefix(result.MkvSeekIndex, "rebuilt")
			if res.SeekIndexRepaired {
				service.Journal("repair", newPath, "", "done", result.MkvSeekIndex)
			}
			if config.PLAYABILITY_TARGET != "" {
				if verdict, perr := mkvmetadata.CheckPlayability(ctx, newPath, config.PLAYABILITY_TARGET); perr != nil {
					pp.Printf("Warning: playability check failed for %s: %v\n", filepath.Base(newPath), perr)
				} else {
					result.MkvPlayability = verdict
				}
			}
			setMkvMetadata(media, result)
		}

		if config.WRITE_CONTENT_HASHES {
			if err := mkvmetadata.WriteHashes(ctx, newPath); err != nil {
				pp.Printf("Warning: content hashing failed for %s: %v\n", filepath.Base(newPath), err)
			} else {
				res.Hashed = true
				service.Journal("hash", newPath, "", "done", "CONTENT_SHA256 written")

				if dedupIndex != nil {
					checkDuplicate(ctx, media, newPath, &res)
				}
			}
		}
	}

	printMediaTable(media)

	return res, nil
}

// dryRunMedia reports what the real run would do: read-only analysis, no move,
// no write. The MKV analysis runs on the file at its current location.
func dryRunMedia(media types.Normalizable, originPath, newPath, extension string, convert bool) (mediaResult, error) {
	var res mediaResult
	res.SourceDir = filepath.Dir(originPath)
	res.PlannedGone = []string{originPath}

	if convert {
		pp.Printf("   would convert to MKV: %s\n", newPath)
		service.Journal("convert", originPath, newPath, "planned", "")
		res.Converted = true
	} else {
		pp.Printf("   would move to: %s\n", newPath)
		service.Journal("move", originPath, newPath, "planned", "")
	}

	if strings.ToLower(filepath.Ext(originPath)) == ".mkv" {
		if config.MERGE_SUBTITLE_SIDECARS {
			for _, sub := range mkvmetadata.FindSubtitleSidecars(originPath) {
				pp.Printf("   would merge subtitle: %s\n", filepath.Base(sub.Path))
				service.Journal("merge_sub", sub.Path, newPath, "planned", "")
				res.SubtitlesMerged++
				res.PlannedGone = append(res.PlannedGone, sub.Path)
			}
		}

		result, err := mkvmetadata.PlanMkvMetadata(media, originPath)
		if err != nil {
			pp.Printf("Warning: MKV analysis failed for %s: %v\n", filepath.Base(originPath), err)
		} else {
			res.SeekIndexRepaired = strings.HasPrefix(result.MkvSeekIndex, "would rebuild")
			if res.SeekIndexRepaired {
				service.Journal("repair", newPath, "", "planned", result.MkvSeekIndex)
			}
			if config.PLAYABILITY_TARGET != "" {
				if verdict, perr := mkvmetadata.CheckPlayability(context.Background(), originPath, config.PLAYABILITY_TARGET); perr == nil {
					result.MkvPlayability = verdict
				}
			}
			setMkvMetadata(media, result)
		}

		if config.WRITE_CONTENT_HASHES {
			res.Hashed = true
			service.Journal("hash", newPath, "", "planned", "")
		}
	}

	printMediaTable(media)

	return res, nil
}

// checkDuplicate flags a freshly hashed file whose content (identical track
// payloads, whatever the filename/metadata/track order) is already in the
// library. Report-only: the duplicate is named in the table and the journal,
// nothing is deleted.
func checkDuplicate(ctx context.Context, media types.Normalizable, path string, res *mediaResult) {
	fp, err := mkvmetadata.FingerprintFile(ctx, path)
	if err != nil || fp == "" {
		return
	}

	if existing, ok := dedupIndex.Lookup(fp); ok && existing != path {
		pp.Printf("   %s: same content as %s\n", filepath.Base(path), existing)
		service.Journal("duplicate", path, existing, "detected", "identical track payloads (fingerprint match)")
		res.Duplicate = true
		switch v := media.(type) {
		case *types.Serie:
			v.MkvMetadata.MkvDuplicateOf = existing
		case *types.Movie:
			v.MkvMetadata.MkvDuplicateOf = existing
		}
		return
	}

	if err := dedupIndex.Add(fp, path); err != nil {
		pp.Printf("Warning: dedup index write failed: %v\n", err)
	}
}

func setMkvMetadata(media types.Normalizable, result types.FileInfos) {
	switch v := media.(type) {
	case *types.Serie:
		v.MkvMetadata = result
	case *types.Movie:
		v.MkvMetadata = result
	}
}

func printMediaTable(media types.Normalizable) {
	printMu.Lock()
	service.PrintStructTable(media)
	pp.Println("________________________________________________________________________________")
	printMu.Unlock()
}

func printStats(stats ProcessStats, elapsed time.Duration) {
	if len(stats.Errors) > 0 {
		pp.Println("\nErrors encountered:")
		for _, err := range stats.Errors {
			pp.Printf("  - %v\n", err)
		}
	}

	totalCount := stats.MoviesCount + stats.SeriesCount
	if totalCount == 0 && stats.Skipped == 0 {
		pp.Println("No videos found")
		return
	}

	pp.Println("")
	if config.DRY_RUN {
		pp.Println("DRY RUN summary (nothing was modified):")
	}
	pp.Println("Movies processed:", stats.MoviesCount)
	pp.Println("Series processed:", stats.SeriesCount)
	if stats.Skipped > 0 {
		pp.Println("Skipped (collision):", stats.Skipped)
	}
	if stats.Converted > 0 {
		pp.Println("Converted to MKV:", stats.Converted)
	}
	if stats.SeekIndexRepaired > 0 {
		pp.Println("Seek indexes repaired:", stats.SeekIndexRepaired)
	}
	if stats.SubtitlesMerged > 0 {
		pp.Println("Subtitle sidecars merged:", stats.SubtitlesMerged)
	}
	if stats.Hashed > 0 {
		pp.Println("Files hashed:", stats.Hashed)
	}
	if stats.Salvaged > 0 {
		pp.Println("Damaged files salvaged:", stats.Salvaged)
	}
	if stats.Duplicates > 0 {
		pp.Println("Content duplicates detected:", stats.Duplicates)
	}
	if stats.JunkRemoved > 0 || stats.DirsRemoved > 0 {
		pp.Println("Cleanup: junk files:", stats.JunkRemoved, "- dirs:", stats.DirsRemoved)
	}
	pp.Println("Total videos processed:", totalCount)
	pp.Println("Elapsed:", elapsed.Round(time.Second).String())
	pp.Println("________________________________________________________________________________")
}
