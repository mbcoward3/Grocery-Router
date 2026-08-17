# Self-improvement, and the observable signal set

Type: grilling
Status: **closed** — 8 August 2026. Output: `.scratch/spec/signals.md`
Blocked by: — (ticket 01 names the nouns; the questions below do not wait on it)

> **Closed 8 August 2026.** Ten decisions came out of a three-round grilling; four were adopted
> from the recommendation without a round and are marked *adopted, not grilled* in the output.
> The headline: **the off-plan rate — `Cook`s with no `Meal` — is the recall gap measured
> directly**, and it is the only metric v1 has that measures the thing the product exists to
> fix. Two capture surfaces, with the Session guaranteed. Four signals added: the off-plan cook,
> a coarse effort three-state, the Step 2 defect count, and origin marking on every guessable
> field. The scoring run reports and never acts. Section 9 of the output is the unmeasurable
> list, and *did the prose land* is on it.

## Question

Which questions about its own quality can this system answer from what it records, and
which can it never answer without stopping to ask a person?

Then design so the first list is as long as it can be, and the second list is **stated**
rather than discovered in week six.

## Why this is a ticket and not a report

Decision 23. **Self-improvement is a design driver, not a feature bolted to the end.** The
old repo had the instinct — `review.py` already computes accept rate per reason kind from
`decisions.jsonl` — but it arrived as a reporting screen over a log that was designed for
other reasons. This ticket inverts that. **The signal comes first, and the schema, the
planner output and the session are shaped to produce it.**

Ticket 10 is the evidence. One cooked week produced four findings, and the system could
have observed exactly one of them. The other three needed a human to say them out loud,
and in week six nobody will.

## The state today, from ticket 10

| Tuning question | The signal that answers it | Exists? |
|---|---|---|
| Did a reason kind land? | accept rate per kind — offered, dropped, kept | yes, `review.py` |
| Did acquisition earn its place? | `acquired` rows with a kept cook | yes, via provenance |
| Is a recipe stale? | last-cooked dates | yes, automatic |
| Was the week too many nights? | nights proposed vs cooks recorded | yes |
| Is a yield claim right? | the meal's cooks over the days after it | needs per-night cooks stored |
| Is an effort estimate right? | actual hands-on minutes | **no signal at all** |
| Was the list complete? | items added by hand at the shop | **no signal at all** |
| What did the household cook instead? | an off-plan cook, recorded | **no signal at all** |

The bottom four rows are the work. `profile.md` already admits every effort rating is the
system's guess, and nothing in the system will ever correct one.

## Settle

- **The prediction inside a reason.** *"Serves 8 — one cook, two nights"* is a claim with a
  truth value that later events settle. It was false and nobody scored it. Does a reason
  carry a structured, checkable prediction next to its prose? What is the closed set of
  prediction shapes, and what scores each one?

  Weigh this against decision 21 — **the reason is the product**. A sentence written to be
  machine-checkable is a sentence written for a machine. That is a real cost, not a
  formality.

  > **Partly answered by ticket 01, 7 August 2026, and the answer is no.** A `Reason` is a
  > **kind plus prose**, both required, and it carries no structured prediction. The `covers:`
  > field this ticket implies was proposed and refused. So the closed set of prediction shapes
  > is empty, and **accept rate per kind is the whole of what can be scored** about reason
  > quality. Design the rest of this ticket against that constraint rather than reopening it —
  > and note in the output that *was this particular claim true* has no answer in v1.

- **The closed set of recorded events.** Every event, what it carries, and which tuning
  question it feeds. An event that answers no question does not belong. A question with no
  event is a guess forever, and gets written down as one.

- **The off-plan cook.** Ticket 10 finding 4. The household cooked a corpus recipe the
  planner held and did not offer. This is the recall gap failing in the open and it is the
  single most valuable signal the week produced. How does the system see it, and what does
  it cost the household to record it? If the answer costs more than one tap, it will not
  happen.

- **Every guessed value names its replacement.** An effort rating, an aisle, a portion
  conversion. Each one is a guess today. For each, name the observation that would replace
  it — or state plainly that none exists and the value stays a guess for good.

- **The scoring run.** A replay over the log that answers *is the planner any good* with no
  household in the loop. What does it output, what does it compare against, and does a bad
  score change anything on its own or only raise a flag?

- **What still needs asking**, and how the system asks without an interview. Some facts
  have no observable form. *How many enchiladas is an adult* is one. The rule from
  `docs/step2-design.md` §2.1 governs: **never block, never guess, ask in place, at most
  once.**

- **The cold start.** Today n is 1. Every metric here reads zero or noise for the first
  several weeks. What does the system show in that window, and what stops an accept rate
  over three proposals from being treated as a finding? The map's trap list has
  over-reading a count on it already.

- **Gaming.** A planner that learns its predictions get scored can stop predicting. Refusing
  to claim a second night scores better than claiming one and being wrong. What keeps
  honest, falsifiable reasons cheaper than safe ones?

## Standing requirement

**State what cannot be measured as unmeasurable.** The output of this ticket includes a
list of quality questions with no structural answer. That list is the honest half of the
work, and it is the same discipline `profile.md` applies in its Known gaps section. A
self-improvement design that claims to observe everything is the map's central failure at
its largest scale — a plausible value where there should have been a gap.

## Blocks

Ticket 04 — an event nobody records is a column nobody needs.
Ticket 05 — the planner's output shape carries the prediction, or it does not.
Ticket 11 — the spec needs a section for this.
