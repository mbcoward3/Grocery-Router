# Project Proposal: Pantry Router

**Decide the week, confirm the list, source the items.**

*Status: draft v2.3*

> **Revision note (v2.3).** The corpus got seeded and the household interview ran. Both
> contradicted things v2.2 asserted. The repertoire is ~30–35, not 60. Effort turned out
> to be two axes rather than one. Day-assignment turned out to be the wrong output shape.
> And the system's first attempt at lateral inference — reading a preference off a recipe
> count — was wrong in a way the household had to correct. Sections 1, 4, 5, 6, 11, 12 and
> 13 changed. Sides are deferred by decision, not oversight (§5).

---

## 1. The bet

A household of two adults, a 3-year-old, and a 1-year-old, plus frequent visitors. Everyone eats the same meal. **Both adults cook, plan, and use the tool equally** - there is no primary operator, so this is one shared brain rather than two profiles that need reconciling.

**The problem is recall, not discovery — but the gap is smaller than v2.2 claimed.** That draft asserted ~60 known recipes against ~15 that surface. Seeding the corpus measured it: **25 recorded, and the household puts the true repertoire at 30–35.** The recall gap is real but it is roughly 15-of-32, not 15-of-60.

> **Proposals surface known-good meals they wouldn't have remembered, and they get cooked.**

**That halving matters, because it moves the household toward the cold-start end of its own curve.** With ~32 proven recipes, retrieval alone cannot fill a year of weeks without heavy repetition. Acquisition is not just the stranger's path (§4) — it is this household's path too, sooner than v2.2 expected. The corpus is also narrow where it is deep: **beef is 12 of 25**, and roughly 18 of 25 are plain American comfort food.

**The corpus arrived off a saved document, not out of memory**, which is a better result than the unaided-recall exercise v2.2 planned and also destroys that baseline measurement (§12). It confirms §4's central claim from the other direction: the recipes memory couldn't retrieve were already written down somewhere.

**The opponent is Kroger's "buy it again."** Good at what it does, and a rut-amplifier by construction: it can only return what you recently bought, which is the same 15. The competitive position is breadth, not efficiency. Efficiency is a constraint - don't make the week slower - never the goal.

**The general form.** This household is the sharpest instance of the problem, not the whole of it. Someone arriving with no corpus has the same underlying constraint - they cook a fraction of what they'd enjoy - but the binding gap is *acquisition* rather than *retrieval*. Both are operations on the same corpus, and corpus size decides the mix. Design against this household concretely; keep the corpus-size assumption out of the architecture.

## 2. The design principle

**Opinionated where judgment matters, deterministic where correctness matters.**

The planner should be a model with a point of view about food and about this household. Ingredient parsing, unit conversion, SKU matching, and cart writes should be boring, testable code. Conflating those two is how this becomes a demo instead of a system.

Concretely, the planner is not a scoring function with tuned weights. It's:

```
household profile (natural language)
  + corpus (recipes, last-cooked dates, outcomes)
  + this week's constraints (effort, guests, what's on sale)
  -> five dinners, each with a reason
```

**The learning is the profile being revised, not parameters being fit.** That matters because 20 meal signals a month can't train anything from scratch - but a model already knows that someone who likes miso salmon might like a miso-glazed pork chop, that Wednesday should be fast, that a household with a toddler isn't doing whole fish. We're not estimating taste from nothing; we're handing evidence to something with priors.

This dissolves most of the machinery an earlier draft needed: recency dials, known-to-new ratios, novelty rates, adjacency scoring. The model handles them, and handles them better than a tuned constant would.

**The known-to-new ratio gets a specific note**, because it's the one that keeps trying to come back as a setting. It is derived, not dialed: corpus size sets the baseline mix, and the week's risk appetite adjusts it. Net-new proposals are candidates, so a week with guests on Thursday is a week to propose proven food, and a quiet week can carry two experiments. That is a judgment about context, which is the model's job - not a constant to expose in a settings screen.

**Three constraints keep it honest:**

**Inspectable.** The profile is a page either of them can read and edit: *you cook fast on weeknights, you've drifted away from Italian, you rarely finish braises.* Correcting it directly is a better trust mechanism than any opaque score. It's a household profile, not a personal one - signals aren't attributed to whoever produced them, which is a simplification, not an oversight (see §13).

**Grounded.** Every profile claim carries its evidence. "Seems to like bright acidic food, based on 6 of your last 10 additions" is useful. The same sentence with nothing behind it is a hallucinated preference that poisons everything downstream. No claim without a trace.

**Gentle.** The profile shifts slower than they can audit it. Revisions are proposed and visible, never silent. Autonomy - the model acting without review - is explicitly out of scope for v1.

## 3. The gating unknown

The inferred pantry, consumption rates, and catalog seeding all need to know **what actually arrived**. Kroger's public API can't supply it: the cart is write-only and order history isn't exposed. Pickup substitutes and shorts items, so "ordered" is never "received."

**Mechanism: parse the Kroger order confirmation emails.** They carry delivered line items, substitutions, and out-of-stocks, automatically, requiring nothing of either of them.

**Phase 1b spike.** If it fails, the pantry and sizing work collapse and the honest response is a planning-only tool that never claims to know what's in the house. Weekly manual confirmation isn't a fallback; it's the chore this exists to delete.

Note the scope precisely: this gates §8 and the Phase 2 branch, **not** the core loop. Corpus, profile, Step 1, Step 2, and the cart write all function with zero reconciliation. Earlier drafts called this "the gating unknown" and blocked all further architecture on it, which was too strong.

**Second use, worth more than the first.** The same emails are a retroactive baseline instrument (§12). Distinct-meals-per-month today can't be self-reported - it fails to the same recall gap the product exists to fix - but months of order history read directly out of the inbox, without contaminating anything by observing themselves.

## 4. The corpus

**Getting the corpus in is the highest-value work in the project**, and it has the same failure as the weekly planning: asked to list their recipes, they'll produce the 15 they can remember. **You cannot self-report your way out of a recall gap.**

They live in their heads and **in their saved artifacts** - a recipe document, links texted back and forth, photos of things that worked. Those already logged what memory can't retrieve.

> **Done, and it validated the mechanism.** Seeding pulled **25 recipes from one saved document** - 8 links, 6 typed-out ingredient lists, 11 screenshots named in the text layer. That is the accelerator below landing before Phase 1 rather than at 1b, and it produced in one pass roughly what thread-mining was projected to yield. The prediction that survived: nobody recalled 25 recipes: they recalled *where the list was*.

- **Bootstrap by mining saved artifacts for recipes** - the recipe document, then the message thread. One-time extraction, not a feature. Yielded 25; the thread may still add to it
- **Capture at the moment of recall, from either phone.** Recipes come back in the kitchen on a Tuesday, not at an onboarding screen. Adding one takes five seconds from anywhere: a share-sheet target, or forwarding a link. They already share recipes by message, so hook that habit rather than building a new one
- **Let the model prompt laterally** when they do sit down. Not a form - a conversation that works outward from what's already in the corpus, by protein, cuisine, season, and "what do you make when there's no time." Starting from empty, it works outward from the setup profile and the model's own priors instead. **This is the general-purpose bootstrap**; artifact mining and receipt parsing are accelerators layered on top, available to this household and not to everyone. **Run it one question at a time.** Batched multi-select produced a dismissal; asked singly, the same questions produced the design corrections in §5, §6 and §13 - including one the household volunteered unprompted, which a fixed form would have had no slot for
- **Mine reconciliation data** once §3 lands: ingredients bought repeatedly imply recipes not yet in the system

**The corpus is never done, and growing it is a goal of the tool rather than a precondition for using it.** The system must be useful at 0, 15, and 60. Two jobs sit on one continuum: *retrieval* surfaces proven recipes that fell out of rotation, and *acquisition* adds new ones that earn their way in. Corpus size decides the mix - at 0 every dinner is acquisition, at 60 most of the value is retrieval - and the model reads that off the corpus rather than a mode flag. There is no separate onboarding mode, only a different place on the same curve.

**Starting from empty needs one thing this household won't feel: a safe first week.** With nothing proven, every proposal is a candidate, and five unproven dinners is five chances to lose the user before the loop ever runs. Propose fewer, lean on high-prior low-variance recipes, and let the corpus earn its way up to a full week.

### Membership is earned

**The corpus is proven-only.** Nothing enters until it has been cooked and liked. That strict bar is what makes the corpus trustworthy as a model input: every item in it is known-good, so retrieval is purely a recall problem and never a quality gamble. Acquisition carries the gamble instead, which is the real asymmetry between the two jobs.

Three states, and nothing is ever deleted:

- **Candidate** - a new recipe the system proposed. Cooked and liked, it enters the corpus. Cooked and it flopped, it stops surfacing - but **the flop is recorded with its reason**, never silently dropped. At low corpus sizes it's the most informative signal the system will receive all week (§9)
- **Corpus** - proven. Deprioritized when consistently passed over, never demoted for a single miss
- **Archived** - proven but repeatedly avoided. Stops surfacing, stays retrievable on request

The 25 existing recipes entered the corpus directly, since they've already cleared the bar in real life. Only genuinely new suggestions pass through candidate. One exception is already logged: *chicken and dumplings* was written with a question mark in the source and nobody remembers why, so it sits in the corpus marked uncertain and gets resolved the first time it comes up.

That direct entry is worth naming honestly: it's self-reported proof, so the real bar is **proven-or-attested**. Someone attesting to eight recipes at setup is doing exactly what this household does with sixty. The epistemics are identical; only the volume differs, which is why cold start is a corpus size and not a second product.

Archiving rather than deleting is the right fit for a second brain: a brain doesn't erase things, it just stops recalling them. Reaching into the archive on purpose is a feature.

**Setup, ~60 seconds.** Hard constraints only: household in AE, allergies and vetoes, weeknight effort ceiling, store and pickup location. Everyone eats the same meal, so proposals should be family-edible - a light filter, not a design driver. Starting from empty, setup extends into the lateral-prompting conversation above, since the profile has to carry the weight the corpus normally would.

## 5. Scope

5 dinners a week, which is **not** 5 cooking events - some nights are leftovers by design. Confirmed and now specific: leftovers do **both** jobs here, covering next-day lunches *and* standing in for a dinner, so the planner should target **5–6 cooks across 7 nights** and favour dishes that scale. Servings in adult-equivalents (~2.5 base, per-meal control for guests); "serves 4" means nothing until converted. Kroger pickup today, Costco later, store layer pluggable. We build the cart, they check out. No nutrition tracking, recipe hosting, or maintained inventory.

**Dinners only, and sides are explicitly deferred.** The corpus is mains-only: vegetables look nearly absent from it, but that is a recording artifact rather than the diet - sides get served and never get written down. The household's call is to reach a working dinners-only steady state first, so this is a deferral, not an oversight.

State the cost plainly, because it is a correctness gap and not just a missing feature: **grocery lists generated from a mains-only corpus will be systematically short.** Step 2 must say so on the list itself rather than presenting a complete-looking week that quietly omits every side. A visible "sides not included" line costs nothing and prevents the tool being blamed for the household getting home without green beans - the same reasoning as flagging likely-owned items instead of dropping them (§6).

## 6. The three steps

### Step 0 - The prep job, before anyone opens anything

**Runs unattended, shortly before the weekly session.** Its job is to have every slow or
external answer already in hand, so the session never waits on a network call and never
asks the household something it could have looked up.

- **Sale hunting.** Pull this week's Kroger promotions. This is the one input that is
  genuinely time-sensitive and genuinely external, and it is useless if fetched while
  someone is sitting there.
- **Seasonality window.** What is actually good right now, cross-checked against those
  promo prices. Where the model's prior and the real price agree, the signal is strong.
- **Staleness pass.** Read last-cooked dates and compute what has fallen out of rotation.
- **Open loops.** Which of last week's meals have no recorded outcome yet - this becomes
  the feedback prompt that opens the session.

Output is a **briefing**: a small cached artifact the session reads instantly. If the prep
job fails, the session still runs on a stale briefing and says so. It must degrade, never
block.

### Step 0.5 - One session a week

**The household interacts with this tool once.** Everything - feedback, planning,
adjusting, ordering - happens in a single sitting, and the tool asks for nothing between
sessions. That constraint is a design driver, not a nicety: a tool that pings them on a
Wednesday is a tool that gets ignored by the second week.

The order inside the session matters, and it is **feedback first**:

1. **Directed feedback on last week.** Not a blank "how did it go" - specific, answerable
   prompts drawn from the briefing's open loops. *You planned five, here are the two with
   no outcome recorded. Cooked, skipped, or flopped?* This is the corpus-tuning step, and
   putting it first means the week that follows is planned against a corpus that already
   knows what just happened.
2. **Next week's proposals**, planned with that fresh signal plus the briefing.
3. **Adjust in place.** Opt out of individual suggestions, change the number of dinners,
   add guests, turn a dial - then **request additional generation to fill the gaps** left
   by whatever was dropped. Regeneration is targeted at the holes, never a reroll of the
   whole week, so accepted proposals stay put.
4. **Confirm**, and the list and cart follow deterministically.

Two consequences worth stating. The feedback step is the *only* reliable moment signals
get captured, so it has to be fast and specific enough that nobody skips it - a bare "how
was it" collects nothing. And because generation is incremental, the planner must be able
to fill a partial week without re-proposing what is already accepted.

### Step 1 - The week

**A pool, not a grid.** v2.2 rendered the week as five day-bound rows. The interview killed that for this household: hard nights are **unpredictable week to week**, with no recurring squeeze to plan around. Binding meals to named days would guarantee a wrong assignment most weeks and train them to ignore the day column - and a plan you have to mentally re-shuffle every night is worse than no plan.

So the week is a **set with a required effort mix**, picked from night-of:

```
This week - 5 cooks, 7 nights

  Chicken thighs + rice        4 AE   low active    [cook big -> covers a second night]
  Chicken piccata            2.5 AE   low active    [not since March, and you have capers]
  Tacos                        6 AE   low active    [+2 guests]
  Beef stew                  2.5 AE   med active    [long, but unattended]
  Miso salmon                2.5 AE   med active    [new - close to your sheet pan salmon]

  3 of 5 are low-active, so any night can be a bad night.
  Stew and the thighs both scale -> 2 leftover nights covered.
```

Day-binding stays available as a **rendering choice**, not a structural one - a household with a fixed Thursday class should get the grid. What must not be structural is the assumption that the week *has* a known shape. Keep that out of the planner the same way corpus size is kept out of the architecture (§1).

Swap, reroll, pin, adjust servings. **Every proposal shows why it surfaced**, which is the tool doing its job in the open and the thing that makes a forgotten recipe land. It also makes rejection legible: swapping out a meal the system just said hasn't been made since March is a signal about the *recipe*, not the week.

**One pool, not tiers.** Anchors and deep cuts are the same ~32; the difference is recall frequency, which is a measurement rather than a tag. Surfacing a forgotten one makes it recent and the split re-forms on its own.

Standing planning constraints, given to the model rather than coded: vary protein and cuisine, share perishables across meals and **show the coupling**, and hold the week's candidate count to what corpus size and the week's risk appetite justify. Two of these arrived from the interview with real numbers attached:

**Effort is two axes, and only one of them is capped.** *Active* time - hands on, at the stove - is capped at 20–30 minutes on weeknights. *Passive* time is not capped at all: slow cookers, braises and long oven sits are fine on a Tuesday. v2.2's single effort scalar collapsed these, and the corpus shows exactly what that costs: beef stew and pot roast rated `high` and would have been filtered out of every weeknight, when their length is entirely unattended. **A planner that reads total time will systematically discard the household's easiest meals.** One or two weekend nights carry a real active-time budget; the rest of the week does not.

**Protein balance stays a judgment, not a quota.** An earlier v2.3 draft set a 3–4 beef-night cap against the measured 48% share. That was wrong and is removed. **The corpus is the household's own expression of what it wants, and it self-corrects** - if they lean toward chicken, chicken recipes accumulate and the mix moves without anyone setting a number. A quota overrides that signal rather than reading it, and it puts the tool in the position of arguing with them about their own food. Vary protein across a week because variety is good planning; never to hit a target. Seasonality enters here too, and needs no infrastructure: it isn't a preference, it's a proxy for price and quality, and the model already knows what's good in August. Cross-check it against actual Kroger promo pricing - where the prior and the price agree, the signal is strong. Coupling is the cost of good proposals - independent draws degrade gracefully, a coupled set cascades - so visible links let them repair a broken Wednesday instead of abandoning the week. Sales are opportunistic: flag when something already planned is on sale.

### Step 2 - The list

Ingredients and quantities, scaled to the week's AE totals, aggregated, one screen. Deterministic - no model in this path beyond parsing.

```
Chicken thighs        3 lb
Yellow onion          3
Soy sauce             1/4 cup      [you likely have this]
Rice                  2 cups       [you likely have this]
```

Likely-owned items are flagged, never auto-removed: a silent drop means someone gets home without eggs, a visible flag costs one tap.

**Quantity edits are two things.** Correcting a parse ("1 tbsp, not 1 cup") is a bug report; changing an amount ("we go through more rice") is a preference. Same gesture, opposite fixes.

### Step 3 - The sourcing

Doesn't render until Phase 4. Through Phase 3 everything is Kroger and a comparison screen with one source is theater. The adapter interface exists from day one; the UI doesn't.

## 7. The sizing model

Sam's rule, verbatim: *if it'll still be good the next time we need it, and we can use all of it before it goes bad, get the bigger size.*

> **Buy the larger pack when** shelf life > time until next reorder, **and** pack size <= consumption before spoilage

Plus trip cadence and write capability. Deterministic and inspectable - it generates Step 3's explanation text. Freezing rewrites both terms and gets its own branch at Phase 4.

## 8. The inferred pantry

Staple suggestion plus purchase history *is* a pantry model, so name it honestly: **inferred, confidence-weighted, never authoritative.** Confidence decays as time-since-purchase exceeds typical consumption. Soy sauce runs out. Below threshold the flag becomes a question. Entirely dependent on §3.

## 9. What feeds the profile

Every signal below is evidence attached to a profile claim, not a number in a weight vector:

| Signal | Evidence for |
|---|---|
| Swaps and rerolls, with an optional reason chip | Tired of it / busy night / guests / not actually a favorite |
| Rejecting a shown reason | The recipe was rated higher than it deserved |
| **Candidate cooked and flopped, with reason** | **Where taste ends - the highest-value signal at low corpus size** |
| Candidate cooked and kept | Which priors were right, and what to reach for next |
| Cooked-through over months | Which of the corpus are real |
| Servings adjustments | Consumption and guest patterns |
| Strike-outs in Step 2 | What they keep stocked |
| Quantity edits (preference kind) | Consumption rates |
| Ordered but not cooked | Rescue candidates for next week |

The reason chip matters most early, when a bare swap is unattributable and four lessons share one gesture. It matters less as the profile accumulates enough context to guess.

Candidate outcomes invert that curve - they matter most when there's least else to go on, and they arrive at only 1-3 a week. A system doing acquisition seriously cannot afford to discard them.

## 10. Architecture

```
search(canonical_item, quantity) -> [{ sku, price, promo_price, pack_size, unit, availability }]
capabilities: { can_write_cart, can_check_stock, has_live_pricing }
```

One store adapter interface, so a new store is a new file. Deterministic pipeline: Ingest -> Parse -> Normalize -> Aggregate -> Suggest -> Source -> Write. Hard parts in order: parsing ("juice of 1 lemon", "1 (14.5 oz) can"), unit reconciliation (3 tbsp oil -> a bottle), SKU matching (40 yogurts).

The model sits above this, at planning and profile revision only.

**No retrieval layer.** Sixty recipes with ingredients and metadata is roughly 40k tokens, so the whole corpus goes into context on every call. That holds through a few hundred recipes, which this household will never exceed. No vector store, no embeddings, no RAG - and writing that down so it doesn't get added reflexively later.

**Week 1 must not be worse than their status quo.** Cold-matching 40 SKUs at the highest-churn moment is how this dies. Early weeks draw from recipes already added, so items are things they already buy and the personal catalog fills from repeats.

## 11. Roadmap

**0 - Baseline.** Watch their real process end to end, including "buy it again," *before* either of them has seen anything. Count distinct recipes cooked in a month today - by inbox archaeology if the §3 spike lands, by observation if it doesn't. **Partly spent already:** seeding the corpus means they have now seen the system reason about their food, so the clean-room version of this is gone. Inbox archaeology is unaffected and is now the only uncontaminated baseline available (§12).

**1 - The loop, built cold-start first.** Setup interview -> profile v0 -> Step 1 -> Step 2 -> Kroger cart, working correctly at corpus size zero. This is the path every user hits, this household included, and it holds no assumption that a corpus already exists.

**1b - This household's accelerators.** ~~Recipe-document mining~~ **done, out of order** - 25 recipes landed before Phase 1 exists. Remaining: message-thread mining for the 5–10 unwritten regulars, five-second capture, receipt-email spike. The sequencing note below still holds for what's left, but the corpus is no longer hypothetical while the loop gets built, so **Phase 1 must be tested at corpus size zero deliberately** - the populated corpus is now sitting right there, and that is exactly the condition under which cold start silently rots (§13).

**2 - Inferred pantry.** Reconciliation feeding staple flags with decay. Gated on the spike.

**3 - Profile revision.** The model proposes profile updates from accumulated evidence; they review them. Sale flags.

**4 - Multi-store.** Costco, freezer branch, Step 3 rendered.

**5 - Messy ingest.** Screenshots and cookbook photos. Report the parse-quality gap honestly.

**6 - Costco pricing.** Receipt parsing, warehouse side.

Sequencing note: 1 before 1b is deliberate. Building this household's bootstrap first would bake a populated corpus into the foundation and require unwinding it later.

## 12. Evaluation

**Primary at high corpus size: repertoire breadth.** Distinct recipes cooked over a rolling quarter, against the ~15 that rotate today. If the dormant ones stay dormant, the project failed regardless of how good the carts are. **The target is now ~32, not 60** - and since that ceiling is low enough to hit inside a quarter, breadth alone stops being a sufficient metric sooner than v2.2 assumed. Once the repertoire is exhausted, acquisition rate below is the live number.

**The unaided-recall baseline is gone and is not recoverable.** v2.2 required writing down how many recipes they could list cold, before the first run, as the only way to distinguish a tool that surfaced forgotten meals from one that agreed with what they'd have picked anyway. The corpus came off a saved document instead, so that number was never taken. Two consequences, both accepted: the *count* is better evidence than unaided recall would have been, and the *discrimination* it was meant to provide has to come from the surfaced-and-cooked secondary metric instead. Watch that one harder than v2.2 planned to.

**Primary at low corpus size: acquisition rate.** Net-new recipes proposed, cooked, and kept, per month - plus the keep rate, since acquisition that mostly flops is churn rather than growth. Someone starting at zero has no ~15 to beat, so breadth isn't measurable yet and corpus growth is the only honest read. The two metrics hand off as the corpus fills; they don't compete.

**Secondary: surfaced-and-cooked** (proposals they wouldn't have recalled that got made), and **unprompted weekly reuse**.

**Profile quality:** how often a proposed profile revision gets edited, and whether claims survive review. A profile nobody ever corrects is either very good or never read - check which.

**Constraint, not goal: total cognitive load.** Count generation decisions (choices made from nothing) and verification decisions (proposals reviewed, including accepted ones). Ceiling, not target: the week must not get slower.

**Diagnostic,** logged with the reasoning shown at decision time: strike rate, override rate, and cook-through split by **recipe failed** vs. **week failed**.

**Target:** distinct recipes per month meaningfully above baseline, decisions at or below baseline, in **4 of 6 consecutive weeks**. A four-week conjunction fails on one flu.

## 13. Risks

- **Reconciliation may be unobtainable** (§3). Highest severity for the pantry branch and the retroactive baseline; does not block the core loop.
- **Early weeks are all gamble when the corpus is empty.** Retrieval is risk-free and acquisition is not, and someone starting at zero has only acquisition. Two flops in week one loses them. This household will never encounter that failure mode in testing, which is exactly why it will go unnoticed.
- **Signals aren't attributed.** Two people cook and plan, and a swap by one is recorded as a household preference. Mostly fine, occasionally wrong - if one of them quietly hates mushrooms, the profile will learn "the household is lukewarm on mushrooms" instead. Watch for profile claims that feel half-true; that's the tell.
- **Ungrounded profile claims.** The model will happily produce confident opinions about taste from three data points. No claim without a trace to specific events, and they see the trace.
- **Staleness-only surfacing selects for duds.** "Longest since cooked" preferentially returns recipes that fell out of rotation *for a reason*. The profile has to carry why something stopped being made, not just when.
- **Staleness surfacing has no data yet, and this matters far less than an earlier draft claimed.** That draft said a core §1 mechanism was "unavailable until the tool has been used for months" and that early weeks were therefore "a weaker test of the central bet." **Both statements were wrong and are retracted.** Staleness is not the mechanism that closes the recall gap - having the whole corpus in front of a planner instead of in someone's memory on a Sunday evening is. At 24 recipes the entire corpus goes into context on every call (§10), so the planner can propose across all of it deliberately from day one. The household rotates ~15, which means roughly **9 dormant recipes are surfaceable in week one with zero history**. Staleness is a ranking refinement for corpora too large to reason over wholesale; at this size it barely earns its place, and dates accrue ~5 per session anyway, so coverage is dense within about a month. The real risk here is the opposite of the one first written down: **a claim that the tool needs months to become useful licenses a mediocre first week**, when week one holds the largest stock of forgotten recipes it will ever have.
- **The system will over-read the corpus, and already has.** Reading the seeded corpus, it inferred a sandwich preference from a count of seven and reported it as "a real, distinctive pattern, not noise." The household's answer: *I wouldn't put much weight to this, just happens to be so.* No malice and no hallucination - the count was correct - but a confident preference claim was manufactured from a frequency artifact, which is §2's ungrounded-claim failure arriving through a side door that "no claim without a trace" does not close. **A trace to a real count is still not evidence of a preference.** Counts justify a question, never a conclusion; the profile now carries that instance as a standing caution.
- **Effort was mis-modelled on the first pass**, and the same shape of error will recur. A single scalar looked obviously right and silently excluded the household's easiest meals (§6). The tell was the household volunteering that a question didn't fit - which only happened because the interview ran one question at a time and left room to answer sideways. Batched forms suppress exactly this signal.
- **Breadth fights the sizing model.** The big bag of rice pays off only if they keep cooking rice. Rotating ~32 recipes means shorter, less predictable reorder intervals - though a smaller repertoire than v2.2 assumed makes this milder, and the household's heavy reuse of shortcut staples (seasoning packets, canned soup, biscuit tubes, cream cheese) cuts against it further.
- **Cart API is write-only.** Local state, expect drift.
- **Plans break on Wednesdays.** Replanning out of scope, stated not silent: the plan is immutable after ordering, and the coupling display exists so they can repair by hand.
- **Recipe sites are litigious about scraping.** Only pages the user personally navigated to.
- **Generalizing may sand off the edge.** Earlier drafts held that value lives entirely in learned state, making a stranger's first week a plumbing test and nothing more. That's still true of *retrieval* value, but acquisition value is real from week one, so cold start is now a product path. The cost is that the sharpest version of this document is the one written for a specific household. **That edge got blunter this round, not by generalizing but by measuring:** the household turned out to have ~32 recipes rather than 60, which puts it closer to the middle of its own curve than to the retrieval-heavy end it was designed around. The corollary is that cold start is now the *shared* path rather than the stranger's path, which strengthens §4 and weakens the claim that this household is the sharpest instance of the problem. Keep corpus size out of the architecture; keep the design concrete.

---

## Next session

1. **Run `plan.py` on a real week and cook it.** `profile.md` and `corpus.md` are populated; this is now the binding next step and everything below is secondary to it. The bar is *would you cook this week?* — answered by cooking it, not by scoring the output.
2. **Teach `plan.py` the two-axis effort model and the pool output** before that run, or the first week will filter out stew and pot roast and hand back a grid nobody wants.
3. Start recording last-cooked dates from the first week, since nothing else can start the clock (§13).
4. How many *additional* recipes are in the message thread, beyond the 25? That sizes what's left of the bootstrap and should find the 5–10 unwritten regulars.
5. A sample Kroger order confirmation email, to scope the spike - and check whether it yields a retroactive baseline. This is now the only uncontaminated baseline still available (§12).
6. Write a second `profile.md` for a household with an empty corpus, and see whether the format survives having no evidence to point at. Untouched by this round and still worth doing - more so now that a populated corpus is sitting in the repo tempting Phase 1 to assume one.
