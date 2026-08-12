#!/bin/sh
set -eu

fail() {
  printf 'Vondel brand verification failed: %s\n' "$1" >&2
  exit 1
}

[ ! -e web/public/silo-icon-1024.png ] || fail "legacy Silo icon is still shipped"
[ ! -e web/public/silo-wordmark-sidebar.png ] || fail "legacy Silo wordmark is still shipped"

grep -q '<title>Vondel</title>' web/index.html || fail "web title is not Vondel"
grep -q '"name": "Vondel"' web/public/site.webmanifest || fail "manifest name is not Vondel"
grep -q 'DefaultServerName.*= "Vondel"' internal/branding/branding.go || fail "default server name is not Vondel"
grep -q '^# Vondel Server$' README.md || fail "README identity is not Vondel Server"
grep -q '^module github.com/Vondel-Media/vondel-server$' go.mod || fail "Go module path is not Vondel"
grep -q 'ENTRYPOINT \["vondel"\]' Dockerfile || fail "container executable is not Vondel"

printf '%s\n' 'Vondel public-brand checks passed.'
printf '%s\n' 'Compatibility note: selected Silo protocol, plugin, storage, and environment identifiers remain intentionally.'
