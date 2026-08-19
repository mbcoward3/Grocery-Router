# Approved bootstrap corpus

Each file in `corpus/recipes/` is one individually approved recipe. Its strict YAML front
matter is the machine-readable bootstrap record. Its generated Markdown body provides normal
recipe details, ingredients, instructions, grocery preview, and review decisions.

The running application never reads these files. `corpus-ingest` validates the complete set,
checks it against `trueup/recipes.csv`, and inserts it transactionally into an empty migrated
SQLite database.

Rules:

- Only verified, individually approved recipes belong here.
- `format_version` is currently `1`.
- Unknown YAML fields fail parsing.
- `corpus-render` regenerates the readable body; `corpus-audit` rejects body/front-matter drift.
- Source ingredient text is retained exactly in `source_text`.
- Quantities use exact strings such as `"2"`, `"1/2"`, `"10.5"`, or `"1 1/2"`.
- Numeric quantities specify exactly one `unit` or `package`.
- Unit keys must exist in the exact-unit migration.
- Grocery item key, name, section, and shopping mode must agree across every recipe.
- Every backfill, rewrite, or resolved conflict appears in `review` with `approved: true`.
- Agent-selected calls and household outcomes are retained in each recipe's approved review
  fields.
- The ledger status and approval date must match the file.

Validate and load with:

```sh
go run ./cmd/grocery-router corpus-render
go run ./cmd/grocery-router corpus-audit
go run ./cmd/grocery-router corpus-ingest
```
