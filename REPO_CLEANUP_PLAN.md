# Grocery Router v1 — Repository Cleanup Plan

Status: **required before implementation consolidation**  
Product authority: [`V1_SPEC.md`](V1_SPEC.md)  
Corpus migration: [`TRUE_UP_PLAN.md`](TRUE_UP_PLAN.md)

This document governs preservation, deletion, branch consolidation, and creation of the clean
Go/React/SQLite implementation line. It is intentionally separate from the product spec so
historical Git mechanics do not become product context.

## 1. Goal

The existing Git history is retained, but the active tree is rebuilt around this specification.
Cleanup is a staged migration with a rollback point, not an indiscriminate deletion.

## 2. Safety and branch consolidation

Before deleting anything:

1. Fetch the latest remote state and record the commit IDs of `main`, `v1-rebuild`, and all
   open work branches.
2. Preserve the current dirty `v1-rebuild` work on an explicitly archival branch/commit so
   no uncommitted file is accidentally lost, even though those specifications are not
   carried into v1.
3. Establish one clean v1 rebuild branch from the latest shared history.
4. Commit this specification as the only active product authority.
5. Do not merge the open Compose/deployment PR into v1; close or mark it superseded after
   preserving its branch in Git history.
6. Make the rebuild branch the sole integration line. No feature work continues on the
   Python, deployment, or prior specification branches.

The archive is a recovery mechanism, not active context. Archived files must not remain in
the working tree merely because they once cost effort.

## 3. Evidence inventory

Before removing legacy data files, classify every top-level and source artifact as:

- **canonical raw evidence** — preserve actively;
- **supporting migration evidence** — preserve under `sources/` only if it materially helps
  true-up;
- **derived and untrusted** — available in Git history but removed from the active tree; or
- **obsolete implementation/context** — remove from the active tree.

The expected treatment is:

| Material | Treatment |
|---|---|
| `sources/Recipes.pdf` | Preserve; canonical membership evidence |
| Useful typed inputs, transcripts, and URL index | Preserve as supporting migration evidence |
| Existing `recipes/*.md`, `corpus.md`, `items.md`, candidates/profile/week files | Treat as untrusted derived hints; consult from history or temporary migration workspace, not as v1 truth |
| Existing Python application and tests | Remove after archival; behavior is not a v1 contract |
| Existing deployment, container, Talos, Flux, Cockroach/Postgres material | Remove |
| `.scratch/spec`, prior context/glossary, obsolete architecture docs | Remove after this specification is safely committed |
| Prototype screenshots and stale generated artifacts | Remove unless deliberately retained outside active project context |
| `PRODUCT_DECISIONS.md` | Preserve as current rationale |
| `archive/interviews/v1-scope-interview.md` | Preserve as non-authoritative historical evidence; never routine agent context |

A checksum/manifest of preserved raw evidence should be created before true-up so accidental
source changes are visible.

## 4. Clean active tree

The first cleanup commit should leave only:

- canonical/supporting source evidence;
- `V1_SPEC.md`;
- a minimal current `README.md`;
- concise agent instructions;
- repository configuration still relevant to the rebuild; and
- the new project scaffold as it is introduced.

It should remove:

- the disposable Python application;
- deployment and container infrastructure;
- superseded architecture documents;
- contradictory issue maps and specifications;
- stale run commands;
- derived recipe Markdown from runtime paths;
- generated weeks and prototype state from active runtime assumptions; and
- documentation for abandoned product names, storage systems, planners, and hosting paths.

Deletion should be reviewable as a dedicated cleanup commit, separate from the first Go
implementation commit.

## 5. Rebuild scaffold

After cleanup, establish:

- Go module and package boundaries;
- Goose migration directory;
- sqlc configuration, queries, and generated-code boundary;
- local SQLite configuration with foreign keys enabled;
- React/TypeScript frontend;
- test commands;
- corpus ingestion/validation command structure;
- raw-source and reusable-fixture locations; and
- one documented local development command or concise command sequence.

Do not add Docker, cloud manifests, authentication scaffolding, ADK dependencies, or generic
framework abstractions during setup.

## 6. Documentation end state

The final active documentation is intentionally small:

- `README.md` — setup, run, test, true-up, and audit commands;
- `V1_SPEC.md` — product and implementation contract;
- `TRUE_UP_PLAN.md` and `REPO_CLEANUP_PLAN.md` — focused execution plans;
- `UP_NEXT.md` — deferred scope;
- `PRODUCT_DECISIONS.md` — current rationale and revisit conditions;
- `archive/interviews/v1-scope-interview.md` — historical, non-authoritative interview record;
- schema/architecture notes only where migrations and code are insufficient; and
- `AGENTS.md` — current commands, invariants, and hard-won ingestion rules.

A statement useful only for historical explanation belongs in Git history, not `AGENTS.md`.
No active document may point to removed files or present a deferred feature as implemented.

## 7. Branch closeout

After the clean application line passes its initial migration and scaffold tests:

- merge it through the chosen mainline review path;
- update the repository default branch if necessary;
- close superseded PRs;
- delete obsolete remote feature branches only after confirming their commits remain
  reachable through merged history or an intentional archive reference; and
- verify a fresh clone contains no contradictory setup or product direction.

---

## 8. Cleanup completion checklist

Cleanup is complete when a fresh clone has one active product direction, the raw corpus is
preserved, obsolete work is reachable only through history/archive references, current run
commands are accurate, and no open PR or active branch can be mistaken for the v1 integration
line.
