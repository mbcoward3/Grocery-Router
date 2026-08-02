# Corpus

Recipes this household has cooked and liked. Read by `plan.py` on every run.

**Membership is earned (§4).** Nothing goes in here until it's been cooked and liked.
That strict bar is what makes the corpus trustworthy as a planner input: everything in it
is known-good, so surfacing one is a recall problem and never a quality gamble. A recipe
bookmarked but never made does not belong here — that's a candidate, and the planner
proposes those on its own, marked `[candidate]`.

**No ingredient lists, deliberately.** They feed the shopping list, which is a separate
deterministic step. The planner doesn't need them.

**Growing it is the point, not a chore.** Add a line whenever one comes back to you, and
whenever a candidate gets cooked and kept.

## Provenance

Seeded from `Recipes.pdf` — 25 recipes: 8 saved links, 6 typed out with ingredients, 11
saved as screenshots. All entered the corpus directly on the proven-or-attested bar (§4),
since they were collected as things this household actually makes.

`Protein` and `Cuisine` are read off the recipe. **`Active` and `Passive` are inferred and
unverified** — the system's guess, not something you said. Correct any line that's wrong;
that's the profile-correction mechanism working (§2).

`Last cooked` is empty because it was never recorded. It fills in as the tool gets used.
Until then the planner can't do staleness-based surfacing, which is a real limitation and
not a bug.

## Format

| Recipe | Protein | Cuisine | Active | Passive | Last cooked | Notes |
|---|---|---|---|---|---|---|
| Chicken and biscuits casserole | chicken | American | med | ~35m oven | | thecountrycook.net |
| Sausage and peppers | pork | Italian-American | med | — | | chefjeanpierre.com |
| Crock pot Italian beef sandwiches | beef | American | low | hours, slow cooker | | iowagirleats.com |
| Meatloaf | beef | American | med | ~60m oven | | natashaskitchen.com |
| Beef stew with carrots and potatoes | beef | American | med | long simmer | | onceuponachef.com; braise |
| 3-ingredient teriyaki chicken | chicken | Japanese-ish | low | — | | tasty.co; check sauce for peanut |
| Chicken veggie stir fry | chicken | Chinese-ish | med | — | | tasty.co; check sauce for peanut |
| Easy salmon dinner | fish | American | low | — | | tasty.co |
| Chili | beef | American | low | simmer | | |
| Enchiladas | beef | Tex-Mex | med | ~25m oven | | protein confirmed beef |
| Chicken noodle soup | chicken | American | med | simmer | | |
| Tacos | beef | Tex-Mex | low | — | | ground beef + seasoning packet |
| Hamburgers | beef | American | low | — | | |
| Pork loin and rice | pork | American | low | oven | | |
| Cheesy pasta | beef | American | low | — | | ground beef, elbows, cream cheese, marinara |
| Biscuits and gravy | pork | American | low | short oven | | sausage; breakfast-for-dinner |
| BLT | pork | American | low | — | | bacon; barely cooking |
| Meatball subs | beef | Italian-American | low | — | | |
| Chicken and dumplings | chicken | American | low | simmer | | lilluna.com; **status uncertain — see below** |
| Sliders | beef | American | low | short oven | | |
| Beef dip Sammies | beef | American | low | slow cooker | | |
| Chicken chili | chicken | American | low | simmer | | cream cheese |
| Zuppa toscana | pork | Italian-American | med | simmer | | sausage |
| Beef pot roast | beef | American | med | very long | | braise |
| Tuna melt | fish | American | low | — | | |

**Effort is two numbers, and only the first one is capped.**

- **Active** — hands-on time. `low` ≈ under 15 min, `med` ≈ 15–30, `high` ≈ 30+.
  The household ceiling is 20–30 min active on weeknights, so `low` and `med` are both
  weeknight-eligible and `high` is weekend-only.
- **Passive** — unattended time. **Not capped.** Slow cookers, braises and long oven times
  are fine on a weeknight.

This split matters: under a single effort scalar, beef stew and pot roast rated `high` and
would have been wrongly filtered out of every weeknight. They're `med` active — the length
is all unattended.

## Flagged for confirmation

- **Chicken and dumplings** was written `Chicken and dumplings?` in the source, and the
  household doesn't remember what the question mark meant. Left in the corpus, marked
  uncertain. Resolve it the first time it comes up in a real week: if it turns out to be
  aspirational rather than cooked-and-liked, it moves out to candidates.
- ~~Enchiladas protein~~ — confirmed beef.

## Counts, as of seeding

Protein: **beef 12**, chicken 6, pork 5, fish 2. Cuisine: ~18 American, 3
Italian-American, 2 Tex-Mex, 2 loosely Asian.

Beef at 48% is above what the household wants — see the soft cap in `profile.md`. The
cuisine spread is recorded as fact, not as a problem; nobody has said they want it widened.
