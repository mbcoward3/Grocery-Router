#!/usr/bin/env bash
set -euo pipefail

repo=${1:?GitOps checkout is required}
root="$repo/apps/grocery-router/previews"
now=$(date -u +%s)

for marker in "$root"/pr-*/.expires-at; do
  [[ -e "$marker" ]] || continue
  expires=$(date -u -d "$(<"$marker")" +%s)
  if (( expires <= now )); then
    echo "Removing expired preview $(basename "$(dirname "$marker")")"
    rm -rf "$(dirname "$marker")"
  fi
done

"$(dirname "$0")/render-preview-kustomization.sh" "$root"
