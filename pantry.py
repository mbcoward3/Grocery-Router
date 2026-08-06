#!/usr/bin/env python3
"""The core the app and the CLIs sit on.

Three jobs, and the split between them is the architecture:

**Read.** `corpus.md`, `candidates.md`, `profile.md` and `weeks/*.md` load into
plain dicts. These files stay the database - hand-editable markdown in git, so
history and audit come free and correcting the tool is still just editing a file.

**Propose.** `propose()` is one call with two implementations behind it. `rank()`
scores the corpus deterministically and always works; `planner/model.py` plans
with a model when a key is present, and anything it proposes that fails
validation is dropped and made up from the ranker. Two implementations behind one
call is what lets the app be alive with no setup.

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

def log(event: str, **payload) -> None:
    """Append one decision to `decisions.jsonl`.

    The first argument is `event` rather than `kind` so that `kind=` in a payload
    is a normal keyword and hits the guard below instead of colliding with the
    parameter. Every caller passes it positionally.

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
        rec = {"at": dt.datetime.now().isoformat(timespec="seconds"), "kind": event}
        # A payload key of `at` or `kind` would overwrite the record's own, and
        # `kind` is an easy one to reach for - the drop route wanted to record
        # which *reason* kind a meal was turned down against, and calling it
        # `kind` would have quietly replaced the decision type with it. The log
        # is the one file that has to stay literally true, so this is loud.
        clash = {"at", "kind"} & set(payload)
        if clash:
            raise ValueError(f"log payload may not use reserved key(s): "
                             f"{', '.join(sorted(clash))}")
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
    # Which *kind* of reason surfaced this - `stale`, `protein`, `low`, `model`.
    # Recorded because § 7 of the brief asks which reasons get accepted and which
    # get dropped, and the sentence alone cannot answer that: two meals stale at
    # different distances are two different sentences and one kind. Not persisted
    # to the week file, which has a fixed line format; the decision log is where
    # it needs to survive, and that is append-only.
    reason_kind: str = ""
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
                 candidate: bool = False) -> tuple[str, str]:
    """The `(kind, text)` of the best reason nothing else in the week has used. Five true
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
            return kind, text
    for kind, text in real:
        if text not in used_texts:
            used_texts.add(text)
            return kind, text
    return options[-1]


def rank(nights: int = 5, guests: float = 0.0, risk: str = "normal",
         keep: list[Meal] | None = None, today: dt.date | None = None,
         avoid: set[str] | None = None) -> list[Meal]:
    """Deterministic ranker. Fills the week up to `nights`, leaving `keep` alone.

    Gap-filling is the whole point: the session lets the household drop a meal
    and ask for another. Two things follow, and missing either makes the feature
    useless - the meals already accepted must not be re-rolled, and a meal that
    was just turned down must not come straight back. `avoid` carries the second.

    **Public, and named, because it is not a fallback to apologise for.** It is
    what runs in the hosted demo, in CI, and any week the model is unreachable,
    and `docs/architecture.md` is explicit that it has to stay genuinely good.
    `propose()` chooses between this and the model planner; anything that wants
    the deterministic answer specifically should call this directly and get it
    with no environment variable in the way.

    Does not write to the decision log. `propose()` logs once, for whichever
    implementation actually produced the week - two entries for one proposal
    would quietly corrupt every count read back off the log.
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
            candidate=best_cand,
        ))
        picked[-1].reason_kind, picked[-1].reason = _pick_reason(
            best_row, best_gap, picked, ae, used_kinds, used_texts, best_cand)
        if best_cand:
            used_candidates += 1

    return picked


def _log_proposal(picked: list[Meal], keep: list[Meal], nights: int, guests: float,
                  risk: str, week: str = "", **extra) -> None:
    log("proposed", nights=nights, guests=guests, risk=risk, week=week or monday(),
        kept=[m.slug for m in keep],
        added=[{"recipe": m.slug, "reason": m.reason, "candidate": m.candidate,
                "kind": m.reason_kind, "protein": m.protein, "cuisine": m.cuisine}
               for m in picked[len(keep):]],
        **extra)


def propose(nights: int = 5, guests: float = 0.0, risk: str = "normal",
            keep: list[Meal] | None = None, today: dt.date | None = None,
            avoid: set[str] | None = None, planner: str | None = None,
            client=None, week: str = "") -> list[Meal]:
    """Plan the week. **The one call, with two implementations behind it.**

    `planner/` decides which: `"ranker"` or `"model"`, from the argument, then
    `PANTRY_PLANNER`, then whether an API key is present. Callers do not have to
    care, and the session does not have a mode switch - a household with a key
    gets the better planner, a browser tab with no key gets a real week anyway.

    **The model may fill part of a week, and the ranker finishes it.** Every pick
    that fails validation - a slug that resolves to nothing, a reason claiming a
    recency no date supports, a peanut recipe - is dropped rather than repaired,
    which can leave the week short. Topping up from the ranker is why that is
    safe to do: refusing a bad pick costs a good reason, never a night's dinner.

    Falling back is recorded, never silent. `decisions.jsonl` gets the planner
    that produced the week and, when the model was asked and could not deliver,
    the sentence saying why - so `PANTRY_PLANNER=model` quietly degrading into
    the ranker for a month is a thing someone can find out.

    `client` is passed through to the model planner as its HTTP seam; tests use
    it, nothing else does. `week` is the Monday this proposal is *for*, which the
    log needs and cannot infer - proposing next week on a Sunday is a normal
    thing to do, and `review.py` grouping by wall clock would file it under the
    wrong one.
    """
    import planner as planners            # deferred: the browser build has no planner/

    keep = list(keep or [])
    which = planners.which(planner)
    if which != planners.MODEL:
        picked = rank(nights, guests, risk, keep, today, avoid)
        _log_proposal(picked, keep, nights, guests, risk, week, planner="ranker")
        return picked

    from planner import model as model_planner

    try:
        result = model_planner.plan(nights=nights, guests=guests, risk=risk, keep=keep,
                                    today=today, avoid=avoid, client=client)
    except model_planner.PlannerUnavailable as exc:
        picked = rank(nights, guests, risk, keep, today, avoid)
        _log_proposal(picked, keep, nights, guests, risk, week,
                      planner="ranker", asked="model", fallback=str(exc))
        return picked

    picked = keep + result.meals

    # **A short week is not always a failed week, and the difference matters.**
    #
    # `profile.md` says seven nights is not seven cooks - a big cook covers the
    # next day's lunches and can stand in for a dinner - and the prompt tells the
    # planner to scale a meal on purpose and say which night it feeds. It is also
    # told outright that proposing fewer nights than asked for is allowed and is
    # sometimes correct. So four cooks for five nights is frequently the *right*
    # answer, and topping it up to five silently deletes the leftover night the
    # week was built around.
    #
    # What the top-up is actually for is the picks validation took away. So it
    # makes up exactly those and no more: a refused pick still costs a good
    # reason and never a dinner, and a deliberate short week survives intact.
    # Told apart by whether anything was dropped, which is the only honest signal
    # available - a model that returned four meals and had none refused chose the
    # number, and it was allowed to.
    # Nothing survived at all is not a short week, it is a failed one: there is
    # no deliberate plan left to respect, so the ranker takes the whole thing.
    make_up = (nights - len(picked) if not result.meals
               else min(nights - len(picked), len(result.dropped)))
    topped_up = []
    if make_up > 0:
        picked = rank(len(picked) + make_up, guests, risk, picked, today, avoid)
        topped_up = [m.slug for m in picked[len(keep) + len(result.meals):]]

    _log_proposal(picked, keep, nights, guests, risk, week,
                  planner="model" if result.meals else "ranker",
                  asked="model", model=result.model,
                  note=result.note, coupling=result.coupling, gaps=result.gaps,
                  dropped=result.dropped, warnings=result.warnings,
                  topped_up=topped_up, planned_short=nights - len(picked),
                  fallback="" if result.meals else
                           "the model proposed nothing that survived validation")
    return picked


def last_proposal() -> dict | None:
    """The most recent `proposed` record, or `None`.

    How the session finds out which planner ran and what it had to say. The
    decision log is the right home for it rather than a variable on this module:
    a week is re-read from disk on every request and this process may not be the
    one that planned it, so anything held in memory would be gone by the time
    anybody looked. It also means the answer survives a restart, which is the
    same argument the log was built on.
    """
    records = decisions({"proposed"})
    return records[-1] if records else None


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


def _ensure_column(table: Table, name: str) -> tuple[Table, int]:
    """Make sure a column exists, adding it to the header and padding every row.

    Same move `onboard.ensure_yield_column` makes on the corpus. These files are
    hand-edited markdown and a column that appears in code before it appears on
    disk is a normal state, not a corruption - so it is migrated in place rather
    than demanded of whoever last touched the file.
    """
    if name in table.header:
        return table, table.header.index(name)
    lines = table.lines

    def widen(line: str, cell: str, pad: str) -> str:
        """Append one cell, keeping whatever spacing the file already uses. These
        are files a person edits by hand and diffs by eye; a migration that
        reformats the table it touched is a migration nobody can review."""
        body = line.rstrip()
        if not body.endswith("|"):
            body += "|"
        return f"{body}{pad}{cell}{pad}|"

    sep = table.header_i + 1
    has_sep = (sep < len(lines) and lines[sep].strip()
               and set(lines[sep].strip()) <= set("|-: "))
    # `| --- |` and `|---|` are both in use across these files.
    pad = " " if has_sep and " " in lines[sep].strip("| ") else ""

    lines[table.header_i] = widen(lines[table.header_i], name.title(), " ")
    if has_sep:
        lines[sep] = widen(lines[sep], "---", pad)
    for i in table.where:
        lines[i] = widen(lines[i], "", " ")
    CANDIDATES.write_text("\n".join(lines) + "\n", encoding="utf-8")
    table = _rows(CANDIDATES)
    return table, table.header.index(name)


def add_candidate(title: str, *, source: str, protein: str = "", cuisine: str = "",
                  yield_: str = "", active: str = "", passive: str = "",
                  proposed: str = "", found_by: str = "") -> bool:
    """Add a never-cooked recipe to `candidates.md`. Returns `False` if it is
    already known, raises `RuleViolation` if it does not belong here at all.

    **The door `upsert_corpus` refuses to be.** Onboarding says outright that a
    capture is not evidence about this household and that the row belongs in
    candidates; until now nothing put it there, so every candidate in the file
    was typed by a person and the tool could not grow its own corpus.

    Two refusals, and both protect the thing that makes the corpus worth having:

    **A source is required.** A candidate with no page behind it is the failure
    this whole project exists to avoid - `1 lb chicken breast because soup
    usually has chicken`. Acquisition may only add a recipe somebody could go and
    read, so the citation is a precondition of the write and not a nicety
    attached after it.

    **Nothing already in the corpus may be re-added.** A proven recipe reappearing
    as a gamble would quietly destroy the guarantee that corpus membership means
    cooked and liked, and it is the same mistake in reverse as putting a capture
    straight into the corpus.
    """
    if not (source or "").strip():
        raise RuleViolation(
            f"{title!r}: a candidate needs a source. A recipe nobody can go and read "
            f"is not an acquisition, it is an invention."
        )
    sl = slug(title)
    if any(r["slug"] == sl for r in load_corpus()):
        raise RuleViolation(
            f"{sl}: already in the corpus, where it is proven. A candidate carries the "
            f"gamble and this recipe has stopped being one."
        )
    if any(r["slug"] == sl for r in load_candidates()):
        return False

    table = _rows(CANDIDATES)
    table, _ = _ensure_column(table, "source")
    lines, where, header = table.lines, table.where, table.header
    values = {"recipe": title, "protein": protein, "cuisine": cuisine,
              "yield": yield_, "active": active, "passive": passive,
              "proposed": proposed or "—", "outcome": "untested", "source": source}
    cells = [values.get(col, "") for col in header]
    at = (where[-1] if where else table.header_i + 1) + 1
    lines.insert(at, "| " + " | ".join(cells) + " |")
    CANDIDATES.write_text("\n".join(lines) + "\n", encoding="utf-8")
    log("acquired", recipe=sl, source=source, found_by=found_by)
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
