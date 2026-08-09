# Project agent memory

This file is the project's committed home for project-intrinsic agent knowledge: build, test, release, architecture, and sharp-edge notes that should travel with the code.

- Add durable project-specific notes here as they are discovered through real work.

## The rule everything else follows

**The model picks meals. The model never produces a line of the shopping list.**

Enforced as a process boundary, not a convention: `gr/planner.py` launches `claude -p` with
`--tools ""`, so the planner has no file access and cannot open a recipe file. Its prompt
carries no ingredient list. Every quantity, unit conversion, merge and aisle is code in
`gr/`. Do not move arithmetic into a prompt, and do not give the planner tools.

Three real bugs this prevents are recorded at `.scratch/spec/map.md:79-96` and
`items.md:150-182`. Read them before changing `gr/items.py`.

## Where the truth lives

The markdown files **are** the state. No database, no cache. `profile.md:3-4` calls
correcting them the trust mechanism, which only works if nothing keeps a second copy.
`weeks/<sunday>.md` holds the week, its shopping list and the shopper's `- [x]` ticks in one
file. A week is Sunday to Saturday, named by its Sunday.

The `Slug` column in `corpus.md` and `candidates.md` is the join to `recipes/<slug>.md`.
It is data because deriving it from the title fails silently on
*Crock pot Italian beef sandwiches* → `recipes/crock-pot-italian-beef.md`. Never guess it.

## Commands

```sh
python3 -m unittest discover -s tests    # no dependencies; Python 3.12 stdlib only
python3 -m gr.audit                      # every ingredient line with no items.md row
```

`tests/test_core.py::TestSixteenRegressions` is sixteen cases off the `items.md:150-182`
notes, one per bug this project already paid for. Treat a failure there as a reintroduced
bug, not a stale test.

## Sharp edges

- **`items.md` grows on parse failure.** An unresolved line is the mechanism working — it
  gets printed on the list in full. A mis-merge is worse, because it is silent. When adding
  a noise word to `gr/items.py`, check it does not name an aisle: `dried`, `fresh`, `powder`,
  `ground`, `sauce`, `juice` and `and`/`or` are excluded on purpose.
- **Never invent a side, a yield, or a last-cooked date.** `sides.md` is empty on purpose,
  seven yields are genuinely `unknown`, and no last-cooked date exists anywhere. Code strips
  any recency claim the planner writes (`gr/planner.py:_drop_recency`).
- **`per portion` recipes are not scaled.** Multiplying `2 lb ground beef` by the household
  size once ordered 5 lb of beef for burger night; `corpus.md:13-14` calls that 2 lb "a
  convenience, not a batch". Those recipe files carry `scaling: unscaled`.
- **The `prototype` tag does not exist on this remote**, though `README.md`, `map.md:29-38`
  and `.scratch/spec/issues/12-ingredient-grammar-and-items.md` all tell you to read it. The
  parser was re-derived in `gr/`.
- **Two spec files are numbered 12** in `.scratch/spec/issues/`, and "ticket 12" means
  different ones in different places.
- **Driving the `claude` CLI:** capture stdout alone (stderr corrupts the JSON), redirect
  stdin from `/dev/null`, read `structured_output`, and trust the exit code and `is_error` —
  never `subtype`, which has been observed saying `success` alongside a 404.

## Maintaining this file

Keep this file for knowledge useful to almost every future agent session in this project.
Do not repeat what the codebase already shows; point to the authoritative file or command instead.
Prefer rewriting or pruning existing entries over appending new ones.
When updating this file, preserve this bar for all agents and keep entries concise.
