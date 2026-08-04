# What the second onboarding pass turned up

**Brief: [`brief-onboarding-pass-2.md`](brief-onboarding-pass-2.md). Pass 1:
[`onboarding-findings.md`](onboarding-findings.md).**

Pass 1 captured all 25 recipes and reported honestly on what it could not get. Pass 2 went
back to the sources for the closeable part of that, and captured three schema features that
did not exist the first time.

The headline is not the recoveries. It is that **the biggest single item on the outstanding
list — fifteen unknown yields — was partly a schema error, not a data gap.** A third of it
dissolved on inspection.

---

## 1. The fifteen yields, and what they actually were

| | count | what happened |
|---|---|---|
| Recovered from the source | 5 | found the page, read the servings, cited it |
| Never a question | 3 | the recipe has no batch size and no source could state one |
| Genuinely open | 7 | a batch dish whose source is silent or gone |

**Recovered (5).** Every one was a screenshot whose address bar named a site but not a page.
Searching the site for the captured ingredient list identified the recipe in each case, and
in each case the fetched list matched the capture line for line — which also closed four
separate *content may be missing* flags as a side effect.

| Recipe | Source found | Yield |
|---|---|---|
| Chili | julieseatsandtreats.com — *Easy Chili Recipe* | 4 servings |
| Enchiladas | southernbite.com — *5 Ingredient Beef Enchiladas* | 8 enchiladas |
| Meatball subs | spendwithpennies.com — *Meatball Sub* | 4 servings |
| Sliders | natashaskitchen.com — *Cheeseburger Sliders* | 24 sliders |
| Beef pot roast | dinnerthendessert.com — *Ultimate Slow Cooker Pot Roast* | 8 servings |

The pot roast is the one with an asterisk. **The origin returns 403 to every user-agent
tried** — Cloudflare, not the polite kind of block pass 1 found on `natashaskitchen.com`.
The servings figure and the method step an ad had covered come from a mirror that names
dinnerthendessert.com as its source and reproduces the ingredient list exactly. That is
weaker evidence than an origin and the recipe file says so on its face.

**Never a question (3).** BLT, hamburgers and tacos have no batch size. You make as many as
there are people. `2lb ground beef` on the hamburgers is how much gets bought, not how much
a batch makes, and their ingredient lists carry almost no quantities because there is
nothing fixed to quantify. **No source states a yield for these because none could**, and
asking the household would have been asking a question with no answer.

This produced a schema change — `docs/step2-design.md` §2.5. `yield` now takes three shapes
rather than one number, and `per portion` is a real value. Two of the recoveries landed in
the second shape: *8 enchiladas* and *24 sliders* are portion counts, not adult-equivalents,
and turning them into AE needs one number from the household that no source can supply.

**Genuinely open (7).** Chicken noodle soup, pork loin and rice, cheesy pasta, biscuits and
gravy, chicken chili, zuppa toscana, tuna melt. Each is a real batch dish whose source
never said, and four of the seven have no retrievable source at all — two temporary sandbox
URLs, a Facebook post, and an expired Instagram story. Nothing but the household closes
these, and **they now surface at the moment they matter**: `shop.py` prints *"not scaled —
the recipe doesn't know how much it makes"* on any week that uses one. That is a better
place for the question than a to-do list.

## 2. The five partial captures

| Recipe | Was | Now |
|---|---|---|
| Chili | partial — two screenshots, unknown seam | **complete** — source confirms the lines are adjacent |
| Enchiladas | partial — list may start above the screenshot | **complete** — the recipe has five ingredients and all five are captured |
| Chicken noodle soup | partial — no chicken in the list | **still partial**, with a different reason |
| Beef dip Sammies | partial — card cut off mid-sentence | **still partial**, permanently |
| Tuna melt | partial — story frames may not join | **still partial**, permanently |

The soup is the interesting one and it is the worked example the brief predicted. Its
missing chicken was never recoverable, because **the gap was variation rather than
truncation** — this household sources the chicken two ways, so there was no single right
line to find. The chicken is now supplied by the household through a variant block, not
recovered. What stays unknown is whether anything *else* was below the fold, which the
capture cannot answer, so the file stays `partial` for that reason alone. Recovering the
answerable part of a gap does not license claiming the rest.

## 3. Variants: three captured, and one of them fixed a bug in the docs

All three are in place — chicken noodle soup (rotisserie / whole young chicken), beef dip
Sammies (slow cooker / stovetop braise), meatball subs (frozen / homemade). The corpus rows
now read `rotisserie *or* whole bird — see variants` in the passive column, which is where
the difference actually lives.

The whole-bird soup is the corpus's only `produces:`. Boiling the bird makes the stock that
`replaces:` the six cups of bought broth, and `shop.py` resolves it correctly: choosing that
variant removes chicken broth from the list and adds a whole chicken.

**The other two `produces:` pairs the design doc claimed do not exist.** §2.4 was written
asserting that the beef pot roast makes jus for the beef dip Sammies, and that the crock pot
Italian beef produces shredded beef for something. Read against the actual files, neither
survives: the beef dip makes its own liquid from its own roast and never buys jus, and
nothing in the corpus consumes shredded beef. Both were plausible and both were invented —
by me, in a design document, about data that was sitting right there. §2.4 now says so, and
`test_shop.py` pins the count at one so the claim cannot drift back.

## 4. `accepts:` — five, each with the sentence that justified it

| Recipe | Line | Source sentence |
|---|---|---|
| Meatball subs | provolone | *"shredded provolone **or mozzarella** cheese"* |
| Sliders | shredded cheddar | *"6 oz medium cheddar, shredded (**or use more sliced cheese**)"* |
| Enchiladas | flour tortillas | *"We much prefer flour tortillas in this recipe, **but corn are more traditional**."* |
| Meatloaf | Panko breadcrumbs | *"3/4 cup Panko breadcrumbs (**or gluten-free bread crumbs**)"* |
| Biscuits and gravy | sausage tube | *"1 sausage tube **spicy or regular**"* |

Nothing was inferred. Several near-misses were left alone on purpose — the tuna melt's
*bread of choice* and *cheese of choice* are unspecified ingredients, not declared
tolerances, and the chicken and biscuits casserole's *"or two smaller 6 ounce cans"* is
packaging, not substitution.

**Four of the five are inert, and that is worth knowing.** `items.md` already maps
*gluten-free bread crumbs* and *sliced cheddar* and *corn tortillas* onto the same canonical
item as the thing they replace, so consolidation has nothing to merge. Only the meatball
subs' provolone → mozzarella crosses a canonical boundary, and no other recipe in the corpus
uses mozzarella, so it has never fired on real data either. The machinery is tested
synthetically and is correct; it is simply waiting for a corpus large enough to need it.

## 5. What the pass did to the corpus

- **24 rows, unchanged in membership.** Nothing moved in or out.
- **Yields:** 15 unknown → 7 unknown, 2 portion counts, 3 per-portion, 5 in AE.
- **Two stale claims removed** that had nothing to do with this pass but were sitting in
  `corpus.md`: chicken and dumplings still described as *left in the corpus* after it moved
  to candidates, and a pointer to a *soft cap* on beef in `profile.md` that was deleted
  weeks ago. Counts corrected to 24 throughout.

## 6. Two questions worth one sentence each

Not a to-do list. These are the only things a person can answer that a source cannot, and
they are worth answering at the moment the recipe comes up rather than in a sitting:

1. **How many enchiladas is one adult?** Turns `8 enchiladas` into a usable yield.
2. **How many sliders is one adult?** Same shape. A slider is a slider, so both answers are
   reusable forever.

And seven yields that will close themselves the first time each dish gets cooked.
