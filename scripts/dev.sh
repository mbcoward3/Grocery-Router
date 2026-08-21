#!/usr/bin/env bash
set -euo pipefail

if [[ ! -d web/node_modules ]]; then
  npm ci --prefix web
fi

recipe_count="$(sqlite3 data/grocery-router.db 'select count(*) from recipes;' 2>/dev/null || true)"
if [[ -z "$recipe_count" || "$recipe_count" == "0" ]]; then
  go run ./cmd/grocery-router corpus-ingest
fi

mkdir -p bin
go build -o bin/grocery-router ./cmd/grocery-router
bin/grocery-router serve &
api_pid=$!

cleanup() {
  kill "$api_pid" 2>/dev/null || true
  wait "$api_pid" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

npm run dev --prefix web
