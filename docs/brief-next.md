# Brief — what to build next

**For the next agent.** Read `docs/architecture.md` first; it records decisions made in
interview and they are settled. Then `docs/pantry-router-proposal.md` for why the product
exists, and `docs/step2-design.md` for the shopping list.

This brief is ordered. The order is the recommendation.

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
- 171 tests, standard library only.

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

## 2. Acquisition — the tool cannot find a recipe

`candidates.md` has three entries and a human put them there. The tool has never searched
for a recipe, never read a page it was not handed, never judged fit against the profile.
The proposal calls acquisition half the job. It is **0% built**, and it is the half that
makes the corpus grow.

**Shape:** given a gap in the week and the profile, search, fetch, capture with the existing
`onboard.py`, and land it in `candidates.md` with the source cited and a reason for the
reach. Most of the machinery exists and has never been pointed at the open web.

**Done means:** a session can fill a gap with a recipe nobody had bookmarked, the capture is
complete enough for the shopping list, and the candidate says where it came from.

**The bar:** never invent a recipe. Every candidate resolves to a real page. `1 lb chicken
breast` because soup usually has chicken is the failure this project exists to avoid.

## 3. Onboarding in the app

Adding a recipe means running a CLI with a URL. For a tool whose thesis is closing the gap
between the 15 you reach for and the 60 you like, **the growth path being a terminal command
is close to fatal** — and it breaks the onboardability requirement outright.

**Shape:** a box in the session. Paste a URL, get a captured recipe, land it in candidates.
`onboard.py` already does the work; this is a route and a form.

**Done means:** a person who has never opened a terminal can add a recipe.

## 4. Kroger

Sales in the briefing are fabricated and marked `demo`. There are no prices and no cart.
`docs/architecture.md` settles the shape: **the tool fills a cart and a human submits it.**
Never unattended spending.

Do it in two pieces, and the first is worth having alone:

- **Prices and promotions into the Step 0 briefing.** Makes the briefing real.
- **SKU matching and the cart write.** Canonical item → a product someone would actually
  buy. This is the hard part and it is where pack sizing (§7 of the proposal) and the
  `accepts:` tolerances finally earn their keep.

**Open:** how Kroger is talked to at all, and what the product does when the SKU match is
wrong or the API is down. Deliberately unanswered — answer it before building.

## 5. Session depth

Today: drop a meal, ask for another, four dials. Missing, roughly in order of how often it
would be wanted:

- **Swap *this* meal** for something similar, rather than drop-and-refill
- **See the recipe** without leaving the session
- **Servings per meal** — guests on Thursday only, which the profile explicitly asks for
- **Lock** a meal so a refill cannot touch it (the field exists, the UI does not)
- **Edit the profile in the app.** `profile.md` opens by saying correcting the file *is* the
  trust mechanism; in a hosted deployment that is currently unreachable.

## 6. Sides

The corpus is mains-only and every list says so. Vegetables are absent from the data, not
the diet — they get cooked and never written down. The household asked to reach a
dinners-only steady state first; that is reached. **Every grocery list this tool produces is
systematically short until this is solved.**

## 7. Read the decision log back

`decisions.jsonl` records every proposal, drop, dial change and outcome. **Nothing reads
it.** It was built because it cannot be backfilled — now use it: which reasons get accepted,
which get dropped, whether breadth is actually increasing. The metrics strip currently shows
five numbers computed off the corpus, not off behaviour.

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
