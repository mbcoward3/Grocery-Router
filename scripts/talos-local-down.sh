#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
# shellcheck source=platform-versions.env
source "$REPO_ROOT/scripts/platform-versions.env"
STATE_ROOT="$REPO_ROOT/.local/talos"
CLUSTER_STATE="$STATE_ROOT/state"
OWNER_MARKER="$STATE_ROOT/owned-$TALOS_CLUSTER_NAME"
export TALOSCONFIG="$STATE_ROOT/talosconfig"
export KUBECONFIG="$STATE_ROOT/kubeconfig"

if [[ "${1:-}" != "--confirm" || "${2:-}" != "$TALOS_CLUSTER_NAME" || $# -ne 2 ]]; then
  echo "Destructive cleanup is opt-in. Run:" >&2
  echo "  $0 --confirm $TALOS_CLUSTER_NAME" >&2
  exit 2
fi

if [[ ! -d "$CLUSTER_STATE/$TALOS_CLUSTER_NAME" || ! -f "$OWNER_MARKER" ]]; then
  echo "Refusing cleanup: no marked task-managed state for $TALOS_CLUSTER_NAME" >&2
  exit 1
fi
if [[ "$(cat "$OWNER_MARKER")" != "$TALOS_CLUSTER_NAME" ]]; then
  echo "Refusing cleanup: ownership marker does not name $TALOS_CLUSTER_NAME" >&2
  exit 1
fi
talos_client=$(talosctl version --client --short 2>&1 \
  | grep -Eo 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1 || true)
if [[ "$talos_client" != "$TALOS_VERSION" ]]; then
  echo "Refusing cleanup with talosctl ${talos_client:-unknown}; $TALOS_VERSION is required." >&2
  exit 1
fi

# talosctl scopes deletion to the exact name and state root. Never call docker rm/prune here.
talosctl cluster destroy \
  --name "$TALOS_CLUSTER_NAME" \
  --state "$CLUSTER_STATE" \
  --talosconfig "$TALOSCONFIG"
rm -f -- "$OWNER_MARKER"
echo "Destroyed only Talos cluster $TALOS_CLUSTER_NAME; local config files were retained."
