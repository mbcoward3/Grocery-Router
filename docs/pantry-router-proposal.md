# Project Proposal: Pantry Router

**Decide the week, confirm the list, source the items.**

*Status: draft v2.1*

---

## 1. The bet

A household of two adults, a 3-year-old, and a 1-year-old, plus frequent visitors. Everyone eats the same meal. **Both adults cook, plan, and use the tool equally** - there is no primary operator, so this is one shared brain rather than two profiles that need reconciling.

**The problem is recall, not discovery.** The household has roughly 60 recipes it has tried and likes. Under the stress of picking a week, about 15 surface. The other 45 aren't missing or disliked - they just don't come to mind on a Sunday. The gap between 15 and 60 is the product.

> **Proposals surface known-good meals they wouldn't have remembered, and they get cooked.**

**The opponent is Kroger's "buy it again."** Good at what it does, and a rut-amplifier by construction: it can only return what you recently bought, which is the same 15. The competitive position is breadth, not efficiency. Efficiency is a constraint - don't make the week slower - never the goal.

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

**Three constraints keep it honest:**

**Inspectable.** The profile is a page either of them can read and edit: *you cook fast on weeknights, you've drifted away from Italian, you rarely finish braises.* Correcting it directly is a better trust mechanism than any opaque score. It's a household profile, not a personal one - signals aren't attributed to whoever produced them, which is a simplification, not an oversight (see §13).

**Grounded.** Every profile claim carries its evidence. "Seems to like bright acidic food, based on 6 of your last 10 additions" is useful. The same sentence with nothing behind it is a hallucinated preference that poisons everything downstream. No claim without a trace.

**Gentle.** The profile shifts slower than they can audit it. Revisions are proposed and visible, never silent. Autonomy - the model acting without review - is explicitly out of scope for v1.

## 3. The gating unknown

The inferred pantry, consumption rates, and catalog seeding all need to know **what actually arrived**. Kroger's public API can't supply it: the cart is write-only and order history isn't exposed. Pickup substitutes and shorts items, so "ordered" is never "received."

**Mechanism: parse the Kroger order confirmation emails.** They carry delivered line items, substitutions, and out-of-stocks, automatically, requiring nothing of Sam.

**Phase 1 spike, before further architecture.** If it fails, the pantry and sizing work collapse and the honest response is a planning-only tool that never claims to know what's in the house. Weekly manual confirmation isn't a fallback; it's the chore this exists to delete.

## 4. The corpus

**Getting the 60 in is the highest-value work in the project**, and it has the same failure as the weekly planning: asked to list 60 recipes, they'll produce the 15 they can remember. **You cannot self-report your way out of a recall gap.**

The 45 live in their heads and **in their messages** - links texted back and forth, "let's do the miso salmon," photos of things that worked. The thread already logged what memory can't retrieve.

- **Bootstrap by mining the message thread for recipe links.** One-time extraction, not a feature. Plausibly 20-30 recipes for free
- **Capture at the moment of recall, from either phone.** Recipes come back in the kitchen on a Tuesday, not at an onboarding screen. Adding one takes five seconds from anywhere: a share-sheet target, or forwarding a link. They already share recipes by message, so hook that habit rather than building a new one
- **Let the model prompt laterally** when they do sit down. Not a form - a conversation that works outward from what's already in the corpus, by protein, cuisine, season, and "what do you make when there's no time"
- **Mine reconciliation data** once §3 lands: ingredients bought repeatedly imply recipes not yet in the system

**The corpus is never done, and the system must be useful at 15, 30, and 60.** At 15 there's nothing forgotten to surface, so early weeks lean new and shift toward retrieval as the pool fills. The model handles that transition on its own given the corpus size; it needs no separate onboarding mode.

### Membership is earned

**The corpus is proven-only.** Nothing enters until it has been cooked and liked. That strict bar is what makes the corpus trustworthy as a model input: every item in it is known-good, so surfacing is purely a recall problem and never a quality gamble.

Three states, and nothing is ever deleted:

- **Candidate** - a new recipe the system proposed. Cooked and liked, it enters the corpus. Cooked and it flopped, it's discarded and doesn't come back
- **Corpus** - proven. Deprioritized when consistently passed over, never demoted for a single miss
- **Archived** - proven but repeatedly avoided. Stops surfacing, stays retrievable on request

The 60 existing recipes enter the corpus directly, since they've already cleared the bar in real life. Only genuinely new suggestions pass through candidate.

Archiving rather than deleting is the right fit for a second brain: a brain doesn't erase things, it just stops recalling them. Reaching into the archive on purpose is a feature.

**Setup, ~60 seconds.** Hard constraints only: household in AE, allergies and vetoes, weeknight effort ceiling, store and pickup location. Everyone eats the same meal, so proposals should be family-edible - a light filter, not a design driver.

## 5. Scope

5 dinners a week, which is **not** 5 cooking events - some nights are leftovers by design. Lunch is a byproduct: leftovers become lunches, so certain meals scale up on purpose. Servings in adult-equivalents (~2.5 base, per-meal control for guests); "serves 4" means nothing until converted. Kroger pickup today, Costco later, store layer pluggable. We build the cart, she checks out. No nutrition tracking, recipe hosting, or maintained inventory.

## 6. The three steps

### Step 1 - The week

```
Mon   Chicken thighs + rice          4 AE   [cook big -> Tue]
Tue   Leftovers (Mon)
Wed   Chicken piccata              2.5 AE   [not since March, and you have capers]
Thu   Tacos                          6 AE   [+2 guests]
Fri   Miso salmon                  2.5 AE   [new - close to your sheet pan salmon]
```

Swap, reroll, pin, adjust servings. **Every proposal shows why it surfaced**, which is the tool doing its job in the open and the thing that makes a forgotten recipe land. It also makes rejection legible: swapping out a meal the system just said hasn't been made since March is a signal about the *recipe*, not the week.

**One pool, not tiers.** Anchors and deep cuts are the same 60; the difference is recall frequency, which is a measurement rather than a tag. Surfacing a forgotten one makes it recent and the split re-forms on its own.

Standing planning constraints, given to the model rather than coded: vary protein and cuisine, at least one low-effort night, share perishables across meals and **show the coupling**. Seasonality enters here too, and needs no infrastructure: it isn't a preference, it's a proxy for price and quality, and the model already knows what's good in August. Cross-check it against actual Kroger promo pricing - where the prior and the price agree, the signal is strong. Coupling is the cost of good proposals - independent draws degrade gracefully, a coupled set cascades - so visible links let her repair a broken Wednesday instead of abandoning the week. Sales are opportunistic: flag when something already planned is on sale.

### Step 2 - The list

Ingredients and quantities, scaled to the week's AE totals, aggregated, one screen. Deterministic - no model in this path beyond parsing.

```
Chicken thighs        3 lb
Yellow onion          3
Soy sauce             1/4 cup      [you likely have this]
Rice                  2 cups       [you likely have this]
```

Likely-owned items are flagged, never auto-removed: a silent drop means she gets home without eggs, a visible flag costs one tap.

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
| Cooked-through over months | Which of the 60 are real |
| Servings adjustments | Consumption and guest patterns |
| Strike-outs in Step 2 | What they keep stocked |
| Quantity edits (preference kind) | Consumption rates |
| Ordered but not cooked | Rescue candidates for next week |

The reason chip matters most early, when a bare swap is unattributable and four lessons share one gesture. It matters less as the profile accumulates enough context to guess.

## 10. Architecture

```
search(canonical_item, quantity) -> [{ sku, price, promo_price, pack_size, unit, availability }]
capabilities: { can_write_cart, can_check_stock, has_live_pricing }
```

One store adapter interface, so a new store is a new file. Deterministic pipeline: Ingest -> Parse -> Normalize -> Aggregate -> Suggest -> Source -> Write. Hard parts in order: parsing ("juice of 1 lemon", "1 (14.5 oz) can"), unit reconciliation (3 tbsp oil -> a bottle), SKU matching (40 yogurts).

The model sits above this, at planning and profile revision only.

**No retrieval layer.** Sixty recipes with ingredients and metadata is roughly 40k tokens, so the whole corpus goes into context on every call. That holds through a few hundred recipes, which this household will never exceed. No vector store, no embeddings, no RAG - and writing that down so it doesn't get added reflexively later.

**Week 1 must not be worse than her status quo.** Cold-matching 40 SKUs at the highest-churn moment is how this dies. Early weeks draw from recipes already added, so items are things they already buy and the personal catalog fills from repeats.

## 11. Roadmap

**0 - Baseline.** Watch her real process end to end, including "buy it again," *before* she's seen anything. Count distinct recipes cooked in a month today.

**1 - Corpus + the loop.** Message-thread mining, five-second capture, receipt-email spike. Then profile v0 from setup constraints -> Step 1 -> Step 2 -> Kroger cart.

**2 - Inferred pantry.** Reconciliation feeding staple flags with decay. Gated on the spike.

**3 - Profile revision.** The model proposes profile updates from accumulated evidence; Sam reviews them. Sale flags.

**4 - Multi-store.** Costco, freezer branch, Step 3 rendered.

**5 - Messy ingest.** Screenshots and cookbook photos. Report the parse-quality gap honestly.

**6 - Costco pricing.** Receipt parsing, warehouse side.

## 12. Evaluation

**Primary: repertoire breadth.** Distinct recipes cooked over a rolling quarter, against the ~15 that rotate today. If the 45 stay dormant, the project failed regardless of how good the carts are.

**Secondary: surfaced-and-cooked** (proposals they wouldn't have recalled that got made), and **unprompted weekly reuse**.

**Profile quality:** how often Sam edits a proposed profile revision, and whether claims survive her review. A profile she never corrects is either very good or never read - check which.

**Constraint, not goal: total cognitive load.** Count generation decisions (choices made from nothing) and verification decisions (proposals reviewed, including accepted ones). Ceiling, not target: the week must not get slower.

**Diagnostic,** logged with the reasoning shown at decision time: strike rate, override rate, and cook-through split by **recipe failed** vs. **week failed**.

**Target:** distinct recipes per month meaningfully above baseline, decisions at or below baseline, in **4 of 6 consecutive weeks**. A four-week conjunction fails on one flu.

## 13. Risks

- **Reconciliation may be unobtainable** (§3). Highest severity; resolve before further architecture.
- **Signals aren't attributed.** Two people cook and plan, and a swap by one is recorded as a household preference. Mostly fine, occasionally wrong - if one of them quietly hates mushrooms, the profile will learn "the household is lukewarm on mushrooms" instead. Watch for profile claims that feel half-true; that's the tell.
- **Ungrounded profile claims.** The model will happily produce confident opinions about taste from three data points. No claim without a trace to specific events, and Sam sees the trace.
- **Staleness-only surfacing selects for duds.** "Longest since cooked" preferentially returns recipes that fell out of rotation *for a reason*. The profile has to carry why something stopped being made, not just when.
- **Breadth fights the sizing model.** The big bag of rice pays off only if she keeps cooking rice. Rotating 60 recipes means shorter, less predictable reorder intervals.
- **Cart API is write-only.** Local state, expect drift.
- **Plans break on Wednesdays.** Replanning out of scope, stated not silent: the plan is immutable after ordering, and the coupling display exists so she can repair by hand.
- **Recipe sites are litigious about scraping.** Only pages the user personally navigated to.
- **Onboarding is a reviewability test, not validation.** A stranger reaching a cart proves the plumbing generalizes and says nothing about value, which lives in learned state.

---

## Next session

1. Watch her plan and order a week, before showing her anything.
2. How many recipe links are in the message thread? That sizes the bootstrap.
3. A sample Kroger order confirmation email, to scope the spike.
4. Write profile v0 by hand, as if the system had produced it. If you can't write a useful one from what you already know, the format is wrong.
