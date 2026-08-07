# Extract the ingredient grammar and the item table

Type: task
Status: open
Blocked by: —

## Question

What does the existing ingredient parser actually know, and what does `items.md` actually
contain?

**This is the most expensive knowledge in the old repository and the map did not capture
it.** `shop.py` parses all 27 recipe files — 265 ingredient lines — with zero unparseable
lines and zero unrecognised items. That result was earned over several passes and a
rebuild from scratch will re-derive it badly, or silently, unless it is written down first.

The code is no longer in the working tree. Read it at the `prototype` tag —
`git show prototype:shop.py`, `git show prototype:test_shop.py`,
`git show prototype:onboard.py`. `items.md` is still in the working tree, as data.

Extract and record, as a specification rather than as code:

1. **The grammar.** Every line shape the parser accepts, with a real example of each from
   `recipes/`. The known-hard cases are stated in the old design: `juice of 1 lemon`,
   `1 (14.5 oz) can`, fractions, ranges, parenthetical sizes, trailing preparation notes
   (`, diced`), and optional quantities.
2. **What it refuses, and how.** A line the parser cannot read comes out as raw text with a
   flag. It is never dropped and never guessed at. Record the exact failure behaviour.
3. **The item table.** `items.md` — every canonical item, its aisle, its unit, its density
   or conversion, and the `accepts:` tolerances that let a recipe line match it.
4. **Unit reconciliation.** How `3 tbsp oil` becomes a bottle, and per-item conversion.
5. **Scaling.** How a recipe scales against its own yield, across all three yield shapes,
   and what happens when the yield is unknown — the answer is *say it was not scaled*, not
   *assume one*.
6. **Aggregation.** How the same item across several meals becomes one line, how a side
   sharing an onion with a main collapses, and how provenance stays attached to every line.
7. **The mis-merge rule.** `onion powder` matched `onion` across thirteen lines, and
   `dried thyme` matched fresh thyme across five more. A partial match is accepted only
   when every word it leaves behind is noise. Record the exact rule.
8. **The acceptance fixture.** `test_shop.py` holds the week of 2 August as a graded
   fixture. That fixture must survive the rewrite — it is the only end-to-end check made of
   real food.

Output is a document the Go implementation is written against, plus the fixture data in a
language-neutral form.
