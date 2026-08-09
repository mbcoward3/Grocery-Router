"""The Grocery Router web interface.

The planner chooses meals, then ``gr.shoplist`` builds every displayed line without
consulting a model. Local development may persist generated state in markdown; production
uses the configured database store.
"""

from __future__ import annotations

import argparse
import html
import json
import mimetypes
import socket
import threading
from collections import Counter
from datetime import date
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import parse_qs, quote, unquote, urlparse

from . import repo as R
from . import session as SE
from . import storage as ST
from . import weekfile as W
from .notices import Notice

DEFAULT_HOST = "0.0.0.0"
DEFAULT_PORT = 8765
_WRITE_LOCK = threading.Lock()


def _e(value: object) -> str:
    return html.escape(str(value), quote=True)


def _human(value: str) -> str:
    return value.replace("_", " ")


def _page(title: str, body: str, page_class: str = "") -> str:
    return f"""<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
  <meta name="theme-color" content="#FBFAF7">
  <title>{_e(title)} · Grocery Router</title>
  <link rel="stylesheet" href="/static/styles.css">
  <script src="/static/app.js" defer></script>
</head>
<body class="{_e(page_class)}">
  {body}
</body>
</html>"""


def _header(week: SE.Week | None, list_view: bool = False) -> str:
    if week:
        title = f"Week of {week.sunday.strftime('%B %-d')}"
        sub = f"{len(week.meals)} meals · {week.target_ae:g} adult-equivalents"
    else:
        title = "Plan this week"
        sub = "A meal pool first. The list is your recipes, added up by code."
    nav = (
        '<a class="button secondary small" href="/">Back to planning</a>'
        if list_view else
        ('<a class="button secondary small" href="/list">Open phone list</a>' if week else "")
    )
    return f"""
<header class="site-header">
  <div>
    <div class="eyebrow">Grocery Router</div>
    <h1>{_e(title)}</h1>
    <p>{_e(sub)}</p>
  </div>
  {nav}
</header>"""


def _controls(week: SE.Week | None) -> str:
    nights = week.nights if week else 5
    guests = week.guests if week else 0
    label = "Regenerate the week" if week else "Plan the week"
    note = (
        "This asks the planner for a new pool. It usually takes about a minute. "
        "The model sees the catalogue and profile, never a recipe ingredient."
    )
    return f"""
<section class="section" id="planning">
  <div class="section-heading"><span>1</span><h2>Set the week</h2></div>
  <p class="section-intro">{_e(note)}</p>
  <form class="card controls" method="post" action="/plan" data-wait-form>
    <label><span>Nights</span><input name="nights" type="number" min="1" max="7" value="{nights}" inputmode="numeric"></label>
    <label><span>Guests</span><input name="guests" type="number" min="0" max="12" value="{guests}" inputmode="numeric"></label>
    <button class="button primary" type="submit" data-wait-label="Planning…">{_e(label)}</button>
    <span class="form-status" aria-live="polite"></span>
  </form>
</section>"""


def _notices(week: SE.Week | None, repo: R.Repo) -> str:
    if week:
        notices = week.notices
    else:
        # The durable gaps still apply before the first pool exists.
        notices = [
            Notice("sides",
                   "sides: none recorded — this list is short by design, not by accident.",
                   "The tool will not invent a side."),
            Notice("recency",
                   "no last-cooked dates exist, so nothing is ranked by recency.",
                   "Every recipe is unranked rather than overdue."),
            Notice("effort",
                   "effort ratings are the system's guess — correct any that are wrong in corpus.md.",
                   "None came from the household."),
            Notice("corpus",
                   "cooked anything this week that isn't in corpus.md? Add it — that's the product.",
                   f"The {len(repo.corpus)} recorded recipes are a floor, not a census."),
        ]
    rows = "".join(
        f'<li><strong>{_e(n.text)}</strong><span>{_e(n.detail)}</span></li>' for n in notices
    )
    return f"""
<section class="callout" aria-labelledby="before-title">
  <h2 id="before-title">Before you start</h2>
  <ul>{rows}</ul>
</section>"""


def _pool_counts(repo: R.Repo, week: SE.Week) -> str:
    rows = [repo.row(m.slug) for m in week.meals]
    rows = [r for r in rows if r]
    fit = sum(1 for row in rows if row.active.lower() in {"low", "med"})
    proteins = Counter(row.protein for row in rows)
    protein_text = " · ".join(f"{name} {count}" for name, count in sorted(proteins.items()))
    proven = sum(1 for meal in week.meals if not meal.untried)
    return f"""
<div class="pool-counts" aria-label="Plain counts for this meal pool">
  <div><strong>{fit} of {len(rows)}</strong><span>low or medium active</span></div>
  <div><strong>{len(proteins)}</strong><span>proteins · {_e(protein_text)}</span></div>
  <div><strong>{proven} of {len(week.meals)}</strong><span>cooked and liked here</span></div>
</div>
<p class="count-caveat">Counts describe this pool. They do not infer a household pattern.</p>"""


def _meal_card(repo: R.Repo, meal) -> str:
    row = repo.row(meal.slug)
    active = row.active if row else "unknown"
    passive = row.passive if row else "unknown"
    tags = [meal.yield_raw or "yield unknown", f"{active} active"]
    if passive and passive != "—":
        tags.append(passive)
    if meal.untried:
        tags.insert(0, "candidate")
    if meal.scale and not meal.scale.scaled:
        tags.append("not scaled")
    tags_html = "".join(f'<span class="tag">{_e(tag)}</span>' for tag in tags)
    warning = ""
    if meal.scale and not meal.scale.scaled:
        warning = f'<p class="meal-warning">{_e(meal.scale.note)}</p>'
    return f"""
<article class="card meal-card">
  <div class="meal-main">
    <div>
      <h3>{_e(meal.label)}</h3>
      <p class="reason"><span>{_e(meal.reason_kind)}</span>{_e(meal.reason)}</p>
    </div>
    <div class="meal-actions">
      <a href="/recipe/{quote(meal.slug)}">Recipe</a>
      <form method="post" action="/swap">
        <input type="hidden" name="slug" value="{_e(meal.slug)}">
        <button type="submit">Swap</button>
      </form>
    </div>
  </div>
  <div class="tags">{tags_html}</div>
  {warning}
</article>"""


def _pool(repo: R.Repo, week: SE.Week) -> str:
    meals = "".join(_meal_card(repo, meal) for meal in week.meals)
    planner = (
        f"Chosen by {_e(week.planner_source)}. One model call selected the meals; code checked every pick."
        if week.planner_source != "code" else
        "Chosen by code because the planner did not answer. This is a plain catalogue sweep, not a considered selection."
    )
    error = f'<div class="error"><strong>Planner error:</strong> {_e(week.planner_error)}</div>' if week.planner_error else ""
    notes = "".join(f"<li>{_e(note)}</li>" for note in week.planner_notes)
    changed = f'<div class="planner-notes"><strong>Code changed the answer</strong><ul>{notes}</ul></div>' if notes else ""
    return f"""
<section class="section">
  <div class="section-heading"><span>2</span><h2>This week's pool</h2></div>
  {_pool_counts(repo, week)}
  {error}{changed}
  <div class="meal-pool">{meals}</div>
  <div class="side-card"><strong>Sides</strong><p>Nothing written down yet — so every list is short on vegetables and starches, and says so.</p></div>
  <p class="planner-source">{planner}</p>
</section>"""


def _list_summary(week: SE.Week) -> str:
    count = len(week.shopping.buy) + len(week.shopping.staples) + len(week.shopping.unknown)
    unknown = len(week.shopping.unknown)
    return f"""
<section class="section">
  <div class="section-heading"><span>3</span><h2>The list</h2></div>
  <p class="section-intro">Deterministic. No model touches this — it is your recipes, scaled and added up.</p>
  <div class="card list-summary">
    <div><strong>{count}</strong><span>lines to check</span></div>
    <div><strong>{unknown}</strong><span>printed verbatim because code refused them</span></div>
    <a class="button primary" href="/list">Open the phone list</a>
  </div>
</section>"""


def render_plan(root: Path, store: ST.Store | None = None) -> str:
    repo = R.load(root)
    backend = store or ST.from_environment(root)
    week = SE.load_existing(root, store=backend)
    content = _header(week) + '<main class="page">' + _notices(week, repo) + _controls(week)
    if week:
        content += _pool(repo, week) + _list_summary(week)
    content += """
<section class="trust-note">
  <div class="eyebrow">The profile</div>
  <p>Catalogue and household inputs stay reviewable markdown. Generated plans and list ticks use the configured durable store.</p>
</section>
</main>"""
    return _page("Planning", content, "plan-page")


def _checkbox_row(key: str, checked: bool, title: str, quantity: str,
                  sources: list[str], flags: list[str] | None = None) -> str:
    source_text = ", ".join(sources) if sources else "source recorded in the week file"
    flags_html = "".join(f'<span class="line-flag">{_e(flag)}</span>' for flag in (flags or []))
    return f"""
<label class="list-row{' is-checked' if checked else ''}">
  <input type="checkbox" data-list-key="{_e(key)}"{' checked' if checked else ''}>
  <span class="check-box" aria-hidden="true"></span>
  <span class="list-copy">
    <span class="list-name"><b>{_e(quantity)}</b>{' ' if quantity else ''}{_e(title)}</span>
    <span class="provenance">for {_e(source_text)}</span>
    {flags_html}
  </span>
</label>"""


def render_list(root: Path, store: ST.Store | None = None) -> str:
    backend = store or ST.from_environment(root)
    week = SE.load_existing(root, store=backend)
    if week is None:
        body = _header(None, list_view=True) + '<main class="phone-page"><div class="callout"><h2>No list yet</h2><p>Plan the week on the laptop first.</p><a class="button primary" href="/">Plan the week</a></div></main>'
        return _page("Shopping list", body, "list-page")

    ticks = backend.read_ticks(week.sunday)
    sections = []
    for aisle, lines in week.shopping.by_aisle().items():
        rows = []
        for line in lines:
            key = W.line_key("buy", line.item)
            flags = list(line.flags)
            if line.stranded:
                flags.append("only one meal needs this")
            rows.append(_checkbox_row(key, key in ticks, _human(line.item),
                                      line.quantity_text(), line.sources, flags))
        sections.append(f'<section class="aisle"><h2>{_e(aisle)}</h2>{"".join(rows)}</section>')

    staple_rows = []
    for line in week.shopping.staples:
        key = W.line_key("staple", line.item)
        staple_rows.append(_checkbox_row(key, key in ticks, _human(line.item),
                                         line.quantity_text(), line.sources, line.flags))
    if staple_rows:
        sections.append('<section class="aisle staples"><h2>Probably have — check first</h2>' + "".join(staple_rows) + "</section>")

    unknown_rows = []
    for line in week.shopping.unknown:
        key = W.line_key("unknown", f"{line.meal_slug}-{line.raw}")
        unknown_rows.append(_checkbox_row(key, key in ticks, line.raw, "", [line.meal_title],
                                         [f"unresolved: {line.reason}"]))
    if unknown_rows:
        sections.append('<section class="aisle unknown"><h2>Unknown — printed, never dropped</h2>' + "".join(unknown_rows) + "</section>")

    total = len(week.shopping.buy) + len(week.shopping.staples) + len(week.shopping.unknown)
    done = len(ticks)
    sides_notice = next((n.text for n in week.notices if n.key == "sides"), "")
    body = _header(week, list_view=True) + f"""
<main class="phone-page">
  <div class="list-status" data-list-status data-total="{total}">
    <strong><span data-done>{done}</span> of {total}</strong><span>checked</span>
  </div>
  <div class="callout compact"><strong>{_e(sides_notice)}</strong></div>
  <p class="list-rule">Every quantity below comes from recipe files and deterministic code. Tap a row to check it off; ticks are saved in <code>{_e(week.state_ref)}</code>.</p>
  {''.join(sections)}
  <div class="list-finish">
    <strong>Cooked anything this week that isn't in corpus.md?</strong>
    <span>Add it — that's the product.</span>
  </div>
</main>"""
    return _page("Shopping list", body, "list-page")


def render_recipe(root: Path, slug: str) -> str | None:
    repo = R.load(root)
    recipe = repo.recipes.get(slug)
    if recipe is None:
        return None
    text = recipe.path.read_text(encoding="utf-8")
    body = f"""
<header class="site-header">
  <div><div class="eyebrow">Recipe file</div><h1>{_e(recipe.title)}</h1><p>This is the markdown the shopping-list code reads.</p></div>
  <a class="button secondary small" href="/">Back to planning</a>
</header>
<main class="page"><pre class="card recipe-text">{_e(text)}</pre></main>"""
    return _page(recipe.title, body, "recipe-page")


def _local_ip() -> str:
    """Best-effort LAN address, without sending application data anywhere."""
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    try:
        sock.connect(("192.0.2.1", 80))
        return sock.getsockname()[0]
    except OSError:
        try:
            return socket.gethostbyname(socket.gethostname())
        except OSError:
            return "127.0.0.1"
    finally:
        sock.close()


def make_handler(root: Path, store: ST.Store | None = None):
    root = root.resolve()
    backend = store or ST.from_environment(root)
    static_root = (root / "static").resolve()

    class Handler(BaseHTTPRequestHandler):
        server_version = "GroceryRouter/1"

        def log_message(self, fmt, *args):
            print(f"{self.address_string()} - {fmt % args}")

        def _send(self, content: bytes | str, status=HTTPStatus.OK,
                  content_type="text/html; charset=utf-8", headers=None):
            if isinstance(content, str):
                content = content.encode("utf-8")
            self.send_response(status)
            self.send_header("Content-Type", content_type)
            self.send_header("Content-Length", str(len(content)))
            self.send_header("X-Content-Type-Options", "nosniff")
            for name, value in (headers or {}).items():
                self.send_header(name, value)
            self.end_headers()
            self.wfile.write(content)

        def _redirect(self, path):
            self._send(b"", HTTPStatus.SEE_OTHER, headers={"Location": path})

        def _form(self) -> dict[str, str]:
            length = min(int(self.headers.get("Content-Length", "0") or 0), 16_384)
            values = parse_qs(self.rfile.read(length).decode("utf-8", "replace"))
            return {key: items[-1] for key, items in values.items() if items}

        @staticmethod
        def _number(value: str | None, default: int, low: int, high: int) -> int:
            try:
                return max(low, min(high, int(value or default)))
            except ValueError:
                return default

        def do_GET(self):
            parsed = urlparse(self.path)
            path = parsed.path
            if path == "/":
                self._send(render_plan(root, backend))
                return
            if path == "/list":
                self._send(render_list(root, backend))
                return
            if path == "/health/live":
                self._send("ok\n", content_type="text/plain; charset=utf-8")
                return
            if path in {"/health", "/health/ready"}:
                try:
                    backend.ping()
                except Exception as exc:
                    print(f"readiness check failed: {type(exc).__name__}")
                    self._send("not ready\n", HTTPStatus.SERVICE_UNAVAILABLE,
                               "text/plain; charset=utf-8")
                    return
                self._send("ok\n", content_type="text/plain; charset=utf-8")
                return
            if path.startswith("/recipe/"):
                page = render_recipe(root, unquote(path[len("/recipe/"):]))
                self._send(page or "Not found", HTTPStatus.OK if page else HTTPStatus.NOT_FOUND)
                return
            if path.startswith("/static/"):
                relative = unquote(path[len("/static/"):])
                candidate = (static_root / relative).resolve()
                if static_root not in candidate.parents or not candidate.is_file():
                    self._send("Not found", HTTPStatus.NOT_FOUND, "text/plain; charset=utf-8")
                    return
                mime = mimetypes.guess_type(candidate.name)[0] or "application/octet-stream"
                self._send(candidate.read_bytes(), content_type=mime,
                           headers={"Cache-Control": "public, max-age=3600"})
                return
            self._send("Not found", HTTPStatus.NOT_FOUND, "text/plain; charset=utf-8")

        def do_POST(self):
            path = urlparse(self.path).path
            if path == "/plan":
                form = self._form()
                nights = self._number(form.get("nights"), 5, 1, 7)
                guests = self._number(form.get("guests"), 0, 0, 12)
                with _WRITE_LOCK:
                    SE.plan_week(root, nights=nights, guests=guests, store=backend)
                self._redirect("/")
                return
            if path == "/swap":
                form = self._form()
                with _WRITE_LOCK:
                    week = SE.load_existing(root, store=backend)
                    if week:
                        SE.swap(R.load(root), week, form.get("slug", ""),
                                form.get("replacement") or None)
                self._redirect("/")
                return
            if path == "/api/toggle":
                form = self._form()
                key = form.get("key", "")
                with _WRITE_LOCK:
                    week = SE.load_existing(root, store=backend)
                    if not week or not key or len(key) > 300:
                        self._send(json.dumps({"error": "unknown list row"}),
                                   HTTPStatus.BAD_REQUEST, "application/json")
                        return
                    state = backend.toggle_tick(week.sunday, key)
                self._send(json.dumps({"checked": state}), content_type="application/json")
                return
            self._send("Not found", HTTPStatus.NOT_FOUND, "text/plain; charset=utf-8")

    return Handler


def serve(root: Path | str = ".", host: str = DEFAULT_HOST, port: int = DEFAULT_PORT):
    root = Path(root).resolve()
    store = ST.from_environment(root)
    server = ThreadingHTTPServer((host, port), make_handler(root, store))
    actual_port = server.server_address[1]
    lan = _local_ip()
    print("Grocery Router is ready.", flush=True)
    print(f"Laptop: http://127.0.0.1:{actual_port}/", flush=True)
    print(f"iPhone: http://{lan}:{actual_port}/list", flush=True)
    print("Keep this laptop awake and on the same network while shopping.", flush=True)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nStopped.")
    finally:
        server.server_close()


def main(argv: list[str] | None = None):
    parser = argparse.ArgumentParser(description="Grocery Router local web app")
    parser.add_argument("--host", default=DEFAULT_HOST,
                        help="address to bind (default: 0.0.0.0, reachable on the LAN)")
    parser.add_argument("--port", type=int, default=DEFAULT_PORT)
    parser.add_argument("--root", type=Path, default=Path("."))
    args = parser.parse_args(argv)
    serve(args.root, args.host, args.port)


if __name__ == "__main__":
    main()
