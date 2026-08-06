# Brief — what to build next

**For the next agent.** Read `docs/architecture.md` first; it records decisions made in
interview and they are settled. Then `docs/pantry-router-proposal.md` for why the product
exists, and `docs/step2-design.md` for the shopping list.

**All seven items are built.** What follows is kept in place rather than deleted, because
the *constraints* in each one were the expensive part and re-deriving them would cost more
than reading them. Each section now says what it became and what is still open under it.

Reports: `docs/model-planner-findings.md`, `docs/acquisition-findings.md`,
`docs/the-seven-findings.md`.

The one thing that has not changed is at the bottom of this file and it is still the most
important line in it.

---

## Where the project actually is

**Working, verified:**

- **The shopping list.** `shop.py`, deterministic end to end. 27 recipe files, 265
  ingredient lines, 0 unparseable, 0 unrecognised. Graded against a hand-built fixture.
- **The weekly session.** `app.py` + `web/index.html`. Feedback → corpus writes → metrics.
- **The write rules.** `pantry.py` refuses what the prose used to only assert.
- **A hosted demo.** https://huggingface.co/spaces/MattCow/pantry-router — the real Python
  under Pyodide, not a port.
- **The model planner.** `planner/`, two implementations behind `pantry.propose()`. See
  §1 below and `docs/model-planner-findings.md`.
- 317 tests, standard library only.

**Written but never run:** `.github/workflows/*`. CI has never executed; the deploy has
never fired. The Space was pushed by hand.

---

## 1. Put a model in the planner — **built**

*Kept in place because the constraints are the interesting part and re-deriving them would
be expensive. `docs/model-planner-findings.md` is the report.*

**What it is.** `planner/` holds the choice, the prompt, the model implementation and the
constraint checks; the ranker stayed in `pantry.py` as the now-public `pantry.rank()`.
Selection is an explicit argument, then `PANTRY_PLANNER`, then whether a key is present.
No key is a supported configuration, not a degraded one — it is what the demo and CI run,
and CI asserts it.

**The design line, and it is narrower than the brief implied:** *the model selects and
explains; it does not state facts.* It gets the corpus with a slug column, a computed
`days since` column and no ingredients, and returns slugs and reasons. Every other field is
read off the corpus row, so an invented one has nowhere to land. A slug resolving to
nothing is dropped and never nudged to its neighbour. A reason claiming recency about a row
with no date is dropped. The ranker fills whatever was dropped, which is what makes
refusing cheap: it costs a good reason, never a dinner.

**The hard constraints are code now** — `planner/constraints.py`, tested with no key and no
network. Family-edible is the exception and is documented as one: there is no honest
mechanical test, and corpus membership is the proxy.

**Still open here:** nobody has read a model-planned week and said whether the reasons land.
That is question 1 in the findings and it is the whole product claim.

## 2. Acquisition — **built**

`acquire.py`, plus `pantry.add_candidate()` as the write door and a *Find something new*
button in the session. Report: `docs/acquisition-findings.md`.

**Where it searches was the real decision.** Not a search engine: the sites the household
already cooks from, read off the corpus and candidates files. Nine of eleven expose the
WordPress REST search API — documented, public, no key, no scraping, and no way to surface
a page that does not exist. The surface grows on its own as recipes get promoted from new
sites. Widening past it is the household's call, not a default.

**No model runs in it**, deliberately. Every fit signal that matters is computable — peanut
off the capture, active off the source's stated prep time, protein and cuisine off the
ingredients, duplicates off the corpus. What a model would add is taste, and the household
supplies that by cooking the thing.

**Still open here:** the search is full-text and noisy, so relevance leans hard on the
protein gate — which means a genuinely vegetarian main gets refused along with the cake.
That is §6's to fix and is called out in the findings.

## 3. Onboarding in the app — **built**

A box in the session. It goes through `acquire.from_url`, which is the same capture, the
same constraint check and the same write door acquisition uses — a recipe somebody pasted
and a recipe the tool found are the same recipe and get the same row.

Hard constraints still apply to something a person chose; relevance does not. The
no-protein filter exists because full-text search returns cake, and applying it to a pasted
link would refuse a vegetarian main somebody deliberately went and found.

## 4. Kroger — **built, except the cart write**

`adapters/`, behind the interface `docs/architecture.md` reserved. Both open questions are
answered there rather than here, because they turned out to be architecture.

Prices and promotions reach the Step 0 briefing when credentials are set, and the invented
`DEMO` lines remain, still labelled, when they are not. SKU matching is deterministic in
`adapters/match.py` and this is where pack sizing and the `accepts:` tolerances finally
earn their keep.

**The rule that shapes all of it:** a match that is not confident is not made. `onion
powder` → `onion` cost one wasted vegetable in a list a human reads; the same error against
a cart costs money on an item nobody chose. A gap in a cart is a smaller failure than a
stranger's guess in it.

**Not built:** the cart write. It needs a user OAuth token from a real redirect through a
registered callback, which needs a hosted URL this project does not have. Everything up to
it exists and is tested; `./app.py` → *Price the cart* shows exactly what would be sent.
Writing that path untested against an API nobody has run is how a plausible thing that has
never worked gets committed.

## 5. Session depth — **built**

All five. Swap is one decision rather than a drop plus a refill, which matters to
`review.py`. Servings are per meal and reach the shopping list. Lock persists and
Reshuffle is what gives it meaning — and Reshuffle had to *decline* what it re-rolled,
because the ranker is deterministic and re-running it returns the same week. Recipes open
in place. The profile is editable, written whole, and rolled back if it stops parsing.

## 6. Sides — **machinery built, file empty**

`sides.md` is the store, `pantry.add_side` is the door, and sides flow through the same
capture, the same parser and the same aggregation as everything else — so a side sharing an
onion with a main comes out as one line. Three routes in: `./acquire.py --sides`,
`./onboard.py --url <link> --side`, and a box in the session that takes a name or a link.
The grocery list's *"sides are not included"* line is conditional now and stops appearing
once there are any.

**The file is empty and that is deliberate.** Seeding it would be inventing what this
household eats, which is the one thing this project refuses to do anywhere else. Every list
is still systematically short — the difference is that closing it is now a thing somebody
can do in thirty seconds rather than a thing that needs building.

## 7. Read the decision log back — **built**

`review.py`, and the same numbers under the session's metrics strip. Which reasons get
accepted and which get dropped, whether breadth is increasing, and the model against the
ranker on real weeks. Reason *kinds* are recorded now, because the sentence alone cannot
answer the first question — two meals stale at different distances are two sentences and
one kind.

**Still open here:** it is reading a log with one household-week in it. Every number is
correct and none of them means anything yet.

---

## Do not touch

Settled. Re-opening any of these is a regression, and most were paid for with a mistake.

- **Step 2 never calls a model.** Deterministic end to end. It has a receipt (see §1's trap)
  and `domain` purity is meant to become an import-graph test.
- **`promote()` is the only door into `corpus.md`,** and only for a candidate whose cook was
  kept. `onboard.py` used to violate this and now refuses.
- **No claim without a trace.** A profile claim with no evidence fails the write.
- **Files are the database.** SQLite was decided and then reversed on purpose —
  `docs/architecture.md` has the reasoning. No database until SaaS, and that is a real
  project, not a config change.
- **No protein quota.** An earlier draft had one; it was removed because the corpus is the
  household's own expression of what it wants and a quota fights that signal.
- **Effort is two numbers.** Active is capped at 20–30 min on weeknights; passive is not
  capped at all. A single scalar wrongly excluded every slow-cooker meal.
- **Output is a pool, not a grid.** Hard nights are unpredictable here, so binding meals to
  named days is wrong. A wrong day label trains people to ignore the column.
- **`yield` has three shapes** — adult-equivalents, a portion count, and `per portion` for
  dishes with no batch at all. Asking how many a BLT serves is a question with no answer.
- **A flop is never deleted.** At this corpus size it is the most informative signal the
  system gets all week.
- **Members are attribution, not accounts.** No login until hosting.
- **Demo mode must never write to the repo.** It has leaked once, and there is a test.
- **The reason is the product.** Five true sentences that are all the same sentence are no
  reasons at all.

## Traps this project has already fallen into

Every one of these was a real bug in this repo. They rhyme, and the rhyme is worth reading:
**the failure is always a plausible value where there should have been a gap.**

- **Two implementations of one thing drift.** `onboard.py` and `shop.py` both parse
  ingredients and disagree on what the item *is* in three of twelve hard cases. This is the
  argument for the model planner and the ranker sharing everything they can.
- **Silent mis-merges beat loud gaps, and that is backwards.** `onion powder` resolved to
  `onion` across thirteen lines — a fresh onion in the cart for a teaspoon of spice. Then
  `dried thyme` to fresh thyme across five more. A partial match is now only accepted when
  every word it leaves behind is noise.
- **Unknown is not the extreme.** A recipe with no last-cooked date was scoring as
  *maximally* stale, ranking above one measured dormant for six months.
- **A candidate inherited a corpus recipe's reason** and claimed membership it did not have.
- **A dial that does nothing.** Candidates lose every head-to-head by design, so the risk
  dial nudging a score changed no outcome. It reserves slots now.
- **Over-reading a count.** Seven of the twenty-five recipes were sandwiches. That looked
  like a signal; the household said *"I wouldn't put much weight to this, just happens to be
  so."* It is recorded in `profile.md` as an explicit non-pattern, and it is the failure mode
  this whole project is built to resist.

## Still open, and only the household can close it

- **Seven yields** — genuine batch dishes whose sources never stated servings.
- **Two portion rates** — how many enchiladas is an adult, and the same for sliders.
- **No week has ever been cooked through the tool.** Every `Last cooked` is empty. The
  ranker is running on staleness it cannot measure, and no proposal has ever been judged.
  **Nothing on this list matters as much as one real week.**

## Report

Same standard as the two onboarding briefs, which were the most useful artifacts those runs
produced. What worked, what did not, what the schema could not express, and the questions
worth asking the household — separated from the ones that answer themselves.
