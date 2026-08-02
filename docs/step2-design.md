# Step 2 — The List

**Design doc. Concrete enough to build from.**

Step 1 chooses the week. Step 2 turns that week into a grocery list. This document
specifies Step 2 and the storage it needs, which does not currently exist.

---

## 1. The missing layer

`corpus.md` says ingredients "feed the shopping list, which is a separate deterministic
step." Nothing in the repo stores ingredients. Step 2 is specified to consume data with
no home, which is why the first real list was assembled by hand and why the planner could
not compute coupling (it was asked to share perishables across recipes whose contents it
cannot see).

**The fix is a split, not an addition to the corpus.** Two different things got conflated:

| | `corpus.md` | `recipes/` |
|---|---|---|
| **Is** | a planning *index* | a recipe *store* |
| **Holds** | title, protein, cuisine, active, passive, last-cooked | ingredients, quantities, yield, source |
| **Read by** | the planner, every run, whole file in context | Step 2, only for the chosen week |
| **Size** | one line per recipe, stays small | unbounded |
| **Edited** | by hand, often | on capture, rarely |

The corpus stays deliberately thin because it goes into the model's context on every call
(§10: no retrieval layer). The recipe store is never sent to the model at all — Step 2 is
deterministic code. **Keeping them separate is what lets both properties hold at once.**

One field moves into the corpus: **yield**. The planner needs it to reason about
leftovers, and it is one number. The cold run guessed the Italian beef needed a double
batch when the source recipe already serves 8 — a yield column prevents exactly that.

## 2. Recipe files

`recipes/<slug>.md`, one per recipe. Human-editable markdown, parseable with the standard
library, same as every other file here.

```markdown
# Crock Pot Italian Beef Sandwiches

source:  https://iowagirleats.com/crock-pot-italian-beef-sandwiches/
yield:   8 AE
active:  low
passive: 10 hr slow cooker

## Ingredients

- 3 lb chuck roast, trimmed and cut into large pieces
- 1 envelope Italian salad dressing mix
- 8 oz pepperoncini pepper slices, plus extra for serving
- 8 oz Chicago-style giardiniera, drained, plus extra for serving
- 14.5 oz can beef broth
- provolone cheese slices
- buns
```

Steps are optional and unparsed. They exist so the cook can read the file; Step 2 ignores
them.

**Ingredient line grammar**, in parse order:

```
- <qty> <unit> <item>, <note>
- <qty> <item>, <note>          # unit omitted -> each
- <item>                        # no quantity -> "some", flagged
```

The known-hard cases (§10), all of which appear in the 25 already seeded:

| Line | Difficulty |
|---|---|
| `1 (14.5 oz) can beef broth` | nested quantity — the can is the unit, the oz is the size |
| `3 cups bell peppers, sliced` | volume of a countable thing |
| `1 envelope Italian salad dressing mix` | non-standard unit, packaging-defined |
| `juice of 1 lemon` | quantity expressed as a source object |
| `salt, to taste` | no quantity, and a staple — must not reach the list |
| `8 oz pepperoncini, plus extra for serving` | two quantities, one of them unmeasured |

Parse failures are **surfaced, never dropped**. An ingredient the parser cannot read goes
onto the list as raw text with a flag. Silently losing a line means someone gets home
without the chuck roast.

## 3. Canonical items

`items.md` — one hand-maintained table, the normalization target and the thing that makes
aggregation possible.

```
| canonical      | aisle   | staple | each_equiv     | synonyms                        |
|----------------|---------|--------|----------------|---------------------------------|
| bell_pepper    | produce | no     | 1 ea = 1 cup   | green/red/yellow bell pepper    |
| onion          | produce | no     | 1 ea = 1 cup   | white onion, yellow onion       |
| garlic         | produce | no     | 1 head = 10 cl | garlic clove, minced garlic     |
| cumin          | pantry  | yes    |                | ground cumin                    |
| olive_oil      | pantry  | yes    |                | garlic olive oil                |
```

Four jobs, one table:

- **Merge** — `3 cups bell peppers` (fajitas) and `1 green bell pepper` (sausage and
  peppers) become one line.
- **Convert** — `each_equiv` is what lets cups and counts add up. This is the whole of
  unit reconciliation for produce, and it is per-item, not general.
- **Group** — `aisle` produces the sectioned list.
- **Flag** — `staple: yes` routes an item to *probably have, check before you go* rather
  than dropping it (§6: a silent drop means no eggs, a visible flag costs one tap).

The table starts small and grows on parse failure. An unrecognized item defaults to
`aisle: other, staple: no` and gets reported, so the miss is visible rather than silent.

## 4. Pipeline

```
week (from Step 1)
  -> load        recipes/<slug>.md for each chosen meal
  -> parse       ingredient lines -> (qty, unit, item, note)
  -> scale       recipe yield vs. this week's AE -> multiplier per recipe
  -> normalize   item -> canonical, via items.md synonyms
  -> convert     units -> a common unit per canonical item, via each_equiv
  -> aggregate   sum across recipes, keeping provenance
  -> classify    staple -> flagged section; everything else -> aisle sections
  -> emit        the list
```

Deterministic end to end. No model in this path — parsing is code, not a prompt. Every
stage is independently testable, and provenance is carried through so any line can answer
*which meals is this for.*

**Scaling is per-recipe, not per-week.** The Italian beef serves 8 and the week needs it
to cover two nights; the salmon serves 1 and needs ×3. A single week-level multiplier
would get both wrong.

**Round after aggregating, never before.** 1.5 peppers + 1.5 peppers is 3 peppers, not 4.

## 5. Coupling belongs here

Resolving the contradiction the cold run found. Coupling moves out of the planner:

- The planner cannot compute it — the corpus has no ingredients by design.
- Asked to show coupling anyway, the model *manufactured* it, choosing a candidate built
  around peppers so it would couple with an existing pick. That silently converts coupling
  from an observation about the week into a filter on which recipes get proposed.
- Step 2 has the ingredient data already and can compute it exactly.

So: **the planner stops claiming coupling.** Step 2 emits it, from the aggregation's
provenance, as a fact rather than a guess:

```
Bell peppers      5     -> sheet pan fajitas, sausage and peppers
Onion             2     -> sheet pan fajitas, sausage and peppers
Salmon           18 oz  -> parchment salmon                [no shared items]
```

Which also answers *what breaks if a night is skipped* for free: an item with one source
meal is stranded, an item with two is not.

## 6. Output

Sectioned by aisle, staples flagged at the bottom, provenance available per line. The
hand-built list for the week of Aug 2 is the **acceptance fixture** — the correct output
is already known, which makes this gradeable before a single week is cooked.

Two things the list must say, because the corpus cannot supply them:

- **Sides are not included** (§5). The corpus is mains-only; a complete-looking list that
  silently omits every vegetable side is worse than one that admits the gap.
- **Any ingredient line that failed to parse**, verbatim.

## 7. Not in scope here

SKU matching, pack sizing (§7), and the cart write are Step 3 and the store adapter. Step
2 ends at canonical items with quantities. The adapter interface in §10 takes it from
there, and stays behind that boundary so a second store is a new file.

## 8. Build order

1. `recipes/` schema + parser, tested against the six recipes already typed out in the
   source PDF and the four fetched from links. Ten real recipes is a real test set.
2. `items.md` + normalize/convert, seeded from those ten.
3. Aggregate + emit, graded against the Aug 2 fixture.
4. Coupling report, which falls out of provenance at near-zero extra cost.
5. Backfill the remaining corpus recipes into `recipes/` as they get cooked — no need to
   do all 25 up front, since Step 2 only loads the chosen week.
