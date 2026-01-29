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
- ✅ **MKV metadata**: Sets correct audio/subtitle tracks
- ✅ **Multi-language**: Supports 10+ languages
- ✅ **Parallel processing**: Fast batch operations

## 📋 Prerequisites

- **Go 1.21+** - [Install Go](https://go.dev/doc/install)
- **mkvtoolnix** (optional) - For MKV metadata updates

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

Edit `config/constants.go`:
```go
const (
    ORIGIN_PATH = "/home/user/Downloads/"     // Source folder
    DEST_PATH   = "/media/videos/"            // Destination folder
    
    // Recursive scanning
    RECURSIVE_SCAN = true   // false = only scan root folder
    
    // MKV Metadata Configuration
    PREFERRED_AUDIO_LANG    = "en"   // en, fr, de, es, ja, etc.
    FALLBACK_AUDIO_LANG     = ""     // Leave empty for no fallback
    PREFERRED_SUBTITLE_LANG = "en"   
    FALLBACK_SUBTITLE_LANG  = ""
    SUBTITLE_FORCED_ONLY    = true   // Only select forced subtitles
    
    MAX_WORKERS = 10  // Parallel processing workers
)
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

## 📖 Usage
```bash
# Process all videos in source folder
make start
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

When mkvtoolnix is installed:
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

### mkvtoolnix not found
```bash
# Ubuntu/Debian
sudo apt install mkvtoolnix

# macOS
brew install mkvtoolnix
```

### Permission denied errors
Ensure you have read/write permissions on source and destination folders.

### No files processed
Check that:
- File extensions match supported formats
- Files are not empty (size > 0)
- Source path is correct in `config/constants.go`

- `RECURSIVE_SCAN` is set appropriately
