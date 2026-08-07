# Assemble the spec

Type: task
Status: open
Blocked by: 01, 02, 03, 04, 05, 06, 07, 08, 09, 10

## Question

Write the specification, as one document a future agent team can build from without reading
this repository.

This is the destination. Everything above it is a decision that had to be made first.

Required sections:

1. **What this is, and who it is for.** One household. The recall gap. The bet.
2. **What v1 refuses to be**, with the reason next to each refusal. The Out-of-scope list
   on the map is the source. A design doc that only lists upsides is a sales document.
3. **The domain model** (ticket 01) — the vocabulary, verbatim, because every later
   sentence depends on it.
4. **The three steps.** The prep briefing, the weekly session (tickets 02, 03), and the
   list.
5. **The model boundary, as an enforced rule.** What a model may do, what it may never do,
   and the import-graph test that makes *"no model in the shopping list"* a property of the
   build rather than a paragraph nothing checks.
6. **The planner contract** (05) and **the acquisition contract** (06).
7. **Onboarding** (08) and **the import** (07).
8. **The schema** (04), stated as a first cut to iterate on.
9. **The stack and the deployment** (09), including the generated-types rule from decision 4.
10. **The rules, and what enforces each one.** A table: rule, mechanism, test.
11. **The traps**, copied from the map. Every one was a real bug and re-deriving them costs
    more than reading them.
12. **Done**, and how it is measured after it ships.

Two standing requirements for the writing itself:

- **Every decision carries its cost.** That discipline is what made this project's earlier
  reversals cheap to evaluate.
- **State what is unknown as unknown.** The failure this whole product is built to resist
  is a plausible value where there should have been a gap. A spec that guesses is the same
  bug at a larger scale.

Also update or delete the fifteen documents in `docs/`, which now contradict each other and
this spec. Leaving them in place hands the next team the mess this effort removed.
