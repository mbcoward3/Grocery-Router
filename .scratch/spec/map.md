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

### Standing rule

**Do not delete, archive or rewrite anything in this repository until ticket 12 is
resolved.** "Rebuild from scratch" is a decision about the code, not a licence to lose the
knowledge inside it. The ingredient grammar and the item table are the most expensive
assets here and they exist nowhere else.

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

### Definition of done for v1

v1 is complete when it **produces the household's weekly grocery list and improves that
process**. Sustained weekly use is the success measure that follows, not the gate.

## Decisions so far

<!-- one line per closed ticket -->

## Not yet specified

- **How the list reaches the person in the aisle.** Decision 4 has no offline story, so
  ticking items off is a server round-trip. Whether that is acceptable is unresolved.
- **How "improves that process" gets measured** once v1 ships. The old repo proposed
  distinct-recipes-per-quarter and cognitive-load counts; neither has a baseline.
- **What the CLI subcommands read.** They need either database access or an export path,
  and that choice depends on the schema.
- **The degradation posture for a half-finished session.** The old repo stated it for the
  prep job, the planner and acquisition, and never for the session itself.
- **How TypeScript types get generated from Go**, and what fails the build when they drift.
  Depends on the shape of the API the frontend tickets settle.
- **Whether the corpus needs an archived state**, or whether provenance plus outcomes is
  enough to stop surfacing a recipe.

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
