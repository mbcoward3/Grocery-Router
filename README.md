# Grocery Router

**Decide the week, confirm the list, source the items.**

The household cooks a fraction of what it enjoys. Roughly 32 recipes have been tried and
liked; under the stress of picking a week, about 15 surface. The gap between 15 and 32 is
the product.

## Status: the deterministic core runs. The interface is not built yet.

`gr/` holds the working half of the tool: it plans a week, builds a real shopping list from
the recipe files, and writes both into `weeks/<sunday>.md`. There is no user interface yet —
the web app is designed separately and lands on top of this.

**Start at [`.scratch/spec/map.md`](.scratch/spec/map.md)** — the destination, the settled
decisions with their costs, the traps this project already paid for, and the open tickets.

### The one rule the code is built around

**The model picks meals. The model never produces a line of the shopping list.**

That is a process boundary, not a convention. The planner subprocess runs with `--tools ""`,
so it has no file access and physically cannot open a recipe file. Its prompt carries
`corpus.md`, `profile.md`, `candidates.md` and `sides.md`, and none of those holds an
ingredient list. Every quantity, conversion, merge and aisle on the list is arithmetic in
`gr/`, checkable line by line against the recipe files.

### Running it

Python 3.12, no dependencies. The planner needs the `claude` CLI on `PATH`; without it the
week is still planned, by code, and the week file says so.

```sh
python3 -m unittest discover -s tests    # 81 tests, including the sixteen regressions
python3 -m gr.audit                      # parse every recipe, print every unresolved line
```

To plan a week from Python:

```python
from gr import session
week = session.plan_week(nights=5, guests=0)   # writes weeks/<sunday>.md
```

One planner call costs roughly $0.15–$0.30 and takes about a minute.

### What lives where

| | |
|---|---|
| `gr/items.py` | The item table and the mis-merge rule. The most expensive knowledge here |
| `gr/parse.py` | One ingredient line in, a quantity and an item out — or a refusal with a reason |
| `gr/recipes.py` | Recipe files, the four yield shapes, and the multiplier each one earns |
| `gr/shoplist.py` | Aggregation, unit reconciliation, staple routing, the unknown channel |
| `gr/planner.py` | The one model call, and every check that refuses to trust it |
| `gr/weekfile.py` | `weeks/<sunday>.md` — the week, the list, and the ticks, in one file |
| `gr/session.py` | Plan, build, write. The seam an interface sits on |
| `gr/notices.py` | The five gaps `profile.md` names, computed from live data |
| `gr/audit.py` | `python3 -m gr.audit` — every ingredient line with no `items.md` row |

Current resolution: **250 of 254 ingredient lines (98.4%)**. All four remaining misses are
correct refusals — two ingredients on one line, an unstated choice between two, and one
malformed source line. A refused line is printed on the list in full, never dropped.

## What is here

| | |
|---|---|
| `corpus.md` | Recipes cooked and liked. 25 rows |
| `recipes/` | The ingredients, verbatim. 27 files, 265 lines |
| `items.md` | The canonical item table — aisle, unit, conversion, match tolerances |
| `profile.md` | The household. One rule: **no claim without a trace** |
| `candidates.md` | Found, not yet cooked |
| `sides.md` | Empty on purpose. Seeding it would invent what this household eats |
| `sources/` | The original inputs the corpus was read out of |
| `decisions.jsonl` | Every proposal, drop and outcome. Cannot be backfilled at any price |
| `weeks/` | Planned weeks |
| `.scratch/spec/` | The map and its tickets |

## Where the prototype went

**The `prototype` tag does not exist on this remote.** `git tag -l` is empty and
`git ls-remote origin` shows no tags. The 265-line parser it held — described elsewhere in
this repository as the most expensive knowledge in the old codebase — is not recoverable
from here, and the captain's decision is that it is deliberately not being recovered.

It has been re-derived instead, in `gr/parse.py` and `gr/items.py`, and it reaches 98.4% on
the same 254 lines. Any document in `.scratch/spec/` that tells you to run
`git show prototype:shop.py` is describing something that is gone.
