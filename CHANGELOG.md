# Changelog

All notable changes to normalize_video are documented here. The format is
based on [Keep a Changelog](https://keepachangelog.com/), and the project
follows [Semantic Versioning](https://semver.org/).

## [0.2.0] - 2026-07-09

**Highlights**

- **In-place seek index repair** - the Cues rebuild no longer copies the file:
  the new index is appended inside the Segment and the SeekHead repointed,
  cluster bytes untouched. One read pass instead of a read+write of the whole
  file (was ~2 min per large file on a WSL drive mount, now roughly half).
- **Playability report** (`--playability chrome`) - know at import time what
  will direct-play, remux or transcode on the target player, virtually free.
- **Salvage** (`--salvage`) - damaged files are recovered instead of failed.
- **Duplicate detection** (`--dedup`) - a re-mux or renamed copy of content
  already in the library is flagged at import, for free on hashed files.

### Added

- **`--playability PROFILE`** (off by default): every processed MKV gets a
  direct-play / remux / transcode verdict against a playback capability
  profile (chrome, safari, firefox, chromecast-gen3, ...), with the blocking
  tracks and reasons named (`MkvPlayability` in the per-file table). Head-only
  metadata read, no decode - virtually free. Also reported in `--dry-run`.
- **`--salvage`** (off by default): when the normal MKV processing fails on a
  structurally damaged file (bad sector, truncated download), the intact
  metadata and clusters are carried over verbatim, damaged spans are skipped
  (listed in the journal: clusters kept, bytes skipped, damaged ranges), the
  seek index is rebuilt and the pipeline retries once. Mislabeled non-Matroska
  files are excluded from salvage.
- **`--dedup`** (off by default, implies `--hashes`): every import's
  content-identity hash (the mkvgo Fingerprint "Presentation": per-track
  payload digests, sorted content-wise, so filename/metadata/track order do
  not matter) is checked against `DEST/.normalize_fingerprints.jsonl` and
  recorded there. The hash is rebuilt from the `CONTENT_SHA256` tags written
  at import - one metadata read, no extra media scan (verified byte-identical
  to the full-read `matroska.Fingerprint` in tests). Duplicates are
  report-only: `MkvDuplicateOf` in the per-file table, a `duplicate` journal
  line, a summary count - nothing is deleted.

### Changed

- Seek index repair uses `matroska.ReindexInPlace` (mkvgo): surgical, needs
  write access to the file only, crash-safe - every byte about to be
  overwritten is journaled inside the file first, the result is verified
  (head-only) while the journal still allows a rollback, and an interrupted
  run self-recovers on the next attempt. The previous single-pass full
  rewrite (`EditMetadata`) remains as automatic fallback, and still handles
  the in-place metadata edit overflow case.
- `MkvSeekIndex` reports `rebuilt in place (was: ...)` for the new path;
  `rebuilt (was: ...)` still means the full-rewrite fallback ran.
- A file whose layout cannot hold a head-discoverable index
  (`ErrIndexNotHeadDiscoverable`) falls back to the full rewrite with an
  informational note instead of a warning - it is an expected layout case,
  and the in-place attempt rolls back byte-identical.
- mkvgo upgraded v0.16.0 -> v0.17.0.

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
