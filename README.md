# Pantry Router

**Decide the week, confirm the list, source the items.**

The household cooks a fraction of what it enjoys. Roughly 60 recipes have been tried and
liked; under the stress of picking a week, about 15 surface. The gap between 15 and 60 is
the product. Full reasoning in [`docs/pantry-router-proposal.md`](docs/pantry-router-proposal.md).

Step 1 (the week) and Step 2 (the list) exist. The Kroger cart comes later.

## Setup

None. Python 3, standard library only. No dependencies, no build step, no database.

## The session

```sh
./prep.py     # optional: cache the briefing first
./app.py      # http://127.0.0.1:8765
```

One page, once a week, in order: what happened last week, what to cook this week, the
dials, the list. The server owns nothing — state lives in `weeks/<date>.md` and the same
markdown files you edit by hand, so it can be killed at any point without losing anything.

Feedback is the part that matters. Answering *kept it / nope / didn't cook* on last week is
what stamps `Last cooked`, promotes a candidate that earned its place, and gives the ranker
something to rank on. Without it every recipe looks equally forgotten forever.

Architecture and the decisions behind it: [`docs/architecture.md`](docs/architecture.md).

**It runs locally, and nothing is deployed.** The container and the one-command deploy both
work, but Hugging Face now requires PRO to host a Docker Space and the free alternatives
each want something. [`docs/deploy.md`](docs/deploy.md) has what was tried, what it
returned, and the one option that is genuinely free.

`app.py` binds localhost and **refuses to serve the real corpus on a public interface** —
pass `--demo` and it works off a scratch copy instead, where planning, feedback, promotion
and the list all run for real and none of it reaches the repo.

## Planning from the terminal

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

## The list

```sh
./shop.py --week crock-pot-italian-beef,sausage-and-peppers        # a week
./shop.py --week chicken-noodle-soup:whole-young-chicken --guests 2
./shop.py --audit                                                  # what the tables don't know
```

Loads only the recipes the week actually uses, applies the chosen variant, scales each
recipe against its own yield, converts units per item, adds it up, and prints the list by
aisle with provenance on every line — plus what's shared between meals and what's stranded
if a night falls through.

**Deterministic end to end. There is no model in this path and there must not be one** —
parsing a recipe is code, not a prompt. Nothing is silently dropped: a line the parser
can't read comes out as raw text with a flag, an item with no `items.md` row is printed
verbatim so you can buy it by eye, and a recipe whose yield nobody knows says it wasn't
scaled instead of pretending.

`--audit` parses all 27 recipe files and reports every gap. It currently reports none:
265 ingredient lines, all parsed, all recognised. Run `python3 test_shop.py` after
touching any of it — the week of 2 August is in there as an acceptance fixture.

## The rules, and what enforces them

Files don't refuse a bad write, so `pantry.py` does. Everything that mutates household data
goes through it, and these are tested in `test_pantry.py` rather than stated in prose:

- **Membership is earned.** `promote()` is the only function that may add a row to
  `corpus.md`, and only for a candidate whose cook was *kept*. Onboarding used to append
  never-cooked recipes straight into the corpus; it now refuses and says why.
- **No claim without a trace.** A profile claim with no evidence fails the write.
- **No writer overwrites a human value.** `Last cooked` is the one field the tool owns.
- **A flop is never deleted.** It stays in `candidates.md` with the reason — at this corpus
  size it's the most informative signal the system gets all week.

Every proposal, drop, dial change and outcome is appended to `decisions.jsonl`. That's what
lets a change to the ranker be replayed against real history instead of argued about, and
it's the one thing that can't be backfilled.

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
document — six links fetched, six typed notes parsed, eleven screenshots read. The run is
reported honestly in [`docs/onboarding-run.md`](docs/onboarding-run.md), and what it means
is in [`docs/onboarding-findings.md`](docs/onboarding-findings.md).

A second pass went back to the sources for what the first pass could only flag:
[`docs/onboarding-pass-2-findings.md`](docs/onboarding-pass-2-findings.md). Five of the
fifteen unknown yields were recovered from the pages the screenshots showed the address bar
of; three turned out not to be questions at all. Seven remain, and only the household can
close them.

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

Log outcomes, so cooked-and-kept candidates enter the corpus on their own. Then the Kroger
cart — SKU matching and pack sizing, behind a store adapter so a second store is a new
file. Then the inferred pantry, which depends on parsing order confirmation emails and may
not be obtainable at all.

The one thing that would improve every part of this at once is cooking a week and saying
what happened.
