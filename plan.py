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
import os
import re
import sys
from pathlib import Path

from planner import model as model_planner
from planner.prompt import EMPTY_CORPUS_NOTE, PLANNER_PROMPT

ROOT = Path(__file__).resolve().parent
DEFAULT_MODEL = model_planner.DEFAULT_MODEL


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
    """One HTTP implementation, in `planner/model.py`. This is the CLI's posture
    on top of it: a person at a terminal wants the error and their shell back,
    where the app wants to fall through to the ranker. The request is the same
    either way, and keeping it one function is the standing lesson from two
    ingredient parsers that were written from one spec and still disagreed."""
    try:
        return model_planner.call(prompt, model=model, api_key=api_key)
    except model_planner.PlannerUnavailable as exc:
        sys.exit(str(exc))


def print_week(args):
    """The validated week, exactly as the session would show it.

    Worth its own flag rather than being folded into the default. What this file
    has always printed is the model's prose, unread by any code - useful to a
    person, and *not* what the app puts on the board. This runs the real
    pipeline: `pantry.propose()`, slug validation, the hard-constraint checks,
    and a ranker top-up for anything dropped. Which makes it the one command that
    answers "is the model planner any good", for someone who did not build it and
    should not have to launch a web server to find out.

    Prints what was refused as well as what survived. A week that quietly lost
    three picks to validation looks identical to a week the model planned
    outright, and those are very different facts.
    """
    import household
    import pantry

    hh = household.here()

    meals = pantry.propose(hh, nights=args.nights, risk=args.risk, planner=args.planner,
                           client=None)
    last = pantry.last_proposal(hh) or {}

    used = last.get("planner", "ranker")
    if used == "model":
        print(f"planned by {last.get('model') or 'a model'}\n")
    elif last.get("fallback"):
        print(f"planned by the ranker — the model was asked and {last['fallback']}\n")
    else:
        print("planned by the ranker, deterministically\n")

    if last.get("note"):
        print(f"{last['note']}\n")
    for meal in meals:
        tag = " [candidate]" if meal.candidate else ""
        print(f"  {meal.title + tag:38} {meal.yield_ or 'unknown':14} "
              f"{meal.active or 'med':5} active   {meal.reason}")
    print(f"\n{pantry.effort_mix(meals)}")
    if last.get("planned_short"):
        print(f"{last['planned_short']} of the {args.nights} nights are leftovers by "
              f"design, not cooks")

    for label, key in (("coupling", "coupling"), ("gaps", "gaps")):
        if last.get(key):
            print(f"\n{label}: {last[key]}")
    for dropped in last.get("dropped", []):
        print(f"\ndropped: {dropped}", file=sys.stderr)
    for warning in last.get("warnings", []):
        print(f"\nnote: {warning}", file=sys.stderr)
    if last.get("topped_up"):
        print(f"\nnote: {len(last['topped_up'])} night(s) filled by the ranker",
              file=sys.stderr)


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
    p.add_argument(
        "--week",
        action="store_true",
        help="print the validated week the session would show, with what was dropped",
    )
    p.add_argument(
        "--planner",
        choices=["ranker", "model", "auto"],
        default=None,
        help="with --week: which planner to use (default: auto, on whether a key is set)",
    )
    args = p.parse_args()

    if args.week:
        return print_week(args)

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
