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

### 2.1 Everything below here is optional

The three features that follow — variants, substitution, produced items — are
**enrichments, not requirements**. A recipe file with a title, a source and a flat
ingredient list is complete and always will be. Each of these degrades to exactly the
current behaviour when absent: no variants means one implicit variant, no `accepts:` means
no substitution, no `produces:` means nothing.

This is not politeness, it is the cold-start constraint (§4 of the proposal). Onboarding a
recipe has to stay a five-second operation, and someone arriving with an empty corpus must
never be asked to fill in a substitution matrix before they can eat. **The capture tool
records these when the source or the household states them, and never blocks on them.**
They accumulate over months, the way the corpus itself does.

### 2.2 Variants

Some recipes are cooked more than one way, and the choice changes the shopping list, the
effort, or both. This is already in the corpus at least three times: the chicken noodle
soup (rotisserie chicken, or a whole young chicken boiled for its stock), the meatball subs
(`16 frozen meatballs or homemade below`), and the enchiladas.

```markdown
## Variants

### Rotisserie          <!-- default: first listed -->
active:  ~20 min
passive: 20 min simmer
+ 1 rotisserie chicken, meat pulled

### Whole young chicken
active:  ~35 min
passive: 1.5-2 hr
+ 1 young chicken, 3-4 lb
  replaces: 6 cups chicken broth
  produces: 8 cups chicken stock, keeps 4 days or freezes
```

Lines prefixed `+` are added to the base list. `replaces:` removes a base line. A variant
carries its own effort, since that is usually the whole difference.

**One corpus row, not two.** Splitting a meal into two rows fragments its last-cooked
history and clutters an index that must stay small enough to send to a model on every run.

**The planner picks the variant and names it** — *"chicken noodle soup — rotisserie,
20 min"* — because the choice is a planning decision, not a shopping one. Two hours of
passive time versus twenty minutes decides whether the meal can be proposed at all. The
session shows the choice with one tap to flip it.

### 2.3 Substitution

An ingredient may declare what else would do:

```markdown
- 1 cup shredded provolone cheese    accepts: mozzarella, colby jack
```

**Tolerance is declared, never inferred.** A model reasoning that cheese is cheese will
eventually swap the ingredient that was the point. Sources often state it outright — the
meatball subs already read `provolone or mozzarella` — and the household can state the
rest.

Applied in **Step 2, not the planner**, for the same reason coupling moved: consolidation
needs the whole week's items visible at once. Given declared tolerances it is mechanical —
if a recipe accepts colby jack and colby jack is already in the week's cart, propose the
merge.

**Shown, never silent**, with a one-tap undo:

```
Colby jack, 2 bags -> 1 bag    [subs: meatball subs, enchiladas]   [keep separate]
```

Worth noting because it is not obvious: substitution is what **makes the pack-sizing model
work** (§7 of the proposal). The larger bag pays off only if it gets used, and
consolidation is what creates the second use.

### 2.4 Produced items

A cook can output an ingredient, not just a meal. The corpus already contains real pairs:
the beef pot roast produces jus and the beef dip Sammies needs au jus; the crock pot
Italian beef produces shredded beef and juice well beyond one dinner.

```markdown
produces: 2 cups au jus, keeps 4 days

- 2 cups au jus    may come from: beef pot roast
```

**This is leftovers generalised.** The system already models *cook big Monday, eat
Tuesday*; here the leftover is an ingredient rather than a meal. Framing it that way lets
it reuse machinery that has to exist anyway instead of introducing a dependency graph.

Three rules keep it from becoming one:

**Always buy the fallback.** The link saves you *using* the bought item, never buying it.
Skipping the pot roast must not leave the beef dip with no jus and none in the cart. This
is the strongest coupling in the system and it is the one that breaks rather than degrades
(§13 of the proposal).

**Never a schedule.** A link implies producer-before-consumer, which quietly reintroduces
the day-ordering the pool model deliberately removed. Render it as a note — *"if you do the
pot roast first, the beef dip is nearly free"* — not a sequence.

**A tiebreaker, never a driver.** The planner has already been caught manufacturing
coupling once, choosing a candidate built around an ingredient already in the week.
Output-linking pulls harder in the same direction, and left unchecked it converges on a
small set of mutually reinforcing recipes — which fights breadth, the point of the whole
project.

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
week (from Step 1, with a variant chosen per meal)
  -> load        recipes/<slug>.md for each chosen meal
  -> resolve     apply the chosen variant's + and replaces: lines
  -> parse       ingredient lines -> (qty, unit, item, note, accepts)
  -> scale       recipe yield vs. this week's AE -> multiplier per recipe
  -> normalize   item -> canonical, via items.md synonyms
  -> convert     units -> a common unit per canonical item, via each_equiv
  -> aggregate   sum across recipes, keeping provenance
  -> consolidate merge items whose accepts: lists overlap something already in the week
  -> link        mark lines available from another meal's produces:, keep the buy
  -> classify    staple -> flagged section; everything else -> aisle sections
  -> emit        the list, with every merge and link shown and reversible
```

Three of those stages are no-ops on a recipe that declares nothing, which is the point
(§2.1). `resolve` passes the base list through, `consolidate` finds no tolerances, `link`
finds no producers.

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
