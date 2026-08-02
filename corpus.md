# Corpus

Recipes this household has cooked and liked. Read by `plan.py` on every run.

**Membership is earned (§4).** Nothing goes in here until it's been cooked and liked.
That strict bar is what makes the corpus trustworthy as a planner input: everything in it
is known-good, so surfacing one is a recall problem and never a quality gamble. A recipe
you've bookmarked but never made does not belong here — that's a candidate, and the
planner proposes those on its own.

**Seeding it:** one sitting, no notes, no scrolling your messages. Whatever you reach
unaided is the starting corpus. Write that number down before you run `plan.py` the first
time — once you've seen it propose a week you can never measure it again, and it's the
only number that tells you later whether the tool did anything or just agreed with you.

**No ingredient lists, deliberately.** They feed the shopping list, which is a separate
deterministic step. The planner doesn't need them — it already knows roughly what goes in
chicken piccata.

**Growing it is the point, not a chore.** Add a line whenever one comes back to you, and
whenever a candidate gets cooked and kept.

## Format

`Last cooked` can be as vague as "spring?" — vagueness is honest and the planner handles
it. Leave `Notes` empty unless there's something a stranger wouldn't guess.

| Recipe | Protein | Cuisine | Effort | Last cooked | Notes |
|---|---|---|---|---|---|
| e.g. Chicken piccata | chicken | Italian | medium | March? | kids eat it if capers are optional |
| e.g. Miso salmon | fish | Japanese | low | last month | sheet pan, Wednesday food |

**Effort** is your weeknight scale, not an objective one: `low` = you'd do it on a bad
Tuesday, `medium` = fine any night with a plan, `high` = weekend or company.

*(The `e.g.` rows above are ignored by `plan.py`. Delete them or leave them — add your
real entries below.)*
