"""The model planner: the second implementation behind `pantry.propose()`.

The ranker can say *not cooked in 11 months* and *the only beef this week*. It
cannot say *you drifted away from Italian around March and this is the one you
kept going back to before that*, and the proposal document is explicit that the
reason **is** the product - a forgotten recipe lands on the reason, not on the
suggestion. That sentence is what this module is for.

**The division of labour, which is the whole design.** The model *selects and
explains*. It does not name food, and it does not state facts about a meal.
Every field on the `Meal` it produces except `reason` is read back off the corpus
row that its slug resolved to. So a hallucinated protein has nowhere to land, a
hallucinated yield has nowhere to land, and a hallucinated *recipe* resolves to
nothing and is dropped.

That is deliberate, and it is the shape of a bug this repo has a receipt for.
Asked to reason about ingredient coupling from a corpus index that contained no
ingredients, a model manufactured the coupling and then chose a candidate
*because of the coupling it had invented*. The lesson taken was not "models are
untrustworthy" - it was **give it only what the corpus actually contains, and
give it nothing to invent into.** The catalogue below is that principle as code:
it is generated from the same rows the ranker scores, it carries a computed
`days since` column so no date arithmetic is ever asked for, and it carries no
ingredients at all.

**Degrading is normal, not exceptional.** No key, no network, a 500, a truncated
reply, unparseable JSON, a week of invented slugs - every one of them ends the
same way, with `pantry.propose()` falling through to the ranker and recording
what happened in `decisions.jsonl`. The ranker is not an apology; it is what runs
in the hosted demo and in CI, so the fallback path is the well-tested one.

Standard library only.
"""

from __future__ import annotations

import datetime as dt
import json
import os
import re
import urllib.error
import urllib.request
from dataclasses import dataclass, field

import pantry

from . import constraints
from .prompt import EMPTY_CORPUS_NOTE, PLANNER_PROMPT, SELECTION_CONTRACT

API_VERSION = "2023-06-01"
DEFAULT_MODEL = os.environ.get("PANTRY_MODEL", "claude-sonnet-5")
TIMEOUT = float(os.environ.get("PANTRY_TIMEOUT", "90"))


def api_url() -> str:
    base = os.environ.get("ANTHROPIC_BASE_URL", "https://api.anthropic.com").rstrip("/")
    return f"{base}/v1/messages"


class PlannerUnavailable(Exception):
    """The model could not be reached, or did not answer usably.

    One exception type for every failure between here and a parsed week, because
    every one of them has the same consequence: the ranker plans instead. The
    message is kept human - it is written into the decision log and shown in the
    session, so *why* the week is deterministic this time is never a mystery.
    """


@dataclass
class Plan:
    """What the model planner hands back.

    `dropped` and `warnings` are as much the product as `meals`. A week that
    silently lost two picks to validation looks identical to a week the model
    only half-filled, and those are very different facts about how well this is
    working.
    """
    meals: list = field(default_factory=list)
    note: str = ""
    coupling: str = ""
    gaps: str = ""
    dropped: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)
    model: str = ""
    raw: str = ""


# --------------------------------------------------------------------------- #
# What the model is given
# --------------------------------------------------------------------------- #

CATALOGUE_COLUMNS = ["slug", "recipe", "protein", "cuisine", "yield", "active",
                     "passive", "days since"]


def _days_since_cell(row: dict, today: dt.date) -> str:
    """`unknown` is a value here, not a blank to be filled in or glossed.

    The ranker learned this the hard way: a recipe with no last-cooked date was
    scoring as *maximally* stale, ranking above one measured dormant for six
    months, because absence had been read as an extreme. The word is spelled out
    rather than left empty so that a blank cell can never be mistaken for a zero,
    and `constraints.check_meal` refuses any recency claim made about a row whose
    cell reads this.
    """
    gap = pantry.days_since(row, today)
    return "unknown" if gap is None else str(gap)


def catalogue(rows: list[dict], today: dt.date) -> str:
    """The rows, as a table whose first column is the only name that counts.

    The model picks by slug and the slug is validated back against this exact
    index, so a pretty title is never load-bearing. Nothing here comes from
    anywhere but the corpus files.
    """
    out = ["| " + " | ".join(CATALOGUE_COLUMNS) + " |",
           "| " + " | ".join("---" for _ in CATALOGUE_COLUMNS) + " |"]
    for row in rows:
        out.append("| " + " | ".join([
            row["slug"],
            row.get("recipe", ""),
            row.get("protein", "") or "—",
            row.get("cuisine", "") or "—",
            row.get("yield", "") or "unknown",
            row.get("active", "") or "—",
            row.get("passive", "") or "—",
            _days_since_cell(row, today),
        ]) + " |")
    return "\n".join(out)


def week_block(nights: int, guests: float, risk: str, keep: list, avoid: set) -> str:
    lines = [f"Nights to plan:       {nights}",
             f"Already on the board: " +
             (", ".join(m.title for m in keep) if keep else "nothing yet"),
             f"Guests:               {guests:g} extra adult-equivalents"
             if guests else "Guests:               none",
             f"Risk appetite:        {risk}"]
    if avoid:
        lines.append("Turned down already:  " + ", ".join(sorted(avoid)))
    return "\n".join(lines)


def build_prompt(corpus: list[dict], cands: list[dict], profile_text: str,
                 nights: int, guests: float, risk: str, keep: list, avoid: set,
                 today: dt.date) -> str:
    """Assemble the request. Order matters only in that the instruction comes
    first and the data last, which is where a long context puts its attention.
    """
    if corpus:
        corpus_section = ("These have all been cooked and liked here. Proposing one is "
                          "risk-free.\n\n" + catalogue(corpus, today))
    else:
        corpus_section = EMPTY_CORPUS_NOTE

    if cands:
        cand_section = ("Never cooked here. Every one of these is a gamble, and the "
                        "prose must mark it `[candidate]`.\n\n" + catalogue(cands, today))
    else:
        cand_section = "None waiting."

    return (
        f"{PLANNER_PROMPT}{SELECTION_CONTRACT}\n\n"
        f"---\n\n## This week\n\n```\n"
        f"{week_block(nights, guests, risk, keep, avoid)}\n```\n\n"
        f"---\n\n## Household profile\n\n{profile_text.strip()}\n\n"
        f"---\n\n## Catalogue — corpus\n\n{corpus_section}\n\n"
        f"---\n\n## Catalogue — candidates\n\n{cand_section}\n"
    )


# --------------------------------------------------------------------------- #
# The call
# --------------------------------------------------------------------------- #

def call(prompt: str, model: str | None = None, api_key: str | None = None,
         max_tokens: int = 4000) -> str:
    """One HTTP implementation, shared with `plan.py`.

    The CLI wants to exit on an error and the app wants to fall back to the
    ranker, so the *posture* differs between the two callers - but the request
    does not, and this project has already paid for two implementations of one
    thing that drifted. The exception is the seam: raise here, and let each
    caller decide what an unreachable model means to it.
    """
    api_key = api_key or os.environ.get("ANTHROPIC_API_KEY")
    if not api_key:
        raise PlannerUnavailable("no ANTHROPIC_API_KEY set")
    payload = json.dumps({
        "model": model or DEFAULT_MODEL,
        "max_tokens": max_tokens,
        "messages": [{"role": "user", "content": prompt}],
    }).encode()
    req = urllib.request.Request(api_url(), data=payload, headers={
        "content-type": "application/json",
        "anthropic-version": API_VERSION,
        "x-api-key": api_key,
    })
    try:
        with urllib.request.urlopen(req, timeout=TIMEOUT) as resp:
            body = json.load(resp)
    except urllib.error.HTTPError as e:
        detail = e.read().decode(errors="replace")[:400]
        raise PlannerUnavailable(f"API error {e.code}: {detail}") from e
    except urllib.error.URLError as e:
        raise PlannerUnavailable(f"could not reach the API: {e.reason}") from e
    except (TimeoutError, OSError) as e:
        raise PlannerUnavailable(f"could not reach the API: {e}") from e
    except json.JSONDecodeError as e:
        raise PlannerUnavailable(f"the API returned something that was not JSON: {e}") from e

    if body.get("stop_reason") == "max_tokens":
        # A truncated reply loses the JSON block, which is the last thing in it.
        # Saying so beats "no json block found", which sends the next person
        # looking at the prompt instead of at the token ceiling.
        raise PlannerUnavailable("the reply hit max_tokens and was cut off before the plan")
    text = "".join(b.get("text", "") for b in body.get("content", []) if b.get("type") == "text")
    if not text.strip():
        raise PlannerUnavailable("the API returned an empty reply")
    return text


# --------------------------------------------------------------------------- #
# What comes back
# --------------------------------------------------------------------------- #

JSON_BLOCK = re.compile(r"```json\s*(\{.*?\})\s*```", re.S)


def extract(text: str) -> dict:
    """Pull the envelope out of the reply. The prose around it is kept, not parsed.

    Takes the **last** JSON block, because a model that talks about its output
    format before producing it will sometimes emit an illustrative one first.
    """
    blocks = JSON_BLOCK.findall(text)
    if not blocks:
        raise PlannerUnavailable("the reply had no ```json block, so no week could be read")
    try:
        data = json.loads(blocks[-1])
    except json.JSONDecodeError as e:
        raise PlannerUnavailable(f"the reply's json block did not parse: {e}") from e
    if not isinstance(data, dict) or not isinstance(data.get("meals"), list):
        raise PlannerUnavailable("the reply's json block had no `meals` array")
    return data


def _meal_from(row: dict, reason: str, candidate: bool):
    """Build the `Meal` from the row, taking exactly one thing from the model.

    Listing the fields out rather than looping is the point: every line here is a
    fact the model was told not to supply and could not have supplied anyway.
    """
    return pantry.Meal(
        slug=row["slug"],
        title=row["recipe"],
        protein=row.get("protein", ""),
        cuisine=row.get("cuisine", ""),
        yield_=row.get("yield", ""),
        active=(row.get("active") or "").lower(),
        passive=row.get("passive", ""),
        variants=pantry.variants_for(row["slug"]),
        variant=(pantry.variants_for(row["slug"]) or [""])[0],
        reason=(reason or "").strip(),
        # One kind for everything a model wrote. The ranker's kinds describe
        # *which rule* surfaced a meal, and a model's reason is not the output of
        # a rule - claiming `stale` or `protein` for it would put a made-up
        # provenance into the decision log, which is the one file that has to
        # stay literally true.
        reason_kind="model",
        candidate=candidate,
    )


def select(data: dict, index: dict, taken: set, limit: int) -> tuple[list, list[str]]:
    """Turn the envelope into meals, dropping everything that cannot be trusted.

    Returns `(meals, dropped)`, where each entry in `dropped` is a sentence
    saying what was refused and why. **Nothing is repaired.** A slug that nearly
    matches is not resolved to its neighbour - `onion powder` resolving to
    `onion` across thirteen lines is the bug that taught this repo that a silent
    mis-merge is worse than a loud gap.
    """
    meals, dropped = [], []
    for entry in data["meals"]:
        if len(meals) >= limit:
            dropped.append("more meals were proposed than there are nights to fill")
            break
        if not isinstance(entry, dict):
            dropped.append(f"a meal entry was {type(entry).__name__}, not an object")
            continue

        sl = str(entry.get("slug", "")).strip()
        row = index.get(sl)
        if row is None:
            # Deliberately not fuzzy-matched. If this fires often the fix is the
            # catalogue or the prompt, and a near-match would hide that signal.
            dropped.append(f"{sl or '(no slug)'!r} is not in the corpus or the candidates")
            continue
        if sl in taken:
            dropped.append(f"{row['recipe']} was already on the board or turned down")
            continue

        reason = str(entry.get("reason", "")).strip()
        if not reason:
            # The reason is the product. A pick without one is not a cheaper
            # pick, it is a suggestion, and the ranker writes better ones.
            dropped.append(f"{row['recipe']} came back with no reason")
            continue

        meal = _meal_from(row, reason, candidate=row.get("_candidate", False))
        why = constraints.check_meal(meal, row)
        if why:
            dropped.append(why)
            continue

        meals.append(meal)
        taken.add(sl)
    return meals, dropped


# --------------------------------------------------------------------------- #
# The entry point
# --------------------------------------------------------------------------- #

def plan(nights: int = 5, guests: float = 0.0, risk: str = "normal",
         keep: list | None = None, today: dt.date | None = None,
         avoid: set | None = None, client=None, model: str | None = None) -> Plan:
    """Plan the week with a model. Raises `PlannerUnavailable` if it cannot.

    `client` is the seam for tests: any callable taking the assembled prompt and
    returning the reply text. It defaults to the real API. Injecting it is what
    lets the validation, the constraint enforcement and the fallback all be
    tested without a key and without a network - which matters, because those
    paths are the ones that have to work when the model is having a bad day.

    Raising rather than returning an empty plan is on purpose: `propose()` needs
    to tell *the model was not available* apart from *the model planned zero
    meals*, and only one of those is a fallback.
    """
    today = today or dt.date.today()
    keep = list(keep or [])
    avoid = set(avoid or ())
    limit = nights - len(keep)
    if limit <= 0:
        return Plan(meals=[], note="the week was already full", model=model or DEFAULT_MODEL)

    corpus = pantry.load_corpus()
    cands = [r for r in pantry.load_candidates()
             if "flopped" not in (r.get("outcome") or "")]

    index: dict[str, dict] = {}
    for row in corpus:
        index[row["slug"]] = dict(row, _candidate=False)
    for row in cands:
        # A slug in both files is a corpus row; membership is earned and the
        # corpus is the stronger claim. `promote()` is supposed to make this
        # impossible, and it is cheaper to be right here than to rely on that.
        index.setdefault(row["slug"], dict(row, _candidate=True))

    profile_text = pantry.PROFILE.read_text(encoding="utf-8") \
        if pantry.PROFILE.exists() else ""
    prompt = build_prompt(corpus, cands, profile_text, nights, guests, risk,
                          keep, avoid, today)

    client = client or (lambda p: call(p, model=model))
    text = client(prompt)
    data = extract(text)

    taken = {m.slug for m in keep} | avoid
    meals, dropped = select(data, index, taken, limit)

    warnings = constraints.check_week(keep + meals)
    stale = constraints.unchecked(meals)
    if stale:
        warnings.append("no peanut scan on record for " + ", ".join(stale))

    return Plan(
        meals=meals,
        note=str(data.get("note", "")).strip(),
        coupling=str(data.get("coupling", "")).strip(),
        gaps=str(data.get("gaps", "")).strip(),
        dropped=dropped,
        warnings=warnings,
        model=model or DEFAULT_MODEL,
        raw=text,
    )
