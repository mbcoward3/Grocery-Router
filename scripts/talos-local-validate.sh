#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
# shellcheck source=platform-versions.env
source "$REPO_ROOT/scripts/platform-versions.env"
STATE_ROOT="$REPO_ROOT/.local/talos"
export TALOSCONFIG="$STATE_ROOT/talosconfig"
export KUBECONFIG="$STATE_ROOT/kubeconfig"
CONTEXT="admin@$TALOS_CLUSTER_NAME"

for tool in talosctl kubectl flux; do
  command -v "$tool" >/dev/null || { echo "missing prerequisite: $tool" >&2; exit 1; }
done

kubectl --context "$CONTEXT" get nodes -o wide
talosctl --context "$TALOS_CLUSTER_NAME" version
flux check --pre

rendered=$(mktemp)
trap 'rm -f "$rendered"' EXIT
kubectl kustomize "$REPO_ROOT/clusters/local" > "$rendered"
test -s "$rendered"
kubectl --context "$CONTEXT" apply --dry-run=client --validate=true -f "$rendered"
echo "Talos API, Kubernetes API, Flux prerequisites, and local manifests validated."
