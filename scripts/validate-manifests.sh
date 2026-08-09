#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
command -v kubectl >/dev/null || { echo "kubectl is required" >&2; exit 1; }

for target in deploy/base deploy/overlays/local deploy/overlays/production clusters/local clusters/production; do
  output=$(mktemp)
  kubectl kustomize "$REPO_ROOT/$target" > "$output"
  test -s "$output"
  python3 - "$output" "$target" <<'PY'
import sys
from pathlib import Path
text = Path(sys.argv[1]).read_text()
if "kind: Deployment" in text and "kind: Service" not in text:
    raise SystemExit(f"{sys.argv[2]}: Deployment rendered without Service")
if not any(marker in text for marker in (
    "grocery-router:unconfigured",
    "sha-0000000000000000000000000000000000000000",
)):
    raise SystemExit(f"{sys.argv[2]}: expected auditable image placeholder is missing")
PY
  rm -f "$output"
done

echo "All reusable resources and cluster overlays render successfully."
