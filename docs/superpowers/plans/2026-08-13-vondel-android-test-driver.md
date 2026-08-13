# Vondel Android TV Native Test Driver Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the completed Android/Fire TV client controllable by the shared semantic harness through device-level D-pad events, externally discoverable semantic IDs, sanitized Media3 observations, and a harness build absent from release.

**Architecture:** Production Compose UI retains stable test tags exposed as resource IDs. A separate `harness` build type adds an instrumentation driver and diagnostics wrappers; UIAutomator is authoritative for remote input/focus while Media3 listeners report typed quality facts.

**Tech Stack:** Kotlin, Jetpack Compose, UIAutomator, AndroidX Test, Media3/ExoPlayer, Gradle, Android TV/Fire TV.

**Spec:** `vondel-server/docs/superpowers/specs/2026-08-13-vondel-cross-platform-client-test-harness-design.md`

## Global Constraints

- Begin after Android Watch Tasks 1–6 and the shared harness protocol are reviewed.
- Stable semantic IDs ship; harness activity, sockets, launch arguments, diagnostics, and credentials must be absent from release.
- UIAutomator sends real device keys and observes focused nodes; coordinates and arbitrary key codes are prohibited.
- Human content descriptions remain separate from opaque semantic IDs.
- Remote cleartext remains prohibited; use `adb reverse` to a loopback origin.
- Use strict red-green TDD and commit after every task.

---

### Task 1: Semantic IDs and external focus discovery

**Files:**
- Create: `navigation/src/main/kotlin/media/vondel/navigation/SemanticId.kt`
- Modify: `navigation/src/main/kotlin/media/vondel/navigation/VondelRoot.kt`
- Modify: `watch/src/main/kotlin/media/vondel/watch/**`
- Create: `navigation/src/test/kotlin/media/vondel/navigation/SemanticIdTest.kt`
- Create: `app-tv/src/androidTest/kotlin/media/vondel/app/tv/SemanticFocusDiscoveryTest.kt`

**Interfaces:**
- Produces stable route/card/detail/episode/control/modal/error/overlay IDs through `testTag`.
- Enables `testTagsAsResourceId = true` at the TV semantics root.

- [ ] **Step 1: Write failing uniqueness and UIAutomator discovery tests**

```kotlin
@Test fun watchIdsAreUniqueStableAndTitleFree() {
    val ids = SemanticId.watchFixture(listOf("4242", "8080"), listOf("8080-s1-e1"))
    assertEquals(ids.size, ids.toSet().size)
    assertTrue("watch.movie.4242" in ids)
    assertFalse(ids.any { "Invented Crossing" in it })
}
```

- [ ] **Step 2: Verify RED**

Run unit tests and focused `SemanticFocusDiscoveryTest`.

Expected: failure because taxonomy/resource exposure does not exist.

- [ ] **Step 3: Implement semantic tags**

Assign tags independently from TalkBack labels; test every focusable Watch control has one unique externally discoverable ID and hidden/disabled nodes are not reported focusable.

- [ ] **Step 4: Verify GREEN**

Run: `./gradlew :navigation:test :app-tv:connectedDebugAndroidTest --console=plain`

Expected: PASS on a TV emulator with UIAutomator resolving the focused resource ID.

- [ ] **Step 5: Commit**

```bash
git add navigation watch app-tv
git commit -m "feat(testability): stabilize Android TV semantics"
```

### Task 2: Harness build type and typed diagnostics

**Files:**
- Modify: `app-tv/build.gradle.kts`
- Modify: `watch/build.gradle.kts`
- Create: `app-tv/src/harness/kotlin/media/vondel/app/tv/harness/HarnessTvActivity.kt`
- Create: `app-tv/src/harness/kotlin/media/vondel/app/tv/harness/HarnessDiagnosticsRegistry.kt`
- Create: `app-tv/src/harness/AndroidManifest.xml`
- Create: `watch/src/harness/kotlin/media/vondel/watch/harness/PlaybackDiagnosticsV1.kt`
- Create: `app-tv/src/test/kotlin/media/vondel/app/tv/harness/DiagnosticsAllowlistTest.kt`

**Interfaces:**
- Produces a distinct `harness` application ID/build type initialized from debug.
- Produces route, focus expectation, modal, presentation, and typed playback snapshot; types cannot represent URLs, headers, titles, account/profile IDs, tokens, or raw errors.

- [ ] **Step 1: Write failing source-set and allowlist tests**

```kotlin
@Test fun diagnosticsJsonHasClosedKeys() {
    assertEquals(setOf("schema","sequence","route","focused_id","modal_state","playback","presentation"), fixtureSnapshot().toJsonObject().keys)
}
```

- [ ] **Step 2: Verify RED**

Run: `./gradlew :app-tv:testDebugUnitTest :app-tv:assembleHarness --console=plain`

Expected: failure because harness build/support does not exist.

- [ ] **Step 3: Implement isolated composition**

Hoist production Watch route/playback/presentation state into injectable composition seams, then observe them from harness-only wrappers. Do not add privileged diagnostics interfaces to main source sets.

- [ ] **Step 4: Verify GREEN**

Run harness and release assembly plus allowlist tests.

Expected: harness compiles and release dependency/source graph excludes harness packages.

- [ ] **Step 5: Commit**

```bash
git add app-tv watch
git commit -m "feat(test-harness): isolate Android TV diagnostics"
```

### Task 3: UIAutomator native-driver agent

**Files:**
- Create: `app-tv/src/androidTest/kotlin/media/vondel/app/tv/harness/DriverProtocol.kt`
- Create: `app-tv/src/androidTest/kotlin/media/vondel/app/tv/harness/HarnessDriverServer.kt`
- Create: `app-tv/src/androidTest/kotlin/media/vondel/app/tv/harness/AndroidNativeDriverTest.kt`
- Modify: `app-tv/build.gradle.kts`

**Interfaces:**
- Implements `vondel_driver_command_v1` over a nonce-authenticated local-abstract-socket JSONL session reached by `adb forward`.
- Uses `UiDevice` for Up/Down/Left/Right/Center/Back, repeat, hold, lifecycle, relaunch, and screenshots; uses `UiObject2` for unique focused/visible IDs.

- [ ] **Step 1: Write failing real-key and ambiguity tests**

```kotlin
@Test fun dpadRightReturnsCorrelatedFocusedId() {
    val before = driver.observe("run:0")
    val after = driver.press("run:1", RemoteKey.Right)
    assertNotEquals(before.focusedId, after.focusedId)
    assertEquals("run:1", after.requestId)
}
```

- [ ] **Step 2: Verify RED**

Run: `./gradlew :app-tv:connectedHarnessAndroidTest --console=plain`

Expected: failure because driver server does not exist.

- [ ] **Step 3: Implement bounded socket/remote protocol**

Require random nonce, one client, 1 MiB messages, correlation, monotonic sequence, timeouts, clean shutdown, and no coordinate/action-code fields. Fail on zero/multiple meaningful focus when focus is expected.

- [ ] **Step 4: Verify GREEN**

Use `adb forward`, run connected test, then run shared `resume_movie.yaml` through the harness CLI.

Expected: PASS with real device-key events and no coordinates.

- [ ] **Step 5: Commit**

```bash
git add app-tv
git commit -m "feat(test-harness): drive Android TV remote semantics"
```

### Task 4: Media3 telemetry and focus/playback acceptance

**Files:**
- Create: `watch/src/harness/kotlin/media/vondel/watch/harness/Media3DiagnosticsListener.kt`
- Create: `app-tv/src/harness/kotlin/media/vondel/app/tv/harness/FocusTrapFixture.kt`
- Create: `app-tv/src/androidTest/kotlin/media/vondel/app/tv/harness/HarnessAcceptanceTest.kt`
- Modify: `core-identity/src/test/kotlin/media/vondel/identity/DiagnosticRedactorTest.kt`
- Modify: `ORIGINALITY.md`

**Interfaces:**
- Reports input formats, selected tracks, video size, MIME/codec/frame rate/bitrate/color facts, bandwidth estimate, buffering, discontinuity, closed errors, dropped frames, timeline, duration, and playing state.
- Provides a harness-only deliberate trap and normal test-controller bootstrap over `adb reverse`.

- [ ] **Step 1: Write failing independent playback/trap test**

Run shared scenario; require content/file identity, h264, at least 720p, advancing timeline, resume, and origin restoration; crawl deliberate trap and require shortest replay `down,right`.

- [ ] **Step 2: Verify RED**

Run connected harness tests and harness CLI.

Expected: telemetry or trap assertion fails before listener/fixture wiring.

- [ ] **Step 3: Implement Media3 listener and acceptance fixture**

Attach harness-only `AnalyticsListener`/`Player.Listener`; keep all sensitive fields structurally absent. Bootstrap through inherited instrumentation values, normal public APIs, and loopback `adb reverse`—never weaken release networking.

- [ ] **Step 4: Verify GREEN**

Expected: normal scenario passes; deliberate trap exits `1` with replay and sanitized evidence; reset/replay reproduces it.

- [ ] **Step 5: Commit**

```bash
git add watch app-tv core-identity ORIGINALITY.md
git commit -m "test(tv): verify Android harness journeys"
```

### Task 5: Release absence, emulator CI, and device acceptance

**Files:**
- Create: `scripts/verify-release-surface.sh`
- Create: `scripts/verify-release-surface_test.sh`
- Create: `docs/originality/android-test-driver.md`
- Modify: `.github/workflows/ci.yml`
- Modify: `README.md`
- Modify: `ORIGINALITY.md`

**Interfaces:**
- Produces a self-tested APK scanner for manifest, DEX packages, resources, socket/protocol markers, fixture terms, controller endpoints, switches, and credentials.

- [ ] **Step 1: Write failing scanner canary test**

Create temporary APK-like archives with each forbidden marker; require scanner failure and a clean canary pass.

- [ ] **Step 2: Verify RED**

Run: `./scripts/verify-release-surface_test.sh`

Expected: FAIL because scanner does not exist.

- [ ] **Step 3: Implement scanner and CI**

Inspect decoded manifest, DEX strings/classes, resources, archive names, and raw bytes. Positively scan the harness APK to prove detection. Add PR emulator journey and main-branch crawl jobs.

- [ ] **Step 4: Run full acceptance**

Run full unit tests; mobile/TV debug; harness app/test; release; connected harness test; release scanner; clean-room self-test; authorized sanitized audit; and `git diff --check`. Run with `ANDROID_SERIAL` on uncontested Android TV and Fire TV before claiming hardware acceptance.

- [ ] **Step 5: Record evidence and commit**

```bash
git add scripts docs/originality .github README.md ORIGINALITY.md
git commit -m "test(tv): prove Android harness release isolation"
```
