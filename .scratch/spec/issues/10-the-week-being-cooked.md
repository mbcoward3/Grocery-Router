# What the week being cooked right now taught us

Type: task
Status: open — two nights outstanding
Blocked by: —

## What this is

The household cooked the week of 3 August 2026 from meals the old tool selected. This is
the only evidence in the project's history that the project did not produce itself. Every
document in `docs/` ends on the same line: *no week has ever been cooked through the tool.*
That stopped being true this week.

Captured 7 August 2026, from the household. Friday night and Saturday night were still
ahead. Close those two rows and this ticket is done.

**This ticket collects facts, not opinions.** The first draft ran a four-question interview
— did the reasons land, was the list right, how did it feel. That was the wrong shape. A
quality signal the system can only get by asking is a signal it will not have in week six.
The interview became ticket 12. See decision 23.

## The record

| Night | Date | Meal | Source | Outcome |
|---|---|---|---|---|
| Sun | 2 Aug | Crock pot Italian beef sandwiches | plan #1 | cooked; no verdict recorded |
| Mon | 3 Aug | Sheet pan chicken fajitas | plan #5, `acquired` | cooked and kept — *"worked out good"* |
| Tue | 4 Aug | — | — | no cook; the household was at an event |
| Wed | 5 Aug | Ground beef tacos | **corpus, off-plan** | cooked; *"an old tried and true"* |
| Thu | 6 Aug | — | — | no cook; Chick-fil-A |
| Fri | 7 Aug | Sausage and peppers | plan #3 | planned for tonight |
| Sat | 8 Aug | Easy salmon dinner | plan #4 | planned for tomorrow |
| — | — | 3-ingredient teriyaki chicken | plan #2 | **never cooked** — see finding 1 |

Five cooks across seven nights. Three of the five plan meals cooked so far, one lost, one
still ahead. One cook came from outside the plan.

## What it taught us

### 1. The list decides what gets cooked. The plan does not.

The teriyaki chicken never happened because its ingredients were never bought. The week's
list was assembled by hand, and it carried 1 lb of chicken breast for the fajitas and no
chicken thighs for the teriyaki. Nobody found the omission until the night.

The failure was not a missing ingredient. It was a missing dinner, and it was silent.

Two consequences:

- **`docs/step2-design.md` §6 needs a rule it does not have.** That section requires the
  list to declare missing sides and unparsed lines. It never requires the list to cover
  every meal in the week. **Add it: every meal contributes at least one line, or the list
  names the meal it could not cover.**
- **This is evidence for the deterministic Step 2, not against it.** A human dropped a
  whole meal from a five-meal list. Code does not make that error.

### 2. The one reason that predicted the future was wrong, and the profile already knew

*"Serves 8 — one cook, two nights."* It gave one dinner and some lunches.

The arithmetic looks sound: 8 AE against a base of ~2.5 AE is three dinners. It failed
because **lunches eat the leftovers**, and nothing in the planner represents that.

`profile.md` states it outright: *"Leftover behavior: both lunches and second dinners. A
big cook covers the next day's lunches **and** can stand in for a dinner."* The household
said this. The planner did not read it, and claimed the dinner anyway.

So **yield alone cannot support a second-night claim.** Ticket 01's yield cluster must
carry a consumption model, or the planner must stop making the claim. Decision 17 is not
wrong; it is incomplete.

> **Closed by ticket 01, 7 August 2026 — as an execution error, not a model error.** The
> household's judgement: the two dinners were available and were not taken. The arithmetic
> supports it — 8 AE against a 2.5 AE base, less two adult lunches, still leaves a dinner.
> **Neither branch above was taken.** No consumption model exists, no `covers:` field exists,
> and the planner keeps the claim. Decision 17 stands unchanged and is not incomplete.
>
> The cost, recorded: a second-night claim now lives entirely in the prose of a `Reason` and
> has no structured half. Nothing in v1 checks whether one held. The only lever ticket 12
> inherits is the reason **kind**, which answers *do `yield` reasons get accepted* and can
> never answer *was this claim true*.

### 3. The acquisition path produced a keeper on its first attempt

Sheet pan chicken fajitas was the tool's first genuine find. Nothing in the household's
saved recipes contained it. It was cooked on the first night of the week and kept.

That is one data point and it is the only one. Do not over-read it — over-reading a count
is on the map's trap list. Record it and let the accept rate accumulate.

### 4. The household reached past the plan for a recipe the tool already held

Wednesday was ground beef tacos. Tacos are corpus row 12. The planner had them in context,
did not propose them, and the household cooked them anyway.

**This is the recall gap failing in the open, and no event in the system can see it.**
There is no way to record *"I cooked something else."* It is the highest-value signal the
week produced and the system is blind to it. Ticket 12 owns this.

> **Closed by ticket 12, 8 August 2026.** A `Cook` needs no `Meal`, and the feedback step
> carries one **"cooked something else?"** row over a corpus picker — two taps, date optional.
> **The off-plan rate is now v1's primary self-measurement**: `Cook`s with no `Meal` as a
> fraction of all cooks. If the dish is not in the corpus, it enters at once as a Recipe with a
> name, `provenance: proven-here` and no ingredient lines. See `.scratch/spec/signals.md` § 3.
> The cost, recorded: the count conflates *the planner missed it* with *we simply wanted tacos*.

### 5. The week the file names is not the week that was cooked

The file says `Week of 2026-08-03`, `nights: 5`, which reads Monday to Friday. The cooking
ran Sunday 2 August to Saturday 8 August. The first meal landed the night before the week
started.

Decision 15 survives this and is strengthened by it: **the week is a pool, not a day grid.**
No day label assigned in advance would have held. But the *boundary* of the pool is
unsettled, and ticket 01's week cluster has to answer it.

> **Closed by ticket 01, 7 August 2026, and the finding mostly dissolves.** A Week runs
> **Sunday to Saturday** and is named by its Sunday, because shopping is a weekend event and
> the week the shopping serves starts when the shopping does. 2 August 2026 was a Sunday, so
> the cooking ran Sun 2 Aug to Sat 8 Aug — exactly one week, and nothing crossed a boundary.
> The file was misnamed by one day; it should read `Week of 2026-08-02`. What survives of the
> finding is a naming bug, not a model gap.

### 6. Two of seven nights had no cook, and neither was the planner's fault

An event and a takeout night. `profile.md` targets 5–6 cooks and predicted this correctly.
Five cooks landed. **That claim holds** and needs no change.

## Contradictions with the map

Per this ticket's standing rule, anything that contradicts a decision gets re-opened in
writing rather than worked around.

| Decision | Status after this week |
|---|---|
| 15 — the week is a pool, not a day grid | **Confirmed.** Nights moved freely; no grid would have survived. Ticket 01 set the pool's boundary at Sunday–Saturday, which the cooked week fits exactly |
| 17 — yield has three shapes | **Unchanged.** Ticket 01 closed finding 2 as an execution error and added no consumption model. Four shapes, counting `unknown` |
| 21 — the reason is the product | **Untested.** Finding 4 shows an unproposed recipe still got cooked. Ticket 12 must make this measurable |
| 8 — no deterministic ranker | **Untouched.** The ranker proposed this week, so nothing here tests the model planner |

## What is still open

- **Friday and Saturday.** Two rows in the record above.
- **The Italian beef verdict.** It was cooked and nobody recorded whether it was kept. That
  gap is itself finding 4 in miniature — there was no capture path, so there is no answer.
