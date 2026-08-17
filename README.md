# Grocery Router

Grocery Router v1 will turn this household's verified recipe corpus into an editable weekly
recipe pool and a deterministic grocery checklist.

## Status

**The previous prototype has been removed from the active tree. v1 implementation has not
started.** The current repository contains the raw corpus evidence and the settled context for
the clean Go/React/SQLite rebuild.

## Start here

1. [`V1_SPEC.md`](V1_SPEC.md) — product, behavior, data model, architecture, and acceptance
2. [`TRUE_UP_PLAN.md`](TRUE_UP_PLAN.md) — recipe inventory, review, migration, and audit
3. [`REPO_CLEANUP_PLAN.md`](REPO_CLEANUP_PLAN.md) — preservation and consolidation record
4. [`UP_NEXT.md`](UP_NEXT.md) — explicitly deferred features
5. [`PRODUCT_DECISIONS.md`](PRODUCT_DECISIONS.md) — rationale and revisit conditions

The raw interview is archived at
[`archive/interviews/v1-scope-interview.md`](archive/interviews/v1-scope-interview.md). It is
historical and non-authoritative; routine work should use the documents above.

## Raw corpus

- `sources/Recipes.pdf` defines which family recipes belong in the initial corpus.
- `sources/inputs/` contains supporting typed text, transcripts, and saved URLs. It is useful
  migration evidence but does not override the PDF's membership or an authoritative linked
  recipe.
- `SOURCE_MANIFEST.sha256` records the preserved source files before true-up.

There is currently no application to run. The next work is the clean scaffold and true-up
pilot described by the plans above.
