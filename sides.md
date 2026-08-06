# Sides

Vegetables and starches this household actually serves. Read alongside `corpus.md`.

**This file is empty on purpose, and that is the whole point of it.** `profile.md`
records that vegetables look nearly absent from the corpus but that this is a *recording
artifact, not the diet* — sides get cooked here and never got written down. Every grocery
list this tool has ever produced has been systematically short because of it.

So the machinery reads this file, the session proposes from it, and the shopping list
includes whatever is in it. What it will not do is guess. A seeded list of ten plausible
vegetables would make the tool look finished and make every list wrong in a new way — the
same failure as a recipe with `1 lb chicken breast` because soup usually has chicken. **A
side belongs here once somebody says it does.**

## How a row gets here

Three ways, and all of them end in a real ingredient list:

- `./onboard.py --url <link> --side` — capture one from a page, like any recipe.
- `./acquire.py --sides` — the tool goes looking, against the sites you already cook from.
- Type a row in by hand and put the ingredients in `recipes/<slug>.md`. That is not a
  worse route than the other two; it is the same one with fewer steps.

## What the columns mean

**`Goes with`** is optional and is a hint, not a rule — a protein, a cuisine, or blank.
Blank means it goes with anything, which is true of most of them. The session uses it to
avoid putting the same starch next to the same protein twice in a week, and nothing else
reads it.

**`Season`** is also optional. It is a proxy for price and quality, not a preference —
`summer`, `winter`, or blank for year-round.

**`Active`** is the same two-number rule the corpus uses: hands-on minutes only. A tray of
carrots that roasts for forty minutes is `low`.

| Side | Goes with | Season | Active | Passive | Last served | Notes |
|---|---|---|---|---|---|---|
