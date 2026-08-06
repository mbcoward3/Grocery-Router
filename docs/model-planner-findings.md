# Findings — putting a model in the planner

*Same standard as the two onboarding briefs. What worked, what did not, what the schema
could not express, and the questions worth asking the household — separated from the ones
that answer themselves.*

Task §1 from `docs/brief-next.md`. The brief called it the single biggest gap, on the
grounds that no model had ever run in this product: `plan.py` printed a prompt to a
terminal, `app.py` used the ranker, and `state()` hardcoded `"planner": "ranker"`.

---

## What got built

`planner/` — the choice, the prompt, the model implementation, the constraint checks.
`pantry.propose()` is still the one call. `pantry.rank()` is the ranker, now public and
named rather than being the body of `propose()`.

Selection is an explicit argument, then `PANTRY_PLANNER`, then whether `ANTHROPIC_API_KEY`
is set. **No key is a supported configuration**, not a degraded one — it is what the hosted
demo and CI run, and CI asserts it rather than trusting it.

171 tests, 42 of them new, standard library only, no key and no network.

## The design decision that mattered

The brief's model boundary said a model may *propose the week and write the reasons*.
Building it, that had to narrow into something much sharper before it was safe:

> **The model selects and explains. It does not state facts.**

It is handed the corpus as a table with a slug column, a computed `days since` column and
no ingredients at all. It returns slugs and reasons. Protein, cuisine, yield, active and
passive are read back off the corpus row it resolved to, so an invented field has nowhere
to land — not because it is filtered, but because nothing reads it.

Everything else fell out of that one sentence:

- A slug that resolves to nothing is **dropped, never nudged to its nearest neighbour.**
  This project already knows what a silent mis-merge costs: `onion powder` resolved to
  `onion` across thirteen lines and put a fresh onion in the cart for a teaspoon of spice.
- A reason claiming recency about a row with no last-cooked date is dropped. This is the
  invented-coupling receipt in its other coat — most of this corpus has no date, and *you
  haven't made this since March* is a fact about the household that nobody has.
- Membership is read here, never claimed there. The model is told explicitly not to mark
  candidates in the machine-readable block, because it is not in a position to know.
- `days since` is **computed and handed over** rather than asked for. Date arithmetic is a
  gap, and the standing lesson of this repo is that a gap gets filled with something
  plausible.

**Whatever gets dropped, the ranker fills in.** That is what makes refusing cheap enough to
do freely: a refused pick costs a good reason, never a night's dinner. It is also why the
ranker had to stay genuinely good rather than becoming an apology — it is now load-bearing
on the model's best day, not just its worst.

## What worked

- **The prompt survived reuse intact.** The brief said it was not naive and not to rewrite
  it, and that held. It moved to `planner/prompt.py` and gained one appended section — a
  machine-readable envelope *after* the prose, never instead of it. Nothing in the original
  text needed changing to serve a program instead of a person.
- **The `client` seam is the whole test strategy.** Every failure path — an invented
  recipe, a peanut recipe, a truncated reply, a 529, a week of pure hallucination — is a
  three-line stub. Those are the paths a real key exercises least and that have to work
  most, and they are now the best-tested code in the planner.
- **The capture already knew about peanuts.** `onboard.py` writes a `peanut:` verdict into
  every recipe file. The constraint check reads that verdict back rather than scanning
  ingredients a second time, so there is one peanut implementation and a fix to it fixes
  the shopping list too. Twenty-three of twenty-seven captures carry it.
- **`decisions.jsonl` absorbed the new planner without a schema change.** It is where the
  session now reads which planner ran, which means the answer survives a restart and does
  not depend on the process that planned the week still being alive.

## What did not

- **`planner/` does not contain the ranker**, which is what `docs/architecture.md`
  sketched. Moving it would have bought a tidier directory listing at the price of a
  circular import and a diff across 129 passing tests. Recorded in `architecture.md` as a
  deviation rather than quietly left to be discovered.
- **The static build broke, and the reason is worth keeping.** `mkdirTree('/app/recipes')`
  had been creating `/app` as a side effect. Generalising it to per-file directories
  removed that accident and the page went blank with `ErrnoError` — a green unit suite and
  a dead public URL, which is the exact failure mode `smoke_static.py` exists for. It
  caught it. `mkdirTree('/app')` is now explicit.
- **A truncated reply used to look like a formatting failure.** The JSON block is the last
  thing in the answer, so `max_tokens` presents as *no json block found*, which sends the
  next person to read the prompt instead of the token ceiling. It is now checked and named.

## What the schema could not express

- **Family-edible has no honest mechanical test.** It is a hard constraint in `profile.md`
  and there is no keyword list that would check it without inventing one. The proxy is
  corpus membership — everything in `corpus.md` has been cooked and eaten by this
  household, children included — and that proxy is *why* the model may only pick from the
  catalogue. Written down as a boundary rather than papered over with something that looked
  like a check.
- **Reason quality is not testable.** *Traceable to the profile or the corpus* is the bar,
  and what is enforced is narrower: no recency without a date, no empty reason, no fact
  about a meal that did not come from its row. Whether a surviving reason is a *good* one
  is a judgment, and the only instrument for it is a household reading five of them.
- **Active is a three-value enum and the ceiling is in minutes.** `profile.md` caps
  weeknights at 20–30 minutes active; the corpus records `low | med | high`. The check
  reads *at most two high-active cooks* off the sentence about weekend nights, which is a
  count from the profile rather than a taste judgment — but it cannot tell a 25-minute
  `med` from a 35-minute one. Every effort rating is also the system's guess and marked
  unverified. This constraint is the softest thing in a file about hard constraints.
- **The corpus has no high-active rows at all.** The ceiling could not bite on real data,
  so the test flips rows to `high` in a scratch copy to make it bite.

## Questions that answer themselves

Recorded so nobody spends a household conversation on them.

- *Which model?* `$PANTRY_MODEL`, defaulting to `claude-sonnet-5`. Cost is explicitly not a
  design driver here (`architecture.md`), so this is a quality question, and it is settled
  by running a week and reading it.
- *Should a constraint violation be repaired instead of dropped?* No. Substituting the
  ranker's sentence under the model's pick would attribute a reason to a decision that was
  never made for it — which is precisely how a candidate once inherited a corpus recipe's
  reason and claimed membership it did not have.
- *Should the week's shape be day-bound?* Already answered in `profile.md`: hard nights are
  unpredictable, so the output is a pool. The prompt reads it off the profile.

## Questions worth asking the household

1. **Does the model's reason actually land?** Not *is it true* — that is enforced now — but
   does reading it make you want to cook the thing. This is the entire product claim and it
   has never been put in front of anyone. Five model reasons and five ranker reasons for
   the same week, unlabelled, is the cheapest possible version of this test.
2. **Do you want the repertoire widened?** Still open from `profile.md`, and it now
   matters more: the model has real latitude over candidate count and risk, and the honest
   setting for that dial depends on an answer nobody has given.
3. **Is `med` inside the weeknight ceiling?** Nine of twenty-four rows are `med` and every
   rating is the system's guess. If some of them are 35-minute cooks, the ceiling check is
   watching the wrong column.
4. **What happens when a plan breaks?** Still `[...]` in `profile.md`. The prompt spends a
   paragraph on coupling and cascade specifically so a broken Wednesday is repairable, and
   it is writing against a blank.

## Still true, and unchanged by any of this

**No week has ever been cooked through the tool.** Every `Last cooked` is empty. A model
planner that cannot see a single cook is running on the same missing evidence the ranker
was — it just writes better sentences about the gap. The brief said nothing on its list
matters as much as one real week, and putting a model behind the planner has not moved
that by a day.
