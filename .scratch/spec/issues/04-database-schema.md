# The first-cut database schema

Type: grilling
Status: open
Blocked by: 01 ✓, 12-self-improvement ✓

> **Inputs now waiting.** `.scratch/spec/signals.md` § 2 gives the closed `Event` set and the
> rule that state tables and the log never overlap. § 5 adds an `origin` column
> (`stated | observed | system-guess`) to every guessable field. Both are marked *adopted, not
> grilled* and are cheap to re-open.

## Question

What tables does v1 have, and which invariants does the schema enforce rather than the
application?

The old repo used markdown files and enforced its rules in one write module, because files
do not refuse a bad write. Postgres does. So the question is which of those rules become
constraints and which stay in code.

Settle:

- **The tables**, from the domain model. Recipes, ingredient lines, items, corpus
  membership with provenance, weeks, meals, outcomes, profile claims, sides, the decision
  log.
- **The rules, as constraints.** The old repo's table of intentions:
  - *No claim without a trace* → `evidence NOT NULL`.
  - *No writer overwrites a human value* → per-cell checks plus row-level history.
  - *Nothing is ever deleted* → no `DELETE`, a state column instead.
  - *Membership carries provenance* → a non-null enum, not a nullable flag.
- **History.** Git was the audit log for free and Postgres is not. Which tables need an
  append-only history, and does the session show it?
- **The decision log.** It was `decisions.jsonl`, append-only, one object per line. Is it a
  table, or does it stay a file? Its argument was never about storage: *a decision that was
  not recorded cannot be recovered.*
- **Migrations.** Ordered files, applied by a Job before the app rolls.
- **Export.** A household must be able to take its data and go. What format?

Write the schema as SQL. It is a first cut to iterate on, not a final answer — say plainly
which parts you expect to change.
