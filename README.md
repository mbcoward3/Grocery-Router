# Grocery Router

**Decide the week, confirm the list, source the items.**

The household cooks a fraction of what it enjoys. Roughly 32 recipes have been tried and
liked; under the stress of picking a week, about 15 surface. The gap between 15 and 32 is
the product.

## Status: being specified, not built

There is no application here yet. A prototype was built, it taught the domain, and it is
being replaced. What remains in this directory is the part worth keeping.

**Start at [`.scratch/spec/map.md`](.scratch/spec/map.md)** — the destination, the settled
decisions with their costs, the traps this project already paid for, and the open tickets.
Work one ticket per session.

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

Everything else is at the tag **`prototype`** — about 33MB of Python, a browser build, a
container, and fifteen documents that had begun to contradict each other.

```sh
git show prototype:shop.py            # read one file
git switch -c look-at-it prototype    # walk the whole thing
```

It is kept for one reason, and the reason has a ticket: `shop.py` parses all 265 ingredient
lines with zero failures, and that grammar exists nowhere else in writing.
