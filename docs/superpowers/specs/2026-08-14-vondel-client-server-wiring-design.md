# Vondel Client Server Wiring Design

**Date:** 2026-08-14
**Status:** Approved design
**Scope:** `vondel-server`, `vondel-apple`, and `vondel-android`

## Objective

Connect the completed TV Watch vertical slices to a real Vondel Server. A viewer powers on an
Apple TV or Android TV, finds their server on the local network or types its address, signs in
either on the television or by approving the device from a phone or browser they are already
signed in on, picks a household profile, browses their own library, plays their own media through
a server-owned playback plan, and finds the same resume point on every device.

The slice's client boundaries do not move. Fixture transport is replaced by production transport
behind the interfaces the Watch slice already defined: the catalog source, the playback-plan
source, and the progress sync sink. What changes is what stands behind them, plus the onboarding
and identity surfaces that had no server to talk to before.

This design extends the approved TV Watch vertical slice, the clean-room client design, and the
profile-login and shared-devices design. It authorizes additive `/api/v1` work on the server.

## Binding boundaries

- Client code may use only the Vondel contracts repository, invented fixtures, public platform
  documentation, and the approved Vondel designs. Neither client repository reads the other, and
  neither reads any reference client.
- The contracts repository remains the cross-platform semantic authority. Where the server and the
  contracts disagree, the disagreement is named and closed deliberately — never absorbed silently
  by a client.
- `/api/v1` work is additive. New behavior adds endpoints and fields; feature detection goes
  through capability endpoints, never version sniffing. The one correction taken against an
  existing response in this design is recorded in the pre-lock removals table before it ships.
- The client cleartext policy does not move. HTTPS everywhere; plain HTTP only against loopback.
- Live TV, IPTV, DVR, EPG, XMLTV, `.strm`, and arbitrary remote-stream shortcuts stay out of scope
  and are not latent capabilities of anything designed here.
- No document, test, fixture, or commit in any of the three repositories names a real deployment
  origin, hostname, account, password, token, or pairing code.

## Milestone scope

### Included

- server identity and aggregate capability advertisement;
- local-network server discovery by mDNS, with an operator switch, plus manual origin entry;
- first-run onboarding on both televisions;
- on-screen credential sign-in through the platform keyboard;
- device-code pairing sign-in with approval from an already signed-in session;
- household profile selection and PIN verification;
- exact-scope activation derived from real server identity;
- Watch home and detail documents served by `/api/v1`;
- playback against real protocol-v3 plans, including plan expiry and header refresh;
- two-way progress synchronization with a newest-updated-at merge;
- error taxonomy mapping onto the clients' five safe failure categories;
- security invariants for token storage, redaction, and transport.

### Excluded

- autoplay-next, trailers, downloads, casting, DRM, picture-in-picture, and skip-intro;
- alternate-audio and full subtitle selection beyond the slice's existing preferred-track toggle;
- Watch UI on iPhone, iPad, macOS, Android phone, or Android tablet;
- Listen, Read, cross-media Now, and Live TV;
- remote playback handoff from a paired phone to a paired television;
- multi-organization selection on `/api/v1`;
- session-level playback progress reporting distinct from durable progress;
- server administration UI beyond the pairing approval surface and the discovery setting.

## What the server already has

The wiring is smaller than it looks because four of the required surfaces exist today.

- **Credential sign-in.** `POST /api/v1/auth/login`, `POST /api/v1/auth/refresh`,
  `GET /api/v1/auth/me`, and `GET /api/v1/profiles/` serve the shapes the contracts declare.
  Login and setup are rate limited.
- **Device-code pairing.** `POST /api/v1/auth/device/start`, `GET /api/v1/auth/device`,
  `POST /api/v1/auth/device/poll`, and the authenticated `POST /api/v1/auth/device/approve` and
  `/deny` implement the whole flow, including a match code, a purpose, an expiry, a poll interval,
  and a distinct remote-playback-handoff approval that this milestone does not use. A browser
  approval page already answers at `/activate` in the web frontend.
- **Playback.** `GET /api/v1/playback/capability` and `POST /api/v1/playback/start` speak
  protocol v3 and emit plans matching the contracts' plan schema, including `expires_at`,
  `stream.headers`, and `stream.header_refresh`. `POST /api/v1/playback/{session_id}/replan` and
  `DELETE /api/v1/playback/{session_id}` complete the session lifecycle.
- **Progress.** `GET /api/v1/progress` and `POST /api/v1/sync/progress` accept and return the
  contract shapes, with a server-side newest-event merge when the client supplies `updated_at`.

What the server does not have is server identity fit for scope keying, an aggregate capability
endpoint, a Watch document, and any local-network presence.

## Server identity

Scope keying needs a server identifier that is stable across restarts, stable across the several
URLs one deployment answers on, and never empty. Today `GET /api/v1/health` carries `server_name`
and `server_id`, but both are sourced from the Jellyfin-compatibility configuration and are omitted
when that configuration is absent. Health keeps exactly that behavior; nothing that reads it today
changes.

A new public endpoint answers the question properly:

`GET /api/v1/server/identity` returns a stable `server_id`, an operator-facing `server_name`, the
API major versions the deployment serves, and whether the setup flow has been completed. The
identifier is a random UUID generated once, stored in a plain (not encrypted) server setting, and
never regenerated. Restoring a backup carries the identifier with it, which is the correct
behavior: it is the same server.

`GET /api/v1/capabilities` is public and returns the aggregate capability set the clients already
decode: `schema_version`, `media_types`, and `features`. This milestone adds the feature tokens
`watch_document_v1`, `device_pairing_v1`, and `progress_sync_v1` alongside the existing playback
and events tokens. Both clients maintain an allowlist of known feature tokens and ignore the rest,
so recognizing a new token is a deliberate client change rather than an accident.

Clients that reach an older server find neither endpoint. The fallback is defined and narrow:
`server_id` falls back to the normalized origin string, and an absent capability document means no
optional feature is available. Watch is then unavailable with the client's own explanation, not a
blank screen.

## Discovery

### What the server broadcasts

The server registers one mDNS/DNS-SD service of type `_vondel._tcp` in the `local.` domain. The
instance name is the operator-facing server name. The port is the port the API answers on.

The TXT record carries only what a client needs to build a candidate and tell two servers apart:

- `txtvers=1` — the TXT record layout version, so the record can grow.
- `id=<server_id>` — the stable server identifier, for deduplicating one server seen on several
  interfaces and for recognizing a server whose address changed.
- `name=<server_name>` — the same operator-facing name as the instance name, for clients that do
  not surface instance names.
- `api=1` — the API major versions served, as a comma-separated list.
- `scheme=https` or `scheme=http` — the scheme the advertised port speaks.

The record deliberately omits the software version and build, the library counts, the media types
present, the account or profile names, the tenant identity, and anything about setup state. A LAN
broadcast is unauthenticated and unencrypted; a precise build string turns discovery into a
targeting aid, and household names are nobody else's business. API major version is enough for a
client to decide whether to bother connecting; the exact build is discoverable after
authentication and not before.

Three rules bound the broadcast:

1. **It is an operator setting.** A server setting governs it, it defaults to enabled, and the
   responder starts and stops when the setting changes without a restart. Disabling it removes the
   service registration; the server keeps serving normally.
2. **It never advertises before the server is claimed.** A deployment that has not completed setup
   does not broadcast. An unclaimed server announcing itself on a shared network invites a stranger
   to claim it first.
3. **It never advertises on interfaces that are not private or link-local.** Multicast does not
   route past the local segment anyway, but the interface filter is explicit rather than assumed.

### What the clients do with it

Each client browses for `_vondel._tcp` for a bounded window, resolves each instance, and presents
candidates deduplicated by `id`, sorted by name. A candidate carries a display name and a proposed
origin. It carries no trust.

Nothing discovered is used until it is confirmed over the transport the client would use anyway:
the client builds an origin through the platform's existing origin validator, fetches
`GET /api/v1/server/identity`, and requires the returned `server_id` to equal the advertised `id`.
A mismatch discards the candidate. Every subsequent request uses the validated origin, never the
advertised address.

Manual origin entry is present on the same screen at all times, is reachable by one directional
focus move, and is validated by the same origin rule. Discovery is a convenience; typing is the
contract.

A discovered candidate advertising `scheme=http` on a non-loopback host is shown, and shown as not
connectable, with the reason stated plainly. It is never silently promoted to HTTPS and never
opened over cleartext. The alternative — a debug allowance for private address ranges — is
rejected: it would be a permanent hole in the one rule that keeps a television from streaming a
household's library in the clear across a hotel network.

### The development posture

A developer's server is usually plain HTTP. The supported way to reach it from a television is a
loopback forward, which both platforms already permit because loopback is the one cleartext
exception: an Android TV run uses a reverse socket forward so the device reaches the server at a
loopback address, and an Apple TV simulator run reaches a loopback address directly. A physical
Apple TV requires a TLS-terminating reverse proxy in front of the development server. No release
configuration, transport-security exception, or network-security-config domain is added for a
development host on either platform.

## Onboarding

Both televisions run the same five-step flow. Each step is one screen, each screen has one primary
action, and the primary action holds initial focus.

1. **Find your server.** Discovery runs immediately and fills a list. Manual entry sits below it.
   Selecting a candidate or submitting an origin validates it and probes identity and capabilities.
   A server that answers is named on screen before anything is typed into it.
2. **Choose how to sign in.** Credential entry is the primary action. Pairing is the secondary
   action, and it is offered only when the server advertises `device_pairing_v1` and the device
   capability endpoint answers.
3. **Sign in.** Either the credential screen (platform keyboard, obscured password field, no
   password ever rendered in a label, no autofill of a previous account's password) or the pairing
   screen (see below).
4. **Choose a profile.** `GET /api/v1/profiles/` lists the household. A profile with `has_pin`
   requires PIN entry, which yields a profile token that authorizes profile-scoped requests.
5. **Watch.** The activation opens and the Watch home renders.

Back moves one step without discarding the previous step's input. A server that becomes
unreachable mid-flow returns to step one with the origin preserved. The flow never dead-ends and
never spins without an explanation.

Both clients persist the last successful origin and the last profile so a relaunch resumes at step
five when the stored tokens are still good, at step three when they are not, and at step one when
the origin no longer answers.

## Authentication

### Credentials on screen

`POST /api/v1/auth/login` with a username and password, returning an access token, a refresh
token, an expiry in seconds, and the account. This ships first because it depends on nothing new
and because a household with one television and no phone must still be able to sign in.

The password is read through the platform keyboard, held only for the duration of the request,
and never written to a log, a diagnostic, a crash report, or a UI element. The login endpoint is
rate limited server-side; the client honors a rate-limited response as a recoverable network
condition with a bounded backoff rather than as a credential failure.

### Device-code pairing

The pairing flow is already implemented server-side; this design specifies how the televisions use
it and what the approval surface must guarantee.

1. The television calls `POST /api/v1/auth/device/start` with its device name and platform and a
   client purpose identifying a television login. It receives a device code it keeps secret, a
   short user code it displays, a match code, a verification URI, a complete verification URI, an
   expiry, and a poll interval.
2. The television displays the user code, the verification URI, and the match code. Displaying the
   complete verification URI as a scannable code is permitted. The device code is never displayed,
   never logged, and never placed in a diagnostic.
3. The viewer opens the verification URI on a phone or computer where they are already signed in,
   or signs in there, and is shown the requesting device's name, platform, an IP hint, and the
   match code before approving. The match code is what makes "approve this device" mean "approve
   *my* device" — the approval surface must show it, and the television must show the same value.
4. The television polls `POST /api/v1/auth/device/poll` at the server-supplied interval, never
   faster, and stops at the server-supplied expiry. Pending, denied, expired, and consumed are
   distinct outcomes with distinct copy. Approval returns the same token pair and account a login
   returns, after which the flow continues at profile selection.

The approval surface exists in the web frontend today at `/activate` and accepts a code typed by
hand as well as a token carried in the URL. This design requires that it also be reachable from
the signed-in navigation rather than only by URL, so a viewer who has the television in front of
them and no link to follow can still find it. Its placement in the signed-in UI is an open
question below.

Pairing does not weaken the account boundary: approval requires an authenticated session, and the
server records which identity approved. A television paired by one account never obtains another
account's library.

### Token lifecycle

Both clients hold exactly one token pair per activation. Storage is the platform secure store
already in place — keychain on Apple, keystore-backed storage on Android — keyed by a digest of
the exact account scope, through the existing credential vault. Tokens never touch preferences,
files, logs, or diagnostics.

Refresh is proactive and single-flight. A client refreshes when the access token is inside the
last fifth of its advertised lifetime, or immediately on a 401, whichever comes first, through the
existing per-lease refresh coordinator so a burst of concurrent requests produces one refresh. The
refresh endpoint rotates both tokens; the vault write replaces both atomically, and a failed write
invalidates the in-memory pair rather than leaving a half-rotated state.

A refresh that fails with 401 ends the session: playback stops, a final checkpoint is written
locally, the vault entry for that scope is removed, and onboarding resumes at the sign-in step
with the origin preserved. Local progress for that scope survives; it re-synchronizes on the next
successful sign-in. An explicit "forget this server" action removes the vault entry and the scoped
progress store together.

## Scope and lease derivation

The clients key every piece of state by an exact account scope and a per-activation lease
generation. Against a real server the scope components resolve as follows:

- **Server** — `server_id` from `GET /api/v1/server/identity`, falling back to the normalized
  origin string when the endpoint is absent. Two different origins that report the same
  `server_id` are the same server and share stored state; two origins with different identifiers
  never do.
- **Organization** — the legacy default. `/api/v1` is the Silo-compatible projection with an
  implicit tenant, and clients must not invent an organization identifier or send an organization
  header on it. The identified organization form of the scope is reserved for a future native API
  and is not used by this milestone.
- **Account** — the account identifier from the login or refresh response, rendered as a string.
- **Profile** — the selected profile's identifier.

A new lease generation is created on sign-in, on profile switch, on server switch, on sign-out and
back in, and whenever a refresh returns a different account identity. A token refresh that
preserves identity does not create one — the activation is the same activation with fresher
credentials.

Switching any scope component follows the discipline the slice already implements: invalidate the
previous activation, cancel catalog and detail work, close active playback with a final
checkpoint, close the old progress store before opening the new one, and reject every late
callback carrying the old lease. Returning later to the same identity produces a new generation,
and results from the previous one remain stale even though the identity fields match.

## Watch documents over the API

The server gains two profile-scoped endpoints that return the contracts' `watch_document_v1`
shape byte-for-byte:

- `GET /api/v1/watch/home` — the Watch home document: a snapshot timestamp, a featured content
  identifier, the items the profile may see, and the profile's progress rows for those items.
- `GET /api/v1/watch/items/{content_id}` — the detail document for one movie or one series,
  carrying the series' seasons and its episodes in deterministic season and episode order, each
  episode with its playable file identifier.

Both require the profile header and the profile's viewer access, and both honor library
restrictions: a document never names an item the profile may not see.

Serving the composition rather than making the clients join it is a deliberate choice. The
alternative — clients calling catalog, item detail, episodes, and progress and joining the results
— costs three or four round trips per screen on hardware where the first frame of the home screen
is the whole first impression, duplicates a non-trivial join in two languages, and leaves the
editorial featured choice to whichever client implemented it last. The document also makes the
disposable fixture service and the real server interchangeable behind one client interface, which
is what keeps the fixture-backed tests honest.

The featured item is a closed server-owned choice, not a coordinate system: the server names one
content identifier, and the client's own Stage & Chapters rules decide what to do with it. Server
presentation still cannot supply layout, view trees, or focus order.

Clients detect the endpoints through `watch_document_v1` in the capability document. A server
without it renders Watch as unavailable with the client's own copy.

## Playback with real plans

Playback goes through `POST /api/v1/playback/start` with protocol version 3 and the file
identifier taken from the Watch document. The request declares:

- a fresh playback attempt identifier per attempt, including per retry;
- the profile identifier and profile header;
- the subtitle fidelity preference the slice already sends;
- client codec capabilities at the highest evidence tier the platform can honestly support —
  platform-attested where the platform exposes a decoder registry, declared as the floor, and
  never a tier the client cannot substantiate;
- a client playback context declaring the television form factor and the delivery classes the
  platform genuinely supports;
- an explicit start position: zero for `Play` and `Play Again`, the validated local checkpoint
  position for `Resume`.

The start position is sent explicitly on purpose. Omitting it asks the server to apply its own
resume policy, which would silently overrule the checkpoint the viewer just saw on the detail
screen. The client decides *where this press of this button starts*; the server remains the sole
authority on *how it is delivered*. Progress persistence stays at the server default because
durable progress is synchronized, not client-owned.

The returned plan passes through the existing validator before any engine sees it. Nothing about
the validator relaxes for a real server: unsupported delivery, protocol, container, or codecs;
malformed, non-HTTPS, or cross-origin stream information; inconsistent requested and effective
file identifiers; invalid timeline values; and unsatisfiable header requirements all fail closed,
and there is no fallback to a catalog-provided or user-provided URL.

### Expiry and header refresh

The slice validates `expires_at` once and then ignores it. That is not sufficient against a real
server, where a plan is minted with a long lifetime and a television may sit paused for hours.

- A plan is revalidated immediately before the engine loads it, not only when it arrives.
- A plan whose remaining lifetime is under a floor of sixty seconds is treated as expired and
  replaced before playback starts.
- A plan that expires during playback is not torn down mid-frame; the client requests a
  replacement, and only a failure to obtain one becomes a visible interruption.
- An authentication or authorization failure on the stream itself stops playback, writes a
  checkpoint, refreshes the session if that is the cause, and requests a new plan through the
  replan endpoint when a session identifier exists or a fresh start otherwise.
- `stream.header_refresh` of `session` obliges the client to re-fetch headers from the plan's
  refresh URL before reusing them. The current server always emits `none`, so this path is
  specified and tested but not exercised in production yet. A plan declaring `session` without a
  usable refresh URL is invalid and fails closed.

Exiting playback releases the server session so a transcode or remux session is not orphaned.
Retry always allocates a new attempt identifier and never reuses an expired stream URL or a stale
authorization header.

The production plan source performs none of the three substitutions the debug fixture source
performs: stream headers come from the plan, the player start comes from the negotiated start
position and the plan's timeline, and the source duration is the server's. The client's
duration-contradiction guard therefore runs against real numbers, which is the point.

## Progress synchronization

### Outbound

Checkpoints reach the server through `POST /api/v1/sync/progress` with the profile header. Each
item carries the media item identifier, the position, the duration, a force-overwrite flag that is
always false, and the checkpoint's own update timestamp in RFC 3339.

The media item identifier is the episode's content identifier for an episode and the movie's
content identifier for a movie — the same identity the Watch document uses. Sending `updated_at`
with force-overwrite false is what selects the server's newest-event merge, which is the same
newest-updated-at rule the clients apply locally. Force-overwrite is never sent true: a television
has no business overruling a newer write from another device.

The sink batches. Changed checkpoints accumulate, coalesce by key keeping the newest, and flush in
bounded batches on the coordinator's existing cadence and lifecycle events. Failures retry with
bounded exponential backoff and jitter. Nothing is dropped on failure: unsent checkpoints remain
in the local store, which is durable on both platforms, and are re-sent on the next flush or the
next activation. A sync failure is never surfaced as a playback error and never blocks the player.

### Inbound

The Watch home document carries the profile's progress rows. On activation, and on each refresh,
those rows merge into the local store per content and episode identity, newest update timestamp
winning. A local row that is newer is kept and queued for sending. A server row that is newer
replaces the local one, including its completed state.

Two identity gaps shape the merge. The server's progress row carries no media file identifier
while the local checkpoint key includes one; the file identifier comes from the Watch document's
entry for that item, and a server row for an item absent from the document is held as a resume
hint that cannot become a checkpoint until a file identifier is known. The server's progress row
also carries no series linkage, so an episode row is only attributable through the document that
named the episode.

### Where the two completion rules differ

The client completes an item at ninety percent of duration or the final two minutes, whichever
comes first. The server completes at ninety percent and discards any progress below five percent
of duration. For content longer than twenty minutes the two rules coincide exactly. Below twenty
minutes the client completes earlier, and below five percent the server keeps no resume point at
all, so a resume position in the opening minutes of a title is device-local by design.

The resolution is stated rather than papered over: the server's value is authoritative whenever
its update timestamp is newer, so a second device sees the server's view; the client's local rule
continues to govern the moment of completion during playback, which is what makes completion feel
immediate. Whether the client should instead read the server's thresholds is an open question
below.

## Failure taxonomy

Every server outcome maps onto one of the five safe categories the clients already render. The
mapping is exhaustive; an unmapped outcome is a bug, not a sixth category.

- **Network interruption** — transport failures, TLS failures, timeouts, cancellations, 5xx
  responses, and rate-limited responses. Retry is offered, backoff is bounded, and the latest safe
  position is preserved.
- **Authentication or session expiry** — 401 on any call, an unauthorized or invalid-credentials
  error body, and any refresh failure. Playback stops, progress is preserved, and the refresh
  boundary runs before anything is retried.
- **Authorization failure** — 403, forbidden, profile-required, profile-unverified, disabled
  account, and capability-unavailable. Playback stops and the surface returns to a safe
  unavailable state. It is never retried as another profile and never escalated.
- **Invalid or unsupported plan** — a rejected plan, an `adaptation_unavailable` outcome, a
  playable outcome without a complete plan, a malformed or schema-violating response, a 404 for a
  named item, and a 426 client-upgrade-required. The copy says this client version cannot play
  this item; there is no URL fallback.
- **Decode failure** — engine-reported decode and renderer errors. Position is preserved, Back is
  offered, and only broad codec and delivery categories are recorded.

Pairing has its own bounded outcomes — pending, denied, expired, and already-used — which are
onboarding states rather than playback failures and are rendered as such.

## Security invariants

- The origin rule is unchanged and is the only gate: HTTPS for every remote host, plain HTTP only
  against loopback, no embedded credentials, no path, no query, no fragment. No release
  configuration on either platform adds a transport exception for a private address range.
- Tokens, refresh tokens, profile tokens, device codes, user codes, and match codes never appear
  in logs, diagnostics, crash reports, analytics, accessibility labels, or screenshots. The
  existing redaction machinery on both platforms is extended to cover the new identifiers, and the
  extension is proven by test rather than asserted.
- Diagnostics may record a redacted origin identity, a delivery class, a broad codec category, a
  normalized player state, a safe error class, and timings. They may not record complete media
  URLs, query strings, titles, viewing history, account names, or profile identifiers.
- Credentials live only in the platform secure store, keyed by a digest of the exact scope. Sign-out
  removes the entry; forgetting a server removes the entry and the scoped progress store.
- No credential, origin, or code belonging to a real deployment is committed to any of the three
  repositories, in code, tests, fixtures, documentation, or commit messages. Test backends are the
  contracts repository's disposable fixture service and a development server supplied entirely
  through environment configuration at run time.
- The server's discovery broadcast is off before setup completes, is bounded to private and
  link-local interfaces, carries no build version and no household identity, and can be switched
  off entirely by the operator.

## Accessibility

The onboarding, pairing, and profile surfaces inherit the slice's discipline rather than inventing
their own. Every actionable element has a stable native accessibility label, role, and focus state.
Focus order matches semantic reading order. A user code is exposed as an accessible string with
character-by-character spoken form rather than as a word. Password and PIN fields are marked secure
so their contents are never spoken or displayed. Progress and error states are announced when they
change rather than only rendered. Reduced-motion and reduced-transparency settings simplify these
screens as they do the Watch screens.

## Verification strategy

### Server

- Endpoint tests for server identity, aggregate capabilities, and both Watch documents, asserted
  against the contracts' schemas rather than against hand-written expectations.
- Profile scoping and library-restriction tests proving a document never names an unpermitted item.
- Progress sync result vocabulary tests covering applied, ignored, and error rows.
- Discovery responder tests covering the setting, the setup gate, the interface filter, the TXT
  record contents, and start/stop without restart.
- Pairing tests covering purpose enforcement, match code presentation, approval identity, denial,
  expiry, and reuse.
- The existing client-contract conformance run, extended to cover the new endpoints.

### Both clients

- Discovery result mapping, deduplication, identity confirmation, and refusal of cleartext
  candidates.
- Origin validation for typed input, including rejection cases.
- Sign-in, refresh rotation, single-flight refresh, and session-end behavior with a fake transport.
- Pairing state machine including interval honoring, expiry, denial, and reuse.
- Scope and lease derivation, including switch discipline and late-callback rejection.
- Watch document decoding against the contract fixtures, unchanged from the slice.
- Plan expiry, near-expiry replacement, header refresh, and replan behavior.
- Progress batching, coalescing, backoff, offline persistence, and the inbound merge.
- Redaction proofs for every new identifier.
- An end-to-end journey against the disposable fixture service, and the same journey against a
  development server supplied by environment configuration.
- Release-artifact inspection proving no fixture source, fixture token, or development origin is
  present.

Connected-device execution is required before claiming hardware acceptance. Simulator, emulator,
and packaged-artifact evidence is reported separately when an uncontested device is unavailable.

## Delivery order

1. Server identity, aggregate capability, Watch documents, progress result vocabulary, discovery
   broadcast, and pairing surface completion.
2. Client discovery, origin validation, and identity probing.
3. Client credential sign-in, session lifecycle, profile selection, and scope derivation.
4. Client onboarding surfaces.
5. Production catalog and plan sources replacing fixture transport.
6. Progress synchronization.
7. Device-code pairing sign-in.
8. Cross-repository conformance, accessibility, hardware, and clean-room review.

Apple and Android work may proceed concurrently once the server slice is committed and its
conformance evidence is stable.

## Contract gaps

These are the places where the server as it stands today and the contracts the clients were built
against do not agree. Each is closed deliberately by this design.

1. **No Watch document endpoint.** The contracts define `watch_document_v1` and the fixture
   service serves it under a fixture path; the server serves nothing equivalent. Closed by adding
   the two `/api/v1/watch` endpoints. The contract's OpenAPI document does not yet describe them,
   so the schema is the authority and the OpenAPI addition is follow-up work in the contracts
   repository.
2. **Progress sync result vocabulary.** The contract declares `results[].status` as
   `updated`, `ignored`, or `error`; the server emits `ok` for both applied and skipped rows, so a
   client cannot distinguish an applied write from one discarded by the minimum-resume floor.
   Closed by aligning the server, recorded as a pre-lock correction before it ships.
3. **No aggregate capability endpoint.** Both clients decode a `CapabilitySet` of
   `schema_version`, `media_types`, and `features`, and the contracts define the schema, but no
   path serves it. Closed by adding `GET /api/v1/capabilities`.
4. **Server identity is borrowed.** `server_id` and `server_name` on health come from the
   Jellyfin-compatibility configuration and may be absent, which is not a sound basis for scope
   keying. Closed by adding `GET /api/v1/server/identity` backed by a dedicated stored identifier,
   leaving health untouched.
5. **Playback start omits fields the server supports.** The contract's start request does not
   declare `start_position` or `progress_persistence`, both of which the server accepts and both of
   which a resume-capable client needs. The contract's start request permits additional
   properties, so clients may send them today; the declarations are follow-up work in the contracts
   repository.
6. **Profile verification is undeclared.** Profile-scoped endpoints accept a profile token
   obtained by verifying a profile PIN, and the contract declares neither the verification endpoint
   nor the token header. Clients implement it against the server's behavior; the declaration is
   follow-up work in the contracts repository.
7. **Progress rows carry no file or series identity.** The contract's progress entry has a media
   item identifier only, while the clients key checkpoints by content, episode, and media file.
   Closed in the merge rule rather than on the wire.
8. **Completion thresholds differ.** The client's ninety-percent-or-final-two-minutes rule and the
   server's ninety-percent-with-a-five-percent-floor rule diverge for content under twenty minutes
   and in the opening five percent of any title. Closed by making the server authoritative on merge
   and naming the divergence.
9. **Device pairing is undeclared.** The contract declares only the device capability endpoint;
   start, lookup, poll, approve, and deny are absent from it. Clients implement against the
   server's behavior; the declaration is follow-up work in the contracts repository.

## Risks and open questions

- **Where the approval surface lives.** The pairing approval page answers at a known path but is
  not linked from the signed-in navigation. Whether it belongs in the account menu, in a devices
  settings screen, or as a prompt raised by an event channel when a pairing request appears is a
  product decision for the maintainer.
- **Discovery default.** This design defaults the broadcast on because a television that finds the
  server by itself is the entire reason to build discovery. A privacy-first default of off, with
  discovery enabled during setup, is a defensible alternative and is the maintainer's call.
- **Which API the Watch documents belong to.** They are specified on `/api/v1` so the shipping
  clients can use them. If the native API is where server-composed presentation should live
  long-term, the `/api/v1` endpoints become a projection to maintain.
- **Completion thresholds.** Exposing the server's watched and minimum-resume thresholds to
  clients, so both sides complete at the same instant, would remove the divergence at the cost of a
  new client dependency on a server setting.
- **Pairing protocol version.** The device capability endpoint advertises protocol version two for
  device login. Whether a television login should advertise a distinct version, and what a version
  bump would mean for already-paired devices, is unresolved.
- **Header refresh.** The server never emits session-scoped header refresh today. The client path
  is specified and tested, but it is untested against a server that actually uses it, and it will
  stay that way until one does.
- **Multi-server households.** Scope keying supports several servers, and the clients store state
  per server, but the onboarding flow presents one at a time. A server switcher is deliberately
  deferred.
- **Dependency addition.** The discovery responder is the first mDNS dependency in the server
  repository. Its module, version, license, and maintenance status need recording at the point it
  is added, and a vendored minimal responder is the fallback if none is acceptable.

## Acceptance criteria

- A television finds a broadcasting server on the local network, and a typed origin reaches the
  same server, with the discovered identity confirmed over the validated transport before use.
- An operator can switch the broadcast off, and the server stops advertising without a restart.
- A viewer signs in with credentials on the television, and a viewer signs in by approving the
  television from a signed-in session, and both arrive at the same activation.
- Profile selection and PIN verification produce a profile-scoped activation, and a profile never
  sees an item it is not permitted to see.
- Watch home and detail render the viewer's own library from `/api/v1`, and the client code path is
  the same one the fixture service exercises.
- Playback plays real media through a validated protocol-v3 plan, handles expiry and refresh
  without a false failure, and releases its session on exit.
- A checkpoint written on one device appears on another, and the newest update wins in both
  directions.
- Every server failure renders as one of the five safe categories, with the correct one for each.
- No token, code, complete URL, title, or profile identifier appears in any log or diagnostic, and
  no release artifact contains a fixture source, fixture token, or development origin.
- Server tests, both clients' unit and integration suites, the contract conformance run,
  accessibility checks, and clean-room audits pass, with hardware behavior claimed only after
  connected-device execution.
