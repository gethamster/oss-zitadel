#!/bin/sh
set -eu
if [ -n "${ZITADEL_NODE_HOME:-}" ]; then export PATH="$ZITADEL_NODE_HOME/bin:$PATH"; fi
export GOTOOLCHAIN=go1.26.10 GOMAXPROCS=2 GOFLAGS=-p=2
export NODE_OPTIONS=--max-old-space-size=4096 NX_DAEMON=false NX_NO_CLOUD=true
export PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 CYPRESS_INSTALL_BINARY=0
export GOCACHE="$HOME/zitadel-go-build" GOMODCACHE="$HOME/zitadel-go-modules"
test "$(node -p 'process.versions.node.split(".")[0]')" = 22
python3 .dagger/prepare.py
corepack pnpm install --frozen-lockfile --store-dir "$HOME/zitadel-pnpm-store"
case "$1" in
  test) corepack pnpm nx run @zitadel/api:test-unit ;;
  build) corepack pnpm nx run @zitadel/api:build; python3 .dagger/package.py ;;
  *) exit 2 ;;
esac
