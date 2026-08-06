#!/usr/bin/env python3
"""The weekly session. One page, once a week.

    ./app.py                 # http://127.0.0.1:8765
    ./app.py --port 9000 --no-open

The session in order: what happened last week, what to cook this week, the dials,
the list. Nothing else. The household touches this once and is done.

**The server owns nothing.** State lives in `weeks/<date>.md` and the markdown
files that were already the database; this process is a view over them and can be
killed at any point without losing a thing. Every mutation goes through
`pantry.py`, which is where the rules live.

Standard library only, no build step, no dependencies.
"""

import argparse
import json
import os
import shutil
import sys
import tempfile
import threading
import webbrowser
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

import pantry
import planner
import review
import shop

ROOT = Path(__file__).resolve().parent
WEB = ROOT / "web"
DEMO: Path | None = None


# --------------------------------------------------------------------------- #
# Demo mode
# --------------------------------------------------------------------------- #

DEMO_FILES = ("corpus.md", "candidates.md", "sides.md", "profile.md", "items.md")
DEMO_SRC = ROOT / "demo"


def _demo_source(name: str) -> Path:
    """`demo/` wins where it has an opinion; the root fills the rest.

    Only three files differ - the corpus, the candidates and the profile. The
    recipes and the item table are public recipe data with nothing private in
    them, so they are shared rather than duplicated and left to drift.
    """
    override = DEMO_SRC / name
    return override if override.exists() else ROOT / name


def start_demo() -> Path:
    """Run against a scratch copy of an invented household's files.

    Two separate problems, and the demo directory solves the second one:

    **A stranger's hands on the corpus.** Everything a session writes -
    last-cooked dates, promotions, the decision log - is real behaviour that has
    to work, so it cannot be stubbed; it just must not land in the repo. Hence
    the temp copy, and Reset to put it back.

    **A stranger's eyes on the household.** The real `profile.md` carries who
    lives here, their ages and a food allergy. `demo/` replaces it, so a hosted
    deployment serves an invented family - which also makes the better demo,
    since the invented one can have the cooking history the real one has not
    accumulated yet.
    """
    global DEMO
    DEMO = Path(tempfile.mkdtemp(prefix="pantry-demo-"))
    # Repoint *before* populating. Seeding the demo's first week while pantry
    # still points at the repo writes the demo's data into the real household -
    # which it did, once, and which is the whole failure this mode exists to
    # prevent.
    pantry.ROOT = DEMO
    pantry.CORPUS = DEMO / "corpus.md"
    pantry.CANDIDATES = DEMO / "candidates.md"
    pantry.SIDES = DEMO / "sides.md"
    pantry.PROFILE = DEMO / "profile.md"
    pantry.WEEKS = DEMO / "weeks"
    pantry.CACHE = DEMO / ".cache"
    pantry.DECISIONS = DEMO / "decisions.jsonl"
    pantry._FILE_INDEX = None
    reset_demo()
    return DEMO


def reset_demo() -> None:
    if DEMO is None:
        return
    for name in DEMO_FILES:
        shutil.copy(_demo_source(name), DEMO / name)
    if not (DEMO / "recipes").exists():
        shutil.copytree(ROOT / "recipes", DEMO / "recipes")
    shutil.rmtree(DEMO / "weeks", ignore_errors=True)
    (DEMO / "decisions.jsonl").unlink(missing_ok=True)
    # Carry over a briefing cached at image-build time, so a hosted visitor
    # lands on a full page instead of an empty card.
    shutil.rmtree(DEMO / ".cache", ignore_errors=True)
    if (ROOT / ".cache").exists():
        shutil.copytree(ROOT / ".cache", DEMO / ".cache")
    pantry._FILE_INDEX = None
    seed_last_week()          # last, or the rmtree above deletes it


def seed_last_week():
    """Give the demo a week to give feedback on.

    The first stage of a session is *what happened last week*, and with no
    previous week it does not render at all - so a first-time visitor never sees
    the feedback loop, which is the mechanism the whole product rests on. The
    seeded week is left unanswered on purpose: the visitor gets to be the one
    who closes it and watch the corpus change.
    """
    import datetime as _dt
    last = (_dt.date.fromisoformat(pantry.monday()) - _dt.timedelta(days=7)).isoformat()
    if pantry.read_week(last):
        return

    # Named rather than re-proposed. Running the ranker for last week returns
    # this week's answer - correctly, since nothing has said those meals were
    # cooked - but a visitor reads two identical weeks as a broken tool. These
    # four are the ones whose last-cooked dates in demo/corpus.md already fall
    # inside that window, so the seed agrees with the history rather than
    # inventing a second one. Three proven to confirm, one candidate to promote.
    want = ["blt", "chili", "tacos", "chicken-and-dumplings"]
    index = {r["slug"]: r for r in pantry.load_corpus() + pantry.load_candidates()}
    proven = {r["slug"] for r in pantry.load_corpus()}

    w = pantry.Week(date=last, nights=4, status="planning")
    for sl in want:
        row = index.get(sl)
        if row is None:
            continue
        w.meals.append(pantry.Meal(
            slug=sl, title=row["recipe"], protein=row.get("protein", ""),
            cuisine=row.get("cuisine", ""), yield_=row.get("yield", ""),
            active=(row.get("active") or "").lower(), passive=row.get("passive", ""),
            variants=pantry.variants_for(sl), candidate=sl not in proven,
            reason="planned last week"))
    pantry.write_week(w)
    pantry.DECISIONS.unlink(missing_ok=True)   # the seed is not a real decision


# --------------------------------------------------------------------------- #
# State
# --------------------------------------------------------------------------- #

def current_week() -> pantry.Week:
    date = pantry.monday()
    week = pantry.read_week(date)
    if week is None:
        week = pantry.Week(date=date)
        week.meals = pantry.propose(week.nights, week.guests, week.risk,
                                    week=week.date)
        pantry.write_week(week)
    return week


def state() -> dict:
    week = current_week()
    prev = pantry.previous_week(week.date)
    corpus = pantry.load_corpus()
    return {
        "week": week.to_json(),
        "previous": prev.to_json() if prev else None,
        "briefing": pantry.briefing(),
        "members": pantry.load_members(),
        "counts": {"corpus": len(corpus), "candidates": len(pantry.load_candidates()),
                   "sides": len(pantry.load_sides())},
        "metrics": metrics(),
        "planner": planner_state(),
        "demo": DEMO is not None,
    }


def planner_state() -> dict:
    """Which planner produced the week on the board, and what it had to say.

    This used to be the string `"ranker"`, hardcoded, which was true and stopped
    being true the moment a model could plan. It is read back off the decision
    log rather than held in memory because the week is re-read from disk on every
    request and this process may not be the one that planned it.

    `dropped` and `warnings` are surfaced rather than swallowed on purpose. A
    model week that quietly lost three picks to validation and got topped up by
    the ranker is *not* the same week as one the model planned outright, and the
    difference is exactly what someone evaluating whether this works needs to
    see.
    """
    last = pantry.last_proposal() or {}
    return {
        "used": last.get("planner", "ranker"),
        "asked": last.get("asked", ""),
        "available": planner.which(),
        "model": last.get("model", ""),
        "note": last.get("note", ""),
        "coupling": last.get("coupling", ""),
        "gaps": last.get("gaps", ""),
        "dropped": last.get("dropped", []),
        "warnings": last.get("warnings", []),
        "fallback": last.get("fallback", ""),
        "planned_short": last.get("planned_short", 0),
    }


def metrics() -> dict:
    """The product claim, as numbers. It says the gap between the recipes you
    reach for and the ones you like *is* the product - so the honest measures are
    how much of the corpus is still dormant and how much of it gets cooked.

    Read off the decision log and the corpus. Zeroes are correct at week one and
    are shown as zeroes rather than hidden.
    """
    corpus = pantry.load_corpus()
    cooked = {r["slug"] for r in corpus if (r.get("last cooked") or "").strip()}
    applied = pantry.decisions({"feedback_applied"})
    proposed = pantry.decisions({"proposed"})
    surfaced = {a["recipe"] for d in proposed for a in d.get("added", [])}
    drops = len(pantry.decisions({"drop"}))
    offered = sum(len(d.get("added", [])) for d in proposed)
    weeks = review.breadth()
    return {
        "corpus": len(corpus),
        "cooked_ever": len(cooked),
        "dormant": len(corpus) - len(cooked),
        "surfaced_ever": len(surfaced),
        "kept": sum(1 for d in applied if d.get("outcome", "").startswith("kept")),
        "accept_rate": None if not offered else round(100 * (offered - drops) / offered),
        "weeks": len(pantry.list_weeks()),
        # §7: the five numbers above are read off the corpus and describe what
        # the household owns. These are read off `decisions.jsonl` and describe
        # what the tool did, which is the only thing that can answer *is it any
        # good* for someone who did not build it.
        "behaviour": {
            "reasons": review.reasons()[:6],
            "trend": weeks[-6:],
            "widening": (len(weeks) > 1
                         and weeks[-1]["distinct_so_far"] > weeks[0]["distinct_so_far"]),
        },
    }


def refill(week: pantry.Week) -> pantry.Week:
    """Fill up to `nights`, leaving everything already on the board alone.

    This is the gap-filling the session is built around: dropping one meal and
    asking for another must not re-roll the four the household already accepted.
    """
    week.meals = pantry.propose(week.nights, week.guests, week.risk, keep=week.meals,
                                avoid=set(week.declined), week=week.date)
    pantry.write_week(week)
    return week


def grocery_list(week: pantry.Week) -> dict:
    """Step 2, called in process.

    This used to shell out to `shop.py`. It doesn't any more, for a reason worth
    recording: the browser build has no subprocesses. Calling the same functions
    directly is what lets one implementation serve both, rather than a second
    copy of the pipeline that has to be kept honest against the first.
    """
    missing = [m.title for m in week.meals if not m.has_file]
    have = [m for m in week.meals if m.has_file]
    specs = [m.file + (f":{m.variant}" if m.variant else "") for m in have]
    # Sides go through the same pipeline as everything else - same parser, same
    # aggregation, same consolidation - so a side sharing an onion with a main
    # merges into one line rather than appearing twice. That is the whole reason
    # they are captured as recipe files instead of as a list of words.
    sides = [r for r in pantry.load_sides()
             if r["slug"] in week.sides and pantry.recipe_file(r["slug"]).exists()]
    side_missing = [r.get("side") or r["recipe"] for r in pantry.load_sides()
                    if r["slug"] in week.sides and not pantry.recipe_file(r["slug"]).exists()]
    specs += [pantry.file_index().get(r["slug"], r["slug"]) for r in sides]
    if not specs:
        return {"ok": True, "markdown": "_Nothing planned yet._"}

    shop.configure(pantry.ROOT)
    ae = pantry.BASE_AE + week.guests
    # One number per meal, so guests on Thursday scale Thursday and not the
    # week. Meals with no override get the week's number, so a week nobody has
    # touched produces exactly the list it did before.
    try:
        built = shop.build(specs, [m.ae(ae) for m in have] + [ae] * len(sides))
    except (FileNotFoundError, SystemExit) as exc:
        return {"ok": False, "markdown": f"```\n{exc}\n```"}
    meals, lines, unknown, merges, links, scales, items = built
    out = shop.emit(meals, lines, unknown, merges, links, ae, scales, items,
                    sides=len(sides))
    out += "\n" + shop.coupling_report(lines, items)
    if missing or side_missing:
        # Surfaced, never silent. A list that quietly omits a meal is the failure
        # this whole pipeline is built to avoid — and a side typed in by name,
        # with no capture behind it, is exactly that failure in miniature. It is
        # on the week, it contributes nothing, and the only honest thing to do is
        # say which one you are shopping for yourself.
        out += ("\n\n## Not on this list\n\n*No ingredient file exists for these yet, so "
                "nothing was added for them:*\n\n"
                + "\n".join(f"- {t}" for t in missing)
                + ("\n" if missing and side_missing else "")
                + "\n".join(f"- {t} (side)" for t in side_missing) + "\n")
    return {"ok": True, "markdown": out}


# --------------------------------------------------------------------------- #
# Routes
# --------------------------------------------------------------------------- #

def post_dials(body):
    week = current_week()
    before = {"nights": week.nights, "guests": week.guests, "risk": week.risk}
    week.nights = max(1, min(14, int(body.get("nights", week.nights))))
    week.guests = max(0.0, float(body.get("guests", week.guests)))
    week.risk = body.get("risk", week.risk)
    week.meals = week.meals[:week.nights]
    pantry.log("dials", week=week.date, before=before,
               after={"nights": week.nights, "guests": week.guests, "risk": week.risk})
    refill(week)
    return state()


def post_drop(body):
    week = current_week()
    meal = next((m for m in week.meals if m.slug == body["slug"]), None)
    week.meals = [m for m in week.meals if m.slug != body["slug"]]
    if body["slug"] not in week.declined:
        week.declined.append(body["slug"])
    pantry.write_week(week)
    # The most useful signal in the session: offered, and turned down.
    pantry.log("drop", week=week.date, recipe=body["slug"],
               reason_shown=meal.reason if meal else "",
               reason_kind=review.kind_of(body["slug"]),
               candidate=bool(meal and meal.candidate))
    return state()


def post_lock(body):
    week = current_week()
    for m in week.meals:
        if m.slug == body["slug"]:
            m.locked = bool(body.get("locked"))
    pantry.write_week(week)
    return state()


def post_servings(body):
    """Servings for one meal, rather than for the week.

    `profile.md` asks for this outright: guests are frequent and come on
    particular nights, so a week-level dial buys food for four dinners nobody is
    eating. Both model-planned weeks scaled a single meal on their own the first
    time they ran, unprompted, which is the strongest signal available that the
    dial was in the wrong place.

    Zero clears it and the meal goes back to the week's number, which is why the
    override is `0` rather than `None` - the session sends a number either way.
    """
    week = current_week()
    for m in week.meals:
        if m.slug == body["slug"]:
            m.ae_override = max(0.0, float(body.get("ae") or 0))
    pantry.write_week(week)
    pantry.log("servings", week=week.date, recipe=body["slug"],
               ae=float(body.get("ae") or 0))
    return state()


def post_swap(body):
    """Replace one meal with another, rather than dropping and refilling.

    Two clicks became one, and the difference is not only convenience: a drop
    followed by a refill is two decisions in the log for one intent, which makes
    the accept rate in `review.py` read a swap as a rejection *and* the
    replacement as an unrelated offer. It is one decision, so it is logged as one.

    The dropped meal still goes to `declined`, so gap-filling cannot hand back
    the thing that was just turned down.
    """
    week = current_week()
    meal = next((m for m in week.meals if m.slug == body["slug"]), None)
    if meal is None:
        return state()
    if meal.locked:
        return dict(state(), error=f"{meal.title} is locked.")
    week.meals = [m for m in week.meals if m.slug != body["slug"]]
    if body["slug"] not in week.declined:
        week.declined.append(body["slug"])
    week.meals = pantry.propose(week.nights, week.guests, week.risk,
                                keep=week.meals, avoid=set(week.declined),
                                week=week.date)
    pantry.write_week(week)
    added = [m.slug for m in week.meals if m.slug not in
             {*week.declined, *(x.slug for x in week.meals[:-1])}]
    pantry.log("swap", week=week.date, out=body["slug"],
               reason_kind=review.kind_of(body["slug"]),
               into=added[-1] if added else "")
    return state()


def post_reshuffle(body):
    """Re-roll everything that is not locked.

    What makes the lock mean anything. Until now `refill` kept every meal already
    on the board, so a locked meal and an unlocked one were treated identically
    and the field was decoration. Locking is the household saying *this one is
    settled* - and that is only a statement if something else can move.
    """
    week = current_week()
    locked = [m for m in week.meals if m.locked]
    moved = [m.slug for m in week.meals if not m.locked]
    # **The re-rolled meals are declined, not merely re-proposed.** The ranker is
    # deterministic: hand it the same corpus and the same locked meal and it
    # returns the same four others, so a reshuffle that only re-ran it would be a
    # button that changes nothing. This project has shipped one of those before -
    # the risk dial nudged a score in a fight candidates lose by design - and the
    # lesson recorded then was that a control has to change an outcome.
    #
    # So reshuffle means *not these*, which is also what it reads as. Same
    # semantics as a drop, which is the honest way to spend the corpus: press it
    # enough times in one week and the pool runs out, and that is a true fact
    # about a 24-recipe corpus rather than a bug.
    for slug in moved:
        if slug not in week.declined:
            week.declined.append(slug)
    week.meals = pantry.propose(week.nights, week.guests, week.risk, keep=locked,
                                avoid=set(week.declined), week=week.date)
    pantry.write_week(week)
    pantry.log("reshuffle", week=week.date, kept=[m.slug for m in locked],
               rerolled=moved)
    return state()


def get_recipe(slug: str) -> dict:
    """The recipe, to read without leaving the session.

    Served as its own markdown rather than rendered into something prettier. The
    file is the store, it is what the household edits when the tool is wrong, and
    showing anything else here would put a second version of the truth on screen.
    """
    path = pantry.recipe_file(slug)
    if not path.exists():
        return {"ok": False, "slug": slug,
                "markdown": "_No capture on file for this one yet._"}
    return {"ok": True, "slug": slug, "markdown": path.read_text(encoding="utf-8")}


def post_variant(body):
    week = current_week()
    for m in week.meals:
        if m.slug == body["slug"]:
            m.variant = body.get("variant", "")
    pantry.write_week(week)
    pantry.log("variant", week=week.date, recipe=body["slug"], variant=body.get("variant", ""))
    return state()


def post_fill(body):
    refill(current_week())
    return state()


def post_acquire(body):
    """Go and find a recipe nobody had bookmarked, for a gap in this week.

    **The one route that touches the open web**, and it takes the same posture
    `prep.py` does: degrades, never blocks. A source that cannot be reached is
    skipped, and a run that finds nothing returns a normal session with a line
    saying so rather than an error. In the browser build there are no sockets at
    all, so every source is unreachable and this lands on exactly that path -
    which is the honest outcome there and not a failure worth a stack trace.

    Nothing it finds enters the week. A candidate goes into `candidates.md` and
    has to win a slot from the planner like any other, because membership is
    earned and being newly acquired is not a claim about this household.
    """
    week = current_week()
    notes: list[str] = []
    try:
        import acquire        # deferred, and inside the guard: see the docstring
        found = acquire.acquire(week.meals, want=int(body.get("want", 1)),
                                log=notes.append)
    except Exception as exc:                       # never take the session down
        found = []
        notes.append(f"acquisition failed: {type(exc).__name__}: {exc}")
    out = state()
    out["acquired"] = [{"title": f.rec["title"], "source": f.rec["source"],
                        "reason": f.reason(),
                        "ingredients": len(f.rec["ingredients"]),
                        "questions": f.rec.get("questions", [])} for f in found]
    out["acquire_log"] = notes
    return out


def post_onboard(body):
    """Paste a URL, get a captured recipe in candidates.

    `docs/brief-next.md` §3: adding a recipe meant running a CLI with a URL, and
    for a tool whose whole thesis is closing the gap between the fifteen you
    reach for and the sixty you like, the growth path being a terminal command is
    close to fatal. It also broke the onboardability requirement outright.

    Thin on purpose. `acquire.from_url` does the capture, the constraint check
    and the write, and it is the same function and the same door acquisition
    uses - a recipe somebody pasted and a recipe the tool found are the same
    recipe and get the same row.
    """
    url = (body.get("url") or "").strip()
    if not url.startswith(("http://", "https://")):
        return dict(state(), onboarded=None,
                    onboard_error="That does not look like a link. Paste the "
                                  "address of a recipe page.")
    try:
        import acquire
        found = acquire.from_url(url)
    except Exception as exc:
        reason = str(exc) or type(exc).__name__
        return dict(state(), onboarded=None, onboard_error=reason)
    return dict(state(), onboard_error="", onboarded={
        "title": found.rec["title"], "source": found.rec["source"],
        "ingredients": len(found.rec["ingredients"]),
        "yield": found.rec.get("yield") or "unknown",
        "questions": found.rec.get("questions", []),
    })


def suggest_sides(week, want: int = 2) -> list[str]:
    """Pick sides for the week. Deterministic, and small on purpose.

    Least-recently-served first, then anything whose `Goes with` is already
    represented by a protein on the board, and never two rows with the same
    `Goes with` in one week. That is the whole rule - there is no scoring model
    here because there is nothing to score yet.

    **Returns nothing while `sides.md` is empty**, which is the state today and
    is not a failure. `profile.md` says vegetables are missing from the data
    rather than the diet; the honest response is an empty suggestion and a
    grocery list that keeps saying it is short, not a plausible vegetable.
    """
    rows = [r for r in pantry.load_sides() if r["slug"] not in week.sides]
    if not rows:
        return []
    proteins = {(m.protein or "").lower() for m in week.meals}
    today = pantry.monday()

    def key(row):
        served = (row.get("last served") or "").strip()
        goes = (row.get("goes with") or "").strip().lower()
        return (0 if not served else 1, served,
                0 if (goes and goes in proteins) else 1, row["slug"])

    picked, used = [], set()
    for row in sorted(rows, key=key):
        goes = (row.get("goes with") or "").strip().lower()
        if goes and goes in used:
            continue
        picked.append(row["slug"])
        if goes:
            used.add(goes)
        if len(picked) >= want:
            break
    return picked


def post_side(body):
    """Add or remove one side from the week, or fill it from `sides.md`."""
    week = current_week()
    if body.get("suggest"):
        for slug in suggest_sides(week, int(body.get("want", 2))):
            if slug not in week.sides:
                week.sides.append(slug)
    elif body.get("remove"):
        week.sides = [s for s in week.sides if s != body["remove"]]
    elif body.get("slug"):
        known = {r["slug"] for r in pantry.load_sides()}
        if body["slug"] in known and body["slug"] not in week.sides:
            week.sides.append(body["slug"])
    pantry.write_week(week)
    pantry.log("sides", week=week.date, sides=list(week.sides))
    return state()


def post_add_side(body):
    """Capture a side from a link, or take one by name.

    Two routes because sides arrive two ways and both are real. A link goes
    through the same capture as any recipe, so the ingredients reach the shopping
    list. A name with no link is allowed - *green beans* is a side, everyone knows
    what it is, and refusing it until somebody finds a web page for roasting them
    would be the tool getting in the way of the file it is asking to be filled.
    A named side with no capture carries no ingredients, and the list says so
    rather than pretending.
    """
    url = (body.get("url") or "").strip()
    name = (body.get("name") or "").strip()
    try:
        if url:
            import acquire
            rec = onboard_capture(url)
            pantry.add_side(rec["title"], source=url,
                            active=body.get("active", ""),
                            passive=rec.get("passive") or "",
                            goes_with=body.get("goes_with", ""),
                            season=body.get("season", ""))
            return dict(state(), side_added=rec["title"], side_error="")
        if not name:
            return dict(state(), side_error="Give a link or a name.")
        pantry.add_side(name, goes_with=body.get("goes_with", ""),
                        season=body.get("season", ""), active=body.get("active", ""),
                        notes="typed in")
        return dict(state(), side_added=name, side_error="")
    except Exception as exc:
        return dict(state(), side_error=str(exc) or type(exc).__name__)


def onboard_capture(url: str) -> dict:
    """Capture a page as a recipe file, without the candidate row.

    A side is not a candidate: `candidates.md` exists so an unproven *dinner*
    carries its gamble visibly, and a vegetable does not have one. So this
    borrows the capture and stops short of the door into candidates.
    """
    import onboard
    rec = onboard.from_url(url)
    if rec.get("status") != "complete" or not rec.get("ingredients"):
        raise ValueError("the page carries no machine-readable recipe")
    rec["slug"] = pantry.slug(rec["title"])
    (pantry.ROOT / "recipes").mkdir(parents=True, exist_ok=True)
    pantry.recipe_file(rec["slug"]).write_text(onboard.render_recipe(rec),
                                               encoding="utf-8")
    pantry._FILE_INDEX = None
    return rec


def post_profile(body):
    """Edit `profile.md` from the session.

    `profile.md` opens by saying that correcting the file **is** the trust
    mechanism and that it beats any opaque score. In a hosted deployment that
    sentence was false - the file was unreachable, so the household had less
    control over its own stated preferences than a local user did.

    **The whole file, replaced whole**, the same way a week is written. It is
    markdown a person is meant to read and edit, and offering a form with fields
    would be this code deciding which parts of the household's own description of
    itself are editable.

    Two guards, and neither is about taste. An empty file is refused, because
    saving nothing over the only record of a peanut allergy is not an edit anyone
    means to make. And the result has to still parse as the profile - if the
    Members section stops being findable, the write is rejected and the reason
    said out loud rather than discovered three weeks later by a planner with no
    constraints.
    """
    text = body.get("text") or ""
    if not text.strip():
        return dict(state(), profile_error="Refusing to save an empty profile.")
    before = pantry.PROFILE.read_text(encoding="utf-8") if pantry.PROFILE.exists() else ""
    pantry.PROFILE.write_text(text, encoding="utf-8")
    if before and not pantry.load_members():
        pantry.PROFILE.write_text(before, encoding="utf-8")
        return dict(state(), profile_error=(
            "That would have left no Members section, so nothing was saved. "
            "Keep a `## Members` heading with a name under it."))
    pantry.log("profile_edited", bytes_before=len(before), bytes_after=len(text))
    return dict(state(), profile_error="", profile_saved=True)


def post_feedback(body):
    week = pantry.previous_week(pantry.monday())
    if week is None:
        return state()
    week.feedback[body["slug"]] = {"outcome": body["outcome"], "by": body.get("by", "")}
    pantry.write_week(week)
    return state()


def post_apply(body):
    week = pantry.previous_week(pantry.monday())
    if week is None:
        return {"applied": [], **state()}
    try:
        applied = pantry.apply_feedback(week)
    except pantry.RuleViolation as exc:
        return {"applied": [], "refused": str(exc), **state()}
    week.status = "cooked"
    pantry.write_week(week)
    return {"applied": applied, **state()}


def post_order(body):
    week = current_week()
    week.status = "ordered"
    pantry.write_week(week)
    pantry.log("ordered", week=week.date,
               meals=[{"recipe": m.slug, "variant": m.variant} for m in week.meals],
               ae=week.ae)
    return state()


def post_reset(body):
    reset_demo()
    return state()


ROUTES = {
    "/api/reset": post_reset,
    "/api/dials": post_dials,
    "/api/drop": post_drop,
    "/api/lock": post_lock,
    "/api/variant": post_variant,
    "/api/fill": post_fill,
    "/api/acquire": post_acquire,
    "/api/onboard": post_onboard,
    "/api/servings": post_servings,
    "/api/swap": post_swap,
    "/api/reshuffle": post_reshuffle,
    "/api/profile": post_profile,
    "/api/side": post_side,
    "/api/side/add": post_add_side,
    "/api/feedback": post_feedback,
    "/api/apply": post_apply,
    "/api/order": post_order,
}


def handle(path: str, body: dict | None = None) -> tuple[int, dict]:
    """Route one request. **The only place routing lives.**

    Two front doors call this: the HTTP server below, and the browser build,
    where there is no server at all and the page's `fetch` is shimmed to call
    straight into Python. Keeping them on one function is what stops the hosted
    version drifting from the local one.
    """
    if body is None:
        if path == "/api/state":
            return 200, state()
        if path == "/api/list":
            return 200, grocery_list(current_week())
        if path == "/api/profile":
            return 200, {"text": pantry.PROFILE.read_text(encoding="utf-8")
                                 if pantry.PROFILE.exists() else ""}
        if path.startswith("/api/recipe/"):
            return 200, get_recipe(path[len("/api/recipe/"):])
        return 404, {"error": "not found"}
    fn = ROUTES.get(path)
    if fn is None:
        return 404, {"error": "not found"}
    try:
        return 200, fn(body)
    except Exception as exc:                       # surfaced, never swallowed
        return 500, {"error": f"{type(exc).__name__}: {exc}"}


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *_):
        pass

    def _send(self, code, body, ctype="application/json"):
        data = body if isinstance(body, bytes) else body.encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", ctype)
        self.send_header("Content-Length", str(len(data)))
        self.send_header("Cache-Control", "no-store")
        self.end_headers()
        self.wfile.write(data)

    def do_GET(self):
        if self.path in ("/", "/index.html"):
            return self._send(200, (WEB / "index.html").read_bytes(), "text/html; charset=utf-8")
        code, payload = handle(self.path)
        self._send(code, json.dumps(payload))

    def do_POST(self):
        length = int(self.headers.get("Content-Length") or 0)
        body = json.loads(self.rfile.read(length) or b"{}")
        code, payload = handle(self.path, body)
        self._send(code, json.dumps(payload))


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--port", type=int, default=int(os.environ.get("PORT", 8765)))
    ap.add_argument("--host", default=os.environ.get("HOST", "127.0.0.1"),
                    help="0.0.0.0 to accept connections from outside this machine")
    ap.add_argument("--demo", action="store_true",
                    default=os.environ.get("PANTRY_DEMO") == "1",
                    help="work off a scratch copy so nothing writes to the repo")
    ap.add_argument("--no-open", action="store_true")
    args = ap.parse_args()

    if args.demo:
        start_demo()

    # Refuse the combination that would let a stranger write to the real corpus.
    if args.host != "127.0.0.1" and not args.demo:
        print("Refusing to serve the real corpus on a public interface.\n"
              "  Pass --demo to run against a scratch copy, or keep --host 127.0.0.1.",
              file=sys.stderr)
        return 2

    server = ThreadingHTTPServer((args.host, args.port), Handler)
    url = f"http://{'127.0.0.1' if args.host == '0.0.0.0' else args.host}:{args.port}"
    print(f"Pantry Router — {url}{'  [demo: writes go to a scratch copy]' if args.demo else ''}")
    print(f"  corpus {len(pantry.load_corpus())} · candidates {len(pantry.load_candidates())} "
          f"· week {pantry.monday()}")
    if not args.no_open and args.host == "127.0.0.1":
        threading.Timer(0.4, lambda: webbrowser.open(url)).start()
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nbye")


if __name__ == "__main__":
    sys.exit(main() or 0)
