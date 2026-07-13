# Changelog

All notable changes to normalize_video are documented here. The format is
based on [Keep a Changelog](https://keepachangelog.com/), and the project
follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

Two false verdicts on the seek index, both of which had this pipeline rewrite
the index of files that seek perfectly well. Measured on the library: of 12
sampled files, 8 would have been needlessly reindexed; the corrected rule
leaves 428 of 462 alone and flags 34 whose video cues really do leave holes
(33s to 67s - a seek there lands that far from its target).

### Fixed

- **The index verdict now judges the video cues, not the presence of others.**
  A cue keyed on an audio track is inert: the keyframe index a player seeks
  with is built from the video-keyed cues and drops the rest. Yet a single
  audio cue condemned a file - and real muxers routinely cue *every* track, so
  files with a dense, perfectly seekable video index were reindexed for
  nothing (one library file: 1436 video cues, flagged over 2096 audio ones).
  `SeekIndexIssue` now reports `cues keyed on non-video track` only when NOT
  ONE cue keys on video (the defect it was built for - every seek lands
  mid-GOP), and gains `video cues leave a Ns hole` for an index too coarse to
  seek with (over 30s between consecutive video cues, or before the first /
  after the last).
- **The index verdict is read from the full parse, not the head-only triage.**
  Cues sitting at the tail with no SeekHead pointing at them are found by the
  bounded scan back from EOF - the layout many muxers produce, and one every
  player handles. The head-only check only follows SeekHead pointers, so it
  called those files unindexed (three of twelve sampled library files: 0 cues
  head-only, over 1500 video cues in reality) and had us rebuild an index they
  already had. The pipeline already pays for a full read, so the verdict now
  comes from there - the same truth a player sees. The triage keeps what the
  parse cannot give: cluster-stream damage and A/V start delays. (mkvgo fixes
  its head-only path too, but the verdict belongs on the data we already have.)
- **The post-edit index check no longer fires on files that never had a
  head-discoverable index.** It compares the head-only view before and after
  the edit: only an index the edit LOST is rebuilt.

## [0.3.0] - 2026-07-13

**Highlights**

- **Damaged files are repaired, not just salvaged** - `--salvage` now runs a
  surgical resync repair first (lying size fields corrected with zero loss,
  damage cut around block by block, video resumed at the next keyframe), and
  only falls back to the lossy best-effort salvage when that refuses.
- **"Repair" vs "re-download", said out loud** - damage running to the end of
  the file is an incomplete source, not corruption: no tool can restore bytes
  that were never written. The file reports `re-download advised` instead of
  pretending it was fixed.
- **A/V desync detected on every file, fixed on demand** (`--retime`) - the
  repack defect where audio starts a few hundred ms late is measured natively
  and cancelled without re-encoding.
- **MP4 files go through mkvgo too** - the same triage and the same retime
  repair, whether they are converted to MKV or kept as MP4.

### Added

- **Container-agnostic triage** (mkvgo `Diagnose`): every import is classified
  in one call - seek-index health, per-track A/V start delay, structural
  integrity - and each finding names its own repair, which is what the
  pipeline runs. The engine is chosen from the file's FIRST BYTES, never its
  extension, so a mislabeled file still routes correctly.
- **Surgical damage repair** (`--salvage`, reported in `MkvDamage`): a damaged
  cluster stream is repaired by one verified resync rewrite - lying size
  fields over intact payloads corrected losslessly, damage cut around block by
  block instead of dropping whole clusters (a repair typically loses a few KB
  where a plain skip would lose seconds of media), video resumed at the next
  keyframe after a gap (clean cut: a short jump instead of frames decoding
  into artifacts), Segment size sealed and seek index rebuilt in the same
  pass. The result is deep-verified against the source before replacing it:
  only a defect the repair ADDED can refuse it, so a correct repair is not
  blocked by a defect the file already carried. The uncapped best-effort
  salvage now only runs when this refuses (mostly-damaged source).
- **Truncated-tail verdict**: damage reaching the end of the file marks the
  source incomplete - the playable prefix is kept, and the table and journal
  say `re-download advised`.
- **`--retime`** (off by default, reported in `MkvAudioSync` on every file):
  each audio track's start is measured against the first video keyframe, and a
  delay >= 100ms is cancelled by shifting the audio - mkvgo picks between a
  2-bytes-per-block in-place patch (crash-safe journal) and a verified
  rewrite, then re-reads the result to check every shifted track moved by
  exactly the requested amount. No re-encode. Without the flag the delay is
  only reported, with the flag that would fix it.
- **MP4 files use the mkvgo pipeline**: kept as MP4, they get the same
  head-only triage (box-layout truncation, missing `moov`, edit-list audio
  delays) and the same opt-in retime, which re-bases the `moov` edit list - no
  sample touched, a few bytes whatever the file size. Converted
  (`--convert-mp4`), they are triaged BEFORE the remux: a truncated or
  `moov`-less source has nothing a remux could carry over cleanly, so the
  conversion is skipped, the file is moved as-is and the table says why.
- **Interrupted-repair recovery**: trailing bytes past the declared Segment
  end can be the crash journal of an in-place repair killed mid-write. The
  file is rolled back to its pre-repair bytes and re-analyzed before anything
  else touches it.
- **Journal**: new `repair_damage`, `retime` and `diagnose` operations; the
  salvage line now reports the losslessly rebuilt regions and the truncated
  tail. New summary counters: damaged files repaired, A/V desyncs retimed.

### Fixed

- **The metadata edit could leave the seek index unreachable.** The in-place
  edit rewrites the head region, and metadata grown past the SeekHead's
  reserved slot left no room for it: the Cues element stayed in the file with
  nothing pointing at it - an index that exists but that no head-only reader
  (that is, any player) can find. Found while testing the repair path, fixed
  upstream in mkvgo 0.21.1 (`EditInPlace` now sizes the SeekHead slot to fit,
  or refuses the edit rather than trading the index for it). Two things stay
  on this side: the index is re-checked head-only (milliseconds) after every
  edit and rebuilt if it ever goes missing again - the operation reported
  success while corrupting the file once, so the invariant is asserted where
  it matters rather than assumed - and on a damaged file the metadata edit
  runs BEFORE the repair rewrite, so the index that rewrite builds is the one
  the file keeps.

### Changed

- mkvgo upgraded v0.17.0 -> v0.21.1. Beyond the features above, 0.21.1 closes
  six paths that reported success while leaving the file wrong; two are on
  this pipeline's route - the `EditInPlace` index loss above, and a stale Cues
  span of certain exact sizes that made `ReindexInPlace` void one byte too
  far, break the element chain, and commit it through its journal as a
  success.
- `SeekIndexIssue` heuristics are now backed by the mkvgo triage
  (`CueHealth`), which also spots cues referencing tracks that no longer
  exist (a stale index) - the head-only check the local heuristic missed.

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
