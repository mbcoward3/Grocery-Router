# Shared recipe catalog

The shared catalog is separate from the PDF-controlled Coward household bootstrap corpus.

- `candidates/` contains generated, unapproved structured candidates and deterministic Markdown review views.
- `profiles/` contains compact, unapproved Jev semantic profiles; complete model responses remain in `.local-run/`.
- `recipes/` is reserved for individually human-approved recipe documents.
- `releases/` contains immutable publication manifests.

The checked-in `initial-review-2026-09-29` release is **reviewable only**. It may be published to the stable development database for visual inspection, but its recipes cannot be trialed, adopted, selected into a week, or treated as approved truth.

## Initial bounded workflow

```sh
# Fetch at most the explicitly manifested public pages.
task recipe-source-pilot

# Create compact packets and run isolated Pi workers (hard maximum: 25).
task catalog-agent-inputs -- --source-run .local-run/recipe-source-pilot/<run>
task catalog-amplify

# Build strict candidates. Read model/prompt/time from .local-run/catalog-agent/proposals/run.json.
task catalog-candidate-build -- \
  --source-run .local-run/recipe-source-pilot/<run> \
  --proposals .local-run/catalog-agent/proposals \
  --prompt-hash <sha256> \
  --run-at <RFC3339>

# Publish the non-adoptable review batch to development PostgreSQL.
task catalog-publish -- \
  --release catalog/releases/initial-review-2026-09-29.yaml \
  --database-url "$GROCERY_ROUTER_DATABASE_URL"
```

No publication command performs network or model calls. Raw source pages, pilot artifacts, Pi output, Jev responses, and credentials must never be committed.
