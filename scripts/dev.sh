#!/usr/bin/env bash
set -euo pipefail

if [[ ! -d web/node_modules ]]; then
  npm ci --prefix web
fi

if [[ -z "${GROCERY_ROUTER_DATABASE_URL:-}" ]]; then
  docker compose up -d --wait postgres
  export GROCERY_ROUTER_DATABASE_URL='postgres://grocery_router:grocery_router@localhost:5432/grocery_router?sslmode=disable'
fi

go run ./cmd/grocery-router bootstrap

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
