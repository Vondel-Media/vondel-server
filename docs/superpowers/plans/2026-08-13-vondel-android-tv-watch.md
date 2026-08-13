# Vondel Android TV Watch Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Stage & Chapters movie-and-series Watch flow with Media3 playback and exact-scope local resume behavior on Android TV.

**Architecture:** A new `:watch` Android library owns Watch models, composition, scoped persistence, Media3 adaptation, and Compose TV surfaces in focused files. The TV app injects production dependencies normally and fixture transport only into debug AndroidTest launches; mobile remains unchanged.

**Tech Stack:** Kotlin 2.1.20, Android API 26+, Jetpack Compose BOM 2025.04.01, KSP 2.1.20-1.0.32, Media3 1.6.1, Room 2.7.1, coroutines 1.10.2, kotlinx.serialization 1.8.1, AndroidX Test.

**Spec:** `vondel-server/docs/superpowers/specs/2026-08-13-vondel-tv-watch-vertical-slice-design.md`

## Global Constraints

- Work in `vondel-android`; commands assume that repository root is the cwd.
- Begin only after the contract-fixture plan is committed and reviewed.
- Keep mobile compiling; add no Watch UI to phone or tablet.
- Play and release artifacts contain no fixture token, fixture origin, synthetic media, or hidden demo mode.
- Accept playback URLs only from validated protocol-v3 plans.
- Key state by exact `AccountScope` plus `ScopeLease` generation.
- Do not implement Live TV, IPTV, DVR, EPG, `.strm`, or arbitrary remote-stream shortcuts.
- Follow strict red-green TDD and commit after every task.

---

### Task 1: Create the Watch module and decode contracts

**Files:**
- Modify: `settings.gradle.kts`
- Modify: `gradle/libs.versions.toml`
- Create: `watch/build.gradle.kts`
- Create: `watch/src/main/kotlin/media/vondel/watch/WatchDocument.kt`
- Create: `watch/src/main/kotlin/media/vondel/watch/PlaybackPlan.kt`
- Create: `watch/src/main/kotlin/media/vondel/watch/PlaybackPlanValidator.kt`
- Create: `watch/src/test/kotlin/media/vondel/watch/WatchDocumentTest.kt`
- Create: `watch/src/test/kotlin/media/vondel/watch/PlaybackPlanValidatorTest.kt`

**Interfaces:**
- Produces serializable `WatchDocument`, `WatchItem`, `WatchProgress`, `WatchDetail`, `Season`, and `Episode` values.
- Produces `PlaybackPlanValidator.validate(plan, origin, now): ValidatedPlaybackPlan`.
- Consumes contract fixtures through `VONDEL_CONTRACTS_ROOT`, falling back only to the standard adjacent checkout.

- [ ] **Step 1: Write failing fixture and validator tests**

```kotlin
val validated = PlaybackPlanValidator.validate(
    plan = decodeContractFixture("playback/plan_4242.json"),
    origin = ServerOrigin("http://127.0.0.1:18080"),
    now = Instant.parse("2026-08-13T12:00:00Z"),
)
assertEquals(PlaybackDelivery.OriginalHttp, validated.delivery)
assertEquals("h264", validated.videoCodec)
```

Assert stable series ordering, unique positive file IDs, ignored additive fields, and rejection of all named negative fixtures.

- [ ] **Step 2: Verify RED**

Run: `./gradlew :watch:test --tests '*WatchDocumentTest' --tests '*PlaybackPlanValidatorTest' --console=plain`

Expected: configuration failure because `:watch` does not exist.

- [ ] **Step 3: Add the module, models, and strict validator**

Define:

```kotlin
interface WatchCatalogSource {
    suspend fun home(lease: ScopeLease): ResponseEnvelope<WatchDocument>
    suspend fun detail(contentId: String, lease: ScopeLease): ResponseEnvelope<WatchDetail>
}

data class ValidatedPlaybackPlan(
    val streamUri: URI,
    val startSeconds: Double,
    val durationSeconds: Double,
    val seekable: Boolean,
    val headers: Map<String, String>,
)
```

Use closed known enums with an explicit unknown decode path where additive compatibility is required. Reject unsupported delivery/protocol/container/codecs, invalid identity/timeline, expired plans, non-loopback HTTP, and disallowed cross-origin resolution.

- [ ] **Step 4: Verify GREEN**

Run: `./gradlew :watch:test --tests '*WatchDocumentTest' --tests '*PlaybackPlanValidatorTest' --console=plain`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add settings.gradle.kts gradle/libs.versions.toml watch
git commit -m "feat(watch): validate Android TV Watch contracts"
```

### Task 2: Stage & Chapters and Room-backed progress

**Files:**
- Modify: `watch/build.gradle.kts`
- Modify: `gradle/libs.versions.toml`
- Create: `watch/src/main/kotlin/media/vondel/watch/WatchHomeComposer.kt`
- Create: `watch/src/main/kotlin/media/vondel/watch/progress/WatchCheckpointEntity.kt`
- Create: `watch/src/main/kotlin/media/vondel/watch/progress/WatchProgressDao.kt`
- Create: `watch/src/main/kotlin/media/vondel/watch/progress/WatchProgressDatabase.kt`
- Create: `watch/src/main/kotlin/media/vondel/watch/progress/WatchProgressCoordinator.kt`
- Create: `watch/src/test/kotlin/media/vondel/watch/WatchHomeComposerTest.kt`
- Create: `watch/src/test/kotlin/media/vondel/watch/progress/WatchProgressCoordinatorTest.kt`
- Create: `watch/src/androidTest/kotlin/media/vondel/watch/progress/WatchProgressDatabaseTest.kt`

**Interfaces:**
- Produces chapters in closed order `ContinueWatching`, `Films`, `Series`.
- Produces Room database files keyed by an opaque SHA-256 digest of exact scope.
- Produces `record(event: PlaybackProgressEvent, lease: ScopeLease)` with stale-lease rejection.

- [ ] **Step 1: Write failing composer, coordinator, and Room tests**

Cover continue stage precedence, featured fallback, empty chapter omission, exact-scope database isolation, close-before-switch, monotonic timestamps, stale generation, long and short completion thresholds, seek-only non-completion, and idempotent batching through a recording `ProgressSyncSink`.

```kotlin
assertEquals(listOf(ContinueWatching, Films, Series), home.chapters.map { it.kind })
assertNull(database.checkpoints().findByKey(completedKey))
```

- [ ] **Step 2: Verify RED**

Run: `./gradlew :watch:testDebugUnitTest :watch:compileDebugAndroidTestKotlin --console=plain`

Expected: build failure because composer and persistence types do not exist.

- [ ] **Step 3: Implement composition and scoped Room storage**

Add KSP `2.1.20-1.0.32`, Room runtime/KTX/compiler/testing `2.7.1`, and their plugin/library aliases to `gradle/libs.versions.toml`. Define:

```kotlin
interface WatchProgressStore : AutoCloseable {
    val scope: AccountScope
    suspend fun checkpoints(): List<WatchCheckpoint>
    suspend fun write(checkpoint: WatchCheckpoint)
}

fun interface ProgressSyncSink {
    suspend fun submit(checkpoints: List<WatchCheckpoint>, lease: ScopeLease)
}
```

Store IDs, positions, durations, completion, and timestamps only—never titles or URLs. Use a transaction for compare-and-write monotonicity. Inject `Clock` and cadence; default cadence is 15 seconds.

- [ ] **Step 4: Verify GREEN**

Run unit tests and the Room instrumentation test on an emulator or managed device. If no device is available, build the AndroidTest APK and report execution as unclaimed rather than passing.

- [ ] **Step 5: Commit**

```bash
git add gradle/libs.versions.toml watch
git commit -m "feat(watch): persist scoped Android progress"
```

### Task 3: Media3 playback session and state authority

**Files:**
- Modify: `watch/build.gradle.kts`
- Modify: `gradle/libs.versions.toml`
- Create: `watch/src/main/kotlin/media/vondel/watch/playback/PlaybackState.kt`
- Create: `watch/src/main/kotlin/media/vondel/watch/playback/PlaybackEngine.kt`
- Create: `watch/src/main/kotlin/media/vondel/watch/playback/Media3PlaybackEngine.kt`
- Create: `watch/src/main/kotlin/media/vondel/watch/playback/PlaybackSessionController.kt`
- Create: `watch/src/test/kotlin/media/vondel/watch/playback/PlaybackSessionControllerTest.kt`
- Create: `watch/src/androidTest/kotlin/media/vondel/watch/playback/Media3FixtureTest.kt`

**Interfaces:**
- Produces `StateFlow<PlaybackState>` and suspend commands `load`, `play`, `pause`, `seekBy`, `seekTo`, and `stop`.
- Consumes only `ValidatedPlaybackPlan`.

- [ ] **Step 1: Write failing pure state-machine tests**

```kotlin
sealed interface PlaybackState {
    data object Idle : PlaybackState
    data object Loading : PlaybackState
    data class Ready(val snapshot: PlaybackSnapshot) : PlaybackState
    data class Playing(val snapshot: PlaybackSnapshot) : PlaybackState
    data class Paused(val snapshot: PlaybackSnapshot) : PlaybackState
    data class Seeking(val snapshot: PlaybackSnapshot) : PlaybackState
    data class Buffering(val snapshot: PlaybackSnapshot) : PlaybackState
    data object Completed : PlaybackState
    data class Failed(val failure: PlaybackFailure) : PlaybackState
}
```

Test legal transitions, clamped seeks, bounded repeats, serialized callbacks, stale session rejection, completion only after time advancement, and idempotent stop.

- [ ] **Step 2: Verify RED**

Run: `./gradlew :watch:testDebugUnitTest --tests '*PlaybackSessionControllerTest' --console=plain`

Expected: build failure because playback types do not exist.

- [ ] **Step 3: Implement Media3 adapter**

Add `androidx.media3:media3-exoplayer` and `media3-ui`. Map `Player.Listener`, playback state, `isPlaying`, discontinuity reasons, errors, duration, current position, and tracks into one coroutine-owned controller. Release the player and remove listeners exactly once on stop.

- [ ] **Step 4: Verify GREEN and generated-media playback**

Run: `./gradlew :watch:testDebugUnitTest :watch:assembleDebugAndroidTest --console=plain`

Run `Media3FixtureTest` on an emulator/device against the loopback fixture service before claiming native playback execution.

- [ ] **Step 5: Commit**

```bash
git add gradle/libs.versions.toml watch
git commit -m "feat(playback): add Media3 TV Watch session"
```

### Task 4: Stage & Chapters, Backdrop Ledger, and Season Stage Compose UI

**Files:**
- Create: `watch/src/main/kotlin/media/vondel/watch/ui/WatchRoute.kt`
- Create: `watch/src/main/kotlin/media/vondel/watch/ui/WatchHome.kt`
- Create: `watch/src/main/kotlin/media/vondel/watch/ui/MovieDetail.kt`
- Create: `watch/src/main/kotlin/media/vondel/watch/ui/SeriesDetail.kt`
- Create: `watch/src/main/kotlin/media/vondel/watch/ui/WatchFocusModel.kt`
- Create: `watch/src/test/kotlin/media/vondel/watch/ui/WatchFocusModelTest.kt`
- Create: `watch/src/test/kotlin/media/vondel/watch/ui/WatchSemanticsTest.kt`
- Modify: `navigation/src/main/kotlin/media/vondel/navigation/VondelRoot.kt`
- Modify: `navigation/build.gradle.kts`

**Interfaces:**
- Produces routes `Home`, `Movie(contentId)`, `Series(contentId)`, and `Player(selection)`.
- Produces deterministic focus restoration from origin ID to surviving item, chapter, or stage action.
- Consumes presentation theme and Watch models without changing mobile navigation.

- [ ] **Step 1: Write failing focus and semantics tests**

Assert chapter and reading order, stage-first focus, origin restoration, nearest-survivor fallback, detail primary action, stable episode order, no dead actions, headings, content descriptions, and animator-scale behavior.

- [ ] **Step 2: Verify RED**

Run: `./gradlew :watch:testDebugUnitTest :navigation:testDebugUnitTest --console=plain`

Expected: build failure because UI and focus types do not exist.

- [ ] **Step 3: Implement approved Compose TV surfaces**

Use lazy rows with stable keys, explicit `FocusRequester` edges, `focusRestorer`, semantic headings, Nocturne Atlas tokens, and client-owned empty/error states. Replace only the TV Watch destination's empty content; `VondelMobileRoot` remains unchanged. Render explicit loading, content, recoverable-failure, and unavailable states; cached content may remain visible during refresh failure.

- [ ] **Step 4: Verify GREEN and app builds**

```bash
./gradlew :watch:testDebugUnitTest :navigation:testDebugUnitTest :app-mobile:assembleDebug :app-tv:assembleDebug --console=plain
```

Expected: `BUILD SUCCESSFUL`.

- [ ] **Step 5: Commit**

```bash
git add watch navigation
git commit -m "feat(tv): add Android TV Watch browsing"
```

### Task 5: Quiet Timeline, overlay occupancy, and lifecycle checkpoints

**Files:**
- Create: `watch/src/main/kotlin/media/vondel/watch/ui/QuietTimeline.kt`
- Create: `watch/src/main/kotlin/media/vondel/watch/ui/TvPlayback.kt`
- Create: `watch/src/main/kotlin/media/vondel/watch/ui/PlaybackOccupancy.kt`
- Create: `watch/src/test/kotlin/media/vondel/watch/ui/QuietTimelineTest.kt`
- Modify: `presentation/src/main/kotlin/media/vondel/presentation/OverlayPolicy.kt`
- Modify: `presentation/src/test/kotlin/media/vondel/presentation/OverlayPolicyTest.kt`

**Interfaces:**
- Produces protected caption, active-control, and upper critical-notice regions.
- Emits checkpoint events for pause, completed seek, background, error, exit, and completion.

- [ ] **Step 1: Write failing timeline and occupancy tests**

Cover stable-playback auto-hide, controls pinned while accessibility focus is inside, two-stage Back, ten-second seek, spoken time semantics, preferred-caption toggle, campaign postponement, lifecycle progress events, and safe messages for invalid plan, network interruption, decode failure, authentication expiry, and authorization failure.

- [ ] **Step 2: Verify RED**

Run: `./gradlew :watch:testDebugUnitTest :presentation:test --console=plain`

Expected: build failure because timeline and occupancy mapping do not exist.

- [ ] **Step 3: Implement Quiet Timeline**

Use Compose for the client-owned information hierarchy and Media3 for playback. Toggle only a preferred usable caption track. Show audio only when Media3 reports more than one selectable audio group and route selection through Media3's track-selection API; otherwise hide it. Active system controls remain authoritative over custom overlays.

- [ ] **Step 4: Verify GREEN**

Run: `./gradlew :watch:testDebugUnitTest :presentation:test --console=plain`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add watch presentation
git commit -m "feat(tv): add Quiet Timeline controls"
```

### Task 6: Debug-only fixture injection and Android TV acceptance

**Files:**
- Create: `watch/src/debug/kotlin/media/vondel/watch/fixture/FixtureWatchCatalogSource.kt`
- Create: `watch/src/debug/kotlin/media/vondel/watch/fixture/FixturePlaybackPlanSource.kt`
- Modify: `app-tv/src/main/kotlin/media/vondel/app/tv/TvActivity.kt`
- Create: `app-tv/src/androidTest/kotlin/media/vondel/app/tv/WatchJourneyTest.kt`
- Modify: `app-tv/build.gradle.kts`
- Modify: `README.md`
- Modify: `ORIGINALITY.md`
- Modify: `docs/originality/visual-system.md`

**Interfaces:**
- Consumes instrumentation arguments `watchFixtureOrigin` and `watchFixtureToken` only from debug source code.
- Release variant contains no fixture classes and no install/network exception added for a fixture host. APK inspection must prove the fixture token and debug source type names are absent.

- [ ] **Step 1: Write failing Android TV journey test**

Start from Watch and assert movie → details → Play → Pause → seek → exit → Resume, then series → season → episode → Play, using test tags and semantics rather than visible copy.

- [ ] **Step 2: Verify RED**

Run: `./gradlew :app-tv:assembleDebugAndroidTest --console=plain`

Expected: compilation failure because fixture sources and journey tags do not exist.

- [ ] **Step 3: Implement debug-only bootstrap**

Select fixture sources only when both instrumentation arguments are present in a debuggable build. The normal activity continues to use production composition and never defaults to loopback or a fixture token. Add a release-variant assertion that fixture class names and `fixture-watch-token` are absent from the APK.

- [ ] **Step 4: Run the full evidence matrix**

```bash
./gradlew test :app-mobile:assembleDebug :app-tv:assembleDebug :app-tv:assembleDebugAndroidTest --console=plain --rerun-tasks
VONDEL_CONTRACTS_ROOT=../vondel-client-contracts ./scripts/verify-clean-room_test.sh
git diff --check
```

Run the Watch journey on an emulator and then an uncontested Android TV device before claiming hardware acceptance.

- [ ] **Step 5: Record evidence and commit**

Record contract authority, Media3/Room versions and licenses, generated-media source, exact test counts, device status, and sanitized audit result. Extend `DiagnosticRedactorTest` to prove complete media URLs, queries, titles, profile IDs, and playback headers are absent from diagnostic output.

```bash
git add app-tv watch README.md ORIGINALITY.md docs/originality
git commit -m "test(tv): verify Android TV Watch journey"
```
