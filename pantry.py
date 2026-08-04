#!/usr/bin/env python3
"""The core the app and the CLIs sit on.

Three jobs, and the split between them is the architecture:

**Read.** `corpus.md`, `candidates.md`, `profile.md` and `weeks/*.md` load into
plain dicts. These files stay the database - hand-editable markdown in git, so
history and audit come free and correcting the tool is still just editing a file.

**Propose.** `propose()` ranks the corpus deterministically and always works.
When a model is available it plans instead, and the ranker becomes the fallback.
Two implementations behind one call is what lets the app be alive with no setup.

**Write.** Every mutation goes through here, and the rules the project has been
stating in prose are enforced as code:

  - only `promote()` may insert a row into `corpus.md`, and only for a candidate
    the household cooked and kept
  - no writer overwrites a value a person put there
  - a week is written whole, never patched in place

Standard library only.
"""

from __future__ import annotations

import datetime as dt
import re
from dataclasses import dataclass, field, asdict
from pathlib import Path

ROOT = Path(__file__).resolve().parent
CORPUS = ROOT / "corpus.md"
CANDIDATES = ROOT / "candidates.md"
PROFILE = ROOT / "profile.md"
WEEKS = ROOT / "weeks"
CACHE = ROOT / ".cache"

# From profile.md: 2 adults, a 3-year-old, a 1-year-old.
BASE_AE = 2.5


# --------------------------------------------------------------------------- #
# Markdown tables
# --------------------------------------------------------------------------- #

def _rows(path: Path) -> tuple[list[str], list[dict], list[int]]:
    """(lines, parsed rows, line index per row). Keeps line numbers so a write
    can put a value back exactly where it came from."""
    lines = path.read_text(encoding="utf-8").splitlines() if path.exists() else []
    header, header_i = None, None
    out, where = [], []
    for i, line in enumerate(lines):
        s = line.strip()
        if not s.startswith("|"):
            continue
        cells = [c.strip() for c in s.strip("|").split("|")]
        if set("".join(cells)) <= set("-: "):
            continue
        if header is None:
            header, header_i = [c.lower() for c in cells], i
            continue
        if len(cells) < len(header):
            cells += [""] * (len(header) - len(cells))
        row = dict(zip(header, cells))
        if not row.get("recipe"):
            continue
        out.append(row)
        where.append(i)
    return lines, out, where


def slug(text: str) -> str:
    return re.sub(r"[^a-z0-9]+", "-", (text or "").lower()).strip("-")


def load_corpus() -> list[dict]:
    _, rows, _ = _rows(CORPUS)
    for r in rows:
        r["slug"] = slug(r["recipe"])
        r["proven"] = True
    return rows


def load_candidates() -> list[dict]:
    _, rows, _ = _rows(CANDIDATES)
    for r in rows:
        r["slug"] = slug(r["recipe"])
        r["proven"] = False
    return rows


def recipe_file(sl: str) -> Path:
    return ROOT / "recipes" / f"{sl}.md"


def variants_for(sl: str) -> list[str]:
    path = recipe_file(sl)
    if not path.exists():
        return []
    out, inside = [], False
    for line in path.read_text(encoding="utf-8").splitlines():
        s = line.strip()
        if s.startswith("## "):
            inside = s[3:].strip().lower() == "variants"
        elif inside and s.startswith("### "):
            out.append(re.sub(r"<!--.*?-->", "", s[4:]).strip())
    return out


# --------------------------------------------------------------------------- #
# Effort and yield, read off the index
# --------------------------------------------------------------------------- #

ACTIVE_RANK = {"low": 0, "med": 1, "high": 2}


def active_rank(row: dict) -> int:
    return ACTIVE_RANK.get((row.get("active") or "").strip().lower(), 1)


def yield_ae(row: dict) -> float | None:
    """Only the AE shape converts. `8 enchiladas` and `per portion` deliberately
    do not - see docs/step2-design.md 2.5."""
    m = re.match(r"^\s*(\d+(?:\.\d+)?)\s*AE\b", row.get("yield", ""))
    return float(m.group(1)) if m else None


def days_since(row: dict, today: dt.date) -> int | None:
    raw = (row.get("last cooked") or "").strip()
    if not raw:
        return None
    for fmt in ("%Y-%m-%d", "%Y/%m/%d", "%d %b %Y", "%b %d %Y"):
        try:
            return (today - dt.datetime.strptime(raw, fmt).date()).days
        except ValueError:
            continue
    return None


# --------------------------------------------------------------------------- #
# Propose
# --------------------------------------------------------------------------- #

@dataclass
class Meal:
    slug: str
    title: str
    protein: str = ""
    cuisine: str = ""
    yield_: str = ""
    active: str = ""
    passive: str = ""
    variant: str = ""
    variants: list[str] = field(default_factory=list)
    reason: str = ""
    candidate: bool = False
    locked: bool = False

    def to_json(self):
        d = asdict(self)
        d["yield"] = d.pop("yield_")
        return d


def _reason(row: dict, gap: int | None, picked: list[Meal]) -> str:
    """The reason is the product. It must be traceable to the corpus, the profile
    or this week - never invented, and never a claim about recency the dates do
    not support."""
    if gap is None:
        return "never recorded as cooked — the dates start empty, so this is unranked, not stale"
    if gap > 120:
        return f"not cooked in {gap // 30} months"
    if gap > 45:
        return f"last cooked {gap} days ago"
    proteins = [m.protein for m in picked]
    if row.get("protein") and row["protein"] not in proteins:
        return f"the only {row['protein']} in the week so far"
    if active_rank(row) == 0:
        return "low active — the week needs nights a bad day cannot break"
    return f"last cooked {gap} days ago"


def propose(nights: int = 5, guests: float = 0.0, risk: str = "normal",
            keep: list[Meal] | None = None, today: dt.date | None = None) -> list[Meal]:
    """Deterministic ranker. Fills the week up to `nights`, leaving `keep` alone.

    Gap-filling is the whole point: the session lets the household drop a meal and
    ask for another, and re-proposing the ones they already accepted would make
    that useless.
    """
    today = today or dt.date.today()
    keep = list(keep or [])
    taken = {m.slug for m in keep}
    ae = BASE_AE + guests

    pool = [r for r in load_corpus() if r["slug"] not in taken]
    cands = [r for r in load_candidates()
             if r["slug"] not in taken and "flopped" not in (r.get("outcome") or "")]
    want_cands = {"low": 0, "normal": 1, "high": 2}.get(risk, 1)

    picked = list(keep)
    used_candidates = sum(1 for m in keep if m.candidate)

    def score(row, candidate):
        gap = days_since(row, today)
        s = 0.0
        s += 100.0 if gap is None else min(gap, 365) / 3.0
        proteins = [m.protein for m in picked]
        if row.get("protein"):
            s -= 25.0 * proteins.count(row["protein"])
        cuisines = [m.cuisine for m in picked]
        if row.get("cuisine"):
            s -= 8.0 * cuisines.count(row["cuisine"])
        # The week needs enough low-active cooks that no single bad night breaks it.
        low_so_far = sum(1 for m in picked if ACTIVE_RANK.get(m.active, 1) == 0)
        if low_so_far < max(2, nights // 2) and active_rank(row) == 0:
            s += 18.0
        if active_rank(row) == 2:
            s -= 12.0
        # A big yield is worth more when there are more mouths.
        y = yield_ae(row)
        if y and y >= ae * 2:
            s += 6.0
        if candidate:
            s -= 20.0        # unproven carries the gamble
        return s, gap

    while len(picked) < nights:
        best, best_row, best_gap, best_cand = None, None, None, False
        for row, candidate in [(r, False) for r in pool] + [(r, True) for r in cands]:
            if candidate and used_candidates >= want_cands:
                continue
            if row["slug"] in {m.slug for m in picked}:
                continue
            s, gap = score(row, candidate)
            if best is None or s > best:
                best, best_row, best_gap, best_cand = s, row, gap, candidate
        if best_row is None:
            break
        vs = variants_for(best_row["slug"])
        picked.append(Meal(
            slug=best_row["slug"], title=best_row["recipe"],
            protein=best_row.get("protein", ""), cuisine=best_row.get("cuisine", ""),
            yield_=best_row.get("yield", ""), active=(best_row.get("active") or "").lower(),
            passive=best_row.get("passive", ""), variant=vs[0] if vs else "", variants=vs,
            reason=_reason(best_row, best_gap, picked), candidate=best_cand,
        ))
        if best_cand:
            used_candidates += 1

    return picked


def effort_mix(meals: list[Meal]) -> str:
    low = sum(1 for m in meals if ACTIVE_RANK.get(m.active, 1) == 0)
    if not meals:
        return "nothing planned yet"
    safe = "no single bad night breaks the week" if low >= 2 else \
           "only one easy night — one bad day could break the week"
    return f"{low} of {len(meals)} are low-active; {safe}"


# --------------------------------------------------------------------------- #
# The week: weeks/YYYY-MM-DD.md
# --------------------------------------------------------------------------- #

@dataclass
class Week:
    date: str
    nights: int = 5
    guests: float = 0.0
    risk: str = "normal"
    status: str = "planning"          # planning | ordered | cooked
    meals: list[Meal] = field(default_factory=list)
    feedback: dict = field(default_factory=dict)   # slug -> kept | flopped | not cooked

    @property
    def ae(self) -> float:
        return BASE_AE + self.guests

    def path(self) -> Path:
        return WEEKS / f"{self.date}.md"

    def to_json(self):
        return {"date": self.date, "nights": self.nights, "guests": self.guests,
                "risk": self.risk, "status": self.status, "ae": self.ae,
                "meals": [m.to_json() for m in self.meals],
                "feedback": self.feedback, "effort": effort_mix(self.meals)}


MEAL_LINE = re.compile(r"^- (.+?) \| (.*?) \| (.*?) active \| variant: (.*?) \| (.*)$")


def read_week(date: str) -> Week | None:
    path = WEEKS / f"{date}.md"
    if not path.exists():
        return None
    w = Week(date=date)
    section = None
    for line in path.read_text(encoding="utf-8").splitlines():
        s = line.strip()
        if s.startswith("## "):
            section = s[3:].strip().lower()
            continue
        m = re.match(r"^([a-z]+):\s*(.*)$", s)
        if m and section is None:
            key, val = m.group(1), m.group(2).strip()
            if key == "nights":
                w.nights = int(val or 5)
            elif key == "guests":
                w.guests = float(val or 0)
            elif key in ("risk", "status"):
                setattr(w, key, val)
            continue
        if section == "meals" and s.startswith("- "):
            m = MEAL_LINE.match(s)
            if m:
                title, yld, active, variant, reason = m.groups()
                cand = title.endswith("[candidate]")
                title = title.replace("[candidate]", "").strip()
                sl = slug(title)
                w.meals.append(Meal(slug=sl, title=title, yield_=yld, active=active,
                                    variant="" if variant == "—" else variant,
                                    variants=variants_for(sl), reason=reason,
                                    candidate=cand))
        if section == "feedback" and s.startswith("- "):
            name, _, outcome = s[2:].partition(":")
            if outcome:
                w.feedback[slug(name)] = outcome.strip()
    index = {r["slug"]: r for r in load_corpus() + load_candidates()}
    for meal in w.meals:
        row = index.get(meal.slug)
        if row:
            meal.protein = row.get("protein", "")
            meal.cuisine = row.get("cuisine", "")
    return w


def write_week(w: Week) -> Path:
    """Written whole, every time. A week is small and a rewrite keeps the file
    honest; patching lines in place is how these files drift."""
    WEEKS.mkdir(exist_ok=True)
    out = [f"# Week of {w.date}", "",
           "*Written by `app.py`. Edit it by hand if it is wrong — that is still the "
           "correction mechanism.*", "",
           f"nights: {w.nights}", f"guests: {w.guests:g}", f"risk:   {w.risk}",
           f"status: {w.status}", "", "## Meals", ""]
    for m in w.meals:
        tag = " [candidate]" if m.candidate else ""
        out.append(f"- {m.title}{tag} | {m.yield_ or 'unknown'} | {m.active or 'med'} "
                   f"active | variant: {m.variant or '—'} | {m.reason}")
    if w.feedback:
        out += ["", "## Feedback", ""]
        index = {m.slug: m.title for m in w.meals}
        for sl, outcome in w.feedback.items():
            out.append(f"- {index.get(sl, sl)}: {outcome}")
    out.append("")
    w.path().write_text("\n".join(out), encoding="utf-8")
    return w.path()


def list_weeks() -> list[str]:
    WEEKS.mkdir(exist_ok=True)
    return sorted(p.stem for p in WEEKS.glob("*.md"))


def previous_week(before: str) -> Week | None:
    earlier = [d for d in list_weeks() if d < before]
    return read_week(earlier[-1]) if earlier else None


def monday(today: dt.date | None = None) -> str:
    today = today or dt.date.today()
    return (today - dt.timedelta(days=today.weekday())).isoformat()


# --------------------------------------------------------------------------- #
# Write: the only paths that may change the corpus
# --------------------------------------------------------------------------- #

def _set_cell(path: Path, target_slug: str, column: str, value: str,
              overwrite: bool = False) -> bool:
    lines, rows, where = _rows(path)
    if not rows:
        return False
    header = [c.strip().lower() for c in lines[where[0] - 1].strip().strip("|").split("|")] \
        if where else []
    for row, i in zip(rows, where):
        if row["slug"] != target_slug:
            continue
        cells = [c.strip() for c in lines[i].strip().strip("|").split("|")]
        if column not in header:
            return False
        col = header.index(column)
        while len(cells) <= col:
            cells.append("")
        if cells[col] and not overwrite:
            return False          # never overwrite a value a person put there
        cells[col] = value
        lines[i] = "| " + " | ".join(cells) + " |"
        path.write_text("\n".join(lines) + "\n", encoding="utf-8")
        return True
    return False


def record_cooked(target_slug: str, when: str) -> bool:
    """Stamp `Last cooked`. This is the write that makes retrieval possible at
    all - without it every corpus row looks equally forgotten forever."""
    return _set_cell(CORPUS, target_slug, "last cooked", when, overwrite=True)


def promote(target_slug: str, when: str) -> bool:
    """Move a candidate into the corpus. **The only function that may add a row
    to `corpus.md`.** Membership is earned by being cooked and kept; a recipe
    having a file, a URL or an enthusiastic reason is not that.
    """
    cands = {r["slug"]: r for r in load_candidates()}
    row = cands.get(target_slug)
    if row is None:
        return False
    if any(r["slug"] == target_slug for r in load_corpus()):
        return False

    lines, rows, where = _rows(CORPUS)
    header_i = where[0] - 1
    header = [c.strip().lower() for c in lines[header_i].strip().strip("|").split("|")]
    cells = [row.get(col, "") for col in header]
    if "last cooked" in header:
        cells[header.index("last cooked")] = when
    if "notes" in header:
        note = row.get("notes", "")
        cells[header.index("notes")] = (note + "; " if note else "") + "promoted from candidates"
    lines.insert(where[-1] + 1, "| " + " | ".join(cells) + " |")
    CORPUS.write_text("\n".join(lines) + "\n", encoding="utf-8")

    _set_cell(CANDIDATES, target_slug, "outcome", f"cooked and kept {when} → corpus",
              overwrite=True)
    return True


def record_flop(target_slug: str, when: str, note: str = "") -> bool:
    """A candidate that was cooked and did not land. It stays in `candidates.md`
    with the reason, because at this corpus size that is the most informative
    signal the system gets all week and deleting it throws it away."""
    text = f"flopped {when}" + (f" — {note}" if note else "")
    return _set_cell(CANDIDATES, target_slug, "outcome", text, overwrite=True)


def apply_feedback(w: Week) -> list[str]:
    """Turn a week's feedback into corpus writes. Idempotent."""
    done = []
    for sl, outcome in w.feedback.items():
        o = outcome.lower()
        if o.startswith("not"):
            continue
        meal = next((m for m in w.meals if m.slug == sl), None)
        if meal is None:
            continue
        if meal.candidate:
            if o.startswith("kept") and promote(sl, w.date):
                done.append(f"{meal.title} → corpus")
            elif o.startswith("flop") and record_flop(sl, w.date):
                done.append(f"{meal.title} → flopped, kept in candidates")
        elif record_cooked(sl, w.date):
            done.append(f"{meal.title} → last cooked {w.date}")
    return done


# --------------------------------------------------------------------------- #
# Step 0: the briefing the prep job leaves behind
# --------------------------------------------------------------------------- #

def briefing() -> dict:
    """Read whatever the prep job cached. **Degrades, never blocks** - a session
    with no briefing is a normal session with one less card."""
    path = CACHE / "briefing.md"
    if not path.exists():
        return {"available": False, "lines": [], "generated": "",
                "note": "No prep run yet. Run ./prep.py before a session."}
    text = path.read_text(encoding="utf-8")
    gen = ""
    m = re.search(r"^generated:\s*(.+)$", text, re.M)
    if m:
        gen = m.group(1).strip()
    lines = [l[2:].strip() for l in text.splitlines() if l.strip().startswith("- ")]
    return {"available": True, "lines": lines, "generated": gen,
            "note": "demo data" if "DEMO" in text else ""}
