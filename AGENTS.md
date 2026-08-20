# Grocery Router agent context

## Authority

Read in this order:

1. `V1_SPEC.md`
2. `TRUE_UP_PLAN.md`
3. `UP_NEXT.md`
4. `PRODUCT_DECISIONS.md` for rationale

`archive/interviews/v1-scope-interview.md` is historical and non-authoritative. Do not load it
as routine context; consult it only to recover nuance not settled by the current documents.

## Current state

The old application and contradictory context were intentionally removed. There is no runnable
web v1 yet. Do not recover the Python prototype or prior deployment infrastructure as a base.

The corpus foundation is Go, SQLite, Goose, and sqlc. `trueup/recipes.csv` inventories all 25
PDF recipes. Completed household re-review includes 24 approved bootstrap recipes under
`corpus/recipes/` and explicitly excludes Chicken and Dumplings. The frontend direction is
React/TypeScript.
Runtime AI, Docker, hosting, authentication, pantry inference, scaling, and recipe discovery
are out of v1.

## Corpus rules

- `sources/Recipes.pdf` controls initial corpus membership.
- Linked websites control linked recipe content at true-up time.
- `archive/trueup-evidence/inputs/` is archived supporting evidence only.
- A source may be incomplete; a selectable recipe may not be.
- Every recipe is reviewed individually before verification.
- Approved recipes remain as strict Markdown bootstrap files and are ingested one-way.
- SQLite is runtime truth; application code never reads or synchronizes recipe Markdown.
- Ingredient-to-grocery mapping happens during ingestion, never week generation.
- No ingredient contribution may be silently dropped.

## Commands

```sh
go test ./...
go tool sqlc generate
go run ./cmd/grocery-router trueup-inventory
go run ./cmd/grocery-router corpus-render
go run ./cmd/grocery-router corpus-audit
go run ./cmd/grocery-router corpus-ingest
go run ./cmd/grocery-router migrate
```

`internal/database/migrations/` is the schema authority. Generated files in `internal/store/`
must match `internal/store/queries/` and `sqlc.yaml`. SQLite foreign keys must be enabled on
every connection; use `internal/database.Open`.

## Working rule

When a desirable capability is outside `V1_SPEC.md`, record or refine it in `UP_NEXT.md`; do
not add speculative schema, controls, dependencies, or placeholder architecture to v1.
