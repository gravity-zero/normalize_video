package config

import (
	"flag"
	"fmt"
	"os"
)

// Compile-time settings: naming rules and language preferences.
const (
	REGEXSERIES       = `\b[sS]\s*(\d{1,2})\s*[-._ ]*\s*[eE]\s*(\d{1,3})\b`
	REGEXSERIESEXTEND = `\b(\d{1,2})\s*[xX]\s*(\d{1,3})\b`

	PREFERRED_AUDIO_LANG    = "fr"
	PREFERRED_SUBTITLE_LANG = "fr"

	FALLBACK_AUDIO_LANG    = ""
	FALLBACK_SUBTITLE_LANG = ""

	SUBTITLE_FORCED_ONLY = true
)

// Runtime settings: defaults below, overridable via CLI flags (see ParseFlags).
var (
	ORIGIN_PATH    = "/mnt/e/DDL/"
	DEST_PATH      = "/mnt/e/Cartoon/"
	RECURSIVE_SCAN = true
	MAX_WORKERS    = 10

	// Show the full plan (moves, renames, MKV fixes, conversions) without
	// touching anything on disk
	DRY_RUN = false

	// Keep running: rescan ORIGIN_PATH periodically and process files once
	// their size is stable (safe on WSL /mnt/* where inotify does not work)
	WATCH                  = false
	WATCH_INTERVAL_SECONDS = 30

	// Rebuild the Cues index of MKV files that lack one (broken seeking in
	// players). Only affected files pay the cost: a full file copy
	REPAIR_SEEK_INDEX = true

	// Embed subtitle sidecars (movie.srt, movie.fr.srt, movie.fr.forced.ass)
	// into the MKV and delete the sidecar. Off by default: one full file
	// copy per merged subtitle
	MERGE_SUBTITLE_SIDECARS = false

	// Remux MP4-family files (mp4, m4v, mov) to MKV at the destination, no
	// re-encode, so the whole MKV pipeline applies to them. Off by default
	CONVERT_MP4 = false

	// Store a per-track SHA-256 (CONTENT_SHA256 tag) so files self-verify.
	// Off by default: one full read per file
	WRITE_CONTENT_HASHES = false

	// After processing, delete whitelisted junk files (nfo, jpg, txt...) from
	// the SOURCE folders a video was moved out of, then remove them if empty.
	// Never touches video files. Off by default
	CLEANUP_SOURCE = false

	// What to do when the destination file already exists: "" keeps the
	// historical behavior (silent overwrite), or one of skip / replace / suffix
	ON_COLLISION = ""

	// Append one JSON line per operation (move, repair, merge, convert...)
	// to this file. Empty = disabled
	JOURNAL_PATH = ""
)

var collisionModes = map[string]bool{"": true, "skip": true, "replace": true, "suffix": true}

// ParseFlags overrides the runtime settings from the command line.
func ParseFlags() {
	flag.StringVar(&ORIGIN_PATH, "origin", ORIGIN_PATH, "source folder to scan")
	flag.StringVar(&DEST_PATH, "dest", DEST_PATH, "destination folder")
	flag.BoolVar(&RECURSIVE_SCAN, "recursive", RECURSIVE_SCAN, "scan subfolders")
	flag.IntVar(&MAX_WORKERS, "workers", MAX_WORKERS, "parallel workers")
	flag.BoolVar(&DRY_RUN, "dry-run", DRY_RUN, "show the plan without touching anything")
	flag.BoolVar(&WATCH, "watch", WATCH, "keep running and process new files as they finish downloading")
	flag.IntVar(&WATCH_INTERVAL_SECONDS, "watch-interval", WATCH_INTERVAL_SECONDS, "seconds between watch scans")
	flag.BoolVar(&REPAIR_SEEK_INDEX, "repair-index", REPAIR_SEEK_INDEX, "rebuild missing/broken MKV Cues")
	flag.BoolVar(&MERGE_SUBTITLE_SIDECARS, "merge-subs", MERGE_SUBTITLE_SIDECARS, "embed .srt/.ass sidecars into the MKV")
	flag.BoolVar(&CONVERT_MP4, "convert-mp4", CONVERT_MP4, "remux mp4/m4v/mov to mkv (no re-encode)")
	flag.BoolVar(&WRITE_CONTENT_HASHES, "hashes", WRITE_CONTENT_HASHES, "write per-track CONTENT_SHA256 tags")
	flag.BoolVar(&CLEANUP_SOURCE, "cleanup", CLEANUP_SOURCE, "delete junk files and empty folders from emptied source dirs")
	flag.StringVar(&ON_COLLISION, "on-collision", ON_COLLISION, "when destination exists: skip, replace or suffix (default: overwrite)")
	flag.StringVar(&JOURNAL_PATH, "journal", JOURNAL_PATH, "append one JSON line per operation to this file")
	flag.Parse()

	if !collisionModes[ON_COLLISION] {
		fmt.Fprintf(os.Stderr, "invalid -on-collision value %q (want skip, replace or suffix)\n", ON_COLLISION)
		os.Exit(2)
	}
	if MAX_WORKERS < 1 {
		MAX_WORKERS = 1
	}
	if WATCH_INTERVAL_SECONDS < 1 {
		WATCH_INTERVAL_SECONDS = 1
	}
}

var Extensions = []string{
	"avi", "mkv", "mp4", "mpeg", "mpg",
	"mov", "wmv", "flv", "webm",
	"m4v", "3gp", "ogv",
	"ts", "mts", "m2ts",
}

// MP4Convertible lists the extensions -convert-mp4 can remux to MKV
// (MP4-family only: ts/m2ts are MPEG-TS, not ISO base media).
var MP4Convertible = map[string]bool{"mp4": true, "m4v": true, "mov": true}

var Qualities = []string{
	"480p", "720p", "1080p", "2160p", "4k", "8k", "uhd",
	"cam", "hdcam", "ts", "telesync", "screener",
	"dvdrip", "bdrip", "brrip", "hdtv",
	"web", "webdl", "web-dl", "webrip",
	"bluray", "remux",
}
