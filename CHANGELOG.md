# Changelog

All notable changes to normalize_video are documented here. The format is
based on [Keep a Changelog](https://keepachangelog.com/), and the project
follows [Semantic Versioning](https://semver.org/).

## [0.1.0] - 2026-07-08

First tagged release.

**Highlights**

- **Runtime CLI** - every behavior is now a flag (`--origin`, `--dest`,
  `--dry-run`, `--watch`, ...), no recompile needed; `--dry-run` previews the
  full plan (moves, MKV fixes, conversions, cleanup) without touching a file.
- **MKV pipeline on mkvgo v0.16** - seek index repair, real forced/default
  flags, subtitle sidecar embedding, lossless MP4 to MKV conversion,
  self-verifying content hashes.
- **Watch mode built for WSL** - polling with size-stability detection, since
  inotify never fires for Windows-side writes on `/mnt/*` drvfs mounts.

### Added

- **CLI flags** (`config.ParseFlags`): `--origin`, `--dest`, `--recursive`,
  `--workers`, plus one flag per feature below. Compile-time config keeps only
  the language preferences and naming regexes.
- **`--dry-run`**: read-only analysis of what a real run would do, per file:
  destination path, subtitle sidecars that would merge, seek index that would
  rebuild (`would rebuild (missing Cues)`), planned conversions and a faithful
  cleanup preview (files whose move is planned are simulated absent).
- **Seek index repair** (`--repair-index`, on by default): an MKV without a
  Cues index, or with cues keyed on a non-video track, seeks slowly or not at
  all in players. The metadata edit and the reindex share a single read+write
  pass (`EditMetadata` rebuilds SeekHead + Cues while rewriting); healthy
  files are edited in place, instantly. Progress logged every 25%.
- **Track flags**: the preferred audio/subtitle tracks get `FlagDefault`, and
  a subtitle selected as forced (by name or container flag) gets a real
  `FlagForced` written - players stop guessing from track names. Language
  matching now uses the BCP47 field (`fr-CA` matches `fr`).
- **Subtitle sidecars** (`--merge-subs`, off): `movie.srt`, `movie.fr.srt`,
  `movie.fr.forced.ass` next to the source are embedded into the MKV with the
  right ISO 639-2 language and a forced-aware track name, then deleted.
- **MP4 conversion** (`--convert-mp4`, off): `mp4`/`m4v`/`mov` are remuxed to
  MKV straight to their normalized destination (move + convert in one pass,
  no re-encode), source removed on success, plain move as fallback.
- **Content hashes** (`--hashes`, off): per-track SHA-256 stored as
  `CONTENT_SHA256` tags - the file carries its own integrity proof,
  verifiable later with `matroska.VerifyContentHashes`. One full read per
  file; the tag write is in-place and instant. Idempotent.
- **Source cleanup** (`--cleanup`, off): after a video moves out, its source
  folder is swept - whitelisted junk only (`nfo, txt, jpg, jpeg, png, gif,
  sfv, md5, url, website, torrent`), then empty dirs bottom-up. Hard refusals:
  the origin root, any folder still holding a video file anywhere beneath it,
  any folder with a non-whitelisted file. Video files are never deleted.
- **Collision modes** (`--on-collision`, off): `skip`, `replace` or `suffix`
  (`Movie - 1080P (1).mkv`) when the destination already exists.
- **Operations journal** (`--journal path`, off): one JSON line per operation
  (`move`, `convert`, `merge_sub`, `repair`, `hash`, `cleanup`, `collision`)
  with from/to and `done`/`planned`/`failed`/`skipped` status.
- **Watch mode** (`--watch`, `--watch-interval`): periodic rescans of the
  origin; a file is processed once size+mtime are stable across two scans
  (download finished). Failed files are not retried in a loop; Ctrl+C stops
  cleanly and prints the totals.
- **Progressive output**: a `->` line the moment a worker picks a file,
  25%-step progress on long copies, per-file tables serialized across
  workers, and a final summary (counts per feature, elapsed time).
- **Mislabeled files**: a `.mkv` that is really an MP4-family container is
  detected (`ErrNotMatroska`) and skipped with a clear warning.

### Fixed

- Destination collisions no longer overwrite silently when a collision mode
  is set (`os.Rename` used to clobber the existing file without a trace).
- `pp.Printf` mangles `%d` arguments (it colorizes them to strings);
  percentages now render correctly.

### Changed

- mkvgo upgraded v0.1.0 -> v0.16.0 (stale v0.1.0 go.sum hash purged).
