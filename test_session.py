#!/usr/bin/env python3
"""Tests for session depth: servings, swap, lock, the recipe, the profile.

    python3 test_session.py

`docs/brief-next.md` §5. These go through `app.handle`, which is the one place
routing lives and what both the HTTP server and the browser build call — so a
route that works here works in a browser tab with no socket behind it.
"""

import shutil
import tempfile
import unittest
from pathlib import Path

import app
import pantry
import shop


REAL = Path(__file__).resolve().parent


class Session(unittest.TestCase):
    def setUp(self):
        self.tmp = Path(tempfile.mkdtemp())
        for name in ("corpus.md", "candidates.md", "sides.md", "profile.md", "items.md"):
            shutil.copy(REAL / name, self.tmp / name)
        shutil.copytree(REAL / "recipes", self.tmp / "recipes")
        self._saved = {k: getattr(pantry, k) for k in
                       ("ROOT", "CORPUS", "CANDIDATES", "SIDES", "PROFILE", "WEEKS", "CACHE",
                        "DECISIONS")}
        pantry.ROOT = self.tmp
        pantry.CORPUS = self.tmp / "corpus.md"
        pantry.CANDIDATES = self.tmp / "candidates.md"
        pantry.SIDES = self.tmp / "sides.md"
        pantry.PROFILE = self.tmp / "profile.md"
        pantry.WEEKS = self.tmp / "weeks"
        pantry.CACHE = self.tmp / ".cache"
        pantry.DECISIONS = self.tmp / "decisions.jsonl"
        pantry._FILE_INDEX = None
        shop.configure(self.tmp)

    def tearDown(self):
        for k, v in self._saved.items():
            setattr(pantry, k, v)
        pantry._FILE_INDEX = None
        shop.configure(self._saved["ROOT"])
        shutil.rmtree(self.tmp, ignore_errors=True)

    def get(self, path):
        code, body = app.handle(path)
        self.assertEqual(code, 200, body)
        return body

    def post(self, path, **body):
        code, out = app.handle(path, body)
        self.assertEqual(code, 200, out)
        return out

    def week(self):
        return self.get("/api/state")["week"]

    def first(self):
        return self.week()["meals"][0]["slug"]


# --------------------------------------------------------------------------- #

class TestServings(Session):
    """`profile.md` asks for this outright, and both model-planned weeks scaled a
    meal on their own the first time they ran."""

    def test_servings_are_set_per_meal(self):
        slug = self.first()
        meals = self.post("/api/servings", slug=slug, ae=6)["week"]["meals"]
        self.assertEqual(meals[0]["ae_override"], 6.0)
        self.assertEqual([m["ae_override"] for m in meals[1:]], [0.0] * (len(meals) - 1))

    def test_zero_puts_the_meal_back_on_the_week(self):
        slug = self.first()
        self.post("/api/servings", slug=slug, ae=6)
        self.assertEqual(self.post("/api/servings", slug=slug,
                                   ae=0)["week"]["meals"][0]["ae_override"], 0.0)

    def test_it_survives_a_reload(self):
        """The field existed before and the week file had nowhere to put it."""
        slug = self.first()
        self.post("/api/servings", slug=slug, ae=5.5)
        self.assertEqual(pantry.read_week(pantry.monday()).meals[0].ae_override, 5.5)

    def test_it_reaches_the_shopping_list(self):
        """The point of the feature. A dial that changes no output is the bug
        this project has already shipped once."""
        week = pantry.Week(date=pantry.monday())
        row = next(r for r in pantry.load_corpus() if r["slug"] == "chili")
        week.meals = [pantry.Meal(slug="chili", title=row["recipe"],
                                  yield_=row["yield"], active=row["active"],
                                  reason="test")]
        pantry.write_week(week)
        before = self.get("/api/list")["markdown"]
        self.post("/api/servings", slug="chili", ae=9)
        after = self.get("/api/list")["markdown"]
        self.assertIn("1 lb ground beef", before)
        self.assertIn("3 lb ground beef", after)

    def test_a_meal_with_no_override_is_unchanged_by_another_meals_guests(self):
        """Guests on Thursday are a fact about Thursday. Scaling the week to them
        buys four dinners nobody eats."""
        week = pantry.Week(date=pantry.monday())
        rows = {r["slug"]: r for r in pantry.load_corpus()}
        week.meals = [pantry.Meal(slug=s, title=rows[s]["recipe"],
                                  yield_=rows[s]["yield"], active=rows[s]["active"],
                                  reason="t")
                      for s in ("chili", "meatloaf")]
        pantry.write_week(week)
        before = self.get("/api/list")["markdown"]
        self.post("/api/servings", slug="chili", ae=9)
        after = self.get("/api/list")["markdown"]
        # Only the lines that belong to meatloaf alone. A line the two meals
        # share is one quantity covering both, so it moves when either does -
        # that is the aggregation working, not the override leaking.
        checked = 0
        for line in before.splitlines():
            if line.startswith("- ") and line.rstrip().endswith("— meatloaf"):
                self.assertIn(line, after)
                checked += 1
        self.assertGreater(checked, 0)


class TestSwap(Session):
    def test_a_swap_replaces_one_meal_and_leaves_the_rest(self):
        before = [m["slug"] for m in self.week()["meals"]]
        after = [m["slug"] for m in self.post("/api/swap", slug=before[0])["week"]["meals"]]
        self.assertNotIn(before[0], after)
        for slug in before[1:]:
            self.assertIn(slug, after)
        self.assertEqual(len(after), len(before))

    def test_the_swapped_out_meal_does_not_come_straight_back(self):
        slug = self.first()
        self.post("/api/swap", slug=slug)
        self.post("/api/fill")
        self.assertNotIn(slug, [m["slug"] for m in self.week()["meals"]])

    def test_it_is_one_decision_not_two(self):
        """A drop followed by a refill is two decisions for one intent, and
        `review.py` would read the swap as a rejection and the replacement as an
        unrelated offer."""
        self.post("/api/swap", slug=self.first())
        self.assertEqual(len(pantry.decisions({"swap"})), 1)
        self.assertEqual(len(pantry.decisions({"drop"})), 0)

    def test_a_locked_meal_cannot_be_swapped(self):
        slug = self.first()
        self.post("/api/lock", slug=slug, locked=True)
        out = self.post("/api/swap", slug=slug)
        self.assertIn("locked", out.get("error", ""))
        self.assertIn(slug, [m["slug"] for m in out["week"]["meals"]])


class TestLock(Session):
    """The field existed and nothing could act on it, which made it decoration."""

    def test_a_lock_survives_a_reload(self):
        slug = self.first()
        self.post("/api/lock", slug=slug, locked=True)
        self.assertTrue(pantry.read_week(pantry.monday()).meals[0].locked)

    def test_reshuffle_keeps_what_is_locked_and_moves_the_rest(self):
        """The ranker is deterministic, so re-running it returns the same week.
        Reshuffle means *not these* - it declines what it re-rolled, the same way
        a drop does. Without that it is a button that changes nothing, which this
        project has shipped once already."""
        week = self.week()["meals"]
        keep = week[0]["slug"]
        self.post("/api/lock", slug=keep, locked=True)
        after = [m["slug"] for m in self.post("/api/reshuffle")["week"]["meals"]]
        self.assertIn(keep, after)
        self.assertNotEqual([m["slug"] for m in week], after)
        for slug in (m["slug"] for m in week[1:]):
            self.assertNotIn(slug, after)

    def test_reshuffle_with_everything_locked_changes_nothing(self):
        before = [m["slug"] for m in self.week()["meals"]]
        for slug in before:
            self.post("/api/lock", slug=slug, locked=True)
        self.assertEqual([m["slug"] for m in self.post("/api/reshuffle")["week"]["meals"]],
                         before)

    def test_a_lock_keeps_its_servings(self):
        slug = self.first()
        self.post("/api/servings", slug=slug, ae=7)
        self.post("/api/lock", slug=slug, locked=True)
        meals = self.post("/api/reshuffle")["week"]["meals"]
        kept = next(m for m in meals if m["slug"] == slug)
        self.assertEqual(kept["ae_override"], 7.0)


class TestSeeTheRecipe(Session):
    def test_the_recipe_comes_back_as_its_own_markdown(self):
        out = self.get("/api/recipe/chili")
        self.assertTrue(out["ok"])
        self.assertIn("## Ingredients", out["markdown"])
        self.assertIn("ground hamburger", out["markdown"])

    def test_a_meal_with_no_capture_says_so_rather_than_failing(self):
        out = self.get("/api/recipe/nothing-on-file")
        self.assertFalse(out["ok"])
        self.assertIn("No capture", out["markdown"])


class TestEditingTheProfile(Session):
    """`profile.md` opens by saying correcting it *is* the trust mechanism. In a
    hosted deployment that sentence was false."""

    def test_the_profile_can_be_read_and_written(self):
        text = self.get("/api/profile")["text"]
        self.assertIn("## Members", text)
        self.post("/api/profile", text=text + "\n- **Test line.** *Because so.*\n")
        self.assertIn("Test line", pantry.PROFILE.read_text())

    def test_an_empty_profile_is_refused(self):
        before = pantry.PROFILE.read_text()
        out = self.post("/api/profile", text="   ")
        self.assertIn("empty", out["profile_error"])
        self.assertEqual(pantry.PROFILE.read_text(), before)

    def test_a_profile_that_stops_parsing_is_rolled_back(self):
        """Losing the Members section is how attribution silently stops working,
        and it would be found weeks later by a planner with no constraints."""
        before = pantry.PROFILE.read_text()
        out = self.post("/api/profile", text="# Just a heading, nothing else")
        self.assertIn("Members", out["profile_error"])
        self.assertEqual(pantry.PROFILE.read_text(), before)

    def test_the_edit_is_recorded(self):
        self.post("/api/profile", text=self.get("/api/profile")["text"] + "\n")
        self.assertEqual(len(pantry.decisions({"profile_edited"})), 1)

    def test_the_planner_sees_the_edit(self):
        """The write is only worth anything if the next proposal reads it."""
        text = self.get("/api/profile")["text"]
        self.post("/api/profile", text=text.replace("- Michael", "- Michael\n- Robin"))
        self.assertIn("Robin", self.get("/api/state")["members"])

class TestSides(Session):
    """§6 — the largest known correctness gap. Every grocery list this tool has
    produced has been systematically short, and every one has said so."""

    def add(self, name, *raw, source="https://x.test/s/"):
        """A side with a real capture behind it."""
        import onboard
        slug = pantry.slug(name)
        pantry.recipe_file(slug).write_text(
            f"# {name}\n\nsource:   {source}\nyield:    4 AE\n"
            f"peanut:   none seen\nstatus:   complete\n\n## Ingredients\n\n"
            + "\n".join(f"- {r}" for r in raw) + "\n")
        pantry._FILE_INDEX = None
        return pantry.add_side(name, source=source, active="low")

    def test_it_starts_empty_and_that_is_the_point(self):
        """Seeding it would be inventing what this household eats, which is the
        one thing this project refuses to do anywhere else."""
        self.assertEqual(pantry.load_sides(), [])
        self.assertEqual(self.get("/api/state")["counts"]["sides"], 0)

    def test_the_list_says_it_is_short_while_there_are_none(self):
        self.assertIn("Sides are not included", self.get("/api/list")["markdown"])

    def test_a_side_reaches_the_shopping_list(self):
        self.add("Roasted carrots", "2 lb carrots", "2 tbsp olive oil")
        self.post("/api/side", slug="roasted-carrots")
        out = self.get("/api/list")["markdown"]
        self.assertIn("carrots", out)
        self.assertIn("1 side(s) included", out)
        self.assertNotIn("Sides are not included", out)

    def test_a_side_merges_with_a_main_rather_than_repeating(self):
        """The reason sides are captured as recipe files rather than as a list of
        words: same parser, same aggregation, one line for one onion."""
        week = pantry.Week(date=pantry.monday())
        row = next(r for r in pantry.load_corpus() if r["slug"] == "chili")
        week.meals = [pantry.Meal(slug="chili", title=row["recipe"],
                                  yield_=row["yield"], active=row["active"], reason="t")]
        pantry.write_week(week)
        self.add("Onion salad", "1 onion", "1 tbsp vinegar")
        self.post("/api/side", slug="onion-salad")
        out = self.get("/api/list")["markdown"]
        # The Produce section only. The coupling report below it names shared
        # items again by design, and the vinegar is a different ingredient.
        produce = out.split("## Produce")[1].split("\n##")[0]
        onions = [l for l in produce.splitlines()
                  if l.startswith("- ") and "onions" in l]
        self.assertEqual(len(onions), 1, onions)
        self.assertIn("chili", onions[0])
        self.assertIn("onion salad", onions[0])

    def test_a_side_typed_in_by_name_is_allowed_and_reported(self):
        """`green beans` is a side and everyone knows what it is. Refusing it
        until somebody finds a web page would be the tool getting in the way of
        the file it is asking to be filled — but it carries no ingredients, and
        the list says which one you are shopping for yourself."""
        self.post("/api/side/add", name="Green beans")
        self.post("/api/side", slug="green-beans")
        out = self.get("/api/list")["markdown"]
        self.assertIn("Not on this list", out)
        self.assertIn("Green beans (side)", out)

    def test_sides_do_not_count_as_nights(self):
        """A side is not a cook. Folding it into `meals` would have the effort
        mix, the night count and the shortfall all answered wrongly."""
        before = self.week()
        self.add("Roasted carrots", "2 lb carrots")
        self.post("/api/side", slug="roasted-carrots")
        after = self.week()
        self.assertEqual(len(after["meals"]), len(before["meals"]))
        self.assertEqual(after["effort"], before["effort"])

    def test_a_side_survives_a_reload(self):
        self.add("Roasted carrots", "2 lb carrots")
        self.post("/api/side", slug="roasted-carrots")
        self.assertEqual(pantry.read_week(pantry.monday()).sides, ["roasted-carrots"])

    def test_a_side_can_be_removed(self):
        self.add("Roasted carrots", "2 lb carrots")
        self.post("/api/side", slug="roasted-carrots")
        out = self.post("/api/side", remove="roasted-carrots")
        self.assertEqual(out["week"]["sides"], [])

    def test_suggesting_from_an_empty_file_suggests_nothing(self):
        """Not a failure. The honest response to no data is no suggestion, not a
        plausible vegetable."""
        self.assertEqual(self.post("/api/side", suggest=True)["week"]["sides"], [])

    def test_suggest_prefers_what_has_not_been_served(self):
        self.add("Roasted carrots", "2 lb carrots")
        self.add("Green salad", "1 head lettuce")
        picked = self.post("/api/side", suggest=True, want=1)["week"]["sides"]
        self.assertEqual(len(picked), 1)

    def test_the_same_side_is_not_added_twice(self):
        self.add("Roasted carrots", "2 lb carrots")
        self.assertFalse(self.add("Roasted carrots", "2 lb carrots"))
        self.assertEqual(len(pantry.load_sides()), 1)

    def test_a_side_never_reaches_the_corpus_or_the_candidates(self):
        """Different store, different bar. A side is not a proven dinner and it
        is not an unproven one either."""
        before = (len(pantry.load_corpus()), len(pantry.load_candidates()))
        self.add("Roasted carrots", "2 lb carrots")
        self.assertEqual((len(pantry.load_corpus()), len(pantry.load_candidates())),
                         before)

    def test_a_side_with_no_name_is_refused(self):
        with self.assertRaises(pantry.RuleViolation):
            pantry.add_side("")


if __name__ == "__main__":
    unittest.main(verbosity=2)
