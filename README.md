# Grocery Router

Grocery Router v1 will turn this household's verified recipe corpus into an editable weekly
recipe pool and a deterministic grocery checklist.

## Status

**The previous prototype has been removed from the active tree.** The Go/SQLite corpus
foundation is now in place; the web application has not been built. The PDF-controlled ledger
contains 25 recipes. Seven have been individually approved in the strict Markdown bootstrap
format; 18 remain to true up.

## Start here

1. [`V1_SPEC.md`](V1_SPEC.md) — product, behavior, data model, architecture, and acceptance
2. [`TRUE_UP_PLAN.md`](TRUE_UP_PLAN.md) — recipe inventory, review, migration, and audit
3. [`UP_NEXT.md`](UP_NEXT.md) — explicitly deferred features
4. [`PRODUCT_DECISIONS.md`](PRODUCT_DECISIONS.md) — rationale and revisit conditions

The raw interview is archived at
[`archive/interviews/v1-scope-interview.md`](archive/interviews/v1-scope-interview.md). It is
historical and non-authoritative; routine work should use the documents above.

## Raw corpus

- `sources/Recipes.pdf` defines which family recipes belong in the initial corpus.
- `sources/inputs/` contains supporting typed text, transcripts, and saved URLs. It is useful
  migration evidence but does not override the PDF's membership or an authoritative linked
  recipe.
- `SOURCE_MANIFEST.sha256` records the preserved source files before true-up.

## Development

Requires Go 1.25. The repository pins sqlc as a Go tool.

```sh
go test ./...                                      # schema, ingestion, and inventory invariants
go tool sqlc generate                              # regenerate typed query code
go run ./cmd/grocery-router trueup-inventory       # validate all 25 PDF inventory rows
go run ./cmd/grocery-router corpus-render          # refresh checked human-readable sections
go run ./cmd/grocery-router corpus-audit           # validate approved Markdown recipes
go run ./cmd/grocery-router corpus-ingest          # migrate and load an empty local SQLite DB
go run ./cmd/grocery-router migrate                # migrate without loading the corpus
```

There is currently no web application to run. The next work is the difficult-recipe schema
pilot and one-at-a-time true-up described in `TRUE_UP_PLAN.md`.
