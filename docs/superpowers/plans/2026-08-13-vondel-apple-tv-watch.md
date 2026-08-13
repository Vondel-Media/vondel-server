# Vondel Apple TV Watch Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Stage & Chapters movie-and-series Watch flow with native tvOS playback and exact-scope local resume behavior.

**Architecture:** Foundation-only Watch models, composition, validation, and progress live in `VondelCore`; AVPlayer adaptation lives in a focused `VondelPlayback` target; SwiftUI Watch surfaces live in `VondelNavigation`. Fixture transport is supplied only to tests and debug UI-test launch configuration.

**Tech Stack:** Swift 6, SwiftUI, AVFoundation/AVKit, XCTest, tvOS 17+, JSON fixtures from `vondel-client-contracts`.

**Spec:** `vondel-server/docs/superpowers/specs/2026-08-13-vondel-tv-watch-vertical-slice-design.md`

## Global Constraints

- Work in `vondel-apple`; commands assume that repository root is the cwd.
- Begin only after the contract-fixture plan is committed and reviewed.
- Keep iOS and macOS compiling; add no Watch UI to those targets.
- Production builds contain no fixture token, fixture origin, synthetic media, or hidden demo mode.
- Use AVFoundation/AVKit through client-owned interfaces; never accept an arbitrary catalog URL.
- Key state by exact `AccountScope` plus `ScopeLease` generation.
- Do not implement Live TV, IPTV, DVR, EPG, `.strm`, or arbitrary remote-stream shortcuts.
- Follow strict red-green TDD and commit after every task.

---

### Task 1: Watch documents and playback-plan validator

**Files:**
- Create: `Sources/VondelCore/Watch/WatchDocument.swift`
- Create: `Sources/VondelCore/Watch/PlaybackPlan.swift`
- Create: `Sources/VondelCore/Watch/PlaybackPlanValidator.swift`
- Create: `Tests/VondelCoreTests/Watch/WatchDocumentTests.swift`
- Create: `Tests/VondelCoreTests/Watch/PlaybackPlanValidatorTests.swift`

**Interfaces:**
- Produces: `WatchDocument`, `WatchItem`, `WatchProgress`, `MovieDetail`, `SeriesDetail`, `Season`, and `Episode` as `Decodable & Sendable` values.
- Produces: `PlaybackPlanValidator.validate(_:origin:now:) throws -> ValidatedPlaybackPlan`.
- Consumes: the exact positive and negative JSON fixtures committed by the contract-fixture plan.

- [ ] **Step 1: Write failing fixture-decoding tests**

Load contract JSON through a test helper that requires `VONDEL_CONTRACTS_ROOT` or discovers the standard adjacent `vondel-client-contracts` checkout. Assert movie `4242`, series `8080`, stable episode order, positive unique file IDs, and unknown additive fields being ignored. Do not copy contract fixtures into the client repository.

Use this validator surface in tests:

```swift
let validated = try PlaybackPlanValidator().validate(
    decodedPlan,
    origin: try ServerOrigin("http://127.0.0.1:18080"),
    now: Date(timeIntervalSince1970: 1_786_644_800)
)
XCTAssertEqual(validated.delivery, .originalHTTP)
XCTAssertEqual(validated.videoCodec, .h264)
```

- [ ] **Step 2: Verify RED**

Run: `swift test --filter 'WatchDocumentTests|PlaybackPlanValidatorTests'`

Expected: build failure because the Watch types and validator do not exist.

- [ ] **Step 3: Implement minimal closed models and validator**

Define:

```swift
public protocol WatchCatalogSource: Sendable {
    func home(for lease: ScopeLease) async throws -> ResponseEnvelope<WatchDocument>
    func detail(contentID: String, for lease: ScopeLease) async throws -> ResponseEnvelope<WatchDetail>
}

public struct ValidatedPlaybackPlan: Sendable, Equatable {
    public let streamURL: URL
    public let startSeconds: TimeInterval
    public let durationSeconds: TimeInterval
    public let seekable: Bool
    public let headers: [String: String]
}
```

Reject every named negative fixture, non-loopback HTTP, disallowed cross-origin resolution, unsupported delivery/protocol/container/codecs, invalid identity/timeline, and expired plans.

- [ ] **Step 4: Verify GREEN**

Run: `swift test --filter 'WatchDocumentTests|PlaybackPlanValidatorTests'`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add Sources/VondelCore/Watch Tests/VondelCoreTests/Watch
git commit -m "feat(watch): validate TV Watch contracts"
```

### Task 2: Stage & Chapters composition and exact-scope progress

**Files:**
- Create: `Sources/VondelCore/Watch/WatchHomeComposer.swift`
- Create: `Sources/VondelCore/Watch/WatchProgressStore.swift`
- Create: `Sources/VondelCore/Watch/FileWatchProgressStore.swift`
- Create: `Sources/VondelCore/Watch/WatchProgressCoordinator.swift`
- Create: `Tests/VondelCoreTests/Watch/WatchHomeComposerTests.swift`
- Create: `Tests/VondelCoreTests/Watch/WatchProgressCoordinatorTests.swift`

**Interfaces:**
- Produces: `WatchHomeComposer.compose(document:checkpointByItem:) throws -> WatchHome`.
- Produces: chapters in closed order `.continueWatching`, `.films`, `.series` and a featured stage.
- Produces: `WatchProgressCoordinator.record(_:for:) async throws` accepting only the active `ScopeLease`.

- [ ] **Step 1: Write failing composition and progress tests**

Cover continue-watching stage precedence, featured fallback, empty chapter omission, deterministic item order, exact-scope isolation, stale-generation rejection, monotonic writes, completion at long-content thresholds, short-content 90 percent, seek-only non-completion, and idempotent batching through a recording `ProgressSyncSink`.

```swift
XCTAssertEqual(home.chapters.map(\.kind), [.continueWatching, .films, .series])
XCTAssertEqual(home.stage.item.contentID, "4242")
XCTAssertNil(try await store.checkpoint(for: completedKey))
```

- [ ] **Step 2: Verify RED**

Run: `swift test --filter 'WatchHomeComposerTests|WatchProgressCoordinatorTests'`

Expected: build failure because composer and progress types do not exist.

- [ ] **Step 3: Implement actors and atomic file storage**

Define:

```swift
public protocol WatchProgressStore: Sendable {
    var scope: AccountScope { get }
    func checkpoints() async throws -> [WatchCheckpoint]
    func write(_ checkpoint: WatchCheckpoint) async throws
    func close() async
}

public protocol ProgressSyncSink: Sendable {
    func submit(_ checkpoints: [WatchCheckpoint], lease: ScopeLease) async throws
}
```

`FileWatchProgressStore` uses application-support storage, a scope-derived SHA-256 directory key, write-to-temporary plus atomic replace, and no titles or URLs. The coordinator records on a 15-second cadence and explicit lifecycle events; inject the clock for tests.

- [ ] **Step 4: Verify GREEN**

Run: `swift test --filter 'WatchHomeComposerTests|WatchProgressCoordinatorTests'`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add Sources/VondelCore/Watch Tests/VondelCoreTests/Watch
git commit -m "feat(watch): compose scoped Watch progress"
```

### Task 3: Native AVPlayer session

**Files:**
- Modify: `Package.swift`
- Create: `Sources/VondelPlayback/PlaybackSession.swift`
- Create: `Sources/VondelPlayback/AVPlayerPlaybackSession.swift`
- Create: `Sources/VondelPlayback/PlaybackSessionController.swift`
- Create: `Tests/VondelPlaybackTests/PlaybackSessionControllerTests.swift`
- Create: `Tests/VondelPlaybackTests/AVPlayerFixtureTests.swift`

**Interfaces:**
- Produces: Swift package product `VondelPlayback` depending on `VondelCore` and linking AVFoundation/AVKit.
- Produces: `PlaybackSessionController.state: AsyncStream<PlaybackState>` and commands `load`, `play`, `pause`, `seekBy`, `seekTo`, and `stop`.

- [ ] **Step 1: Write failing state-machine tests with a fake engine**

```swift
public enum PlaybackState: Sendable, Equatable {
    case idle, loading, ready(PlaybackSnapshot), playing(PlaybackSnapshot)
    case paused(PlaybackSnapshot), seeking(PlaybackSnapshot), buffering(PlaybackSnapshot)
    case completed, failed(PlaybackFailure)
}
```

Test legal transitions, clamped seeks, bounded remote repeats, stale callback rejection, completion reporting only after time advancement, and stop idempotency.

- [ ] **Step 2: Verify RED**

Run: `swift test --filter PlaybackSessionControllerTests`

Expected: build failure because `VondelPlayback` and its state types do not exist.

- [ ] **Step 3: Implement controller and AVPlayer adapter**

Use an actor for state authority. Bridge `AVPlayerItem.status`, time control status, end notification, periodic time observer, seek completion, duration, and media-selection characteristics. Remove every observer on `stop` and `deinit`. Apply only validator-approved headers.

Add these package boundaries in `Package.swift`:

```swift
.library(name: "VondelPlayback", targets: ["VondelPlayback"]),
.target(name: "VondelPlayback", dependencies: ["VondelCore"], linkerSettings: [
    .linkedFramework("AVFoundation"), .linkedFramework("AVKit"),
]),
.testTarget(name: "VondelPlaybackTests", dependencies: ["VondelCore", "VondelPlayback"]),
```

Add `VondelPlayback` to `VondelNavigation` dependencies so only the native adapter crosses into the TV presentation target.

- [ ] **Step 4: Verify GREEN with synthetic media**

Run: `swift test --filter 'PlaybackSessionControllerTests|AVPlayerFixtureTests'`

Expected: PASS; fixture integration may skip only when the fixture service origin is not supplied by the test harness, with the missing variable named.

- [ ] **Step 5: Commit**

```bash
git add Package.swift Sources/VondelPlayback Tests/VondelPlaybackTests
git commit -m "feat(playback): add native Apple TV session"
```

### Task 4: Backdrop Ledger, Season Stage, and focus restoration

**Files:**
- Create: `Sources/VondelNavigation/Watch/WatchRoute.swift`
- Create: `Sources/VondelNavigation/Watch/WatchHomeView.swift`
- Create: `Sources/VondelNavigation/Watch/MovieDetailView.swift`
- Create: `Sources/VondelNavigation/Watch/SeriesDetailView.swift`
- Create: `Sources/VondelNavigation/Watch/WatchFocusModel.swift`
- Modify: `Sources/VondelNavigation/VondelRootView.swift`
- Create: `Tests/VondelNavigationTests/WatchFocusModelTests.swift`
- Create: `Tests/VondelNavigationTests/WatchRenderingTests.swift`

**Interfaces:**
- Produces: `WatchRoute.home`, `.movie(contentID:)`, `.series(contentID:)`, and `.player(WatchSelection)`.
- Produces: `WatchFocusModel.restore(origin:survivingIDs:) -> WatchFocusTarget`.
- Consumes: `WatchHome`, `WatchDetail`, presentation theme, and playback selection from Tasks 1-2.

- [ ] **Step 1: Write failing focus and rendering tests**

Assert stage-first reading order, chapter order, origin-card restoration, nearest surviving fallback, primary detail action focus, episode order, no dead trailer/save actions, accessibility labels, and reduced-motion behavior.

- [ ] **Step 2: Verify RED**

Run: `swift test --filter 'WatchFocusModelTests|WatchRenderingTests'`

Expected: build failure because Watch views and focus model do not exist.

- [ ] **Step 3: Implement the approved native views**

Use `LazyHStack` chapters with stable content IDs, `@FocusState`, semantic headings, and Nocturne Atlas tokens. Replace only the `.watch` television destination's empty content with `WatchHomeView`; all non-TV shells remain unchanged. Render explicit loading, content, recoverable-failure, and unavailable states; cached content may remain visible during refresh failure.

- [ ] **Step 4: Verify GREEN and compile every target**

```bash
swift test --filter 'WatchFocusModelTests|WatchRenderingTests'
xcodegen generate
xcodebuild -project Vondel.xcodeproj -scheme VondelMobile -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO build
xcodebuild -project Vondel.xcodeproj -scheme VondelTV -destination 'generic/platform=tvOS Simulator' CODE_SIGNING_ALLOWED=NO build
xcodebuild -project Vondel.xcodeproj -scheme VondelMac -destination 'generic/platform=macOS' CODE_SIGNING_ALLOWED=NO build
```

Expected: all commands exit `0`.

- [ ] **Step 5: Commit**

```bash
git add Sources/VondelNavigation Tests/VondelNavigationTests
git commit -m "feat(tv): add Apple TV Watch browsing"
```

### Task 5: Quiet Timeline and lifecycle progress wiring

**Files:**
- Create: `Sources/VondelNavigation/Watch/QuietTimelineView.swift`
- Create: `Sources/VondelNavigation/Watch/TVPlaybackView.swift`
- Create: `Sources/VondelNavigation/Watch/PlaybackOccupancy.swift`
- Modify: `Sources/VondelNavigation/Watch/WatchRoute.swift`
- Modify: `Sources/VondelCore/Presentation/OverlayPolicy.swift`
- Create: `Tests/VondelNavigationTests/QuietTimelineTests.swift`
- Modify: `Tests/VondelCoreTests/Presentation/OverlayPolicyTests.swift`

**Interfaces:**
- Produces protected occupancy for captions, visible controls, and the upper critical-notice zone.
- Consumes native playback snapshots and emits pause, seek, background, error, exit, and completion progress events.

- [ ] **Step 1: Write failing timeline, occupancy, and checkpoint tests**

Cover control auto-hide only during stable playback, controls pinned while accessibility focus is present, Back/Menu two-stage behavior, ±10-second seek, spoken time labels, caption occupancy, campaign postponement, lifecycle checkpoint calls, and safe messages for invalid plan, network interruption, decode failure, authentication expiry, and authorization failure.

- [ ] **Step 2: Verify RED**

Run: `swift test --filter 'QuietTimelineTests|OverlayPolicyTests'`

Expected: build failure because playback views and occupancy mapping do not exist.

- [ ] **Step 3: Implement Quiet Timeline**

Render the timeline in SwiftUI while AVPlayer remains the engine. Caption toggling controls only the preferred usable legible option. Show audio selection only through a platform-native picker; otherwise hide it. Never place custom overlays over system player controls.

- [ ] **Step 4: Verify GREEN**

Run: `swift test --filter 'QuietTimelineTests|OverlayPolicyTests|WatchProgressCoordinatorTests'`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add Sources/VondelNavigation Sources/VondelCore/Presentation Tests
git commit -m "feat(tv): add Quiet Timeline playback controls"
```

### Task 6: Test-only fixture bootstrap and tvOS acceptance

**Files:**
- Create: `Tests/VondelCoreTests/Watch/FixtureWatchCatalogSource.swift`
- Create: `Tests/VondelCoreTests/Watch/FixtureWatchIntegrationTests.swift`
- Create: `Apps/tvOS/UITestWatchBootstrap.swift`
- Create: `Tests/VondelTVUITests/WatchJourneyTests.swift`
- Modify: `project.yml`
- Modify: `Apps/tvOS/VondelTVApp.swift`
- Modify: `README.md`
- Modify: `ORIGINALITY.md`

**Interfaces:**
- Consumes launch arguments `-vondel-ui-test-watch-origin` and `-vondel-ui-test-watch-token` only in `#if DEBUG` code compiled into the UI-test build.
- Production bootstrap remains fixture-free and uses no default origin or token. Release-build inspection must also prove the fixture token and debug source type names are absent.

- [ ] **Step 1: Write the failing end-to-end UI test**

Launch the disposable fixture service, then assert Watch → movie details → Play → Pause → seek → exit → Resume, followed by series → season → episode → Play. Assert stable accessibility identifiers rather than visible copy.

- [ ] **Step 2: Verify RED**

Run the generated `VondelTVUITests` scheme on a tvOS simulator.

Expected: FAIL because test-only bootstrap and journey wiring do not exist.

- [ ] **Step 3: Implement debug-only injection and app routing**

Keep the fixture source type out of release source membership. In `VondelTVApp`, select test bootstrap only when both named launch arguments exist in a DEBUG build; otherwise use production bootstrap.

- [ ] **Step 4: Run full acceptance evidence**

```bash
swift test -Xswiftc -warnings-as-errors
xcodegen generate
xcodebuild -project Vondel.xcodeproj -scheme VondelTV -destination 'generic/platform=tvOS Simulator' CODE_SIGNING_ALLOWED=NO build
xcodebuild -project Vondel.xcodeproj -scheme VondelMobile -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO build
xcodebuild -project Vondel.xcodeproj -scheme VondelMac -destination 'generic/platform=macOS' CODE_SIGNING_ALLOWED=NO build
VONDEL_CONTRACTS_ROOT=../vondel-client-contracts ./scripts/verify-clean-room_test.sh
git diff --check
```

Run the UI journey on a simulator and then an uncontested Apple TV device before claiming hardware acceptance.

- [ ] **Step 5: Record evidence and commit**

Update provenance with contract authority, dependencies, generated-media source, exact verification, and sanitized audit result. Extend `DiagnosticRedactorTests` to prove complete media URLs, queries, titles, profile IDs, and playback headers are absent from diagnostic output.

```bash
git add Apps project.yml Tests README.md ORIGINALITY.md docs/originality
git commit -m "test(tv): verify Apple TV Watch journey"
```
