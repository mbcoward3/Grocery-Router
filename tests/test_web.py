"""The web surface over the deterministic core."""

from __future__ import annotations

import sys
import tempfile
import threading
import unittest
from pathlib import Path
from urllib.error import HTTPError
from urllib.parse import urlencode
from urllib.request import Request, urlopen

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from gr import planner as PL  # noqa: E402
from gr import repo as R  # noqa: E402
from gr import session as SE  # noqa: E402
from gr import web  # noqa: E402
from gr import weekfile as W  # noqa: E402


class TestWebApp(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        for name in ("items.md", "corpus.md", "candidates.md", "profile.md", "sides.md"):
            (self.root / name).write_text((ROOT / name).read_text(encoding="utf-8"),
                                          encoding="utf-8")
        (self.root / "recipes").mkdir()
        for source in (ROOT / "recipes").glob("*.md"):
            (self.root / "recipes" / source.name).write_text(
                source.read_text(encoding="utf-8"), encoding="utf-8")
        (self.root / "weeks").mkdir()
        (self.root / "static").symlink_to(ROOT / "static", target_is_directory=True)
        repo = R.load(self.root)
        result = PL.fill(repo, PL.PlannerResult(source="code"), nights=4)
        self.week = SE.assemble(repo, result.meals, W.sunday_of(), 4, 1,
                                planner_source="code")

    def tearDown(self):
        self.tmp.cleanup()

    def test_plan_shows_reasons_counts_and_every_known_gap(self):
        page = web.render_plan(self.root)
        self.assertIn("This week's pool", page)
        self.assertIn(self.week.meals[0].reason, page)
        self.assertIn("Counts describe this pool", page)
        self.assertIn("sides: none recorded", page)
        self.assertIn("effort ratings are the system&#x27;s guess", page)
        self.assertIn("isn&#x27;t in corpus.md", page)

    def test_phone_list_has_large_checkable_rows_and_provenance(self):
        page = web.render_list(self.root)
        first = self.week.shopping.buy[0]
        self.assertIn('data-list-key=', page)
        self.assertIn(web._human(first.item), page)
        self.assertIn("for ", page)
        self.assertIn("Every quantity below comes from recipe files", page)

    def test_no_recipe_ingredient_is_embedded_in_planner_controls(self):
        page = web.render_plan(self.root)
        # The web page triggers gr.planner; it does not add a second model/list path.
        self.assertNotIn("name=\"ingredient", page)
        self.assertIn('action="/plan"', page)

    def test_http_startup_surface_and_tick_persistence(self):
        server = web.ThreadingHTTPServer(("127.0.0.1", 0), web.make_handler(self.root))
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        base = f"http://127.0.0.1:{server.server_address[1]}"
        try:
            health = urlopen(base + "/health", timeout=2).read().decode()
            self.assertEqual(health, "ok\n")
            page = urlopen(base + "/list", timeout=2).read().decode()
            self.assertIn("Shopping list", page)

            line = self.week.shopping.buy[0]
            key = W.line_key("buy", line.item)
            request = Request(base + "/api/toggle", data=urlencode({"key": key}).encode(),
                              method="POST")
            result = urlopen(request, timeout=2).read().decode()
            self.assertIn('"checked": true', result)
            self.assertIn(key, W.read_ticks(self.week.path))

            reloaded = urlopen(base + "/list", timeout=2).read().decode()
            self.assertIn(f'data-list-key="{key}" checked', reloaded)
        finally:
            server.shutdown()
            server.server_close()
            thread.join(timeout=2)

    def test_static_paths_cannot_leave_static_directory(self):
        server = web.ThreadingHTTPServer(("127.0.0.1", 0), web.make_handler(self.root))
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        base = f"http://127.0.0.1:{server.server_address[1]}"
        try:
            with self.assertRaises(HTTPError) as error:
                urlopen(base + "/static/../profile.md", timeout=2)
            self.assertEqual(error.exception.code, 404)
        finally:
            server.shutdown()
            server.server_close()
            thread.join(timeout=2)


if __name__ == "__main__":
    unittest.main(verbosity=2)
