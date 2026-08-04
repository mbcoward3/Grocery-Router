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
        "previous": (prev.to_json() | {"applied": bool(prev.feedback)}) if prev else None,
        "briefing": pantry.briefing(),
        "counts": {"corpus": len(corpus), "candidates": len(pantry.load_candidates())},
        "planner": "model" if pantry.__dict__.get("_model") else "ranker",
    }


def refill(week: pantry.Week) -> pantry.Week:
    """Fill up to `nights`, leaving everything already on the board alone.

    This is the gap-filling the session is built around: dropping one meal and
    asking for another must not re-roll the four the household already accepted.
    """
    week.meals = pantry.propose(week.nights, week.guests, week.risk, keep=week.meals)
    pantry.write_week(week)
    return week


def grocery_list(week: pantry.Week) -> dict:
    specs = [m.slug + (f":{m.variant}" if m.variant else "") for m in week.meals]
    if not specs:
        return {"ok": True, "markdown": "_Nothing planned yet._"}
    cmd = [sys.executable, str(ROOT / "shop.py"), "--week", ",".join(specs),
           "--ae", str(pantry.BASE_AE), "--guests", str(week.guests)]
    proc = subprocess.run(cmd, capture_output=True, text=True)
    if proc.returncode != 0:
        return {"ok": False, "markdown": f"```\n{proc.stderr.strip()}\n```"}
    return {"ok": True, "markdown": proc.stdout}


# --------------------------------------------------------------------------- #
# Routes
# --------------------------------------------------------------------------- #

def post_dials(body):
    week = current_week()
    week.nights = max(1, min(14, int(body.get("nights", week.nights))))
    week.guests = max(0.0, float(body.get("guests", week.guests)))
    week.risk = body.get("risk", week.risk)
    week.meals = week.meals[:week.nights]
    return refill(week) and state()


def post_drop(body):
    week = current_week()
    week.meals = [m for m in week.meals if m.slug != body["slug"]]
    pantry.write_week(week)
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
    return state()


def post_fill(body):
    return refill(current_week()) and state()


def post_feedback(body):
    week = pantry.previous_week(pantry.monday())
    if week is None:
        return state()
    week.feedback[body["slug"]] = body["outcome"]
    pantry.write_week(week)
    return state()


def post_apply(body):
    week = pantry.previous_week(pantry.monday())
    if week is None:
        return {"applied": [], **state()}
    applied = pantry.apply_feedback(week)
    week.status = "cooked"
    pantry.write_week(week)
    return {"applied": applied, **state()}


def post_order(body):
    week = current_week()
    week.status = "ordered"
    pantry.write_week(week)
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
