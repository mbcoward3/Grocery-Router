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

`Protein` and `Cuisine` are read off the recipe. **`Effort` is inferred and unverified** —
it's the system's guess at your weeknight scale, not something you said. Correct any line
that's wrong; that's the profile-correction mechanism working (§2).

`Last cooked` is empty because it was never recorded. It fills in as the tool gets used.
Until then the planner can't do staleness-based surfacing, which is a real limitation and
not a bug.

## Format

| Recipe | Protein | Cuisine | Effort | Last cooked | Notes |
|---|---|---|---|---|---|
| Chicken and biscuits casserole | chicken | American | medium | | thecountrycook.net |
| Sausage and peppers | pork | Italian-American | medium | | chefjeanpierre.com |
| Crock pot Italian beef sandwiches | beef | American | low | | iowagirleats.com; slow cooker, sandwich |
| Meatloaf | beef | American | medium | | natashaskitchen.com |
| Beef stew with carrots and potatoes | beef | American | high | | onceuponachef.com; braise |
| 3-ingredient teriyaki chicken | chicken | Japanese-ish | low | | tasty.co |
| Chicken veggie stir fry | chicken | Chinese-ish | low | | tasty.co |
| Easy salmon dinner | fish | American | low | | tasty.co |
| Chili | beef | American | medium | | |
| Enchiladas | beef | Tex-Mex | medium | | protein unconfirmed |
| Chicken noodle soup | chicken | American | medium | | |
| Tacos | beef | Tex-Mex | low | | ground beef + seasoning packet |
| Hamburgers | beef | American | low | | |
| Pork loin and rice | pork | American | medium | | |
| Cheesy pasta | beef | American | low | | ground beef, elbows, cream cheese, marinara |
| Biscuits and gravy | pork | American | low | | sausage; breakfast-for-dinner |
| BLT | pork | American | low | | bacon; sandwich, barely cooking |
| Meatball subs | beef | Italian-American | low | | sandwich |
| Chicken and dumplings | chicken | American | low | | lilluna.com; canned/biscuit-tube shortcut version |
| Sliders | beef | American | low | | sandwich |
| Beef dip Sammies | beef | American | low | | sandwich |
| Chicken chili | chicken | American | medium | | |
| Zuppa toscana | pork | Italian-American | medium | | sausage |
| Beef pot roast | beef | American | high | | braise |
| Tuna melt | fish | American | low | | sandwich |

**Effort** is your weeknight scale, not an objective one: `low` = you'd do it on a bad
Tuesday, `medium` = fine any night with a plan, `high` = weekend or company.

## Flagged for confirmation

- **Chicken and dumplings** was written `Chicken and dumplings?` in the source. The
  question mark reads as uncertainty about whether it's actually made and liked. If it's
  aspirational rather than proven, it belongs out of the corpus — say so and it moves.
- **Enchiladas** protein is a guess (beef, to match the rest of the Tex-Mex here).
