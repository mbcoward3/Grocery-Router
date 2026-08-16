"""Tests for the code that refuses to trust the planner.

None of these call a model. That is the point: every guard here has to hold whatever the
model returns, including the returns a model has never yet produced. The scout observed
one model behaving well without the recency guard, on one sample. One sample is not proof
of safety, and the opposite has happened in this project before.
"""

from __future__ import annotations

import sys
import unittest
from datetime import date
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from gr import planner as PL          # noqa: E402
from gr import repo as R              # noqa: E402
from gr import session as SE          # noqa: E402
from gr import weekfile as W          # noqa: E402

REPO = R.load(ROOT)


def pick(slug, kind="plain", reason="a reason.", title=None):
    return {"slug": slug, "title": title or slug, "reason_kind": kind, "reason": reason}


class TestConstraintChecks(unittest.TestCase):

    def test_an_unknown_slug_is_dropped_and_never_nudged(self):
        result = PL.check(REPO, [pick("chicken-parmesan-supreme")], nights=5)
        self.assertEqual(result.meals, [])
        self.assertEqual(len(result.dropped), 1)
        self.assertIn("slug", result.dropped[0][1])

    def test_a_hallucinated_title_does_not_get_fuzzy_matched(self):
        # Guessing the join is the silent-failure class this project keeps paying for.
        result = PL.check(REPO, [pick("beef-stew", title="Beef stew")], nights=5)
        self.assertEqual(result.meals, [])

    def test_a_duplicate_pick_is_dropped(self):
        result = PL.check(REPO, [pick("chili"), pick("chili")], nights=5)
        self.assertEqual(len(result.meals), 1)
        self.assertEqual(len(result.dropped), 1)

    def test_a_peanut_reason_is_dropped(self):
        result = PL.check(
            REPO, [pick("tacos", reason="Serve with a peanut sauce on the side.")],
            nights=5)
        self.assertEqual(result.meals, [])
        self.assertIn("peanut", result.dropped[0][1])

    def test_at_most_two_unknown_yield_meals_survive(self):
        unknowns = ["zuppa-toscana", "tuna-melt", "cheesy-pasta", "chicken-chili"]
        result = PL.check(REPO, [pick(s) for s in unknowns], nights=5)
        self.assertEqual(len(result.meals), 2)
        self.assertEqual(len(result.dropped), 2)
        for _, why in result.dropped:
            self.assertIn("unknown yield", why)

    def test_stale_is_rewritten_to_never(self):
        result = PL.check(REPO, [pick("chili", kind="stale", reason="A plain reason.")],
                          nights=5)
        self.assertEqual(result.meals[0].reason_kind, "never")
        self.assertTrue(result.notes)

    def test_recency_claims_are_stripped_from_the_prose(self):
        claims = [
            "You last cooked this in March. It is overdue.",
            "This has not been made in weeks.",
            "A dormant recipe worth surfacing.",
            "You haven't made this recently.",
        ]
        for claim in claims:
            with self.subTest(claim=claim):
                result = PL.check(REPO, [pick("chili", reason=claim)], nights=5)
                text = result.meals[0].reason
                self.assertNotIn("overdue", text.lower())
                self.assertNotIn("dormant", text.lower())
                self.assertNotIn("weeks", text.lower())
                self.assertTrue(result.notes, "the household must be told code edited it")

    def test_a_reason_that_is_only_a_recency_claim_is_replaced_not_emptied(self):
        result = PL.check(REPO, [pick("chili", reason="It has been ages since you made this.")],
                          nights=5)
        self.assertTrue(result.meals[0].reason.strip())
        self.assertIn("unranked", result.meals[0].reason)

    def test_an_honest_reason_survives_untouched(self):
        reason = ("Low active time, and it puts fish in a week that is otherwise all "
                  "beef and chicken.")
        result = PL.check(REPO, [pick("easy-salmon-dinner", reason=reason)], nights=5)
        self.assertEqual(result.meals[0].reason, reason)
        self.assertEqual(result.notes, [])


class TestFill(unittest.TestCase):

    def test_drops_are_topped_up_from_the_corpus(self):
        result = PL.check(REPO, [pick("not-a-real-slug")], nights=5)
        result = PL.fill(REPO, result, nights=5)
        self.assertEqual(len(result.meals), 5)

    def test_a_filled_meal_says_code_chose_it(self):
        result = PL.fill(REPO, PL.PlannerResult(), nights=3)
        for meal in result.meals:
            self.assertIn("chosen by code", meal.reason)

    def test_fill_respects_the_unknown_yield_cap(self):
        result = PL.fill(REPO, PL.PlannerResult(), nights=8)
        unknown = [m for m in result.meals
                   if REPO.row(m.slug).yield_.shape == "unknown"]
        self.assertLessEqual(len(unknown), 2)

    def test_fill_never_repeats_a_meal(self):
        result = PL.fill(REPO, PL.PlannerResult(), nights=7)
        slugs = [m.slug for m in result.meals]
        self.assertEqual(len(slugs), len(set(slugs)))


class TestPlannerPrompt(unittest.TestCase):

    def test_no_ingredient_line_reaches_the_prompt(self):
        """The guarantee the design actually makes, asserted rather than assumed.

        No line of any `recipes/*.md` ingredient list may appear in the prompt. That —
        plus `--tools ""`, which stops the planner opening one itself — is what makes
        invented ingredient coupling impossible rather than merely discouraged.

        It is **not** a claim that the word "beef" never appears. `corpus.md`'s own Notes
        column says things like `ground beef + seasoning packet`, written by the
        household, and `profile.md` discusses the shortcuts this household cooks with.
        Those stay: one of them is the peanut warning on the two bought sauces. What the
        planner never gets is a quantity, a unit, or a list it could add up.
        """
        prompt = PL.build_prompt(REPO, nights=5, guests=0).lower()
        for recipe in REPO.recipes.values():
            for line in recipe.lines:
                body = line.raw.lstrip("- ").strip().lower()
                if len(body) < 12:
                    continue
                self.assertNotIn(body, prompt,
                                 f"an ingredient line reached the prompt: {body!r}")

    def test_the_prompt_carries_no_quantity_the_model_could_add_up(self):
        prompt = PL.build_prompt(REPO, nights=5, guests=0).lower()
        for phrase in ["tablespoon", "teaspoon", " tbsp", " tsp",
                       " oz ", "pound of", "cups of"]:
            self.assertNotIn(phrase, prompt,
                             f"a measurement reached the planner's prompt: {phrase!r}")

    def test_the_prompt_carries_every_slug(self):
        prompt = PL.build_prompt(REPO, nights=5, guests=0)
        for row in REPO.all_rows:
            self.assertIn(row.slug, prompt)

    def test_the_prompt_states_the_sides_gap_rather_than_hiding_it(self):
        prompt = PL.build_prompt(REPO, nights=5, guests=0)
        self.assertIn("NO sides are recorded", prompt)
        self.assertIn("Do not propose a side", prompt)

    def test_the_call_removes_the_model_s_tools(self):
        """`--tools ""` is a process boundary, not a convention.

        This asserts the argument is built. The boundary itself is the CLI's: a planner
        launched this way physically cannot open a recipe file.
        """
        seen = {}

        def fake_run(cmd, **kwargs):
            seen["cmd"] = cmd
            raise FileNotFoundError

        import subprocess
        original = subprocess.run
        subprocess.run = fake_run
        try:
            PL.call_claude("hello", PL.SCHEMA)
        finally:
            subprocess.run = original

        cmd = seen["cmd"]
        self.assertIn("--tools", cmd)
        self.assertEqual(cmd[cmd.index("--tools") + 1], "")
        self.assertIn("--output-format", cmd)
        self.assertEqual(cmd[cmd.index("--output-format") + 1], "json")
        self.assertIn("--json-schema", cmd)


class TestPlannerFailureHandling(unittest.TestCase):
    """Every row of the failure table the scout observed."""

    def _with_fake_claude(self, stdout, stderr=b"", returncode=0):
        import subprocess

        class Result:
            pass

        def fake_run(cmd, **kwargs):
            r = Result()
            r.stdout, r.stderr, r.returncode = stdout, stderr, returncode
            return r

        original = subprocess.run
        subprocess.run = fake_run
        try:
            return PL.call_claude("hello", PL.SCHEMA)
        finally:
            subprocess.run = original

    def test_success_reads_structured_output(self):
        out = b'{"is_error":false,"structured_output":{"picks":[]},"result":"{}"}'
        structured, error, _ = self._with_fake_claude(out)
        self.assertEqual(structured, {"picks": []})
        self.assertEqual(error, "")

    def test_subtype_success_with_is_error_true_is_a_failure(self):
        # Observed: the envelope said "subtype":"success" alongside a 404.
        out = (b'{"is_error":true,"subtype":"success","api_error_status":404,'
               b'"result":"model not found"}')
        structured, error, _ = self._with_fake_claude(out, returncode=1)
        self.assertIsNone(structured)
        self.assertIn("404", error)

    def test_empty_stdout_is_handled(self):
        # A malformed --json-schema writes to stderr only and leaves stdout empty.
        structured, error, _ = self._with_fake_claude(
            b"", stderr=b"Error: --json-schema is not valid JSON", returncode=1)
        self.assertIsNone(structured)
        self.assertIn("json-schema", error)

    def test_a_budget_error_with_no_result_key_is_handled(self):
        out = (b'{"is_error":true,"subtype":"error_max_budget_usd",'
               b'"errors":["budget exceeded"]}')
        structured, error, _ = self._with_fake_claude(out, returncode=1)
        self.assertIsNone(structured)
        self.assertIn("budget", error)

    def test_a_missing_cli_does_not_stop_the_week(self):
        import subprocess

        def fake_run(cmd, **kwargs):
            raise FileNotFoundError

        original = subprocess.run
        subprocess.run = fake_run
        try:
            result = PL.plan(REPO, nights=5, guests=0)
        finally:
            subprocess.run = original

        self.assertEqual(result.source, "code")
        self.assertEqual(len(result.meals), 5, "a failed call must still produce a week")
        self.assertIn("not on PATH", result.error)
        self.assertTrue(any("code chose this week" in n for n in result.notes))


class TestWeekFile(unittest.TestCase):

    def setUp(self):
        import tempfile
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        for name in ("items.md", "corpus.md", "candidates.md", "profile.md", "sides.md"):
            (self.root / name).write_text((ROOT / name).read_text(encoding="utf-8"),
                                          encoding="utf-8")
        (self.root / "recipes").mkdir()
        for src in (ROOT / "recipes").glob("*.md"):
            (self.root / "recipes" / src.name).write_text(
                src.read_text(encoding="utf-8"), encoding="utf-8")
        (self.root / "weeks").mkdir()
        self.repo = R.load(self.root)

    def tearDown(self):
        self.tmp.cleanup()

    def _week(self):
        result = PL.fill(self.repo, PL.PlannerResult(source="code"), nights=4)
        return SE.assemble(self.repo, result.meals, date(2026, 8, 9), 4, 0,
                           planner_source="code")

    def test_the_sunday_names_the_week(self):
        self.assertEqual(W.sunday_of(date(2026, 8, 12)), date(2026, 8, 9))
        self.assertEqual(W.sunday_of(date(2026, 8, 9)), date(2026, 8, 9))
        self.assertEqual(W.sunday_of(date(2026, 8, 8)), date(2026, 8, 2))

    def test_the_week_and_the_list_are_one_file(self):
        week = self._week()
        text = week.path.read_text(encoding="utf-8")
        self.assertIn("## Meals", text)
        self.assertIn("## Shopping list", text)
        self.assertIn("## Probably have", text)

    def test_the_sides_shortfall_is_printed_verbatim(self):
        text = self._week().path.read_text(encoding="utf-8")
        self.assertIn(
            "sides: none recorded — this list is short by design, not by accident.", text)

    def test_a_tick_survives_being_written_again(self):
        week = self._week()
        key = next(W.line_key("buy", l.item) for l in week.shopping.buy)

        self.assertTrue(W.toggle_tick(week.path, key))
        self.assertIn(key, W.read_ticks(week.path))

        # Regenerating the same week must not lose what the shopper already ticked.
        SE.assemble(self.repo, week.meals, week.sunday, 4, 0, planner_source="code")
        self.assertIn(key, W.read_ticks(week.path),
                      "a shopper who loses their ticks mid-aisle will not use this twice")

        self.assertFalse(W.toggle_tick(week.path, key))
        self.assertNotIn(key, W.read_ticks(week.path))

    def test_a_week_reloads_from_its_file(self):
        week = self._week()
        reloaded = SE.load_existing(self.root, date(2026, 8, 9))
        self.assertIsNotNone(reloaded)
        self.assertEqual([m.slug for m in reloaded.meals], [m.slug for m in week.meals])
        self.assertTrue(all(m.reason for m in reloaded.meals))

    def test_every_decision_is_logged(self):
        self._week()
        lines = (self.root / "decisions.jsonl").read_text(encoding="utf-8").splitlines()
        self.assertEqual(len(lines), 1)
        import json
        record = json.loads(lines[0])
        self.assertEqual(record["week"], "2026-08-09")
        self.assertEqual(len(record["added"]), 4)

    def test_a_swap_changes_one_meal_and_leaves_the_rest(self):
        week = self._week()
        before = [m.slug for m in week.meals]
        swapped = SE.swap(self.repo, week, before[1])
        after = [m.slug for m in swapped.meals]
        self.assertNotEqual(after[1], before[1])
        self.assertEqual(after[0], before[0])
        self.assertEqual(after[2:], before[2:])

    def test_must_include_explicitly_restores_beef_dip_over_beef_pot_roast(self):
        week = self._week()
        original = [m.slug for m in week.meals]
        self.assertIn("beef-dip-sammies", original)
        self.assertIn("beef-pot-roast", original)

        accidental = SE.swap(self.repo, week, "beef-dip-sammies")
        accidental_slugs = [m.slug for m in accidental.meals]
        self.assertNotIn("beef-dip-sammies", accidental_slugs)
        displaced_index = original.index("beef-dip-sammies")
        roast_index = accidental_slugs.index("beef-pot-roast")

        restored = SE.include_recipe(
            self.repo, accidental, "beef-dip-sammies", outgoing="beef-pot-roast"
        )
        after = [m.slug for m in restored.meals]
        self.assertEqual(after[roast_index], "beef-dip-sammies")
        self.assertEqual(after[displaced_index], accidental_slugs[displaced_index])
        self.assertEqual(after.count("beef-dip-sammies"), 1)
        self.assertNotIn("beef-pot-roast", after)
        self.assertEqual(len(after), len(original))

    def test_must_include_replaces_only_the_final_meal_and_updates_the_list(self):
        week = self._week()
        before = [m.slug for m in week.meals]
        self.assertNotIn("hamburgers", before)
        self.assertNotIn("ground_beef", [line.item for line in week.shopping.buy])

        included = SE.include_recipe(self.repo, week, "hamburgers")
        after = [m.slug for m in included.meals]

        self.assertEqual(len(after), len(before))
        self.assertEqual(after[:-1], before[:-1])
        self.assertEqual(after[-1], "hamburgers")
        self.assertEqual(after.count("hamburgers"), 1)
        self.assertIn("buns", [line.item for line in included.shopping.buy])
        reloaded = SE.load_existing(self.root, date(2026, 8, 9))
        self.assertEqual([m.slug for m in reloaded.meals], after)
        self.assertIn("buns", [line.item for line in reloaded.shopping.buy])

    def test_must_include_is_a_no_op_when_already_present_or_invalid(self):
        week = self._week()
        original = week.path.read_text(encoding="utf-8")
        existing = week.meals[0].slug

        self.assertIs(SE.include_recipe(self.repo, week, existing), week)
        self.assertIs(SE.include_recipe(self.repo, week, "not-a-recipe"), week)
        self.assertIs(SE.include_recipe(self.repo, week, ""), week)
        self.assertIs(SE.include_recipe(self.repo, week, "hamburgers",
                                        outgoing="not-in-the-week"), week)
        self.assertEqual(week.path.read_text(encoding="utf-8"), original)
        self.assertEqual([m.slug for m in week.meals].count(existing), 1)

    def test_must_include_preserves_the_unknown_yield_cap(self):
        slugs = ["cheesy-pasta", "chicken-chili", "hamburgers", "tacos"]
        meals = [PL.MealPlan(slug=slug, title=self.repo.row(slug).title,
                             reason_kind="plain", reason="chosen for test",
                             yield_raw=self.repo.row(slug).yield_raw, scale=None,
                             untried=False)
                 for slug in slugs]
        week = SE.assemble(self.repo, meals, date(2026, 8, 9), 4, 0,
                           planner_source="code")

        included = SE.include_recipe(self.repo, week, "zuppa-toscana")
        after = [m.slug for m in included.meals]
        unknowns = [slug for slug in after
                    if self.repo.row(slug).yield_.shape == "unknown"]
        self.assertEqual(len(after), 4)
        self.assertEqual(len(set(after)), 4)
        self.assertIn("zuppa-toscana", after)
        self.assertLessEqual(len(unknowns), 2)
        self.assertEqual(after[2:], slugs[2:])


if __name__ == "__main__":
    unittest.main(verbosity=2)
