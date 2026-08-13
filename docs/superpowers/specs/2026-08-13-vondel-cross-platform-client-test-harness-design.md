# Vondel Cross-Platform Client Test Harness Design

**Status:** Approved design

**Date:** 2026-08-13

## Purpose

Vondel needs a first-class, cross-platform client test system that proves remote-control navigation, focus reachability, playback identity and quality, accessibility, server-side effects, and presentation variants across Apple TV, Android TV, and Fire TV. The system must automate user actions without screen coordinates and must make failures deterministic and replayable.

The test harness is a product-quality subsystem, not a collection of fragile UI scripts. A rendered control that cannot be reached with a remote is a functional failure.

## Repository and architecture

Create a separate `vondel-client-test-harness` repository. It owns platform-neutral scenarios, focus exploration, server test-controller integration, orchestration, and evidence. Native platform drivers remain authoritative for actual input and observations.

The system has five components:

1. **Scenario engine.** Reads versioned YAML or JSON scenarios, resolves values returned by a seeded test run, executes actions, and evaluates assertions. Scenarios use semantic identifiers and expected behavior, never coordinates or platform implementation details.
2. **Focus explorer.** Sends real Up, Down, Left, Right, Select, and Back/Menu events and builds a directed graph of observed focus transitions for every reachable route and selected configuration.
3. **Native drivers.** XCUITest drives Apple clients. Compose Test and UIAutomator drive Android and Fire TV clients. Drivers translate shared actions into native remote events and normalize observations without hiding genuine platform differences.
4. **Server test controller.** Creates and resets isolated, deterministic test runs through a restricted non-production control plane. It returns a run ID, short-lived credentials, expected semantic state, and sanitized server observations.
5. **Evidence reporter.** Produces replayable traces, focus graphs, sanitized client and server telemetry, failure screenshots or recordings, environment metadata, and JUnit, JSON, and HTML reports.

Appium may be added as optional orchestration glue, but it is not the authoritative driver or assertion boundary. Platform-native frameworks are required for TV focus behavior and playback telemetry.

## Stable testability contract

Every interactive client control has a stable semantic accessibility identifier. Identifiers describe product identity rather than visible copy or position, for example:

```text
watch.movie.4242
watch.series.8080
detail.primary.play
playback.pause
playback.timeline
```

Stable accessibility identifiers remain in production because they also improve accessibility and external UI automation. Privileged state inspection is test-build-only.

Dedicated client test builds expose a sanitized diagnostics channel to their native test runner. It reports:

- current route and focused semantic identifier;
- requested content, episode, and file identifiers;
- playback state and timeline position;
- selected codec, resolution, dynamic range, frame rate, bitrate estimate, audio track, and captions;
- buffering and decoder failures;
- progress checkpoints submitted and acknowledged; and
- active presentation, theme, and layout identifiers.

The channel never reports passwords, access or refresh tokens, headers, complete media URLs, query strings, personal titles, or profile names. Production client binary inspection must prove that the diagnostics channel and its launch switches are absent.

## Deterministic run lifecycle

Each run follows a closed lifecycle:

1. The harness creates a cryptographically random run identifier and requests a named, versioned seed such as `watch_standard`, `restricted_profile`, or `campaign_during_playback`.
2. The server atomically resets only the dedicated test tenant and returns short-lived account credentials and expected semantic state. Credentials remain in memory and are redacted from all output.
3. The native driver installs or launches the designated test build and signs in through the normal client UI. Authentication remains part of the tested product path.
4. Authored scenarios execute critical journeys. The focus explorer then crawls all reachable routes for the selected presentation and accessibility configurations.
5. Observations from the client diagnostics channel, accessibility tree, native playback framework, and server are correlated by run identifier.
6. On failure, the harness freezes evidence before cleanup and emits the shortest known replay sequence.
7. The server expires credentials and destroys or resets the run. Failed cleanup marks the environment contaminated and prevents reuse until health checks pass.

The harness never receives database access. Setup and assertions use the restricted control plane; all client behavior uses ordinary public authentication, authorization, catalog, playback, and progress APIs.

## Server test-control plane

The test controller is a separately built non-production component, not a hidden route in the normal server binary. Production builds must not contain its route names, seed package, control credential types, or handler registration.

The narrow control API supports only:

- creating a run from an approved versioned seed;
- querying expected state and sanitized observed events;
- applying named, bounded fault conditions;
- switching approved presentation and layout variants;
- resetting or destroying a run; and
- checking environment health and contamination.

It does not support arbitrary SQL, shell execution, filesystem paths, unrestricted JSON mutation, production tenant identifiers, or arbitrary media URLs. Seeds are reviewed declarative documents, not imperative scripts.

Security requirements are:

- unavailable in production builds, including to administrators;
- bound only to loopback or an explicitly approved private test network;
- mutually authenticated runner identity;
- short-lived, single-run credentials;
- dedicated test-tenant namespace;
- exclusive concurrency lease per mutable tenant;
- strict time, storage, bandwidth, and request limits;
- complete sanitized audit events;
- automatic expiry and cleanup; and
- startup refusal against a production mode, production database identity, or unapproved tenant.

Static development credentials may be used only for an initial manual connectivity probe. Mature automated runs always use per-run accounts created by the test controller.

## Scenario language

Scenarios are versioned, declarative, and platform-neutral. They may invoke reusable flows, but each scenario retains explicit assertions at its behavioral boundary.

Example:

```yaml
schema: vondel_client_scenario_v1
scenario: resume_movie
seed: watch_standard
steps:
  - launch:
      profile: primary
  - focus:
      id: watch.movie.4242
  - press: select
  - assert:
      route: movie.4242
  - activate:
      id: detail.primary.play
  - assert_playback:
      content_id: "4242"
      codec: h264
      minimum_height: 720
      advancing_for_seconds: 10
  - press: back
  - relaunch: true
  - assert:
      focused_id: watch.resume.4242
```

Supported operations include:

- seed and reset server state;
- launch, terminate, reinstall, background, foreground, and reconnect a client;
- tap, repeat, or hold remote buttons;
- focus or activate a semantic identifier;
- wait for a route, focus target, playback state, diagnostics event, or server event;
- assert visible and accessibility state;
- assert allowed or denied navigation;
- assert playback identity and quality;
- apply named network, server, timeline, codec, and media faults;
- change presentation, layout, theme, language, text size, reduced-motion, and screen-reader configuration;
- compare persisted progress and other server effects; and
- invoke bounded focus exploration from the current route.

Coordinate input is prohibited except when an unavoidable operating-system surface exposes no semantic automation path. Any exception must be platform-scoped, documented, and excluded from product focus-graph evidence.

## Focus exploration

The explorer models an observable state as:

```text
route + focused_id + modal_state + playback_state + layout_variant
```

It uses breadth-first exploration to find the shortest route to a failure. From each state it exercises the applicable directional and activation actions, records the resulting state, deduplicates states, and enforces per-transition timeouts and per-scenario depth and action limits.

The explorer enforces these invariants:

- exactly one meaningful element has focus when the screen expects focus;
- every required control is reachable from the screen entry element;
- directional movement is deterministic for a stable state;
- a boundary press either remains intentionally stable or follows a declared edge;
- modal, notice, campaign, keyboard, picker, and error surfaces cannot trap focus after dismissal;
- returning from details or playback restores the originating control when it survives;
- when an origin disappears, focus moves to the declared nearest fallback;
- hidden, disabled, clipped, and off-screen controls cannot receive focus;
- loading, refresh, empty, offline, and recoverable-error states remain escapable;
- rapid and held input cannot strand focus or bypass authorization; and
- every reachable route has a working Back/Menu path unless it is the root.

Critical screens declare expected directional edges. Other screens are discovered and checked against invariants without requiring enormous pixel or graph golden files.

The explorer runs against server-defined layouts, themes, languages, text sizes, reduced-motion settings, screen readers, and overlay combinations. Pairwise configuration generation is the default; explicitly high-risk combinations receive exhaustive coverage.

## Playback verification

Playback assertions use independent evidence layers:

1. **Request identity:** sanitized server events identify the requested content, episode, file, profile scope, playback attempt, and advertised device capabilities.
2. **Plan identity:** client test diagnostics identify the sanitized validated plan and selected stream identifiers.
3. **Decoder reality:** AVPlayer or ExoPlayer observations identify the selected codec, rendition, resolution, frame rate, dynamic range, audio format, bitrate estimate, and active subtitle or audio selection.
4. **Time behavior:** the harness proves advancement during play, stability during pause, seek accuracy within declared tolerance, completion behavior, buffering recovery, and resume restoration.
5. **Media fingerprint:** deterministic synthetic video frames, audio windows, and subtitle cues identify the exact content, episode, rendition, frame counter, channel layout, and quality tier independently of UI labels.
6. **Server effects:** progress, completion, session termination, authorization, and fallback decisions match the expected run state.

Quality requirements are capability-aware. A scenario may require the highest compatible rendition and must record why a lower rendition was selected. Physical-device evidence includes output mode and reported decoder capability.

Fault coverage includes expired plans, authorization loss, malformed timelines, unsupported codecs, network interruption, slow startup, stalls, corrupt media, subtitle failure, and valid fallback selection. Reports distinguish server-plan, transport, decoder, and client state-machine failures.

Synthetic fixtures are rights-clear and generated deterministically. No copyrighted reference media is required.

## Evidence, privacy, and reproducibility

Every failure report contains:

- the minimal known action sequence;
- expected and observed semantic state;
- the focus graph with the failing edge highlighted;
- screenshots and, where permitted, a short recording around failure;
- sanitized playback and network chronology;
- correlated sanitized server events;
- client and server commits, build identifiers, OS, device, seed, layout, scenario, and harness versions; and
- a single replay command.

Credentials, tokens, headers, complete URLs, account names, personal catalog data, and profile names are redacted before artifacts are written. Synthetic titles may appear. Evidence retention is configurable. Security-sensitive runs may suppress images while retaining structural diagnostics.

A scenario must be deterministic before entering required CI. A timing-sensitive or device-specific scenario is quarantined with an owner, reason, and expiration date; it is not silently retried until it passes.

## Execution matrix

- **Pull requests:** focused authored scenarios on one Apple TV simulator and one Android TV emulator.
- **Main branch:** full authored suite, selected layouts, themes, accessibility settings, fault injection, and focus exploration.
- **Scheduled device laboratory:** physical Apple TV, Android TV, and Fire TV models with device health checks and decoder/output evidence.
- **Release candidate:** full clean-server run, high-risk configuration matrix, and server/client binary inspection proving test-control code and credentials are absent.

## Delivery phases

### Phase 1: Harness foundation

Create the separate repository, scenario schema, native driver contracts, semantic-ID rules, evidence formats, and simulator/emulator execution. Prove one shared Watch movie journey and one generated focus graph.

### Phase 2: Deterministic server integration

Add the non-production test controller, versioned seeds, per-run credentials, atomic reset, named fault injection, and server-event correlation. Remove static credentials from automated runs.

### Phase 3: Playback and exploratory validation

Add synthetic media fingerprints, AVPlayer and ExoPlayer telemetry, capability-aware quality assertions, full focus crawling, presentation/accessibility matrices, and shortest-path replay.

### Phase 4: Physical-device laboratory

Add Apple TV, Android TV, and Fire TV workers, device reservation and health checks, scheduled runs, output/decoder evidence, and release-candidate gates.

## Acceptance criteria

The first complete vertical slice must prove:

- one shared scenario runs unchanged on Apple TV and Android TV;
- all navigation uses real remote events and no client coordinates;
- the crawler detects a deliberately seeded focus trap and emits its shortest replay;
- playback independently proves exact synthetic content and rendition identity plus an advancing timeline;
- progress survives relaunch only under the correct exact account and profile scope;
- a failed run produces a one-command deterministic replay with sanitized evidence;
- contaminated or incompletely reset test environments are quarantined; and
- production client and server binaries contain no test-control routes, privileged diagnostics channel, fixture credentials, or test launch switches.

Long-term coverage is maintained through a registry. Every declared client capability maps to an authored scenario, focus invariant, contract test, or explicit unsupported/manual classification with an owner. CI rejects newly declared capabilities that have no test coverage classification.

## Non-goals

- Pixel-coordinate UI automation as a normal client control mechanism.
- Visual similarity testing against any reference client.
- Shipping fixture tokens, synthetic media, privileged diagnostics, or test-control routes in production.
- Allowing the harness direct database, shell, or filesystem access.
- Testing or introducing Live TV, IPTV, DVR, EPG, `.strm`, or arbitrary remote-stream shortcuts.
- Treating screenshots alone as proof of playback identity or quality.
