# Brief — the recipe onboarding tool

**For the next agent.** Read this, then `docs/step2-design.md`, `corpus.md`, `profile.md`,
and two or three files in `recipes/` before writing anything.

---

## What you are building

A tool that takes a raw, unorganised recipe and produces two things:

1. **`recipes/<slug>.md`** — the content: ingredients with quantities, yield, source.
2. **A row in `corpus.md`** — the planning metadata: protein, cuisine, active, passive.

Those are deliberately separate stores and must stay separate. `corpus.md` is a thin index
that goes into the planner's context on *every* run, so it stays one line per recipe.
`recipes/` is bulk content that is never sent to a model and is loaded only for the meals
actually chosen in a given week. Merging them breaks both properties. `docs/step2-design.md`
§1 has the full reasoning.

Right now this job is done by hand, badly, one recipe at a time. Twenty-three of the
household's twenty-five recipes have no content file at all.

## Three input modalities

The tool must handle all three, because the real corpus is all three:

| Input | Approach | Reality |
|---|---|---|
| **A URL** | Fetch, extract the recipe | Some sites 403 on automated fetch — `thecountrycook.net` already did. Handle the failure; do not pretend it worked. |
| **Loose text** | Parse an ingredient list that may have no quantities and no method | The household's own typed notes. `Onion`, `Buns`, `Salt` with no amounts is normal here. |
| **A screenshot** | Read the image | Eleven recipes exist only as photos of other apps and websites. |

## Hard rules

**Never invent.** This is the whole of it. A missing quantity is recorded as missing, not
estimated. An unreadable screenshot is reported as unreadable. A recipe whose yield is not
stated gets `yield: unknown`, and that is a perfectly good outcome. The household can
answer any of these in two seconds; a plausible-looking wrong number they never notice
costs them a meal.

**Mark inferred metadata as inferred.** `protein` and `cuisine` can be read off a recipe
confidently. `active` and `passive` effort usually cannot — they are the system's guess at
this household's scale, and `corpus.md` already carries them as explicitly unverified.
Keep that distinction visible.

**Effort is two axes, not one.** `active` is hands-on time; `passive` is unattended
(slow cooker, oven, braise, marinating). Only active time is capped. A four-hour pot roast
with twenty minutes of searing is a weeknight meal here. Collapsing these to one number
silently deletes this household's easiest dinners — that bug already happened once.

**Yield matters more than it looks.** Record it in adult-equivalents where the source
states servings. The planner uses it to reason about leftovers, and it has already gone
wrong once: a cold planner run proposed "double batch" for a recipe that already served 8,
because nothing in the corpus told it the yield.

**Flag peanut.** The household has a peanut allergy at ingredient level — peanuts, peanut
butter, peanut sauce, peanut oil. Nothing currently in the corpus contains any, but
bought sauces are where it would hide. Surface it rather than filtering silently.

**Preserve the ingredient line verbatim** alongside anything you parse out of it. The
downstream list builder has to handle `1 (14.5 oz) can beef broth` and `3 cups bell
peppers, sliced`; if onboarding normalises those early and gets it wrong, the original is
gone.

**These twenty-three enter the corpus directly.** They are things the household already
cooks — proven-or-attested, `corpus.md` §"Membership is earned". They do not pass through
candidate. Do not add `last cooked` dates; none were ever recorded and inventing them
would poison the planner's staleness logic.

## The live test set

`sources/Recipes.pdf` is the household's real saved recipe document, committed so this is
reproducible. Twenty-three recipes in `corpus.md` have no file in `recipes/` yet. They are
your test set, and they are conveniently split across all three modalities.

**Six URLs** (in the PDF's text layer, page 1):

- Chicken and biscuits casserole — `thecountrycook.net/chicken-and-biscuits-casserole/` *(403s on fetch — a real case, not a bug to hide)*
- Meatloaf — `natashaskitchen.com/meatloaf-recipe/`
- Beef stew with carrots and potatoes — `onceuponachef.com/recipes/beef-stew-with-carrots-potatoes.html`
- 3-ingredient teriyaki chicken — `tasty.co/recipe/3-ingredient-teriyaki-chicken`
- Chicken veggie stir fry — `tasty.co/recipe/chicken-veggie-stir-fry`
- Easy salmon dinner — `tasty.co/recipe/easy-salmon-dinner`

**Six typed as loose text** (PDF pages 9–11) — Tacos, Hamburgers, Pork loin and rice,
Cheesy pasta, Biscuits and gravy, BLT. Most have partial or no quantities. `Pork loin and
rice` lists `Soup sauce`, which may be a typo — ask, do not guess.

**Eleven as screenshots only** — Chili, Enchiladas, Chicken noodle soup, Meatball subs,
Chicken and dumplings, Sliders, Beef dip Sammies, Chicken chili, Zuppa toscana, Beef pot
roast, Tuna melt. The PDF's text layer names them but carries none of their content; the
images are embedded in the PDF.

Two already-done examples to match: `recipes/crock-pot-italian-beef.md` (from a URL) and
`recipes/sausage-and-peppers.md` (from a URL). Both were fetched by hand this session.

## Run it live

Do not stop at a tool that passes unit tests. **Run it over all twenty-three and commit
the output**, then report what actually happened:

- How many produced a complete file, and how many are partial.
- Per modality — URL, text, screenshot — the success rate. These will differ a lot and the
  difference is the useful finding.
- Every recipe that needs a human answer, with the specific question. A short list of
  precise questions is a good outcome; a corpus of confident guesses is a bad one.
- Anything that made you want to change the schema. The schema is four days old and was
  designed against four recipes; twenty-three will stress it.

The roadmap calls messy ingest "Phase 5 — report the parse-quality gap honestly." That
honesty requirement is the deliverable, not a caveat on it.

## Things that are already known to be true

Do not re-derive these; they cost a household interview to establish.

- Weeknight ceiling is 20–30 minutes **active**. Passive time is uncapped.
- Hard nights are unpredictable week to week, so nothing should be bound to named days.
- Leftovers do double duty — second dinners *and* lunches. Five nights is not five cooks.
- The household cooks with shortcuts on purpose: seasoning packets, canned soup, biscuit
  tubes, cream cheese. That is a preference to serve, not a deficiency to correct.
- No protein quota exists. An earlier draft added one and it was removed — the corpus is
  the household's own expression of what it wants, and it self-corrects.
- The corpus is mains-only. Sides are deliberately deferred; do not start capturing them.
