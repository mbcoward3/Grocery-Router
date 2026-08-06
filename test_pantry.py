#!/usr/bin/env python3
"""Tests for the write rules, the week record, and the ranker.

    python3 test_pantry.py

Files do not refuse a bad write, so one module has to. These are the tests for
that refusal - the equivalent of the constraints a schema would enforce, listed
in docs/architecture.md. Everything runs against a temporary copy; nothing here
touches the real corpus.
"""

import datetime as dt
import os
import shutil
import tempfile
import unittest
from pathlib import Path

import household
import pantry
from household import Household

REAL = Path(__file__).resolve().parent


class Isolated(unittest.TestCase):
    """Point pantry at a scratch copy of the household's files.

    Also pins the planner deterministic. `propose()` now picks between the ranker
    and the model on whether `ANTHROPIC_API_KEY` is set, and a test suite that
    starts making paid network calls because of an environment variable someone
    exported in their shell is a suite nobody can trust. These are the ranker's
    tests; `test_planner.py` is where the model planner is tested, against a stub.
    """

    def setUp(self):
        self._env = {k: os.environ.get(k) for k in ("PANTRY_PLANNER", "ANTHROPIC_API_KEY")}
        os.environ["PANTRY_PLANNER"] = "ranker"
        os.environ.pop("ANTHROPIC_API_KEY", None)
        self.tmp = Path(tempfile.mkdtemp())
        for name in ("corpus.md", "candidates.md", "sides.md", "profile.md"):
            shutil.copy(REAL / name, self.tmp / name)
        shutil.copytree(REAL / "recipes", self.tmp / "recipes")
        # One household, rooted in the scratch copy. The harness used to
        # save and reassign eight module globals in `pantry`; the household
        # is an argument now, so isolation is a value rather than a ritual
        # every new test file had to remember to repeat. Four of them once
        # forgot, and the suite wrote into the real `sides.md`.
        self.hh = Household(root=self.tmp, id="test")

    def tearDown(self):
        for k, v in self._env.items():
            if v is None:
                os.environ.pop(k, None)
            else:
                os.environ[k] = v
        self.hh.forget()
        shutil.rmtree(self.tmp, ignore_errors=True)


# --------------------------------------------------------------------------- #

class TestMembershipIsEarned(Isolated):
    """The rule the corpus's entire value rests on. It was stated in five prose
    locations, enforced nowhere, and the one writer violated it."""

    def test_promote_is_the_only_door(self):
        before = len(pantry.load_corpus(self.hh))
        pantry.promote(self.hh, "sheet-pan-chicken-fajitas", "2026-08-03", "kept")
        self.assertEqual(len(pantry.load_corpus(self.hh)), before + 1)

    def test_a_flop_may_not_be_promoted(self):
        with self.assertRaises(pantry.RuleViolation):
            pantry.promote(self.hh, "sheet-pan-chicken-fajitas", "2026-08-03", "flopped")

    def test_a_recipe_that_was_never_a_candidate_may_not_be_promoted(self):
        with self.assertRaises(pantry.RuleViolation):
            pantry.promote(self.hh, "duck-a-l-orange", "2026-08-03", "kept")

    def test_promotion_is_idempotent(self):
        pantry.promote(self.hh, "sheet-pan-chicken-fajitas", "2026-08-03", "kept")
        n = len(pantry.load_corpus(self.hh))
        self.assertFalse(pantry.promote(self.hh, "sheet-pan-chicken-fajitas", "2026-08-10", "kept"))
        self.assertEqual(len(pantry.load_corpus(self.hh)), n, "no duplicate row")

    def test_a_flop_is_never_deleted(self):
        pantry.record_flop(self.hh, "parchment-garlic-butter-salmon", "2026-08-03", "too fiddly")
        row = next(r for r in pantry.load_candidates(self.hh)
                   if r["slug"] == "parchment-garlic-butter-salmon")
        self.assertIn("flopped", row["outcome"])
        self.assertIn("too fiddly", row["outcome"])


class TestNoClaimWithoutATrace(Isolated):
    def test_evidence_is_required(self):
        with self.assertRaises(pantry.RuleViolation):
            pantry.add_profile_claim(self.hh, "Taste", "Loves fennel", "")

    def test_a_claim_with_evidence_lands_with_its_trace_attached(self):
        pantry.add_profile_claim(self.hh, "Taste", "Reaches for acid", "four corpus recipes use lemon", "Sam")
        text = self.hh.profile.read_text()
        self.assertIn("Reaches for acid", text)
        self.assertIn("Because four corpus recipes use lemon", text)
        self.assertIn("Sam", text)


class TestNoOverwritingAHumanValue(Isolated):
    def test_a_populated_cell_is_left_alone(self):
        self.assertFalse(pantry._set_cell(self.hh.corpus, "meatloaf", "protein", "pork"))
        row = next(r for r in pantry.load_corpus(self.hh) if r["slug"] == "meatloaf")
        self.assertEqual(row["protein"], "beef")

    def test_last_cooked_is_the_one_field_the_tool_owns(self):
        self.assertTrue(pantry.record_cooked(self.hh, "meatloaf", "2026-08-03"))
        row = next(r for r in pantry.load_corpus(self.hh) if r["slug"] == "meatloaf")
        self.assertEqual(row["last cooked"], "2026-08-03")


# --------------------------------------------------------------------------- #

class TestWeek(Isolated):
    def test_a_week_survives_a_round_trip(self):
        w = pantry.Week(date="2026-08-03", nights=4, guests=1.5, risk="high")
        w.meals = pantry.propose(self.hh, 4, 1.5, "high")
        w.declined = ["chili"]
        w.feedback = {"meatloaf": {"outcome": "kept", "by": "Sam"}}
        pantry.write_week(self.hh, w)
        back = pantry.read_week(self.hh, "2026-08-03")
        self.assertEqual(back.nights, 4)
        self.assertEqual(back.guests, 1.5)
        self.assertEqual(back.risk, "high")
        self.assertEqual(back.declined, ["chili"])
        self.assertEqual(back.feedback["meatloaf"], {"outcome": "kept", "by": "Sam"})
        self.assertEqual([m.title for m in back.meals], [m.title for m in w.meals])

    def test_variants_survive_a_round_trip(self):
        w = pantry.Week(date="2026-08-03")
        w.meals = [pantry.Meal(slug="chicken-noodle-soup", title="Chicken noodle soup",
                               variant="Whole young chicken", reason="x")]
        pantry.write_week(self.hh, w)
        self.assertEqual(pantry.read_week(self.hh, "2026-08-03").meals[0].variant, "Whole young chicken")


class TestFeedbackLoop(Isolated):
    """The write that makes the whole thing work. Without it every recipe looks
    equally forgotten forever and the ranker has nothing to rank on."""

    def setUp(self):
        super().setUp()
        self.w = pantry.Week(date="2026-07-27")
        self.w.meals = [
            pantry.Meal(slug="meatloaf", title="Meatloaf", reason="x"),
            pantry.Meal(slug="chili", title="Chili", reason="x"),
            pantry.Meal(slug="sheet-pan-chicken-fajitas", title="Sheet pan chicken fajitas",
                        reason="x", candidate=True),
        ]

    def test_a_cooked_corpus_recipe_gets_its_date(self):
        self.w.feedback = {"meatloaf": {"outcome": "kept", "by": "Michael"}}
        pantry.apply_feedback(self.hh, self.w)
        row = next(r for r in pantry.load_corpus(self.hh) if r["slug"] == "meatloaf")
        self.assertEqual(row["last cooked"], "2026-07-27")

    def test_a_kept_candidate_is_promoted(self):
        self.w.feedback = {"sheet-pan-chicken-fajitas": {"outcome": "kept", "by": "Sam"}}
        pantry.apply_feedback(self.hh, self.w)
        self.assertIn("sheet-pan-chicken-fajitas", {r["slug"] for r in pantry.load_corpus(self.hh)})

    def test_not_cooked_writes_nothing(self):
        self.w.feedback = {"chili": {"outcome": "not cooked", "by": ""}}
        pantry.apply_feedback(self.hh, self.w)
        row = next(r for r in pantry.load_corpus(self.hh) if r["slug"] == "chili")
        self.assertEqual(row["last cooked"].strip(), "")

    def test_attribution_reaches_the_log(self):
        self.w.feedback = {"meatloaf": {"outcome": "kept", "by": "Sam"}}
        pantry.apply_feedback(self.hh, self.w)
        rec = pantry.decisions(self.hh, {"feedback_applied"})[-1]
        self.assertEqual(rec["by"], "Sam")


class TestDecisionLog(Isolated):
    def test_a_proposal_records_its_reasons(self):
        pantry.propose(self.hh, 3, 0, "normal")
        rec = pantry.decisions(self.hh, {"proposed"})[-1]
        self.assertEqual(len(rec["added"]), 3)
        self.assertTrue(all(a["reason"] for a in rec["added"]))

    def test_gap_filling_distinguishes_kept_from_added(self):
        first = pantry.propose(self.hh, 3, 0, "normal")
        pantry.propose(self.hh, 4, 0, "normal", keep=first)
        rec = pantry.decisions(self.hh, {"proposed"})[-1]
        self.assertEqual(len(rec["kept"]), 3)
        self.assertEqual(len(rec["added"]), 1)

    def test_logging_never_breaks_a_session(self):
        # A household rooted somewhere that does not exist, so the append
        # fails at the filesystem. The log is evidence, not a dependency.
        pantry.log(Household(root=self.tmp / "nope"), "proposed", x=1)


# --------------------------------------------------------------------------- #

class TestRanker(Isolated):
    def test_it_fills_the_week(self):
        self.assertEqual(len(pantry.propose(self.hh, 5, 0, "normal")), 5)

    def test_kept_meals_are_never_re_rolled(self):
        first = pantry.propose(self.hh, 4, 0, "normal")
        again = pantry.propose(self.hh, 5, 0, "normal", keep=first)
        self.assertEqual([m.slug for m in again[:4]], [m.slug for m in first])

    def test_a_declined_meal_does_not_come_back(self):
        week = pantry.propose(self.hh, 5, 0, "normal")
        dropped = week[2].slug
        kept = [m for m in week if m.slug != dropped]
        again = pantry.propose(self.hh, 5, 0, "normal", keep=kept, avoid={dropped})
        self.assertNotIn(dropped, [m.slug for m in again])

    def test_the_risk_dial_reserves_candidate_slots(self):
        """Candidates lose every head-to-head by design, so a score nudge leaves
        the dial doing nothing at all."""
        counts = {r: sum(1 for m in pantry.propose(self.hh, 5, 0, r) if m.candidate)
                  for r in ("low", "normal", "high")}
        self.assertEqual(counts["low"], 0)
        self.assertEqual(counts["normal"], 1)
        self.assertEqual(counts["high"], 2)

    def test_reasons_are_not_all_the_same_sentence(self):
        reasons = [m.reason for m in pantry.propose(self.hh, 5, 0, "normal")]
        self.assertEqual(len(set(reasons)), 5,
                         "five true sentences that are all the same sentence are no reasons")

    def test_no_recency_is_claimed_without_a_date(self):
        """The corpus has no last-cooked dates. Saying 'not cooked since March'
        anyway is the invention this project exists to avoid."""
        for m in pantry.propose(self.hh, 6, 0, "high"):
            self.assertNotRegex(m.reason, r"\b(months?|days) ago|not cooked in")

    def test_staleness_is_used_once_a_date_exists(self):
        old = (dt.date.today() - dt.timedelta(days=200)).isoformat()
        pantry.record_cooked(self.hh, "tuna-melt", old)
        reasons = {m.slug: m.reason for m in pantry.propose(self.hh, 8, 0, "low")}
        self.assertIn("months", reasons.get("tuna-melt", ""))

    def test_a_candidate_never_claims_corpus_membership(self):
        for m in pantry.propose(self.hh, 6, 0, "high"):
            if m.candidate:
                self.assertTrue(m.reason.startswith("new here"))
                self.assertNotIn("in the corpus", m.reason)

    def test_the_week_gets_enough_low_active_nights(self):
        meals = pantry.propose(self.hh, 5, 0, "normal")
        self.assertGreaterEqual(sum(1 for m in meals if m.active == "low"), 2)

    def test_a_flopped_candidate_is_not_proposed_again(self):
        pantry.record_flop(self.hh, "sheet-pan-chicken-fajitas", "2026-07-27")
        slugs = [m.slug for m in pantry.propose(self.hh, 8, 0, "high")]
        self.assertNotIn("sheet-pan-chicken-fajitas", slugs)


class TestFileIndex(Isolated):
    def test_the_index_and_the_store_may_disagree_on_a_name(self):
        """`corpus.md` says *Crock pot Italian beef sandwiches*; the file is
        `crock-pot-italian-beef.md`. Resolved by each file's own title."""
        m = pantry.Meal(slug="crock-pot-italian-beef-sandwiches", title="x")
        self.assertEqual(m.file(self.hh), "crock-pot-italian-beef")
        self.assertTrue(m.has_file(self.hh))

    def test_every_planned_meal_can_find_its_ingredients(self):
        missing = [r["recipe"] for r in pantry.load_corpus(self.hh) + pantry.load_candidates(self.hh)
                   if not pantry.recipe_file(self.hh, r["slug"]).exists()]
        self.assertEqual(missing, [])


class TestMembers(Isolated):
    def test_members_come_off_the_profile(self):
        self.assertEqual(pantry.load_members(self.hh), ["Michael", "Sam"])


class TestDemoIsolation(unittest.TestCase):
    """A hosted deployment serves `demo/`. These are the tests that keep the real
    household out of it — checked rather than trusted, because the failure is
    silent, public, and about a family rather than a bug."""

    def setUp(self):
        import app
        self.app = app

    def test_the_private_files_are_overridden(self):
        for name in ("profile.md", "candidates.md", "corpus.md"):
            with self.subTest(file=name):
                self.assertEqual(self.app._demo_source(name), REAL / "demo" / name)

    def test_public_recipe_data_is_shared_not_duplicated(self):
        """`items.md` and `recipes/` are published recipes with nothing private
        in them. Copying them would only let them drift."""
        self.assertEqual(self.app._demo_source("items.md"), REAL / "items.md")

    def test_no_household_identity_reaches_the_demo_profile(self):
        text = (REAL / "demo" / "profile.md").read_text()
        real_members = pantry.load_members(household.here())
        self.assertTrue(real_members, "the real profile should name its members")
        for name in real_members:
            self.assertNotRegex(text, rf"\b{name}\b")
        for detail in ("peanut", "3-year-old", "1-year-old"):
            self.assertNotIn(detail, text.lower())

    def test_the_demo_household_names_itself(self):
        text = (REAL / "demo" / "profile.md").read_text()
        self.assertIn("invented", text.lower(),
                      "a fabricated profile has to say it is fabricated")

    def test_the_handwritten_family_recipe_is_not_in_the_demo_corpus(self):
        """Every other recipe came off a public site. The beef dip is a
        photograph of someone's recipe card."""
        self.assertNotIn("Beef dip", (REAL / "demo" / "corpus.md").read_text())

    def test_the_demo_corpus_says_its_history_is_invented(self):
        self.assertIn("none of it is real", (REAL / "demo" / "corpus.md").read_text())


if __name__ == "__main__":
    unittest.main(verbosity=2)
