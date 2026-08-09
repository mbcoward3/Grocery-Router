"""One planning session: pick the meals, build the list, write the week file.

This is the seam the interface sits on. It holds no state of its own — it reads the
markdown files, does the work, writes the week file back, and returns what it built.
Anything that needs to survive a restart is in the files.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import date
from pathlib import Path

from . import planner as PL
from . import repo as R
from . import shoplist as S
from . import weekfile as W
from .notices import Notice, open_questions, week_notices


@dataclass
class Week:
    sunday: date
    nights: int
    guests: int
    meals: list[S.MealPlan]
    shopping: S.ShoppingList
    notices: list[Notice]
    questions: list[str]
    planner_source: str = "planner"
    planner_error: str = ""
    planner_notes: list[str] = field(default_factory=list)
    dropped: list[tuple[str, str]] = field(default_factory=list)
    cost_usd: float | None = None
    path: Path | None = None
    target_ae: float = 0.0


def last_week_text(repo: R.Repo, before: date) -> str:
    """The most recent week file, for the planner to read as context."""
    directory = repo.root / "weeks"
    if not directory.exists():
        return ""
    files = sorted(p for p in directory.glob("*.md") if p.stem < before.isoformat())
    return files[-1].read_text(encoding="utf-8") if files else ""


def assemble(repo: R.Repo, meals: list[S.MealPlan], sunday: date, nights: int,
             guests: int, planner_source: str, planner_error: str = "",
             planner_notes: list[str] | None = None,
             dropped: list[tuple[str, str]] | None = None,
             cost_usd: float | None = None, write: bool = True) -> Week:
    """Build the list for a set of meals and write the week file.

    `S.build` sets each meal's scale as it goes, so the meals passed in come back with
    their multipliers filled in.
    """
    shopping = S.build(repo, meals, guests=guests)
    week = Week(
        sunday=sunday, nights=nights, guests=guests, meals=meals, shopping=shopping,
        notices=week_notices(repo, meals, shopping), questions=open_questions(repo),
        planner_source=planner_source, planner_error=planner_error,
        planner_notes=planner_notes or [], dropped=dropped or [], cost_usd=cost_usd,
        target_ae=repo.target_ae(guests),
    )

    path = W.week_path(repo.root, sunday)
    if write:
        ticks = W.read_ticks(path)
        text = W.render(repo, sunday, meals, shopping, nights, guests,
                        planner_source, planner_error, week.planner_notes, ticks)
        W.write(path, text)
        W.log_decision(repo.root, sunday, meals, nights, guests, planner_source,
                       week.dropped)
    week.path = path
    return week


def plan_week(root: Path | str = ".", nights: int = 5, guests: int = 0,
              sunday: date | None = None, avoid: list[str] | None = None,
              model: str = PL.MODEL, write: bool = True) -> Week:
    """The whole session: one planner call, every check, the list, the file."""
    repo = R.load(root)
    sunday = sunday or W.sunday_of()
    result = PL.plan(repo, nights=nights, guests=guests,
                     last_week=last_week_text(repo, sunday), avoid=avoid, model=model)
    return assemble(repo, result.meals, sunday, nights, guests,
                    planner_source=(result.source if result.source == "code" else model),
                    planner_error=result.error, planner_notes=result.notes,
                    dropped=result.dropped, cost_usd=result.cost_usd, write=write)


def swap(repo: R.Repo, week: Week, slug: str, replacement: str | None = None) -> Week:
    """Replace one meal without re-rolling the week.

    A swap is code, not a second model call. The household asked for a different dish,
    not a different week, and re-running the planner would move the four they kept.
    """
    meals = list(week.meals)
    index = next((i for i, m in enumerate(meals) if m.slug == slug), None)
    if index is None:
        return week

    chosen = {m.slug for m in meals}
    if replacement is None:
        kept = [repo.row(m.slug) for m in meals if m.slug != slug]
        kept = [r for r in kept if r is not None]
        proteins = [r.protein for r in kept]
        unknowns = sum(1 for r in kept if r.yield_.shape == "unknown")
        candidates = [
            r for r in repo.all_rows
            if r.slug not in chosen and r.slug in repo.recipes
            and not (r.yield_.shape == "unknown" and unknowns >= 2)
        ]
        if not candidates:
            return week
        candidates.sort(key=lambda r: (proteins.count(r.protein),
                                       0 if r.yield_.shape != "unknown" else 1,
                                       r.title))
        row = candidates[0]
    else:
        row = repo.row(replacement)
        if row is None or row.slug not in repo.recipes:
            return week

    meals[index] = S.MealPlan(
        slug=row.slug, title=row.title, reason_kind="plain",
        reason=("swapped in by code at your request. Code does not write reasons the way "
                "the planner does — regenerate the week if you want one."),
        yield_raw=row.yield_raw, scale=None, untried=row.untried,
    )
    return assemble(repo, meals, week.sunday, week.nights, week.guests,
                    planner_source=week.planner_source,
                    planner_error=week.planner_error,
                    planner_notes=week.planner_notes, dropped=week.dropped)


def load_existing(root: Path | str = ".", sunday: date | None = None) -> Week | None:
    """Rebuild a Week from a week file that already exists on disk.

    The meals come from the file, so a week survives a restart. The list is recomputed
    from the recipe files rather than read back, because the recipe files are the truth
    and a stale number in a rendered list must never outlive them.
    """
    repo = R.load(root)
    sunday = sunday or W.sunday_of()
    path = W.week_path(repo.root, sunday)
    if not path.exists():
        return None

    text = path.read_text(encoding="utf-8")
    nights, guests, source = 5, 0, "planner"
    meals: list[S.MealPlan] = []
    in_meals = False
    pending: S.MealPlan | None = None

    for line in text.splitlines():
        stripped = line.strip()
        if stripped.startswith("nights:"):
            nights = int(stripped.split(":", 1)[1].strip() or 5)
        elif stripped.startswith("guests:"):
            guests = int(stripped.split(":", 1)[1].strip() or 0)
        elif stripped.startswith("meals chosen by:"):
            source = stripped.split(":", 1)[1].strip()
        elif stripped.startswith("## "):
            in_meals = stripped[3:].strip().lower() == "meals"
        elif in_meals and stripped.startswith("- "):
            body = stripped[2:]
            kind = "plain"
            if "`" in body:
                head, _, tail = body.rpartition("`")
                kind = head.rsplit("`", 1)[-1] or "plain"
                body = head.rsplit("`", 1)[0]
            title = body.split("|")[0].strip()
            row = next((r for r in repo.all_rows if r.title == title
                        or f"{r.title} [candidate]" == title), None)
            if row:
                pending = S.MealPlan(slug=row.slug, title=row.title, reason_kind=kind,
                                     reason="", yield_raw=row.yield_raw, scale=None,
                                     untried=row.untried)
                meals.append(pending)
        elif in_meals and pending is not None and stripped and not stripped.startswith("⚠"):
            if not pending.reason:
                pending.reason = stripped

    if not meals:
        return None

    week = assemble(repo, meals, sunday, nights, guests, planner_source=source,
                    write=False)
    week.path = path
    return week
