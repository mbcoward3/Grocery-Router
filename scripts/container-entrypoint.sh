#!/bin/sh
set -eu

if [ ! -s "${GROCERY_ROUTER_DATABASE}" ]; then
  grocery-router corpus-ingest
else
  grocery-router migrate
fi

exec grocery-router serve "$@"
