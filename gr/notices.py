"""What this tool does not know, stated in the tool.

`profile.md` names five gaps. A tool that quietly plans around them teaches the household
to trust numbers that are not there, which is the failure this project exists to stop.
Every one of them is computed from live data here so it cannot drift out of date, and
every one is meant to be shown where the household will read it — not behind a click.

Nothing in this module invents anything. Where the answer is missing it says so.
"""

from __future__ import annotations

from dataclasses import dataclass

from .repo import Repo
from .shoplist import MealPlan, ShoppingList


@dataclass
class Notice:
    key: str
    text: str
    detail: str = ""


def week_notices(repo: Repo, meals: list[MealPlan], shopping: ShoppingList) -> list[Notice]:
    """The shortfalls that apply to this particular week."""
    out: list[Notice] = []

    # 1. Sides. The single biggest reason a list is short.
    if not repo.sides:
        out.append(Notice(
            "sides",
            "sides: none recorded — this list is short by design, not by accident.",
            "sides.md is empty on purpose. Vegetables and starches get cooked here and "
            "never got written down, so every list this tool produces is systematically "
            "short until somebody types four of them in. The tool will not guess one: a "
            "seeded list of plausible vegetables would make it look finished and make "
            "every list wrong in a new way.",
        ))
    else:
        out.append(Notice(
            "sides",
            f"sides: {len(repo.sides)} recorded.",
            "The list still only covers what is written down.",
        ))

    # 2. Unknown yields. Capped in code, and marked per meal.
    unscaled = [m for m in meals if m.scale and not m.scale.scaled]
    if unscaled:
        out.append(Notice(
            "unscaled",
            f"{len(unscaled)} of {len(meals)} meals were not scaled to your household.",
            "Their amounts are exactly what the recipe says, because nothing in the data "
            "says how much the recipe makes. Each one is marked on the list. Unknown-yield "
            "meals are capped at two a week in code — the planner drifts toward them.",
        ))

    # 3. No last-cooked dates. Nothing is ranked by recency, and no claim survives.
    out.append(Notice(
        "recency",
        "no last-cooked dates exist, so nothing here is ranked by recency.",
        "Every recipe is unranked rather than overdue. Any claim the planner made about "
        "when something was last cooked was removed by code before you read it. Recall "
        "still works without dates: the whole corpus goes to the planner every run.",
    ))

    # 4. Effort ratings. Said once, where it will be seen.
    out.append(Notice(
        "effort",
        "effort ratings are the system's guess — correct any that are wrong in corpus.md.",
        "None of them came from the household.",
    ))

    # 5. The corpus is a floor.
    out.append(Notice(
        "corpus",
        "cooked anything this week that isn't in corpus.md? Add it — that's the product.",
        f"The corpus holds {len(repo.corpus)} recipes and the repertoire is nearer 30–35. "
        f"Five to ten regulars were never written down, and they are the highest-value "
        f"input this tool can get.",
    ))

    if shopping.unknown:
        out.append(Notice(
            "unknown-lines",
            f"{len(shopping.unknown)} ingredient lines could not be read and are printed "
            f"in full at the end of the list.",
            "A line the parser refuses is never dropped and never guessed at. Reading one "
            "and adding a row to items.md is the fix, and it is meant to be routine.",
        ))

    if shopping.missing_recipes:
        out.append(Notice(
            "missing-files",
            "some meals have no ingredient file: "
            + ", ".join(shopping.missing_recipes),
            "Their ingredients are missing from this list entirely.",
        ))

    return out


def open_questions(repo: Repo) -> list[str]:
    """The questions one sitting would close forever.

    Each answer is a number the household already knows and the data does not.
    """
    questions = []
    for row in repo.corpus:
        if row.yield_.shape == "unknown":
            questions.append(
                f"How many adults does **{row.title}** feed? "
                f"(corpus.md says `unknown` — the source never stated it.)")
    for noun in repo.household.open_conversions:
        questions.append(
            f"How many **{noun}s** is one adult? "
            f"(Answer it in the Portion conversions table in profile.md and every "
            f"recipe measured in {noun}s scales from then on.)")
    return questions
