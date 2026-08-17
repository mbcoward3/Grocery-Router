# Domain model — Grocery Router v1

*Output of ticket 01. Settled in a five-round grilling session, 7 August 2026.*

This document names every noun v1 has, says exactly what each one means, gives its type and
its invariants, and — the part that took the work — says which terms it is deliberately
**not** a synonym for. The old repository grew its vocabulary across fifteen documents and
several words ended up carrying two meanings. Every merge below is recorded with the reason
it was made, so a future reader can tell a deliberate collapse from an accident.

**The rule this document is written to.** The map's trap list says every bug in the old repo
rhymes: *the failure is always a plausible value where there should have been a gap.* A
domain model is where that failure gets designed in or designed out, because a type that
cannot express "this property does not apply here" forces every reader to invent a value.
Several decisions below exist for no other reason.

---

## 1. Recipes

### Recipe

A dish this household can cook, with an ingredient list.

```
Recipe {
  id
  title
  role          main | side
  state         found | adopted
  provenance    proven-here | asserted | acquired
  origin        Recipe?          -- set only on an adopted duplicate
  source        URL?             -- where it came from, if anywhere
  yield         Yield
  active        EffortRange
  passive       EffortRange
  protein       string?
  cuisine       string?
  goes_with     string?          -- sides only, a hint
  season        summer | winter | null   -- sides only
  retired       Retirement?
  ingredients   [IngredientLine]
  variants      [Variant]
  notes         text
}
```

**One noun, not two.** `corpus.md` and `recipes/` were two files because a markdown corpus
had to stay small enough to send to a model on every run, and an ingredient store does not.
That is a storage property, not a domain distinction. In Postgres they are one table.

**What survives the merge is an invariant, not a file.** The planner's view of a Recipe
contains **no ingredient lines**. Decision 9 is enforced by the import-graph test it already
names, plus by the shape of the projection the planner is given — not by two files sitting
apart. This is the single most load-bearing rule in the model: a model given a corpus with no
ingredients still invented ingredient coupling once, and a model given the real ones will do
worse.

**A side is a Recipe.** `role` separates a main from a side, and nothing else does. Capture,
parsing, scaling, aggregation, last-cooked and verdicts are identical for both. Two tables
with overlapping columns is the two-implementations trap applied to data, and that trap has
already cost this project two ingredient parsers that disagreed in three of twelve hard
cases.

*Not a synonym for:* **Capture**, **corpus row**, **candidate**, **bank recipe**. See §8.

### Recipe state — `found` and `adopted`

A Recipe is either as it was found, or it is ours.

- **`found`** — created by acquisition, exactly as the source page stated it, and **never
  edited**. It is the permanent answer to *what did the page actually say.*
- **`adopted`** — the household's own copy. Freely editable. This is where `accepts:`
  tolerances, variants, corrections and household notes live.

**Adoption happens at suggestion time.** When the planner puts a found Recipe in front of the
household, the duplicate is created and `origin` points back. The found original stays
untouched forever.

An imported or hand-typed Recipe is **`adopted` from creation and has no `origin`**. That is
a normal state, not a gap — the 27 recipes ticket 07 imports are the household's from the
first keystroke, and no page exists behind them.

**Why keep the original at all.** Because an adopted Recipe stops being the same thing. A
household copy that says *mozzarella or cheddar, whichever the week wants* is no longer the
recipe that was acquired, and the acquired one is the evidence that the recipe was real —
decision 12's schema.org bar means nothing if the verified capture can be edited away.
Decision 19 already forbids deletion; this is the same rule applied to correction.

### Provenance

Where a Recipe came from. **Immutable, set once, never recomputed.**

| Value | Means |
|---|---|
| `proven-here` | this household has cooked it |
| `asserted` | the household says it belongs, without a recorded cook |
| `acquired` | the tool found it |

Decision 13 made provenance replace *"membership is earned"*. That replacement only works if
provenance is kept strictly to **origin** and never asked to carry quality.

*Not a synonym for:* **untried**. These are two axes and the old vocabulary had one word for
both. The evidence is in the working tree: chicken and dumplings is `asserted` and unproven;
sheet pan chicken fajitas is `acquired` and proven — cooked and kept on 3 August. A single
scale cannot hold both rows.

### Untried

**Derived, never stored.** True when no `Cook` with a `kept` Verdict exists for this Recipe.

This is what `candidate` became. The word `candidate` dies as a stored noun because it named
two facts at once — *we found this elsewhere* and *we have not proven it here*. The session
still shows the `[candidate]` marker, and it now reads off `untried` rather than off
provenance, so a proven acquisition stops being marked as a gamble the moment it earns it.

### Retirement

An instruction: **never offer this Recipe again.**

```
Retirement { at, reason }
```

Explicit household action only. **Never derived from a flop count.**

*Not a synonym for:* **flop**. A flop is a `Cook` with a `nope` Verdict — it is evidence, and
at this corpus size it is the most informative signal the system gets all week. A retirement
is an instruction, and it can happen with no flop behind it: being done with a dish is a
reason on its own.

Invariants:

- Reversible. The retirement and the reversal are both logged.
- Carries a reason. Decision 21 applies to the household's decisions, not only the planner's.
- **Stops the planner offering it, and stops nothing else.** A retired Recipe stays readable,
  keeps its history, and still produces a correct shopping list if the household cooks it.

This closes the map's open question *"whether the corpus needs an archived state"*. It does,
and this is it. Nothing is deleted; decision 19 holds.

### Variant

A way of cooking one Recipe that changes the shopping list, the effort, or both.

```
Variant { name, active, passive, adds: [IngredientLine], replaces: [IngredientLine], produces: text? }
```

**One Recipe, not two.** Splitting the chicken noodle soup into a rotisserie row and a
whole-bird row would fragment its last-cooked history. The planner picks the variant and
names it, because two hours of passive time versus twenty minutes decides whether the meal
can be proposed at all.

A Recipe with no declared variant has one implicit variant. Absence is not a gap.

---

## 2. Yield and scale

### Yield

A fact about the dish. **Four shapes, and the fourth is a real value.**

```
Yield =
  | AdultEquivalents(n)          -- "8 AE"          a batch, in people
  | Portions(count, noun)        -- "8 enchiladas"  a batch, in things
  | PerPortion                   -- no batch exists
  | Unknown                      -- a batch dish whose source never said
```

**`PerPortion` is the shape the type exists for.** A BLT has no batch size; you make one per
person, and its ingredient list carries no quantities because there is nothing fixed to
quantify. No source states a yield for a BLT because none could. A model asked *how many does
a BLT serve* answers four, confidently, forever. The value of the schema is that it can say
**this recipe does not have that property**, which is a thing a plausible guess can never say.

**`Unknown` is not the extreme and it is not a gap to fill.** The old repo scored a recipe
with no last-cooked date as *maximally* stale — unknown treated as an endpoint. Yield must not
repeat that. `Unknown` means one thing only: a genuine batch dish whose source never stated
servings. Seven rows are in that state today and nothing but the household can close them.

### Adult-equivalent (AE)

The unit yield and household size are both expressed in. The household's base is ~2.5 AE per
dinner — two adults, a three-year-old, a one-year-old.

*Not a synonym for:* **portion**, **serving**. A serving is what a source prints; an AE is
what an adult eats here.

### PortionConversion

The bridge from `Portions` to AE. **Household-stated, per portion noun, and reusable forever.**

```
PortionConversion { noun, per_adult }   -- "enchilada", 2
```

No source can supply this — it is a fact about the eaters, not about the dish. A slider is a
slider, so one sentence from the household closes it for every recipe measured in sliders.

Two are outstanding: *how many enchiladas is an adult*, and the same for sliders.

### Scaling

Per Recipe, never per week. The Italian beef serves 8 and needs no multiplier; the salmon
serves 2 and needs one. A single week-level factor gets both wrong.

| Yield shape | Multiplier |
|---|---|
| `AdultEquivalents(n)` | `target_AE / n` |
| `Portions(count, noun)` with a conversion | `target_AE / (count / per_adult)` |
| `Portions(count, noun)` with no conversion | ×1, **and the list says it was not scaled** |
| `PerPortion` | `target_AE`, applied to the per-portion amounts; the batch never enters |
| `Unknown` | ×1, **and the list says it was not scaled** |

**Round after aggregating, never before.** 1.5 peppers plus 1.5 peppers is 3 peppers, not 4.

### What Yield does *not* carry

**Yield is not a coverage claim.** Ticket 10 finding 2 recorded that *"serves 8 — one cook,
two nights"* produced one dinner and some lunches, and proposed that the yield cluster carry a
consumption model.

**That finding is closed as an execution error, not a model error.** The household's judgement:
the two dinners were available and were not taken. The arithmetic supports it — 8 AE against a
2.5 AE base, less two adult lunches, still leaves a dinner. Nothing is added to the model, and
no `covers:` field exists.

**The cost, stated plainly.** A second-night claim now lives entirely in the prose of a
`Reason` and has no structured half. Nothing in v1 checks whether such a claim held. Ticket 12
inherits this: the only lever left for scoring it is the reason **kind** (§7), which can answer
*do `yield` reasons get accepted* but can never answer *was this particular claim true*.

*Not a synonym for:* **coverage**, **nights**, **leftovers**. None of the three is a stored
noun in v1.

---

## 3. Effort

### EffortRange

Two axes, and only one of them is capped.

```
EffortRange { low_minutes, high_minutes, source }
source = from-source | household-stated | system-guess
```

- **Active** — hands-on minutes. The household ceiling is 20–30 minutes on weeknights.
- **Passive** — unattended minutes. **Not capped at all.** Slow cookers, braises and long oven
  times are fine on a weeknight.

**Minutes are stored. The `low|med|high` bucket is derived, and exists only for display.**

The old corpus recorded buckets while the ceiling was in minutes, and `med` spans 15–30 —
straddling the ceiling. Comparing a bucket to a ceiling forces the code to invent a boundary,
and an invented boundary is the map's central failure in miniature. Nine of twenty-four rows
sit on that boundary today.

**`source` is required on every effort value.** `profile.md` already admits that every rating
in the corpus is the system's guess and that nothing will ever correct one. Recording the
source makes the guess visible at the point of use instead of only in a note. Ticket 12 owns
the question of what observation could replace a `system-guess`; today the honest answer is
that none exists.

**Why two axes and not one scalar.** Under a single effort number, beef stew and pot roast
rated `high` and were wrongly filtered out of every weeknight. They are moderate active; the
length is all unattended.

*Not a synonym for:* **effort** as a single scalar. That term is retired.

---

## 4. The week

### Week

**Sunday to Saturday**, named by its Sunday date.

A Week is a **pool** with a required effort mix, not a day grid (decision 15). No meal is
assigned to a named night in advance, because which nights are hard is unpredictable here and
a wrong day label trains people to ignore the column.

**This closes ticket 10 finding 5.** The finding read the file `Week of 2026-08-03` against
cooking that ran Sunday 2 August to Saturday 8 August and concluded the boundary was unsettled.
2 August 2026 was a Sunday. Under a Sunday start, that week is exactly one week and nothing
crossed a boundary. The file was misnamed by one day. The boundary is settled because shopping
is a weekend event, and the week the shopping serves starts when the shopping does.

### Session

The one sitting per Week where everything happens: feedback, then the week, then adjustment,
then the list. One Session per Week.

### Meal

A slot in a Week's pool.

```
Meal { week, recipe, variant, scale, reason }
```

*Not a synonym for:* **Cook**. Five cooks across seven nights is not five meals, and a Meal
that never became a Cook is an ordinary state. The teriyaki chicken was a Meal with no Cook.
**It owes no explanation** — no `Unused` noun exists and no reason set is attached to it.

That decision is deliberate and its cost is named: the week's most expensive failure was a
Meal that never got cooked because the hand-written list omitted its ingredients. **The fix is
a Step 2 rule and not a domain noun** — *every meal contributes at least one line to the list,
or the list names the meal it could not cover.* Ticket 10 finding 1 stands unchanged, and
nothing in the vocabulary is shaped around one bad week.

### Cook

An event: **this dish was made, on this date.**

```
Cook { recipe, variant, at, meal?, verdict? }
```

**`meal` is optional, and that is the point.** A Cook with no Meal is an **off-plan cook** —
the household reached past the plan for a recipe the tool already held. Wednesday 5 August was
ground beef tacos, corpus row 12, in the planner's context and never proposed. That is the
recall gap failing in the open, and it is the highest-value signal the week produced. The
model can see it only because a Cook does not require a Meal.

*Not a synonym for:* **Repeat**, **Meal**, and **the cook** meaning a person.

**Terminology.** `Cook` is the event. The person is **the household**, always. The old design
doc used "the cook" for a person, and that word is retired.

### Repeat

An event: **leftovers from a Cook fed another occasion.** Nobody cooked.

```
Repeat { cook, at }
```

Carries the date and the originating Cook, and nothing else. **No verdict of its own** — the
dish was judged when it was cooked.

Recorded with one tap on the originating Cook. Ticket 12's rule governs: a signal that costs
more than one tap will not get recorded.

*Not a synonym for:* **Cook**. A Repeat is not a smaller Cook and does not count toward the
week's cook target.

### Verdict

The judgement on a Cook.

```
Verdict { value: kept | nope, reason }
```

**Optional, and attached to a Cook — never to a Meal.**

Optional because *cooked, and nobody said whether it was kept* is a real state and must read as
a visible gap rather than a silent default. The Italian beef sat in exactly that state for a
week.

**`didn't cook` is not a Verdict value.** It was never a judgement about food, and putting it
on the same axis as `kept` and `nope` was the collapse this cluster existed to undo. A meal
that was not cooked is a Meal with no Cook (§ Meal above).

*Not a synonym for:* **outcome** as a three-valued field. That term is retired.

### Not nouns

**Night** and **leftover** are not stored. A night is a date on a Cook or a Repeat. A leftover
is what a Repeat implies. Neither earns a table, and giving either one would reintroduce the
day grid decision 15 removed.

---

## 5. The profile

Three nouns, and they behave differently enough that one would have been wrong.

### Claim

Something true about how this household eats. **Soft, advisory, and evidence is mandatory.**

```
Claim { text, evidence, evidence_kind, polarity }
evidence_kind = self-report | count-off-the-corpus | observed-event
polarity      = holds | refuted
```

**The rule that governs the profile: no claim without a trace.** Self-report is allowed and is
usually the honest trace. What is forbidden is a claim with no trace at all. A profile that
quietly accumulates ungrounded assertions poisons every week downstream.

**`polarity` exists because of the sandwiches.** Seven of twenty-five recipes were sandwiches
and the model called it *"a real, distinctive pattern"*. Asked directly, the household said
*"I wouldn't put much weight to this, just happens to be so."* That refutation is more valuable
than the claim ever was, and deleting it invites the same count to be re-read the same way next
quarter. Decision 19 applies here: a refuted Claim is kept, marked, and read by the planner as
*this was checked and it is not a pattern.*

### Gap

A named absence. Its entire job is to stop the planner filling a hole with confident invention.

```
Gap { text }
```

Examples that already exist: the corpus is mains-only and is not evidence about how this
household eats; the repertoire is 30–35 and the corpus is a floor; every effort rating is a
guess.

A Gap is not a to-do. Some of them will never close, and saying so is the point.

### Constraint

A hard rule. **Never violated, and no evidence is required** — the household stated it.

```
Constraint { text, machine_checkable: bool }
```

Examples: the peanut allergy; the weeknight active ceiling; the base household size; everyone
eats the same meal.

**A Constraint is not a Claim, and the difference is not who enforces it.** The difference is
that a Claim without a trace is invalid and a Constraint needs none.

**One Constraint has no honest mechanical test:** *everyone eats the same meal.* The old repo
used corpus membership as the proxy — everything in the corpus has been eaten by this
household, children included — and said so plainly rather than inventing a keyword list that
appeared to check more. That proxy is why the planner may only choose from the catalogue. It is
recorded as a Constraint with `machine_checkable: false`.

> **Open — deferred to ticket 04.** Where a Constraint is stored is not settled. It does not
> belong in the profile alongside Claims and Gaps, because the profile is the household's
> editable surface and a Constraint is enforced. Whether it lives on a Household record, in
> configuration, or somewhere else is a schema question and ticket 04 owns it.

---

## 6. Ingredients, items and the list

**The parser's output and the shopping list's input are two types.** They were never the same
thing, and treating them as one is how a provenance chain gets lost.

### IngredientLine

One line of one Recipe, as written.

```
IngredientLine {
  raw          text            -- verbatim, always kept
  parsed       Parsed?         -- null when the parser refused
  accepts      [string]        -- declared tolerances, never inferred
}
Parsed { qty?, unit?, item_name, note? }
```

**A line the parser cannot read is never dropped and never guessed at.** `parsed` is null,
`raw` goes onto the list flagged, and someone sees it. Silently losing a line means someone
gets home without the chuck roast.

**`accepts` is declared, never inferred.** A model reasoning that cheese is cheese will
eventually swap the ingredient that was the point. Sources often state it outright — the
meatball subs already read *provolone or mozzarella* — and the household states the rest. This
is where an adopted Recipe earns its adoption.

### Item

The canonical thing you buy. One row per item, hand-maintained.

```
Item { canonical, aisle, staple: bool, each_equiv: [Conversion], synonyms: [string] }
```

Four jobs: **merge** synonyms, **convert** units, **group** by aisle, **flag** staples instead
of dropping them.

**`canonical item` and `item` are one noun.** "Canonical" was an adjective doing the work of a
distinction that does not exist — there is no non-canonical Item. What the parser produces is
an `item_name`, which is a **string that resolves to an Item**, and that is not an Item.

**`each_equiv` is per-item, not a general conversion table.** Clauses chain: `1 head = 10
cloves; 1 clove = 1 tsp` is what lets a tablespoon of chopped garlic and four whole cloves add
up to *buy one head*.

**Near-identical Items are kept apart on purpose.** `potato`, `yukon_gold_potato` and
`russet_potato` are three rows. Merging them would be the system deciding a substitution nobody
declared.

**The mis-merge rule.** A partial match is accepted **only when every word it leaves behind is
noise**. `onion powder` matched `onion` across thirteen lines and put a fresh onion in the cart
for a teaspoon of spice. `dried thyme` matched fresh thyme across five more. Neither `dried`
nor `fresh` is noise — they name which aisle you walk to. **A mis-merge is worse than an
unknown line**, because an unknown line gets printed and a mis-merge does not. Ticket 12
(ingredient grammar) owns the exact specification.

**`aisle` is the one field in this project allowed to be a considered guess.** A wrong aisle
costs a few steps and never puts the wrong thing in the cart. With Kroger out of scope, no
observation will replace it in v1, and that is recorded rather than left implied.

### ListLine

One row of one week's shopping list.

```
ListLine { item, quantity, unit, section, sources: [Meal], flags }
```

`sources` is the provenance that survives aggregation, and it is what answers *which meals is
this for* and *what is stranded if a night is skipped.* An item with one source meal is
stranded; an item with two is not.

**Coupling is emitted here, as a fact.** The planner cannot compute it — the planner's view of
a Recipe has no ingredients — and asked to show it anyway, the model manufactured it. Step 2
has the data and computes it exactly.

*Not a synonym for:* **IngredientLine**.

### Pack — deleted

Pack sizing existed only for the Kroger adapter, and Kroger is out of scope as a whole. **No
v1 noun replaces it.** The list states a quantity, not a purchase unit.

---

## 7. Reasons and the log

### Reason

**A kind plus prose. Both are required.**

```
Reason { kind, text }
kind = stale | never | protein | cuisine | yield | passive | low | acquire | plain
```

Decision 21 says the reason is the product, and the prose is that product. Nothing generates
the prose from the kind, and no rule constrains how a sentence is written.

**The kind exists because the sentence cannot be scored.** Two recipes stale at different
distances give two sentences and one kind. Accept rate per kind is the only question about
reason quality that the system can answer with nobody in the loop, and it is what survives of
decision 23 after structured predictions were ruled out.

The kind set is closed and is the old repo's, which had one entry per thing the planner can
actually notice. Ticket 05 may add to it; it may not make it open.

### Event

The decision log's row. **Nothing that was decided is lost** — decision 20.

Ticket 12 owns the closed set of events, because an event that answers no tuning question does
not belong and a tuning question with no event is a guess forever. What ticket 01 fixes is the
nouns those events carry, all defined above: `Meal`, `Cook`, `Repeat`, `Verdict`, `Reason`,
`Retirement`, `Claim`, `Recipe`.

Two events are named here because their absence was the finding that created ticket 12:

- **the off-plan cook** — a `Cook` with no `meal`. Nothing recorded this before.
- **the retirement and its reversal** — both, per §1.

---

## 8. The merge record

What collapsed, and why.

| Old terms | Now | Why |
|---|---|---|
| corpus row, recipe | **Recipe** | The split was a file-size property, not a domain one. The invariant that survives is that the planner's view carries no ingredients |
| candidate | **untried** (derived) + **provenance** | One word for two independent facts. Chicken and dumplings is `asserted`/untried; the fajitas are `acquired`/proven |
| side | **Recipe** with `role: side` | Every mechanism is identical. Two tables with overlapping columns is the two-parsers trap applied to data |
| bank recipe | *deleted* | The scraped bank is v2 |
| canonical item, item | **Item** | There is no non-canonical Item. The parser emits an `item_name`, which is a string |
| pack | *deleted* | Kroger is out of scope as a whole |
| family | **household** | Both were in use for one thing. `household` is the word every existing document already carries |
| the cook (a person) | **the household** | `Cook` is now an event and cannot also be a person |
| outcome: kept / nope / didn't cook | **Verdict** (`kept`\|`nope`) on a **Cook** | Two of the three judge the food and the third says no food happened. One axis could not hold both |
| effort (one scalar) | **active** and **passive** | A single number wrongly excluded every slow-cooker meal from every weeknight |
| leftover, night | *not nouns* | Both are derived. A stored night reintroduces the day grid decision 15 removed |

## 9. What this document does not settle

Named rather than left to be discovered.

- **Where a `Constraint` is stored.** Ticket 04.
- ~~**The closed set of `Event`s.**~~ Settled 8 August 2026: `.scratch/spec/signals.md` § 2.
- **The exact ingredient grammar and mis-merge rule**, as a specification. Ticket 12
  (ingredient grammar).
- **Whether `Reason.kind` gains entries** for a model planner that notices something the
  ranker could not. Ticket 05.
- **Whether a second-night claim is ever checkable.** It is not, in v1, by decision. §2.
- ~~**What replaces a `system-guess` effort value.**~~ Settled 8 August 2026: a coarse
  `faster | as stated | slower` on a `Cook`. Direction only, never minutes. `signals.md` § 4.3.
  The **passive** range keeps no replacement and stays a guess for good.
- **Whether `aisle` ever stops being a guess.** Not in v1 — Kroger is out of scope.
