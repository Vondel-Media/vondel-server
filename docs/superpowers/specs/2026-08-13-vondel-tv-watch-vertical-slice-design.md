# Vondel TV Watch Vertical Slice Design

**Date:** 2026-08-13
**Status:** Approved design
**Scope:** `vondel-client-contracts`, `vondel-apple`, and `vondel-android`

## Objective

Deliver the first complete client-only Watch vertical slice on Apple TV and Android TV without depending on a live Vondel Server. A user can browse an invented movie-and-series catalog, open movie or series details, choose a movie or episode, play deterministic test media with native platform playback, leave, and resume from an exact-scope local checkpoint.

The slice establishes production-quality client boundaries around catalog presentation, playback-plan validation, native player state, progress, focus, accessibility, and error recovery. A later server adapter can replace fixture transport without changing the player or presentation architecture.

This design extends the approved clean-room client and server-directed presentation designs. It does not authorize changes to the active Vondel Server implementation.

## Binding boundaries

- Client code may use only the Vondel contracts repository, invented fixtures, public platform documentation, and the approved Vondel designs.
- Apple and Android share protocol semantics and fixtures, not application or UI code.
- The native clients remain authoritative for focus, accessibility, safe areas, playback controls, and local recovery.
- A generated synthetic clip is development and test input. It is not production media and is not shipped as user-facing sample content.
- A future rights-cleared server video is integration data. Substituting its playback-plan URL must require no player architecture change.
- Live TV, IPTV, DVR, EPG, `.strm`, and arbitrary remote-stream shortcuts are out of scope and are not latent capabilities of this slice.

## Milestone scope

### Included

- tvOS and Android TV Watch home screens;
- movie and series browsing from invented contract fixtures;
- movie details and series season/episode selection;
- playback-plan request and strict protocol-v3 validation;
- native H.264/AAC progressive playback;
- play, pause, ten-second seek, timeline scrubbing, exit, and resume;
- exact-scope local progress and completion;
- deterministic focus restoration;
- native accessibility and protected caption/control/notice occupancy;
- disposable fixture HTTP service and deterministic synthetic media generation;
- interfaces for later catalog transport and server progress synchronization;
- clean-room provenance and independent review evidence.

### Excluded

- production Vondel or Silo transport integration;
- autoplay-next;
- trailers, downloads, casting, DRM, picture-in-picture, and skip-intro;
- full alternate-audio and subtitle selection;
- Watch UI on iPhone, iPad, macOS, Android phone, or Android tablet;
- Listen, Read, cross-media Now, and Live TV functionality;
- server administration or presentation publication work.

The non-TV targets continue to compile against any shared model additions, but this milestone adds no Watch UI to them.

## Contract-led architecture

The contracts repository is the sole cross-platform semantic authority for this slice. It contains wholly invented Watch, playback, and progress fixtures plus a disposable HTTP fixture service. The service exposes deterministic forms of the already documented catalog, playback-start, media-stream, and progress boundaries. It is test infrastructure, not a new production server endpoint.

The existing catalog response remains the transport-level source. A client-side composer maps movies, series, and exact-scope progress into the semantic Watch home document consumed by native presentation. Movie and series detail fixtures provide deterministic metadata needed by this slice. Series fixtures add seasons and episodes with stable content and file identifiers.

The fixture service serves a generated MP4 through the playback-plan stream URL and implements the HTTP behavior required by native players, including `HEAD`, byte ranges, `Content-Length`, `Content-Type`, validators, satisfiable `206` responses, and correct `416` handling. The playback plan continues to declare `original_http`, `http_progressive`, MP4, H.264 video, and AAC audio.

Production clients do not contain fixture credentials, fixture server discovery, synthetic media, or a hidden demo mode. Test composition supplies fixture sources through the same interfaces production transport will implement later.

## Independent client ports

Each platform defines native equivalents of these roles:

- **Watch catalog source:** loads catalog/detail documents for an activation.
- **Watch home composer:** combines catalog entries and progress into semantic Stage & Chapters sections.
- **Playback plan source:** requests a plan for a content/file/profile selection.
- **Playback plan validator:** accepts only supported, internally consistent, unexpired protocol-v3 plans.
- **Native playback session:** exposes loading, ready, playing, paused, seeking, buffering, completed, and failed state plus position, duration, and tracks.
- **Watch progress store:** persists exact-scope checkpoints locally.
- **Progress sync sink:** accepts idempotent checkpoint batches when production transport becomes available.

Apple implements playback with AVFoundation/AVKit and platform-native focus/remote APIs. Android implements playback with Media3 and Compose for TV focus/remote APIs. Neither platform mirrors the other's runtime structure.

## Scope and activation safety

Catalog state, details, playback work, and progress are keyed by canonical server, organization, account, and profile. Each activation has a unique generation or lease.

A server, organization, account, or profile switch:

1. invalidates the previous activation;
2. cancels catalog and detail work;
3. closes active playback;
4. closes the old progress store before opening the new store;
5. rejects late catalog, playback, and checkpoint callbacks.

Returning later to the same identity creates a new activation. Results from the previous activation remain stale even though the identity fields match.

## Watch home: Stage & Chapters

The bundled TV default is **Stage & Chapters**.

- One editorial stage leads the screen.
- Continue Watching supplies the stage item when a valid incomplete checkpoint exists.
- Otherwise the first fixture item explicitly marked featured supplies the stage.
- Horizontal chapters follow in this order: Continue Watching, Films, Series.
- Empty chapters are omitted.
- Watch is unavailable only when capability or policy says the experience is unavailable; an empty library uses an original client-owned empty state.
- Server presentation may select only closed semantic variants. It cannot provide arbitrary coordinates, view trees, or focus order.

Focus movement follows the rendered semantic order. Returning from details restores the originating card when it still exists; otherwise focus moves to the nearest surviving item, then the chapter heading, then the stage action.

## Movie details: Backdrop Ledger

The bundled movie-detail default is **Backdrop Ledger**. One cinematic field contains title, year, runtime, rating when present, concise overview, progress, and primary actions.

- `Play` begins at zero when no valid incomplete checkpoint exists.
- `Resume` begins at the validated checkpoint position.
- Completed items show `Play Again` and begin at zero.
- Save and trailer controls appear only when their corresponding client behavior exists; this milestone does not add them.
- The primary playback action receives initial focus when entering from a Watch card.

## Series details: Season Stage

The bundled series-detail default is **Season Stage**.

- The stage identifies the series and current story.
- A valid incomplete episode checkpoint makes `Resume` the primary action.
- Otherwise the primary action targets the first unwatched episode in deterministic season/episode order.
- Season selection and episode cards are one directional focus move below the stage.
- Completing an episode advances the suggested next episode but does not start it automatically.
- A missing or empty episode list is a malformed detail document, not an empty series.

Returning from playback restores the action for the played episode. Returning to Watch restores the original series card.

## Playback: Quiet Timeline

The shared playback information hierarchy is **Quiet Timeline**. Native platform components and engines implement it rather than sharing a custom player runtime.

The visible control state contains:

- content or episode title;
- elapsed and remaining time;
- seekable timeline and current position;
- one dominant play/pause action;
- ten-second rewind and forward actions;
- a caption action only when the native player reports a usable preferred track; in this milestone it toggles that track on or off;
- an audio action only when the platform supplies a native system track picker without custom client selection UI; otherwise it remains hidden.

Controls auto-hide only during stable playback. They remain visible while paused, seeking, buffering, or showing a recoverable error. The first Back/Menu action dismisses visible controls; when controls are already hidden, Back/Menu exits playback after saving a checkpoint.

Timeline seeks are clamped to the validated seekable range. Remote repeat input is bounded and cannot create unbounded queued seeks. Player callbacks are normalized onto the client's state authority before they mutate UI or progress.

The upper notice zone, caption area, and active control area are reported to the existing overlay occupancy policy. Critical notices may use the upper protected zone. Campaigns cannot cover captions, active controls, or system playback UI.

## Playback-plan validation

The client rejects a plan before creating the native player when any required condition fails, including:

- unsupported protocol version, delivery, stream protocol, container, video codec, or audio codec;
- absent, malformed, non-HTTP(S), expired, or cross-origin-disallowed stream information;
- inconsistent requested and effective media identifiers;
- invalid duration, start position, timeline offset, or seek declaration;
- server-required headers that the platform path cannot safely apply;
- an `adaptation_unavailable` outcome or a playable outcome without a complete plan.

Relative fixture URLs resolve against the validated fixture origin. Production URL and redirect rules remain governed by the documented server-origin contract. There is no fallback from a rejected plan to a catalog-provided or user-provided media URL.

## Progress and completion

The local progress checkpoint contains at least exact scope, content ID, media file or episode identity where applicable, position, duration, completion, update timestamp, and activation generation.

The client attempts a checkpoint:

- on a bounded periodic cadence during playback;
- after a completed seek;
- on pause;
- when the application backgrounds or loses its playback scene;
- on recoverable and terminal player errors;
- before player exit;
- on completion.

Writes are monotonic by logical update time and activation. A late callback cannot overwrite a newer checkpoint or resurrect a completed entry.

For content longer than two minutes, an item is completed when playback reaches either 90 percent of known duration or the final two minutes, whichever threshold occurs first. Content of two minutes or less uses the 90-percent threshold so it cannot be completed at time zero. Completion clears the resume position and marks the item watched. Seeking past the threshold is not sufficient by itself: the client records completion only after native playback reports advancing at or beyond the threshold. `Play Again` starts at zero without erasing the completed state until new playback progress is established.

The progress sync sink accepts idempotent payloads matching the existing progress-sync contract. In this milestone it is exercised by fixtures and fakes; production network scheduling is deferred.

## Screen and player states

Watch and detail surfaces have four explicit states:

- loading;
- content;
- recoverable failure;
- unavailable.

Cached content may remain visible during refresh failure. The UI labels staleness only when the client can determine that the displayed snapshot is materially old. No state permits an endless spinner.

Playback has loading, ready, playing, paused, seeking, buffering, completed, and failed states. State transitions are serialized by one client-owned authority per playback session.

## Failure and recovery

- **Invalid or unsupported plan:** return to details and explain that this client version cannot play the item.
- **Network interruption:** preserve the latest safe position and offer Retry.
- **Decode failure:** preserve position, offer Back, and record only safe codec and delivery categories.
- **Authentication or session expiry:** stop playback, preserve progress, and invoke the existing token-refresh boundary.
- **Authorization failure:** stop playback and return to a safe unavailable state; never retry as another profile.
- **Fixture service unavailable:** fail the integration test deterministically; never activate fixture behavior in production.

Retry creates a new playback attempt identifier and requests a new plan. It does not reuse an expired stream URL or stale authorization header.

## Privacy and diagnostics

Default logs exclude access and refresh tokens, complete media URLs, URL queries, titles, viewing history, profile identifiers, fixture credentials, and server-provided headers. Diagnostics may record redacted origin identity, delivery class, broad codec category, normalized player state, safe error class, and timing.

Any diagnostic export remains local-first and requires the existing explicit user-consent path.

## Accessibility

- Every actionable TV element has a stable native accessibility label, role, and focus state.
- Focus order matches the semantic reading order and does not depend on artwork geometry.
- Reduced-motion and reduced-transparency settings strengthen or simplify presentation.
- Text and focus treatments use validated semantic colors and retain required contrast.
- Playback controls do not auto-hide while accessibility focus is within them.
- Caption occupancy is authoritative over promotional placement.
- Time values have readable spoken forms rather than raw numeric labels.

## Test media

The contracts repository supplies a script that deterministically generates a short MP4 from synthetic color fields, geometric motion, generated text, and generated tone. The script records its tool requirements and exact command. No third-party footage, artwork, music, logo, or font is incorporated.

Generated binary output is disposable build/test output unless a later review explicitly approves a small fixture artifact. Tests validate the media's container, codecs, duration, seekability, byte-range behavior, and reproducible content hash for a pinned toolchain.

## Verification strategy

### Contracts repository

- schema and fixture validation;
- invented catalog/movie/series/season/episode consistency;
- playback-plan success and malformed-plan cases;
- fixture HTTP authentication boundary and byte-range behavior;
- deterministic media generation and inspection;
- progress completion and idempotency fixtures;
- disposable service lifecycle and failure tests;
- originality and provenance guard.

### Apple client

- decoding and plan-validation unit tests;
- Stage & Chapters composition and focus tests;
- Backdrop Ledger and Season Stage rendering tests;
- AVFoundation session state and generated-clip integration tests;
- exact-scope progress, stale callback, and completion tests;
- tvOS UI flow: Watch to details to playback to resume;
- accessibility and reduced-motion checks;
- unsigned tvOS build and clean-room verification.

### Android client

- decoding and plan-validation unit tests;
- Stage & Chapters composition and focus tests;
- Backdrop Ledger and Season Stage Compose tests;
- Media3 session state and generated-clip integration tests;
- exact-scope progress, stale callback, and completion tests;
- Android TV UI flow: Watch to details to playback to resume;
- accessibility, semantics, and animator-scale checks;
- TV APK and AndroidTest APK builds plus clean-room verification.

Connected-device execution is required before claiming hardware acceptance. Simulator, emulator, compiled instrumentation, and packaged APK evidence must be reported separately when an uncontested device is unavailable.

## Delivery order

1. Extend and verify the contracts fixtures and disposable fixture service.
2. Implement independent Apple semantic models, Watch composition, progress, and playback session.
3. Implement independent Android semantic models, Watch composition, progress, and playback session.
4. Add the approved TV views and focus behavior on both platforms.
5. Run cross-repository conformance, platform, accessibility, and clean-room reviews.
6. Preserve server transport and production progress sync as explicit later integration work.

Apple and Android implementation may proceed concurrently only after the contracts slice is committed and its conformance fixtures are stable.

## Acceptance criteria

- Both TV clients browse the same invented movie-and-series semantics through independently written native code.
- Movie and episode playback succeeds against the disposable fixture service using native players.
- Play, pause, ten-second seek, scrubbing, exit, and relaunch resume behave deterministically.
- Completion uses the approved 90-percent-or-final-two-minutes rule and cannot be triggered by seek alone.
- Scope switching closes playback and prevents stale catalog or progress publication.
- Focus returns predictably across Watch, detail, and playback transitions.
- Captions, active controls, and critical notices have authoritative protected occupancy.
- Unsupported or malformed plans fail closed without an arbitrary-URL fallback.
- Production artifacts contain no fixture service, fixture credential, generated test media, or hidden demo mode.
- tvOS and Android TV builds, unit/integration tests, accessibility checks, and clean-room audits pass.
- Hardware playback and remote-control behavior are claimed only after connected-device execution.
