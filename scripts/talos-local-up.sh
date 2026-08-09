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

for tool in docker talosctl kubectl flux; do
  command -v "$tool" >/dev/null || { echo "missing prerequisite: $tool" >&2; exit 1; }
done

talos_client=$(talosctl version --client --short 2>&1 | grep -Eo 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1 || true)
kubectl_client=$(kubectl version --client -o json 2>/dev/null \
  | python3 -c 'import json,sys; print(json.load(sys.stdin)["clientVersion"]["gitVersion"])' \
  2>/dev/null || true)
flux_client=$(flux --version 2>&1 | grep -Eo '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || true)
if [[ "$talos_client" != "$TALOS_VERSION" ]]; then
  echo "talosctl $TALOS_VERSION is required; found ${talos_client:-unknown}" >&2
  exit 1
fi
if [[ "$kubectl_client" != "$KUBECTL_VERSION" ]]; then
  echo "kubectl $KUBECTL_VERSION is required; found ${kubectl_client:-unknown}" >&2
  exit 1
fi
if [[ "v$flux_client" != "$FLUX_VERSION" ]]; then
  echo "flux $FLUX_VERSION is required; found ${flux_client:-unknown}" >&2
  exit 1
fi

docker info >/dev/null 2>&1 || {
  echo "Docker is installed but its selected daemon is not reachable by this user." >&2
  exit 1
}
docker_server=$(docker version --format '{{.Server.Version}}')
if [[ "$(printf '%s\n' 18.03 "$docker_server" | sort -V | head -1)" != "18.03" ]]; then
  echo "Docker server 18.03+ is required; found $docker_server" >&2
  exit 1
fi

mkdir -p "$STATE_ROOT"

if [[ -d "$CLUSTER_STATE/$TALOS_CLUSTER_NAME" ]]; then
  if [[ ! -f "$OWNER_MARKER" ]]; then
    echo "Refusing to adopt unmarked cluster state for $TALOS_CLUSTER_NAME." >&2
    exit 1
  fi
  echo "Reusing only the task-managed cluster state at $CLUSTER_STATE/$TALOS_CLUSTER_NAME"
  "$REPO_ROOT/scripts/talos-local-validate.sh"
  exit 0
fi

# Refuse to adopt or replace containers/networks that this state directory did not create.
if docker ps -a --format '{{.Names}}' | grep -Eq "^${TALOS_CLUSTER_NAME}([.-]|$)"; then
  echo "Refusing to touch a pre-existing $TALOS_CLUSTER_NAME container without task state." >&2
  exit 1
fi
if docker network ls --format '{{.Name}}' | grep -Fxq "$TALOS_CLUSTER_NAME"; then
  echo "Refusing to touch a pre-existing $TALOS_CLUSTER_NAME network without task state." >&2
  exit 1
fi

echo "Creating Docker-provisioned Talos cluster $TALOS_CLUSTER_NAME ($TALOS_CLUSTER_CIDR)"
printf '%s\n' "$TALOS_CLUSTER_NAME" > "$OWNER_MARKER"
talosctl cluster create docker \
  --name "$TALOS_CLUSTER_NAME" \
  --cidr "$TALOS_CLUSTER_CIDR" \
  --state "$CLUSTER_STATE" \
  --talosconfig "$TALOSCONFIG" \
  --controlplanes 1 \
  --workers 1 \
  --kubernetes-version "${KUBERNETES_VERSION#v}" \
  --exposed-ports "${GROCERY_ROUTER_NODEPORT}:${GROCERY_ROUTER_NODEPORT}/tcp" \
  --wait-timeout 15m

"$REPO_ROOT/scripts/talos-local-validate.sh"
