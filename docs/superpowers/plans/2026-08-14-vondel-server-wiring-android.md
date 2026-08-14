# Vondel Android Server-Wiring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Connect the Android TV Watch slice to a real Vondel server — discovery, onboarding, credential and pairing sign-in, real Watch documents, real playback plans, and two-way progress sync — replacing the shipped placeholder Watch experience without moving a client boundary the slice established.

**Architecture:** HTTP transport and the server probes go into `:core-network`, which stays a plain JVM module so it is unit-testable without a device. A new `:discovery` Android library owns the `NsdManager` responder browse, because it needs Android APIs. A new `:onboarding` Android library owns the Compose TV surfaces. Production catalog, plan, and progress sources live in `:watch` beside the interfaces they implement, in `main` rather than `debug`. `:app-tv` replaces `ShippedWatchExperience` with a composition that supplies them.

**Tech Stack:** Kotlin 2.1.20, Android API 26+, Jetpack Compose BOM 2025.04.01, KSP 2.1.20-1.0.32, Media3 1.6.1, Room 2.7.1, coroutines 1.10.2, kotlinx.serialization 1.8.1, OkHttp for transport, AndroidX Test.

**Spec:** `vondel-server/docs/superpowers/specs/2026-08-14-vondel-client-server-wiring-design.md`

## Global Constraints

- Work in `vondel-android`; commands assume that repository root is the cwd.
- Begin only after the server plan's identity, capability, and Watch document tasks are committed and their conformance evidence is stable.
- Clean room: use only the Vondel contracts repository, invented fixtures, public Android documentation, and the approved Vondel designs. Do not read any reference client, and do not read the Apple repository.
- Keep mobile compiling; add no Watch UI to phone or tablet.
- `ServerOrigin` is not relaxed: HTTPS for every remote host, plain HTTP only on loopback. Add no cleartext domain to any network security configuration, in any variant, for a development or private-range host. Reach a development server through a loopback reverse forward instead.
- Play and release artifacts contain no fixture class, fixture token, fixture origin, development origin, synthetic media, or hidden demo mode.
- Accept a playback URI only from a validated protocol-v3 plan. There is no fallback to a catalogue-provided or user-provided URI.
- Key every piece of state by exact `AccountScope` plus `ScopeLease` generation, and create a new generation on sign-in, profile switch, server switch, and identity change — not on a token refresh that preserves identity.
- No token, refresh token, profile token, device code, user code, or match code reaches a log, diagnostic, crash report, or content description.
- Do not implement Live TV, IPTV, DVR, EPG, `.strm`, or arbitrary remote-stream shortcuts.
- Follow strict red-green TDD and commit after every task.

---

### Task 1: Server discovery, transport, and identity confirmation

**Files:**
- Modify: `settings.gradle.kts`
- Modify: `gradle/libs.versions.toml`
- Modify: `core-network/build.gradle.kts`
- Create: `core-network/src/main/kotlin/media/vondel/network/ServerCandidate.kt`
- Create: `core-network/src/main/kotlin/media/vondel/network/ServerIdentityProbe.kt`
- Create: `core-network/src/main/kotlin/media/vondel/network/CapabilityProbe.kt`
- Create: `core-network/src/test/kotlin/media/vondel/network/ServerIdentityProbeTest.kt`
- Create: `discovery/build.gradle.kts`
- Create: `discovery/src/main/AndroidManifest.xml`
- Create: `discovery/src/main/kotlin/media/vondel/discovery/ServerDiscovery.kt`
- Create: `discovery/src/main/kotlin/media/vondel/discovery/NsdServerDiscovery.kt`
- Create: `discovery/src/test/kotlin/media/vondel/discovery/ServerDiscoveryTest.kt`

**Interfaces:**
- Produces `ServerCandidate(displayName, advertisedServerId, proposedOrigin, reachable)`.
- Produces `interface ServerDiscovery { fun candidates(): Flow<List<ServerCandidate>> }` with an `NsdManager` implementation browsing `_vondel._tcp`.
- Produces `ServerIdentityProbe.confirm(origin, expecting): ServerIdentity` against `/api/v1/server/identity`, requiring the returned identifier to equal the advertised one.
- Produces `CapabilityProbe.capabilities(origin): CapabilitySet` against `/api/v1/capabilities`.

- [ ] **Step 1: Write failing discovery and probe tests**

Drive discovery through an injected service-browser fake so no unit test touches the network. Cover candidate mapping from a TXT record, deduplication by advertised identifier across interfaces, stable ordering by name, a cleartext non-loopback candidate marked unreachable rather than promoted, a candidate with a missing or malformed identifier discarded, and a typed origin validated through the existing `ServerOrigin` with its rejection cases.

```kotlin
val confirmed = ServerIdentityProbe(transport).confirm(
    origin = ServerOrigin("https://server.example"),
    expecting = "advertised-identifier",
)
assertEquals("advertised-identifier", confirmed.serverId)
assertFailsWith<ServerIdentityMismatch> { probe.confirm(origin, expecting = "other-identifier") }
```

- [ ] **Step 2: Verify RED**

Run: `./gradlew :core-network:test :discovery:testDebugUnitTest --console=plain`

Expected: configuration failure because `:discovery` does not exist.

- [ ] **Step 3: Add the modules, transport, and probes**

Add `:discovery` to `settings.gradle.kts` and add OkHttp and its mock web server to the version catalogue. Keep `:core-network` a plain JVM module so probes are testable without a device; the `NsdManager` browse is the only Android-dependent piece and it lives in `:discovery`.

Read the TXT record keys `txtvers`, `id`, `name`, `api`, and `scheme`, and treat every advertised value as an untrusted hint: build the origin through `ServerOrigin` and use the advertised address for nothing afterwards. Declare no cleartext permission for discovered hosts.

- [ ] **Step 4: Verify GREEN**

Run: `./gradlew :core-network:test :discovery:testDebugUnitTest --console=plain`

Expected: `BUILD SUCCESSFUL`, with the existing `ServerOrigin` tests unchanged.

- [ ] **Step 5: Commit**

```bash
git add settings.gradle.kts gradle/libs.versions.toml core-network discovery
git commit -m "feat(discovery): discover Vondel servers on the local network"
```

### Task 2: Credential sign-in, session lifecycle, and scope derivation

**Files:**
- Modify: `core-identity/build.gradle.kts`
- Create: `core-network/src/main/kotlin/media/vondel/network/V1AuthClient.kt`
- Create: `core-network/src/main/kotlin/media/vondel/network/AuthenticatedTransport.kt`
- Create: `core-identity/src/main/kotlin/media/vondel/identity/SessionStore.kt`
- Create: `core-identity/src/main/kotlin/media/vondel/identity/AccountActivation.kt`
- Create: `core-network/src/test/kotlin/media/vondel/network/V1AuthClientTest.kt`
- Create: `core-identity/src/test/kotlin/media/vondel/identity/SessionStoreTest.kt`
- Create: `core-identity/src/test/kotlin/media/vondel/identity/AccountActivationTest.kt`

**Interfaces:**
- Produces `V1AuthClient` covering login, refresh, current account, profile listing, and profile PIN verification against `/api/v1`.
- Produces `SessionStore` binding `CredentialVault` and `TokenRefreshCoordinator`: proactive refresh inside the last fifth of the advertised lifetime, single-flight per lease, atomic rotation of both tokens, session end on a 401 from refresh.
- Produces `AccountActivation.derive(identity, account, profile): ScopeLease` with the legacy-default organization and a fresh generation.
- Produces `AuthenticatedTransport` attaching the bearer token, the profile header, and the profile token header, and mapping status codes onto the five safe categories.

- [ ] **Step 1: Write failing auth, session, and scope tests**

Against a mock web server, cover: login decoding; 401 mapping to authentication expiry; 403 with a profile-unverified body mapping to authorization failure and never retried as another profile; a rate-limited response mapping to network interruption with bounded backoff; proactive refresh firing once for concurrent callers; rotation writing both tokens atomically and invalidating the in-memory pair when the keystore write fails; a failed refresh ending the session and removing the vault entry while leaving the scoped progress database alone.

Cover scope derivation: the organization is the legacy default and no organization header is ever sent on `/api/v1`; two origins reporting the same server identifier derive the same scope; a refresh returning a different account identity produces a new generation; a refresh preserving identity does not.

- [ ] **Step 2: Verify RED**

Run: `./gradlew :core-network:test :core-identity:testDebugUnitTest --console=plain`

Expected: build failure because the client, session store, and activation do not exist.

- [ ] **Step 3: Implement the client, session store, and activation**

Reuse `CredentialVault`, `AndroidKeystoreSecretStore`, and `TokenRefreshCoordinator` rather than introducing parallel machinery. Hold the password only for the duration of the login call. Attach the profile token returned by PIN verification to every profile-scoped request for the activation, and drop it when the activation ends.

- [ ] **Step 4: Verify GREEN**

Run: `./gradlew :core-network:test :core-identity:testDebugUnitTest --console=plain`

Expected: `BUILD SUCCESSFUL`.

- [ ] **Step 5: Commit**

```bash
git add core-identity core-network
git commit -m "feat(identity): sign in against a Vondel server"
```

### Task 3: Android TV onboarding surfaces

**Files:**
- Modify: `settings.gradle.kts`
- Create: `onboarding/build.gradle.kts`
- Create: `onboarding/src/main/AndroidManifest.xml`
- Create: `onboarding/src/main/kotlin/media/vondel/onboarding/OnboardingRoute.kt`
- Create: `onboarding/src/main/kotlin/media/vondel/onboarding/OnboardingModel.kt`
- Create: `onboarding/src/main/kotlin/media/vondel/onboarding/ServerPicker.kt`
- Create: `onboarding/src/main/kotlin/media/vondel/onboarding/ManualOriginEntry.kt`
- Create: `onboarding/src/main/kotlin/media/vondel/onboarding/CredentialSignIn.kt`
- Create: `onboarding/src/main/kotlin/media/vondel/onboarding/ProfilePicker.kt`
- Create: `onboarding/src/test/kotlin/media/vondel/onboarding/OnboardingModelTest.kt`
- Create: `onboarding/src/test/kotlin/media/vondel/onboarding/OnboardingSemanticsTest.kt`

**Interfaces:**
- Produces the five-step route: find server, choose method, sign in, choose profile, watch.
- Produces `OnboardingModel` as the single state authority, exposing exactly loading, content, recoverable failure, and unavailable per step.
- Consumes discovery, probes, and the auth client from Tasks 1 and 2 through interfaces, never concrete types.

- [ ] **Step 1: Write failing onboarding model and semantics tests**

Cover: discovery populating the list while manual entry stays reachable by one directional move; selecting a candidate probing identity before anything is typed; Back preserving the previous step's input; a server becoming unreachable mid-flow returning to step one with the origin preserved; a relaunch with valid stored tokens skipping to Watch, with invalid tokens landing on sign-in, and with an unreachable origin landing on server selection; a profile with a PIN requiring verification before the activation opens; the primary action holding initial focus on every step; password and PIN fields marked secure so their contents are never rendered or spoken; a user code exposed with a character-by-character spoken form; and animator-scale behaviour matching the Watch surfaces.

- [ ] **Step 2: Verify RED**

Run: `./gradlew :onboarding:testDebugUnitTest --console=plain`

Expected: configuration failure because `:onboarding` does not exist.

- [ ] **Step 3: Implement the onboarding surfaces**

Use explicit `FocusRequester` edges, `focusRestorer`, semantic headings, and Nocturne Atlas tokens, matching the Watch surfaces' focus discipline. Render every step's four explicit states; no step may spin without an explanation, and no step may dead-end. Persist the last successful origin and profile in ordinary storage; persist nothing secret there.

- [ ] **Step 4: Verify GREEN and app builds**

```bash
./gradlew :onboarding:testDebugUnitTest :app-mobile:assembleDebug :app-tv:assembleDebug --console=plain
```

Expected: `BUILD SUCCESSFUL`, with `:app-mobile` untouched by this task.

- [ ] **Step 5: Commit**

```bash
git add settings.gradle.kts onboarding
git commit -m "feat(tv): add Android TV onboarding"
```

### Task 4: Production Watch catalogue and playback plan sources

**Files:**
- Modify: `watch/build.gradle.kts`
- Create: `watch/src/main/kotlin/media/vondel/watch/net/HttpWatchCatalogSource.kt`
- Create: `watch/src/main/kotlin/media/vondel/watch/net/HttpPlaybackPlanSource.kt`
- Create: `watch/src/main/kotlin/media/vondel/watch/net/PlaybackPlanLifecycle.kt`
- Create: `watch/src/test/kotlin/media/vondel/watch/net/HttpWatchCatalogSourceTest.kt`
- Create: `watch/src/test/kotlin/media/vondel/watch/net/HttpPlaybackPlanSourceTest.kt`
- Modify: `app-tv/src/main/kotlin/media/vondel/app/tv/WatchExperience.kt`
- Modify: `app-tv/build.gradle.kts`

**Interfaces:**
- Produces `HttpWatchCatalogSource : WatchCatalogSource` reading `/api/v1/watch/home` and `/api/v1/watch/items/{content_id}`, answering inside the caller's lease.
- Produces `HttpPlaybackPlanSource : PlaybackPlanSource` posting `/api/v1/playback/start`, validating through the existing `PlaybackPlanValidator`, and exposing replan and session release.
- Produces `PlaybackPlanLifecycle` deciding revalidate, replace, or fail from a plan's remaining lifetime and a sixty-second floor.
- Replaces `ShippedWatchExperience`'s null sources with the production ones when an activation exists, keeping the not-connected rendering when it does not.

- [ ] **Step 1: Write failing transport and lifecycle tests**

Decode the contracts' Watch fixtures through the production source against a mock web server, asserting the same movie, series, episode order, and unique positive file identifiers the slice already asserts, plus rejection of a response failing the document's own validation and of a response arriving under a stale lease.

For plans, assert explicitly that the production source performs **none** of the debug source's three substitutions:

```kotlin
val validated = source.plan(selection)
assertEquals(servedPlan.stream.headers, validated.headers)
assertEquals(servedPlan.timeline.playerStartSeconds, validated.startSeconds)
assertEquals(servedPlan.source.durationSeconds, validated.durationSeconds)
```

Also assert: a fresh attempt identifier per attempt including per retry; an explicit start position of zero for play and the checkpoint position for resume; capabilities declared at a tier the client can substantiate and never higher; a plan revalidated immediately before load; a plan inside the sixty-second floor replaced before playback starts; a plan expiring during playback replaced without a visible interruption unless the replacement fails; a 401 on the stream refreshing the session then replanning; `header_refresh` of `session` re-fetching headers, and a `session` plan without a usable refresh URL failing closed; and session release on exit.

- [ ] **Step 2: Verify RED**

Run: `./gradlew :watch:testDebugUnitTest --tests '*HttpWatchCatalogSourceTest' --tests '*HttpPlaybackPlanSourceTest' --console=plain`

Expected: build failure because the production sources do not exist.

- [ ] **Step 3: Implement the sources and replace the placeholder**

Send the profile header and profile token on every profile-scoped request. Never mutate a served plan. Map every failure onto the five safe categories through the transport from Task 2.

In `:app-tv`, keep the variant seam exactly as it is: the release and debug variants both compose the production experience, and the debug variant swaps in fixtures only when an instrumentation run explicitly asks. The production composition supplies the catalogue source, plan source, checkpoint sink, and lease from the active activation.

- [ ] **Step 4: Verify GREEN and app builds**

```bash
./gradlew :watch:testDebugUnitTest :app-tv:assembleDebug :app-mobile:assembleDebug --console=plain
```

Expected: `BUILD SUCCESSFUL`, with the existing validator and composer tests unchanged.

- [ ] **Step 5: Commit**

```bash
git add watch app-tv
git commit -m "feat(watch): browse and play against a Vondel server"
```

### Task 5: Progress synchronization

**Files:**
- Create: `watch/src/main/kotlin/media/vondel/watch/net/HttpProgressSyncSink.kt`
- Create: `watch/src/main/kotlin/media/vondel/watch/progress/ProgressMerge.kt`
- Create: `watch/src/test/kotlin/media/vondel/watch/net/HttpProgressSyncSinkTest.kt`
- Create: `watch/src/test/kotlin/media/vondel/watch/progress/ProgressMergeTest.kt`
- Modify: `watch/src/main/kotlin/media/vondel/watch/progress/WatchProgressCoordinator.kt`

**Interfaces:**
- Produces `HttpProgressSyncSink : ProgressSyncSink` posting `/api/v1/sync/progress` with the profile header, always sending `updated_at` and never `force_overwrite: true`.
- Produces `ProgressMerge.apply(serverRows, local, fileIds)` implementing newest-updated-at-wins per content and episode identity.
- Leaves the coordinator's completion rule, cadence, and lifecycle triggers unchanged.

- [ ] **Step 1: Write failing sink and merge tests**

Cover: the wire shape exactly, including an episode's media item identifier being the episode's content identifier; batching with coalescing by key keeping the newest; bounded exponential backoff with jitter on failure; nothing dropped on failure and everything re-sent on the next flush, surviving a process restart through the existing Room storage; a sync failure never surfacing as a playback error and never blocking the player; and a contract `ignored` result recorded without being retried as an error.

Cover the merge: a newer server row replacing the local one including its completed state; a newer local row kept and queued; a server row for an item absent from the document held as a resume hint that cannot become a checkpoint until a file identifier is known; and the completion divergence for content under twenty minutes and under the five-percent floor resolving to the server's value when the server's timestamp is newer.

- [ ] **Step 2: Verify RED**

Run: `./gradlew :watch:testDebugUnitTest --tests '*HttpProgressSyncSinkTest' --tests '*ProgressMergeTest' --console=plain`

Expected: build failure because the sink and merge do not exist.

- [ ] **Step 3: Implement the sink and merge**

Attach the sink to the existing coordinator through its existing sink parameter; do not change how or when the coordinator records. Perform the inbound merge on activation and on each Watch document refresh, before composition, so the home screen never renders a stale continue-watching row it could have known was stale.

- [ ] **Step 4: Verify GREEN**

Run: `./gradlew :watch:testDebugUnitTest --console=plain`

Expected: `BUILD SUCCESSFUL`, with the coordinator's existing tests unchanged.

- [ ] **Step 5: Commit**

```bash
git add watch
git commit -m "feat(watch): sync Watch progress with the server"
```

### Task 6: Device-code pairing sign-in

**Files:**
- Create: `core-network/src/main/kotlin/media/vondel/network/DevicePairingClient.kt`
- Create: `core-identity/src/main/kotlin/media/vondel/identity/DevicePairingCoordinator.kt`
- Create: `onboarding/src/main/kotlin/media/vondel/onboarding/PairingSignIn.kt`
- Create: `core-identity/src/test/kotlin/media/vondel/identity/DevicePairingCoordinatorTest.kt`
- Create: `onboarding/src/test/kotlin/media/vondel/onboarding/PairingSignInTest.kt`
- Modify: `onboarding/src/main/kotlin/media/vondel/onboarding/OnboardingModel.kt`

**Interfaces:**
- Produces `DevicePairingClient` covering start, poll, and the capability probe against `/api/v1/auth/device`.
- Produces `DevicePairingCoordinator` as a state machine over pending, approved, denied, expired, and consumed, polling at the server-supplied interval and stopping at the server-supplied expiry.
- Consumes `device_pairing_v1` from the capability document; the pairing option is hidden when it is absent.

- [ ] **Step 1: Write failing pairing tests**

Cover: the poll interval honoured and never shortened; polling stopping at the expiry; each terminal outcome rendering distinct copy; approval yielding the same token pair and account a login yields, continuing to profile selection; the device code never displayed, logged, or placed in a diagnostic; the user code and match code both displayed; and the pairing option absent when the capability is absent.

- [ ] **Step 2: Verify RED**

Run: `./gradlew :core-identity:testDebugUnitTest :onboarding:testDebugUnitTest --console=plain`

Expected: build failure because the pairing client, coordinator, and screen do not exist.

- [ ] **Step 3: Implement pairing and its screen**

Render the user code with generous type and an accessible character-by-character spoken form, the verification URI as text, and the match code as the confirmation value. A scannable representation of the complete verification URI is permitted; the device code is not part of any displayed value.

- [ ] **Step 4: Verify GREEN**

Run: `./gradlew :core-identity:testDebugUnitTest :onboarding:testDebugUnitTest --console=plain`

Expected: `BUILD SUCCESSFUL`.

- [ ] **Step 5: Commit**

```bash
git add core-network core-identity onboarding
git commit -m "feat(identity): add device pairing sign-in"
```

### Task 7: End-to-end acceptance and evidence

**Files:**
- Create: `app-tv/src/androidTest/kotlin/media/vondel/app/tv/ServerWiringJourneyTest.kt`
- Modify: `core-identity/src/test/kotlin/media/vondel/identity/DiagnosticRedactorTest.kt`
- Modify: `app-tv/build.gradle.kts`
- Modify: `README.md`
- Modify: `ORIGINALITY.md`
- Modify: `docs/originality/visual-system.md`

**Interfaces:**
- Consumes a test backend supplied entirely at run time through instrumentation arguments: the contracts repository's disposable fixture service reached over a loopback reverse forward, or a development server whose origin and account come from the same arguments.
- Produces no default origin, no default account, and no committed credential anywhere in the repository.

- [ ] **Step 1: Write the failing journey test**

Assert the whole path against a supplied backend: discovery or manual entry, identity confirmation, credential sign-in, profile selection with PIN, Watch home, movie details, play, pause, seek, exit, resume from the server-known position, series, season, episode, play. Then repeat sign-in through pairing by driving the approval through the backend's API rather than a browser. Assert test tags and semantics rather than visible copy. Skip with the missing argument named when no backend is supplied; a skip is not evidence.

- [ ] **Step 2: Verify RED**

Run: `./gradlew :app-tv:assembleDebugAndroidTest --console=plain`

Expected: compilation failure because the journey wiring and tags do not exist.

- [ ] **Step 3: Prove the release artifact is clean**

Extend the redaction tests to prove that access tokens, refresh tokens, profile tokens, device codes, user codes, match codes, complete media URLs, query strings, titles, and profile identifiers are absent from diagnostic output. Extend the release-variant assertion to prove that fixture class names, the fixture token, and any development origin string are absent from the APK, and that no variant's network security configuration permits cleartext to anything but loopback.

- [ ] **Step 4: Run the full evidence matrix**

```bash
./gradlew test :app-mobile:assembleDebug :app-tv:assembleDebug :app-tv:assembleDebugAndroidTest --console=plain --rerun-tasks
VONDEL_CONTRACTS_ROOT=../vondel-client-contracts ./scripts/verify-clean-room_test.sh
git diff --check
```

Run the journey on an emulator against the fixture service, then against a supplied development server reached over a loopback reverse forward, then on an uncontested Android TV device before claiming hardware acceptance.

- [ ] **Step 5: Record evidence and commit**

Record the contract authority, the OkHttp and NsdManager usage with versions and licences, the exact test counts, which journey runs were emulator and which were device, and the sanitized audit result. Name no origin, account, or credential.

```bash
git add app-tv core-identity README.md ORIGINALITY.md docs/originality
git commit -m "test(tv): verify Android TV server wiring"
```
