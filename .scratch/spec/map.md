# Map — the Grocery Router v1 spec

Label: `wayfinder:map`

## Destination

**A specification a future agent team can build Grocery Router v1 from, without reading
this repository.** It states what v1 is, what it refuses to be, the domain model, the
first-cut database schema, the session it delivers, and what makes it done.

The map is finished when nothing is left to decide before someone starts building.

## Notes

### What this is

A meal-planning and grocery-sourcing tool for **one household — Matt's**. The problem is
recall, not discovery: the household cooks roughly 15 of the ~32 recipes it knows and
likes. Closing that gap is the product.

The existing repository is a prototype that taught the domain. **v1 is a rebuild.** The
product thinking in `docs/` survives; the Python code does not.

### Where the prototype went

The working tree now holds **data and this map only**. Every line of prototype code and all
fifteen superseded documents live at the tag **`prototype`**, and nowhere else.

```sh
git show prototype:shop.py            # read one file
git checkout prototype -- docs/       # bring something back
git switch -c look-at-it prototype    # walk the whole thing
```

**Any file path named in a ticket refers to that tag**, not to the working tree. Ticket 12
exists because the knowledge inside that code — the ingredient grammar and the item table —
is the most expensive asset this project produced, and a rebuild will re-derive it badly
unless it is written down as a specification first.

### Skills every session should consult

- `/grilling` and `/domain-modeling` — the default for any ticket typed `grilling`.
- `/prototype` — for the two UI tickets.
- `/research` — for the one research ticket.

### Settled decisions

These came out of a five-round grilling session and are **not** re-openable. Re-opening one
is a regression.

| # | Decision | Costs us |
|---|---|---|
| 1 | **v1 is for one household.** No signup, no tenancy, no recipe bank | Every v2 concern must be deferred explicitly, not designed for |
| 2 | **Rebuild from scratch.** The old repo is reference, not a base | 331 tests and a working ingredient parser get re-earned |
| 3 | **Go**, one binary with subcommands (`serve`, plus the CLI equivalents) | No frontend story in the language — see decision 4 |
| 4 | **TypeScript + React frontend, embedded in the Go binary via `embed.FS`** | Two languages. **TS domain types must be generated from Go, never hand-written** |
| 5 | **Postgres via CloudNativePG, `instances: 1`**, on the single-node Talos cluster | Needs a storage class (Talos ships none) and an off-cluster backup target first |
| 6 | **Hosted on the cluster, CI/CD through GitHub** | Not a laptop tool any more; a broken deploy is a lost Sunday |
| 7 | **No login in v1** | Anyone on the network is every member, so feedback carries no name. State this cost in the spec |
| 8 | **No deterministic ranker.** One planner, and it is the model | A model outage means "try again", not a quietly worse week |
| 9 | **The model may plan, read messy text, and interpret free text. It may never touch the shopping list** | Step 2 is deterministic Go, enforced by an import-graph test |
| 10 | **Three model jobs in v1:** the planner, acquisition search, onboarding free-text parsing | Model-proposed profile revisions are deferred (see Out of scope) |
| 11 | **Acquisition uses the Anthropic server-side web search tool** (`web_search_20260209`) | Shortcut for v1. The scraped bank is v2 |
| 12 | **Guess loose, verify hard.** The finder only ever proposes URLs; capture refuses any page with no machine-readable schema.org recipe | A vegetarian main is still hard to find. Accepted |
| 13 | **Provenance on every row** — proven here, asserted at import, or acquired — replaces "membership is earned" | Onboarding recipes enter the corpus directly; `review.py`'s successor must split by provenance |
| 14 | **The existing repo data is imported**, as a spec'd and tested path | 27 recipes, 25 corpus rows, `profile.md`, `decisions.jsonl` |
| 15 | **The week is a pool with a required effort mix, not a day grid** | A wrong day label trains people to ignore the column |
| 16 | **Effort is two axes.** Active is capped on weeknights; passive is not capped at all | A single scalar wrongly excludes every slow-cooker meal |
| 17 | **`yield` has three shapes** — adult-equivalents, a portion count, and `per portion` | Asking how many a BLT serves is a question with no answer |
| 18 | **Sides are in v1**, and `sides.md`'s successor starts empty | Every list is short until the household types its sides in once |
| 19 | **Nothing is ever deleted.** A flop stays, with its reason | The most informative signal the system gets all week |
| 20 | **Every proposal, drop, dial change and outcome is logged** | A decision that was not recorded cannot be recovered |
| 21 | **The reason is the product** | Five true sentences that are all the same sentence are no reasons at all |
| 22 | **The name is Grocery Router** | Every existing doc says "Pantry Router" and must be updated |
| 23 | **Self-improvement is a first-class feature.** Every proposal carries a falsifiable prediction, every guessed value names the observation that would replace it, and scoring runs off the log with no interview | Reasons get a machine-checked half, which constrains the prose decision 21 calls the product. The cost lands in v1; the payoff arrives after many weeks of data |

### Traps the old repo already fell into

Every one was a real bug. They rhyme: **the failure is always a plausible value where there
should have been a gap.**

- **Two implementations of one thing drift.** Two ingredient parsers disagreed in three of
  twelve hard cases. This is why decision 4 mandates generated types.
- **Silent mis-merges beat loud gaps, and that is backwards.** `onion powder` resolved to
  `onion`; `dried thyme` to fresh thyme.
- **Unknown is not the extreme.** A recipe with no last-cooked date scored as *maximally*
  stale, above one measured dormant for six months.
- **A dial that does nothing.** The risk dial nudged a score in a fight candidates lose by
  design.
- **Over-reading a count.** Seven of twenty-five recipes were sandwiches; the model called
  it "a real, distinctive pattern". The household said *"just happens to be so."*
- **A model invented ingredient coupling** from a corpus index containing no ingredients,
  then picked a recipe because the invention justified it. This is decision 9's receipt.
- **A fact the household stated, that no code reads.** `profile.md` says leftovers go to
  lunches *and* second dinners. The planner claimed a second dinner off yield alone and was
  wrong the first week it ran. Ticket 10, finding 2.

### Definition of done for v1

v1 is complete when it **produces the household's weekly grocery list and improves that
process**. Sustained weekly use is the success measure that follows, not the gate.

## Decisions so far

<!-- one line per closed ticket -->

- **01 — domain model.** Every noun v1 has, with its type, its invariants, and what it is
  deliberately not a synonym for: `.scratch/spec/domain-model.md`. Glossary at `CONTEXT.md`.
  Ten term clusters resolved; `candidate`, `bank recipe`, `pack`, `family`, `outcome` and
  single-scalar `effort` are retired words. New nouns: **Retirement**, **Repeat**, **Verdict**,
  **Gap**. Ticket 10 findings 2 and 5 closed.
- **12 — self-improvement and signals.** The closed event set, the scoring run, and the list of
  what v1 can never measure: `.scratch/spec/signals.md`. **The off-plan rate — `Cook`s with no
  `Meal` — is the recall gap measured directly**, and it is the only metric that measures the
  product's reason for existing. Two capture surfaces, and the Session is the guaranteed one.
  Four signals added: the off-plan cook, a coarse effort three-state, the Step 2 defect count,
  and an `origin` marker on every guessable field. The scoring run reports and never acts.
  Ticket 10 finding 4 closed. Four of the fourteen decisions are marked *adopted, not grilled*.

## Not yet specified

- **How the list reaches the person in the aisle.** Decision 4 has no offline story, so
  ticking items off is a server round-trip. Whether that is acceptable is unresolved.
  The device is known: **an iPhone, in Safari.** The household owns no iPad and no Mac, so a
  native iOS app is not buildable here and was **considered and rejected on 8 August 2026**.
  The standing candidate is an **installed PWA** — a home-screen icon, full screen, and a
  service worker that holds the list offline. It keeps decision 4 exactly as written. Ticket
  02 or 03 chooses it or does not.
- **Whether a hard-won accept rate ever arrives.** Ticket 12 settled *how* v1 measures itself,
  and the arithmetic says the average reason kind needs about six months to reach a readable
  rate. Nothing shortens that, and no decision depends on it.
- **What the CLI subcommands read.** They need either database access or an export path,
  and that choice depends on the schema.
- **The degradation posture for a half-finished session.** The old repo stated it for the
  prep job, the planner and acquisition, and never for the session itself.
- **How TypeScript types get generated from Go**, and what fails the build when they drift.
  Depends on the shape of the API the frontend tickets settle.
- **Where a hard constraint is stored.** Ticket 01 established that a `Constraint` is not a
  profile `Claim` — it needs no evidence and is enforced in code. Which record holds it is a
  schema question and ticket 04 owns it.

## Out of scope

Ruled beyond this destination. These return only as a fresh effort.

- **Kroger — all of it.** Prices, SKU matching, promotions, the cart write. Moved to v1.5
  as a whole rather than half-built.
- **The scraped recipe bank.** v2. Requires a crawl, `robots.txt` compliance, and a corpus
  of somebody else's content to keep lawful and fresh.
- **Multi-tenancy, signup, identity, billing, row-level security.** v2. `docs/multi-tenancy.md`
  is the prior design and it is premature.
- **Model-proposed profile revisions.** Deferred. It is the most impressive thing this
  product could say to a household and the most likely to produce a confident claim from
  nothing.
- **The inferred pantry**, and parsing order-confirmation emails. Gated on Kroger.
- **The store comparison screen.** One store is theater.
- **Authentication.** Decision 7.
