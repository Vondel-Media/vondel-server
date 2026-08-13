# Vondel Client Test Harness Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create the independent host-side harness that executes shared semantic scenarios, explores TV focus graphs, speaks a versioned native-driver protocol, and emits sanitized replayable evidence.

**Architecture:** A new Go repository owns scenario and driver schemas, orchestration, focus exploration, redaction, reports, and a CLI. Apple and Android implement the same NDJSON command/observation protocol in later plans; this plan proves the engine against an in-memory and process-backed fake driver.

**Tech Stack:** Go 1.26, JSON Schema 2020-12, NDJSON over Unix/TCP transports, `gopkg.in/yaml.v3` 3.0.1, Go standard library.

**Spec:** `vondel-server/docs/superpowers/specs/2026-08-13-vondel-cross-platform-client-test-harness-design.md`

## Global Constraints

- Create a separate `vondel-client-test-harness` repository with an empty, zero-parent Vondel history and no reference-client access.
- Semantic identifiers and closed remote actions are the only navigation vocabulary; coordinate and arbitrary-key input are absent.
- Credentials remain memory-only and are redacted before evidence serialization.
- Driver observations are allowlisted and never contain tokens, headers, complete media URLs, titles, account names, or profile names.
- Scenarios cannot request database, shell, arbitrary filesystem, or arbitrary network access.
- Use strict red-green TDD and commit after each task.
- Live TV, IPTV, DVR, EPG, `.strm`, and arbitrary remote streaming are prohibited.

---

### Task 1: Versioned scenario and native-driver contracts

**Files:**
- Create: `go.mod`
- Create: `README.md`
- Create: `ORIGINALITY.md`
- Create: `schema/scenario/v1.json`
- Create: `schema/driver/command-v1.json`
- Create: `schema/driver/observation-v1.json`
- Create: `fixtures/scenarios/resume_movie.yaml`
- Create: `fixtures/driver/press-right.ndjson`
- Create: `internal/protocol/types.go`
- Create: `internal/protocol/validation_test.go`

**Interfaces:**
- Produces `protocol.DriverCommandV1`, `DriverObservationV1`, `PlaybackObservationV1`, and `ScenarioV1`.
- Produces closed actions `up`, `down`, `left`, `right`, `select`, `back`, `play_pause`, `relaunch`, `background`, `foreground`, `observe`, and `screenshot`.

- [ ] **Step 1: Write failing closed-schema tests**

```go
func TestDriverProtocolRejectsCoordinatesAndUnknownActions(t *testing.T) {
    assertInvalid(t, `{"schema":"vondel_driver_command_v1","request_id":"r1","nonce":"01234567890123456789012345678901","action":{"type":"tap","x":10,"y":20}}`)
    assertInvalid(t, `{"schema":"vondel_driver_command_v1","request_id":"r1","nonce":"01234567890123456789012345678901","action":{"type":"arbitrary_key","code":23}}`)
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/protocol -count=1`

Expected: FAIL because protocol files do not exist.

- [ ] **Step 3: Implement strict types and schemas**

```go
type DriverCommandV1 struct {
    Schema string `json:"schema"`
    RequestID string `json:"request_id"`
    Nonce string `json:"nonce"`
    Action RemoteAction `json:"action"`
}
type DriverObservationV1 struct {
    Schema, RequestID, Route string
    Sequence, MonotonicNS uint64
    FocusExpected bool
    FocusedID *string
    VisibleIDs []string
    ModalState *string
    Playback *PlaybackObservationV1
    Presentation PresentationV1
}
```

Require nonblank IDs, nonce length at least 32 bytes, unique sorted visible IDs, finite numeric fields, and `additionalProperties: false` for privileged objects. Permit only complete-scalar variable substitution in scenario assertion values.

- [ ] **Step 4: Verify GREEN and provenance**

Run: `go test ./internal/protocol -count=1 && git diff --check`

Expected: PASS. Record contract authority, dependencies/licenses, original decisions, implementer reference access `none`, and pending authorized audit in `ORIGINALITY.md`.

- [ ] **Step 5: Commit**

```bash
git add go.mod README.md ORIGINALITY.md schema fixtures internal/protocol
git commit -m "feat(protocol): define client harness contracts"
```

### Task 2: Deterministic semantic scenario executor

**Files:**
- Create: `internal/scenario/decode.go`
- Create: `internal/scenario/validate.go`
- Create: `internal/scenario/executor.go`
- Create: `internal/scenario/executor_test.go`
- Create: `internal/driver/driver.go`
- Create: `internal/driver/fake.go`

**Interfaces:**
- Produces `driver.Driver.Execute(context.Context, protocol.DriverCommandV1) (protocol.DriverObservationV1, error)` and `Close() error`.
- Produces `scenario.Executor.Run(context.Context, protocol.ScenarioV1, RunContext) (RunResult, error)`.

- [ ] **Step 1: Write failing execution and unsafe-variable tests**

```go
func TestExecutorUsesSemanticIDs(t *testing.T) {
    fake := driver.NewFake(healthyTransitions())
    result, err := scenario.NewExecutor().Run(context.Background(), resumeMovie(t), runContext(fake))
    if err != nil || !result.Passed || fake.SawCoordinateAction() { t.Fatalf("run = %#v, %v", result, err) }
}
func TestVariablesCannotPopulateActionTypes(t *testing.T) {
    _, err := scenario.Decode([]byte("schema: vondel_client_scenario_v1\nscenario: bad\nsteps:\n- press: ${SECRET}\n"))
    if !errors.Is(err, scenario.ErrUnsafeVariablePosition) { t.Fatalf("error = %v", err) }
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/scenario ./internal/driver -count=1`

Expected: FAIL because decoder and executor do not exist.

- [ ] **Step 3: Implement bounded decoding and execution**

```go
type RunContext struct {
    RunID string
    Nonce string
    Variables map[string]string
    TransitionTimeout time.Duration
    Driver driver.Driver
}
```

Enforce known YAML fields, reject aliases/duplicate keys, cap input at 1 MiB and 10,000 nodes, derive request IDs as `<run-id>:<step-index>`, and validate request correlation, routes, focus, visibility, playback facts, numeric tolerances, and presentation identifiers.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/scenario ./internal/driver -count=1`

Expected: PASS including cancellation, timeout, malformed input, and mismatched-observation cases.

- [ ] **Step 5: Commit**

```bash
git add internal/scenario internal/driver
git commit -m "feat(engine): execute semantic client scenarios"
```

### Task 3: Bounded focus explorer

**Files:**
- Create: `internal/focus/graph.go`
- Create: `internal/focus/explorer.go`
- Create: `internal/focus/invariants.go`
- Create: `internal/focus/explorer_test.go`
- Create: `fixtures/focus/trap.json`
- Create: `fixtures/focus/healthy.json`

**Interfaces:**
- Produces `Explorer.Explore(context.Context, driver.Driver, EntryFlow, Limits) (Graph, []Violation, error)`.
- `EntryFlow.Replay` deterministically restores root and replays a prefix before each probe.
- Produces shortest paths for missing/multiple focus, unreachable required controls, stalls, traps, hidden focus, nondeterministic edges, and missing Back paths.

- [ ] **Step 1: Write failing deliberate-trap test**

```go
func TestExplorerFindsShortestTrapWithoutCoordinates(t *testing.T) {
    d := newGraphDriver(loadGraph(t, "fixtures/focus/trap.json"))
    graph, violations, err := focus.NewExplorer().Explore(context.Background(), d, d, focus.Limits{MaxDepth: 8, MaxActions: 200, MaxStates: 100})
    if err != nil { t.Fatal(err) }
    trap := requireViolation(t, violations, focus.Trap)
    if diff := cmp.Diff([]string{"down", "right"}, actionNames(trap.Replay)); diff != "" { t.Fatal(diff) }
    if graph.ContainsCoordinates() { t.Fatal("coordinate found") }
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/focus -count=1`

Expected: FAIL because focus exploration does not exist.

- [ ] **Step 3: Implement breadth-first replay exploration**

```go
type StateKey struct { Route, FocusedID, ModalState, PlaybackState, LayoutVariant string }
type Limits struct { MaxDepth, MaxActions, MaxStates int; TransitionTimeout time.Duration }
```

Restore and replay before each edge probe, compare duplicate probes for determinism, store predecessor/action data, ignore `vondel.harness.*`, and allow stable boundaries only when declared.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/focus -count=1`

Expected: PASS for healthy graph and exact classifications for every seeded violation plus limit/timeout tests.

- [ ] **Step 5: Commit**

```bash
git add internal/focus fixtures/focus
git commit -m "feat(focus): crawl semantic TV navigation"
```

### Task 4: Sanitized evidence bundles

**Files:**
- Create: `internal/evidence/redact.go`
- Create: `internal/evidence/bundle.go`
- Create: `internal/evidence/junit.go`
- Create: `internal/evidence/html.go`
- Create: `internal/evidence/evidence_test.go`
- Create: `fixtures/evidence/secret-input.json`

**Interfaces:**
- Produces `Writer.Write(RunResult, EvidenceInput) (Manifest, error)` and `Scan(path) error`.
- Emits manifest, replay, focus graph, JUnit, and HTML; accepts screenshot bytes under generated names only.

- [ ] **Step 1: Write failing leak and traversal tests**

```go
func TestEvidenceNeverSerializesSecretsOrCompleteURLs(t *testing.T) {
    dir := t.TempDir()
    _, err := evidence.NewWriter(dir).Write(secretBearingResult(t), secretBearingInput(t))
    if err != nil { t.Fatal(err) }
    if err := evidence.Scan(dir); err != nil { t.Fatal(err) }
    body := readAllEvidence(t, dir)
    for _, s := range []string{"Bearer ", "refresh_token", "https://media.invalid/path?token="} {
        if bytes.Contains(body, []byte(s)) { t.Fatalf("leaked %q", s) }
    }
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/evidence -count=1`

Expected: FAIL because writer/scanner do not exist.

- [ ] **Step 3: Implement allowlist-first evidence**

Use sequential safe names, `0600` files, `0700` directories, symlink/traversal rejection, 100 MiB run cap, no arbitrary metadata maps, parsed URL rejection, closed failure categories, and atomic manifest publication only after scanning all artifacts.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/evidence -count=1`

Expected: PASS with parsable formats and canary scanner failures.

- [ ] **Step 5: Commit**

```bash
git add internal/evidence fixtures/evidence
git commit -m "feat(evidence): emit sanitized replay reports"
```

### Task 5: Authenticated native-driver transport

**Files:**
- Create: `internal/driver/ndjson.go`
- Create: `internal/driver/unix.go`
- Create: `internal/driver/tcp.go`
- Create: `internal/driver/ndjson_test.go`
- Create: `cmd/fake-native-driver/main.go`
- Create: `cmd/fake-native-driver/main_test.go`

**Interfaces:**
- Produces `DialUnix(path, nonce, timeout)` and `DialTCP(address, nonce, timeout)`.
- Enforces one-client nonce handshake, 1 MiB frames, one-request/one-observation correlation, and monotonic sequence numbers.

- [ ] **Step 1: Write failing nonce/replay tests**

```go
func TestNDJSONRejectsWrongNonceReplayAndMismatchedRequest(t *testing.T) {
    s := startFakeNativeDriver(t)
    require.ErrorIs(t, dialAndObserve(s, "wrong"), driver.ErrUnauthorized)
    c := requireDial(t, s, s.Nonce)
    require.NoError(t, execute(c, "run:1"))
    require.ErrorIs(t, execute(c, "run:1"), driver.ErrReplay)
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/driver ./cmd/fake-native-driver -count=1`

Expected: FAIL because transport does not exist.

- [ ] **Step 3: Implement bounded authenticated NDJSON**

Require owned Unix socket paths or loopback TCP, constant-time nonce comparison, per-transition deadlines, strict request correlation, sequence monotonicity, close-on-cancel, and clean EOF. Reject non-loopback TCP before dialing.

- [ ] **Step 4: Verify GREEN with race detection**

Run: `go test -race ./internal/driver ./cmd/fake-native-driver -count=1`

Expected: PASS with no goroutine leaks on timeout, cancellation, malformed frame, or crash.

- [ ] **Step 5: Commit**

```bash
git add internal/driver cmd/fake-native-driver
git commit -m "feat(driver): connect native harness agents"
```

### Task 6: CLI, coverage registry, and acceptance

**Files:**
- Create: `cmd/vondel-client-harness/main.go`
- Create: `cmd/vondel-client-harness/main_test.go`
- Create: `internal/coverage/registry.go`
- Create: `internal/coverage/registry_test.go`
- Create: `coverage/client-capabilities.yaml`
- Create: `.github/workflows/ci.yml`
- Modify: `README.md`
- Modify: `ORIGINALITY.md`

**Interfaces:**
- Produces `run`, `crawl`, and `coverage` subcommands.
- Nonces enter only through named environment variables or inherited descriptors, never command-line values.
- Coverage classifications are `authored_scenario`, `focus_invariant`, `contract_test`, `manual`, and `unsupported`.

- [ ] **Step 1: Write failing CLI and unclassified-capability tests**

```go
func TestCoverageRejectsUnclassifiedCapability(t *testing.T) {
    r := loadRegistry(t, "capabilities:\n- id: watch.resume\n  owner: clients\n")
    if !errors.Is(r.Validate(), coverage.ErrUnclassified) { t.Fatal("accepted") }
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./cmd/vondel-client-harness ./internal/coverage -count=1`

Expected: FAIL because CLI and registry do not exist.

- [ ] **Step 3: Implement commands and closed classifications**

`manual` and `unsupported` require owner/rationale. Quarantine requires owner/reason/ISO expiry. Exit `0` on pass, `1` on product assertion failure, and `2` on harness/configuration failure. Refuse secret-looking flags.

- [ ] **Step 4: Run complete acceptance**

```bash
go test -race ./... -count=1
go vet ./...
go run ./cmd/vondel-client-harness coverage --registry coverage/client-capabilities.yaml
git diff --check
```

Run the fake native-driver process, then execute `resume_movie.yaml` and the deliberate trap. Expected: healthy run exits `0`; trap exits `1` with replay `down,right`; evidence scanner passes; process arguments contain no nonce.

- [ ] **Step 5: Record evidence and commit**

```bash
git add cmd/vondel-client-harness internal/coverage coverage .github README.md ORIGINALITY.md
git commit -m "test(harness): verify semantic client foundation"
```
