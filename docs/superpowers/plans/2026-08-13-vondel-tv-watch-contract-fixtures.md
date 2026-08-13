# Vondel TV Watch Contract Fixtures Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Provide versioned invented Watch documents, deterministic synthetic media, and a disposable HTTP service that independently written TV clients can use without a live server.

**Architecture:** The contracts repository defines a test-only semantic Watch document over existing catalog, playback-plan, stream, and progress semantics. A Go fixture service serves those documents and a generated MP4; it is never linked into production clients.

**Tech Stack:** Go 1.26, JSON Schema 2020-12, OpenAPI 3.1, `net/http`, FFmpeg/ffprobe test tooling.

**Spec:** `vondel-server/docs/superpowers/specs/2026-08-13-vondel-tv-watch-vertical-slice-design.md`

## Global Constraints

- Work in `vondel-client-contracts`; commands assume that repository root is the cwd.
- Use only invented fixture titles, descriptions, images, tones, and identifiers.
- Do not add a production server endpoint or client UI/runtime code.
- The fixture service binds loopback by default and uses no real credentials.
- Production clients must not ship the fixture service, generated media, or fixture token.
- Live TV, IPTV, DVR, EPG, `.strm`, and arbitrary remote-stream shortcuts are prohibited.
- Follow strict red-green TDD and commit after every task.

---

### Task 1: Versioned Watch semantic documents

**Files:**
- Create: `schema/watch/document.schema.json`
- Create: `fixtures/watch/home_tv_v1.json`
- Create: `fixtures/watch/movie_4242_detail.json`
- Create: `fixtures/watch/series_8080_detail.json`
- Modify: `schema/contract_test.go`
- Modify: `ORIGINALITY.md`

**Interfaces:**
- Produces: `watch_document_v1` JSON with `snapshot`, `featured_content_id`, `items`, and `progress`.
- Produces: item kinds `movie`, `series`, and `episode`; episode identity includes `series_id`, `season_number`, `episode_number`, and `file_id`.
- Produces: progress fields `content_id`, optional `episode_id`, `position_seconds`, `duration_seconds`, `completed`, and `updated_at`.

- [ ] **Step 1: Write failing schema tests**

Add table rows to `TestInventedFixturesValidateAgainstVersionedSchemas` and explicit consistency assertions:

```go
{"watch home", "watch/document.schema.json", "watch/home_tv_v1.json"},
{"movie detail", "watch/document.schema.json", "watch/movie_4242_detail.json"},
{"series detail", "watch/document.schema.json", "watch/series_8080_detail.json"},
```

Add `TestWatchSeriesEpisodesHaveStableOrderingAndFileIDs`, requiring strictly increasing episode numbers within each season and positive unique `file_id` values.

- [ ] **Step 2: Verify RED**

Run: `go test ./schema -run 'TestInventedFixtures|TestWatchSeries' -count=1`

Expected: FAIL because `schema/watch/document.schema.json` and the three fixtures do not exist.

- [ ] **Step 3: Add the closed test document schema and invented fixtures**

The schema root is:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["schema", "snapshot", "items", "progress"],
  "properties": {
    "schema": {"const": "watch_document_v1"},
    "snapshot": {"type": "string", "format": "date-time"},
    "featured_content_id": {"type": "string", "minLength": 1},
    "items": {"type": "array", "items": {"$ref": "#/$defs/item"}},
    "progress": {"type": "array", "items": {"$ref": "#/$defs/progress"}}
  },
  "additionalProperties": true
}
```

Define `$defs.item` with closed known `kind` values but `additionalProperties: true`. Use existing invented titles `The Invented Crossing` and `Eight Quiet Rooms`; add wholly invented seasons and episodes under series `8080`.

- [ ] **Step 4: Verify GREEN and provenance**

Run: `go test ./schema -run 'TestInventedFixtures|TestWatchSeries' -count=1`

Expected: PASS.

Update `ORIGINALITY.md` with contract authority, original decisions, exact fixture paths, dependencies, vocabulary, implementer access, and pending audit evidence.

- [ ] **Step 5: Commit**

```bash
git add schema/watch schema/contract_test.go fixtures/watch ORIGINALITY.md
git commit -m "feat(contracts): add invented TV Watch documents"
```

### Task 2: Deterministic synthetic playback clip

**Files:**
- Create: `scripts/generate-watch-fixture.sh`
- Create: `internal/watchfixture/media_test.go`
- Modify: `.gitignore`
- Modify: `README.md`

**Interfaces:**
- Produces: `build/watch-fixture.mp4`, 1280x720 SDR H.264 with AAC stereo, generated entirely from FFmpeg filters.
- Produces: script arguments `--output PATH` and `--duration SECONDS`; default duration is 12 seconds.

- [ ] **Step 1: Write the failing generator test**

Create `TestGeneratedMediaIsSeekableH264AAC` that runs the script into `t.TempDir()`, invokes `ffprobe -v error -show_streams -show_format -of json`, and asserts MP4, one H.264 video stream, one AAC audio stream, positive duration, and no metadata title/artist.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/watchfixture -run TestGeneratedMedia -count=1`

Expected: FAIL because `scripts/generate-watch-fixture.sh` does not exist.

- [ ] **Step 3: Implement the generator**

Use a strict POSIX shell wrapper around:

```bash
ffmpeg -nostdin -hide_banner -loglevel error -y \
  -f lavfi -i "testsrc2=size=1280x720:rate=30:duration=${duration}" \
  -f lavfi -i "sine=frequency=440:sample_rate=48000:duration=${duration}" \
  -map 0:v:0 -map 1:a:0 -c:v libx264 -pix_fmt yuv420p -profile:v high \
  -g 60 -keyint_min 60 -sc_threshold 0 -c:a aac -b:a 128k -ac 2 \
  -movflags +faststart -map_metadata -1 "$output"
```

Reject missing FFmpeg, non-positive duration, and output outside the caller-supplied path. Ignore `build/` in Git.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/watchfixture -run TestGeneratedMedia -count=1`

Expected: PASS when FFmpeg/ffprobe are installed; otherwise the test must `t.Skip` with the missing executable named.

- [ ] **Step 5: Commit**

```bash
git add scripts/generate-watch-fixture.sh internal/watchfixture/media_test.go .gitignore README.md
git commit -m "test(fixtures): generate rights-clear Watch media"
```

### Task 3: Loopback Watch fixture service

**Files:**
- Create: `internal/watchfixture/server.go`
- Create: `internal/watchfixture/server_test.go`
- Create: `cmd/vondel-watch-fixture/main.go`
- Create: `cmd/vondel-watch-fixture/main_test.go`

**Interfaces:**
- Produces: `watchfixture.New(options Options) (http.Handler, error)`.
- Produces routes: `GET /fixtures/v1/watch/home`, `GET /fixtures/v1/watch/items/{id}`, `GET /api/v1/progress`, `POST /api/v1/playback/start`, `HEAD|GET /api/v1/playback/{session}/stream`, and `POST /api/v1/sync/progress`.
- Requires: `Authorization: Bearer fixture-watch-token` on every route except `GET /health`.

- [ ] **Step 1: Write failing HTTP tests**

Cover unauthorized `401`, invented document responses, playable plan resolution, `HEAD`, full `200`, `Range: bytes=5-14` returning exact `206` bytes and `Content-Range`, unsatisfiable range returning `416`, and progress sync echoing `updated` results.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/watchfixture ./cmd/vondel-watch-fixture -count=1`

Expected: FAIL because `watchfixture.New` and the command do not exist.

- [ ] **Step 3: Implement the handler and command**

Define:

```go
type Options struct {
    Fixtures fs.FS
    MediaPath string
    Now func() time.Time
}

func New(options Options) (http.Handler, error)
```

Decode playback start enough to require protocol `3`, positive `file_id`, non-empty `profile_id`, and a non-empty attempt ID. Return a copy of `fixtures/playback/plan_4242.json` with an unexpired `expires_at` derived from `Now`. Serve media with `http.ServeContent` so validators and ranges use standard behavior. The command defaults to `127.0.0.1:0` and prints one JSON startup record containing only its loopback origin and generated token label.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/watchfixture ./cmd/vondel-watch-fixture -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/watchfixture cmd/vondel-watch-fixture
git commit -m "feat(fixtures): serve disposable TV Watch contracts"
```

### Task 4: Negative playback and progress conformance cases

**Files:**
- Create: `fixtures/playback/plan_expired.json`
- Create: `fixtures/playback/plan_unsupported_codec.json`
- Create: `fixtures/playback/plan_malformed_timeline.json`
- Create: `fixtures/watch/progress_completed_4242.json`
- Modify: `schema/contract_test.go`
- Modify: `internal/watchfixture/server_test.go`
- Modify: `README.md`

**Interfaces:**
- Produces named negative fixtures for client validator tests.
- Produces completion examples for long content and content of two minutes or less.

- [ ] **Step 1: Add failing classification tests**

Add `TestPlaybackNegativeFixturesEachViolateOnlyTheirNamedRule` and `TestProgressCompletionFixtureUsesApprovedThreshold`. Decode a valid baseline, compare each negative fixture, and assert only expiry, codec, or timeline differs.

- [ ] **Step 2: Verify RED**

Run: `go test ./schema ./internal/watchfixture -run 'Negative|Completion' -count=1`

Expected: FAIL because the negative and completed fixtures do not exist.

- [ ] **Step 3: Add the fixtures and document their intended client result**

Record this closed matrix in `README.md`:

```text
plan_expired.json -> rejected: expired
plan_unsupported_codec.json -> rejected: unsupported video codec
plan_malformed_timeline.json -> rejected: invalid timeline
progress_completed_4242.json -> completed, resume position absent
```

- [ ] **Step 4: Verify GREEN and complete repository checks**

Run: `go test ./... -count=1`

Expected: PASS with no failures.

Run: `test -n "$FORBIDDEN_REFERENCE_ROOT" && git diff --check && go run ./cmd/originality-guard -candidate . -reference "$FORBIDDEN_REFERENCE_ROOT"`

Expected: diff check passes; the real reference comparison is performed only by the authorized auditor and its sanitized result is appended to `ORIGINALITY.md` before acceptance.

- [ ] **Step 5: Commit**

```bash
git add fixtures schema internal/watchfixture README.md ORIGINALITY.md
git commit -m "test(contracts): cover Watch playback rejection cases"
```

### Task 5: Contract acceptance checkpoint

**Files:**
- Modify: `README.md`
- Modify: `ORIGINALITY.md`

**Interfaces:**
- Produces the stable fixture names and disposable service behavior consumed by both client plans.

- [ ] **Step 1: Run the full evidence matrix**

```bash
go test ./... -count=1
go vet ./...
git diff --check
```

Expected: every command exits `0`.

- [ ] **Step 2: Exercise the service as an external process**

Generate the clip, start `go run ./cmd/vondel-watch-fixture`, fetch the home document with the fixture bearer token, request a byte range, and stop the process. Assert the returned range length and `206` status.

- [ ] **Step 3: Record exact evidence**

Add commands, tool versions, fixture hash, test counts, and the authorized sanitized originality result to `README.md` and `ORIGINALITY.md`. Do not record local absolute paths.

- [ ] **Step 4: Commit**

```bash
git add README.md ORIGINALITY.md
git commit -m "docs(contracts): record TV Watch fixture evidence"
```

- [ ] **Step 5: Publish the dependency gate**

Push the reviewed contracts branch or merge it according to the repository workflow. Record its commit hash in the Apple and Android task reports; do not begin client fixture integration against uncommitted contract shapes.
