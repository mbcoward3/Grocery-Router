#!/bin/sh
set -eu

grocery-router bootstrap
if [ -n "${GROCERY_ROUTER_CATALOG_RELEASE:-}" ]; then
  grocery-router catalog-publish --release "$GROCERY_ROUTER_CATALOG_RELEASE"
fi
exec grocery-router serve "$@"
