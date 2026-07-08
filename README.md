# Normalize Video

**Normalize Video** is a CLI tool to automatically organize and standardize your video library. It renames files, creates folder structures, and updates MKV metadata.

## 🎯 What It Does

Transform messy video files into a clean, organized library:

### Movies
```
Before: big.buck.bunny.2008.1080p.bluray.x264.mkv
After:  Big Buck Bunny - 1080P.mkv
Location: /destination/Movie/Big Buck Bunny - 1080P.mkv
```

### TV Series
```
Before: blender.studio.s01e01.spring.1080p.web.h264.mkv
After:  Blender Studio S01E01 - 1080P.mkv
Location: /destination/Serie/Blender Studio/S01/Blender Studio S01E01 - 1080P.mkv
```

### What Gets Organized
- ✅ **Automatic detection**: Movies vs TV Series
- ✅ **Smart naming**: Extracts title, season, episode, quality, language
- ✅ **Folder structure**: Auto-creates organized directories
- ✅ **Recursive scanning**: Processes all subfolders
- ✅ **MKV metadata**: Sets correct audio/subtitle tracks (default + forced flags)
- ✅ **Seek index repair**: Rebuilds missing or broken Cues so seeking works in any player (VLC, evey, Kodi, ...)
- ✅ **Multi-language**: Supports 10+ languages
- ✅ **Parallel processing**: Fast batch operations

## 📋 Prerequisites

- **Go 1.21+** - [Install Go](https://go.dev/doc/install)

## 🚀 Quick Start
```bash
# Install dependencies
make init

# Run
make start

# Build
make build
./normalize_video
```

## ⚙️ Configuration

Defaults live in `config/constants.go` (language preferences are compile-time),
everything else is overridable from the command line:

```
./normalize_video [flags]

  --origin PATH          source folder to scan          (default /mnt/e/DDL/)
  --dest PATH            destination folder             (default /mnt/e/Cartoon/)
  --recursive            scan subfolders                (default true)
  --workers N            parallel workers               (default 10)

  --dry-run              show the full plan (moves, renames, MKV fixes,
                         conversions, cleanup) without touching anything
  --watch                keep running: rescan origin periodically and process
                         files once their size is stable (WSL-safe polling,
                         inotify does not fire on /mnt/* drvfs mounts)
  --watch-interval N     seconds between watch scans    (default 30)

  --repair-index         rebuild missing/broken MKV Cues (default true)
  --merge-subs           embed .srt/.ass sidecars into the MKV     (default off)
  --convert-mp4          remux mp4/m4v/mov to mkv, no re-encode    (default off)
  --hashes               write per-track CONTENT_SHA256 tags       (default off)
  --cleanup              delete junk (nfo, jpg, txt...) and empty folders from
                         emptied source dirs - never touches video files (default off)
  --on-collision MODE    when destination exists: skip, replace or suffix
                         (default: historical silent overwrite)
  --journal PATH         append one JSON line per operation (move, repair,
                         merge, convert, hash, cleanup) to PATH    (default off)
```

MKV language preferences (compile-time, `config/constants.go`):
```go
PREFERRED_AUDIO_LANG    = "fr"
PREFERRED_SUBTITLE_LANG = "fr"
FALLBACK_AUDIO_LANG     = ""
FALLBACK_SUBTITLE_LANG  = ""
SUBTITLE_FORCED_ONLY    = true
```

### Recursive Scanning

When `RECURSIVE_SCAN = true`, all subfolders are processed:
```
/Downloads/
  ├── movie1.mkv           ← Processed
  ├── Movies/
  │   └── movie2.mkv       ← Processed
  └── Series/
      └── Show/
          └── episode.mkv  ← Processed
```

When `RECURSIVE_SCAN = false`, only files in the root folder are processed.

### MKV Handling (powered by [mkvgo](https://github.com/gravity-zero/mkvgo))

For every `.mkv`, in addition to renaming:
- **Title**: the container title is set to the normalized name
- **Default tracks**: the preferred audio and subtitle tracks get the default flag, so players pick them automatically
- **Forced flag**: a subtitle selected as forced (by name like "Forcés" or by container flag) gets the real `FlagForced` written, so players honor it without guessing from track names
- **Seek index**: files with a missing Cues index, or cues keyed on a non-video track, are reindexed (`MkvSeekIndex` reports `ok` or `rebuilt (...)`). Without this, seeking/scrubbing is broken or imprecise in most players
- **Mislabeled files**: a `.mkv` that is really an MP4-family container is detected and skipped with a clear warning instead of corrupting it
- **Subtitle sidecars** (opt-in, `MERGE_SUBTITLE_SIDECARS`): external `movie.srt` / `movie.fr.srt` / `movie.fr.forced.ass` files are embedded into the MKV with the right language and forced flag, then deleted - the library stays self-contained

Metadata edits are done in place (instant, no file copy). When the seek index must be rebuilt, the metadata edit and the reindex share a single read+write pass (one file copy, progress logged every 25%). Set `REPAIR_SEEK_INDEX = false` to skip the copy entirely (`MkvSeekIndex` then reports the issue instead of fixing it).

### Cleanup safety rules (`--cleanup`)

Cleanup only ever considers the source folders a video was actually moved out
of, and refuses to act when:
- the folder is the origin root itself (or outside it)
- ANY video file remains anywhere underneath
- any non-whitelisted file is present (archives, subtitles, executables...)

Only whitelisted junk is deleted (`nfo, txt, jpg, jpeg, png, gif, sfv, md5,
url, website, torrent`), then empty folders are removed bottom-up. Video files
are never deleted, whatever their name.

### Content hashes (`--hashes`)

Each track's payload SHA-256 is stored as a `CONTENT_SHA256` tag inside the
MKV: the file carries its own integrity proof (bit-rot, bad copies). Verify
later with `matroska.VerifyContentHashes` (or `mkvgo verify`). Costs one full
read per file (~7s/GB on a WSL drive mount); the tag write itself is in-place
and instant. Idempotent.

## 📖 Usage
```bash
# Preview everything first
./normalize_video --dry-run

# Process all videos in source folder
make start

# Watch mode: process new downloads as they complete
./normalize_video --watch --cleanup --journal ~/normalize.jsonl
```

### Output Example
```
+------------------------------+-------------------------------------------------------+
| KEY                          | VALUE                                                 |
+------------------------------+-------------------------------------------------------+
| Episode                      | E01                                                   |
| MkvMetadata.MkvAudioTrack    | english ac3 5.1                                       |
| MkvMetadata.MkvSubTrack      | english (forced)                                      |
| Normalizer.Title             | Sintel                                                |
| SE                           | S01E01                                                |
| Video.Language               | en                                                    |
| Video.Quality                | 1080p                                                 |
+------------------------------+-------------------------------------------------------+

Movies processed: 5
Series processed: 12
Total videos processed: 17
```

## 📁 File Organization

### Movies
```
/destination/Movie/
  ├── Big Buck Bunny - 1080P.mkv
  ├── Sintel - 720P.mkv
  └── Elephants Dream - 4K.mkv
```

### TV Series
```
/destination/Serie/
  └── Blender Studio/
      ├── S01/
      │   ├── Blender Studio S01E01 - 1080P.mkv
      │   └── Blender Studio S01E02 - 720P.mkv
      └── S02/
          └── Blender Studio S02E01 - 4K.mkv
```

## 🎬 Supported Formats

### Video Extensions
```
avi, mkv, mp4, mpeg, mpg, mov, wmv, flv, webm, m4v, 3gp, ts, mts, m2ts
```

### Quality Detection
```
480p, 720p, 1080p, 2160p, 4k, 8k, uhd
web, webdl, web-dl, webrip
bluray, bdrip, dvdrip, hdtv
```

### Language Support
```
French:     vf, vff, vfi, french, truefrench
English:    vo, english, en
German:     german, deutsch, de
Spanish:    spanish, es
Italian:    italian, it
Japanese:   japanese, ja
Portuguese: portuguese, pt
Russian:    russian, ru
Chinese:    chinese, zh
Arabic:     arabic, ar
Multi:      multi
```

## 🔧 MKV Metadata Management

MKV metadata is handled natively via [mkvgo](https://github.com/gravity-zero/mkvgo) — no external tools required.

- Updates video title
- Sets preferred audio track (based on config)
- Sets preferred subtitle track (forced only by default)

**Example:**
```
File: movie.mkv with tracks:
  - Audio 1: English
  - Audio 2: French
  - Audio 3: German
  - Subtitle 1: English
  - Subtitle 2: French (forced)

Config: PREFERRED_AUDIO_LANG = "fr"
Result: French audio & French forced subtitles set as default
```

## 📝 Examples

### Example 1: Basic Movie
```
Input:  big.buck.bunny.2008.1080p.h264.mkv
Output: Big Buck Bunny - 1080P.mkv
Path:   /destination/Movie/Big Buck Bunny - 1080P.mkv
```

### Example 2: Series with Language
```
Input:  sintel.s01e01.french.720p.web.mkv
Output: Sintel S01E01 - FRENCH - 720P.mkv
Path:   /destination/Serie/Sintel/S01/Sintel S01E01 - FRENCH - 720P.mkv
```

### Example 3: Year-Based Title
```
Input:  elephants.dream.2006.4k.bluray.mkv
Output: Elephants Dream - 4K.mkv
Path:   /destination/Movie/Elephants Dream - 4K.mkv
```

### Example 4: Alternate Series Format
```
Input:  spring.1x05.1080p.mkv
Output: Spring S01E05 - 1080P.mkv
Path:   /destination/Serie/Spring/S01/Spring S01E05 - 1080P.mkv
```

### Example 5: Recursive Scan
```
Input:  /Downloads/subfolder/nested/movie.mkv
Output: Movie - 1080P.mkv
Path:   /destination/Movie/Movie - 1080P.mkv
```

## 🆘 Troubleshooting

### Permission denied errors
Ensure you have read/write permissions on source and destination folders.

### No files processed
Check that:
- File extensions match supported formats
- Files are not empty (size > 0)
- Source path is correct in `config/constants.go`

- `RECURSIVE_SCAN` is set appropriately
