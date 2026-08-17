# Grocery Router

A meal-planning and grocery-sourcing tool for one household. The problem is recall, not
discovery: the household cooks roughly 15 of the ~32 recipes it knows and likes, and closing
that gap is the product.

This file is the glossary. Types, invariants and the reasoning behind each merge live in
[`.scratch/spec/domain-model.md`](.scratch/spec/domain-model.md).

## Language

### Recipes

**Recipe**:
A dish this household can cook, with an ingredient list. A side is a Recipe with `role: side`.
_Avoid_: corpus row, bank recipe, dish, entry

**found**:
The state of a Recipe exactly as acquisition captured it. Never edited.
_Avoid_: source recipe, original, capture

**adopted**:
The state of the household's own copy of a Recipe. Freely editable, and where tolerances and
variants live. Created when the planner first suggests a found Recipe.
_Avoid_: clone, fork, owned copy

**Provenance**:
Where a Recipe came from — `proven-here`, `asserted` or `acquired`. Immutable, and never a
statement about quality.
_Avoid_: membership, status, origin

**untried**:
Derived: no Cook with a `kept` Verdict exists for this Recipe. What the session marks
`[candidate]`.
_Avoid_: candidate, unproven, new

**Retirement**:
An instruction that a Recipe must never be offered again. Explicit, reasoned and reversible.
_Avoid_: archive, delete, hide, disable

**Variant**:
A way of cooking one Recipe that changes the shopping list, the effort, or both. One Recipe,
never two.
_Avoid_: version, option, alternative

### Yield and effort

**Yield**:
A fact about how much a dish makes — adult-equivalents, a portion count, `per portion`, or
`unknown`. Not a claim about how many meals it covers.
_Avoid_: servings, serves, portions

**Adult-equivalent (AE)**:
The unit of appetite. This household's base is ~2.5 AE per dinner.
_Avoid_: serving, portion, head

**PortionConversion**:
How many of a countable portion make one adult. Stated by the household, never by a source.
_Avoid_: serving size, ratio

**Active**:
Hands-on minutes. Capped on weeknights.
_Avoid_: effort, difficulty, prep time

**Passive**:
Unattended minutes. Never capped.
_Avoid_: cook time, wait time

### The week

**Week**:
Sunday to Saturday, named by its Sunday. A pool of Meals with a required effort mix, never a
day grid.
_Avoid_: plan, menu, cycle

**Session**:
The one sitting per Week: feedback, then the week, then adjustment, then the list.
_Avoid_: run, planning session, sitting

**Meal**:
A slot in a Week's pool — a Recipe, a variant, a scale and a Reason. Not something that
happened.
_Avoid_: pick, proposal, slot, dinner

**Cook**:
An event: this dish was made, on this date. May have no Meal behind it, which is how an
off-plan cook is recorded. Never a person.
_Avoid_: cooking, meal, the cook, night

**Repeat**:
An event: leftovers from a Cook fed another occasion. Nobody cooked, and it carries no
Verdict.
_Avoid_: leftover, second night, reheat

**Verdict**:
The judgement on a Cook — `kept` or `nope`, with a reason. Optional, so an unrecorded verdict
reads as a gap.
_Avoid_: outcome, rating, feedback, score

**flop**:
A Cook with a `nope` Verdict. Evidence, and not an instruction to stop offering the Recipe.
_Avoid_: failure, reject

### The profile

**Claim**:
Something true about how this household eats. Advisory, and invalid without a trace. Carries a
polarity, so a refuted claim is kept and marked.
_Avoid_: preference, fact, rule, insight

**Gap**:
A named absence in the profile, recorded so the planner does not fill it with invention.
_Avoid_: unknown, todo, missing

**Constraint**:
A hard rule the planner may never violate. Needs no evidence, because the household stated it.
_Avoid_: preference, requirement, filter

### Ingredients and the list

**IngredientLine**:
One line of one Recipe, kept verbatim alongside whatever the parser could read. A line the
parser refuses is flagged, never dropped.
_Avoid_: ingredient, line item

**Item**:
The canonical thing you buy, with its aisle, its staple flag and its conversions. What a
recipe names is an item name — a string that resolves to an Item.
_Avoid_: canonical item, product, SKU, ingredient

**ListLine**:
One row of one week's shopping list — an Item, a quantity, a section, and the Meals it came
from. Not an IngredientLine.
_Avoid_: item, list item, entry

**staple**:
An Item routed to *probably have, check before you go* rather than dropped from the list.
_Avoid_: pantry item, basic

### Reasons and the log

**Reason**:
Why a Meal was proposed — a kind from a closed set, plus prose. Both required. The prose is
the product; the kind is the only part that can be scored.
_Avoid_: explanation, justification, rationale

**Event**:
A row of the decision log, and only for what no state table holds. Nothing that was decided is
lost.
_Avoid_: log entry, record, audit

**off-plan cook**:
A Cook with no Meal. The household reached past the plan. The rate of these is the recall gap
measured directly, and it is v1's primary self-measurement.
_Avoid_: unplanned meal, substitution

**incomplete**:
Derived: a Recipe with no IngredientLine. The planner may never propose one. How a retrospective
off-plan cook of an unknown dish enters the corpus.
_Avoid_: draft, stub, partial

**Origin**:
Where a field's value came from — `stated`, `observed` or `system-guess`. Every guessable field
carries one, so a guess can never pass as a fact.
_Avoid_: source, confidence, quality

**Scoring run**:
A replay over the log that answers *is the planner any good* with nobody in the loop. It reports
and it never acts.
_Avoid_: review, report, metrics, analytics
