# Vondel Server Hard-Fork Rebrand Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish a legally distinct, buildable `Vondel-Media/vondel-server` hard fork whose public identity and original brand assets are Vondel while its Silo-compatible wire contracts remain intact.

**Architecture:** Rebrand the Go module, executable, web application, deployment defaults, documentation, and reserved visual assets. Preserve compatibility identifiers that are part of public protocols or persisted schemas, including `X-Silo-*` headers, `silo.plugin.v1`, existing plugin IDs, migration names, and database columns; document those retained names as compatibility surfaces rather than Vondel branding.

**Tech Stack:** Go 1.24+, React/TypeScript, Vite, Docker Compose, SVG/PNG/ICO brand assets, GitHub CLI.

## Global Constraints

- Keep `AGPL-3.0-or-later`, upstream copyright notices, source history, warranty notices, and corresponding-source obligations intact.
- Add prominent notices that Vondel is modified from Silo and is not affiliated with or endorsed by Silo Media L.L.C.
- Do not redistribute the Silo logo, wordmark, icon, or other reserved Silo brand assets.
- Use “Silo” only for truthful upstream attribution or compatibility identifiers.
- Keep `X-Silo-*`, `_silocast._tcp`, `silo.plugin.v1`, existing `silo.*` plugin IDs, migration identifiers, database identifiers, and encrypted setting keys unchanged where interoperability or stored-data compatibility requires them.
- Do not change historical migration files solely for cosmetic renaming.
- Do not introduce local absolute paths into repository documentation.

---

### Task 1: Fork identity and legal notices

**Files:**
- Modify: `go.mod`
- Modify: all first-party Go imports referencing the old server module
- Modify: `Makefile`
- Modify: `Dockerfile`
- Modify: `Dockerfile.dev`
- Modify: `README.md`
- Create: `NOTICE`
- Replace: `TRADEMARK.md`

**Interfaces:**
- Produces: canonical module `github.com/Vondel-Media/vondel-server`, executable `vondel`, and unambiguous upstream attribution.

- [ ] Add a failing repository identity test that asserts the module, README title, executable target, and legal notices use Vondel.
- [ ] Run the identity test and confirm it fails on the upstream Silo checkout.
- [ ] Change the module/import path and build metadata path to `github.com/Vondel-Media/vondel-server`.
- [ ] Rename the first-party command from `cmd/silo` to `cmd/vondel` and update build commands.
- [ ] Rewrite the README introduction, installation paths, repository URLs, and attribution for Vondel.
- [ ] Add a Vondel trademark policy and an upstream attribution/modification notice without altering `LICENSE`.
- [ ] Run identity and Go compilation tests.
- [ ] Commit as `chore(brand): establish Vondel fork identity`.

### Task 2: Public product defaults and web identity

**Files:**
- Modify: `internal/branding/branding.go`
- Modify: `internal/config/admin_settings.go`
- Modify: `internal/config/db_loader.go`
- Modify: public notification sender defaults under `internal/notifications/`
- Modify: `web/index.html`
- Modify: `web/public/site.webmanifest`
- Modify: `web/public/sw.js`
- Modify: user-facing React copy under `web/src/`
- Test: `internal/branding/*_test.go`

**Interfaces:**
- Produces: `branding.DefaultServerName == "Vondel"` and Vondel defaults before custom white-label settings load.

- [ ] Change branding tests to expect Vondel defaults and confirm they fail.
- [ ] Replace public defaults, document titles, manifest names, notifications, support text, and UI labels with Vondel.
- [ ] Keep compatibility header names and protocol identifiers unchanged.
- [ ] Run branding tests and web type/lint checks.
- [ ] Commit as `feat(brand): apply Vondel product defaults`.

### Task 3: Original Vondel visual assets

**Files:**
- Create: `brand/vondel-mark.svg`
- Create: `brand/vondel-wordmark.svg`
- Replace: `assets/icon.png`
- Replace: `web/public/favicon.ico`
- Replace: `web/public/apple-touch-icon.png`
- Replace: `web/public/maskable-icon-512.png`
- Replace: `web/public/web-app-icon-192.png`
- Replace: `web/public/web-app-icon-512.png`
- Replace/rename: `web/public/silo-icon-1024.png`
- Replace/rename: `web/public/silo-wordmark-sidebar.png`
- Modify: asset references under `web/src/` and server templates
- Replace: `web/public/NOTICE`

**Interfaces:**
- Produces: an original canal-ribbon `V` mark, complete raster icon set, and no redistributed Silo visual identity.

- [ ] Add a failing asset-reference test that rejects `silo-icon` and `silo-wordmark` paths in shipped web sources.
- [ ] Create deterministic SVG master artwork with a canal-ribbon `V` and warm Amsterdam red accent.
- [ ] Render required PNG/ICO sizes from the SVG master and rename all references.
- [ ] Replace the old brand-asset notice with Vondel copyright and upstream non-affiliation language.
- [ ] Run the asset-reference test and web build.
- [ ] Commit as `feat(brand): add original Vondel visual system`.

### Task 4: Deployment and migration-safe naming

**Files:**
- Modify: `.env.example`
- Modify: `docker-compose.yml`
- Modify: `docker-compose.nvidia.yml`
- Modify: deployment scripts and current operator documentation
- Create: `docs/silo-to-vondel-migration.md`

**Interfaces:**
- Produces: `VONDEL_*` deployment variables, `vondel` service/image names, `/opt/vondel` and `/var/lib/vondel` defaults, plus explicit migration guidance.

- [ ] Add failing compose assertions for Vondel service, image, and storage defaults.
- [ ] Rename deployment-facing settings and paths while providing documented legacy `SILO_*` migration mapping.
- [ ] Keep database schema identifiers and encrypted setting keys unchanged.
- [ ] Validate Compose interpolation and run deployment-focused tests.
- [ ] Commit as `chore(deploy): rebrand Vondel deployment surface`.

### Task 5: Final clearance, verification, and GitHub publication

**Files:**
- Modify: `.github/ISSUE_TEMPLATE/*` and current contribution/security documentation
- Create: `scripts/verify-vondel-brand.sh`

**Interfaces:**
- Produces: a repeatable brand-boundary check and public repository `Vondel-Media/vondel-server`.

- [ ] Create a verifier that rejects prohibited public Silo identity while allowlisting documented compatibility and attribution surfaces.
- [ ] Run `gofmt` and `go mod tidy`.
- [ ] Run `make test-go`, frontend lint/type/build tests, Compose validation, and `make verify-local-paths`.
- [ ] Inspect `git diff`, generated assets, and repository status for accidental upstream brand assets or secrets.
- [ ] Create the public GitHub repository without auto-generated files.
- [ ] Add `origin`, push `main`, set the repository description/topics/homepage, and verify the remote default branch and public files.
- [ ] Report the repository URL, verification evidence, retained compatibility identifiers, and remaining client-fork work.
