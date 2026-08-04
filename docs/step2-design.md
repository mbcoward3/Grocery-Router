# Step 2 — The List

**Design doc. Built — `shop.py`, `test_shop.py`, `items.md`, `recipes/`.**

Step 1 chooses the week. Step 2 turns that week into a grocery list. This document
specified Step 2 before it existed; it now describes something running. Where the two ever
disagree, the code is right and this file is stale — say so here rather than quietly
patching around it. §2.4 and §2.5 both carry corrections that building it forced.

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
They accumulate as recipes get cooked, and nothing waits on them.

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

A cook can output an ingredient, not just a meal.

```markdown
produces: chicken stock, keeps 4 days or freezes

- 6 cups chicken broth    may come from: chicken noodle soup, whole-bird variant
```

**The corpus contains exactly one of these, and it is worth saying how few that is.** This
section was first written claiming three pairs — the pot roast making jus for the beef dip
Sammies, the Italian beef making shredded beef for something. Read against the actual
files, neither survives: the beef dip makes its own liquid from its own roast and never
buys jus, and nothing in the corpus consumes shredded beef. Both were plausible and both
were invented. The one real pair is inside a single recipe — the whole-bird chicken noodle
soup boils its own stock and `replaces:` the bought broth with it.

That ratio is the argument for the three rules below. A feature that fires once in
twenty-four recipes must not be allowed to reshape the twenty-three.

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

### 2.5 Yield is three things, not one

Written first as a single number in adult-equivalents, on the assumption that every recipe
has one and some sources forget to print it. Going back to fifteen sources to fill the
blanks disproved that. **Not every recipe has a yield, and of those that do, not all state
it in people.**

```markdown
yield:   8 AE                    # a batch, in adult-equivalents
yield:   8 enchiladas            # a batch, in portions
portion: enchilada; 2 per adult  # the conversion, household-stated, optional
yield:   per portion             # no batch exists
yield:   unknown                 # a batch dish whose source never said
```

**`N AE`** is the ordinary case and needs no comment.

**`N <things>`** is what published sources actually print for anything countable: *8
enchiladas*, *24 sliders*. That is real information and throwing it away as `unknown` loses
the shopping list's scaling factor. What it lacks is the appetite number, and **no source
can supply that** — it is a fact about the eaters. One sentence from the household turns it
into AE, and it is reusable forever, because a slider is a slider.

**`per portion` is the case that was missing entirely**, and it changes what gets asked.
Burgers, tacos and BLTs have no batch size: you make as many as there are people. `2lb
ground beef` on the hamburgers is a unit of purchase, not a serving. Their ingredient lists
carry almost no quantities — not because the capture failed, but because there is nothing
fixed to quantify. **No source states a yield for these because none could**, and asking
the household is asking a question with no answer. Scaling is `AE × per-portion amount`
and the batch never enters it.

Recognising this shrank the outstanding work: of fifteen `unknown` yields, five were
recoverable from sources, three were never questions, and seven are genuine.

**Why the distinction has to live in the data and not in the planner's head.** A model asked
*how many does a BLT serve* will answer four, confidently, forever. The value of a schema
here is that it can say *this recipe does not have that property* — which is a thing a
plausible guess can never say.

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
  -> scale       recipe yield vs. this week's AE -> multiplier per recipe (§2.5:
                 `N AE` divides; `N things` needs a portion rate; `per portion`
                 multiplies by AE directly; `unknown` scales x1 and says so)
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

## 8. Build order — done

1. ~~`recipes/` schema + parser~~ — **done**, and the test set turned out to be all 27
   files rather than the ten planned. `./shop.py --audit` reports **265 ingredient lines,
   0 unparseable**.
2. ~~`items.md` + normalize/convert~~ — **done**, 119 rows, and `--audit` reports 0 lines
   with no row. It was 27 rows and 154 unknown names when this was written.
3. ~~Aggregate + emit~~ — **done**, graded against the Aug 2 fixture in `test_shop.py`.
4. ~~Coupling report~~ — **done**, and it did fall out of provenance for nearly nothing.
5. Backfill remaining recipes as they get cooked — **not needed**, all 27 exist.

**What building it changed in this document**, since a design doc that survives contact
unamended is usually a design doc nobody read:

- **§2.5 is new.** `yield` was specified as one number in adult-equivalents. Three of the
  corpus's recipes have no batch size at all, and two published sources state portions
  rather than servings. See `docs/onboarding-pass-2-findings.md` §1.
- **§2.4's examples were wrong.** Two of the three `produces:` pairs it claimed do not
  exist in the files. Corrected in place, with the count pinned by a test.
- **The normalizer needed a safety rule that isn't in §3.** Matching sub-phrases of an item
  name let `onion powder` resolve to `onion` — a fresh onion in the cart for a teaspoon of
  spice, silently, across eighteen lines of the corpus. A partial match is now only accepted
  when every word it leaves behind is noise. **A mis-merge is worse than an unknown line**,
  because an unknown line gets printed and a mis-merge does not, and §3's "grows on parse
  failure" model quietly assumed failures are visible.
- **One line can name two items.** `2 tsp thyme and rosemary` matched `thyme` and dropped
  the rosemary. Both are emitted now, and neither carries the quantity, because splitting
  `2 tsp` between them would be a guess.

## 9. What is still not built

- **Step 3.** SKU matching, pack sizing, the Kroger cart. §7 above.
- **Consolidation has never fired on real data.** Four of the corpus's five declared
  tolerances resolve to the same canonical item as the thing they replace, and the fifth has
  no partner in the corpus. It is correct and tested; it is waiting for more recipes.
- **`produces:` has one instance**, inside a single recipe. The cross-recipe case the
  feature was designed for has not occurred.
- **Sides.** The corpus is mains-only and every list says so. This is the largest known
  omission in the output and it is a capture problem, not a code problem.
