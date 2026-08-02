# Hand-run results

Fill in as you go. The pre-run measurement happens **before** you see any output —
recording it afterward contaminates it.

---

## Pre-run

**Recipes recalled unaided, in one sitting:** `[...]`

This is the §1 premise under test. The proposal assumes ~15 surface out of ~60. If you
reach 40 unaided, the recall gap is smaller than the bet assumes and §1 needs rewriting.
If you reach 8, the gap is wider and the cold-start work matters more than it appears to.

**Date / week planned:** `[...]`
**Model used:** `[...]`

---

## The bar

Answer this first, for both runs, before reading the diagnostics below. One word.

| | Would you cook this week? |
|---|---|
| **Warm run** | `[...]` |
| **Cold run** | `[...]` |

Everything past this point is for fixing a near miss. A clear no on warm means stop and
fix the profile format or the prompt before writing code.

---

## Warm run

**Proposals you'd forgotten existed:** `[...]` of 5

This is the product working (§12: surfaced-and-cooked). Zero here with an otherwise
sensible week means the planner is doing competent retrieval of things you'd have picked
anyway — which is the "buy it again" outcome with more steps.

**Candidates proposed:** `[...]` — **stated count and rationale before the plan?** `[...]`

**Reasons — per proposal, one of:**

| Night | Meal | Reason quality |
|---|---|---|
| Mon | | `specific / generic / invented` |
| Tue | | `specific / generic / invented` |
| Wed | | `specific / generic / invented` |
| Thu | | `specific / generic / invented` |
| Fri | | `specific / generic / invented` |

*Specific* = traceable to a real line in the profile or corpus. *Generic* = true of any
household ("a nice easy weeknight meal"). **Invented** = asserts something about you that
isn't in the inputs and isn't true. Any invented reason is a §13 finding and matters more
than a mediocre week — write down the exact sentence.

**Invented claims, verbatim:**
- `[...]`

**Coupling shown, and was it real?** `[...]`
**Effort ceiling respected?** `[...]`
**Protein / cuisine variety?** `[...]`
**Leftover nights handled as leftovers, with the source meal scaled?** `[...]`

**What you'd swap, and why:** `[...]`

*(Per §6, a swap is a signal about the recipe, not the week — note which it was. If
you'd swap something the planner just told you hadn't been made since March, that is the
staleness-selects-for-duds risk in §13 showing up on the first run.)*

---

## Cold run

**Number of dinners proposed:** `[...]` of a possible 5

**Did it propose fewer than five, and say why?** `[...]`

If it confidently produced five, the "safe first week" instruction in the prompt is not
strong enough to survive a model's default helpfulness. That is a prompt bug, and it is
worth fixing and re-running before drawing any conclusion about cold start.

**Did it invent household facts it was never given?** `[...]`

**Verbatim, the worst offender:** `[...]`

**Were the proposals low-variance and broadly likeable, or was it reaching?** `[...]`

**Would a stranger with this profile cook these?** `[...]`

*(You are judging on behalf of the invented household, not yourself. If you can't answer
this without leaking your own taste in, note that — it is exactly the §13 risk that this
household will never encounter the empty-corpus failure mode in testing.)*

---

## The structural question

**Did the cold profile need a different structure, or just thinner content?**

`[...]`

This is the v2.2 claim under test (§4: cold start is a corpus size, not a second
product). Thinner content confirms it. A different structure falsifies it, and §4, §11,
and the roadmap sequencing all need revising before code is written. Answer honestly —
this is the single most decision-relevant line in the file.

---

## Verdict

Pick one:

- [ ] **Build it.** Warm produced a week worth cooking, cold degraded gracefully.
      Proceed to Phase 1 (cold-start loop) per §11.
- [ ] **Fix the inputs and re-run.** The output was close. Name what to change —
      profile format, prompt, or corpus detail — and run it again. Cheap; do it before
      anything else.
- [ ] **Rethink.** The model can't do this from a profile and a recipe list, and §2's
      central bet is wrong. Write down what specifically failed, because it determines
      whether this becomes a different product or no product.

**What changed in the proposal as a result:** `[...]`
