# Corpus

Recipes this household has cooked and liked. Read by `plan.py` on every run.

**Membership is earned (§4).** Nothing goes in here until it's been cooked and liked.
That strict bar is what makes the corpus trustworthy as a planner input: everything in it
is known-good, so surfacing one is a recall problem and never a quality gamble. A recipe
bookmarked but never made does not belong here — that's a candidate, and the planner
proposes those on its own, marked `[candidate]`.

**No ingredient lists, deliberately.** They feed the shopping list, which is a separate
deterministic step. The planner doesn't need them. They live in `recipes/<slug>.md`, one
file per recipe, loaded only for the meals a week actually uses (`docs/step2-design.md` §1).

**`Yield` is the one field that moved in from `recipes/`.** The planner needs it to reason
about leftovers — a cold run once proposed a double batch of a recipe that already served
8. It is in adult-equivalents and it is filled in only where the source states servings.
**`unknown` is a real value, not a gap to fill**: 16 of these came from a screenshot or a
typed note that never said how many it feeds, and a plausible guess is worse than a blank.
Answering those 16 is the highest-value thing anyone can do to this file — see
`docs/onboarding-findings.md` §4.

**Growing it is the point, not a chore.** Add a line whenever one comes back to you, and
whenever a candidate gets cooked and kept.

## Provenance

Seeded from `Recipes.pdf` — 25 recipes: 8 saved links, 6 typed out with ingredients, 11
saved as screenshots. All entered the corpus directly on the proven-or-attested bar (§4),
since they were collected as things this household actually makes.

`Protein` and `Cuisine` are read off the recipe. **`Active` and `Passive` are inferred and
unverified** — the system's guess, not something you said. Correct any line that's wrong;
that's the profile-correction mechanism working (§2).

`Last cooked` is empty because it was never recorded. It fills in as the tool gets used,
about five entries per session. It is **not** a precondition for the tool working: this
table is small enough to hand a planner in full on every run, so it can propose across all
24 from the first week. Dates refine the ranking; they don't unlock the mechanism.

## Format

| Recipe | Protein | Cuisine | Yield | Active | Passive | Last cooked | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Chicken and biscuits casserole | chicken | American | 6 AE | med | ~35m oven |  | thecountrycook.net |
| Sausage and peppers | pork | Italian-American | 4 AE | med | — |  | chefjeanpierre.com |
| Crock pot Italian beef sandwiches | beef | American | 8 AE | low | hours, slow cooker |  | iowagirleats.com |
| Meatloaf | beef | American | 8 AE | med | ~60m oven |  | natashaskitchen.com |
| Beef stew with carrots and potatoes | beef | American | 6 AE | med | long simmer |  | onceuponachef.com; braise |
| 3-ingredient teriyaki chicken | chicken | Japanese-ish | 4 AE | low | — |  | tasty.co; check sauce for peanut |
| Chicken veggie stir fry | chicken | Chinese-ish | 6 AE | med | — |  | tasty.co; check sauce for peanut |
| Easy salmon dinner | fish | American | 2 AE | low | — |  | tasty.co |
| Chili | beef | American | unknown | low | simmer |  |  |
| Enchiladas | beef | Tex-Mex | unknown | med | ~25m oven |  | protein confirmed beef |
| Chicken noodle soup | chicken | American | unknown | med | simmer |  |  |
| Tacos | beef | Tex-Mex | unknown | low | — |  | ground beef + seasoning packet |
| Hamburgers | beef | American | unknown | low | — |  |  |
| Pork loin and rice | pork | American | unknown | low | oven |  |  |
| Cheesy pasta | beef | American | unknown | low | — |  | ground beef, elbows, cream cheese, marinara |
| Biscuits and gravy | pork | American | unknown | low | short oven |  | sausage; breakfast-for-dinner |
| BLT | pork | American | unknown | low | — |  | bacon; barely cooking |
| Meatball subs | beef | Italian-American | unknown | low | — |  |  |
| Sliders | beef | American | unknown | low | short oven |  |  |
| Beef dip Sammies | beef | American | 8 AE | low | slow cooker *or* stovetop braise — see variants |  | 3 lb roast |
| Chicken chili | chicken | American | unknown | low | simmer |  | cream cheese |
| Zuppa toscana | pork | Italian-American | unknown | med | simmer |  | sausage |
| Beef pot roast | beef | American | unknown | med | very long |  | braise |
| Tuna melt | fish | American | unknown | low | — |  |  |

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
