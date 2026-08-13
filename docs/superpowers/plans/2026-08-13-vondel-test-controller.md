# Vondel Non-Production Test Controller Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a separately compiled, non-production server component that creates isolated deterministic client-test runs with normal application credentials, named seeds/faults, sanitized observations, and verified cleanup.

**Architecture:** `cmd/vondel-test-controller` owns a restricted control API and one disposable migrated database plus ordinary loopback Vondel runtime per run. The production `cmd/silo` import graph, router, image, and binary remain free of controller routes, seeds, credentials, and markers.

**Tech Stack:** Go 1.26, Chi, PostgreSQL 16 with pgvector, Goose migrations, mTLS, existing Vondel auth/tenancy/catalog/settings/playback packages.

**Spec:** `vondel-server/docs/superpowers/specs/2026-08-13-vondel-cross-platform-client-test-harness-design.md`

## Global Constraints

- Start in a fresh worktree from a reviewed clean server base; do not use the dirty OPA worktree.
- The controller is absent from production builds, including for administrators; do not add a server test mode or production router mount.
- Every run gets an isolated disposable database and exclusive lease; reset means destroy then recreate.
- The API accepts only named seeds, named bounded faults, and named presentation variants—never SQL, shell, paths, URLs, tenant IDs, or arbitrary JSON mutation.
- Client credentials use normal public authentication/authorization paths and are short-lived; controller identity is separate.
- Controller and observations never persist passwords, tokens, headers, complete URLs, titles, account names, or profile names.
- Use strict red-green TDD and commit after every task.
- Do not implement Live TV, IPTV, DVR, EPG, `.strm`, or arbitrary remote streaming.

---

### Task 1: Production-exclusion skeleton and control protocol

**Files:**
- Create: `cmd/vondel-test-controller/main.go`
- Create: `internal/testcontroller/protocol.go`
- Create: `internal/testcontroller/router.go`
- Create: `internal/testcontroller/exclusion_test.go`
- Create: `internal/testcontroller/testdata/control-v1.json`

**Interfaces:**
- Reserves `/control/v1` only in `testcontroller.NewRouter(Dependencies)`.
- Produces run states `creating`, `ready`, `destroying`, `destroyed`, `expired`, and `contaminated`.
- Produces create/status/destroy schemas with `additionalProperties: false` and no arbitrary maps.

- [ ] **Step 1: Write failing exclusion tests**

```go
func TestProductionServerExcludesTestController(t *testing.T) {
    assertDependencyAbsent(t, "./cmd/silo", "internal/testcontroller")
    assertProductionRouteStatus(t, "/control/v1/health", http.StatusNotFound)
    assertBinaryStringsAbsent(t, build(t, "./cmd/silo"), []string{"vondel_test_controller_v1", "/control/v1/runs"})
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/testcontroller -run Excludes -count=1`

Expected: FAIL because controller package does not exist.

- [ ] **Step 3: Implement the separate command and empty authenticated router**

Define `CreateRunRequestV1{Seed, TTLSeconds}`, `RunStatusV1`, and `DestroyRunResponseV1`. Do not import controller packages from `cmd/silo`, `internal/api`, or production config.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/testcontroller -run Excludes -count=1 && go list -deps ./cmd/silo | ! rg 'internal/(testcontroller|testseed)'`

Expected: PASS; production Dockerfiles remain unchanged.

- [ ] **Step 5: Commit**

```bash
git add cmd/vondel-test-controller internal/testcontroller
git commit -m "feat(test-controller): isolate non-production control plane"
```

### Task 2: Fail-closed startup identity and mTLS

**Files:**
- Create: `internal/testcontroller/config.go`
- Create: `internal/testcontroller/listener.go`
- Create: `internal/testcontroller/config_test.go`
- Modify: `cmd/vondel-test-controller/main.go`

**Interfaces:**
- Produces `Config.Validate() error` requiring private/loopback bind, explicit sentinel, maintenance DSN, approved database suffix/prefix, CA, server certificate/key, client SPIFFE/DNS allowlist, and hard limits.

- [ ] **Step 1: Write failing hostile-configuration tests**

```go
func TestConfigRejectsProductionAndPublicTargets(t *testing.T) {
    for _, mutate := range []func(*Config){publicBind, productionDBName, missingSentinel, missingClientCA, unlimitedTTL} {
        cfg := validConfig(t); mutate(&cfg)
        if cfg.Validate() == nil { t.Fatal("unsafe config accepted") }
    }
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/testcontroller -run 'Config|TLS' -count=1`

Expected: FAIL because validation/listener do not exist.

- [ ] **Step 3: Implement startup refusal**

Require sentinel value `VONDEL_NONPRODUCTION_TEST_CONTROLLER=confirmed`, default maximum four runs, 30-minute TTL, 1 MiB body, 60-second request deadline, private binding, TLS 1.3, verified client certificate identity, and database names matching a configured safe test prefix. Secrets enter through environment/file descriptors and are never logged.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/testcontroller -run 'Config|TLS' -count=1`

Expected: PASS including wrong/untrusted/expired client certificates.

- [ ] **Step 5: Commit**

```bash
git add internal/testcontroller cmd/vondel-test-controller
git commit -m "feat(test-controller): enforce test-only startup identity"
```

### Task 3: Disposable run database and contamination authority

**Files:**
- Create: `internal/testrun/database.go`
- Create: `internal/testrun/manager.go`
- Create: `internal/testrun/lease.go`
- Create: `internal/testrun/manager_test.go`

**Interfaces:**
- Produces `Manager.Create(ctx, CreateOptions)`, `Status`, `Destroy`, `Reset`, and `SweepExpired`.
- Run database names match `vondel_client_harness_<32 lowercase hex>`; destroy terminates sessions, drops exactly that database, and verifies absence.

- [ ] **Step 1: Write failing lifecycle/recovery tests**

```go
func TestRunOwnsDisposableDatabaseAndFailedDropContaminates(t *testing.T) {
    run := requireCreate(t, manager)
    requireDatabaseExists(t, run.DatabaseName)
    manager.FailNextDropForTest()
    require.Error(t, manager.Destroy(ctx, run.ID, run.Lease))
    require.Equal(t, Contaminated, requireStatus(t, run.ID).State)
    require.ErrorIs(t, manager.Reset(ctx, run.ID, run.Lease), ErrContaminated)
}
```

- [ ] **Step 2: Verify RED**

Run: `SILO_TEST_DATABASE_URL="$SILO_TEST_DATABASE_URL" go test ./internal/testrun -count=1`

Expected: FAIL because manager does not exist.

- [ ] **Step 3: Extract hardened database lifecycle**

Reuse the proven clientcontract disposable database algorithm: cryptographic name, validated identifier, create, migrate with `migrations.FS`, exclusive lease, TTL context, idempotent destroy, session termination, drop, absence check, and contamination on uncertainty. Never perform broad table deletion.

- [ ] **Step 4: Verify GREEN and races**

Run: `SILO_REQUIRE_TEST_DATABASE=1 go test -race ./internal/testrun -count=1`

Expected: PASS for concurrent create limits, wrong lease, expiry, repeated destroy, crash recovery, and failed-drop quarantine.

- [ ] **Step 5: Commit**

```bash
git add internal/testrun
git commit -m "feat(test-controller): manage isolated client runs"
```

### Task 4: Named seeds and normal application runtime

**Files:**
- Create: `internal/testseed/registry.go`
- Create: `internal/testseed/watch_standard.go`
- Create: `internal/testseed/watch_standard_test.go`
- Create: `internal/testrun/runtime.go`
- Modify: `internal/testrun/manager.go`

**Interfaces:**
- Produces seed `watch_standard_v1` with invented account/profile/organization, Watch catalog/media, progress, presentation variant, and expected semantic state.
- Produces a loopback application origin plus one-time bootstrap credential envelope after the run reaches `ready`.

- [ ] **Step 1: Write failing seed and ordinary-auth tests**

```go
func TestWatchSeedUsesNormalAuthAndExactTenantScope(t *testing.T) {
    run := requireCreateWithSeed(t, "watch_standard_v1")
    tokens := loginThroughPublicAPI(t, run.AppOrigin, run.Bootstrap)
    requireWatchCatalog(t, run.AppOrigin, tokens, "4242", "8080")
    requireUnauthorizedInOtherRun(t, tokens)
}
```

- [ ] **Step 2: Verify RED**

Run: `SILO_REQUIRE_TEST_DATABASE=1 go test ./internal/testseed ./internal/testrun -run WatchSeed -count=1`

Expected: FAIL because seed/runtime do not exist.

- [ ] **Step 3: Implement compile-time named seed registry**

Use normal account provisioner, tenancy membership, settings normalization, catalog import, auth service, and `api.NewRouter`. Callers select a closed seed name only. Generated media paths and importer rewrites are internal constants owned by the seed package. Bootstrap secrets are returned once and zeroed after retrieval.

- [ ] **Step 4: Verify GREEN**

Run: `SILO_REQUIRE_TEST_DATABASE=1 go test ./internal/testseed ./internal/testrun -run 'WatchSeed|Runtime' -count=1`

Expected: PASS; the ordinary Vondel app route has no controller endpoints.

- [ ] **Step 5: Commit**

```bash
git add internal/testseed internal/testrun
git commit -m "feat(test-controller): seed deterministic Watch runs"
```

### Task 5: Typed sanitized observations

**Files:**
- Create: `internal/testobserve/event.go`
- Create: `internal/testobserve/sink.go`
- Create: `internal/testobserve/redact.go`
- Create: `internal/testobserve/sink_test.go`
- Modify: `internal/testcontroller/router.go`
- Modify: narrow playback/progress/settings/authorization constructor seams selected during implementation

**Interfaces:**
- Produces cursor-based `GET /control/v1/runs/{id}/observations?after=N` and bounded wait.
- Produces allowlisted event kinds for request, authorization, playback plan, playback lifecycle, progress, and presentation.

- [ ] **Step 1: Write failing correlation/redaction tests**

```go
func TestObservationSinkRejectsSecretsAndCorrelatesAction(t *testing.T) {
    sink := NewSink(100)
    sink.Record(Event{RunID: "r", ActionID: "r:7", Kind: PlaybackPlan, ContentID: "4242"})
    require.Equal(t, uint64(1), sink.After(0)[0].Cursor)
    require.NoSecretMaterial(t, mustJSON(sink.After(0)))
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/testobserve ./internal/testcontroller -run Observation -count=1`

Expected: FAIL because sink/routes do not exist.

- [ ] **Step 3: Implement synchronous typed observer seams**

Derive action IDs from validated `X-Request-ID`; bound each run to 10,000 events; use monotonic cursors; wait with context deadline; pseudonymize user/profile IDs; reject arbitrary maps and URL/error strings. Keep the activity log as secondary evidence only.

- [ ] **Step 4: Verify GREEN**

Run: `go test -race ./internal/testobserve ./internal/testcontroller -run Observation -count=1`

Expected: PASS under concurrent recording, cursor expiry, wait cancellation, and secret canaries.

- [ ] **Step 5: Commit**

```bash
git add internal/testobserve internal/testcontroller internal
git commit -m "feat(test-controller): expose sanitized run observations"
```

### Task 6: Named presentation variants and bounded faults

**Files:**
- Create: `internal/testseed/presentation.go`
- Create: `internal/testfault/registry.go`
- Create: `internal/testfault/middleware.go`
- Create: `internal/testfault/registry_test.go`
- Modify: `internal/testcontroller/router.go`

**Interfaces:**
- Produces named presentation variants validated through the settings manifest.
- Produces named faults `expired_playback_plan`, `bounded_response_delay`, `revoke_membership`, and `rendition_unavailable`, each with closed capped parameters.

- [ ] **Step 1: Write failing arbitrary-input and fault-lifetime tests**

```go
func TestFaultRegistryRejectsArbitraryTargetsAndExpires(t *testing.T) {
    require.Error(t, registry.Activate(run, Fault{Name: "proxy", Parameters: map[string]any{"url":"http://example"}}))
    lease := requireActivate(t, registry, BoundedResponseDelay{Millis: 250, Requests: 1})
    requireDelayOnce(t, lease)
    requireNoDelay(t, lease)
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/testfault ./internal/testseed -run 'Fault|Presentation' -count=1`

Expected: FAIL because registries do not exist.

- [ ] **Step 3: Implement closed registries**

Normalize named TV presentation values with the existing settings contract. Cap delay at 5 seconds/20 requests, require fault expiry, bind every fault to one run, and expose activate/deactivate through typed controller endpoints. Do not implement a general proxy.

- [ ] **Step 4: Verify GREEN**

Run: `go test -race ./internal/testfault ./internal/testseed -run 'Fault|Presentation' -count=1`

Expected: PASS with isolation between concurrent runs and automatic expiry.

- [ ] **Step 5: Commit**

```bash
git add internal/testfault internal/testseed internal/testcontroller
git commit -m "feat(test-controller): vary bounded client scenarios"
```

### Task 7: Lifecycle API, audit, and conformance gate

**Files:**
- Create: `internal/testcontroller/api_test.go`
- Create: `internal/testcontroller/audit.go`
- Create: `internal/testcontroller/conformance.go`
- Modify: `internal/testcontroller/router.go`
- Modify: `cmd/vondel-test-controller/main.go`

**Interfaces:**
- Completes create, status, expected-state, observations/wait, presentation, fault, reset, destroy, health, lease, resource, and contamination endpoints.
- Executes the reviewed `vondel-client-conformance` CLI as a typed preflight and parses its JSON report.

- [ ] **Step 1: Write failing end-to-end lifecycle tests**

```go
func TestControllerLifecycleIsAuditedAndCleanupVerified(t *testing.T) {
    c := newMTLSClient(t, controller)
    run := c.Create(t, "watch_standard_v1")
    c.RequireConformancePassed(t, run.ID)
    c.Destroy(t, run.ID, run.Lease)
    c.RequireState(t, run.ID, Destroyed)
    requireDatabaseAbsent(t, run.DatabaseName)
    requireSanitizedAuditSequence(t, run.ID, "create", "ready", "destroy", "destroyed")
}
```

- [ ] **Step 2: Verify RED**

Run: `SILO_REQUIRE_TEST_DATABASE=1 go test ./internal/testcontroller -run Lifecycle -count=1`

Expected: FAIL because lifecycle is incomplete.

- [ ] **Step 3: Implement complete bounded API and append-only audit**

Apply request/body/time/resource limits to every endpoint, require run lease for mutations, redact responses before write, make destroy idempotent, and refuse a client run until conformance passes.

- [ ] **Step 4: Verify GREEN and recovery**

Run: `SILO_REQUIRE_TEST_DATABASE=1 go test -race ./internal/testcontroller ./internal/testrun ./internal/testseed ./internal/testobserve ./internal/testfault -count=1`

Expected: PASS including controller crash, runtime crash, expiry cleanup, failed database drop, wrong lease, and audit redaction.

- [ ] **Step 5: Commit**

```bash
git add internal/testcontroller cmd/vondel-test-controller
git commit -m "feat(test-controller): complete isolated run lifecycle"
```

### Task 8: Security acceptance and production absence

**Files:**
- Create: `scripts/verify-test-controller-absent.sh`
- Create: `scripts/verify-test-controller-absent_test.sh`
- Create: `docs/originality/client-test-controller.md`
- Modify: `.github/workflows/ci.yml`
- Modify: `README.md`
- Modify: `ORIGINALITY.md`

**Interfaces:**
- Produces a self-testing scanner for production binary/import graph/image context.
- Produces exact controller build/run documentation without local paths or credentials.

- [ ] **Step 1: Write failing scanner self-test**

The test injects each forbidden marker into a temporary fake binary and requires the scanner to fail, then scans a clean production `cmd/silo` binary.

- [ ] **Step 2: Verify RED**

Run: `./scripts/verify-test-controller-absent_test.sh`

Expected: FAIL because scanner does not exist.

- [ ] **Step 3: Implement production absence scanner and CI job**

Scan imports, `strings`, symbols, route markers, credential types, seed names, and Docker build products. Scan the separately built controller positively so a broken scanner cannot pass by matching nothing.

- [ ] **Step 4: Run full acceptance**

```bash
go test -race ./... -count=1
go vet ./...
go build -o "$VONDEL_BUILD_DIR/vondel" ./cmd/silo
go build -o "$VONDEL_BUILD_DIR/vondel-test-controller" ./cmd/vondel-test-controller
./scripts/verify-test-controller-absent.sh "$VONDEL_BUILD_DIR/vondel" "$VONDEL_BUILD_DIR/vondel-test-controller"
git diff --check
```

Expected: all pass; authorized sanitized originality audit reports zero unexplained findings before acceptance.

- [ ] **Step 5: Record evidence and commit**

```bash
git add scripts docs/originality .github README.md ORIGINALITY.md
git commit -m "test(test-controller): prove non-production isolation"
```
