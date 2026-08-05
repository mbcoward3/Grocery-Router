#!/usr/bin/env python3
"""Drive the static build in a real browser and fail loudly if it is broken.

    python3 build_static.py && python3 smoke_static.py

**"It built" is not "it runs."** The static build boots Pyodide from a CDN and
shims `fetch` so the page's API calls land in `app.handle()` instead of on a
socket. Either half can break without the build failing, and the failure looks
like a blank page on a public URL rather than a red build.

So this checks the things that would actually be wrong: Python started, a week
came back, the reasons are not five copies of one sentence, the previous week is
there to give feedback on, and the grocery list - the deterministic half, run
under WebAssembly - still produces items.

Set PW_CHROMIUM to use a browser that is already on disk.
"""

import http.server
import os
import socketserver
import sys
import threading
from pathlib import Path

ROOT = Path(__file__).resolve().parent
DIST = ROOT / "dist"
PORT = int(os.environ.get("SMOKE_PORT", "8901"))
BOOT_TIMEOUT_MS = int(os.environ.get("SMOKE_TIMEOUT_MS", "240000"))


def serve():
    handler = lambda *a, **k: http.server.SimpleHTTPRequestHandler(
        *a, directory=str(DIST), **k)
    socketserver.TCPServer.allow_reuse_address = True
    httpd = socketserver.TCPServer(("127.0.0.1", PORT), handler)
    threading.Thread(target=httpd.serve_forever, daemon=True).start()
    return httpd


def main():
    if not (DIST / "index.html").exists():
        print("dist/index.html is missing — run build_static.py first", file=sys.stderr)
        return 1

    from playwright.sync_api import sync_playwright

    httpd = serve()
    problems, logs = [], []
    try:
        with sync_playwright() as p:
            launch = {"args": ["--no-sandbox"]}
            if os.environ.get("PW_CHROMIUM"):
                launch["executable_path"] = os.environ["PW_CHROMIUM"]
            # Pyodide comes off a CDN. A sandboxed build environment may only
            # reach it through a proxy, and Chromium does not read the shell's
            # proxy variables the way curl does.
            proxy = os.environ.get("HTTPS_PROXY") or os.environ.get("https_proxy")
            if proxy:
                # Bypass is not optional: without it Chromium sends the request
                # for the local page through the proxy too, which answers 405,
                # so nothing loads and every later failure is a red herring.
                launch["proxy"] = {"server": proxy, "bypass": "127.0.0.1,localhost"}
            browser = p.chromium.launch(**launch)
            page = browser.new_page()
            page.on("console", lambda m: logs.append(f"[{m.type}] {m.text}"))
            page.on("pageerror", lambda e: logs.append(f"[pageerror] {e}"))
            page.goto(f"http://127.0.0.1:{PORT}/", wait_until="load")

            try:
                page.wait_for_selector("#meals .meal", timeout=BOOT_TIMEOUT_MS)
            except Exception:
                print("Python never started, or no week came back.", file=sys.stderr)
                print("\n".join(logs[-25:]), file=sys.stderr)
                page.screenshot(path="/tmp/smoke-fail.png", full_page=True)
                return 1

            titles = page.eval_on_selector_all("#meals .title", "e => e.map(x => x.textContent)")
            reasons = page.eval_on_selector_all("#meals .why", "e => e.map(x => x.textContent)")
            previous = page.eval_on_selector_all("#fbList .name", "e => e.map(x => x.textContent.trim())")

            print("This week:")
            for t, r in zip(titles, reasons):
                print(f"  {t:34} {r}")
            print(f"Last week: {', '.join(previous) or '(none)'}")

            if len(titles) < 3:
                problems.append(f"only {len(titles)} meals proposed")
            if len(set(reasons)) != len(reasons):
                problems.append("two meals share a reason — the reason is the product")
            if not previous:
                problems.append("no previous week, so the feedback loop cannot be shown")

            page.click("#listBtn")
            try:
                page.wait_for_selector(".listout h2", timeout=BOOT_TIMEOUT_MS)
            except Exception:
                problems.append("the grocery list never rendered")
            else:
                items = page.eval_on_selector_all(".listout li", "e => e.map(x => x.textContent)")
                print(f"Grocery list: {len(items)} lines")
                for line in items[:5]:
                    print(f"  {line.strip()}")
                if len(items) < 8:
                    problems.append(f"the list came back with only {len(items)} lines")

            page.screenshot(path="/tmp/smoke.png", full_page=True)
            browser.close()
    finally:
        httpd.shutdown()

    errors = [l for l in logs if "pageerror" in l or "[error]" in l]
    if errors:
        problems.append(f"{len(errors)} console error(s): {errors[0][:120]}")

    if problems:
        print("\nFAILED:", file=sys.stderr)
        for p_ in problems:
            print(f"  - {p_}", file=sys.stderr)
        return 1
    print("\nok — the real Python ran in the browser")
    return 0


if __name__ == "__main__":
    sys.exit(main())
