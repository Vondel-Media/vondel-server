# Vondel Client Harness and TV Watch Handoff

**Date:** 2026-08-14

**Purpose:** Authoritative continuation note for the cross-platform client test harness and TV Watch vertical slice. This note contains no credentials or reference-client source.

## Governing documents

Read these before taking action:

- `docs/superpowers/specs/2026-08-13-vondel-tv-watch-vertical-slice-design.md`
- `docs/superpowers/plans/2026-08-13-vondel-tv-watch-contract-fixtures.md`
- `docs/superpowers/plans/2026-08-13-vondel-apple-tv-watch.md`
- `docs/superpowers/plans/2026-08-13-vondel-android-tv-watch.md`
- `docs/superpowers/specs/2026-08-13-vondel-cross-platform-client-test-harness-design.md`
- `docs/superpowers/plans/2026-08-13-vondel-client-test-harness-foundation.md`
- `docs/superpowers/plans/2026-08-13-vondel-test-controller.md`
- `docs/superpowers/plans/2026-08-13-vondel-apple-test-driver.md`
- `docs/superpowers/plans/2026-08-13-vondel-android-test-driver.md`
- `docs/superpowers/plans/2026-08-13-vondel-client-device-lab.md`

The work is using `superpowers:subagent-driven-development`: one fresh implementer per task, an independent task-scoped reviewer, at most five fix rounds, then a whole-branch review. Preserve each repository's uncommitted `.superpowers/sdd` ledger and reports.

## Non-negotiable rulings

- Live TV, IPTV, DVR, EPG, `.strm`, and arbitrary remote-stream shortcuts are prohibited.
- Clean-room implementers do not inspect reference-client source. Similarity audits are run separately and return sanitized candidate-side evidence only.
- TV automation uses semantic accessibility IDs and real remote/D-pad events. Coordinates and arbitrary key codes are not part of the harness protocol.
- Production server/client binaries must not contain the test controller, privileged diagnostics, harness transports, launch switches, fixture credentials, or seed packages.
- The test controller is a separate non-production component and remains unavailable in production builds, including to administrators.
- Account/profile data and playback progress are keyed by exact scope and activation generation.
- Do not weaken production cleartext/network policy to reach a development server. Apple uses a harness-managed loopback proxy; Android uses `adb reverse` to loopback.
- The user supplied a development server origin and static login interactively. They are intentionally absent from Git, reports, this note, and command history. Retrieve credentials securely from the user when required. The known origin is `http://vondel:8090`; use it only for ordinary API/conformance checks until the non-production controller exists. Do not run destructive resets against it.

## Repository status

### Design and plan repository

- Worktree: `/Users/jimcole/projects/vondel-server/.worktrees/tv-watch-design`
- Branch: `codex/tv-watch-design`
- HEAD before this handoff commit: `093cb0c84538ff192abaef862b7ad124463cd201`
- Relative to `origin/main`: ahead 4, behind 4 before this note. Rebase deliberately before integration; do not discard the four local design/plan commits.
- Local commits:
  - `3e8b27cb` — TV Watch design
  - `81bceecf` — TV Watch implementation plans
  - `1c715ea7` — cross-platform harness design
  - `093cb0c8` — five harness/controller/driver/lab plans

### Contracts fixture repository — complete

- Worktree: `/Users/jimcole/projects/.worktrees/vondel-client-contracts-tv-watch-contracts`
- Branch: `codex/tv-watch-contract-fixtures`
- HEAD: `e876c657fc7ff568daf870e2f6fa633d8e08c3b2`
- Status: implementation Tasks 1–5 complete; task reviews and final whole-branch review passed.
- Corrected implementation candidate `6e4adaae27106b8201fbf550a2c0f0f378c3d158` passed authorized Apple and Android reference audits with zero unexplained findings. `e876c65` is the evidence-only follow-up.
- Verification included 87 contract tests before the final fix wave, full/race/vet checks, generated SDR H.264/AAC media, HTTP range behavior, direct executable lifecycle cleanup, episode identity, OpenAPI progress shape, strict progress sync, and deterministic completion/expiry fixtures.
- Preserve the untracked user/scratch file `final-fix-report.md`.
- The branch has not been pushed or merged in this sequence.

### Harness foundation repository — Tasks 1–4 complete; Task 5 interrupted

- Repository: `/Users/jimcole/projects/vondel-client-test-harness`
- Branch: `codex/harness-foundation`
- HEAD: `b01e1f4a39025e1d6d7c31ce323258fb948cf8c3`
- This is a new zero-parent repository with module path `github.com/Vondel-Media/vondel-client-test-harness`.
- SDD ledger: `.superpowers/sdd/2026-08-13-vondel-client-test-harness-foundation/progress.md`

Completed and independently approved:

1. Protocol contracts through `e9275d0`
   - Closed scenario/driver schemas and safe semantic identifier grammar.
   - Coordinate and arbitrary-key input absent.
   - JSON Schema/Go scalar presence, nullability, nonce, and ordering rules aligned.
2. Scenario executor through `76cb8c0`
   - Bounded YAML/JSON decoding, deterministic IDs, safe variable placement, executor-owned monotonic playback timing, cancellation, and driver cleanup.
3. Focus crawler through `2a6a496`
   - Bounded BFS, restore/replay before probes, shortest paths, strict global observation sequence, visibility topology, reserved harness namespace filtering, trap/back/stall/reachability invariants, and hostile observation validation.
4. Evidence writer through `b01e1f4`
   - Typed deterministic JSON/JUnit/HTML bundles, manifest-last publication, byte-to-scan binding, metadata-free canonical screenshots, 32-screenshot cap, deep encoding detection, cumulative linear work budget, node/depth limits, rollback, and cross-platform builds.
   - Final Task 4 review: ready; zero open Critical/Important/Minor findings.

Task 5 was interrupted before its RED/GREEN cycle completed. Preserve these untracked partial files:

- `internal/driver/transport.go`
- `internal/driver/transport_test.go`
- `internal/driver/transport_unix.go`
- `internal/driver/transport_unix_other.go`

The partial work contains `DialUnix`, literal-loopback-only `DialTCP`, nonce handshake, command serialization, replay/correlation/sequence checks, frame bounds, and Unix socket identity checks. It is not committed or reviewed. There is no completed Task 5 report. Do not assume it is correct; inspect it, establish the planned failing tests, and either continue it or replace it without losing useful work.

Task 6 has not started. No final harness whole-branch review or authorized originality audit has run.

### Apple TV Watch repository — Tasks 1–3 complete; Task 4 awaiting final verdict

- Worktree: `/Users/jimcole/projects/vondel-apple/.worktrees/apple-tv-watch`
- Branch: `codex/apple-tv-watch`
- HEAD: `429f15e752a9136315676b81321718f361fd1212`
- Relative to `origin/main`: ahead 17 before this note.
- SDD ledger: `.superpowers/sdd/2026-08-13-vondel-apple-tv-watch/progress.md`

Completed and independently approved:

1. Watch documents and playback validator through `afbb59a`
2. Stage/Chapters composition and exact-scope progress through `05cf456`
3. Native AVPlayer session through `d18aa8b`
   - Final verification: 154 tests plus one named environment skip, live generated-media integration, and adversarial seek-generation/coalescing tests.

Task 4 browsing/focus implementation is at the fifth and final normal fix round:

- Initial implementation: `0f0c741`
- Fixes: `f71c7b7`, `d76ac22`, `aa38eeb`, `b2e548d`, `429f15e`
- Current implementation provides chapter-qualified focus/IDs, exact movie/episode return destinations, native exit routing, real retry gating, main-actor playback capability ownership, accepted-attempt cancellation, appearance epochs, stale-control rejection, and atomic published `WatchRouteRenderState` used by the root.
- Implementer verification at `429f15e`: focused 20/20, full 180/180 with one intentional fixture skip, TSAN 20/20 with no race, XcodeGen, unsigned iOS/tvOS/macOS builds, clean-room self-test, and diff checks.
- Review package: `.superpowers/sdd/2026-08-13-vondel-apple-tv-watch/review-b2e548d..429f15e.diff`
- The final independent reviewer was interrupted before returning the verdict. Resume or rerun that exact scoped review. The sole prior finding was that activation changes were not published to SwiftUI; confirm the new atomic published render state closes it.
- If the final reviewer finds any Critical or Important issue, do not start a sixth ordinary fix round. The five-round limit has been reached; document and escalate the unresolved finding before proceeding.
- Preserve untracked `task-4-review.md`.

Apple Tasks 5 (Quiet Timeline/lifecycle progress) and 6 (test-only fixture bootstrap/tvOS acceptance) have not started. No Apple final whole-branch review or final originality audit has run.

### Android TV Watch repository — not started

- Worktree: `/Users/jimcole/projects/vondel-android/.worktrees/android-tv-watch`
- Branch: `codex/android-tv-watch`
- HEAD: `6f4d3c0`
- Baseline `./gradlew test --console=plain` passed before planning.
- Android Watch Tasks 1–6 have not started.
- The prior Android presentation foundation was separately approved, including managed-device Keystore tests on API 26 and API 35.

### Non-production server controller — planned, not started

- Plan: `docs/superpowers/plans/2026-08-13-vondel-test-controller.md`
- Do not implement in `/Users/jimcole/projects/vondel-server/.worktrees/opa-tenant-foundation`; reconnaissance found that worktree dirty with unrelated changes.
- Start a fresh isolated server worktree from a reviewed clean base.
- The existing server contains useful disposable database, account provisioning, tenancy, settings-contract, catalog-seed, conformance, and audit patterns described in the plan.

### Native harness drivers and device lab — planned, not started

- Apple native harness work starts only after Apple Watch Tasks 1–6 stabilize.
- Android native harness work starts only after Android Watch Tasks 1–6 stabilize.
- Device lab starts after foundation, controller, and both native drivers are reviewed.

## Exact next actions

1. Resume the final Apple Task 4 reviewer on `b2e548d..429f15e` using the existing review package. If clean, append `Task 4: complete` to the Apple SDD ledger. If not clean, escalate because fix round 5/5 is exhausted.
2. Resume Harness Foundation Task 5 from the four untracked transport files. Re-establish strict RED before accepting any partial implementation, finish the task, commit, and run an independent task review/fix loop.
3. Execute Harness Foundation Task 6, then run an authorized clean-room audit and final whole-branch review over the zero-parent repository.
4. Execute and review Apple Tasks 5 and 6, then run Apple final acceptance, authorized audit, and whole-branch review.
5. Execute Android Watch Tasks 1–6 with fresh implementers/reviewers, then Android final acceptance/audit/review.
6. Only after functional Watch interfaces stabilize, execute the test-controller and Apple/Android harness-driver plans; then execute the device-lab plan.
7. Use the development server only through a secret-safe, loopback-compatible test path. Never put its password into source, environment dumps, process arguments, reports, screenshots, or this repository.

## Integration state

- None of the new contracts, harness, or Apple TV Watch branches in this handoff have been pushed or merged as part of the current sequence.
- Do not delete worktrees or untracked review/report/partial files.
- Before integration, refresh remote state, rebase each branch deliberately, rerun full repository verification and clean-room audits, then use the `finishing-a-development-branch` workflow to choose merge/push/PR handling.
