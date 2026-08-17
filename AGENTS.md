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
v1 yet. Do not recover the Python prototype or prior deployment infrastructure as a base.

The implementation direction is Go, SQLite, Goose, sqlc, and a React/TypeScript frontend.
Runtime AI, Docker, hosting, authentication, pantry inference, scaling, and recipe discovery
are out of v1.

## Corpus rules

- `sources/Recipes.pdf` controls initial corpus membership.
- Linked websites control linked recipe content at true-up time.
- `sources/inputs/` is supporting evidence only.
- A source may be incomplete; a selectable recipe may not be.
- Every recipe is reviewed individually before verification.
- Ingredient-to-grocery mapping happens during ingestion, never week generation.
- No ingredient contribution may be silently dropped.

## Working rule

When a desirable capability is outside `V1_SPEC.md`, record or refine it in `UP_NEXT.md`; do
not add speculative schema, controls, dependencies, or placeholder architecture to v1.
