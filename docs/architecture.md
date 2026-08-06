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
| 5 | **Markdown files stay the source of truth.** No database until SaaS, and that is an explicit restructure | Concurrency between two members stays crude; the rules below are enforced by code rather than by schema |
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

### `planner/` as built, and the one place it differs

The two implementations exist and the interface is `pantry.propose()`. One thing in the
sketch above did not happen: **the ranker did not move into `planner/`.** It stays in
`pantry.py` next to the corpus loaders and the `Meal` it builds, because moving it would
have bought a tidier directory listing at the price of a circular import and a diff across
a hundred and twenty-nine passing tests. It is `pantry.rank()` — public and named, so
anything wanting the deterministic answer can ask for it directly.

What lives in `planner/` is the choice (`which()`), the model implementation, the prompt,
and the constraint checks. Selection is: an explicit argument, then `PANTRY_PLANNER`, then
whether `ANTHROPIC_API_KEY` is set. **No key is a supported configuration, not a degraded
one** — it is what the hosted demo and CI run, and CI asserts it.

The model boundary above said a model may *propose the week and write the reasons*. In
practice that had to be narrowed to make it safe, and the narrowing is the design:

**The model selects and explains. It does not state facts.** It is handed the corpus as a
table with a slug column, a computed `days since` column and no ingredients, and it returns
slugs and reasons. Protein, cuisine, yield, active and passive are read back off the corpus
row, so there is nowhere for an invented field to land. A slug that resolves to nothing is
dropped — never resolved to its nearest neighbour, because `onion powder` → `onion` is on
the list of what silent mis-merges cost here. A reason claiming recency about a row with no
last-cooked date is dropped, which is the coupling receipt in its other coat: give a model
a gap and it will fill it with something plausible.

Everything dropped gets made up by the ranker, which is what makes refusing cheap: a
refused pick costs a good reason, never a night's dinner. Every drop, fallback and warning
lands in `decisions.jsonl` and under the week in the session. **A model that quietly
degraded to the ranker for a month would be the real failure**, and it is the one this is
built to make impossible.

## Storage — reversed, on purpose

**This started as "SQLite now, Postgres later" and was reversed after a second look.**
Recorded rather than patched away, because the reasoning is what makes the next reversal
cheap to evaluate.

The case for reversing: a database for a household of one buys correctness nobody is
currently violating, and costs the one property this project has leaned on hardest —
`profile.md` opens by saying that correcting the file **is** the trust mechanism, and that
it beats any opaque score. In files that is free. In a hosted database it is a UI feature
to build, and until it is built the household has *less* control over its own preferences
than it has today. Paying that now, for a benefit that arrives with SaaS, is the wrong
order. Git also goes on being the audit log for free.

The cost accepted: two members editing at once is handled crudely, and the rules below get
enforced in code rather than refused by a schema. **The restructure is a real project when
it comes, not a config change** — that is understood and accepted, not hand-waved.

## The rules, and what enforces them

Files do not refuse bad writes, so a single module does. Everything that mutates household
data goes through it; nothing else opens a file for writing.

| Rule | Today | Under a schema later |
|---|---|---|
| Membership is earned — nothing enters `corpus.md` uncooked | One function may insert a corpus row, and only with a recorded outcome | `state='corpus'` requires a `cook_log` row |
| No claim without a trace | A profile claim without evidence fails the write | `evidence NOT NULL` |
| No writer overwrites a human value | Checked per cell before writing | Same check, plus row-level history |
| Step 2 never calls a model | `domain/` may not import anything doing I/O; a test walks the import graph | Unchanged |

There is a live violation of the first rule today: `onboard.py` will append a never-cooked
recipe straight into the proven corpus. It is stated in five prose locations and enforced
nowhere. That is the write module's first job.

**`corpus.md` and `candidates.md` stay two files** rather than collapsing into one table
with a `state` column. Two files is the cruder expression of a three-state model, but the
separation is legible on disk and the promotion path is one function either way.

## The decision log survives the reversal

The one thing that does **not** get deferred, because the argument for it was never about
storage: **a decision that was not recorded cannot be recovered.** Every proposal, dial
change and outcome gets appended to `decisions.jsonl` as it happens — one JSON object per
line, append-only, diffable, and trivially loadable into whatever the harness becomes.

This is what lets a planner change be replayed against real history instead of argued
about, and it is what answers *is it any good* for someone who did not build it. It costs
one function today and cannot be backfilled at any price.

## Members, without accounts

Decision 2 stands — a household has multiple members, and attribution is the point, because
`profile.md` records outright that a stated preference "may be half-true" when nobody knows
which adult said it. That needs a name on a claim, not a login.

So: **members are a field, not an auth system.** Feedback and profile claims carry who said
them; there is no signup, no session, no password. Auth arrives with hosting, and arrives
against data that already knows who said what.

## Still open

1. **How Kroger is actually talked to.** Official API, and what the product does when it is
   unavailable or the SKU match is wrong. Deferred on purpose to the Kroger step.
2. **How the list reaches the person shopping in v1**, given they are walking aisles.
3. **What is allowed to break.** Prep degrades rather than blocks; the same posture has not
   been stated for the planner, the store adapter, or a half-finished session.

## What survives

`pantry.py`, `prep.py`, `app.py` and `web/index.html` were written against markdown as the
source of truth, which is now the decision rather than the thing being replaced. They stand,
and the work in front of us is the write module, the decision log, member attribution, and
making the session good — not a migration.
