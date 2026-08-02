# Household profile

*Read by `plan.py` on every run. Edit it directly whenever it's wrong — correcting this
page is the trust mechanism, and it beats any opaque score (§2).*

**The rule that governs this document: no claim without a trace.** Every line in Taste and
Patterns carries its evidence inline. Right now the honest trace is usually "we know this
about ourselves" — that's self-report, and it's allowed, for the same reason recipes you've
already cooked enter the corpus directly (§4: proven-or-attested). What is *not* allowed is
a claim with no trace at all. If you can't say why you believe a line, cut it. A profile
that quietly accumulates ungrounded assertions is the §13 failure mode, and it poisons
every week downstream.

---

## Hard constraints

*Not preferences. The planner must never violate these, and they need no evidence.*

- **Household:** 2 adults, a 3-year-old, a 1-year-old. Base ~2.5 AE per dinner.
- **Guests:** frequent. Pass `--guests` per week rather than setting a standing number.
- **Everyone eats the same meal.** Proposals must be family-edible — a light filter, not
  a design driver.
- **Allergies:** peanut. No peanuts, peanut butter, peanut sauce or peanut oil as an
  ingredient. Trace-risk and shared-facility products are acceptable — this filters the
  recipe, not the pantry. Bears on nothing currently in the corpus, but rules out satay
  and a wide band of Thai and Chinese dishes when widening, and the stir fry and teriyaki
  are the two existing entries where a bought sauce could carry it.
- **Hard vetoes:** none stated.
- **Weeknight effort ceiling:** 20–30 min **active**, Mon–Fri. One or two weekend nights
  (usually Sat/Sun) are open to something longer and nicer.
- **Passive time is not capped.** Slow-cooker meals fit fine. The ceiling is standing at
  the stove, not elapsed time — see the effort note under Patterns.
- **Store / pickup:** Kroger, `[... location]`

## Taste

*What this household likes. Each claim carries its trace.*

- **Beef runs at about half the repertoire, and that's more than they'd choose.** 12 of 25
  recipes. Asked directly whether this was preference or rut, the answer was "somewhere
  between" — so the planner applies a **soft cap of 3–4 beef nights a week** and fills the
  rest from chicken, pork and fish. It does not push beef away harder than that. *Because
  they said so when asked, against a corpus count.*
- **Cooks with shortcuts on purpose.** Seasoning packets, canned soup, refrigerated
  biscuits, blocks of cream cheese recur across the corpus. A scratch-everything proposal
  is wrong for this household regardless of how good the recipe is. *Because it's visible
  in the ingredient lists of the six typed-out recipes.*
- **Cuisine is narrow: roughly 18 of 25 are plain American comfort food**, plus 3
  Italian-American, 2 Tex-Mex, 2 loosely Asian. Recorded as a fact about the corpus, not
  yet as a preference — nobody has said whether they want it widened. *Because it's a
  count off the corpus.*

> **Not** a pattern: the seven sandwiches. That count looked like a signal and isn't —
> asked directly, the answer was "I wouldn't put much weight to this, just happens to be
> so." Left here as a caution, because it is exactly the kind of number this project is
> built to over-read.

> Worked examples of the shape, not entries to keep:
> - *Reaches for bright, acidic food — lemon, capers, vinegar-forward dressings. Because
>   four of the recipes in the corpus are built on it.*
> - *Drifted away from Italian in the last year. Because we can name five Italian recipes
>   we like and haven't made any since roughly March.*
> - *Rarely finishes braises. Because we start them on weekends and eat them twice.*

## Patterns

*How the week actually runs. Same evidence rule.*

- **Weeknight rhythm:** 20–30 minutes active, five nights. One or two weekend nights have
  real room. *Because they said so.*
- **Which nights are hard: unpredictable, week to week.** No recurring squeeze to plan
  around. **This changes the output shape** — assigning meals to named days is wrong here.
  The planner should propose a *pool* with the right effort mix and let the household pick
  night-of. *Because they said so when asked for a pattern.*
- **Effort is two numbers, not one.** Active time is capped; passive time isn't. Stew, pot
  roast and zuppa toscana are low-active and high-passive, and a single effort scalar
  wrongly excludes them from weeknights. The corpus records both. *Because they raised
  slow cookers unprompted as not fitting the question.*
- **Leftover behavior: both lunches and second dinners.** A big cook covers the next day's
  lunches *and* can stand in for a dinner. So **seven nights is not seven cooks** — the
  planner should target 5–6 and favor dishes that scale. *Because they said so.*
- **What happens when a plan breaks:** `[...]` — *because `[...]`*

## Known gaps

*What this profile doesn't know. Naming these stops the planner filling them in with
confident invention.*

- Signals aren't attributed to either adult (§13) — a preference stated here is a
  household preference, and may be half-true.
- **The corpus is mains-only.** Vegetables look nearly absent from it, but that's a
  recording artifact, not the diet — sides get served and never get written down. Two
  consequences: don't read the corpus as evidence about how this household eats, and
  expect generated grocery lists to be **systematically short** until sides are captured.
- **The repertoire is about 30–35, not 25.** Five to ten regulars were never written down.
  The corpus is a floor.
- **No last-cooked dates exist.** Staleness-based surfacing — a core §1 mechanism — can't
  run until the tool has been used long enough to record them.
- **Effort ratings are the system's guess.** Every one is marked unverified; none came
  from the household.
- **Nobody has said whether they want the repertoire widened.** The corpus is narrow in
  cuisine and protein, and that's measured — but "narrow" is only a problem if they think
  it is, and they haven't been asked.

## Not yet known

*Leave this section, and let it be long. It tells the planner where to be tentative. A
profile that knows everything is a profile that made things up.*
