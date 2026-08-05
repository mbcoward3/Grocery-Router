#!/usr/bin/env python3
"""Build the static, serverless version of the session into `dist/`.

    ./build_static.py

The only free way to host this. It is **not** a rewrite: Pyodide is CPython
compiled to WebAssembly, so `pantry.py`, `shop.py` and `app.py` run unmodified
in the browser, and `web/index.html` is copied byte for byte. The page still
calls `fetch('/api/state')`; a shim in the loader routes that into
`app.handle()` instead of over a socket.

That matters more here than it might elsewhere. A JavaScript port would be a
second copy of the business logic to keep honest against the first, and this
project has already measured what that costs - two ingredient parsers, written
weeks apart from the same spec, disagreed on what the item *was* in three of
twelve hard cases.

What the browser build gives up: nothing persists. Reloading the page is Reset.
That is correct for a demo and wrong for a household, which is why the real one
runs locally against files in git.
"""

import json
import shutil
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent
DIST = ROOT / "dist"

PYODIDE_VERSION = "0.28.3"
PYODIDE_CDN = f"https://cdn.jsdelivr.net/pyodide/v{PYODIDE_VERSION}/full/"
VENDOR = ROOT / "vendor" / "pyodide"

# The runtime is vendored rather than loaded from a CDN. Three reasons, and only
# the first is about convenience: the Space becomes self-contained, the version
# cannot drift under us, and a sandboxed CI can test the real page with no
# network at all. 12MB, fetched once by `--fetch`.
PYODIDE_FILES = ["pyodide.js", "pyodide.mjs", "pyodide.asm.js", "pyodide.asm.wasm",
                 "python_stdlib.zip", "pyodide-lock.json"]

# Everything the app reads at runtime. The demo copies win where they exist -
# see app._demo_source. Recipes and items are public recipe data and shared.
CODE = ["pantry.py", "shop.py", "app.py", "prep.py"]
DATA = ["corpus.md", "candidates.md", "profile.md", "items.md"]


def collect() -> dict:
    """Read every file the app needs into one JSON blob the loader unpacks into
    Pyodide's in-memory filesystem."""
    files = {}
    for name in CODE:
        files[name] = (ROOT / name).read_text(encoding="utf-8")
    for name in DATA:
        demo = ROOT / "demo" / name
        files[name] = (demo if demo.exists() else ROOT / name).read_text(encoding="utf-8")
    for path in sorted((ROOT / "recipes").glob("*.md")):
        files[f"recipes/{path.name}"] = path.read_text(encoding="utf-8")
    return files


LOADER = """
// Route the page's API calls into Python. index.html is unchanged and does not
// know it is not talking to a server.
const REAL_FETCH = window.fetch.bind(window);
let PY = null;

const boot = (async () => {
  const status = document.createElement('div');
  status.className = 'booting';
  status.textContent = 'Starting Python…';
  document.body.appendChild(status);

  const pyodide = await loadPyodide({ indexURL: %(pyodide)s });
  const files = JSON.parse(document.getElementById('payload').textContent);

  pyodide.FS.mkdirTree('/app/recipes');
  for (const [name, text] of Object.entries(files)) {
    pyodide.FS.writeFile('/app/' + name, text);
  }

  await pyodide.runPythonAsync(`
import sys, json
sys.path.insert(0, "/app")
import pantry, shop, app
# There is no repo to protect in a browser tab, but demo mode is also what
# seeds a previous week to give feedback on, which is the part of the session
# that cannot be shown any other way.
app.ROOT = __import__("pathlib").Path("/app")
app.DEMO_SRC = app.ROOT / "demo"
app.start_demo()
`);
  PY = pyodide;
  status.remove();
  return pyodide;
})();

window.fetch = async (url, opts) => {
  const path = String(url);
  if (!path.startsWith('/api/')) return REAL_FETCH(url, opts);
  await boot;
  const body = opts && opts.body ? opts.body : 'null';
  PY.globals.set('_path', path);
  PY.globals.set('_body', body);
  const out = PY.runPython(
    'json.dumps(app.handle(_path, json.loads(_body))[1])'
  );
  return new Response(out, { headers: { 'Content-Type': 'application/json' } });
};
"""


def fetch_runtime():
    """Download the Pyodide runtime into vendor/. Run once; it is committed."""
    import urllib.request
    VENDOR.mkdir(parents=True, exist_ok=True)
    for name in PYODIDE_FILES:
        dest = VENDOR / name
        if dest.exists():
            continue
        print(f"  {name} …", end="", flush=True)
        with urllib.request.urlopen(PYODIDE_CDN + name, timeout=180) as r:
            dest.write_bytes(r.read())
        print(f" {dest.stat().st_size/1024:.0f} KB")


def main():
    if "--fetch" in sys.argv:
        fetch_runtime()
        return 0

    missing = [n for n in PYODIDE_FILES if not (VENDOR / n).exists()]
    if missing:
        print(f"vendor/pyodide is missing {', '.join(missing)}.\n"
              f"Run: ./build_static.py --fetch", file=sys.stderr)
        return 1

    if DIST.exists():
        shutil.rmtree(DIST)
    DIST.mkdir()
    shutil.copytree(VENDOR, DIST / "pyodide")

    page = (ROOT / "web" / "index.html").read_text(encoding="utf-8")
    files = collect()

    inject = (
        f'<script src="pyodide/pyodide.js"></script>\n'
        f'<script type="application/json" id="payload">'
        f'{json.dumps(files)}</script>\n'
        f'<style>.booting{{position:fixed;inset:0;display:grid;place-items:center;'
        f'font:500 15px ui-sans-serif,system-ui,sans-serif;color:var(--muted);'
        f'background:var(--bg);z-index:9}}</style>\n'
        f'<script>{LOADER % {"pyodide": json.dumps("pyodide/")}}</script>\n'
    )

    marker = "<script>"
    at = page.index(marker)
    page = page[:at] + inject + page[at:]
    (DIST / "index.html").write_text(page, encoding="utf-8")

    size = len((DIST / "index.html").read_bytes())
    total = sum(f.stat().st_size for f in DIST.rglob("*") if f.is_file())
    print(f"dist/index.html  {size/1024:.0f} KB  ({len(files)} files embedded)")
    print(f"dist/ total      {total/1024/1024:.1f} MB  (Python runtime vendored, no CDN)")
    if size > 8_000_000:
        print("error: page is over 8MB", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
