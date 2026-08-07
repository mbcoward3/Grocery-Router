# Onboarding a new household

**Decided in interview, same as `docs/architecture.md`. This is the shared understanding.**

A family signs up, walks an interactive onboarding, and comes out with a profile and
whatever corpus they already had. Until they finish it, they are gently prompted at every
login — until they do it, or opt out. A household with little or no corpus gets more new
recipes from a platform recipe bank until theirs fills up.

Costs are written next to decisions, because a design doc that only lists upsides is a
sales document.

---

## What was measured first

Four things were checked against the running code before any of this was decided. Three
of them are defects that a second household exposes and the first one hides.

**A brand-new household cannot acquire anything.**

```
sources for a brand-new household: []
acquire() returns: []      log: 'no sources: the corpus names no domains to search'
ranker proposes: []
```

`acquire.sources()` derives which sites to search *from the corpus*. No corpus, no
domains, no search. So the bank is not only seed recipes — **it is the seed source list**,
and it is what makes a household's own acquisition bootstrap at all.

**The candidate cap starves exactly the household the bank exists for.** A new family with
3 recipes, with 50 bank recipes available and 5 nights to fill:

| risk | corpus | bank | nights | result |
|---|---|---|---|---|
| normal | 3 | 50 | 5 | **4 meals** — `Theirs 0, Bank 0, Theirs 1, Theirs 2` |
| high | 3 | 50 | 5 | 5 meals — `Theirs 0, Bank 0, Theirs 1, Bank 1, Theirs 2` |

`want_cands` is `{low: 0, normal: 1, high: 2}` — a fixed number that ignores how much
corpus exists. Fifty available, one taken, week one comes up short.

**`BASE_AE = 2.5` is one family's size, compiled in.** `pantry.py:43`, commented *"From
profile.md: 2 adults, a 3-year-old, a 1-year-old"*, read in four places including both
shopping-list paths. Every new household would get quantities scaled for somebody else's
children.

**The one hard constraint is hardcoded to one allergen.** `PEANUT_HEADER`,
`PEANUT_BLOCKS = "contains peanut"`, `peanut_verdict()`. A shellfish or gluten family would
declare it at onboarding and the system would enforce **nothing**, while looking like it
handles allergies.

The last two are the same bug twice: a single-household constant that reads as a feature.
They are the price of having built for one real family, which was the right call and is
now due.

---

## Decisions

| # | Decision | Costs us |
|---|---|---|
| 1 | **The recipe bank is scraped from public recipe sites and owned by the platform.** A household's own captures stay private to them | A crawl and a corpus of somebody else's content to keep lawful and fresh |
| 2 | **Onboarding gathers the profile and any existing corpus. The bank fills weeks, not corpora** | Two different things a family might call "my recipes", and the UI has to keep them apart |
| 3 | **Imported recipes go straight into `corpus.md`, fully trusted** | Corpus membership stops meaning one thing — see below, it is the expensive one |
| 4 | **All six profile sections are asked, as a form with model-parsed free text** | A long first session, and most answers are assertions rather than evidence |
| 5 | **Everything degrades visibly rather than gating — except allergens, which are required** | One wall in the first five minutes |
| 6 | **Allergens: block a curated set, warn on anything else** | Two tiers of confidence to explain on screen, and a term list to maintain |
| 7 | **One login per household. Members stay attribution** | Still nobody knows which adult typed it |
| 8 | **Bank fill is a household dial, defaulted higher when the corpus is small** | A dial, and this project has shipped one that changed nothing before |
| 9 | **The platform owns the Anthropic key. No cost constraints for now** | Every signup costs tokens before it earns anything, and nothing currently caps it |

### Decision 3, in full, because it reverses a rule

`onboard.py` used to append never-cooked recipes straight into `corpus.md`. That was called
a live rule violation in `docs/architecture.md` and was deliberately fixed: *membership is
earned, and `promote()` is the only door.*

Importing an existing corpus reverses it, knowingly. The argument for: a family arriving
with 40 recipes they cook every month **has** earned membership — just outside this tool,
with no recorded cook. Routing them through candidates was measured above at one per week,
which is a ten-month onboarding and a week one made of strangers.

The cost, stated plainly: **`corpus.md` now holds two populations that nothing can tell
apart** — recipes proven here and recipes asserted at signup. Every number `review.py`
reports about corpus behaviour mixes them.

**The mitigation is provenance, and it is not optional.** An imported row has to carry a
mark saying so. That is not bookkeeping — it is what turns the cost into an asset:

> *"You told us at onboarding you wanted more fish. You have dropped every fish meal since."*

A stated preference becomes a **hypothesis that behaviour can test**, which is the most
interesting thing this product could say to a household, and it is only possible if
asserted and observed claims are stored distinguishably.

**A related defect ships with it.** `_reasons()` for a row with no last-cooked date emits
*"no record of cooking this yet — unranked, not stale."* Attached to a recipe a family
just said they cook monthly, on day one, that sentence is false. The reason is the product.
An imported row needs its own reason kind before this goes anywhere near a household.

### Decision 5, and why allergens are the exception

The house posture is *degrade, never block* — the tool does what it can honestly do and
stops where it cannot, with a line saying why. Applied here: no quantities on the shopping
list until household composition is given, no allergen blocking until allergens are
declared, each visibly short with a link to the section that fixes it. **The prompt to
finish onboarding is the product being visibly incomplete**, not a banner.

Allergens are the one gate. And the gate is only worth having if:

**"None" is a recorded answer, not an empty section.** If *unanswered* and *actively said
no allergies* look the same on disk, the gate has bought nothing — it is a defaulted guess
with extra steps, and a defaulted `no allergens` is the worst possible place in this
product for a plausible value where there should be a gap. Stored as
`allergens: none declared <date>`, downstream code can tell them apart.

This is the same three-state humility `peanut_verdict()` already has, where `none seen` is
deliberately not `safe`.

### Decision 6, and what the system may promise

A curated set with maintained term lists — peanut, tree nut, shellfish, egg, dairy, soy,
wheat/gluten, fish, sesame — **blocks** a recipe. Anything a household types freehand
**warns** and never blocks. Both tiers are stated on screen, and nothing ever claims a dish
is safe; only that nothing was seen.

The asymmetry is deliberate and it runs the opposite way to the cart rule. In a cart, *a
gap is a smaller failure than a stranger's guess.* Here a missed allergen is far worse than
a false alarm, so the system is generous about warning and conservative about claiming
safety. Term matching over scraped ingredient text is a weak instrument — "gluten" means
wheat, barley, rye, malt, soy sauce and a hundred brand names — and the promise has to stay
inside what the instrument can do.

### Decision 1, and what the bank stores

**Structured fields and a link, not prose.** Ingredient lists and procedures are facts;
headnotes, photos and the author's writing are not ours. The bank stores title, source URL,
and the schema.org fields — which is exactly what `onboard.render_recipe()` already
captures, so this is a constraint the existing capture already satisfies.

Bulk crawling is also a different posture from what exists. Today acquisition fetches on
demand for one household, with `MAX_FETCHES = 12` and a per-host courtesy delay, and three
of the eleven current sources already `Disallow: /wp-json/`. **`robots.txt` compliance is
not optional and has been got wrong here once already.**

The bank is **platform state, not tenant state** — the same distinction `household.py`
draws for the host-keyed adapter caches. It does not live in a household's store, and
nothing about one family reaches another.

---

## What this changes in the code

Ordered by what is unsafe to leave undone.

1. **Household composition replaces `BASE_AE`.** It moves out of `pantry.py` and onto the
   profile, computed from the members onboarding collects. Four call sites.
2. **Allergens generalise.** `peanut_verdict()` becomes a declared-allergen scan with a
   blocking tier and a warning tier; the recipe-file header grows from `peanut:` to a
   verdict per declared term. The three-state shape survives intact.
3. **Provenance on corpus rows.** Imported, suggested, or proven here — and a reason kind
   for an imported row so day one stops saying something false.
4. **`want_cands` scales.** A household dial whose default rises as the corpus thins,
   instead of a constant.
5. **The bank as a source of seed domains**, so `acquire.sources()` has something to return
   for a household with no corpus.
6. **Onboarding itself** — the form, the model parse, the profile render, the
   partial-completion state and the opt-out.

Items 1 and 2 are correctness. Nothing should onboard a real family before they land.

---

## Still open

1. **How big the bank is, which sites it covers, and how often it refreshes.** Nothing
   here decides that, and it is the difference between a weekend crawl and an ongoing job.
2. **What happens when two households capture the same URL.** One canonical capture with
   per-household notes, or two independent copies. Storage is trivial either way; the
   question is whether a correction one family makes should reach anyone else.
3. **What "opted out" persists as**, and whether it ever expires or re-prompts after a
   thin week.
4. **Whether a bank recipe may ever be shown as a *side*.** `sides.md` is empty on purpose
   and the reason was that seeding it invents what a household eats — the bank is now
   exactly that seed, for mains. The same argument has to be made or refused for sides.
5. **Cost, deferred not answered.** Decision 9 says no constraints for now. "For now" ends
   the first time a signup loop or a bad actor runs the interview a thousand times.
