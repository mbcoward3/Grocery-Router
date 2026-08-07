# What the week being cooked right now taught us

Type: task
Status: open
Blocked by: —

## Question

The household is cooking a week from the meals the old tool selected. Results were expected
**8 August 2026**. What happened?

This is the only evidence in the project's history that was not produced by the project
itself. Every document in `docs/` ends on the same line: *no week has ever been cooked
through the tool.* That stops being true this week.

Collect, from the household, one question at a time:

1. **Which meals got cooked, which got skipped, and which flopped?** These are the first
   real `Last cooked` dates and the first real outcomes.
2. **Did the reasons land?** Not *were they true* — that is enforced by code. Did a
   sentence like *"you have not made this since March"* make a forgotten recipe actually
   get cooked? This is the whole product claim and no test can settle it.
3. **Was the list right?** What was missing, what was wrong, what was in the wrong quantity.
   The old list was known to be systematically short because sides were never recorded.
4. **What broke that nobody predicted?** The old repo's most valuable findings all came
   from pointing something at the real world rather than at a fixture.

Feed the answers into the spec. Anything that contradicts a decision on the map gets that
decision re-opened explicitly, in writing — not quietly worked around.

This ticket is unblocked and time-sensitive. Take it first.
