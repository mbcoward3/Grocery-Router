# Grocery Router — Recipe library: shared catalog recipes and household edits

Status: **proposed; awaiting owner approval** (#55, revision 2). Direction approved by the owner on
2026-11-12: catalog fixes flow to households unless a household edits the recipe, the family's 24
starter recipes stay household recipes, and household edits apply without a second approval.
Depends on: [`PRODUCT.md`](PRODUCT.md), [`CATALOG_INGEST_SPEC.md`](CATALOG_INGEST_SPEC.md)

## 1. Problem

Grocery Router is moving to many households sharing one large catalog that suggestion tooling
draws from. Today a household's recipes are loaded once from Markdown files committed to the
repository, cannot be changed afterwards, and past plans read live recipe rows. That pattern does
not scale: a catalog fix would need a copy per household, and every change would need a developer.

## 2. Outcome

- A household's **recipe library** holds two kinds of entries: **catalog recipes it follows**
  (shared, one copy for everyone) and **its own recipes** (household-owned).
- Fixes published to a followed catalog recipe reach every household that follows it.
- A household member can edit any recipe in its library in the app. Editing a followed catalog
  recipe first makes the household its own copy, which no longer receives catalog fixes.
- Past plans never change when a recipe does.
- Committed Markdown is no longer how recipes change.

## 3. Decisions

### 3.1 The library

- Household recipes (the existing `recipes` table, household-owned) and `catalog_recipes`
  (shared, published by the pipeline) stay separate tables.
- A household's library is its own recipes plus `household_catalog_follows (household_id,
  catalog_key)`. Following a catalog recipe copies nothing.
- What a followed recipe contains is the catalog's **active publication** for that key
  (CATALOG_INGEST_SPEC §6.7). A new publication, correction, withdrawal or rollback reaches
  followers at once. A withdrawn recipe stays in the library as unavailable: it cannot be added to
  a plan, and the household can make it its own from the last version it saw.
- A household recipe made from a catalog recipe records `forked_from_catalog_key` and the version
  it came from. It never changes when the catalog does.
- Selectable recipes for Generate, swap and add are: the household's verified recipes plus its
  available followed catalog recipes. Catalog trust stays release provenance, never `verified`
  (D040), and does not need a household approval to be selectable.

### 3.2 Grocery item identity

- **Catalog grocery items are the shared identity.** Catalog recipes reference them directly.
- Each household grocery item may map to one catalog item (`grocery_items.catalog_item_key`,
  nullable). Two requirements combine on a list only when they resolve to the same catalog item, or
  to the same unmapped household item.
- The family's existing items are mapped once by an operator command that proposes matches by key
  and catalog alias, prints them, and writes only with `--apply`. Unmatched items stay unmapped and
  keep working; they simply never combine with catalog items.
- Household placements (D053) and remembered names (D045) attach to the resolved item, so a
  household's "tortillas are in Bakery" applies to both its own and catalog recipes.

### 3.3 Editing in the app

- A **Recipe edit** screen edits name, image, times, yield, ingredient sections and lines
  (source text, item, quantity, unit, package, preparation, optional), and steps.
- Saving runs the same validation that verification uses today and applies at once in one
  transaction. A member's save is the household's approval: the recipe stays `verified` with a new
  `verified_at` and `verified_by`. An invalid edit is rejected with field errors and nothing changes.
- Editing a followed catalog recipe first creates the household copy (§3.1), replaces the follow
  with it, and points the current plan's occurrences at the copy.
- Choosing an item for a line searches the household's items and the catalog's items. Picking a
  catalog item the household has no row for creates the household item mapped to it.
- Members can also unfollow a catalog recipe and archive their own recipes. Neither deletes
  anything that a plan references; archived and unfollowed recipes stop being selectable.

### 3.4 History

Snapshots, as #56 specifies: each recipe occurrence stores what History shows about its recipe,
and each contribution stores the recipe name, ingredient source text and preparation. History reads
only the snapshots. This applies to both kinds of recipe, because both can now change.

### 3.5 The current plan

When a recipe in the current plan changes (a household edit, or a new catalog publication of a
followed recipe), its list is rebuilt with today's rule (`carryStates`): checked, removed and
plan-only quantities carry over by item, manual rows are untouched, and rows no longer needed go.
A catalog publication rebuilds each affected current plan when it is next read, as aggregation
versions already do, rather than writing to every household at publication time.

### 3.6 Plan occurrences

`week_recipes` references exactly one of `recipe_id` (household) or `catalog_key` (followed),
enforced by a check constraint. Requirements for a plan are read from both sources and aggregated
by the identity in §3.2.

### 3.7 The starter recipes and the Markdown files

- The family's 24 recipes stay household recipes of the `Coward` household, editable like any
  other.
- `corpus/recipes/*.md` stays only as the seed `bootstrap` loads into an **empty** database (fresh
  development and test databases). Nothing syncs it, the running app never reads it, and editing a
  file never changes an existing household. When the catalog can seed a new household, the files
  move to `archive/`.
- There is no `corpus-sync` command.

### 3.8 Not in this phase

Suggestions and the rules for which catalog recipes a household is offered (D043), new-household
signup, merging duplicate household items (aliases in `corpus/item-reviews.yaml`), sharing a
household's own recipe with others, and edit history beyond History snapshots.

## 4. Delivery

1. **#56: History snapshots.** As written. Prerequisite for everything below.
2. **In-app recipe editing** for household recipes (§3.3 without the catalog parts), with an API
   endpoint, validation errors, and a current-plan rebuild. Replaces #57.
3. **Item mapping** (§3.2): `catalog_item_key`, the proposing command, and aggregation by resolved
   identity. Needs catalog slice 3 items to exist.
4. **Following catalog recipes** (§3.1, §3.6): follows, selectability, occurrences of catalog
   recipes, rebuild on publication, withdrawal handling, and copy-on-edit. Needs catalog slice 4
   publications.

Each step is its own issue and leaves the app working.

## 5. Acceptance

- Editing one quantity of a household recipe in the app updates it in place, rebuilds the current
  plan's list with checked state carried, and leaves every past plan's History response unchanged.
- An invalid edit changes nothing and names the field.
- A followed catalog recipe picks up a new publication without any household write, and a past plan
  that used it is unchanged.
- Editing a followed recipe creates exactly one household copy, stops it following catalog fixes,
  and keeps the current plan's list correct.
- A withdrawn catalog recipe cannot be added to a plan, and plans that used it still show it.
- A household-mapped item and the catalog item it maps to combine into one row; an unmapped item
  never silently merges with anything.
- Nothing in the startup path changes an existing household's recipes.
