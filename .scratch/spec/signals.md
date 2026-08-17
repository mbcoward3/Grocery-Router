# Signals and self-improvement — Grocery Router v1

Output of ticket 12 (self-improvement). Closed 8 August 2026.

Decision 23 makes self-improvement a design driver and not a feature bolted to the end. This
document is what that means concretely: the surfaces that capture a signal, the closed set of
events, the run that reads them, and — the honest half — the list of quality questions v1 can
never answer.

**The rule this document is written against:** a quality signal the system can only get by
asking is a signal it will not have in week six. Ticket 10 is the evidence. One cooked week
produced four findings and the system could observe exactly one.

**What was already settled before this ticket, and is not re-opened here.** A `Reason` is a
kind plus prose, and it carries no structured prediction (ticket 01). The closed set of
prediction shapes is therefore empty. **Accept rate per kind is the whole of what can be scored
about reason quality**, and *was this particular claim true* has no answer in v1.

---

## 1. The two capture surfaces

A signal is captured in the **Session**, in the **moment**, or in both. The rule governs which:

> **Every signal must be capturable in the Session, retrospectively.** In-the-moment capture is
> an accelerant and never the only path.

The Session is the one visit per Week that is sure to happen. A signal that exists only if
somebody opens the app on Wednesday is a signal the system does not have. The Italian beef
verdict is the proof — there was no capture path, so there is no answer, and there never will
be one.

### The Session

Feedback runs first. It lists last Week's Meals and collects a `Cook`, a `Verdict` and a
`Repeat` against each. Below them sits one row that reads **"cooked something else?"** — see § 3.

### *This week*

One screen, open all Week, carrying the **same affordances as the Session's feedback step**:
mark a `Cook`, set a `Verdict`, tap a `Repeat`, add an off-plan cook. One component, two entry
points, no second design. Ticket 02 draws it once.

### The list

The list's add-item affordance lives on the list, because that tap happens in the aisle. The
list **accepts anything the household types**. If the list refuses paper towels, the household
keeps a second list, and the second list wins.

---

## 2. The closed set of Events

**Two tables, and they never overlap.** State tables hold the nouns — `Recipe`, `Meal`, `Cook`,
`Verdict`, `Repeat`, `Claim`. The `Event` table holds only what has no other home. It never
re-states a `Cook`.

*Adopted, not grilled — see § 10.*

| Event | Carries | The tuning question it feeds |
|---|---|---|
| `proposed` | week, recipe, reason kind, reason prose, planner name | Accept rate per kind — the denominator. A re-proposal writes another row |
| `dropped` | week, recipe, the kind it was shown under | Accept rate per kind — the failure count |
| `verdict-changed` | cook, old value, new value | A judgement that flips. **The one exception to no-duplication**: the row cannot hold both values, and the flip is the finding |
| `list-line-added` | week, item, whether a planned Recipe names it | Step 2 defect count (§ 4.4); the staple flag (§ 5) |
| `retired` / `retirement-reversed` | recipe, reason | What the household refuses, and what it takes back |
| `dial-changed` | week, dial, old value, new value | **Does a dial do anything.** The map's trap list carries a dial that did not. Which dials exist is ticket 02 |
| `asked` | field, question, answered or not | Enforces *at most once* (§ 8). The count of unanswered asks measures how much the system guesses |

**An event that answers no tuning question does not belong on this table.** A tuning question
with no event is a guess forever and is written down as one in § 9.

### What the state tables contribute

The scoring run reads both. These are rows, not events:

| State | The tuning question it feeds |
|---|---|
| `Cook` | Breadth; week size — nights proposed against cooks recorded |
| `Cook` with no `Meal` | **The off-plan rate.** The recall gap, measured directly |
| `Verdict` | Accept rate per kind — the `kept` count. A `nope` is a flop and stays |
| `Verdict` absent | A visible gap, never a default |
| `Repeat` | Whether `yield`-kind reasons produce a second occasion, **in aggregate only** |
| `Cook.effort_actual` | Corrects an `active` range (§ 4.3) |

**On `Repeat` and yield.** A `Repeat` is recorded, so the aggregate question *do `yield`-kind
reasons produce second occasions* is answerable. The specific question *was "serves 8 — one
cook, two nights" true* is not, and never will be, because the claim lives in prose. Do not let
the aggregate stand in for the specific.

---

## 3. The off-plan cook

Ticket 10 finding 4, and the highest-value signal the cooked week produced. The household
cooked ground beef tacos — corpus row 12, in the planner's context, never proposed. Nothing in
the old system could see it.

A `Cook` needs no `Meal`, so the model already allows it. This is the interaction:

1. The feedback step shows one row: **"cooked something else?"**
2. It opens a Recipe picker over the whole corpus. One tap to open, one tap to pick.
3. **The date is optional and defaults to the Week.** A `Cook` with a Week and no date is a
   first-class state. The Week is a pool with no nights in it, so demanding *which night* costs
   a tap and buys nothing.

### When the corpus does not hold it

The household cooked something new. It has a name and no ingredients.

Record the `Cook` at once, against a new Recipe holding a name, `provenance: proven-here`, and
no `IngredientLine`s. **Completeness is derived** from whether any line exists — it is not a new
noun, in the same way `untried` is derived.

- **The planner may never propose an incomplete Recipe.** A Recipe with no lines cannot reach a
  shopping list, and ticket 10 finding 1 is what an uncovered meal costs.
- The next Session asks for the ingredients **once**, and then never asks again (§ 8).
- The Recipe stays, incomplete, and its cook history stays with it. Decision 19.

---

## 4. The four signals that did not exist

`profile.md` already admits every effort rating is the system's guess and that nothing will
correct one. These four rows were the work of this ticket.

### 4.1 The off-plan cook

§ 3. Cost: two taps, in a step the household is already in.

### 4.2 Per-night cooks, for yield

Already carried by `Cook` and `Repeat`. Aggregate only — see § 2.

### 4.3 The effort actual

A **coarse three-state on the `Cook`**, offered beside the `Verdict` that already costs a tap:

```
effort_actual = faster | as stated | slower      (optional)
```

Real minutes will not get typed. The three-state rides on a tap the household already makes,
and it corrects the failure that matters — a range that is systematically wrong, not a range
off by four minutes.

> **The bias, stated:** annoyance drives reporting, so `slower` over-reports. The signal moves
> an `active` range. It never sets one, and it never produces a number.

### 4.4 List completeness

Every hand-added line writes a `list-line-added` Event. **Score only the subset that a planned
Recipe names.**

A hand-added `Item` that appears in a planned Recipe's `IngredientLine`s is a **Step 2 defect**,
and it is checkable with nobody in the loop. Everything else is ordinary shopping. It is counted
separately and never merged into the defect rate. Intent is not knowable, so the system does not
guess at it and builds no category picker.

---

## 5. Origin — how a guess marks itself

**Every field that can be guessed carries an origin.** Ticket 04 makes it a column.

```
origin = stated | observed | system-guess
```

The pattern already exists twice in the domain model — `Claim.evidence_kind`, and the
`system-guess` effort value. This generalises it.

| Field | Origin today | The observation that replaces the guess |
|---|---|---|
| `Recipe` active range | `system-guess` | **`Cook.effort_actual`** (§ 4.3). Direction only, never minutes |
| `Recipe` passive range | `system-guess` | **None exists in v1.** Nobody watches a slow cooker |
| `Item` aisle | `system-guess` | **None exists in v1.** Kroger is out of scope as a whole |
| `Item` staple flag | `system-guess` | **A `list-line-added` Event for a staple.** A staple bought by hand is a staple wrongly assumed present |
| `Item` unit conversion | `system-guess` | **None exists in v1** |
| `PortionConversion` | `stated` only | Never a guess. Asked in place, or it is a `Gap` (§ 8) |
| `Yield` | `stated`, from the source, or `unknown` | `unknown` is a value. It is never inferred, and it is never the extreme |
| `Claim` | `evidence_kind` | `observed-event`, which is the strongest trace the profile can hold |

Four of the eight rows read *none exists in v1*. That is the honest count and it belongs in the
spec.

---

## 6. The scoring run

A replay over the log and the state tables. It answers *is the planner any good* with no
household in the loop.

**It reports, and it never acts.** No auto-tuning of the prompt. No kind suppressed by its own
accept rate. Decision 8 leaves one planner, so there is no second lever to move — and an
automatic control fed by a rate over three proposals is the over-reading trap with a motor on
it. The output lands in the Session as a readable strip. **The household changes the profile.**

### The four numbers

*Adopted, not grilled — see § 10.*

1. **Accept rate per reason kind** — offered, dropped, kept. Three denominators, reported
   separately. A meal that was neither dropped nor cooked is not evidence either way, and
   rolling it into one number invents a verdict for a week nobody finished. Trend against
   itself.
2. **Breadth** — distinct Recipes cooked per quarter. The baseline is the household's stated
   **~15 of 32**, and it is the only number that exists from before the tool.
3. **The off-plan rate** — `Cook`s with no `Meal`, as a fraction of all cooks. **This is the
   recall gap measured directly**, and it is the one metric on this list that measures the thing
   the product exists to fix. The planner held tacos and did not offer them.
4. **The Step 2 defect count** — hand-added lines that a planned Recipe names.

### No baseline exists for number 1

An accept rate against a random-picking baseline is not constructible. Nobody ever offered the
random pick, so nobody accepted or dropped it. There is no counterfactual and v1 does not
manufacture one.

---

## 7. Cold start

n is 1. Every rate here reads noise for months.

**Show counts always. Show a rate only above ten offers per kind.** Below the threshold the line
reads *"3 offers — not enough to say"*, which is a value and not a zero.

This is the map's central trap stated as a display rule: **unknown is not the extreme.** The old
repo scored a recipe with no last-cooked date as maximally stale, and a model called seven
sandwiches "a real, distinctive pattern" when the household called it coincidence.

**The arithmetic, so nobody expects the strip to work in month two.** Five meals a week across
nine reason kinds gives roughly one offer per kind per two weeks. The average kind reaches ten
offers in about **six months**. A rare kind takes longer. Breadth needs a quarter before it says
anything at all.

An uncertainty interval was considered and refused. It is honest and nobody reads it correctly.

---

## 8. Gaming, and the unanswered ask

### Gaming

`plain` scores worst. A planner that sees its own accept rates learns to label everything
`stale`, and the only scorable field becomes a lie.

**The planner never sees an accept rate.** The prompt carries no scores. The model is stateless
across weeks, so it cannot learn to game unless the prompt feeds it the scores — and it will
not. Then code checks the kinds that data can check: `stale` needs a last-cooked date that
supports it, `never` needs no `Cook`. Ticket 05 owns that check.

**The residue, stated:** `plain`, `low` and `passive` stay unverifiable. v1 accepts that.

### The unanswered ask

The rule from `docs/step2-design.md` § 2.1 governs: **never block, never guess, ask in place, at
most once.** Some facts have no observable form — *how many enchiladas is an adult* is one.

**An unanswered ask becomes a `Gap`.** Ticket 01 built that noun for exactly this: a named
absence recorded so the planner does not fill it with invention.

- The system asks once, in place, and writes an `asked` Event.
- If nobody answers, the `Gap` is the record. The Session shows Gaps in a quiet list. It never
  nags.
- A `Cook` with no `Verdict` follows the same path: asked once in the next Session, then a
  permanent gap in the count, and never a default.

---

## 9. What cannot be measured

**The standing requirement of this ticket.** A self-improvement design that claims to observe
everything is the map's central failure at its largest scale — a plausible value where there
should have been a gap. This list is the honest half of the work.

| The question | Why v1 cannot answer it |
|---|---|
| **Was *this* second-night claim true?** | The claim lives in prose. A `Reason` carries no structured prediction, by ticket 01's decision. Only `yield` as a **kind** is scored |
| **Did the prose land?** | Decision 21 calls the prose the product, and **nothing scores the product.** Only the kind is countable. This is the largest gap on the list |
| **Who felt what?** | Decision 7 — no login. No feedback carries a name, so a claim can never be attributed to a member |
| **Would a dropped Meal have been good?** | No counterfactual. A drop ends the evidence |
| **Does the planner beat a random pick?** | No baseline is constructible (§ 6) |
| **Would the household like a Recipe nobody entered?** | The corpus bounds every measurement here |
| **How many minutes did it actually take?** | The three-state gives direction, never magnitude (§ 4.3) |
| **Is a passive range right?** | No observation of any kind exists |
| **Why was there an off-plan cook?** | The count conflates *the planner missed it* with *we simply wanted tacos*. The number is still the best signal v1 has, and it is not clean |
| **What did the week cost?** | Kroger is out of scope as a whole |

---

## 10. Confidence, and what this document does not settle

### Adopted, not grilled

Ten decisions in this document came out of a worked grilling round. Four did not — the household
declined the round and took the recommendation as written. **Treat these four as weaker than the
rest, and re-open any of them cheaply:**

- **§ 2** — the two-table split, and the `verdict-changed` exception to it.
- **§ 6** — the four numbers, and what each compares against.
- **§ 8, gaming** — withholding scores from the prompt.
- **§ 8, the unanswered ask** — an ask becoming a `Gap`.

### Left to other tickets

- **The schema for all of the above.** Ticket 04. § 2 and § 5 are its input.
- **The kind check in code**, and whether `Reason.kind` gains entries. Ticket 05.
- **Which dials exist**, and how the *this week* screen is drawn. Ticket 02.
- **The Session strip that shows § 6.** Ticket 02, then ticket 03.
