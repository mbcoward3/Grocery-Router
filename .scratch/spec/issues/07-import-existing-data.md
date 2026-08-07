# Importing the existing household data

Type: task
Status: open
Blocked by: 04

## Question

How does the data already in this repository become the new app's first state, and what
does it lose on the way?

Decision 14 makes this a spec'd, tested path rather than a one-off script. The data cost
two onboarding passes and a household interview to produce, and it is the only test fixture
in existence made of real food.

What exists today:

- `recipes/` — 27 files, ingredients verbatim, 265 lines, all parsed and recognised by the
  old parser.
- `corpus.md` — 25 rows. Protein, cuisine, yield, effort, notes, source, last cooked.
- `profile.md` — the household interview, with evidence attached to every claim.
- `candidates.md` — acquired but never cooked.
- `sides.md` — empty on purpose.
- `decisions.jsonl` — the log so far.
- `weeks/2026-08-03.md` — one planned week.
- `sources/` — the original inputs: a PDF, URLs, typed text, transcribed screenshots.

Settle:

1. **Provenance for every imported row.** Decision 13. These 25 recipes are *asserted*, not
   proven here — every `Last cooked` is empty. They must be distinguishable forever.
2. **The reason kind for an imported row.** The old code emitted *"no record of cooking
   this yet — unranked, not stale"* for a dateless row. Attached to a recipe the household
   cooks monthly, that sentence is false, and the reason is the product.
3. **The known gaps that carry over.** Seven yields that no source ever stated, and two
   portion rates (how many enchiladas is an adult, and the same for sliders). These import
   as unknown and stay unknown until the household says otherwise. They must not import as
   a guess.
4. **What is dropped.** `candidates.md`, the old week, the old log — do they come across,
   and does the log stay replayable?
5. **The effort ratings.** Every one is the old system's guess, recorded as `low|med|high`
   while the household's real ceiling is in minutes. Import them as guesses, marked.
6. **Re-import or one-shot?** Is this a command that can be run again safely?

The import is a test fixture as much as a migration. Say what it proves.
