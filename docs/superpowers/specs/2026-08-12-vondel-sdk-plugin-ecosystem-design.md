# Vondel SDK and Plugin Ecosystem Design

## Objective

Create a Vondel-owned plugin ecosystem that follows Silo's proven structure,
supports media scanning before native clients exist, and remains compatible
with both Vondel Server and official Silo servers.

The first end-to-end success criterion is:

> A fresh Vondel Server installation can load the Vondel catalog, install
> Vondel-published metadata plugins, add media libraries, scan them, and store
> matched metadata and artwork.

## Compatibility strategy

The ecosystem will use a compatibility mirror rather than introduce a new
plugin protocol. Public project and repository names become Vondel names, but
wire and persistence contracts remain stable where changing them would break
existing installations or official Silo servers.

The initial releases retain:

- the protobuf package and service namespace `silo.plugin.v1`;
- the manifest field `silo_api_version` and its current `v1` value;
- existing stable `silo.*` plugin IDs;
- capability names such as `metadata_provider.v1`, `image_resolver.v1`, and
  `scan_source.v1`;
- existing configuration keys and stored installation identities.

These identifiers are compatibility contracts, not Vondel product branding.
User-facing names, repository names, documentation, release ownership, and
catalog URLs will use Vondel.

## Repository family

The first wave creates these public repositories in `Vondel-Media`:

1. `vondel-plugin-sdk`
2. `vondel-plugins`
3. `vondel-plugin-metadb`
4. `vondel-plugin-tmdb`
5. `vondel-plugin-tvdb`
6. `vondel-plugin-audiobook-metadata`
7. `vondel-plugin-ebook-metadata`
8. `vondel-plugin-manga-metadata`
9. `vondel-plugin-autoscan-arr`

The SDK and catalog retain their upstream Apache-2.0 licenses. Plugin forks
retain their respective upstream licenses, including AGPL-3.0-or-later where
applicable. Every repository will include a notice naming its Silo upstream,
the imported revision, material Vondel modifications, and the lack of Silo
affiliation or endorsement.

Each repository begins with a clean Vondel root commit so reserved Silo visual
assets and unrelated historical automation are not republished through the new
Git history. Required copyright and license notices remain in the source
snapshot and root notice.

## SDK architecture

`vondel-plugin-sdk` becomes the Vondel module consumed by Vondel-owned plugins.
It contains the generated protobuf API, capability constants, manifest tools,
runtime scaffolding, host callback client, and reference examples already
proven upstream.

The Go module path changes to:

```text
github.com/Vondel-Media/vondel-plugin-sdk
```

Generated Go imports and Vondel plugin source imports follow the new module
path. The protobuf package/service names and serialized field numbers remain
unchanged. A compatibility test will launch one test plugin built with the
Vondel SDK against the Vondel host and an unmodified official Silo Server build
at the upstream revision from which Vondel forked. Serialized host fixtures
provide faster contract checks between those end-to-end runs.

Vondel Server will move from the upstream SDK dependency to a tagged Vondel SDK
release only after the SDK's parity and wire-compatibility suites pass.

## Catalog architecture

`vondel-plugins` is the canonical Vondel catalog and catalog tooling repository.
Its manifest points exclusively to Vondel-owned repositories, release assets,
and checksums. It never silently proxies mutable upstream binaries.

The catalog updater receives release dispatches from Vondel plugin repositories,
fetches the tagged manifest and checksums, validates supported platforms and
capabilities, and updates the catalog through a reviewable commit. Repository
tokens and dispatch secrets use Vondel-specific names.

Vondel Server's default catalog URL will switch to the Vondel catalog only after
the initial provider releases exist. Operators may still add the official Silo
catalog as a custom source where the server permits custom catalogs.

## Initial plugin set

### Core movie and television metadata

- MetaDB supplies the local/base metadata provider.
- TMDB supplies movie and series metadata plus `tmdb://` image resolution.
- TVDB supplies television metadata plus `tvdb://` image resolution.

Existing provider IDs, priority semantics, external ID mappings, and image URI
schemes remain stable so rescanning does not fork catalog identity.

### Books and manga

- Audiobook Metadata covers audiobook providers and artwork.
- Ebook Metadata covers ebook metadata sources.
- Manga Metadata covers manga matching, covers, and its optional local index.

These repositories ship after the core movie/television pipeline is green, but
remain part of the first ecosystem wave.

### Scan automation

Autoscan ARR implements `scan_source.v1` for Sonarr and Radarr. The server still
owns timers, path rewrites, validation, deduplication, and scan enqueueing. The
plugin polls its upstream connection and returns changed paths without logging
secrets.

## Data flow

1. Vondel Server downloads the Vondel catalog.
2. The administrator selects a plugin release appropriate for the host OS and
   architecture.
3. The server verifies the published checksum and introspects the embedded
   manifest before installation.
4. The plugin starts on the existing broker and binds its runtime and capability
   services.
5. A library scan extracts local media identity and invokes the configured
   metadata providers in their declared priority order.
6. Providers return metadata, external IDs, and logical image references.
7. Image resolver capabilities convert supported references into fetchable
   image URLs.
8. The server persists normalized catalog data and artwork references.
9. ARR scan-source events may enqueue later incremental rescans through the same
   server-owned scan pipeline.

## Release and dependency order

Releases occur in this order:

1. Publish and tag `vondel-plugin-sdk`.
2. Build MetaDB, TMDB, and TVDB against that exact SDK tag.
3. Publish their multi-platform binaries and checksums.
4. Publish `vondel-plugins` with those verified releases.
5. Switch Vondel Server's SDK dependency and default catalog URL.
6. Run the clean-install movie and series scan acceptance suite.
7. Repeat the plugin release/catalog process for audiobook, ebook, manga, and
   ARR autoscan.

No repository uses local `replace` directives in a release commit. SDK changes
must be tagged before downstream release builds resolve them.

## Supported release platforms

The initial release matrix follows the current Silo plugin ecosystem:

- Linux AMD64
- Linux ARM64
- macOS ARM64 where the plugin supports it upstream

Each release publishes deterministic binary names and a checksum file. The
catalog only advertises assets that exist and whose checksum verification has
passed.

## Error handling and security

- Catalog loading rejects malformed entries, missing checksums, unsupported
  platforms, and incompatible API versions.
- Installation fails closed when checksum or manifest verification fails.
- Plugin startup failures are isolated to the installation and surfaced with
  actionable server logs.
- Provider timeouts and rate limits do not corrupt previously stored metadata.
- API keys and provider credentials remain transient secrets and are redacted
  from logs, test fixtures, releases, and catalog metadata.
- Catalog automation receives only the minimum repository permissions needed
  for release discovery and catalog updates.

## Testing

Every repository runs its upstream unit tests after module and identity changes.
The SDK adds serialized compatibility fixtures that guard protobuf service names,
field numbers, manifest keys, and capability values.

The ecosystem integration suite covers:

- building a self-describing plugin with the tagged Vondel SDK;
- installing it on Vondel Server;
- installing the same test binary on the pinned unmodified official Silo Server
  build;
- installing MetaDB, TMDB, and TVDB through the Vondel catalog;
- scanning representative movie and episodic file layouts;
- persisting titles, external IDs, posters, backdrops, seasons, and episodes;
- rescanning without duplicating catalog identities;
- rejecting a tampered plugin binary;
- recovering cleanly from a provider timeout;
- adding audiobook, ebook, manga, and ARR cases as those releases join the
  catalog.

## Out of scope

The first wave does not create native clients, redesign the plugin protocol,
rename compatibility identifiers, or fork every non-scanning Silo plugin.
Authentication, guest access, request routing, live TV, WHMCS, public catalog,
support, adult-content, themes, and other integrations may follow after the
media-ingestion pipeline is reliable.

## Completion criteria

The first wave is complete when all nine repositories are public under
`Vondel-Media`, their attribution and licenses are correct, release automation
publishes verified assets, the Vondel catalog references only Vondel-controlled
releases, Vondel Server consumes the tagged SDK and catalog, and the full media
scan acceptance matrix passes for the initial plugin set.
