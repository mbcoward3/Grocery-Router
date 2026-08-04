# Architecture

**Decided in interview. This document is the shared understanding, not a proposal.**

Everything before this was product logic — what a yield is, what coupling means, when a
recipe has earned its place. That work is good and it survives. None of it was
architecture, and this file exists because we had built a lot of the former while calling
it the latter.

Where a decision has a consequence that costs us something, the cost is written down next
to it. A decision doc that only lists upsides is a sales document.

---

## What this is

A household meal-planning and grocery-sourcing tool. **Also a portfolio project for a
Forward Deployed Engineer role**, which is a real architectural input rather than a
footnote: the reader is evaluating judgment about where a model belongs and where it is a
liability, and whether the thing actually deploys and works against messy real data.

That reframes two things. Model cost is **not** a design driver — nobody is impressed by
forty cents saved per household. And the system needs to be able to answer *is it any
good* to someone who did not build it.

## Decisions

| # | Decision | Costs us |
|---|---|---|
| 1 | **Laptop now, built so hosting is a deployment change** | Some abstraction that a purely local tool would not need |
| 2 | **A household has multiple members** | A user/household split in every table from day one |
| 3 | **v1 assumes someone walks the aisles.** Kroger pickup arrives with the Kroger step | Aisle sorting becomes vestigial the day pickup lands |
| 4 | **The tool fills a cart; a human submits it** | Never a fully unattended week — deliberately |
| 5 | **SQLite now, Postgres later.** Markdown becomes import/export | "Fix the file by hand" stops being free and becomes UI I have to build |
| 6 | **A model goes wherever it makes the product better** | Non-determinism in more places; the discipline has to be explicit, not incidental |
| 7 | **One process.** Prep is a real job, triggered by a button in v1 | Long jobs block a request until there is a worker |
| 8 | **Log every week; show a few metrics** | A decisions table on day one, unrecoverable if skipped |

## The model boundary, stated as a rule

The contrast *is* the portfolio story, so it gets written down rather than left to taste.

**A model may:** propose the week and write the reasons; read screenshots and messy recipe
text; interpret free-text feedback.

**A model may not:** touch the shopping list. Step 2 is deterministic code end to end. This
is not caution for its own sake — asked to reason about ingredient coupling from a corpus
index that deliberately contains no ingredients, a model manufactured it, and picked a
recipe *because* the invented coupling justified it. The rule has a receipt.

**Enforced, not asserted:** `domain/` may not import anything that performs I/O. A test
walks the import graph. "No model in the shopping list" becomes a property of the build
rather than a paragraph in a design doc that nothing checks.

## Shape

```
web/          HTTP + the page. Knows nothing about SQL.
planner/      Two implementations behind one interface: a deterministic ranker
              that always works, and a model planner. The ranker is not a
              fallback to apologise for — it has to be genuinely good, because
              it is what runs in a demo with no key.
prep/         The Step 0 briefing job. One entry point, called by a button now
              and a scheduler later. Degrades, never blocks.
domain/       Recipes, ingredients, weeks, items, scaling, aggregation.
              Pure. No I/O, no network, no SQL, no model.
store/        The only code that knows what SQLite is. One repository per
              aggregate. Postgres later is an implementation, not a rewrite.
adapters/     Kroger, behind an interface, so a second store is a new file.
```

## Schema, and the rules it enforces

The interesting part is that most of the product rules we had been *stating* become
things the database can *refuse*.

```
household(id, name)
member(id, household_id, name)

recipe(id, household_id, title, slug, protein, cuisine,
       yield_kind, yield_n, yield_unit,          -- the three shapes, 2.5
       active, passive, source, modality, capture_status,
       state)                                    -- candidate | corpus | retired
ingredient(id, recipe_id, position, raw, qty, unit, item, note,
           pack_n, pack_unit, may_come_from)
ingredient_accepts(ingredient_id, item_id)       -- declared, never inferred
variant(id, recipe_id, name, position, active, passive)
variant_line(id, variant_id, op, text)           -- add | replaces | produces

item(id, household_id, canonical, aisle, staple, each_equiv)
synonym(item_id, text)

week(id, household_id, date, nights, guests, risk, status)
week_meal(id, week_id, recipe_id, variant_id, position, reason, locked)
cook_log(id, household_id, recipe_id, week_id, cooked_on,
         outcome, member_id, note)               -- kept | flopped | not cooked

profile_claim(id, household_id, section, text,
              evidence NOT NULL,                 -- "no claim without a trace"
              member_id, created_at)
decision(id, household_id, week_id, kind, payload, created_at)
```

Three of these are rules made structural:

- **`corpus.md` and `candidates.md` collapse into one table** with a `state` column. They
  were always one thing with a boolean on it, and keeping them as two files meant the
  three-state model lived in prose. Promotion becomes a state change with a `cook_log` row
  behind it — and the constraint is that `state = 'corpus'` requires one. Membership is
  earned, in the schema.
- **`profile_claim.evidence` is `NOT NULL`.** The rule `profile.md` opens with, enforced by
  the database instead of by whoever is reading.
- **`decision` is the evaluation harness.** Every proposal, dial change and outcome, stored
  as it happens. It is what lets a planner change be replayed against real history instead
  of argued about. It cannot be reconstructed later, which is why it exists on day one.

## Migration

Everything already built becomes seed data, not waste. `corpus.md`, `candidates.md`,
`items.md`, `recipes/*.md` and `profile.md` are the fixture the importer is written
against — 24 recipes, 27 files, 265 ingredient lines, 119 canonical items, all real. A
one-time import, and the export path back out is what keeps "your data is yours" true
after files stop being the truth.

The determinism work survives intact: `shop.py`'s pipeline is already pure functions over
parsed data, so it moves into `domain/` almost unchanged.

## Still open

1. **How much tenancy gets built now** — multi-tenant schema with one seeded household and
   no login, versus real auth and signup. The shape is settled; only the timing is not.
2. **How Kroger is actually talked to.** Official API, and what the product does when it
   is unavailable or the SKU match is wrong. Deferred on purpose to the Kroger step.
3. **How the list reaches the person shopping in v1**, given they are walking aisles.
4. **What is allowed to break.** Prep degrades rather than blocks; the same posture has not
   been stated for the planner, the store adapter, or a half-finished session.

## What this invalidates

`pantry.py`, `app.py`, `prep.py` and `web/index.html` were written against markdown as the
source of truth and a single household. They are on disk, uncommitted, and get reworked
rather than kept. The ranker and the session flow inside them are worth carrying over; the
storage assumptions are not.
