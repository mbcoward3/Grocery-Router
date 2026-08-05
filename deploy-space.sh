#!/usr/bin/env bash
# Push this project to a Hugging Face Space.
#
#   HF_TOKEN=hf_... ./deploy-space.sh <user>/<space-name>
#
# Spaces read their config from YAML front matter at the top of README.md, and
# GitHub renders that front matter as an ugly table. So the README is generated
# here at push time rather than kept in a branch that drifts: the repo's README
# stays clean, and the Space gets what it needs.
#
# The Space runs in demo mode, because the Dockerfile sets PANTRY_DEMO=1 and
# app.py refuses to serve the real corpus on a public interface. Nothing a
# visitor does reaches these files.

set -euo pipefail

SPACE="${1:-}"
if [[ -z "$SPACE" || "$SPACE" != */* ]]; then
  echo "usage: HF_TOKEN=hf_... $0 <user>/<space-name>" >&2
  exit 64
fi
if [[ -z "${HF_TOKEN:-}" ]]; then
  echo "HF_TOKEN is not set. Create one at https://huggingface.co/settings/tokens" >&2
  echo "  — fine-grained, write access, scoped to this Space only." >&2
  exit 64
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Create the Space if it is not there yet. One HTTP call, so there is nothing to
# install and no CLI to keep current. A Space that already exists returns 409,
# which is a success for our purposes.
CREATE_CODE="$(curl -sS -o /tmp/hf-create.$$ -w '%{http_code}' \
  -X POST https://huggingface.co/api/repos/create \
  -H "Authorization: Bearer ${HF_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"${SPACE#*/}\",\"organization\":\"${SPACE%/*}\",\"type\":\"space\",\"sdk\":\"docker\"}")"
case "$CREATE_CODE" in
  200|201) echo "created space ${SPACE}" ;;
  409)     echo "space ${SPACE} already exists" ;;
  401|403) echo "token rejected (HTTP $CREATE_CODE). It needs write access to ${SPACE}." >&2
           cat /tmp/hf-create.$$ >&2; rm -f /tmp/hf-create.$$; exit 77 ;;
  *)       echo "could not create the space (HTTP $CREATE_CODE):" >&2
           cat /tmp/hf-create.$$ >&2; rm -f /tmp/hf-create.$$; exit 1 ;;
esac
rm -f /tmp/hf-create.$$
STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT

# Everything git tracks, and nothing it doesn't - no __pycache__, no .cache.
git -C "$ROOT" archive HEAD | tar -x -C "$STAGE"

# The source document is 20MB of scanned recipes and the app never opens it -
# it is committed for reproducibility of the onboarding run, not to be served.
# Hugging Face wants anything over 10MB in LFS, so it stays behind.
rm -rf "$STAGE/sources"

cat > "$STAGE/README.md" <<YAML
---
title: Pantry Router
emoji: 🧺
colorFrom: yellow
colorTo: red
sdk: docker
app_port: 7860
pinned: false
---

YAML
cat "$ROOT/README.md" >> "$STAGE/README.md"

cd "$STAGE"
git init -q -b main
git add -A
git -c user.email=deploy@local -c user.name=deploy commit -qm "Deploy $(git -C "$ROOT" rev-parse --short HEAD)"
git push -q --force "https://user:${HF_TOKEN}@huggingface.co/spaces/${SPACE}" main

echo "pushed → https://huggingface.co/spaces/${SPACE}"
echo "the build takes a minute or two; logs are on that page"
