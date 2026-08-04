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
import subprocess
import sys
import threading
import webbrowser
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

import pantry

ROOT = Path(__file__).resolve().parent
WEB = ROOT / "web"


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
        "planner": "ranker",
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
    missing = [m.title for m in week.meals if not m.has_file]
    specs = [m.file + (f":{m.variant}" if m.variant else "")
             for m in week.meals if m.has_file]
    if not specs:
        return {"ok": True, "markdown": "_Nothing planned yet._"}
    cmd = [sys.executable, str(ROOT / "shop.py"), "--week", ",".join(specs),
           "--ae", str(pantry.BASE_AE), "--guests", str(week.guests)]
    proc = subprocess.run(cmd, capture_output=True, text=True)
    if proc.returncode != 0:
        return {"ok": False, "markdown": f"```\n{proc.stderr.strip()}\n```"}
    out = proc.stdout
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


ROUTES = {
    "/api/dials": post_dials,
    "/api/drop": post_drop,
    "/api/lock": post_lock,
    "/api/variant": post_variant,
    "/api/fill": post_fill,
    "/api/feedback": post_feedback,
    "/api/apply": post_apply,
    "/api/order": post_order,
}


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
        if self.path == "/api/state":
            return self._send(200, json.dumps(state()))
        if self.path == "/api/list":
            return self._send(200, json.dumps(grocery_list(current_week())))
        self._send(404, json.dumps({"error": "not found"}))

    def do_POST(self):
        fn = ROUTES.get(self.path)
        if fn is None:
            return self._send(404, json.dumps({"error": "not found"}))
        length = int(self.headers.get("Content-Length") or 0)
        body = json.loads(self.rfile.read(length) or b"{}")
        try:
            return self._send(200, json.dumps(fn(body)))
        except Exception as exc:                       # surfaced, never swallowed
            return self._send(500, json.dumps({"error": f"{type(exc).__name__}: {exc}"}))


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--port", type=int, default=8765)
    ap.add_argument("--no-open", action="store_true")
    args = ap.parse_args()

    server = ThreadingHTTPServer(("127.0.0.1", args.port), Handler)
    url = f"http://127.0.0.1:{args.port}"
    print(f"Pantry Router — {url}")
    print(f"  corpus {len(pantry.load_corpus())} · candidates {len(pantry.load_candidates())} "
          f"· week {pantry.monday()}")
    if not args.no_open:
        threading.Timer(0.4, lambda: webbrowser.open(url)).start()
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nbye")


if __name__ == "__main__":
    main()
