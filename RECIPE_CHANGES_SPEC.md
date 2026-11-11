# Grocery Router — Changing recipes after the first load

Status: **proposed; awaiting owner approval** (#55). Implementation (#56, #57) starts only once
this is approved and merged.
Depends on: [`PRODUCT.md`](PRODUCT.md)

## 1. Problem

A household's recipes can be loaded exactly once. `ingest.Import` refuses a household that already
has recipes, and `bootstrap` skips a populated database. So a fix committed to
`corpus/recipes/*.md` never reaches a database that is already in use, and a second household or a
catalog recipe has no way in. Three things stand in the way:

- verified recipes are immutable in place (`enforce_recipe_lifecycle`,
  `protect_verified_recipe_child`);
- past plans hold restricting references to recipe rows and read recipe names and ingredient text
  from live rows, so editing a recipe would rewrite History, and replacing its ingredients is
  blocked; and
- the current plan's list must be rebuilt when one of its recipes changes.

## 2. Outcome

An operator runs one explicit command that makes a household's recipes match the approved Markdown
corpus. It previews the change, applies it only when asked, never runs at startup, and never
changes what History shows.

## 3. Decisions

### 3.1 Identity

A recipe is matched by `(household_id, key)`, a grocery item by `(household_id, key)`, and a store
section by `(household_id, key)`. A key is permanent. Renaming a key is treated as removing the old
recipe or item and adding a new one; the preview shows both, so a rename is never silent. Display
names may change freely.

### 3.2 History

**Snapshots** (#56). The rows a past plan keeps copy what History shows: each recipe occurrence
stores the recipe's key, name, image, yield and times, and each contribution stores the recipe
name, ingredient source text and preparation. The contribution's reference to the ingredient
becomes nullable with `ON DELETE SET NULL`. History reads only the snapshots; the current plan keeps
reading live recipes. Recipe versions are not introduced.

### 3.3 Change detection

Each recipe stores `content_digest`, a SHA-256 of a canonical encoding of the parsed document, so
whitespace and field order do not count as changes. A recipe whose digest is unchanged is skipped
and not written.

### 3.4 Recipes removed from the corpus

They are never deleted. The preview lists them as "only in the database" and they stay selectable.
Retiring a recipe is a later, separate decision.

### 3.5 Grocery items and sections

- Matched items update their name, reviewed section and shopping mode. New items and sections are
  added; a new section enters before Other.
- Manual rows keep their item reference and the name they were added with.
- A household's own placement of an item (D053) and its remembered names (D045) are never changed
  by sync; a household placement still wins over a new reviewed section.
- Items no recipe uses any more are left in place.
- **Aliases** (two keys becoming one, as recorded in `corpus/item-reviews.yaml`) are out of this
  phase. Merging rewrites references across plans and gets its own issue.

### 3.6 The current plan

If sync changes a recipe in the current plan, or an item on its list, the list is rebuilt in the
same transaction with today's rule (`carryStates`): checked, removed and plan-only quantities carry
over by item, manual rows are untouched, and rows for items no longer needed disappear. The
preview lists every current row whose amount changes, and flags any whose plan-only quantity now
hides a changed amount. The current plan's occurrence snapshots are refreshed. Past plans are never
rebuilt.

### 3.7 Entry point

- `grocery-router corpus-sync` (and `task corpus-sync`) runs the corpus audit, then the whole sync
  in one transaction. Without `--apply` it rolls back and prints the report, so the preview comes
  from the same code that writes. With `--apply` it commits and ends with `applied`.
- It works for an empty household too, producing the same rows as `corpus-ingest`, which stays as
  it is. Any invalid document aborts the run with nothing written.
- It never runs at startup. A deploy or rollback must not change recipe content, so the container
  entrypoint keeps running only `bootstrap`, which still loads an empty database and nothing else.
- **No HTTP edit API in this phase.** When one is specified, recipes changed in the app must carry
  a marker that sync respects: it skips them and reports them as conflicts rather than overwriting
  household edits.

### 3.8 Approval

A person still approves every recipe (PRODUCT.md §2). For the bootstrap corpus that review is the
pull request that changes the Markdown. As a mechanical guard, sync refuses a recipe whose digest
changed but whose `approved_on` date is not later than the one it was last verified with
(stored as `recipes.verified_at`). A changed recipe passes through draft
and reviewable to verified in one transaction, keeping its `id`, so existing references stay valid.

## 4. Delivery

1. **#56: History snapshots.** Migration and queries per §3.2. History responses are unchanged byte
   for byte after the recipe is edited or an ingredient deleted.
2. **#57: `corpus-sync`.** Digest column, `ingest.Sync` sharing per-recipe code with `Import`, an
   exported rebuild of the current plan's list, the command, and its report (§3.3 to §3.8).
3. **Later, separately:** item aliases, recipe retirement, catalog adoption
   ([`CATALOG_INGEST_SPEC.md`](CATALOG_INGEST_SPEC.md) §10), and any in-app editing.

## 5. Acceptance

- Syncing an unchanged corpus twice writes nothing the second time.
- Changing one quantity in a document updates that recipe in place (same `id`), rebuilds the
  current plan's list with checked state carried by the rule above, and leaves every other recipe
  and every past plan's History response unchanged.
- A removed document leaves its recipe in place and listed in the report.
- An invalid document, or a changed one without a later `approved_on`, leaves the database
  unchanged.
- The preview prints the same report as `--apply` and changes no table.
- Nothing in the startup path calls sync.
