# Grocery Router

Grocery Router v1 will turn this household's verified recipe corpus into an editable weekly
recipe pool and a deterministic grocery checklist.

## Status

**The previous prototype has been removed from the active tree.** The Go/SQLite corpus
foundation, current-week HTTP API, and initial React application shell are now in place. The
PDF-controlled ledger contains 25 recipes. Completed household re-review includes 24 in the
strict Markdown bootstrap and explicitly excludes Chicken and Dumplings.

## Start here

1. [`V1_SPEC.md`](V1_SPEC.md) — product, behavior, data model, architecture, and acceptance
2. [`UP_NEXT.md`](UP_NEXT.md) — explicitly deferred features
3. [`PRODUCT_DECISIONS.md`](PRODUCT_DECISIONS.md) — rationale and revisit conditions

The raw interview is archived at
[`archive/interviews/v1-scope-interview.md`](archive/interviews/v1-scope-interview.md). It is
historical and non-authoritative; routine work should use the documents above.

## Raw corpus

- `sources/Recipes.pdf` defines which family recipes belong in the initial corpus.
- `archive/trueup/recipes.csv` records the completed disposition of all 25 PDF recipes.
- `archive/trueup-evidence/inputs/` preserves the supporting typed text, transcripts, and saved
  URLs used during migration. It does not override the PDF's membership or an authoritative
  linked recipe.
- `archive/trueup-evidence/SOURCE_MANIFEST.sha256` records the preserved pre-true-up evidence.

## Development

Requires Go 1.25, Node.js 22.12 or newer, [uv](https://docs.astral.sh/uv/), and
[Task](https://taskfile.dev/). `Taskfile.yml` is the standard command surface; it pins
golangci-lint and SQLFluff, and the repository pins sqlc as a Go tool. The frontend uses strict
TypeScript, Vite, TanStack Router, and TanStack Query.

```sh
task dev                  # bootstrap and run the API and web application
task test                 # schema, ingestion, CLI, and inventory invariants
task lint                 # run the Go and SQL lint policies
task sql-lint             # lint migrations and sqlc query sources with SQLFluff
task ci                   # all checks enforced by GitHub Actions
task format               # apply Go formatting and import organization
task generate             # regenerate typed sqlc query code
task trueup-inventory     # validate all 25 PDF inventory rows
task corpus-render        # refresh checked human-readable recipe sections
task corpus-audit         # validate approved Markdown recipes
task corpus-ingest        # migrate and load an empty local SQLite DB
task migrate              # migrate without loading the corpus
task serve                # start the local JSON API
task web-install          # install locked frontend dependencies
task web-dev              # start Vite with an API development proxy
task web-ci               # generate, type-check, test, and build the frontend
```

Pass command flags after `--`, for example `task migrate -- --database /tmp/router.db`. The CLI
also accepts `GROCERY_ROUTER_DATABASE`, `GROCERY_ROUTER_ROOT`, `GROCERY_ROUTER_CORPUS`, and
`GROCERY_ROUTER_INVENTORY`; command-line flags take precedence. Run
`go run ./cmd/grocery-router --help` for command-specific help.

GitHub Actions runs `task ci` for pull requests and pushes to `main`. Run `task dev` from a fresh
clone to install missing frontend dependencies, ingest the corpus into the local database, and
start both servers. Open `http://localhost:5173`; another device on the same network can use the
Network URL printed by Vite. The Week screen is the first live vertical slice. Grocery and
recipe-detail endpoints and screens are next.
