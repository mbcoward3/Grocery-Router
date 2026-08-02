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
- **Allergies:** `[...]` *(none is a valid answer — write "none")*
- **Hard vetoes:** `[...]` *(foods that are never proposed, ever)*
- **Weeknight effort ceiling:** `[...]` *(e.g. "45 min active, Mon–Thu")*
- **Store / pickup:** Kroger, `[... location]`

## Taste

*What this household likes. Each claim carries its trace.*

- `[...]` — *because `[...]`*
- `[...]` — *because `[...]`*
- `[...]` — *because `[...]`*
- `[...]` — *because `[...]`*

> Worked examples of the shape, not entries to keep:
> - *Reaches for bright, acidic food — lemon, capers, vinegar-forward dressings. Because
>   four of the recipes in the corpus are built on it.*
> - *Drifted away from Italian in the last year. Because we can name five Italian recipes
>   we like and haven't made any since roughly March.*
> - *Rarely finishes braises. Because we start them on weekends and eat them twice.*

## Patterns

*How the week actually runs. Same evidence rule.*

- **Weeknight rhythm:** `[...]` — *because `[...]`*
- **Which nights are hard:** `[...]` — *because `[...]`*
- **Leftover behavior:** `[...]` — *because `[...]`*
  *(Five dinners is not five cooking events. Say which nights are leftovers by design and
  which meals get scaled up to become lunches.)*
- **What happens when a plan breaks:** `[...]` — *because `[...]`*

## Known gaps

*What this profile doesn't know. Naming these stops the planner filling them in with
confident invention.*

- Signals aren't attributed to either adult (§13) — a preference stated here is a
  household preference, and may be half-true.
- `[...]`

## Not yet known

*Leave this section, and let it be long. It tells the planner where to be tentative. A
profile that knows everything is a profile that made things up.*
