#!/usr/bin/env python3
"""Two households, and the bug that made this refactor step one.

**These are the tests the old code could not have passed.** `pantry.py` used to
keep which household it was reading in module-level globals, and `app.py` — a
`ThreadingHTTPServer`, concurrent since the day it was written — reassigned them
inside a request handler. One household made that safe. Two would have made it a
silent data-crossing bug: household A's corpus written into household B's file,
no crash, no traceback, no way to find out.

`docs/multi-tenancy.md` calls this the first thing in the way, ahead of the
database. Everything else in that document is a design decision that can be
revisited. This is a defect, and it had to be fixed before anything had two rows
in it.

The concurrency test below is the one that matters. It is not a demonstration
that threads work — it is a regression test for module state: reintroduce a
global that either household writes and it fails, which is exactly when someone
needs to be told.
"""

from __future__ import annotations

import shutil
import tempfile
import threading
import unittest
from pathlib import Path

import household
import pantry
import shop
from household import Household

REAL = Path(__file__).resolve().parent


def build_household(root: Path, name: str, recipes: list[str]) -> Household:
    """A household with a corpus of its own, sharing nothing with any other."""
    root.mkdir(parents=True, exist_ok=True)
    rows = "\n".join(
        f"| {r} | chicken | american | 4 AE | low | — |  |" for r in recipes)
    (root / "corpus.md").write_text(
        f"# {name}\n\n"
        "| Recipe | Protein | Cuisine | Yield | Active | Passive | Last cooked |\n"
        "|---|---|---|---|---|---|---|\n" + rows + "\n", encoding="utf-8")
    (root / "candidates.md").write_text(
        "# Candidates\n\n| Recipe | Source |\n|---|---|\n", encoding="utf-8")
    (root / "sides.md").write_text(
        "# Sides\n\n| Side | Goes with | Season | Active | Passive | Last served | "
        "Notes |\n|---|---|---|---|---|---|---|\n", encoding="utf-8")
    (root / "profile.md").write_text(
        f"# {name}\n\n## Members\n\n- {name}Cook\n", encoding="utf-8")
    (root / "recipes").mkdir(exist_ok=True)
    for r in recipes:
        slug = pantry.slug(r)
        (root / "recipes" / f"{slug}.md").write_text(
            f"# {r}\n\nyield: 4 AE\n\n## Ingredients\n\n- 1 lb chicken breast\n",
            encoding="utf-8")
    return Household(root=root, id=name.lower())


class TwoHouseholds(unittest.TestCase):
    """A and B have disjoint corpora, so any crossing is unambiguous."""

    A_RECIPES = ["Alpha stew", "Alpha pie", "Alpha bake", "Alpha roast", "Alpha soup",
                 "Alpha hash"]
    B_RECIPES = ["Bravo stew", "Bravo pie", "Bravo bake", "Bravo roast", "Bravo soup",
                 "Bravo hash"]

    def setUp(self):
        self.tmp = Path(tempfile.mkdtemp())
        self.a = build_household(self.tmp / "a", "Alpha", self.A_RECIPES)
        self.b = build_household(self.tmp / "b", "Bravo", self.B_RECIPES)

    def tearDown(self):
        shutil.rmtree(self.tmp, ignore_errors=True)

    def test_each_household_reads_only_its_own_corpus(self):
        titles_a = {r["recipe"] for r in pantry.load_corpus(self.a)}
        titles_b = {r["recipe"] for r in pantry.load_corpus(self.b)}
        self.assertEqual(titles_a, set(self.A_RECIPES))
        self.assertEqual(titles_b, set(self.B_RECIPES))
        self.assertEqual(titles_a & titles_b, set())

    def test_a_write_to_one_is_invisible_to_the_other(self):
        pantry.add_candidate(self.a, "Alpha newcomer", source="https://example.com/a")
        self.assertIn("alpha-newcomer",
                      {r["slug"] for r in pantry.load_candidates(self.a)})
        self.assertNotIn("alpha-newcomer",
                         {r["slug"] for r in pantry.load_candidates(self.b)})
        self.assertNotIn("Alpha newcomer", self.b.candidates.read_text(encoding="utf-8"))

    def test_the_decision_logs_do_not_mix(self):
        pantry.log(self.a, "proposed", week="2026-01-05")
        pantry.log(self.b, "proposed", week="2026-01-05")
        pantry.log(self.b, "proposed", week="2026-01-12")
        self.assertEqual(len(pantry.decisions(self.a)), 1)
        self.assertEqual(len(pantry.decisions(self.b)), 2)

    def test_the_recipe_index_is_per_household(self):
        """The index was a module global that four writers had to remember to
        reset. Cached wrong, it hands one household the other's filenames."""
        self.assertIn("alpha-stew", pantry.file_index(self.a))
        self.assertNotIn("bravo-stew", pantry.file_index(self.a))
        self.assertIn("bravo-stew", pantry.file_index(self.b))
        self.assertNotIn("alpha-stew", pantry.file_index(self.b))

    def test_a_profile_claim_lands_in_one_profile(self):
        pantry.add_profile_claim(self.a, "Members", "- AlphaGuest",
                                 evidence="they said so")
        self.assertIn("AlphaGuest", self.a.profile.read_text(encoding="utf-8"))
        self.assertNotIn("AlphaGuest", self.b.profile.read_text(encoding="utf-8"))

    def test_the_shopping_list_reads_its_own_recipes(self):
        built_a = shop.build(self.a, ["alpha-stew"], 2.5)
        built_b = shop.build(self.b, ["bravo-stew"], 2.5)
        self.assertEqual(built_a[0][0][0].slug, "alpha-stew")
        self.assertEqual(built_b[0][0][0].slug, "bravo-stew")
        with self.assertRaises(FileNotFoundError):
            shop.build(self.a, ["bravo-stew"], 2.5)


class Concurrently(unittest.TestCase):
    """**The regression test for the bug itself.**

    Both households plan, write and re-read a week at the same time, in threads,
    the way two requests to `app.py` always could. Under module globals the two
    threads race on the same eight names and one week ends up holding the other
    household's dinners. Nothing here is a crash; the assertion is that no meal
    a household never had reaches its own week file.
    """

    ROUNDS = 12

    def setUp(self):
        self.tmp = Path(tempfile.mkdtemp())
        self.a = build_household(self.tmp / "a", "Alpha", TwoHouseholds.A_RECIPES)
        self.b = build_household(self.tmp / "b", "Bravo", TwoHouseholds.B_RECIPES)

    def tearDown(self):
        shutil.rmtree(self.tmp, ignore_errors=True)

    def test_two_households_planning_at_once_never_cross(self):
        errors: list[str] = []
        start = threading.Barrier(2)

        def run(hh: Household, mine: set[str], other: set[str]):
            start.wait()
            for i in range(self.ROUNDS):
                date = f"2026-01-{5 + i:02d}"
                week = pantry.Week(date=date, nights=3)
                week.meals = pantry.propose(hh, 3, week=date)
                pantry.write_week(hh, week)
                back = pantry.read_week(hh, date)
                if back is None:
                    errors.append(f"{hh.id}: week {date} did not come back")
                    continue
                titles = {m.title for m in back.meals}
                stray = titles & other
                if stray:
                    errors.append(f"{hh.id}: {sorted(stray)} is not this household's")
                if not titles <= mine:
                    errors.append(f"{hh.id}: unknown meals {sorted(titles - mine)}")

        threads = [
            threading.Thread(target=run, args=(self.a, set(TwoHouseholds.A_RECIPES),
                                               set(TwoHouseholds.B_RECIPES))),
            threading.Thread(target=run, args=(self.b, set(TwoHouseholds.B_RECIPES),
                                               set(TwoHouseholds.A_RECIPES))),
        ]
        for t in threads:
            t.start()
        for t in threads:
            t.join()

        self.assertEqual(errors, [], "\n".join(errors[:8]))

    def test_neither_household_wrote_into_the_others_directory(self):
        """The filesystem's own account, after the fact. A week file under the
        wrong root is the failure in its most literal form."""
        self.test_two_households_planning_at_once_never_cross()
        for week in sorted(self.a.weeks.glob("*.md")):
            self.assertNotIn("Bravo", week.read_text(encoding="utf-8"))
        for week in sorted(self.b.weeks.glob("*.md")):
            self.assertNotIn("Alpha", week.read_text(encoding="utf-8"))


class TheGlobalsStayGone(unittest.TestCase):
    """A guard, not a unit test.

    Every one of these names was a module global that a request handler
    reassigned. Adding one back is not a style regression - it is this bug
    returning, and it returns silently. So the absence is asserted rather than
    trusted to review.
    """

    GONE_FROM_PANTRY = ["ROOT", "CORPUS", "CANDIDATES", "SIDES", "PROFILE",
                        "WEEKS", "CACHE", "DECISIONS", "_FILE_INDEX"]
    GONE_FROM_SHOP = ["ROOT", "RECIPES", "ITEMS", "configure"]

    def test_pantry_holds_no_household_state(self):
        for name in self.GONE_FROM_PANTRY:
            with self.subTest(name=name):
                self.assertFalse(
                    hasattr(pantry, name),
                    f"pantry.{name} is back. Which household a call is about is "
                    f"an argument; see household.py.")

    def test_shop_holds_no_household_state(self):
        for name in self.GONE_FROM_SHOP:
            with self.subTest(name=name):
                self.assertFalse(
                    hasattr(shop, name),
                    f"shop.{name} is back. `configure()` set module paths from "
                    f"inside a request handler, which is the bug.")

    def test_the_household_is_required_and_has_no_default(self):
        """No library function may default to `here()`. A default is the implicit
        global returning one call site at a time, and it fails the way the
        original did: not at all, until it is somebody else's data."""
        for fn in (pantry.load_corpus, pantry.load_candidates, pantry.load_sides,
                   pantry.load_members, pantry.list_weeks, pantry.briefing,
                   pantry.file_index, pantry.decisions):
            with self.subTest(fn=fn.__name__):
                with self.assertRaises(TypeError):
                    fn()

    def test_only_here_constructs_the_default_household(self):
        hh = household.here()
        self.assertEqual(hh.root, household.REPO)
        self.assertEqual(hh.corpus, household.REPO / "corpus.md")


class HostStateStaysShared(unittest.TestCase):
    """The inverse mistake, and it is a real one.

    `acquire/adapters.py` caches which search strategy works per *hostname*, and
    when each host was last hit. Those are not tenant state: 500 households must
    not each re-probe `thecountrycook.net`, and `_last_hit` is the courtesy delay
    between requests - made per-tenant it would turn a polite client into a
    scraper that merely looks polite. Threading a household through these would
    be this refactor applied one module too far.
    """

    def test_the_adapter_caches_are_keyed_by_host_not_by_household(self):
        from acquire import adapters as search_adapters
        for name in ("_chosen", "_robots", "_last_hit"):
            with self.subTest(name=name):
                self.assertTrue(hasattr(search_adapters, name),
                                f"{name} is host state and is shared on purpose")

    def test_the_store_token_is_platform_state(self):
        from adapters import kroger
        self.assertTrue(hasattr(kroger, "_token"),
                        "the OAuth token belongs to this deployment's credentials, "
                        "not to a household")


if __name__ == "__main__":
    unittest.main(verbosity=2)
