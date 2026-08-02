# Hand-run: the planner, before any code

**Question this answers:** if you paste a household profile and a recipe corpus into a
model with this week's constraints, do you get five dinners you would actually cook?

If yes, the loop in §6 is worth building. If no, no amount of engineering saves it, and
the thing to fix is the profile format or the prompt — both of which are text files you
can iterate on in an afternoon.

## Why this before code

The proposal's whole bet (§2) is that the planner is a model with priors, not a scoring
function. That bet is testable right now, by hand, with no repository, no API key, and
no Kroger integration. Everything downstream — Step 2, the cart write, the pantry — is
deterministic plumbing that only pays off if Step 1 produces a week worth cooking.

## The two runs

Run it twice. The comparison is the point.

| Run | Profile | Corpus | Tests |
|---|---|---|---|
| **Warm** | `profile-v0-warm.md` | `corpus.md` (your ~15) | Retrieval — §12's primary metric at high corpus size |
| **Cold** | `profile-v0-cold.md` | *empty* | Acquisition — the v2.2 cold-start claim in §4 and the risk in §13 |

Same week constraints for both. Warm should surface things you'd forgotten and lean
proven. Cold should propose fewer than five, lean low-variance, and never pretend to
know something it wasn't told. If cold produces five confident dinners with invented
reasons, §4's "safe first week" requirement is not being met by the prompt and needs to
be enforced harder.

## Steps

1. Fill in `profile-v0-warm.md`. Blanks are marked `[...]`. Twenty minutes, no research —
   if you have to think hard about a line, the honest answer is usually "we don't know
   yet," and that is a legitimate entry.
2. Fill in `corpus.md` with every recipe you can recall in one sitting. Don't stretch for
   60. The count you reach unaided is itself a measurement — write it down in
   `results.md` before you start.
3. Fill in the week block at the top of `planner-prompt.md` with the real constraints
   for an actual upcoming week. Not a hypothetical one; the test is whether you'd cook it.
4. Paste `planner-prompt.md` + profile + corpus into a fresh chat. Save the output.
5. Repeat with `profile-v0-cold.md` and no corpus, in a **fresh chat** — a warm run
   already in context contaminates the cold one completely.
6. Score both in `results.md`.

## The bar

One question decides this, and it is not "is the output impressive":

> **Would you cook this week?**

Not "is it plausible," not "did it follow the format." Everything in `results.md` past
that question is diagnostic detail for fixing a *near* miss. A clear no on the main
question means stop and fix the inputs before writing code.
