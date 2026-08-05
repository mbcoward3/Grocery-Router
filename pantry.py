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
import json
import re
from dataclasses import dataclass, field, asdict
from pathlib import Path

ROOT = Path(__file__).resolve().parent
CORPUS = ROOT / "corpus.md"
CANDIDATES = ROOT / "candidates.md"
PROFILE = ROOT / "profile.md"
WEEKS = ROOT / "weeks"
CACHE = ROOT / ".cache"
DECISIONS = ROOT / "decisions.jsonl"

# From profile.md: 2 adults, a 3-year-old, a 1-year-old.
BASE_AE = 2.5


# --------------------------------------------------------------------------- #
# The decision log
# --------------------------------------------------------------------------- #

def log(kind: str, **payload) -> None:
    """Append one decision to `decisions.jsonl`.

    **A decision that was not recorded cannot be recovered.** That argument has
    nothing to do with where the data lives, which is why this survived the
    reversal back to files: it is what lets a change to the planner be replayed
    against real history instead of argued about, and it is the honest answer
    when someone who did not build this asks whether it works.

    One JSON object per line, append-only, never rewritten. Failing to log must
    never break a session, so this swallows its own errors - the log is
    evidence, not a dependency.
    """
    try:
        rec = {"at": dt.datetime.now().isoformat(timespec="seconds"), "kind": kind}
        rec.update(payload)
        with DECISIONS.open("a", encoding="utf-8") as fh:
            fh.write(json.dumps(rec, sort_keys=True) + "\n")
    except OSError:
        pass


def decisions(kinds: set[str] | None = None) -> list[dict]:
    if not DECISIONS.exists():
        return []
    out = []
    for line in DECISIONS.read_text(encoding="utf-8").splitlines():
        if not line.strip():
            continue
        try:
            rec = json.loads(line)
        except json.JSONDecodeError:
            continue
        if kinds is None or rec.get("kind") in kinds:
            out.append(rec)
    return out


# --------------------------------------------------------------------------- #
# Markdown tables
# --------------------------------------------------------------------------- #

@dataclass
class Table:
    """A parsed markdown table, with enough line bookkeeping that a write can put
    a value back exactly where it came from."""
    lines: list[str] = field(default_factory=list)
    rows: list[dict] = field(default_factory=list)
    where: list[int] = field(default_factory=list)
    header: list[str] = field(default_factory=list)
    header_i: int = -1

    def __iter__(self):                      # (lines, rows, where), as before
        return iter((self.lines, self.rows, self.where))


def _rows(path: Path) -> Table:
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
        row["slug"] = slug(row["recipe"])
        out.append(row)
        where.append(i)
    return Table(lines, out, where, header or [], header_i if header_i is not None else -1)


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


def load_members() -> list[str]:
    """Who lives here, read off `profile.md`.

    Attribution, not accounts. `profile.md` records that a stated preference
    "may be half-true" because nobody knows which adult said it - that needs a
    name on a claim, not a login. Auth arrives with hosting, against data that
    already knows who said what.
    """
    if not PROFILE.exists():
        return []
    members, inside = [], False
    for line in PROFILE.read_text(encoding="utf-8").splitlines():
        s = line.strip()
        if s.startswith("## "):
            inside = s[3:].strip().lower() == "members"
        elif inside and s.startswith("- "):
            name = s[2:].split("—")[0].split("(")[0].strip()
            if name:
                members.append(name)
    return members


_FILE_INDEX: dict[str, str] | None = None


def file_index() -> dict[str, str]:
    """Map a recipe identity onto the file that holds its ingredients.

    The index and the store are separate on purpose, so their names can drift -
    `corpus.md` says *Crock pot Italian beef sandwiches* and the file is
    `crock-pot-italian-beef.md`. Indexing by each file's own `# Title` as well as
    its stem catches that without renaming anything or hand-maintaining a map.
    """
    global _FILE_INDEX
    if _FILE_INDEX is None:
        _FILE_INDEX = {}
        for path in sorted((ROOT / "recipes").glob("*.md")):
            _FILE_INDEX.setdefault(path.stem, path.stem)
            for line in path.read_text(encoding="utf-8").splitlines():
                if line.startswith("# "):
                    _FILE_INDEX.setdefault(slug(line[2:]), path.stem)
                    break
    return _FILE_INDEX


def recipe_file(sl: str) -> Path:
    return ROOT / "recipes" / f"{file_index().get(sl, sl)}.md"


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

    @property
    def file(self) -> str:
        """The recipe file this meal's ingredients live in — not always its slug."""
        return file_index().get(self.slug, self.slug)

    @property
    def has_file(self) -> bool:
        return recipe_file(self.slug).exists()

    def to_json(self):
        d = asdict(self)
        d["yield"] = d.pop("yield_")
        d["file"] = self.file
        d["has_file"] = self.has_file
        return d


def _reasons(row: dict, gap: int | None, picked: list[Meal], ae: float,
             candidate: bool = False) -> list[tuple[str, str]]:
    """Every true, specific thing that can be said about why this surfaced, best
    first. Each is `(kind, text)`.

    The reason is the product - a forgotten recipe lands because you were told
    *you haven't made this since March*, and without that line it is only a
    suggestion. Which means the day-one case matters most: with no last-cooked
    dates, staleness has nothing to say, and a week of five identical
    "never recorded as cooked" lines is a week with no reasons at all.

    So staleness is one source among several, and the rest are computable from
    the corpus alone. Nothing here is invented; a claim about recency is only
    made when a date supports it.
    """
    out: list[tuple[str, str]] = []
    proteins = [m.protein for m in picked]
    cuisines = [m.cuisine for m in picked]

    if gap is not None and gap > 120:
        out.append(("stale", f"not cooked in {gap // 30} months"))
    elif gap is not None and gap > 45:
        out.append(("stale", f"last cooked {gap} days ago"))

    if row.get("protein") and row["protein"] not in proteins:
        out.append(("protein", f"the only {row['protein']} in the week so far"))

    y = yield_ae(row)
    if y and y >= ae * 2:
        out.append(("yield", f"serves {y:g} — one cook, two nights"))

    passive = (row.get("passive") or "").strip()
    slow = any(w in passive.lower() for w in ("slow cooker", "braise", "hr", "hour"))
    if active_rank(row) == 0 and slow:
        out.append(("passive", f"barely any hands-on time — {passive.split(' *or* ')[0]}"))
    elif active_rank(row) == 0:
        out.append(("low", "low active — a night a bad day cannot break"))

    if row.get("cuisine") and row["cuisine"] not in cuisines and len(picked) > 1:
        out.append(("cuisine", f"the only {row['cuisine']} this week"))

    if candidate:
        # A candidate has never been cooked here, so it can make no claim about
        # this household at all. Every reason above is about the shape of the
        # week, which is the only honest ground it has to stand on - and it must
        # never inherit a corpus recipe's fallback, which would assert
        # membership it does not have.
        out = [(k, "new here — " + t.split(" — ")[0]) for k, t in out]
        out.append(("acquire", "new here — widening the corpus is the other half of the job"))
        return out

    if gap is None:
        out.append(("never", "no record of cooking this yet — unranked, not stale"))
    elif gap > 20:
        out.append(("stale", f"last cooked {gap} days ago"))

    out.append(("plain", "in the corpus and nothing rules it out this week"))
    return out


def _pick_reason(row, gap, picked, ae, used_kinds: set, used_texts: set,
                 candidate: bool = False) -> str:
    """Prefer a reason nothing else in the week has already used. Five true
    sentences that are all the same sentence do not help anyone choose.

    A kind may repeat if the sentence differs - two meals can both be stale at
    different distances and that is still two useful reasons.
    """
    options = _reasons(row, gap, picked, ae, candidate)
    # `plain` says nothing. It must never win over a real reason just because it
    # is the only *kind* left unused - a second meal that is genuinely stale at a
    # different distance beats "nothing rules it out this week" every time.
    real = [(k, t) for k, t in options if k != "plain"]
    for kind, text in real:
        if kind not in used_kinds:
            used_kinds.add(kind)
            used_texts.add(text)
            return text
    for kind, text in real:
        if text not in used_texts:
            used_texts.add(text)
            return text
    return options[-1][1]


def propose(nights: int = 5, guests: float = 0.0, risk: str = "normal",
            keep: list[Meal] | None = None, today: dt.date | None = None,
            avoid: set[str] | None = None) -> list[Meal]:
    """Deterministic ranker. Fills the week up to `nights`, leaving `keep` alone.

    Gap-filling is the whole point: the session lets the household drop a meal
    and ask for another. Two things follow, and missing either makes the feature
    useless - the meals already accepted must not be re-rolled, and a meal that
    was just turned down must not come straight back. `avoid` carries the second.
    """
    today = today or dt.date.today()
    keep = list(keep or [])
    taken = {m.slug for m in keep} | set(avoid or ())
    ae = BASE_AE + guests

    pool = [r for r in load_corpus() if r["slug"] not in taken]
    cands = [r for r in load_candidates()
             if r["slug"] not in taken and "flopped" not in (r.get("outcome") or "")]
    want_cands = {"low": 0, "normal": 1, "high": 2}.get(risk, 1)

    picked = list(keep)
    used_candidates = sum(1 for m in keep if m.candidate)
    used_kinds: set[str] = set()
    used_texts: set[str] = set()

    def score(row, candidate):
        gap = days_since(row, today)
        s = 0.0
        # No date means *unknown*, not *maximally stale*. Scoring it at the top
        # would rank a recipe nobody has recorded above one measured to have
        # fallen out of rotation for six months - asserting dormancy the data
        # does not show, which is the same invention this project bans
        # everywhere else. Unknown sits at the ninety-day mark: ahead of
        # anything cooked recently, behind anything demonstrably stale.
        s += 30.0 if gap is None else min(gap, 365) / 3.0
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
        # Candidates lose every head-to-head by design - they carry the gamble,
        # and at 24 never-cooked proven recipes they would never surface at all.
        # So the risk dial reserves slots rather than nudging a score: acquisition
        # is half the job and cannot be left to win a fight it is built to lose.
        needed = max(0, min(want_cands, len(cands)) - used_candidates)
        force_candidate = needed > 0 and (nights - len(picked)) <= needed

        best, best_row, best_gap, best_cand = None, None, None, False
        for row, candidate in [(r, False) for r in pool] + [(r, True) for r in cands]:
            if candidate and used_candidates >= want_cands:
                continue
            if force_candidate and not candidate:
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
            reason=_pick_reason(best_row, best_gap, picked, ae, used_kinds, used_texts,
                                best_cand),
            candidate=best_cand,
        ))
        if best_cand:
            used_candidates += 1

    log("proposed", nights=nights, guests=guests, risk=risk,
        kept=[m.slug for m in keep],
        added=[{"recipe": m.slug, "reason": m.reason, "candidate": m.candidate}
               for m in picked[len(keep):]])
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
    # slug -> {"outcome": kept | flopped | not cooked, "by": member or ""}
    feedback: dict = field(default_factory=dict)
    declined: list[str] = field(default_factory=list)   # turned down this week

    @property
    def ae(self) -> float:
        return BASE_AE + self.guests

    def path(self) -> Path:
        return WEEKS / f"{self.date}.md"

    def to_json(self):
        return {"date": self.date, "nights": self.nights, "guests": self.guests,
                "risk": self.risk, "status": self.status, "ae": self.ae,
                "meals": [m.to_json() for m in self.meals], "declined": self.declined,
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
        if section == "declined" and s.startswith("- "):
            w.declined.append(slug(s[2:]))
        if section == "feedback" and s.startswith("- "):
            name, _, rest = s[2:].partition(":")
            if rest:
                outcome, _, by = rest.strip().partition(" — ")
                w.feedback[slug(name)] = {"outcome": outcome.strip(), "by": by.strip()}
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
    if w.declined:
        out += ["", "## Declined", "",
                "*Offered this week and turned down. Kept so gap-filling does not hand "
                "back the thing you just refused.*", ""]
        out += [f"- {d.replace('-', ' ')}" for d in w.declined]
    if w.feedback:
        out += ["", "## Feedback", ""]
        index = {m.slug: m.title for m in w.meals}
        for sl, fb in w.feedback.items():
            by = f" — {fb['by']}" if fb.get("by") else ""
            out.append(f"- {index.get(sl, sl)}: {fb['outcome']}{by}")
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
    table = _rows(path)
    lines, rows, where, header = table.lines, table.rows, table.where, table.header
    if not rows:
        return False
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


class RuleViolation(Exception):
    """A write the rules refuse. Raised, never swallowed - a silent refusal is
    how the corpus quietly stops meaning what it says."""


def promote(target_slug: str, when: str, outcome: str = "kept") -> bool:
    """Move a candidate into the corpus. **The only function in this project that
    may add a row to `corpus.md`**, and it requires a recorded cook that was
    kept.

    Membership is earned. Having a file, a URL, a yield or an enthusiastic
    reason is not that. Under a schema this is `state='corpus'` requiring a
    `cook_log` row; here it is this function being the only door.
    """
    if not outcome.lower().startswith("kept"):
        raise RuleViolation(
            f"{target_slug}: a candidate enters the corpus only on a cook that was "
            f"kept, and this one is {outcome!r}. Membership is earned."
        )
    cands = {r["slug"]: r for r in load_candidates()}
    row = cands.get(target_slug)
    if row is None:
        raise RuleViolation(
            f"{target_slug}: not a candidate. Nothing may enter the corpus without "
            f"passing through candidates first — that path is what records the cook."
        )
    if any(r["slug"] == target_slug for r in load_corpus()):
        return False

    table = _rows(CORPUS)
    lines, where, header = table.lines, table.where, table.header
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
    log("promote", recipe=target_slug, when=when)
    return True


def add_profile_claim(section: str, text: str, evidence: str, by: str = "") -> None:
    """Append a claim to `profile.md`, **refusing one with no evidence.**

    The rule `profile.md` opens with: no claim without a trace. It was stated in
    prose and enforced by whoever was reading. Under a schema this is
    `evidence NOT NULL`; here it is this guard.
    """
    if not (evidence or "").strip():
        raise RuleViolation(
            f"claim {text!r} has no evidence. A profile that quietly accumulates "
            f"ungrounded assertions poisons every week downstream — if you cannot "
            f"say why you believe it, it does not go in."
        )
    lines = PROFILE.read_text(encoding="utf-8").splitlines()
    want = section.strip().lower()
    at = None
    for i, line in enumerate(lines):
        if line.strip().lower() == f"## {want}":
            at = i
        elif at is not None and line.strip().startswith("## "):
            break
    if at is None:
        raise RuleViolation(f"no '## {section}' section in profile.md")
    end = next((i for i in range(at + 1, len(lines))
                if lines[i].strip().startswith("## ")), len(lines))
    who = f" *— {by}*" if by else ""
    lines.insert(end, f"- {text} *Because {evidence.rstrip('.')}.*{who}")
    lines.insert(end + 1, "")
    PROFILE.write_text("\n".join(lines) + "\n", encoding="utf-8")
    log("profile_claim", section=section, text=text, evidence=evidence, by=by)


def record_flop(target_slug: str, when: str, note: str = "") -> bool:
    """A candidate that was cooked and did not land. It stays in `candidates.md`
    with the reason, because at this corpus size that is the most informative
    signal the system gets all week and deleting it throws it away."""
    text = f"flopped {when}" + (f" — {note}" if note else "")
    return _set_cell(CANDIDATES, target_slug, "outcome", text, overwrite=True)


def apply_feedback(w: Week) -> list[str]:
    """Turn a week's feedback into corpus writes. Idempotent.

    This is the write that makes the whole thing work. Without it every recipe
    looks equally forgotten forever and the ranker has no staleness to rank on.
    """
    done = []
    for sl, fb in w.feedback.items():
        outcome = fb["outcome"] if isinstance(fb, dict) else str(fb)
        by = fb.get("by", "") if isinstance(fb, dict) else ""
        o = outcome.lower()
        if o.startswith("not"):
            log("feedback_applied", recipe=sl, outcome=outcome, by=by, week=w.date,
                effect="none — not cooked")
            continue
        meal = next((m for m in w.meals if m.slug == sl), None)
        if meal is None:
            continue
        effect = ""
        if meal.candidate:
            if o.startswith("kept"):
                if promote(sl, w.date, outcome):
                    effect = "promoted to corpus"
                    done.append(f"{meal.title} → corpus")
            elif o.startswith("flop") and record_flop(sl, w.date):
                # Never deleted. At this corpus size a flop is the most
                # informative signal the system gets all week.
                effect = "flopped, kept in candidates"
                done.append(f"{meal.title} → flopped, stays in candidates with the reason")
        elif record_cooked(sl, w.date):
            effect = f"last cooked {w.date}"
            done.append(f"{meal.title} → last cooked {w.date}")
        log("feedback_applied", recipe=sl, outcome=outcome, by=by, week=w.date,
            candidate=meal.candidate, effect=effect)
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
