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

Both ship with blanks. Fill them in and the tool works.

## Seeding the corpus

One sitting, no notes, no scrolling your messages. Whatever you reach unaided is the
starting corpus.

**Write that number down before your first run.** Once you've seen the planner propose a
week you can never measure it again, and it's the only number that later distinguishes a
tool that surfaced things you'd forgotten from one that agreed with what you'd have picked
anyway. That distinction is the entire project.

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
