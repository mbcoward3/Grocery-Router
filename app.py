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
import shop

ROOT = Path(__file__).resolve().parent
WEB = ROOT / "web"
DEMO: Path | None = None


# --------------------------------------------------------------------------- #
# Demo mode
# --------------------------------------------------------------------------- #

DEMO_FILES = ("corpus.md", "candidates.md", "profile.md", "items.md")
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
        week.meals = pantry.propose(week.nights, week.guests, week.risk)
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
        "counts": {"corpus": len(corpus), "candidates": len(pantry.load_candidates())},
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
    return {
        "corpus": len(corpus),
        "cooked_ever": len(cooked),
        "dormant": len(corpus) - len(cooked),
        "surfaced_ever": len(surfaced),
        "kept": sum(1 for d in applied if d.get("outcome", "").startswith("kept")),
        "accept_rate": None if not offered else round(100 * (offered - drops) / offered),
        "weeks": len(pantry.list_weeks()),
    }


def refill(week: pantry.Week) -> pantry.Week:
    """Fill up to `nights`, leaving everything already on the board alone.

    This is the gap-filling the session is built around: dropping one meal and
    asking for another must not re-roll the four the household already accepted.
    """
    week.meals = pantry.propose(week.nights, week.guests, week.risk, keep=week.meals,
                                avoid=set(week.declined))
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
    specs = [m.file + (f":{m.variant}" if m.variant else "")
             for m in week.meals if m.has_file]
    if not specs:
        return {"ok": True, "markdown": "_Nothing planned yet._"}

    shop.configure(pantry.ROOT)
    ae = pantry.BASE_AE + week.guests
    try:
        built = shop.build(specs, ae)
    except (FileNotFoundError, SystemExit) as exc:
        return {"ok": False, "markdown": f"```\n{exc}\n```"}
    meals, lines, unknown, merges, links, scales, items = built
    out = shop.emit(meals, lines, unknown, merges, links, ae, scales, items)
    out += "\n" + shop.coupling_report(lines, items)
    if missing:
        # Surfaced, never silent. A list that quietly omits a meal is the failure
        # this whole pipeline is built to avoid.
        out += ("\n\n## Not on this list\n\n*No ingredient file exists for these yet, so "
                "nothing was added for them:*\n\n"
                + "\n".join(f"- {t}" for t in missing) + "\n")
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
               reason_shown=meal.reason if meal else "", candidate=bool(meal and meal.candidate))
    return state()


def post_lock(body):
    week = current_week()
    for m in week.meals:
        if m.slug == body["slug"]:
            m.locked = bool(body.get("locked"))
    pantry.write_week(week)
    return state()


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
