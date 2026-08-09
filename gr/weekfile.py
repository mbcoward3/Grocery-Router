"""Render and parse one generated week as reviewable markdown.

Local development stores this document at ``weeks/<sunday>.md``. Production stores the
same deterministic representation behind ``gr.storage`` and keeps list ticks in durable
rows. A week is Sunday to Saturday and is named by its Sunday.
"""

from __future__ import annotations

import json
import re
from datetime import date, datetime, timedelta
from pathlib import Path

from .notices import Notice, open_questions, week_notices
from .repo import Repo
from .shoplist import MealPlan, ShoppingList

_TICK = re.compile(r"^\s*-\s*\[( |x|X)\]\s*(.*?)\s*$")
_KEY = re.compile(r"<!--\s*key:\s*(\S+)\s*-->")


def sunday_of(day: date | None = None) -> date:
    """The Sunday that names the week containing `day`."""
    day = day or date.today()
    return day - timedelta(days=(day.weekday() + 1) % 7)


def week_path(root: Path, sunday: date) -> Path:
    return Path(root) / "weeks" / f"{sunday.isoformat()}.md"


def _human(item: str) -> str:
    return item.replace("_", " ")


def read_ticks(path: Path) -> set[str]:
    """Which lines are already ticked, keyed by the anchor written into each line."""
    if not Path(path).exists():
        return set()
    ticked = set()
    for line in Path(path).read_text(encoding="utf-8").splitlines():
        m = _TICK.match(line)
        if not m or m.group(1).lower() != "x":
            continue
        key = _KEY.search(m.group(2))
        if key:
            ticked.add(key.group(1))
    return ticked


def line_key(prefix: str, value: str) -> str:
    """A stable anchor for one list row, safe to put in a markdown comment."""
    slug = re.sub(r"[^a-z0-9]+", "-", value.lower()).strip("-")
    return f"{prefix}:{slug}"


def render(repo: Repo, sunday: date, meals: list[MealPlan], shopping: ShoppingList,
           nights: int, guests: int, planner_source: str, planner_error: str = "",
           planner_notes: list[str] | None = None,
           ticks: set[str] | None = None) -> str:
    """Render the whole week file: the shortfalls, the meals, then the list."""
    ticks = ticks or set()
    notices: list[Notice] = week_notices(repo, meals, shopping)

    def box(key: str) -> str:
        return "[x]" if key in ticks else "[ ]"

    out: list[str] = []
    add = out.append

    add(f"# Week of {sunday.isoformat()}")
    add("")
    add("*Written by Grocery Router. Edit it by hand if it is wrong — correcting these "
        "files is the trust mechanism, and it beats any opaque score.*")
    add("")
    add(f"nights: {nights}")
    add(f"guests: {guests}")
    add(f"target: {repo.target_ae(guests):.1f} AE")
    add(f"meals chosen by: {planner_source}")
    if planner_error:
        add(f"planner error: {planner_error}")
    add("")

    add("## What this list does not know")
    add("")
    for notice in notices:
        add(f"- **{notice.text}**")
        if notice.detail:
            add(f"  {notice.detail}")
    for note in planner_notes or []:
        add(f"- code changed the planner's answer — {note}")
    add("")

    add("## Meals")
    add("")
    for meal in meals:
        scale = meal.scale
        bits = [meal.label, meal.yield_raw or "yield unknown"]
        if scale and scale.scaled:
            bits.append(meal.multiplier_text)
        else:
            bits.append("not scaled")
        add(f"- {' | '.join(bits)}  `{meal.reason_kind}`")
        add(f"    {meal.reason}")
        if scale and not scale.scaled:
            add(f"    ⚠ {scale.note}")
        add("")

    add("## Shopping list")
    add("")
    if not shopping.buy:
        add("*Nothing to buy. That is almost certainly wrong — check the unknown lines "
            "below.*")
        add("")
    for aisle, lines in shopping.by_aisle().items():
        add(f"### {aisle}")
        add("")
        for line in lines:
            key = line_key("buy", line.item)
            text = f"{line.quantity_text()} {_human(line.item)}".strip()
            trail = " — " + ", ".join(line.sources) if line.sources else ""
            marks = ""
            if line.stranded:
                marks += "  *[stranded — only this one meal needs it]*"
            for flag in line.flags:
                marks += f"  *[{flag}]*"
            add(f"- {box(key)} {text}{trail}{marks} <!-- key: {key} -->")
        add("")

    add("## Probably have — check before you go")
    add("")
    add("*Staples. Routed here rather than dropped, because `salt, to taste` must not "
        "reach the list as a thing to buy and must not vanish either.*")
    add("")
    for line in shopping.staples:
        key = line_key("staple", line.item)
        text = f"{line.quantity_text()} {_human(line.item)}".strip()
        add(f"- {box(key)} {text} <!-- key: {key} -->")
    if not shopping.staples:
        add("- none")
    add("")

    add("## Unknown items — printed, never dropped")
    add("")
    add("*The parser could not read these lines, or `items.md` has no row for them. They "
        "are here verbatim so nobody gets home without them. Adding a row to `items.md` "
        "is the fix.*")
    add("")
    for unknown in shopping.unknown:
        key = line_key("unknown", f"{unknown.meal_slug}-{unknown.raw}")
        add(f"- {box(key)} [{unknown.meal_slug}] {unknown.raw}  *({unknown.reason})* "
            f"<!-- key: {key} -->")
    if not shopping.unknown:
        add("- none")
    add("")

    questions = open_questions(repo)
    if questions:
        add("## Questions one sitting would close forever")
        add("")
        for question in questions:
            add(f"- {question}")
        add("")

    return "\n".join(out).rstrip() + "\n"


def write(path: Path, text: str) -> Path:
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")
    return path


def toggle_tick(path: Path, key: str) -> bool:
    """Flip one checkbox in the week file and return its new state.

    The markdown is edited in place, so a tick made on a phone in an aisle is in the same
    file the household reads afterwards.
    """
    path = Path(path)
    lines = path.read_text(encoding="utf-8").splitlines()
    new_state = False
    for i, line in enumerate(lines):
        m = _TICK.match(line)
        if not m:
            continue
        found = _KEY.search(m.group(2))
        if not found or found.group(1) != key:
            continue
        new_state = m.group(1).lower() != "x"
        lines[i] = re.sub(r"\[( |x|X)\]", "[x]" if new_state else "[ ]", line, count=1)
        break
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")
    return new_state


def decision_record(sunday: date, meals: list[MealPlan], nights: int,
                    guests: int, planner_source: str,
                    dropped: list[tuple[str, str]]) -> dict:
    """Build the append-only event saved by either persistence backend."""
    return {
        "at": datetime.now().replace(microsecond=0).isoformat(),
        "kind": "proposed",
        "week": sunday.isoformat(),
        "nights": nights,
        "guests": guests,
        "planner": planner_source,
        "added": [
            {"recipe": m.slug, "kind": m.reason_kind, "reason": m.reason,
             "candidate": m.untried,
             "scaled": bool(m.scale and m.scale.scaled),
             "multiplier": round(m.scale.multiplier, 4) if m.scale else None}
            for m in meals
        ],
        "dropped": [{"recipe": what, "why": why} for what, why in dropped],
    }


def log_decision(root: Path, sunday: date, meals: list[MealPlan], nights: int,
                 guests: int, planner_source: str, dropped: list[tuple[str, str]]) -> None:
    """Compatibility helper for callers using the local markdown backend directly."""
    record = decision_record(sunday, meals, nights, guests, planner_source, dropped)
    with open(Path(root) / "decisions.jsonl", "a", encoding="utf-8") as fh:
        fh.write(json.dumps(record, ensure_ascii=False) + "\n")
