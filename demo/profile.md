# Household profile

> **This household is invented.** It exists so a hosted demo has something to plan against.
> The real one lives in `profile.md` at the repository root, is not copied into this
> directory, and never reaches a deployment — `app.py` refuses to serve it on a public
> interface at all.
>
> It is written in the same shape as a real profile, including the parts that are
> deliberately blank, because a demo that quietly fills in everything the tool does not
> know would misrepresent how the tool works.

*Read by `plan.py` on every run. Edit it directly whenever it's wrong — correcting this
page is the trust mechanism, and it beats any opaque score (§2).*

**The rule that governs this document: no claim without a trace.** Every line in Taste and
Patterns carries its evidence inline. Right now the honest trace is usually "we know this
about ourselves" — that's self-report, and it's allowed, for the same reason recipes you've
already cooked enter the corpus directly (§4: proven-or-attested). What is *not* allowed is
a claim with no trace at all. If you can't say why you believe a line, cut it.

---

## Members

*Who lives here and can answer for the household. **Attribution, not accounts** — there is
no login. A name on a claim is what lets this profile stop saying "may be half-true".*

- Ada
- Jo

## Hard constraints

*Not preferences. The planner must never violate these, and they need no evidence.*

- **Household:** 2 adults, one 14-year-old. Base ~3 AE per dinner.
- **Guests:** occasional. Pass `--guests` per week rather than setting a standing number.
- **Everyone eats the same meal.** Proposals must be family-edible — a light filter, not
  a design driver.
- **Allergies:** shellfish. No shrimp, crab, lobster or fish sauce containing shellfish.
  Bears on nothing currently in the corpus; it rules out a wide band of Thai and Cajun
  cooking when widening.
- **Hard vetoes:** none stated.
- **Weeknight effort ceiling:** 30 min **active**, Mon–Fri. Friday and Saturday are open to
  something longer.
- **Passive time is not capped.** Slow-cooker meals fit fine. The ceiling is standing at
  the stove, not elapsed time.
- **Store / pickup:** Kroger, `[... location]`

## Taste

*What this household likes. Each claim carries its trace.*

- **Beef runs at about half the repertoire.** 11 of the 23 in the corpus. **No cap is
  applied.** The corpus is the household's own expression of what it wants, and it
  self-corrects: add chicken recipes and the mix shifts on its own. A quota fights that
  signal instead of reading it. *Because it's a count off the corpus, and a count is not a
  mandate.*
- **Cooks with shortcuts on purpose.** Seasoning packets, canned soup, refrigerated
  biscuits and blocks of cream cheese recur across the corpus. A scratch-everything
  proposal is wrong here regardless of how good the recipe is. *Because it's visible in the
  ingredient lists of six of the typed-out recipes.*
- **Soup gets cooked in winter and forgotten by May.** *Because the three soups in the
  corpus were last cooked in January and February and not since.*

## Patterns

*How the week actually runs. Same evidence rule.*

- **Weeknight rhythm:** 30 minutes active, five nights. *Because they said so.*
- **Which nights are hard: unpredictable, week to week.** No recurring squeeze to plan
  around. **This changes the output shape** — assigning meals to named days is wrong here.
  The planner should propose a *pool* with the right effort mix and let the household pick
  night-of. *Because they said so when asked for a pattern.*
- **Effort is two numbers, not one.** Active time is capped; passive time isn't. Stew, pot
  roast and zuppa toscana are low-active and high-passive, and a single effort scalar
  wrongly excludes them from weeknights. *Because they raised slow cookers unprompted.*
- **Leftover behavior: second dinners, rarely lunches.** A big cook can stand in for a
  dinner but does not get taken to work. So **seven nights is not seven cooks** — target
  5 and favour dishes that scale. *Because they said so.*
- **What happens when a plan breaks:** `[...]` — *because `[...]`*

## Known gaps

*What this profile doesn't know. Naming these stops the planner filling them in with
confident invention.*

- Signals aren't attributed to either adult (§13) — a preference stated here is a
  household preference, and may be half-true.
- **The corpus is mains-only.** Vegetables look nearly absent from it, but that's a
  recording artifact, not the diet — sides get served and never get written down. Expect
  generated grocery lists to be **systematically short** until sides are captured.
- **Effort ratings are the system's guess.** Every one is marked unverified; none came
  from the household.
- **Nobody has said whether they want the repertoire widened.** The corpus is narrow in
  cuisine and protein, and that's measured — but "narrow" is only a problem if they think
  it is, and they haven't been asked.

## Not yet known

*Leave this section, and let it be long. It tells the planner where to be tentative. A
profile that knows everything is a profile that made things up.*
