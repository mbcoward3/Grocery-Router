#!/usr/bin/env bash
set -euo pipefail

root=${1:?Preview root is required}
cat > "$root/kustomization.yaml" <<'EOF'
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - namespace.yaml
  - policy.yaml
EOF

find "$root" -mindepth 1 -maxdepth 1 -type d -name 'pr-*' -printf '%f\n' \
  | sort -V \
  | while read -r directory; do printf '  - %s/\n' "$directory"; done \
  >> "$root/kustomization.yaml"
