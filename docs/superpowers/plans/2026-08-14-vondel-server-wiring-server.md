# Vondel Server Client-Wiring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Serve everything the TV clients need from `/api/v1` — stable server identity, an aggregate capability document, Watch home and detail documents, contract-correct progress sync results, an operator-controlled local-network broadcast, and a complete pairing approval surface.

**Architecture:** New handlers live in `internal/api/handlers` beside their peers and mount inside the existing `/api/v1` route tree. Watch composition lives in a new `internal/watchdoc` package that joins catalog, episode, file, and progress reads into the contracts' `watch_document_v1` shape. The mDNS responder lives in `internal/discovery` behind an interface, is started from `cmd/silo`, and reads its setting live.

**Tech Stack:** Go 1.26, Chi v5, pgx v5, Goose SQL migrations, PostgreSQL 16, the existing auth/access/catalog/userstore/playback packages.

**Spec:** `docs/superpowers/specs/2026-08-14-vondel-client-server-wiring-design.md`

## Global Constraints

- Work in `vondel-server`; commands assume that repository root is the cwd.
- Additive within `/api/v1`: no renamed or removed response field, no changed field type, no repurposed status code. The one value correction in Task 3 is recorded in the pre-lock removals table in `docs/architecture/v1-scope.md` in the same commit that makes it.
- New features are detected through capability endpoints, never through version sniffing.
- Migrations are Goose SQL created with `make migrate-create NAME=...`; never run `goose fix`, never create paired up/down files, never renumber legacy migrations.
- Encrypted `server_settings` rows are GCM-bound to their key name; the new settings in this plan are plain rows and must not be created through the encrypting decorator.
- Watch documents must validate against the contracts repository's `schema/watch/document.schema.json` byte-for-byte; the schema is the authority, not this plan's prose.
- Do not implement Live TV, IPTV, DVR, EPG, `.strm`, or arbitrary remote-stream shortcuts.
- No real hostname, origin, account, password, token, or pairing code appears in code, tests, fixtures, or documentation.
- Follow strict red-green TDD and commit after every task.
- Conventional Commit subjects; one concern per commit.

---

### Task 1: Server identity and aggregate capabilities

**Files:**
- Create: `migrations/sql/<timestamp>_add_server_instance_id.sql`
- Create: `internal/api/handlers/server_identity.go`
- Create: `internal/api/handlers/server_identity_test.go`
- Create: `internal/api/handlers/capabilities.go`
- Create: `internal/api/handlers/capabilities_test.go`
- Modify: `internal/api/router.go`
- Modify: `docs/architecture/v1-scope.md`

**Interfaces:**
- Produces `GET /api/v1/server/identity` (public) returning `server_id`, `server_name`, `api_versions`, and `setup_complete`.
- Produces `GET /api/v1/capabilities` (public) returning `schema_version`, `media_types`, and `features`.
- Produces a `server.instance_id` plain server setting, generated once and never regenerated.
- Leaves `GET /api/v1/health` untouched, including its Jellyfin-compat-sourced identity fields.

- [ ] **Step 1: Write failing identity and capability tests**

Assert that identity answers without authentication, that `server_id` is a non-empty stable value that does not change across two handler constructions backed by the same settings row, that a pre-existing row is reused rather than replaced, and that `api_versions` contains `1`. Assert that capabilities lists the media types the deployment serves and that `features` contains `watch_document_v1`, `device_pairing_v1`, `progress_sync_v1`, and the existing playback and events tokens, with no duplicates.

```go
func TestServerIdentityIsStableAndPublic(t *testing.T) {
    settings := newFakeSettings(t)
    first := performJSONRequest(t, routerWith(t, settings), http.MethodGet, "/api/v1/server/identity", "", "", nil)
    second := performJSONRequest(t, routerWith(t, settings), http.MethodGet, "/api/v1/server/identity", "", "", nil)
    if first.Status != http.StatusOK || first.Body["server_id"] != second.Body["server_id"] {
        t.Fatalf("server identity is not stable: %v then %v", first, second)
    }
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/api/handlers -run 'ServerIdentity|Capabilit' -count=1`

Expected: FAIL because neither handler exists.

- [ ] **Step 3: Implement identity, capabilities, and the setting**

Create the migration with `make migrate-create NAME=add_server_instance_id` and insert nothing: the identifier is minted on first read through `SetIfAbsent` so a fresh install and an upgraded install behave the same. Generate it as a random UUID. Mount both routes inside the `/api/v1` group ahead of the auth routes, with no auth middleware.

Record the identity endpoint's relationship to health in a comment: health keeps its existing fields and its existing source, and nothing reads the new setting into health.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/api/handlers ./internal/api -run 'ServerIdentity|Capabilit' -count=1 && make migrate-validate`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add migrations/sql internal/api/handlers/server_identity.go internal/api/handlers/server_identity_test.go internal/api/handlers/capabilities.go internal/api/handlers/capabilities_test.go internal/api/router.go
git commit -m "feat(api): advertise server identity and capabilities"
```

### Task 2: Watch home and detail documents

**Files:**
- Create: `internal/watchdoc/document.go`
- Create: `internal/watchdoc/compose.go`
- Create: `internal/watchdoc/compose_test.go`
- Create: `internal/api/handlers/watch.go`
- Create: `internal/api/handlers/watch_test.go`
- Modify: `internal/api/router.go`

**Interfaces:**
- Produces `GET /api/v1/watch/home` and `GET /api/v1/watch/items/{content_id}`, both profile-scoped behind `RequireProfile` and viewer access.
- Produces `watchdoc.ComposeHome(ctx, Reader, ProfileScope) (Document, error)` and `watchdoc.ComposeItem(ctx, Reader, ProfileScope, contentID) (Document, error)`.
- Produces documents whose `schema` is `watch_document_v1`, whose episode items always carry `series_id`, `season_number`, `episode_number`, and `file_id`, and whose `progress` rows carry `episode_id` for episodes.

- [ ] **Step 1: Write failing composition and endpoint tests**

Validate every produced document against the contracts schema through a test helper that locates the contracts checkout by environment variable and falls back to the standard adjacent checkout; skip with the missing variable named rather than passing when it cannot be found. Cover:

- a home document listing movies and series with a featured content identifier drawn from a deterministic server rule;
- a home document whose `progress` rows are exactly the profile's rows for items present in `items`;
- a detail document for a series carrying seasons and episodes in ascending season then episode order, each with a positive unique `file_id`;
- a detail document for a movie carrying a `file_id`;
- a request without the profile header answering `400` with the existing `profile_required` vocabulary;
- a profile with library restrictions never receiving a restricted item in `items` or `progress`;
- an unknown content identifier answering `404`.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/watchdoc ./internal/api/handlers -run Watch -count=1`

Expected: FAIL because `internal/watchdoc` and the handler do not exist.

- [ ] **Step 3: Implement composition and handlers**

Define the reader boundary as a narrow interface over the existing catalog, episode, media-file, and progress stores rather than importing their concrete types, so composition is unit-testable without a database:

```go
type Reader interface {
    Items(ctx context.Context, scope ProfileScope) ([]Item, error)
    Item(ctx context.Context, scope ProfileScope, contentID string) (Item, bool, error)
    Episodes(ctx context.Context, scope ProfileScope, seriesID string) ([]Episode, error)
    FilesByContentIDs(ctx context.Context, contentIDs []string) (map[string]int64, error)
    Progress(ctx context.Context, scope ProfileScope, contentIDs []string) ([]Progress, error)
}
```

Emit the document with `encoding/json` field names matching the schema exactly. Choose the featured item by a single closed rule — the most recently added unwatched movie, falling back to the first item in deterministic order — and document that the client's own presentation rules decide what to do with it. Omit an item that has no playable file rather than emitting a zero `file_id`.

Mount both routes inside the authenticated, profile-scoped group next to the progress routes.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/watchdoc ./internal/api/handlers ./internal/api -run Watch -count=1`

Expected: PASS, with schema validation performed against the contracts schema rather than hand-written field lists.

- [ ] **Step 5: Commit**

```bash
git add internal/watchdoc internal/api/handlers/watch.go internal/api/handlers/watch_test.go internal/api/router.go
git commit -m "feat(watch): serve Watch documents over v1"
```

### Task 3: Contract-correct progress sync results

**Files:**
- Modify: `internal/api/handlers/progress.go`
- Modify: `internal/api/handlers/progress_test.go`
- Modify: `docs/architecture/v1-scope.md`

**Interfaces:**
- Produces `results[].status` values `updated`, `ignored`, and `error` only, matching the contracts' enum.
- `ignored` names a row the minimum-resume floor discarded; `updated` names a row that was written; `error` keeps its existing meaning and message field.

- [ ] **Step 1: Write the failing vocabulary test**

Assert that a row above the floor answers `updated`, a row below the minimum-resume floor answers `ignored` rather than a success indistinguishable from a write, an empty media item identifier answers `error` with its existing message, and that no response contains the string `"ok"` in a status position.

```go
func TestSyncProgressUsesContractStatusVocabulary(t *testing.T) {
    resp := performJSONRequest(t, router, http.MethodPost, "/api/v1/sync/progress", belowFloorAndAboveFloorBody, token, profileHeaders)
    if got := statuses(resp); !reflect.DeepEqual(got, []string{"ignored", "updated"}) {
        t.Fatalf("statuses = %v, want [ignored updated]", got)
    }
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/api/handlers -run SyncProgress -count=1`

Expected: FAIL because the handler answers `ok` for both written and skipped rows.

- [ ] **Step 3: Implement the vocabulary and record the pre-lock correction**

Thread the existing `skip` result out of `ResolveProgressState` into the response so a discarded row is reported as discarded. Keep the write paths, the threshold logic, the event publication, and the profile refresh unchanged — this task changes reporting only.

Add a row to the pre-lock removals table in `docs/architecture/v1-scope.md` naming the `ok` status value, this design as its rationale, and the fact that a client cannot otherwise distinguish a write from a discard. The correction must ship before v1 locks.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/api/handlers -run 'Progress|Sync' -count=1`

Expected: PASS, including the existing progress tests unchanged in every respect but the status strings.

- [ ] **Step 5: Commit**

```bash
git add internal/api/handlers/progress.go internal/api/handlers/progress_test.go docs/architecture/v1-scope.md
git commit -m "fix(progress): report contract sync statuses"
```

### Task 4: Local-network discovery broadcast

**Files:**
- Create: `internal/discovery/responder.go`
- Create: `internal/discovery/record.go`
- Create: `internal/discovery/record_test.go`
- Create: `internal/discovery/responder_test.go`
- Create: `migrations/sql/<timestamp>_add_discovery_settings.sql`
- Modify: `cmd/silo/main.go`
- Modify: `go.mod`
- Modify: `go.sum`

**Interfaces:**
- Produces `discovery.Responder` with `Start(ctx) error`, `Stop()`, and `Apply(Settings)` so a setting change takes effect without a restart.
- Produces `discovery.TXT(Identity) []string` building the record from `txtvers`, `id`, `name`, `api`, and `scheme` and nothing else.
- Consumes a `discovery.mdns_enabled` plain server setting defaulting to enabled, and the server identity from Task 1.
- Registers service type `_vondel._tcp` in `local.` on private and link-local interfaces only.

- [ ] **Step 1: Write failing record and lifecycle tests**

Test the record builder and the lifecycle rules without touching the network by putting the multicast registration behind a fake:

```go
func TestTXTRecordCarriesNoBuildOrHouseholdIdentity(t *testing.T) {
    record := discovery.TXT(discovery.Identity{ServerID: "fixture-id", ServerName: "Invented Server", Scheme: "https", APIVersions: []int{1}})
    assertKeys(t, record, []string{"txtvers", "id", "name", "api", "scheme"})
    assertAbsent(t, record, []string{"version", "build", "commit", "libraries", "users", "profiles", "tenant"})
}
```

Cover: the responder does not register while the setting is disabled; it does not register while setup is incomplete; it deregisters and re-registers when the setting flips, with no restart; it filters out interfaces that are neither private nor link-local; and `Stop` is idempotent.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/discovery -count=1`

Expected: FAIL because the package does not exist.

- [ ] **Step 3: Implement the responder**

Add the mDNS dependency with `go get github.com/libp2p/zeroconf/v2` and pin whatever version resolves; record the module path, resolved version, and license in the PR body, and fall back to a vendored minimal responder if the module is unacceptable at review. Keep the dependency behind the `Responder` interface so no other package imports it.

Create the setting migration with `make migrate-create NAME=add_discovery_settings`. Start the responder from `cmd/silo` after the settings repository and server identity are available, subscribe it to setting changes, and stop it on shutdown before the listener closes.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/discovery ./cmd/silo -count=1 && make migrate-validate && go build ./...`

Expected: PASS and a clean build.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum internal/discovery migrations/sql cmd/silo/main.go
git commit -m "feat(discovery): broadcast the server on the local network"
```

### Task 5: Television pairing purpose and approval surface

**Files:**
- Modify: `internal/auth/device_login.go`
- Modify: `internal/api/handlers/auth_device.go`
- Modify: `internal/api/handlers/auth_device_test.go`
- Modify: `web/src/pages/ActivateDevice.tsx`
- Create: `web/src/pages/ActivateDevice.test.tsx`
- Modify: `web/src/App.tsx`

**Interfaces:**
- Produces a television client purpose accepted by `POST /api/v1/auth/device/start` and enforced by `POST /api/v1/auth/device/approve`, distinct from the remote-playback-handoff purpose.
- Produces a device capability response advertising that pairing is available for the television purpose.
- Produces a signed-in navigation entry point to the approval page so it is reachable without typing its URL.

- [ ] **Step 1: Write failing purpose and approval tests**

Server-side, assert that a start request carrying the television purpose returns a user code, a match code, a verification URI, an expiry, and an interval; that approving a television request through the handoff route fails with the existing purpose-mismatch vocabulary and vice versa; that denial, expiry, and reuse each answer their existing distinct statuses; and that the device code never appears in any lookup response.

Frontend-side, assert that the approval page shows the requesting device's name, platform, IP hint, and match code before an approve control is enabled, and that approve and deny call their endpoints and render their outcomes.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/api/handlers ./internal/auth -run Device -count=1` and `cd web && pnpm vitest run src/pages/ActivateDevice.test.tsx`

Expected: FAIL because the purpose is not enforced and the approval page has no test-visible match-code gate.

- [ ] **Step 3: Implement purpose enforcement and surface the page**

Keep every existing device-login behavior. Add the television purpose to the accepted set, enforce it in the approve route the same way the handoff route enforces its own, and advertise pairing in the device capability response without changing the existing fields' types.

In the frontend, show the match code prominently and require the page to have loaded a lookup before enabling approve. Add a navigation entry to the page from the signed-in account area; keep the existing URL working unchanged.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/api/handlers ./internal/auth -run Device -count=1` and `cd web && pnpm run lint && pnpm vitest run src/pages/ActivateDevice.test.tsx`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/auth internal/api/handlers/auth_device.go internal/api/handlers/auth_device_test.go web/src
git commit -m "feat(auth): complete TV pairing approval surface"
```

### Task 6: Conformance and full verification

**Files:**
- Modify: `internal/clientcontract/conformance_test.go`
- Modify: `internal/clientcontract/export_test.go`
- Modify: `docs/architecture/v1-scope.md`
- Modify: `README.md`

**Interfaces:**
- Consumes the contracts repository's conformance CLI against a disposable migrated database, extended to cover server identity, capabilities, both Watch documents, and the corrected progress sync vocabulary.
- Produces no production code path that depends on the contracts repository.

- [ ] **Step 1: Write the failing conformance extension**

Extend the disposable-database run so that after the fixture account is set up and the invented media is seeded, it asserts the new endpoints answer and that each Watch document validates against the contracts schema. Skip with the missing variable named when the admin database URL or the contracts checkout is absent; never pass by default.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/clientcontract -run Conformance -count=1`

Expected: FAIL with the new assertions when a database is available, or SKIP with the missing variable named when it is not — a skip is not evidence.

- [ ] **Step 3: Wire the new endpoints into the seeded run**

Seed enough invented media that a home document has at least one movie, one series with two seasons, and one progress row. Keep every seeded value invented; no real title, account, or origin.

- [ ] **Step 4: Run the full gate**

```bash
make lint
make test
cd web && pnpm run lint && pnpm run format:check
```

Then from the repository root:

```bash
make verify-local-paths
make migrate-status
git diff --check
```

Expected: all commands exit `0`. `make lint` reports pre-existing findings the branch did not introduce; do not add to them.

- [ ] **Step 5: Record evidence and commit**

Note in the PR body which conformance cases ran against a real database and which skipped, the mDNS module and license from Task 4, and the pre-lock correction from Task 3.

```bash
git add internal/clientcontract docs/architecture/v1-scope.md README.md
git commit -m "test(api): verify client wiring endpoints"
```
