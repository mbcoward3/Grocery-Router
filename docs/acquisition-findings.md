# Findings — acquisition

*Same standard as the onboarding briefs and `docs/model-planner-findings.md`.*

Task §2 from `docs/brief-next.md`. The proposal calls acquisition half the job, and it was
0% built: `candidates.md` had three rows and a human typed all three. The tool had never
searched for a recipe, never read a page it was not handed, and never judged fit.

---

## What got built

`acquire.py` — sources, search, gaps, capture, judge, write. `pantry.add_candidate()` as
the write door. A *Find something new* button in the session, and `./acquire.py` from a
terminal. 37 new tests, no network. 209 tests total.

## The decision that mattered: where it looks

Not a search engine. **The sites the household already cooks from**, read off the `Notes`
and `Source` columns of `corpus.md` and `candidates.md`. Nine of the eleven expose the
WordPress REST search API at `/wp-json/wp/v2/search` — a documented public endpoint
returning real posts.

That was not the obvious choice and it is the right one:

- **No key and no dependency.** `urllib` and `json`. The project's "standard library only"
  property is defended by CI, and a search API key would have been the first crack in it.
- **Every result is a real page**, by construction. The bar was *never invent a recipe*,
  and an endpoint that returns posts from a site cannot return a post that does not exist.
- **It is not scraping.** A site that publishes a search API has agreed to be searched. One
  that has not — `tasty.co` 404s — is skipped rather than having its URL structure guessed,
  which is the same refusal `onboard.from_url` already makes when it declines to read a
  recipe off page prose.
- **The surface grows on its own.** The domains come from the corpus, so promoting a recipe
  from a new site puts that site in next week's search. A hardcoded list would have gone
  stale immediately, and it would have been *me* deciding whose food this household likes.

**No model runs in acquisition**, and that is a choice. Every fit signal that matters is
computable: peanut off the capture, active off the source's stated prep time, protein and
cuisine off the ingredients, duplicates off the corpus. What a model would add is taste,
and the household supplies that by cooking the thing. The one place it would genuinely help
is writing the query, and a bad query costs a wasted search rather than a wrong claim.

## What the live runs found

Four bugs, all from pointing it at the real web. Every one was invisible to reasoning about
the code.

**`searchable("fish")` returned `"f"`.** The regex stripping the corpus's `-ish` hedge —
`Chinese-ish`, `Japanese-ish` — had no hyphen anchor, so it ate the `ish` off `fish` and
sent a single letter to nine recipe sites. Caught because the run returned muffins.

**A well-covered week produced no gap at all**, so acquisition did nothing. But the planner
prompt is explicit that at this corpus size acquisition is part of the job *every* week and
not an occasional flourish — 24 recipes cannot fill a year. There is now a `breadth` gap
that fires when the week wants nothing, aimed at the thinnest protein in the corpus.
Deliberately **protein and never cuisine**: `profile.md` records the cuisine narrowness as
measured fact and then says nobody has been asked whether they want it widened, and acting
on an unanswered question is the failure that profile exists to prevent.

**`assess` approved blueberry muffins as a dinner candidate.** The search is full-text, so
asking for `fish` returned muffins, a coffee cake, a fruit salad and a clafoutis — and
every one passed every hard constraint, because nothing about a muffin is unsafe. It is
just not dinner. A capture with no identifiable protein is now refused.

**`2 swordfish steaks` inferred *beef*.** This is the one worth keeping. `onboard.py`'s
`infer_protein` walked proteins in list order and returned on the first hit; the beef list
carries a bare `steak`, and `steak` is a substring of `steaks`. It is precisely the
`onion powder` → `onion` failure again: a partial match accepted while a more specific one
was available. It now takes the **longest matching marker** rather than the first protein,
so `swordfish` (9 characters of evidence) beats `steak` (5).

That bug was not in acquisition. It was in onboarding, where it had been sitting quietly —
capturing a swordfish recipe by URL would have written *beef* into the corpus index. It was
found only because acquisition pointed the same function at pages nobody had hand-picked.

## What worked first time

- **The peanut refusal fired on live open-web data.** Asked for `fish`, the sources
  returned *Thai Massaman Beef Curry* with half a cup of roasted peanuts. Refused, by name,
  with the ingredient line quoted. That is a hard constraint from `profile.md` working
  against a page nobody had vetted, which is the case it was actually written for.
- **The capture is complete enough for the shopping list**, which was in the brief's *done
  means*. The landed recipe carried 20 parsed ingredients, a stated yield of 8 AE, and a
  source line — enough for Step 2 to shop it with no human in between.
- **Active time is read off the source's stated prep**, and says so. Every effort rating in
  `corpus.md` is the system's unverified guess and `profile.md` says as much; these are the
  first ones with a page behind them, and they are still marked inferred.

## What the schema could not express

- **A vegetarian main is refused along with the cake.** The no-protein rule is blunt, and
  it costs exactly this. `profile.md` already records that the corpus is mains-only *and*
  that vegetables are a recording artifact rather than an absence — so a vegetarian dinner
  is a real gap here, and it belongs to §6, not to a keyword patch inside `assess`.
- **Cuisine inference is weaker than protein inference.** The landed fish taco bowl came
  back with an empty cuisine. Not harmful — the planner treats a blank as unknown rather
  than as an extreme — but it means cuisine gaps are the least reliable kind.
- **`Proposed` has two formats now.** The human rows say `wk of 2 Aug`; the tool writes
  `wk of 2026-08-03`. Cosmetic, and left alone rather than rewriting rows a person typed.
- **Relevance has no measure.** Whether *Fish Taco Bowls* is a good suggestion for this
  household is not something any check here can answer. The candidate carries the gamble,
  which is what `candidates.md` is for, and one cook settles it.

## Questions that answer themselves

- *Should it search the whole web?* Not yet. The corpus-derived surface is the household's
  own signal about sources, it is nine real sites, and widening it is a decision somebody
  should make on purpose.
- *Should acquisition put things straight in the corpus?* No — membership is earned, and
  acquisition is the one path most likely to forget it, since it is the one that adds
  recipes. There is a test.

## Questions worth asking the household

1. **Are these the right sites?** The list is inferred from what got cooked, which is a
   proxy for trust and not the same thing. There may be a site they like that never made it
   into the corpus, and one in the corpus they have gone off.
2. **Do you want cuisine widened?** Still the open question from `profile.md`, and it now
   gates a whole class of acquisition. The tool will not act on it unasked.
3. **How many candidates a week is right?** It lands one by default. The prompt says the
   count is a judgment about corpus size and risk appetite, and nobody has said what feels
   like too many unproven dinners in a row.

## Still true

**No week has ever been cooked through the tool.** Acquisition can now widen the corpus,
and nothing widens it in the way that counts — a candidate is promoted by being cooked and
kept, and that has never happened. The tool can now find recipes faster than the household
can prove them.
