# The planner contract

Type: grilling
Status: open
Blocked by: 01, 12

## Question

Exactly what does the planner model receive, exactly what may it return, and what happens
to a return that breaks a rule?

Decision 8 removed the deterministic ranker, so this is the only planner. Decision 9 says
it may select and explain. The old repo narrowed that further and the narrowing is the
design: **the model selects and explains; it does not state facts.**

Settle:

- **The input.** The old prompt gave the corpus as a table with a slug column, a computed
  `days since` column, **and no ingredients** — because a model given a gap fills it with
  something plausible, and this one invented ingredient coupling and then justified a pick
  with the invention. What is the v1 input, and what is deliberately withheld?
- **The output.** Slugs plus reasons, with every other field read back off the row? Or
  something richer now that provenance exists?
- **Reason kinds.** The old repo recorded a kind alongside the sentence, because *"which
  reasons get accepted"* cannot be answered from prose — two meals stale at different
  distances are two sentences and one kind. What is the set of kinds?
- **The prediction inside a reason** (decision 23, ticket 12). *"Serves 8 — one cook, two
  nights"* was false the first week it ran, and nothing scored it. Does a reason carry a
  checkable claim beside its prose, and may the model author that claim or only the code?
- **The constraint checks**, which are code and not prompt text:
  - A declared allergen never reaches the week.
  - A slug that resolves to nothing is dropped, never nudged to its nearest neighbour.
  - A reason claiming a recency that no date supports is dropped.
  - A week exceeding the weeknight active-time ceiling says so out loud rather than
    quietly deleting a meal.
- **What happens on a drop.** With no ranker to fill the hole, a dropped pick is a real
  hole. Re-ask? Show fewer meals and say why? This is the question decision 8 created.
- **Partial weeks.** Regeneration fills holes without re-proposing what is already
  accepted. What does the planner need to know about the locked meals?
- **Model and effort.** `claude-opus-5` is the default. Which effort level, and does the
  prompt get cached?

Consult the `claude-api` skill before writing any request shape.
