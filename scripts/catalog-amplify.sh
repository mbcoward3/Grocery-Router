#!/usr/bin/env bash
set -euo pipefail

root=${GROCERY_ROUTER_ROOT:-.}
inputs=${1:-"$root/.local-run/catalog-agent/inputs"}
output=${2:-"$root/.local-run/catalog-agent/proposals"}
model=${PI_CATALOG_MODEL:-openai-codex/gpt-5.5}
parallel=${PI_CATALOG_PARALLEL:-4}
prompt="$root/pilot/catalog-standardizer-prompt.md"

mapfile -d '' packets < <(find "$inputs" -maxdepth 1 -type f -name '*.json' -print0 | sort -z)
if ((${#packets[@]} == 0)); then
  echo "no catalog agent input packets in $inputs" >&2
  exit 1
fi
if ((${#packets[@]} > 25)); then
  echo "refusing ${#packets[@]} candidates; maximum is 25" >&2
  exit 1
fi
mkdir -p "$output"
export prompt output model
printf '%s\0' "${packets[@]}" | xargs -0 -n1 -P"$parallel" bash -c '
  packet=$1
  name=$(basename "$packet")
  echo "catalog Pi worker: $name" >&2
  pi --print --model "$model" --thinking low --no-tools --no-session \
    --no-context-files --no-extensions --no-skills --no-prompt-templates \
    "@$prompt" "@$packet" > "$output/$name"
  python3 -m json.tool "$output/$name" >/dev/null
' _

python3 - "$output/run.json" "$model" "$prompt" "${#packets[@]}" <<'PY'
import datetime, hashlib, json, pathlib, sys
path, model, prompt, count = sys.argv[1:]
prompt_bytes = pathlib.Path(prompt).read_bytes()
value = {
    "format_version": 1,
    "kind": "pi-proposed",
    "provider": model.split("/", 1)[0],
    "model": model.split("/", 1)[-1],
    "prompt_sha256": hashlib.sha256(prompt_bytes).hexdigest(),
    "run_at": datetime.datetime.now(datetime.timezone.utc).isoformat().replace("+00:00", "Z"),
    "candidates": int(count),
}
pathlib.Path(path).write_text(json.dumps(value, indent=2) + "\n")
PY

echo "wrote ${#packets[@]} untrusted Pi proposals to $output"
