# Domain model and ubiquitous language

Type: grilling
Status: open
Blocked by: —

## Question

What are the nouns of Grocery Router, what does each one mean exactly, and which of them
are the same thing wearing two names?

The old repo grew its vocabulary by accretion across fifteen documents. Several terms now
carry two meanings, and at least two pairs are probably one concept:

- **Recipe, corpus row, candidate, side, bank recipe.** Decision 13 replaced the
  corpus/candidate split with provenance. Does `candidate` survive as a state, or is it
  now `provenance: acquired` plus "no recorded cook"? Is a side a recipe with a flag, or a
  different kind?
- **Yield.** Three shapes (decision 17): adult-equivalents, a portion count, and
  `per portion`. What is the type, and what does scaling do to each shape?
- **Effort.** Two axes (decision 16). The old corpus recorded `low|med|high` while the
  household's ceiling was in minutes, and every rating was the system's guess. Which unit
  wins, and where does the value come from?
- **Week, meal, cook, night, leftover.** Five cooks across seven nights is not five meals.
  A meal that scales covers a second night. What is stored?
- **Outcome.** Kept, nope, didn't cook. Is that three values or two dimensions?
- **Profile claim** and its evidence. What makes a claim well-formed?
- **Ingredient line, item, canonical item, pack.** The parser's output and the shopping
  list's input are not obviously the same type.
- **Decision log entry.** What is the closed set of things that get logged?

Produce a domain model document: each term, its definition, its type, its invariants, and
the terms it is deliberately *not* a synonym for. Record the ones that were merged and why.

Invoke `/domain-modeling`. This ticket blocks most of the map, so it is worth the depth.
