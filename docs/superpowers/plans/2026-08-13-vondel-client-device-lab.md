# Vondel Client Device Lab and CI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Orchestrate deterministic cross-platform scenarios across simulators, emulators, Apple TV, Android TV, and Fire TV with reservations, health gates, pairwise configuration coverage, sanitized artifacts, and release-candidate enforcement.

**Architecture:** The harness repository gains a typed inventory, scheduler, platform launch adapters, configuration generator, and CI workflows. Each worker runs one exclusive device lease and one isolated server run; contamination or cleanup failure quarantines both resources.

**Tech Stack:** Go 1.26, GitHub Actions, `xcodebuild`/`simctl`, ADB/Gradle, JUnit/JSON/HTML evidence, mTLS.

**Spec:** `vondel-server/docs/superpowers/specs/2026-08-13-vondel-cross-platform-client-test-harness-design.md`

## Global Constraints

- Begin after harness foundation, test controller, Apple driver, and Android driver are reviewed.
- Inventory contains no passwords, tokens, private keys, or static client credentials.
- A device and mutable test tenant are exclusively leased to one run.
- Unhealthy, contaminated, or incompletely cleaned resources cannot be reused automatically.
- Artifacts pass the harness evidence scanner before upload.
- Use strict red-green TDD and commit after every task.

---

### Task 1: Typed inventory and exclusive leases

**Files:**
- Create: `internal/lab/inventory.go`
- Create: `internal/lab/lease.go`
- Create: `internal/lab/inventory_test.go`
- Create: `lab/inventory.example.yaml`

**Interfaces:**
- Produces device kinds `apple_tv_simulator`, `apple_tv`, `android_tv_emulator`, `android_tv`, and `fire_tv`.
- Produces `LeaseManager.Acquire(requirements, ttl)` and idempotent `Release`; a lease binds device ID, worker ID, server run ID, and expiry.

- [ ] **Step 1: Write failing exclusivity/secret tests**

```go
func TestInventoryRejectsSecretsAndLeaseIsExclusive(t *testing.T) {
    require.Error(t, DecodeInventory([]byte("devices: [{id: tv, password: secret}]")))
    first := requireAcquire(t, manager, requirements)
    require.ErrorIs(t, manager.Acquire(ctx, requirements, time.Minute), ErrUnavailable)
    require.NoError(t, manager.Release(ctx, first))
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/lab -run 'Inventory|Lease' -count=1`

Expected: FAIL because inventory/lease do not exist.

- [ ] **Step 3: Implement closed inventory and atomic leases**

Inventory stores only stable device facts and environment-variable names for external credentials. Use compare-and-swap lease storage, heartbeat, TTL, owner identity, and contamination flag. Never steal a live lease.

- [ ] **Step 4: Verify GREEN with race detection**

Run: `go test -race ./internal/lab -run 'Inventory|Lease' -count=1`

Expected: PASS under concurrent acquisition, expiry, crash recovery, and contamination.

- [ ] **Step 5: Commit**

```bash
git add internal/lab lab/inventory.example.yaml
git commit -m "feat(lab): reserve client test devices"
```

### Task 2: Platform health and launch adapters

**Files:**
- Create: `internal/lab/apple.go`
- Create: `internal/lab/android.go`
- Create: `internal/lab/health.go`
- Create: `internal/lab/adapter_test.go`

**Interfaces:**
- Produces typed adapters for boot/install/launch/driver-connect/screenshot/log/terminate/uninstall/health.
- Shell execution is internal and constructed from validated typed fields; scenarios cannot supply commands or paths.

- [ ] **Step 1: Write failing command-construction and health tests**

```go
func TestAdaptersRejectUnlistedDevicesAndUnsafeArguments(t *testing.T) {
    require.Error(t, apple.Launch(Device{ID:"../../bad"}, build))
    require.Error(t, android.Launch(Device{Serial:"-s;rm"}, apk))
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/lab -run 'Adapter|Health' -count=1`

Expected: FAIL because adapters do not exist.

- [ ] **Step 3: Implement typed adapters**

Use `exec.CommandContext` argument arrays, inventory allowlists, absolute reviewed build paths, boot/foreground/focus/decoder/storage/network health checks, `adb forward/reverse`, and simulator destinations. Capture only sanitized command summaries.

- [ ] **Step 4: Verify GREEN**

Run fake-tool tests and one simulator/emulator smoke run.

Expected: PASS; failed cleanup quarantines the device.

- [ ] **Step 5: Commit**

```bash
git add internal/lab
git commit -m "feat(lab): launch native client drivers"
```

### Task 3: Configuration matrix and coverage registry enforcement

**Files:**
- Create: `internal/matrix/pairwise.go`
- Create: `internal/matrix/risk.go`
- Create: `internal/matrix/pairwise_test.go`
- Create: `coverage/configurations.yaml`
- Modify: `coverage/client-capabilities.yaml`

**Interfaces:**
- Produces pairwise cases for platform/device, layout, theme, language, text size, reduced motion, screen reader, overlays, and fault.
- Produces explicit exhaustive high-risk cases and deterministic matrix IDs.

- [ ] **Step 1: Write failing pair coverage test**

```go
func TestGeneratedMatrixCoversEveryPairAndRequiredRiskCase(t *testing.T) {
    cases := Generate(loadConfig(t))
    requireAllPairs(t, cases)
    requireCase(t, cases, "tv_large_text_screen_reader_critical_overlay")
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/matrix -count=1`

Expected: FAIL because generator does not exist.

- [ ] **Step 3: Implement deterministic IPO-style pairwise generation**

Sort factors/values, apply documented invalid-combination constraints, add explicit risk cases, hash canonical case data for IDs, and fail coverage when a server-advertised variant is unclassified.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/matrix ./internal/coverage -count=1`

Expected: PASS with stable snapshots under input reordering.

- [ ] **Step 5: Commit**

```bash
git add internal/matrix coverage
git commit -m "feat(lab): generate client configuration matrix"
```

### Task 4: Orchestrator, cleanup, and replay

**Files:**
- Create: `internal/lab/orchestrator.go`
- Create: `internal/lab/orchestrator_test.go`
- Create: `cmd/vondel-client-harness/lab.go`

**Interfaces:**
- Produces `lab run`, `lab replay`, and `lab health` commands.
- Execution order is acquire device, create server run, health/conformance, launch driver, execute scenario/crawl, freeze evidence, destroy server run, clean device, release or quarantine.

- [ ] **Step 1: Write failing crash/cleanup ordering tests**

```go
func TestFailureFreezesEvidenceBeforeCleanupAndQuarantinesOnCleanupError(t *testing.T) {
    result := runWithFailure(t, DriverCrash, CleanupFailure)
    require.True(t, result.EvidenceFinalized)
    require.Equal(t, Contaminated, result.DeviceState)
    require.Equal(t, Contaminated, result.ServerState)
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/lab ./cmd/vondel-client-harness -run 'Orchestrator|Lab' -count=1`

Expected: FAIL because orchestrator does not exist.

- [ ] **Step 3: Implement state-machine orchestration**

Persist a sanitized journal after each phase, use compensating cleanup with bounded deadlines, never upload before evidence scan, and generate a replay command containing only stable IDs and environment-variable names.

- [ ] **Step 4: Verify GREEN**

Run race tests with injected controller/driver/device crashes at every phase.

Expected: no leaked lease; uncertain cleanup always quarantines rather than passes.

- [ ] **Step 5: Commit**

```bash
git add internal/lab cmd/vondel-client-harness
git commit -m "feat(lab): orchestrate deterministic client runs"
```

### Task 5: CI tiers and release-candidate gates

**Files:**
- Create: `.github/workflows/client-pr.yml`
- Create: `.github/workflows/client-main.yml`
- Create: `.github/workflows/client-device-lab.yml`
- Create: `.github/workflows/client-release-candidate.yml`
- Create: `docs/device-lab-operations.md`
- Modify: `README.md`
- Modify: `ORIGINALITY.md`

**Interfaces:**
- PR: one Apple TV simulator and Android TV emulator focused journey.
- Main: full authored suite plus pairwise layouts/accessibility/faults and focus crawl.
- Scheduled: physical Apple TV, Android TV, and Fire TV.
- Release candidate: clean server, high-risk matrix, physical playback, release-absence gates, clean-room audits, and coverage registry.

- [ ] **Step 1: Write failing workflow policy tests**

Parse workflows and require timeouts, concurrency groups, least-privilege permissions, artifact scanning, cleanup under `always()`, no inline secrets, and no unpinned third-party actions.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/lab -run Workflow -count=1`

Expected: FAIL because workflows do not exist.

- [ ] **Step 3: Implement workflows and operations runbook**

Use environment-protected device credentials, per-run evidence directories, explicit reservations, cancellation cleanup, artifact retention, quarantine notifications, and no automatic retry of product assertions.

- [ ] **Step 4: Run final cross-platform acceptance**

Prove one unchanged scenario on Apple/Android, deliberate trap detection on both, exact synthetic playback/rendition/timeline, exact-scope resume, one-command replay, contamination quarantine, release absence, coverage completeness, full tests/race/vet, diff check, and authorized sanitized originality audits.

- [ ] **Step 5: Record evidence and commit**

```bash
git add .github docs README.md ORIGINALITY.md
git commit -m "test(lab): gate cross-platform client releases"
```
