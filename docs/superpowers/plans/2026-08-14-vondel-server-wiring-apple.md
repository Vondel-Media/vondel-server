# Vondel Apple Server-Wiring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Connect the Apple TV Watch slice to a real Vondel server — discovery, onboarding, credential and pairing sign-in, real Watch documents, real playback plans, and two-way progress sync — without moving a single client boundary the slice established.

**Architecture:** Discovery, HTTP transport, session lifecycle, and scope derivation live in `VondelCore` beside the existing `Connection`, `Identity`, and `Watch` code. The production Watch catalog source, plan source, and progress sink implement the slice's existing protocols. Onboarding SwiftUI surfaces live in `VondelNavigation` and are mounted by the production root view, which today supplies no Watch ports at all.

**Tech Stack:** Swift 6, SwiftUI, Foundation `URLSession`, Network framework (`NWBrowser`/`NWConnection`) for Bonjour, AVFoundation/AVKit, XCTest, tvOS 17+.

**Spec:** `vondel-server/docs/superpowers/specs/2026-08-14-vondel-client-server-wiring-design.md`

## Global Constraints

- Work in `vondel-apple`; commands assume that repository root is the cwd.
- Begin only after the server plan's identity, capability, and Watch document tasks are committed and their conformance evidence is stable.
- Clean room: use only the Vondel contracts repository, invented fixtures, public Apple documentation, and the approved Vondel designs. Do not read any reference client, and do not read the Android repository.
- Keep iOS and macOS compiling; add no Watch UI to those targets. Onboarding may be built television-first.
- `ServerOrigin` is not relaxed: HTTPS for every remote host, plain HTTP only on loopback. Add no App Transport Security exception, in any configuration, for any host.
- Production builds contain no fixture token, fixture origin, development origin, synthetic media, or hidden demo mode.
- Accept a playback URL only from a validated protocol-v3 plan. There is no fallback to a catalog-provided or user-provided URL.
- Key every piece of state by exact `AccountScope` plus `ScopeLease` generation, and create a new generation on sign-in, profile switch, server switch, and identity change — not on a token refresh that preserves identity.
- No token, refresh token, profile token, device code, user code, or match code reaches a log, diagnostic, crash report, or accessibility label.
- Do not implement Live TV, IPTV, DVR, EPG, `.strm`, or arbitrary remote-stream shortcuts.
- Follow strict red-green TDD and commit after every task.

---

### Task 1: Server discovery and identity confirmation

**Files:**
- Create: `Sources/VondelCore/Connection/ServerCandidate.swift`
- Create: `Sources/VondelCore/Connection/ServerDiscovery.swift`
- Create: `Sources/VondelCore/Connection/BonjourServerDiscovery.swift`
- Create: `Sources/VondelCore/Connection/ServerIdentityProbe.swift`
- Create: `Tests/VondelCoreTests/Connection/ServerDiscoveryTests.swift`
- Create: `Tests/VondelCoreTests/Connection/ServerIdentityProbeTests.swift`
- Modify: `Package.swift`
- Modify: `Apps/tvOS/Info.plist`

**Interfaces:**
- Produces `ServerCandidate` carrying a display name, an advertised server identifier, a proposed origin string, and a reachability verdict.
- Produces `protocol ServerDiscovery: Sendable { func candidates(within: Duration) -> AsyncStream<[ServerCandidate]> }` with a Bonjour implementation browsing `_vondel._tcp`.
- Produces `ServerIdentityProbe.confirm(origin:expecting:) async throws -> ServerIdentity` fetching `/api/v1/server/identity` and requiring the returned identifier to equal the advertised one.
- Produces `CapabilityProbe.capabilities(origin:) async throws -> CapabilitySet` fetching `/api/v1/capabilities`.

- [ ] **Step 1: Write failing discovery and probe tests**

Drive discovery through an injected browser fake so no test touches the network. Cover candidate mapping from a TXT record, deduplication by advertised identifier across interfaces, stable ordering by name, a cleartext non-loopback candidate marked unreachable rather than promoted, a candidate with a missing or malformed identifier discarded, and a typed origin validated through the existing `ServerOrigin` rule with its rejection cases.

Cover the probe: an identifier mismatch throws, a missing endpoint surfaces as an unsupported-server outcome rather than a crash, and a confirmed identity yields the normalized origin the rest of the app uses.

```swift
let confirmed = try await ServerIdentityProbe(session: fake).confirm(
    origin: try ServerOrigin("https://server.example"),
    expecting: "advertised-identifier"
)
XCTAssertEqual(confirmed.serverID, "advertised-identifier")
```

- [ ] **Step 2: Verify RED**

Run: `swift test --filter 'ServerDiscoveryTests|ServerIdentityProbeTests'`

Expected: build failure because the discovery and probe types do not exist.

- [ ] **Step 3: Implement Bonjour discovery and the probes**

Browse with `NWBrowser` for the `_vondel._tcp` service in the local domain, resolve each result, and read the TXT record keys `txtvers`, `id`, `name`, `api`, and `scheme`. Treat every advertised value as an untrusted hint: build the origin through `ServerOrigin`, and use the advertised address for nothing after the origin exists.

Declare `NSBonjourServices` containing `_vondel._tcp` and a local-network usage description in the tvOS `Info.plist`. Link the Network framework in `Package.swift` for the `VondelCore` target.

- [ ] **Step 4: Verify GREEN**

Run: `swift test --filter 'ServerDiscoveryTests|ServerIdentityProbeTests|ServerOriginTests'`

Expected: PASS, with the existing origin tests unchanged.

- [ ] **Step 5: Commit**

```bash
git add Package.swift Sources/VondelCore/Connection Tests/VondelCoreTests/Connection Apps/tvOS/Info.plist
git commit -m "feat(connection): discover and validate Vondel servers"
```

### Task 2: Credential sign-in, session lifecycle, and scope derivation

**Files:**
- Create: `Sources/VondelCore/Identity/V1AuthClient.swift`
- Create: `Sources/VondelCore/Identity/SessionStore.swift`
- Create: `Sources/VondelCore/Identity/AccountActivation.swift`
- Create: `Sources/VondelCore/Identity/AuthenticatedTransport.swift`
- Create: `Tests/VondelCoreTests/Identity/V1AuthClientTests.swift`
- Create: `Tests/VondelCoreTests/Identity/SessionStoreTests.swift`
- Create: `Tests/VondelCoreTests/Identity/AccountActivationTests.swift`

**Interfaces:**
- Produces `V1AuthClient` covering login, refresh, current account, profile listing, and profile PIN verification against `/api/v1`.
- Produces `SessionStore` binding `CredentialVault` and `TokenRefreshCoordinator`: proactive refresh inside the last fifth of the advertised lifetime, single-flight per lease, atomic rotation of both tokens, session end on a 401 from refresh.
- Produces `AccountActivation.derive(identity:account:profile:) throws -> ScopeLease` building the scope with the legacy-default organization and a fresh generation.
- Produces `AuthenticatedTransport` attaching the bearer token, the profile header, and the profile token header, and mapping status codes onto the five safe categories.

- [ ] **Step 1: Write failing auth, session, and scope tests**

Cover, against a fake transport: a successful login decoding into the existing `V1LoginResponse`; a 401 mapping to authentication expiry; a 403 with a profile-unverified body mapping to authorization failure and never being retried as another profile; a rate-limited response mapping to network interruption with a bounded backoff; proactive refresh firing once for concurrent callers; rotation writing both tokens atomically and invalidating the in-memory pair when the vault write fails; a failed refresh ending the session and removing the vault entry while leaving the scoped progress store alone.

Cover scope derivation: the organization is the legacy default and no organization header is ever sent on `/api/v1`; two origins reporting the same server identifier derive the same scope; a refresh returning a different account identity produces a new generation; a refresh preserving identity does not.

```swift
let lease = try AccountActivation.derive(identity: identity, account: account, profile: profile)
XCTAssertEqual(lease.scope.organization, .legacyDefault)
XCTAssertNotEqual(lease.generation, previousLease.generation)
```

- [ ] **Step 2: Verify RED**

Run: `swift test --filter 'V1AuthClientTests|SessionStoreTests|AccountActivationTests'`

Expected: build failure because the client, session store, and activation types do not exist.

- [ ] **Step 3: Implement the client, session store, and activation**

Reuse `V1AuthContracts`, `CredentialVault`, and `TokenRefreshCoordinator` rather than introducing parallel machinery. Hold the password only for the duration of the login request. Attach the profile token returned by PIN verification to every profile-scoped request for the activation, and drop it when the activation ends.

- [ ] **Step 4: Verify GREEN**

Run: `swift test --filter 'V1AuthClientTests|SessionStoreTests|AccountActivationTests|CredentialVaultTests'`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add Sources/VondelCore/Identity Tests/VondelCoreTests/Identity
git commit -m "feat(identity): sign in against a Vondel server"
```

### Task 3: Apple TV onboarding surfaces

**Files:**
- Create: `Sources/VondelNavigation/Onboarding/OnboardingRoute.swift`
- Create: `Sources/VondelNavigation/Onboarding/ServerPickerView.swift`
- Create: `Sources/VondelNavigation/Onboarding/ManualOriginEntryView.swift`
- Create: `Sources/VondelNavigation/Onboarding/CredentialSignInView.swift`
- Create: `Sources/VondelNavigation/Onboarding/ProfilePickerView.swift`
- Create: `Sources/VondelNavigation/Onboarding/OnboardingModel.swift`
- Create: `Tests/VondelNavigationTests/OnboardingModelTests.swift`
- Create: `Tests/VondelNavigationTests/OnboardingRenderingTests.swift`
- Modify: `Sources/VondelNavigation/VondelRootView.swift`

**Interfaces:**
- Produces the five-step route: find server, choose method, sign in, choose profile, watch.
- Produces `OnboardingModel` as the single state authority, exposing exactly the states loading, content, recoverable failure, and unavailable per step.
- Consumes discovery, probes, and the auth client from Tasks 1 and 2 through protocols, never concrete types.
- Mounts onboarding from `VondelBootstrappedRootView` when no activation exists, leaving the resolved-presentation path unchanged.

- [ ] **Step 1: Write failing onboarding model and rendering tests**

Cover: discovery populating the list while manual entry stays reachable by one directional move; selecting a candidate probing identity before anything is typed; Back preserving the previous step's input; a server becoming unreachable mid-flow returning to step one with the origin preserved; a relaunch with valid stored tokens skipping to Watch, with invalid tokens landing on sign-in, and with an unreachable origin landing on server selection; a profile with a PIN requiring verification before the activation opens; the primary action holding initial focus on every step; secure entry for password and PIN so their contents are never rendered or spoken; and a user code exposed with a character-by-character spoken form.

- [ ] **Step 2: Verify RED**

Run: `swift test --filter 'OnboardingModelTests|OnboardingRenderingTests'`

Expected: build failure because the onboarding route, model, and views do not exist.

- [ ] **Step 3: Implement the onboarding surfaces**

Use `@FocusState`, semantic headings, and Nocturne Atlas tokens, matching the Watch surfaces' focus discipline. Render every step's four explicit states; no step may spin without an explanation, and no step may dead-end. Persist the last successful origin and profile outside the secure store; persist nothing secret there.

Mount onboarding in the production root view only. Every non-television shell stays exactly as it is.

- [ ] **Step 4: Verify GREEN and compile every target**

```bash
swift test --filter 'OnboardingModelTests|OnboardingRenderingTests'
xcodegen generate
xcodebuild -project Vondel.xcodeproj -scheme VondelTV -destination 'generic/platform=tvOS Simulator' CODE_SIGNING_ALLOWED=NO build
xcodebuild -project Vondel.xcodeproj -scheme VondelMobile -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO build
xcodebuild -project Vondel.xcodeproj -scheme VondelMac -destination 'generic/platform=macOS' CODE_SIGNING_ALLOWED=NO build
```

Expected: all commands exit `0`.

- [ ] **Step 5: Commit**

```bash
git add Sources/VondelNavigation/Onboarding Sources/VondelNavigation/VondelRootView.swift Tests/VondelNavigationTests
git commit -m "feat(tv): add Apple TV onboarding"
```

### Task 4: Production Watch catalog and playback plan sources

**Files:**
- Create: `Sources/VondelCore/Watch/HTTPWatchCatalogSource.swift`
- Create: `Sources/VondelCore/Watch/HTTPPlaybackPlanSource.swift`
- Create: `Sources/VondelCore/Watch/PlaybackPlanLifecycle.swift`
- Create: `Tests/VondelCoreTests/Watch/HTTPWatchCatalogSourceTests.swift`
- Create: `Tests/VondelCoreTests/Watch/HTTPPlaybackPlanSourceTests.swift`
- Modify: `Sources/VondelNavigation/VondelRootView.swift`

**Interfaces:**
- Produces `HTTPWatchCatalogSource: WatchCatalogSource` reading `/api/v1/watch/home` and `/api/v1/watch/items/{content_id}`, answering inside the caller's lease.
- Produces `HTTPPlaybackPlanSource` posting `/api/v1/playback/start`, validating through the existing `PlaybackPlanValidator`, and exposing replan and session release.
- Produces `PlaybackPlanLifecycle` deciding revalidate, replace, or fail from a plan's remaining lifetime and a sixty-second floor.
- Supplies the previously empty playback and occupancy ports of the production root view.

- [ ] **Step 1: Write failing transport and lifecycle tests**

Decode the contracts' Watch fixtures through the production source against a fake transport, asserting the same movie, series, episode order, and unique positive file identifiers the slice already asserts, plus rejection of a response that fails the document's own validation and of a response arriving under a stale lease.

For plans, assert: a fresh attempt identifier per attempt including per retry; an explicit start position of zero for play and the checkpoint position for resume; capabilities declared at a tier the client can substantiate and never higher; a plan revalidated immediately before load; a plan inside the sixty-second floor replaced before playback starts; a plan expiring during playback replaced without a visible interruption unless the replacement fails; a 401 on the stream refreshing the session then replanning; `header_refresh` of `session` re-fetching headers, and a `session` plan without a usable refresh URL failing closed; session release on exit; and no substitution of stream headers, player start, or source duration.

- [ ] **Step 2: Verify RED**

Run: `swift test --filter 'HTTPWatchCatalogSourceTests|HTTPPlaybackPlanSourceTests'`

Expected: build failure because the production sources do not exist.

- [ ] **Step 3: Implement the sources and wire the ports**

Send the profile header and profile token on every profile-scoped request. Never mutate a served plan. Map every failure onto the five safe categories through the transport from Task 2.

Supply the catalog source, plan source, and occupancy port to the root view from the active activation; when no activation exists the ports stay absent and the existing unavailable state renders, exactly as it does today.

- [ ] **Step 4: Verify GREEN**

Run: `swift test --filter 'HTTPWatchCatalogSourceTests|HTTPPlaybackPlanSourceTests|PlaybackPlanValidatorTests|WatchHomeComposerTests'`

Expected: PASS, with the existing validator and composer tests unchanged.

- [ ] **Step 5: Commit**

```bash
git add Sources/VondelCore/Watch Sources/VondelNavigation/VondelRootView.swift Tests/VondelCoreTests/Watch
git commit -m "feat(watch): browse and play against a Vondel server"
```

### Task 5: Progress synchronization

**Files:**
- Create: `Sources/VondelCore/Watch/HTTPProgressSyncSink.swift`
- Create: `Sources/VondelCore/Watch/ProgressMerge.swift`
- Create: `Tests/VondelCoreTests/Watch/HTTPProgressSyncSinkTests.swift`
- Create: `Tests/VondelCoreTests/Watch/ProgressMergeTests.swift`
- Modify: `Sources/VondelCore/Watch/WatchProgressCoordinator.swift`

**Interfaces:**
- Produces `HTTPProgressSyncSink: ProgressSyncSink` posting `/api/v1/sync/progress` with the profile header, always sending `updated_at` and never `force_overwrite: true`.
- Produces `ProgressMerge.apply(serverRows:to:fileIDs:)` implementing newest-updated-at-wins per content and episode identity.
- Leaves the coordinator's completion rule, cadence, and lifecycle triggers unchanged.

- [ ] **Step 1: Write failing sink and merge tests**

Cover: the wire shape exactly, including an episode's media item identifier being the episode's content identifier; batching with coalescing by key keeping the newest; bounded exponential backoff with jitter on failure; nothing dropped on failure and everything re-sent on the next flush; a sync failure never surfacing as a playback error and never blocking the player; and a contract `ignored` result recorded without being retried as an error.

Cover the merge: a newer server row replacing the local one including its completed state; a newer local row kept and queued; a server row for an item absent from the document held as a resume hint that cannot become a checkpoint until a file identifier is known; and the completion divergence for content under twenty minutes and under the five-percent floor resolving to the server's value when the server's timestamp is newer.

- [ ] **Step 2: Verify RED**

Run: `swift test --filter 'HTTPProgressSyncSinkTests|ProgressMergeTests'`

Expected: build failure because the sink and merge do not exist.

- [ ] **Step 3: Implement the sink and merge**

Attach the sink to the existing coordinator through its existing sink parameter; do not change how or when the coordinator records. Perform the inbound merge on activation and on each Watch document refresh, before composition, so the home screen never renders a stale continue-watching row it could have known was stale.

- [ ] **Step 4: Verify GREEN**

Run: `swift test --filter 'HTTPProgressSyncSinkTests|ProgressMergeTests|WatchProgressCoordinatorTests'`

Expected: PASS, with the coordinator's existing tests unchanged.

- [ ] **Step 5: Commit**

```bash
git add Sources/VondelCore/Watch Tests/VondelCoreTests/Watch
git commit -m "feat(watch): sync Watch progress with the server"
```

### Task 6: Device-code pairing sign-in

**Files:**
- Create: `Sources/VondelCore/Identity/DevicePairingClient.swift`
- Create: `Sources/VondelCore/Identity/DevicePairingCoordinator.swift`
- Create: `Sources/VondelNavigation/Onboarding/PairingSignInView.swift`
- Create: `Tests/VondelCoreTests/Identity/DevicePairingCoordinatorTests.swift`
- Create: `Tests/VondelNavigationTests/PairingSignInTests.swift`
- Modify: `Sources/VondelNavigation/Onboarding/OnboardingModel.swift`

**Interfaces:**
- Produces `DevicePairingClient` covering start, poll, and the capability probe against `/api/v1/auth/device`.
- Produces `DevicePairingCoordinator` as a state machine over pending, approved, denied, expired, and consumed, polling at the server-supplied interval and stopping at the server-supplied expiry.
- Consumes `device_pairing_v1` from the capability document; the pairing option is hidden when it is absent.

- [ ] **Step 1: Write failing pairing tests**

Cover: the poll interval honored and never shortened; polling stopping at the expiry; each terminal outcome rendering distinct copy; approval yielding the same token pair and account a login yields, continuing to profile selection; the device code never displayed, logged, or placed in a diagnostic; the user code and match code both displayed; and the pairing option absent when the capability is absent.

```swift
XCTAssertEqual(coordinator.pollCount(after: .seconds(9), interval: .seconds(5)), 2)
XCTAssertFalse(diagnostics.joined().contains(session.deviceCode))
```

- [ ] **Step 2: Verify RED**

Run: `swift test --filter 'DevicePairingCoordinatorTests|PairingSignInTests'`

Expected: build failure because the pairing client, coordinator, and view do not exist.

- [ ] **Step 3: Implement pairing and its screen**

Render the user code with generous type and an accessible character-by-character spoken form, the verification URI as text, and the match code as the confirmation value. A scannable representation of the complete verification URI is permitted; the device code is not part of any displayed value.

- [ ] **Step 4: Verify GREEN**

Run: `swift test --filter 'DevicePairingCoordinatorTests|PairingSignInTests|OnboardingModelTests'`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add Sources/VondelCore/Identity Sources/VondelNavigation/Onboarding Tests
git commit -m "feat(identity): add device pairing sign-in"
```

### Task 7: End-to-end acceptance and evidence

**Files:**
- Create: `Tests/VondelTVUITests/ServerWiringJourneyTests.swift`
- Modify: `Tests/VondelCoreTests/Identity/DiagnosticRedactorTests.swift`
- Modify: `Apps/tvOS/UITestWatchBootstrap.swift`
- Modify: `project.yml`
- Modify: `README.md`
- Modify: `ORIGINALITY.md`

**Interfaces:**
- Consumes a test backend supplied entirely at run time: the contracts repository's disposable fixture service, or a development server whose origin and account come from environment configuration.
- Produces no default origin, no default account, and no committed credential anywhere in the repository.

- [ ] **Step 1: Write the failing journey test**

Assert the whole path against a supplied backend: discovery or manual entry, identity confirmation, credential sign-in, profile selection with PIN, Watch home, movie details, play, pause, seek, exit, resume from the server-known position, series, season, episode, play. Then repeat sign-in through pairing by driving the approval through the backend's API rather than a browser. Assert stable accessibility identifiers rather than visible copy. Skip with the missing variable named when no backend is supplied; a skip is not evidence.

- [ ] **Step 2: Verify RED**

Run the generated `VondelTVUITests` scheme on a tvOS simulator.

Expected: FAIL because the journey wiring does not exist.

- [ ] **Step 3: Prove the release build is clean**

Extend the redaction tests to prove that access tokens, refresh tokens, profile tokens, device codes, user codes, match codes, complete media URLs, query strings, titles, and profile identifiers are absent from diagnostic output. Add a release-artifact assertion that no fixture source type name, fixture token, or development origin string is present.

- [ ] **Step 4: Run the full evidence matrix**

```bash
swift test -Xswiftc -warnings-as-errors
xcodegen generate
xcodebuild -project Vondel.xcodeproj -scheme VondelTV -destination 'generic/platform=tvOS Simulator' CODE_SIGNING_ALLOWED=NO build
xcodebuild -project Vondel.xcodeproj -scheme VondelMobile -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO build
xcodebuild -project Vondel.xcodeproj -scheme VondelMac -destination 'generic/platform=macOS' CODE_SIGNING_ALLOWED=NO build
VONDEL_CONTRACTS_ROOT=../vondel-client-contracts ./scripts/verify-clean-room_test.sh
git diff --check
```

Run the journey on a simulator against the fixture service, then against a supplied development server, then on an uncontested Apple TV device before claiming hardware acceptance.

- [ ] **Step 5: Record evidence and commit**

Update provenance with the contract authority, the new platform frameworks used, the exact verification commands, which journey runs were simulator and which were device, and the sanitized audit result. Name no origin, account, or credential.

```bash
git add Apps project.yml Tests README.md ORIGINALITY.md docs/originality
git commit -m "test(tv): verify Apple TV server wiring"
```
