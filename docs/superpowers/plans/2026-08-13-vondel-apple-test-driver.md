# Vondel Apple TV Native Test Driver Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the completed Apple TV client controllable by the shared semantic harness through real remote events, stable accessibility IDs, sanitized playback observations, and a harness-only app/driver absent from release.

**Architecture:** Production UI keeps stable semantic accessibility IDs and typed playback state. A separate `VondelTVHarness` target links `VondelTestSupport`; XCUITest owns `XCUIRemote`, focus discovery, screenshots, and the authenticated NDJSON bridge.

**Tech Stack:** Swift 6, SwiftUI, AVFoundation/AVKit, XCTest/XCUITest, XcodeGen 2.45.4, tvOS 17+.

**Spec:** `vondel-server/docs/superpowers/specs/2026-08-13-vondel-cross-platform-client-test-harness-design.md`

## Global Constraints

- Begin after Apple Watch Tasks 1–6 and the harness protocol are reviewed.
- Stable semantic IDs may ship; privileged diagnostics, launch switches, credentials, and transport must not ship in `VondelTV`.
- Navigation uses `XCUIRemote` and `XCUIElement.hasFocus`, never coordinates or visible labels.
- Diagnostics are allowlist-first and cannot represent secrets, headers, URLs, titles, account/profile identifiers, or raw errors.
- Cleartext remote HTTP remains prohibited; use a harness-managed loopback proxy for development servers.
- Use strict red-green TDD and commit after every task.

---

### Task 1: Semantic IDs and sanitized playback observations

**Files:**
- Create: `Sources/VondelNavigation/Testability/SemanticID.swift`
- Modify: `Sources/VondelNavigation/Watch/*View.swift`
- Modify: `Sources/VondelNavigation/VondelRootView.swift`
- Modify: `Sources/VondelCore/Watch/PlaybackPlan.swift`
- Create: `Sources/VondelPlayback/PlaybackTechnicalSnapshot.swift`
- Create: `Sources/VondelPlayback/PlaybackObservationSink.swift`
- Modify: `Sources/VondelPlayback/AVPlayerPlaybackSession.swift`
- Create: `Tests/VondelNavigationTests/SemanticIDTests.swift`
- Modify: `Tests/VondelPlaybackTests/AVPlayerFixtureTests.swift`
- Modify: `Tests/VondelCoreTests/Identity/DiagnosticRedactorTests.swift`

**Interfaces:**
- Produces stable IDs for route roots, destinations, cards, details, episodes, controls, modals, errors, and overlays.
- Produces sanitized plan/content/file/delivery identity plus actual codec, size, frame rate, dynamic range, bitrate estimate, audio/subtitle selection, timeline, buffering, dropped frames, and closed failure category.

- [ ] **Step 1: Write failing ID and telemetry tests**

```swift
func testIDsAreUniqueStableAndTitleFree() throws {
    let ids = SemanticID.watchFixture(contentIDs: ["4242", "8080"], episodeIDs: ["8080-s1-e1"])
    XCTAssertEqual(Set(ids).count, ids.count)
    XCTAssertTrue(ids.contains("watch.movie.4242"))
    XCTAssertFalse(ids.joined().contains("Invented Crossing"))
}
func testTechnicalSnapshotCannotContainURLsOrHeaders() async throws {
    let snapshot = try await playFixtureAndObserve()
    XCTAssertEqual(snapshot.videoCodec, .h264)
    XCTAssertGreaterThanOrEqual(snapshot.presentationHeight, 720)
    XCTAssertFalse(String(reflecting: snapshot).contains("http"))
}
```

- [ ] **Step 2: Verify RED**

Run: `swift test --filter 'SemanticIDTests|AVPlayerFixtureTests|DiagnosticRedactorTests'`

Expected: build/test failure because IDs and technical snapshot do not exist.

- [ ] **Step 3: Implement identifiers and AVPlayer observation**

Assign `.accessibilityIdentifier` separately from human labels. Observe presentation size, asset formats, media selections, access-log numeric facts, time control, position/duration, and categorized errors. Observation types have no URL/header/title/account/profile/raw-error fields and use a no-op sink by default.

- [ ] **Step 4: Verify GREEN**

Run with the disposable fixture origin: `swift test --filter 'SemanticIDTests|AVPlayerFixtureTests|DiagnosticRedactorTests' -Xswiftc -warnings-as-errors`

Expected: PASS; timeline advances and every rendered interactive Watch element has exactly one unique semantic ID.

- [ ] **Step 5: Commit**

```bash
git add Sources Tests
git commit -m "feat(testability): expose safe Apple TV semantics"
```

### Task 2: Separate harness product and diagnostics registry

**Files:**
- Modify: `Package.swift`
- Modify: `project.yml`
- Create: `Sources/VondelTestSupport/DiagnosticsSnapshot.swift`
- Create: `Sources/VondelTestSupport/DiagnosticsRegistry.swift`
- Create: `Apps/tvOSTestHarness/VondelTVHarnessApp.swift`
- Create: `Tests/VondelTestSupportTests/DiagnosticsRegistryTests.swift`

**Interfaces:**
- Produces `VondelTestSupport`, linked only by `VondelTVHarness` and harness UI tests.
- Publishes versioned monotonic snapshots through a nonfocusable node `vondel.harness.diagnostics.snapshot` that accepts no commands.

- [ ] **Step 1: Write failing allowlist/source-membership tests**

```swift
func testSnapshotJSONContainsOnlyAllowlistedKeys() throws {
    let object = try JSONSerialization.jsonObject(with: DiagnosticsSnapshot.fixture.encoded()) as! [String: Any]
    XCTAssertEqual(Set(object.keys), ["schema", "revision", "route", "focused_id", "modal_state", "playback", "presentation"])
}
```

- [ ] **Step 2: Verify RED**

Run: `swift test --filter DiagnosticsRegistryTests`

Expected: build failure because support target does not exist.

- [ ] **Step 3: Implement isolated product/target**

Give the harness app a distinct bundle ID and `VONDEL_TEST_HARNESS`. Production `VondelTV` source membership and dependencies exclude all harness files. Publish snapshots atomically with monotonic revisions.

- [ ] **Step 4: Verify GREEN and build isolation**

Run `swift test --filter DiagnosticsRegistryTests`, `xcodegen generate`, then unsigned builds of `VondelTVHarness` and `VondelTV`.

Expected: PASS; production target does not link `VondelTestSupport`.

- [ ] **Step 5: Commit**

```bash
git add Package.swift project.yml Sources/VondelTestSupport Apps/tvOSTestHarness Tests/VondelTestSupportTests
git commit -m "feat(test-harness): isolate Apple TV diagnostics"
```

### Task 3: XCUITest native-driver agent

**Files:**
- Create: `Tests/VondelTVHarnessUITests/AppleTVNativeDriver.swift`
- Create: `Tests/VondelTVHarnessUITests/DriverProtocol.swift`
- Create: `Tests/VondelTVHarnessUITests/DriverServer.swift`
- Create: `Tests/VondelTVHarnessUITests/AppleTVNativeDriverTests.swift`
- Modify: `project.yml`

**Interfaces:**
- Implements `vondel_driver_command_v1` through a one-client nonce-authenticated loopback/Unix NDJSON session.
- Uses `XCUIRemote.shared.press` and returns route, unique focused semantic ID, visible IDs, modal, playback, and presentation.

- [ ] **Step 1: Write failing remote/focus tests**

```swift
func testRightPressReportsNewFocusedSemanticID() throws {
    try driver.launch(.fixture)
    let before = try driver.snapshot()
    let after = try driver.press(.right)
    XCTAssertNotEqual(before.focusedID, after.focusedID)
    XCTAssertEqual(after.requestID, "run:1")
}
```

- [ ] **Step 2: Verify RED**

Run the generated `VondelTVHarnessUITests` scheme.

Expected: failure because driver does not exist.

- [ ] **Step 3: Implement native actions and strict observations**

Map only the closed actions to `XCUIRemote`; find focus only through `hasFocus`; fail on zero/multiple meaningful focus; ignore `vondel.harness.*`; cap frames at 1 MiB; require nonce, request correlation, monotonic revisions, timeouts, screenshot, lifecycle, and clean shutdown.

- [ ] **Step 4: Verify GREEN**

Run the UI-test scheme on an Apple TV simulator and run harness `resume_movie.yaml` against the live agent.

Expected: PASS with no coordinate calls.

- [ ] **Step 5: Commit**

```bash
git add Tests/VondelTVHarnessUITests project.yml
git commit -m "feat(test-harness): drive Apple TV remote semantics"
```

### Task 4: Focus and playback acceptance

**Files:**
- Create: `Apps/tvOSTestHarness/FocusTrapFixtureView.swift`
- Create: `Tests/VondelTVHarnessUITests/HarnessAcceptanceTests.swift`
- Modify: `Apps/tvOSTestHarness/VondelTVHarnessApp.swift`
- Modify: `ORIGINALITY.md`

**Interfaces:**
- Provides a harness-only deliberate trap and normal controller bootstrap.
- Proves the shared scenario, shortest trap replay, playback identity/quality/timeline, resume, and Back focus restoration.

- [ ] **Step 1: Write failing acceptance test**

Run `resume_movie.yaml`, crawl the deliberate trap, require replay `down,right`, then assert h264, at least 720p, advancing timeline, resumable progress, and origin-focus restoration.

- [ ] **Step 2: Verify RED**

Run the harness CLI against `VondelTVHarnessUITests`.

Expected: trap/playback assertion fails before wiring exists.

- [ ] **Step 3: Implement harness-only trap and test-controller bootstrap**

Bootstrap material enters only the separate harness app through inherited environment/file descriptor and is never a production switch. Use normal public client APIs and the production playback validator.

- [ ] **Step 4: Verify GREEN**

Expected: normal scenario passes; deliberate trap exits as a product failure with shortest replay and sanitized evidence; reset/replay reproduces it.

- [ ] **Step 5: Commit**

```bash
git add Apps/tvOSTestHarness Tests/VondelTVHarnessUITests ORIGINALITY.md
git commit -m "test(tv): verify Apple harness journeys"
```

### Task 5: Release absence and final acceptance

**Files:**
- Create: `scripts/verify-test-support-absent.sh`
- Create: `scripts/verify-test-support-absent_test.sh`
- Create: `docs/originality/apple-test-driver.md`
- Modify: `.github/workflows/ci.yml`
- Modify: `README.md`
- Modify: `ORIGINALITY.md`

**Interfaces:**
- Produces a self-tested release scanner for frameworks, symbols, strings, source membership, switches, identifiers, fixture terms, and credentials.

- [ ] **Step 1: Write failing scanner canary test**

Inject each forbidden marker into temporary fake bundles and require failure; require a clean minimal bundle to pass.

- [ ] **Step 2: Verify RED**

Run: `./scripts/verify-test-support-absent_test.sh`

Expected: FAIL because scanner does not exist.

- [ ] **Step 3: Implement scanner and CI**

Use `otool -L`, `nm`, `strings`, bundle traversal, and project membership. Positively scan `VondelTVHarness` to prove canaries are discoverable.

- [ ] **Step 4: Run full acceptance**

Run full warnings-as-errors tests, XcodeGen, Release `VondelTV` build, scanner, harness simulator journey, clean-room self-test, authorized sanitized audit, and `git diff --check`. Run on an uncontested Apple TV before claiming hardware acceptance.

- [ ] **Step 5: Record evidence and commit**

```bash
git add scripts docs/originality .github README.md ORIGINALITY.md
git commit -m "test(tv): prove Apple harness release isolation"
```
