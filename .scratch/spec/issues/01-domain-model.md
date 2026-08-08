# Domain model and ubiquitous language

Type: grilling
Status: **closed** — 7 August 2026. Output: `.scratch/spec/domain-model.md` and `CONTEXT.md`
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

## Outcome

Settled over five rounds of grilling with the household, 7 August 2026.
Full model with types, invariants and the merge record: **`.scratch/spec/domain-model.md`**.
Glossary: **`CONTEXT.md`** at the repo root.

Seventeen questions closed. The answers each cluster reached:

| Cluster | Resolution |
|---|---|
| Recipe / corpus row / candidate / side / bank recipe | One noun, `Recipe`, with `role: main\|side`. `candidate` dies; it splits into immutable **provenance** and derived **untried**. `bank recipe` deleted (v2) |
| The household's own copy | A Recipe is `found` or `adopted`. Acquisition writes a `found` Recipe that is never edited; the duplicate is made **at suggestion time** and is where tolerances and variants live |
| Never surface again | New: **Retirement**. Explicit, reasoned, reversible, and distinct from a flop. Closes the map's open question about an archived state |
| Yield | Four shapes, `Unknown` among them and real. **No consumption model.** Ticket 10 finding 2 closed as an execution error, not a model error |
| Effort | **Minutes stored**, bucket derived for display. Every value records its `source`, and `system-guess` is a first-class value |
| Week / meal / cook / night / leftover | Week is **Sunday to Saturday**. `Meal` and `Cook` are separate nouns; a Cook needs no Meal, which is how an off-plan cook gets seen. `Repeat` is the leftovers event. `night` and `leftover` are not nouns |
| Outcome | **Verdict** — `kept` or `nope` — on a Cook, and optional. `didn't cook` is not a verdict value, and a Meal with no Cook owes no explanation |
| Profile claim | Three nouns: **Claim** (evidence mandatory, carries polarity), **Gap**, **Constraint** (no evidence, enforced). Where a Constraint lives is deferred to ticket 04 |
| Ingredient line / item / canonical item / pack | `IngredientLine` and `ListLine` are two types. `canonical item` merges into **Item**. `pack` deleted with Kroger |
| Decision log entry | `Reason` is **kind plus prose**, both required. The closed set of events stays with ticket 12 |

Two findings in ticket 10 are closed by this ticket — finding 2 (yield and the second night)
and finding 5 (the week boundary). Both are marked there.
