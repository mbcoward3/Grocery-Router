# Corpus

Recipes this household has cooked and liked. Read by `plan.py` on every run.

> **This is the demo household, and none of it is real.** The recipes are genuine and
> so are their ingredients; the cooking history is invented so the tool has something
> to demonstrate. The actual household's corpus is not in this directory and never
> reaches a hosted deployment.

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
8. It takes three shapes, and which one applies is a fact about the dish
(`docs/step2-design.md` §2.5):

- **`N AE`** — a batch dish whose source states servings, in adult-equivalents.
- **`N <things>`** — a batch measured in portions rather than people, like `8 enchiladas`.
  Real information; it needs one number from the household to become AE.
- **`per portion`** — no batch exists. Burgers, tacos and BLTs scale with the headcount,
  which is *why* no source states a yield for them. Asking is a question with no answer.

**`unknown` is a real value, not a gap to fill**, and it now means only one thing: a genuine
batch dish whose source never said. Seven rows are in that state and the household is the
only thing that can answer them. Five more were closed by going back to the sources, and
three were never questions at all.

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
| Chicken and biscuits casserole | chicken | American | 6 AE | med | ~35m oven | 2026-06-19 | thecountrycook.net |
| Sausage and peppers | pork | Italian-American | 4 AE | med | — | 2026-06-14 | chefjeanpierre.com |
| Crock pot Italian beef sandwiches | beef | American | 8 AE | low | hours, slow cooker |  | iowagirleats.com |
| Meatloaf | beef | American | 8 AE | med | ~60m oven | 2026-07-20 | natashaskitchen.com |
| Beef stew with carrots and potatoes | beef | American | 6 AE | med | long simmer | 2026-02-18 | onceuponachef.com; braise |
| 3-ingredient teriyaki chicken | chicken | Japanese-ish | 4 AE | low | — |  | tasty.co; check sauce for peanut |
| Chicken veggie stir fry | chicken | Chinese-ish | 6 AE | med | — | 2026-06-05 | tasty.co; check sauce for peanut |
| Easy salmon dinner | fish | American | 2 AE | low | — |  | tasty.co |
| Chili | beef | American | 4 AE | low | 20m simmer | 2026-07-30 | julieseatsandtreats.com |
| Enchiladas | beef | Tex-Mex | 8 enchiladas | med | 30-35m oven | 2025-10-21 | southernbite.com; protein confirmed beef |
| Chicken noodle soup | chicken | American | unknown | med | rotisserie *or* whole bird — see variants | 2026-01-03 |  |
| Tacos | beef | Tex-Mex | per portion | low | — | 2026-07-27 | ground beef + seasoning packet |
| Hamburgers | beef | American | per portion | low | — | 2026-07-23 |  |
| Pork loin and rice | pork | American | unknown | low | oven | 2025-12-12 |  |
| Cheesy pasta | beef | American | unknown | low | — | 2026-07-25 | ground beef, elbows, cream cheese, marinara |
| Biscuits and gravy | pork | American | unknown | low | short oven | 2026-05-29 | sausage; breakfast-for-dinner |
| BLT | pork | American | per portion | low | — | 2026-08-01 | bacon; barely cooking |
| Meatball subs | beef | Italian-American | 4 AE | low | frozen *or* homemade — see variants | 2026-07-12 | spendwithpennies.com |
| Sliders | beef | American | 24 sliders | low | 12-15m oven | 2026-07-16 | natashaskitchen.com |
| Chicken chili | chicken | American | unknown | low | simmer |  | cream cheese |
| Zuppa toscana | pork | Italian-American | unknown | med | simmer | 2026-01-26 | sausage |
| Beef pot roast | beef | American | 8 AE | med | 8-10 hr slow cooker | 2025-11-19 | dinnerthendessert.com |
| Tuna melt | fish | American | unknown | low | — | 2025-09-08 |  |

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

- ~~Chicken and dumplings~~ — **moved to `candidates.md`** by household decision. Rather
  than leave one unproven entry weakening the guarantee that everything here is
  cooked-and-liked, it waits until it is actually cooked.
- ~~Enchiladas protein~~ — confirmed beef.
- **Seven yields still say `unknown`**, and every one is a genuine batch dish whose source
  never stated servings: chicken noodle soup, pork loin and rice, cheesy pasta, biscuits
  and gravy, chicken chili, zuppa toscana, tuna melt. Nothing but the household can close
  these. Two more are portion counts waiting on one number each — *how many enchiladas is
  an adult*, and the same for sliders.

## Counts

Protein: **beef 12**, chicken 5, pork 5, fish 2, across 24. Cuisine: 17 American, 3
Italian-American, 2 Tex-Mex, 2 loosely Asian.

Beef at 50% is recorded, not corrected. **There is no quota** — the corpus is the
household's own expression of what it wants, and adding chicken recipes shifts the mix on
its own (`profile.md`). The cuisine spread is likewise a fact, not a problem; nobody has
said they want it widened.
