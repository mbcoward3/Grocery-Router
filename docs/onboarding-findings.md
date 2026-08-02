# What onboarding twenty-three recipes turned up

**Hand-written analysis of the run in [`onboarding-run.md`](onboarding-run.md), which is
generated.** The numbers there; what they mean here.

All twenty-three recipes in `corpus.md` that had no content file now have one, and every
corpus row now carries a yield. Nothing failed outright. That headline hides the real
finding, which is that **the three modalities fail in three different ways, and only one
of them fails visibly.**

---

## 1. Per-modality, and why the numbers differ

| | URL | Loose text | Screenshot |
|---|---|---|---|
| Recipes | 6 | 6 | 11 |
| Complete capture | 6 | 6 | 6 |
| Partial capture | 0 | 0 | 5 |
| Ingredient lines | 76 | 42 | 111 |
| Lines with a stated quantity | 70 (92%) | 6 (**14%**) | 95 (86%) |
| Yield stated by the source | 6/6 | **0/6** | 1/11 |
| Open questions raised | 0 | 7 | 27 |

**URL is the only modality that is actually solved.** Six for six, structured
`schema.org/Recipe` data on every page, quantities and servings and prep and cook times
included. If the household's future recipes arrive as links, this problem is finished.

**Loose text captures everything the household wrote and nothing more.** Six for six with
no truncation — but only 14% of lines carry a quantity, and not one of the six states a
yield. That is not a parser failure. `Onion`, `Buns`, `Mayo` is what they typed, because
these are the six recipes they know by heart. The gap is real and it is in the source, and
it will surface at the shopping list, not here: *buy an onion* is fine, *buy 2 lb of ground
beef* needs the number.

**Screenshots are where the losses hide.** 86% of the lines that were captured carry a
quantity — as good as a URL — but 5 of 11 captures are provably short of content, and only
one of the eleven states a yield. The failure mode is not bad reading, it is **bad
framing**: someone screenshotting a phone captures what is on the screen, and the recipe
does not fit on the screen.

Concretely, from the eleven:

- **Chicken noodle soup** was captured in one screenshot that stops mid-list. The visible
  lines end at the ramen noodles, and *there is no chicken in the recipe at all* — though
  the page's own blurb says "tender chicken". A recipe whose title ingredient is missing.
- **Chili** and **Tuna melt** were captured in two screenshots each, at different scroll
  positions. Whether anything fell in the seam cannot be told from the images.
- **Enchiladas** starts at the first ingredient with the "Ingredients" heading cut off
  above it, so there is no way to confirm the list starts where the screenshot does.
- **Beef dip Sammies** is a photograph of a handwritten card, cut off mid-sentence at
  "then turn down to". No quantity appears on it at all.

None of these are unreadable. Every one is *readable and incomplete*, which is worse,
because a complete-looking file is what a shopping list will trust.

## 2. The screenshot path is not automated here, and that matters

`onboard.py --image` calls a vision model, and there was no `ANTHROPIC_API_KEY` in this
environment, so **that code path is written but was not exercised on this run.** The
eleven screenshots were transcribed by a model reading the rendered PDF pages, and the
transcriptions are committed verbatim in
[`sources/inputs/transcripts/`](../sources/inputs/transcripts/) so the run reproduces from
the committed PDF alone.

This is worth stating plainly rather than burying: **the URL and text paths are
deterministic code, the screenshot path has a model in it.** Which is exactly why the
transcript is a committed artifact — a model in the loop needs its output to be
inspectable by the household, not just consumed by the next stage.

## 3. `thecountrycook.net` did not 403

The brief flagged it as a known 403. With a browser `User-Agent` it returns 200 and
serves complete recipe data; so do `natashaskitchen.com` and `onceuponachef.com`, both of
which **do** 403 on Python's default agent. All three were tested both ways.

The 403 was real, it was just a property of the request, not of the site. The failure
path is still implemented and still needed — the tool reports a block as a block, does not
retry a 403, and asks for a screenshot instead — but it did not fire on any of these six.

## 4. Thirty-four questions, and what they are worth

The full list is in the run report. They fall into four kinds, and the kinds are not
equally urgent:

**Worth asking now, because a shopping list is wrong without them (4):**

1. **Chicken noodle soup** — the list is cut off after the ramen noodles and has no
   chicken. How much chicken, and what else is below the fold?
2. **Beef dip Sammies** — how big a chuck roast? It is the only quantity on the card and
   the one that decides how many it feeds.
3. **Pork loin and rice** — `Soup sauce`. A typo, or shorthand? Cream of mushroom soup and
   a bottled sauce lead to different carts. *(The tool refuses this on principle: an item
   name made only of category words names nothing you can buy.)*
4. **Chili** — the two screenshots do not join. Is anything missing between
   `1/2 tsp oregano` and `2 Tbsp tomato paste`?

**Worth asking once, because they settle a corpus row (3):**

5. **Beef dip Sammies** — `corpus.md` records the passive time as *slow cooker*; the card
   describes a stovetop braise in a large pan. Which is how you actually make it?
6. **Chicken and dumplings** — still the `?` from the source document. Cooked and liked,
   or bookmarked? It is the one recipe in the corpus whose membership is unproven.
7. **Cheesy pasta** — the corpus calls it American; the ingredients (marinara, Italian
   seasoning) read Italian-American. Not important on its own, only for variety planning.

**Worth answering in bulk, over a coffee (16):** sixteen of the thirty-four questions are
the same question — *how many adults does this feed?* — asked of the sixteen recipes whose
source never states servings. Every typed note and ten of the eleven screenshots. The
household can answer each in two seconds and nothing else can answer them at all. This is
the single highest-value list in this document, because yield is what the planner uses for
leftovers and it has already gone wrong once.

**Worth recording, not asking (the rest):** can sizes for the chicken chili's six `2 cans`
lines, the tuna can size, whether the meatball subs use frozen or homemade. They resolve
themselves at the store.

## 5. Peanut: clean, and one open note closed

No peanut ingredient in any of the twenty-three, and **no bought sauce that could carry
one**. That second half is the interesting part: `corpus.md` carries *check sauce for
peanut* on the teriyaki chicken and the veggie stir fry, on the reasoning that a bought
sauce is where the allergen hides. Their ingredient lists are now visible, and both make
their own sauce — soy sauce and brown sugar in one, soy sauce, brown sugar and broth in
the other. **Neither uses a bought sauce, so the note can come off both rows** once
someone confirms the brand of soy sauce is the usual one.

## 6. What the twenty-three did to the schema

The schema was designed against four recipes. Five things it did not survive intact:

**a. `active` and `passive` were not knowable from a source.** The four hand-made files
carry `active: low` / `passive: 10 hr slow cooker`, which read like facts from the recipe.
They are not — no source states hands-on time, and mapping a stated *cook time* onto
passive is exactly the collapse the brief warns about (a forty-minute stir fry is forty
minutes of standing there). **The new files record what the source actually said**:
`times: prep 15 min, cook 65 min (source)`, `active: not stated in source`, and a passive
line naming the method word found in the source's own steps, marked unverified. The
inferred `low`/`med` stays in `corpus.md` where it is already documented as a guess. This
is a real divergence from the four existing files and it is deliberate.

**b. `§1`'s table and `§2`'s example disagree** about whether `active`/`passive` belong in
`recipes/` at all — the table puts them in the corpus, the worked example puts them in the
file. Kept in both, source-derived in the file and inferred in the corpus. Worth settling.

**c. Capture completeness needed a field.** `status: complete | partial | failed` is new.
Without it there is no difference between a recipe with five ingredients and a recipe with
five of its ingredients. Five of eleven screenshots are `partial`, and that number is the
whole point of this exercise.

**d. Open questions needed somewhere to live.** A recipe file now ends in
`## Open questions`, so the gap is visible to the person cooking rather than only in a
report they will not re-read. A note prefixed `!` means *content may be missing* and marks
the file partial; `?` means *this needs an answer*.

**e. Ingredient sub-lists exist.** The meatball subs have a `### Meatballs` sub-list and
the tuna melt a `### For the sandwich`. Flattening them loses the fact that the meatball
lines are an *alternative* to the frozen meatballs line, not additional to it.

Two smaller ones: `modality:` and `images:` were added because provenance turned out to
matter (three recipes point at URLs that will never resolve again — two temporary sandbox
addresses and an expired Instagram story), and the grammar in §2 needed the note to split
on the **last** top-level comma rather than the first, or
`2 lb boneless, skinless chicken thighs, cubed` becomes an ingredient called `boneless`.

## 7. What this says about Step 2

**`items.md` has 27 rows and these recipes use 154 item names it does not know.** That is
not a failure — §3 says the table grows on parse failure and an unknown item defaults to
`aisle: other` and gets reported. But it is the size of the remaining job, and it is now
measured instead of guessed. The full list is in the run report.

**Protein inference agreed with the corpus 22 times out of 23** and never disagreed. The
single miss is Chicken noodle soup, and it missed *because the chicken is missing from the
capture* — the parse gap propagating into the inference, which is the system working.

**Cuisine inference is weak and should stay out of the way.** It could name a cuisine for
only 3 of 23. That is by design after a first pass claimed a tuna melt was
Italian-American because it contains pepperoncini, and a teriyaki chicken was Chinese
because it contains soy sauce. One marker is not evidence; the tool now requires two and
otherwise says nothing.
