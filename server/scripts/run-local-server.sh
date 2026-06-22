#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(cd "$SCRIPT_DIR/../.." && pwd)
SERVER="$ROOT/server"
GO="${GO:-go}"

export PATH="/usr/local/go/bin:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin:$PATH"

export DIARY_ADDR="${DIARY_ADDR:-127.0.0.1:18080}"
export DIARY_VAULT_DIR="${DIARY_VAULT_DIR:-$ROOT/vault}"
export DIARY_IMPORT_DIR="${DIARY_IMPORT_DIR:-$ROOT/imports}"
export DIARY_DATA_DIR="${DIARY_DATA_DIR:-$SERVER/tmp/data}"
export DIARY_API_TOKEN="${DIARY_API_TOKEN:-local-dev-token}"

mkdir -p "$DIARY_VAULT_DIR" "$DIARY_IMPORT_DIR" "$DIARY_DATA_DIR"

cd "$SERVER"
"$GO" build -o "$SERVER/tmp/diary-server-local" ./cmd/diary-server

exec "$SERVER/tmp/diary-server-local"
