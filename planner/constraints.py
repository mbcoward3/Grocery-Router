"""The hard constraints, enforced rather than asserted.

`profile.md` opens its constraint list with *"Not preferences. The planner must
never violate these."* Until now nothing checked. The deterministic ranker never
needed a checker because it can only ever pick a corpus row and can only ever
write a reason it computed itself; a model planner can do neither of those things
by construction, so the check has to become code.

This module is the same move `pantry.py` made for the write rules: the rule was
stated in prose in five places and refused in none of them. Here the rule is a
function, and `planner/model.py` drops any pick that fails it.

**What is checked, and what a violation costs.** A violating meal is dropped and
the week is topped up from the ranker. It is not repaired, and its reason is not
rewritten - substituting the ranker's sentence under the model's pick would
attribute a reason to a decision that was never made for it, which is exactly the
mistake that let a candidate inherit a corpus recipe's reason and claim
membership it did not have.

**What is not checked, said out loud.** *Family-edible* is in the profile as a
hard constraint and there is no honest mechanical test for it. Corpus membership
is the proxy - everything in `corpus.md` has been cooked and eaten by this
household, children included - and that proxy is why the model may only choose
from the catalogue. A candidate is the weaker case and it is marked as one. That
is the whole of the guarantee, and inventing a keyword list that appeared to
check more would be worse than admitting the boundary.

Standard library only.
"""

from __future__ import annotations

import re

import pantry

# The recipe capture already answers this. `onboard.py` scans every ingredient
# line with a term list and writes its verdict into the recipe file's `peanut:`
# header - `none seen`, `check label`, or `CONTAINS PEANUT`. Reading that header
# back is deliberate: a second peanut scanner living here would be a second
# implementation of one thing, and the two-parsers bug is the trap this repo has
# fallen into most expensively. If the capture is wrong, the fix belongs in the
# capture, where it also fixes the shopping list.
PEANUT_HEADER = re.compile(r"^peanut:\s*(.+)$", re.M)

# `check label` is not a violation, and that is the profile's call, not a
# shortcut: *"Trace-risk and shared-facility products are acceptable - this
# filters the recipe, not the pantry."* The teriyaki and the stir fry both sit
# here because a bought sauce could carry it, and both are in the corpus because
# this household has cooked and eaten them.
PEANUT_BLOCKS = "contains peanut"

# A claim about when something was last cooked, in the shapes a planner writes
# them. Only ever applied to a row with no last-cooked date, where any such claim
# is necessarily invented - see `check_meal`.
RECENCY_CLAIM = re.compile(
    r"\b("
    r"since\s+(january|february|march|april|may|june|july|august|september|october|"
    r"november|december|last\s+\w+)"
    r"|haven'?t\s+(made|cooked|had)"
    r"|hasn'?t\s+(been\s+)?(made|cooked)"
    r"|not\s+cooked\s+in"
    r"|last\s+cooked"
    r"|in\s+(months|weeks|a\s+while)"
    r"|\d+\s*(days?|weeks?|months?|years?)\s+ago"
    r"|fell\s+out\s+of\s+rotation"
    r"|dormant\s+(for|since)"
    r")\b",
    re.I,
)

# From `profile.md`: the 20-30 minute active ceiling is Mon-Fri, and *"one or two
# weekend nights (usually Sat/Sun) are open to something longer and nicer."* Two
# is therefore the number of high-active cooks a week can absorb, and it is a
# count off that sentence rather than a taste judgment. Below two nights the cap
# is the week itself.
HIGH_ACTIVE_PER_WEEK = 2


def peanut_verdict(sl: str) -> str:
    """The capture's recorded verdict for one recipe, lowercased.

    Returns `""` when the recipe has no file or the file predates the header.
    Four captures are in that state today, and **unknown is not treated as
    unsafe.** That is the same discipline the ranker applies to a missing
    last-cooked date: a recipe with no date was scoring as *maximally* stale
    until it was fixed, because absence had been quietly read as an extreme.
    Absence is absence. It is reported by `unchecked()` so a person can see it,
    and it does not silently block a meal this household already eats.
    """
    path = pantry.recipe_file(sl)
    if not path.exists():
        return ""
    m = PEANUT_HEADER.search(path.read_text(encoding="utf-8"))
    return m.group(1).strip().lower() if m else ""


def check_meal(meal, row: dict | None) -> str | None:
    """Why this pick may not go in the week, or `None` if nothing rules it out.

    `row` is the catalogue row the slug resolved to - `None` means it resolved to
    nothing at all, which is the failure the whole validation exists for.
    """
    if row is None:
        return f"{meal.slug!r} is not in the corpus or the candidates"

    if PEANUT_BLOCKS in peanut_verdict(meal.slug):
        return f"{meal.title} contains peanut, and the allergy is a hard constraint"

    # The one invention that is cheap to catch. Most of this corpus has no
    # last-cooked date, the prompt says so twice, and "you haven't made this
    # since March" about a row with no date is a fact about this household that
    # nobody has. A row that *does* carry a date can say whatever the date
    # supports, and this check stays out of its way.
    if not (row.get("last cooked") or "").strip() and RECENCY_CLAIM.search(meal.reason or ""):
        return (f"{meal.title} has no last-cooked date, so its reason claims a "
                f"recency nothing supports")

    return None


def check_week(meals: list) -> list[str]:
    """Constraints that are properties of the week rather than of one meal.

    Returned as text for the session to show, not raised. There is exactly one
    today, and it is deliberately not enforced by dropping a meal: which of four
    high-active cooks is the wrong one is a judgment the household makes, and
    silently deleting the fourth would hide the fact that the week is unrunnable.
    """
    out = []
    high = [m.title for m in meals if (m.active or "").strip().lower() == "high"]
    cap = min(HIGH_ACTIVE_PER_WEEK, len(meals))
    if len(high) > cap:
        out.append(f"{len(high)} high-active cooks ({', '.join(high)}) against a "
                   f"weeknight ceiling that leaves room for {cap}")
    return out


def unchecked(meals: list) -> list[str]:
    """Meals whose capture never recorded a peanut verdict.

    Not a violation and not a silence. The household is owed the difference
    between *scanned, nothing found* and *never scanned*, and four recipes are in
    the second state.
    """
    return [m.title for m in meals if not peanut_verdict(m.slug)]
