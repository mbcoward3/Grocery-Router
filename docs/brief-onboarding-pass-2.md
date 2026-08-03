# Brief — onboarding, second pass

**For the next agent.** Pass 1 captured all 25 recipes and reported honestly on what it
could not get. This pass closes what is closeable and captures three things the schema
did not have last time.

Read `docs/brief-recipe-onboarding.md` first — it still governs. Then
`docs/step2-design.md` §2.1–2.4, `docs/onboarding-findings.md`, and this.

---

## What changed since pass 1

**The rule is now *never invent, but do recover*.** Pass 1 read "never invent" as *flag the
gap and stop*. That was too conservative. When a capture is short and the recipe is
identifiable, go and find the real thing — re-fetch, or search by title and distinctive
ingredients — fill from it, and **cite it, marking those lines recovered rather than
captured**. Sourcing is not inventing. `1 lb chicken breast` because soup usually has
chicken still is.

**Three optional fields exist**: `## Variants`, `accepts:`, `produces:`. Specified in
`docs/step2-design.md` §2.2–2.4.

**They are optional by construction and must stay that way.** A recipe with a title, a
source and a flat ingredient list is complete. Onboarding has to remain a five-second
operation and has to work for a household with an empty corpus, so none of these may
become a question that blocks a capture. Record them where a source or the household has
already stated them. **If capturing one would mean guessing, do not capture it.**

**The corpus is 24, not 25.** Chicken and dumplings moved to `candidates.md` by household
decision, along with the two recipes the planner proposed for the week of 2 August. Both
files are read by the planner; only `corpus.md` carries the proven guarantee.

## Target 1 — the fifteen unknown yields

The single highest-value job here. Yield drives leftover planning and has already gone
wrong once, when a planner run proposed a double batch of a recipe that already served 8.

```
Chili · Enchiladas · Chicken noodle soup · Tacos · Hamburgers · Pork loin and rice
Cheesy pasta · Biscuits and gravy · BLT · Meatball subs · Sliders · Chicken chili
Zuppa toscana · Beef pot roast · Tuna melt
```

**Split them before you start.** Roughly nine come from published pages — the screenshots
name their sites in the address bar, and `spendwithpennies.com` is visible on the meatball
subs. Those are recoverable: find the page, read the servings, cite it. The remaining six
are the household's own typed notes, published nowhere, and **no amount of searching will
produce a yield for `BLT`**. Do not burn effort proving that. Report them as the
irreducible list and let the household answer in one sitting.

## Target 2 — the five partial captures

`beef-dip-sammies` · `chicken-noodle-soup` · `chili` · `enchiladas` · `tuna-melt`

Each is *readable and incomplete*, which is worse than unreadable, because a
complete-looking file is what a shopping list will trust. Apply the recovery rule to each,
then either promote it to `complete` with citations or leave it `partial` with the reason
sharpened.

Two carry a warning:

- **Chicken noodle soup** — its source was a temporary sandbox URL that will never resolve.
  More importantly, **its gap is variation, not truncation**: the household cooks it two
  ways, so there was never one right chicken line to recover. This one gets a variant
  block (below), not a better fetch. It is the worked example of the recovery rule's limit.
- **Beef dip Sammies** — a photograph of a handwritten card with no publication behind it.
  The roast size and the method are now answered; the simmer time and temperature are cut
  off mid-sentence and are gone. Leave that open and say so.

## Target 3 — the three known variants

All three are already established. Do not re-litigate them, capture them.

| Recipe | Variants | Status |
|---|---|---|
| Chicken noodle soup | rotisserie chicken · whole young chicken boiled for its stock | household-confirmed |
| Beef dip Sammies | slow cooker · stovetop braise | household-confirmed; block already written, verify it |
| Meatball subs | frozen meatballs · the homemade sub-list already in the file | stated by the source, household not yet asked |

The whole-chicken soup variant is also the corpus's first `produces:` — boiling the bird
yields the stock that `replaces:` the 6 cups of chicken broth. Get that one right; it is
the worked example for the whole feature.

## Target 4 — sweep for `accepts:` and `produces:`

Now, while all 27 files are open, because doing it later means re-reading everything.

**`accepts:`** — only where a source states it. The meatball subs already read
`1 cup shredded provolone **or mozzarella** cheese`. That is a free tolerance sitting in the
text. Never infer that one cheese stands in for another; the household will add the rest
over time.

**`produces:`** — the corpus contains at least one real pair beyond the soup: **beef pot
roast makes jus, and beef dip Sammies needs au jus.** The crock pot Italian beef produces
shredded beef and juice well beyond one dinner. Record the output and the consumer's
`may come from:`, and remember the rule — **the fallback is always bought.** The link saves
using the item, never buying it.

## Do not touch

Resolved by the household this session. Re-opening any of these is a regression:

- `Soup sauce` is **soy sauce**. The raw line is deliberately corrected in that one file.
- Beef dip Sammies is a **3 lb roast, about 8 AE**.
- Cheesy pasta stays tagged **American**, against the inference from its ingredients.
- Chicken and dumplings is a **candidate**, not corpus.
- No protein quota exists anywhere. An earlier draft had one; it was removed on purpose.

## Report

Same standard as pass 1 — that report was the most useful artifact of the run.

- Yields closed by recovery, and the irreducible list that needs the household.
- Partial captures promoted, and the ones that are permanently short with the reason.
- Every `accepts:` and `produces:` captured, **with the sentence in the source that
  justified it**. A tolerance with no quotation behind it is an inference and does not
  belong.
- Anything the new fields could not express. They are four days old and were designed
  against three examples; twenty-seven will stress them.
