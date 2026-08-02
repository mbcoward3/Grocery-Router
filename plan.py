#!/usr/bin/env python3
"""Propose a week of dinners from the household profile and the recipe corpus.

Reads profile.md and corpus.md, assembles the planner prompt with this week's
constraints, and either prints the week or prints the prompt for pasting into a
chat. No dependencies beyond the standard library.

    ./plan.py --guests "2 adults Thu" --busy "Wed" --risk low
    ./plan.py --print-prompt | pbcopy

Set ANTHROPIC_API_KEY to have it call the model directly. Without a key it
falls back to printing the prompt, which is the same tool with one more step.
"""

import argparse
import json
import os
import re
import sys
import urllib.error
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parent
API_URL = "https://api.anthropic.com/v1/messages"
API_VERSION = "2023-06-01"
DEFAULT_MODEL = os.environ.get("PANTRY_MODEL", "claude-sonnet-5")

PLANNER_PROMPT = """You are the planner for a household meal tool. You propose a week of
dinners, and every proposal shows why it surfaced.

You are given a household profile in natural language, a corpus of recipes the household
has cooked and liked, and this week's constraints. Your output is the week, each night
with a reason.

**What you are actually for.** This household cooks a fraction of what it enjoys. Some
weeks the binding problem is *retrieval* - surfacing proven recipes that fell out of
rotation - and some weeks it is *acquisition*, adding new recipes that earn their way in.
Corpus size decides the mix: read it off the corpus you were given rather than asking.
At a hundred recipes most of your value is retrieval. At zero, every proposal is unproven
and your job is to not lose the household in week one. There is no mode switch; it is one
continuum. At thirty-ish you are in the middle: retrieval still works, but a corpus that
small cannot fill a year of weeks on its own, so acquisition is part of the job every
week and not an occasional flourish.

**The opponent is "buy it again."** You are competing on breadth, not efficiency.
Efficiency is a constraint - do not make the week slower - never the goal. A week of the
five things they cooked last month is a failure even if every night is easy.

### Corpus membership

- **Corpus** recipes are proven: cooked and liked. Proposing one is risk-free.
- A **candidate** is a net-new recipe you are proposing that has never been cooked here.
  Mark every one explicitly as [candidate]. Do not disguise a candidate as a proven
  recipe, and do not propose a candidate without saying what in the profile or corpus
  made you reach for it.
- Hold the week's candidate count to what corpus size and this week's risk appetite
  justify. This is a judgment call about context and it is yours to make - but state the
  count you chose and why, in one line, before the plan.

### Planning constraints

- Vary protein and cuisine across the week. If the profile names a cap on a particular
  protein, respect it as a soft cap: you may exceed it, but say why in that line.
- **Effort is two numbers and only one of them is capped.** *Active* time is hands-on,
  at the stove. *Passive* time is unattended - slow cookers, braises, long oven sits.
  The weeknight ceiling in the profile applies to **active time only**. A four-hour pot
  roast with twenty minutes of searing is a weeknight meal; a forty-minute stir fry that
  needs all forty at the stove is not. Do not add the two together, and do not treat a
  long total time as disqualifying. Getting this wrong throws away the household's
  easiest meals.
- At least one genuinely low-active night, more if the week is busy. If the profile says
  hard nights are unpredictable, weight this higher: any night might be the bad one, so
  the week needs enough low-active options that no single night can break it.
- Share perishables across meals, **and show the coupling.** If Tuesday and Friday both
  use the same bunch of cilantro, say so. Coupling is the cost of good proposals: a
  coupled set cascades when one night breaks, so visible links let them repair a broken
  Wednesday instead of abandoning the week.
- The nights you are given are not that many cooking events. Some nights are leftovers by
  design; scale the source meal up on purpose and say which night it feeds.
- Servings in adult-equivalents. Base is in the profile; override per meal for guests.
- Seasonality matters and needs no data from you beyond knowing what is good right now.
  It is a proxy for price and quality, not a preference.
- Everyone eats the same meal, including any young children the profile mentions.
  Family-edible is a light filter, not a design driver - do not propose only beige food.

### Reasons

Every proposal carries a reason, and the reason is the product. A forgotten recipe lands
because you said *you haven't made this since March*; without that line it is just a
suggestion. Reasons must be specific and traceable to the profile, the corpus, or the
week's constraints. Never invent a fact about this household to justify a proposal - if
your reason is a guess, say it is a guess.

### If the corpus is empty or very small

Propose **fewer nights than asked for.** Five unproven dinners is five chances to lose the
household before the loop ever runs. Lean on low-variance, high-prior recipes - food that
is hard to get wrong and broadly liked - and let the corpus earn its way up to a full
week. Say plainly that you are proposing fewer, and why. Do not pad the week to look
complete, and do not manufacture confidence you have no basis for.

### Output format

One line before the plan: the candidate count you chose and why.

**Choose the shape from the profile.** If it says which nights are hard, bind meals to
days. If it says hard nights are unpredictable, **do not bind meals to days** - output an
unordered pool of cooks and state the effort mix, so they can pick night-of. A wrong day
label is worse than no day label: it trains them to ignore the column.

Pool form:

    <meal>                <N> AE   <low|med|high> active   [<reason>]

Day-bound form, only when the profile justifies it:

    Mon   <meal>          <N> AE   <low|med|high> active   [<reason>]

Either way, follow the list with one line naming the effort mix - how many low-active
cooks, and whether any single bad night can break the week.

Then a short **Coupling** section: which meals share perishables, and what breaks if one
night is skipped.

Then, if anything in the profile or corpus was too thin to plan against, say so in one or
two lines. Do not fill a gap with invention. In particular: if the corpus has no
last-cooked dates, you cannot claim a recipe hasn't been made since some month. Say the
dates are missing and surface on other grounds instead of inventing recency.

Do not produce a shopping list. That is a separate deterministic step and not your job.
If the corpus is mains-only, say once that sides are not included, so the week is not
mistaken for complete."""

EMPTY_CORPUS_NOTE = """The corpus is empty. No recipes have been cooked and logged yet.
Every proposal you make is therefore a candidate. Follow the small-corpus instruction
above: propose fewer nights, lean low-variance, and say why."""


def count_recipes(corpus_text):
    """Count real recipe rows in the corpus table.

    A row counts if it is a table row with a non-empty first cell that is not a
    separator, a header, or one of the worked examples.
    """
    count = 0
    for line in corpus_text.splitlines():
        line = line.strip()
        if not line.startswith("|"):
            continue
        cells = [c.strip() for c in line.strip("|").split("|")]
        if not cells or not cells[0]:
            continue
        first = cells[0]
        if set(first) <= set("-: "):
            continue
        if first.lower() in ("recipe", "meal"):
            continue
        if first.lower().startswith("e.g."):
            continue
        count += 1
    return count


def build_week_block(args):
    lines = [
        f"Nights to plan:       {args.nights}",
        f"Guests:               {args.guests or 'none'}",
        f"Known-busy nights:    {args.busy or 'none'}",
        f"Risk appetite:        {args.risk}",
    ]
    if args.sale:
        lines.append(f"On sale this week:    {args.sale}")
    if args.notes:
        lines.append(f"Anything else:        {args.notes}")
    return "\n".join(lines)


def build_prompt(profile_text, corpus_text, args):
    n = count_recipes(corpus_text)
    corpus_section = corpus_text if n else EMPTY_CORPUS_NOTE
    return (
        f"{PLANNER_PROMPT}\n\n"
        f"---\n\n## This week\n\n```\n{build_week_block(args)}\n```\n\n"
        f"---\n\n## Household profile\n\n{profile_text.strip()}\n\n"
        f"---\n\n## Corpus\n\n{corpus_section.strip()}\n"
    )


def call_api(prompt, model, api_key):
    payload = json.dumps(
        {
            "model": model,
            "max_tokens": 4000,
            "messages": [{"role": "user", "content": prompt}],
        }
    ).encode()
    req = urllib.request.Request(
        API_URL,
        data=payload,
        headers={
            "content-type": "application/json",
            "anthropic-version": API_VERSION,
            "x-api-key": api_key,
        },
    )
    try:
        with urllib.request.urlopen(req) as resp:
            body = json.load(resp)
    except urllib.error.HTTPError as e:
        detail = e.read().decode(errors="replace")
        sys.exit(f"API error {e.code}: {detail}")
    except urllib.error.URLError as e:
        sys.exit(f"Could not reach the API: {e.reason}")
    return "".join(b.get("text", "") for b in body.get("content", []))


def read_input(path, label):
    if not path.exists():
        sys.exit(f"Missing {label}: {path}. See README.md.")
    text = path.read_text()
    if not text.strip():
        sys.exit(f"{label} is empty: {path}")
    return text


def main():
    p = argparse.ArgumentParser(
        description="Propose a week of dinners.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    p.add_argument("--guests", help='e.g. "2 adults Thu"')
    p.add_argument("--busy", help='nights you will not want to cook, e.g. "Wed, Thu"')
    p.add_argument(
        "--risk",
        choices=["low", "normal", "high"],
        default="normal",
        help="tolerance for an unproven recipe this week (default: normal)",
    )
    p.add_argument("--sale", help="anything you know is on sale")
    p.add_argument("--nights", type=int, default=5, help="dinners to plan (default: 5)")
    p.add_argument("--notes", help="anything else the planner should know")
    p.add_argument("--profile", type=Path, default=ROOT / "profile.md")
    p.add_argument("--corpus", type=Path, default=ROOT / "corpus.md")
    p.add_argument("--model", default=DEFAULT_MODEL, help=f"default: {DEFAULT_MODEL}")
    p.add_argument(
        "--print-prompt",
        action="store_true",
        help="print the assembled prompt instead of calling the model",
    )
    args = p.parse_args()

    profile_text = read_input(args.profile, "profile")
    corpus_text = read_input(args.corpus, "corpus")

    n = count_recipes(corpus_text)
    if re.search(r"\[\.\.\.\]", profile_text):
        print(
            "note: profile.md still has unfilled [...] blanks - the week will be "
            "correspondingly vague\n",
            file=sys.stderr,
        )
    print(f"note: {n} recipe(s) in corpus\n", file=sys.stderr)

    prompt = build_prompt(profile_text, corpus_text, args)

    api_key = os.environ.get("ANTHROPIC_API_KEY")
    if args.print_prompt or not api_key:
        if not api_key and not args.print_prompt:
            print(
                "note: no ANTHROPIC_API_KEY set, printing the prompt instead\n",
                file=sys.stderr,
            )
        print(prompt)
        return

    print(call_api(prompt, args.model, api_key))


if __name__ == "__main__":
    main()
