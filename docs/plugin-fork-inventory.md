# Vondel Plugin Fork Inventory

Audit completed on 2026-08-12. This is the Task 7 whole-family gate for the eight private repositories created by the full plugin-forks plan.

## Audit result

- All eight repositories are `PRIVATE` on GitHub with default branch `main`.
- Every audited local `main` is clean and exactly matches `origin/main`.
- Each repository has exactly one zero-parent Vondel root. No upstream Git history or upstream tag is reachable from a Vondel repository.
- No repository currently has a local or GitHub tag. No repository has a GitHub release.
- The catalog and five licensed plugin imports have successful GitHub Actions CI at the exact audited HEAD. Ebook and Autoscan ARR intentionally have no workflow.
- Every imported license is byte-for-byte identical to the license at its pinned upstream snapshot.
- Every Go repository uses its Vondel module path, pins `github.com/Vondel-Media/vondel-plugin-sdk v0.13.3`, and has no `replace` directive in current files or reachable history.
- Compatibility identities remain in the Silo v1 namespace. Repository and build ownership changed to Vondel; wire, storage, provider, capability, configuration, and image-scheme identities did not.
- Known embedded TMDB and TVDB upstream credentials are absent from current files and all reachable Vondel history. No executable workflow contains a Silo catalog dispatch, credential-bearing URL, public-registry publication, or repository-visibility mutation.
- Ebook and Autoscan ARR remain private permission gates containing no copied upstream source.

## Exact repository state

| Repository | Audited HEAD | Zero-parent root | Upstream snapshot | License | Module / SDK | CI at HEAD | Tags / releases |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `Vondel-Media/vondel-plugins` | `8092a780eee2091894bf3a25087b115a925d12cf` | `42f7f3ca2dc78246eb68368cc323bc0e9899915a` | `Silo-Server/silo-plugins@934b867f69e6b140dcaf90efe56c43da63893f3f` | Apache-2.0 | `github.com/Vondel-Media/vondel-plugins`; SDK `v0.13.3` | success, run [`31597408813`](https://github.com/Vondel-Media/vondel-plugins/actions/runs/31597408813) | none / none |
| `Vondel-Media/vondel-plugin-metadb` | `0115c5b222fd2d913289bdb011694aad796d5ce7` | `304fb84e1619afa64f54d535477c86aa7b2536ce` | `RXWatcher/silo-plugin-metadb@daf85c59aac440538d13223db6d644ccd240e345` | AGPL-3.0-or-later | `github.com/Vondel-Media/vondel-plugin-metadb`; SDK `v0.13.3` | success, run [`31599574925`](https://github.com/Vondel-Media/vondel-plugin-metadb/actions/runs/31599574925) | none / none |
| `Vondel-Media/vondel-plugin-tmdb` | `cad31a3a683b50107a126dd7a252c48527c10cfe` | same as HEAD | `Silo-Server/silo-plugin-metadata-tmdb@v1.2.21` (`7bae6ba7d99f49587128dfcb0a56bee6800c6ad0`) | AGPL-3.0-or-later | `github.com/Vondel-Media/vondel-plugin-tmdb`; SDK `v0.13.3` | success, run [`31599405419`](https://github.com/Vondel-Media/vondel-plugin-tmdb/actions/runs/31599405419) | none / none |
| `Vondel-Media/vondel-plugin-tvdb` | `084b05a2d2b3d218168e379574e73d9e529f9672` | same as HEAD | `Silo-Server/silo-plugin-metadata-tvdb@v1.2.24` (`a04e5a21427e2d9c2f63522603190c81f0017da8`) | AGPL-3.0-or-later | `github.com/Vondel-Media/vondel-plugin-tvdb`; SDK `v0.13.3` | success, run [`31600316437`](https://github.com/Vondel-Media/vondel-plugin-tvdb/actions/runs/31600316437) | none / none |
| `Vondel-Media/vondel-plugin-audiobook-metadata` | `fea20422d4d086519b9517ab0dae870092324068` | `5c35b8cf1e6cc56b0f2eb232f0f8e389a5c146ea` | `Silo-Server/silo-plugin-metadata-audiobook@v0.1.4` (`f85c630ecbbe5ef760cae1af4b8fdfb1d666397c`) | AGPL-3.0-only | `github.com/Vondel-Media/vondel-plugin-audiobook-metadata`; SDK `v0.13.3` | success, run [`31599245243`](https://github.com/Vondel-Media/vondel-plugin-audiobook-metadata/actions/runs/31599245243) | none / none |
| `Vondel-Media/vondel-plugin-manga-metadata` | `13d41c7406d729fa18595c9a3aee965efa6628ed` | `600c4981452a587359e4f94a431c4ba7843b7a1c` | `Silo-Server/silo-plugin-metadata-manga@v0.1.1` (`1458a58663081bff45da3230f783d55d4955852a`) | AGPL-3.0-only; MangaBaka data remains CC BY-NC-SA 4.0 | `github.com/Vondel-Media/vondel-plugin-manga-metadata`; SDK `v0.13.3` | success, run [`31599256457`](https://github.com/Vondel-Media/vondel-plugin-manga-metadata/actions/runs/31599256457) | none / none |
| `Vondel-Media/vondel-plugin-ebook-metadata` | `5583aaf95fe3690f9be4f32df03da6f276c6688f` | same as HEAD | metadata only: `Silo-Server/silo-plugin-metadata-ebook@v0.1.1` (`70d8acb4dbb68f8cc594099258211cd9a3a3082f`) | none at reviewed snapshot; import blocked | none | intentionally none | none / none |
| `Vondel-Media/vondel-plugin-autoscan-arr` | `bde63ecf3f9e74d4bb84c696a54edc17b2f101ff` | same as HEAD | metadata only: `Silo-Server/silo-plugin-autoscan-arr@v0.1.2` (`7987ddae852549f5f2ef4e00b6f25dfa5497ddad`) | none at reviewed snapshot; import blocked | none | intentionally none | none / none |

All eight origins use `git@github.com:Vondel-Media/<repository>.git` for fetch and push. Upstream remotes are fetch-only:

| Repository | Upstream fetch URL | Upstream push URL |
| --- | --- | --- |
| `vondel-plugins` | `https://github.com/Silo-Server/silo-plugins.git` | `DISABLED` |
| `vondel-plugin-metadb` | `https://github.com/RXWatcher/silo-plugin-metadb.git` | `no_push` |
| `vondel-plugin-tmdb` | `https://github.com/Silo-Server/silo-plugin-metadata-tmdb.git` | `no_push` |
| `vondel-plugin-tvdb` | `https://github.com/Silo-Server/silo-plugin-metadata-tvdb.git` | `no_push` |
| `vondel-plugin-audiobook-metadata` | `https://github.com/Silo-Server/silo-plugin-metadata-audiobook.git` | `DISABLED` |
| `vondel-plugin-manga-metadata` | `https://github.com/Silo-Server/silo-plugin-metadata-manga.git` | `DISABLED` |
| `vondel-plugin-ebook-metadata` | `https://github.com/Silo-Server/silo-plugin-metadata-ebook.git` | `DISABLED` |
| `vondel-plugin-autoscan-arr` | `https://github.com/Silo-Server/silo-plugin-autoscan-arr.git` | `DISABLED` |

## License evidence

The catalog license has SHA-256 `6cf63d87586c7c25c3ab8e62eef5c75bbaa982b0a6f9c00c59e2720255aac9ec`. The five licensed plugin snapshots share preserved `LICENSE` SHA-256 `be81e1ded9055f0a6c6ea337716140e73fc10a3d18a482f5ade024364cc0f2e0`. Each digest was recomputed from the Vondel file and its exact pinned upstream file and matched byte-for-byte.

The Ebook and Autoscan ARR snapshots contain no redistribution license. Each Vondel repository therefore has exactly four tracked files:

- `.gitignore`
- `README.md`
- `SOURCE_IMPORT_BLOCKED`
- `UPSTREAM.md`

They have no copied source, fixture, manifest, Go module, workflow, or license assertion. Import remains blocked until the copyright holder supplies a redistribution license or written permission.

## Compatibility inventory

All imported plugin manifests declare `silo_api_version: v1` and support `darwin/arm64`, `linux/amd64`, and `linux/arm64`.

| Plugin | Manifest version | Plugin ID | Capability IDs and invariant metadata | Preserved configuration |
| --- | --- | --- | --- | --- |
| MetaDB | `1.0.0` | `silo.metadb` | `metadata_provider.v1/metadb`; movie, series, season, episode priorities all `1` | `SILO_METADB_URL`, `SILO_METADB_API_KEY`, `SILO_METADB_S3_ENDPOINT`, `SILO_METADB_S3_REGION`, `SILO_METADB_S3_BUCKET`, `SILO_METADB_S3_ACCESS_KEY`, `SILO_METADB_S3_SECRET_KEY` |
| TMDB | `1.2.21` | `silo.tmdb` | `metadata_provider.v1/tmdb`; priorities movie `2`, series/season/episode `3`; `image_resolver.v1/tmdb`, scheme `tmdb`, priority `100` | required global secret/password `api_key`, held in memory |
| TVDB | `1.2.24` | `silo.tvdb` | `metadata_provider.v1/tvdb`; series/season/episode priorities all `2`; `image_resolver.v1/tvdb`, scheme `tvdb`, priority `100` | required global secret/password `api_key`; bearer token remains in memory |
| Audiobook Metadata | `0.1.4` | `silo.audiobook-metadata` | `metadata_provider.v1/audiobook-metadata`; audiobook priority `2` | none |
| Manga Metadata | `0.1.1` | `silo.manga-metadata` | `metadata_provider.v1/manga-metadata`; manga priority `1` | `enabled_sources`, `enable_local_dump`, `dump_path`, `dump_refresh_hours`, `enable_anilist_banners` |

MetaDB intentionally does not advertise `image_resolver.v1`; adding that capability requires a separate behavior and registration compatibility review.

Each release-capable plugin preserves the later catalog asset contract:

- `plugin-linux-amd64`
- `plugin-linux-arm64`
- `plugin-darwin-arm64`
- `checksums.txt`

The manifests retain `__CHECKSUM__` until the release/catalog publication phase.

## Automation inventory

All listed workflows are active:

- Catalog: `CI` and `Update Catalog`.
- MetaDB: `CI`; this plan does not create a release workflow for MetaDB.
- TMDB and TVDB: `CI` and tag-only `Private Release`.
- Audiobook and Manga: `CI` and tag-only `Private release`.
- Ebook and Autoscan ARR: no workflows.

The catalog updater uses Vondel-only source and push credentials and a literal Vondel repository allowlist. Release workflows verify repository privacy and an exact source tag before preparing the four expected assets. None has been triggered because there are no tags or releases.

## Verification evidence

For the catalog, the audit passed:

```text
GOWORK=off go mod tidy -diff
GOWORK=off go test ./...
GOWORK=off go vet ./...
GOWORK=off go build ./...
jq empty manifest.json
git diff --check
```

For MetaDB, TMDB, TVDB, Audiobook, and Manga, the audit additionally passed `go test -race ./...` and clean cross-builds for Linux amd64, Linux arm64, and Darwin arm64. Each repository's applicable private-source, embedded-credential, fork-policy, and private-release guard suites also passed. Manga's optional networked `MANGABAKA_LIVE=1` test was not run and is not part of the non-secret CI gate.

## Secret and publication scan

The audit derived the two known credentials directly from the exact upstream TMDB and TVDB commits, without reproducing their values. Their non-secret fingerprints were:

- TMDB: length 32, SHA-256 prefix `b606194c5bac`
- TVDB: length 36, SHA-256 prefix `883b8232685a`

For every Vondel repository, exact-value searches returned zero current files and zero reachable commit paths. Reachable-history searches also returned zero Go `replace` directives. Direct workflow searches returned zero credential-bearing URLs, Silo catalog dispatch tokens/endpoints, public-registry publication commands, and visibility-to-public mutations.

Broad whole-history searches match deliberately forbidden examples or detection regexes in TMDB/TVDB guard tests and `scripts/verify-private-source.sh`. These are negative test fixtures, not credentials or executable workflow actions. Factual upstream URLs in `NOTICE` and `UPSTREAM.md` are required provenance and are not publication regressions.

## Delivery gate and next phase

The source-fork phase is complete, but the private GitHub catalog and private GitHub releases cannot be consumed anonymously by the current server. There are deliberately no tags, plugin release assets, or catalog entries yet.

The next phase must:

1. create reviewed private plugin releases with the exact four-asset contract;
2. add those releases to the private catalog only after checksum and platform validation;
3. establish a local or authenticated staging delivery path instead of anonymous GitHub fetching; and
4. run the core movie/series scan acceptance suite against staged TMDB and TVDB delivery before production publication decisions.
