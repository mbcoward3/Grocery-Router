# Candidates

> **Demo data.** See `demo/corpus.md`. The recipes are real; this household is not.

Recipes that have **not** been cooked and liked here. Read by `plan.py` alongside the
corpus, and proposed marked `[candidate]` so the household always knows which is which.

**Why this file exists separately.** `corpus.md` is proven-only, and that strict bar is
what makes it trustworthy as a planner input: everything in it is known-good, so surfacing
one is a recall problem and never a quality gamble. A candidate carries the gamble instead.
Mixing the two would quietly destroy the guarantee that makes the corpus worth having.

**Three states, and nothing is ever deleted.** Cooked and liked, a candidate moves to
`corpus.md`. Cooked and it flopped, it stays here with the reason recorded — at a small
corpus that is the most informative signal the system gets all week, and it must never be
silently dropped. Never cooked, it just waits.

| Recipe | Protein | Cuisine | Yield | Active | Passive | Proposed | Outcome |
|---|---|---|---|---|---|---|---|
| Chicken and dumplings | chicken | American | 6 AE | low | simmer | wk of 27 Jul | untested — cooked last week, not yet reported |
| Sheet pan chicken fajitas | chicken | Tex-Mex | 4 AE | low | 30m oven | — | untested |
| Parchment garlic butter salmon | fish | American | 1 AE | low | 30m oven | wk of 6 Jul | flopped 2026-07-11 — serves one, and scaling it to three meant three parcels and three times the fiddling |

## Notes

**Parchment garlic butter salmon is the worked example of why a flop is never deleted.**
It was proposed, cooked, and did not land — and the reason is specific enough to be useful:
the recipe serves 1 AE, so a household of three needs it tripled, and the tripling is what
made it annoying rather than the food. That is a fact about *this household and this
recipe*, it took a real week to learn, and the planner now knows not to propose it again.
Deleting the row would have thrown that away and left the recipe free to resurface.
