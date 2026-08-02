# Pantry Router

**Decide the week, confirm the list, source the items.**

The household cooks a fraction of what it enjoys. Roughly 60 recipes have been tried and
liked; under the stress of picking a week, about 15 surface. The gap between 15 and 60 is
the product. Full reasoning in [`docs/pantry-router-proposal.md`](docs/pantry-router-proposal.md).

This is Step 1 of three — the week. The shopping list and the Kroger cart come later.

## Setup

None. Python 3, standard library only.

## Use

```sh
./plan.py                                          # a normal week
./plan.py --guests "2 adults Thu" --busy Wed       # a real one
./plan.py --risk low --nights 4
```

With `ANTHROPIC_API_KEY` set it prints the week. Without one it prints the assembled
prompt to stdout — paste that into a chat and you get the same result with one more step.
`--print-prompt` forces that behavior either way.

```sh
./plan.py --print-prompt | pbcopy
```

Options: `--guests`, `--busy`, `--risk {low,normal,high}`, `--sale`, `--nights`,
`--notes`, `--profile`, `--corpus`, `--model`. `--model` defaults to `$PANTRY_MODEL`.

## Adding a recipe

```sh
./onboard.py --url https://natashaskitchen.com/meatloaf-recipe/
./onboard.py --text notes.txt              # a typed ingredient list, quantities optional
./onboard.py --image screenshot.png        # needs ANTHROPIC_API_KEY for the vision step
./onboard.py --transcript transcript.md    # a screenshot you transcribed by hand
```

Each run writes `recipes/<slug>.md` — the ingredients, verbatim — and fills in the
recipe's row in `corpus.md`. It never overwrites a value a person put in the corpus; where
its own reading disagrees, it says so and leaves yours alone.

**It does not guess.** A quantity the source omits is recorded as omitted, a yield nobody
states stays `unknown`, and anything it cannot settle comes out as a question with the
recipe's name on it. `./onboard.py --batch <dir> --report <file>` does a directory of
inputs and writes the tally, including how much of each capture is actually missing.

Run `python3 test_onboard.py` after touching the ingredient grammar.

## The two files that matter

Everything the planner knows lives in two markdown files you edit by hand.

**[`profile.md`](profile.md)** — the household. Hard constraints, taste, patterns. One
rule: *no claim without a trace.* Every preference carries its evidence inline, because a
confident opinion about your taste drawn from nothing poisons every week downstream. If
you can't say why you believe a line, cut it.

**[`corpus.md`](corpus.md)** — recipes you've cooked and liked. Membership is earned:
nothing goes in until it's been made and kept. That bar is what makes the corpus safe to
draw from — surfacing one is a recall problem, never a quality gamble. Bookmarked-but-never-cooked
doesn't belong; the planner proposes those itself, marked `[candidate]`.

`corpus.md` is seeded with 25 recipes. `profile.md` still ships with blanks — fill it in
and the tool works.

## Seeding the corpus

Done, from a saved recipe document: 25 recipes, as 8 links, 6 typed-out ingredient lists,
and 11 screenshots. That's the §1b thread-mining accelerator arriving on day one rather
than at phase 1b.

All 25 now have an ingredient file in `recipes/`, onboarded by `onboard.py` from the same
document — six links fetched, six typed notes parsed, eleven screenshots read. Five of the
eleven screenshots are short of content and say so; the run is reported honestly in
[`docs/onboarding-run.md`](docs/onboarding-run.md), and what it means is in
[`docs/onboarding-findings.md`](docs/onboarding-findings.md). Sixteen of the 25 still have
no yield, because no source ever stated one.

It also means the unaided-recall baseline was never measurable here — the recipes came off
a saved document, not out of memory, which is a different and better thing. The saved doc
is exactly the artifact §4 predicted: *the 45 live in their heads and in their messages.*

## What it does

Reads both files, adds this week's constraints, asks for the week. Every proposal shows
why it surfaced — *you haven't made this since March* — because the reason is what makes a
forgotten recipe land. Net-new suggestions are marked `[candidate]` and never disguised as
proven.

An empty corpus is a supported state, not an error: the planner proposes fewer nights,
leans on food that's hard to get wrong, and says so. Five unproven dinners is five chances
to lose you in week one.

## Where it goes

Log outcomes, so cooked-and-kept candidates enter the corpus on their own. Then the
shopping list — deterministic, no model past parsing. Then the Kroger cart. Then the
inferred pantry, which depends on parsing order confirmation emails and may not be
obtainable at all.
