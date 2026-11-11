# Grocery Router — Household Substitution Groups

Status: **proposed 2026-11-09; not authorized.** Nothing here is built until the owner approves it.  
Depends on: [`PRODUCT.md`](PRODUCT.md), [`CATALOG_INGEST_SPEC.md`](CATALOG_INGEST_SPEC.md)  
Rationale: [`PRODUCT_DECISIONS.md`](PRODUCT_DECISIONS.md) D047, D048

## 1. Problem

Recipes name specific products: one wants a yellow onion, another a white onion; one wants
shredded cheddar, another a Mexican blend. Many households buy one product to cover both, and
others do not. Today the list always keeps them apart, so a household that substitutes freely
sees near-duplicate rows and has to combine them in their head while shopping.

This is a preference, not a fact about the recipes, and it differs between households. It must
be settled without anyone reviewing the pipeline's output item by item.

## 2. Two layers

| Layer | Question | Who decides | Where it lives |
| --- | --- | --- | --- |
| Identity | Is this the same product? (`Kosher Salt` / `Salt`) | The pipeline: rules, aliases, overrides | Vocabulary snapshot (one item) |
| Substitution | Would this household buy one to cover both? (`Yellow Onion` / `White Onion`) | The pipeline proposes groups; each household chooses | Snapshot groups + household choices |

Identity is shared and strict. Substitution is per household and explicit.

## 3. Determinism contract

1. **Recipes never change.** A recipe keeps the item it names. Combining groups list rows; it
   does not remap ingredients, so V1's rule that mapping happens only during ingestion holds.
2. **Groups are frozen data.** Group membership is part of the vocabulary snapshot (catalog) or
   `corpus/item-reviews.yaml` (bootstrap corpus). An AI may classify a pair once; the result is
   stored with its method and reason and reused. It changes only through a new snapshot, which
   is qualified and logged.
3. **The list is a pure function** of the week's recipes, the snapshot, and the household's
   recorded choices. No AI runs while building or shopping a list.
4. **Errors are asymmetric by design.** `alias` requires a rule or override because a wrong
   alias silently merges products. `group` may come from AI because a wrong group only produces
   one unnecessary question, and nothing combines without the household's answer.
5. **Nothing is inferred silently.** A household's choice comes only from an explicit answer.
   Unanswered groups stay separate.

## 4. Behavior

### 4.1 Groups

- A group has a key, a display name (`Onions`, `Shredded cheese`), and two or more member items.
- Every member must share one store section and one shopping mode; classification rejects a
  group that does not (the pair becomes `distinct`).
- An item belongs to at most one group. Groups are flat: no nesting.

### 4.2 Household choice

- Each household holds, per group, `combine`, `separate`, or no answer. No answer means separate.
- **Asking.** When the current week's list contains two or more members of a group the household
  has not answered, the list shows one quiet card below the add bar:
  *"Yellow onion and white onion: combine on your list?"* `Combine` / `Keep separate`.
  One card at a time, the same placement as the section picker. Dismissing hides it for this
  week only.
- **Changing.** A household can review and change answers in one place. A change rebuilds the
  current week only; history is never rebuilt.

### 4.3 Combined rows

- Requirements of combined members aggregate exactly as one item: compatible units add
  (`2 each + 1 each = 3 each`); incompatible ones stay separate quantities on the row, and an
  unspecified requirement adds `+ more`, as in [`PRODUCT.md`](PRODUCT.md) §5.
- The row shows the group's display name. Details list each recipe's original wording and the
  member item it named.
- Checked, removed, and adjusted state carry across combining and splitting with the V1
  rebuild rule: checked or removed only if every source row was; an adjustment only from a
  single source row.
- Manual lines are never combined.

## 5. Data

- `grocery_item_groups (household_id, group_key, grocery_item_id)` and `item_groups
  (household_id, group_key, display_name)`, imported with the corpus or a catalog publication.
- `household_group_choices (household_id, group_key, choice, decided_at)`.
- Migration impact: additive tables only. Existing lists are unchanged until the household
  answers a question, which rebuilds the current week.

## 6. Failure behavior

- A group whose members no longer share a section or mode after a vocabulary change is ignored
  (members stay separate) and reported by `corpus-audit` or `catalog-audit`.
- A choice for a group that no longer exists is kept but has no effect.
- If combining would exceed exact 64-bit arithmetic, the members stay separate and the
  rebuild still succeeds.

## 7. Acceptance tests

- Two recipes naming yellow and white onion show two rows and one question; `Combine` yields one
  `Onions` row whose amount is the exact sum and whose details name both recipes and items.
- `Keep separate` keeps two rows and never asks again; changing the answer rebuilds only the
  current week.
- The same recipes, snapshot, and answers always produce the same list (property test over
  rebuild order).
- A checked member and an unchecked member combine to an unchecked row; splitting restores each
  member's state where unambiguous.
- No AI call is reachable from list building (dependency test).
- A group that spans two sections is rejected at classification.

## 8. Decisions needed (owner: Matt)

| Decision | Default |
| --- | --- |
| Unanswered groups | Keep separate and ask |
| Combined row name | The group's display name |
| Where answers are changed | A small "Combined items" list reached from the grocery header |

## 9. Out of scope

Recipe-level alternatives (`butter or oil`), cross-category swaps (butter for oil), learning
preferences from behavior, and per-recipe exceptions stay in [`UP_NEXT.md`](UP_NEXT.md).
