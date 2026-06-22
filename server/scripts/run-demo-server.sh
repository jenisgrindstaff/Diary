#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(cd "$SCRIPT_DIR/../.." && pwd)

DEMO_WORKDIR="$ROOT/server/tmp/demo-vault"
DEMO_DATA_DIR="$ROOT/server/tmp/demo-data"
DEMO_IMPORT_DIR="$ROOT/server/tmp/demo-imports"

rm -rf "$DEMO_WORKDIR" "$DEMO_DATA_DIR" "$DEMO_IMPORT_DIR"
mkdir -p "$ROOT/server/tmp"
cp -R "$ROOT/demo-vault" "$DEMO_WORKDIR"

export DIARY_ADDR="${DIARY_ADDR:-127.0.0.1:18080}"
export DIARY_VAULT_DIR="$DEMO_WORKDIR"
export DIARY_IMPORT_DIR="$DEMO_IMPORT_DIR"
export DIARY_DATA_DIR="$DEMO_DATA_DIR"
export DIARY_API_TOKEN="${DIARY_API_TOKEN:-local-dev-token}"

exec "$ROOT/server/scripts/run-local-server.sh"
