# Findings — the rest of the brief

*Covers §3 onboarding, §4 Kroger, §5 session depth, §6 sides, §7 the decision log, and the
acquisition adapter framework. §1 and §2 have their own reports.*

---

## What got built

| | |
|---|---|
| §1 model planner | `planner/` — two implementations behind `pantry.propose()` |
| §2 acquisition | `acquire/` — five search strategies, all eleven sources reachable |
| §3 onboarding in the app | a box in the session, through the same door acquisition uses |
| §4 Kroger | `adapters/` — prices, promotions, SKU matching; no cart write |
| §5 session depth | servings, swap, lock, reshuffle, read the recipe, edit the profile |
| §6 sides | `sides.md`, three routes in, empty on purpose |
| §7 the decision log | `review.py`, and behaviour numbers in the session |

317 tests, standard library only, none of them needing a key, a credential or a network.

## The bugs that only running it could find

Every one of these was invisible to reading the code, and most were found by pointing
something at the real world rather than at a fixture.

**The top-up deleted the leftover night.** Two subagents given the real planner prompt both
returned four cooks for five nights *on purpose*, with a meal scaled to cover a second
dinner — which `profile.md` asks for. `propose()` topped it up to five and silently deleted
the plan. My own stubs never caught it because I wrote them, and I wrote them full.

**`searchable("fish")` returned `"f"`.** The regex stripping the corpus's `-ish` hedge had
no hyphen anchor. Nine recipe sites were asked for a single letter.

**`assess` approved blueberry muffins as dinner.** Full-text search returns cake, and
nothing about a muffin is unsafe.

**`2 swordfish steaks` inferred *beef*.** First-protein-wins over a bare `steak` marker.
This one was in `onboard.py`, not in acquisition — capturing a swordfish recipe by URL
would have written *beef* into the corpus index, quietly, and had been able to for months.

**Three sources were being scraped against their `robots.txt`.** `southernbite.com`,
`spendwithpennies.com` and `onceuponachef.com` all `Disallow: /wp-json/`, and the first
implementation hit it anyway. Fixing that surfaced a second one: `RobotFileParser.read()`
fetches with Python's default user agent, these CDNs answer it with 403, and CPython reads
a 403 on `robots.txt` as *disallow everything* — so nine working sources went unreachable
in one commit, all incorrectly.

**`onboard.py` contains `</script>`.** It parses JSON-LD out of recipe pages. Adding it to
the browser payload closed the payload tag early and made `dist/index.html` unparseable
JSON — a blank page on a public URL, behind a green build. `smoke_static.py` caught it, as
it had caught `mkdirTree('/app')` two commits earlier.

**The test suite wrote into the real `sides.md`.** Four harnesses did not repoint
`pantry.SIDES`. CI's demo-leak check runs after the app and would not have caught it; there
is one that runs after the tests now.

**Reshuffle was a button that changed nothing.** The ranker is deterministic, so re-running
it with the same locked meal returns the same week. It had to *decline* what it re-rolled
to mean anything — the same trap as the risk dial nudging a score in a fight candidates
lose by design.

## The decisions worth keeping

**Searching is an interface, not an endpoint.** Acquisition started on `/wp-json/wp/v2/
search`, which nine of eleven sources happened to expose. That made "can this site be
searched" a property of the site's CMS rather than of this code. Five strategies now run
from *the site told us how* to *we worked it out*, and all eleven sources are reachable.

The reason the loose end of that ladder is safe is where the invention bar actually sits:
**an adapter only ever proposes URLs.** Nothing in `acquire/adapters.py` parses a recipe.
`onboard.from_url` refuses any page without machine-readable schema.org data, so the
sloppiest scrape costs a wasted request and can never cost a wrong ingredient. Guess loose,
verify hard.

**A match that is not confident is not made.** The Kroger rule, and the sharpest version of
something this project keeps arriving at. `onion powder` → `onion` was a *cheap* mistake —
one wasted vegetable in a list a human reads first. Against a cart it is money spent on an
item nobody chose. A gap in a cart is a smaller failure than a stranger's guess in it.

**Decision 4 stopped being a promise.** *The tool fills a cart and a human submits it* is
now a property of Kroger's auth model: catalogue reads come from client credentials, cart
writes need a user token from a browser redirect. There is no credential in this codebase
that can spend money.

**`sides.md` is empty and stays that way.** Seeding it with ten plausible vegetables would
make the tool look finished and every grocery list wrong in a new way. The machinery is
built; the file is the household's to fill.

**Reason kinds are recorded, not just reason text.** §7 asks which reasons get accepted,
and the sentence alone cannot answer it — two meals stale at different distances are two
sentences and one kind.

## What the schema still cannot express

- **Family-edible has no honest mechanical test.** Corpus membership is the proxy, and that
  is why the model may only pick from the catalogue.
- **A vegetarian main is refused along with the cake.** The no-protein gate is blunt in
  exactly one direction. It is skipped for a recipe somebody pasted, since a person
  choosing a link is not noise — but acquisition still cannot go looking for one.
- **The active ceiling is in minutes and the corpus records `low|med|high`.** Nine of
  twenty-four rows are `med` and every rating is the system's guess.
- **Cuisine inference is weaker than protein inference**, so cuisine gaps are the least
  reliable kind of acquisition.
- **Relevance has no measure.** Whether *Fish Taco Bowls* is a good suggestion for this
  household is not a thing any check here can answer. One cook settles it.

## Questions worth asking the household

1. **Do the model's reasons land?** Not *are they true* — that is enforced. Five model
   reasons and five ranker reasons for the same week, unlabelled, is the cheapest version
   of this test, and `./plan.py --week --planner ranker|model` produces both.
2. **What are your sides?** Ten minutes of typing closes the largest correctness gap in the
   product. Nothing else in this repo can close it.
3. **Are these the right sites?** The search surface is inferred from what got cooked,
   which is a proxy for trust and not the same thing.
4. **Do you want cuisine widened?** Still unanswered from `profile.md`, and it now gates a
   whole class of acquisition. The tool will not act on it unasked.
5. **Is `med` inside the weeknight ceiling?**
6. **What happens when a plan breaks?** Still `[...]` in `profile.md`.

## Still true, and the most important line here

**No week has ever been cooked through the tool.** Every `Last cooked` is empty, no
proposal has ever been judged, and nothing acquired has been promoted.

The tool can now plan with a model, find recipes nobody bookmarked, capture them from a
paste, price them at a store, and read its own history back. **All of it is running on zero
weeks of evidence.** `review.py` reports honestly on a log with one household-week in it —
every number correct, none of them meaning anything yet.

The brief said nothing on its list mattered as much as one real week. Seven items later
that is still true, and it is now the only thing left that this repo cannot do for itself.
