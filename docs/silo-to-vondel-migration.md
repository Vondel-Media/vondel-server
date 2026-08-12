# Migrating from Silo to Vondel

Vondel deliberately retains selected Silo protocol and storage identifiers so
an existing installation can be evaluated without a destructive data rewrite.
Back up PostgreSQL, the data directory, and `SECRET_KEY` before changing images.

For an existing Compose deployment:

1. Stop the Silo application container, leaving PostgreSQL and Redis intact.
2. Set `VONDEL_IMAGE=ghcr.io/vondel-media/vondel-server:latest`.
3. Point `VONDEL_DATA_ROOT` at the existing host data root. It may still be
   named `/opt/silo`; renaming it is optional.
4. Keep the same database credentials and `SECRET_KEY`.
5. Start Vondel and inspect its logs before allowing schema migrations to run
   against your only database copy.

Internal container paths such as `/var/lib/silo`, `/tmp/silo-transcode`, and
the `SILO_PLUGIN_CACHE_DIR` setting remain compatibility contracts for now.
Likewise, `X-Silo-*`, `_silocast._tcp`, and existing plugin IDs are not public
branding and must not be rewritten casually: official Silo clients and servers
may depend on them.
