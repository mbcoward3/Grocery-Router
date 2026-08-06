#!/usr/bin/env python3
"""Tests for reading the decision log back.

    python3 test_review.py

`decisions.jsonl` was built because a decision that was not recorded cannot be
recovered, and then nothing read it for the life of the project. These are the
tests for the reading — over a log this file writes itself, one decision at a
time, in the shapes the session actually produces.
"""

import shutil
import tempfile
import unittest
from pathlib import Path

import pantry
import review

REAL = Path(__file__).resolve().parent


class Logged(unittest.TestCase):
    def setUp(self):
        self.tmp = Path(tempfile.mkdtemp())
        for name in ("corpus.md", "candidates.md", "sides.md", "profile.md"):
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

    def tearDown(self):
        for k, v in self._saved.items():
            setattr(pantry, k, v)
        pantry._FILE_INDEX = None
        shutil.rmtree(self.tmp, ignore_errors=True)

    def propose(self, week, *, added, planner="ranker", **extra):
        """One `proposed` record, in the shape `_log_proposal` writes."""
        pantry.log("proposed", week=week, nights=len(added), guests=0, risk="normal",
                   kept=[], planner=planner,
                   added=[{"recipe": slug, "reason": f"because {kind}", "kind": kind,
                           "candidate": cand, "protein": protein, "cuisine": ""}
                          for slug, kind, cand, protein in added], **extra)

    def drop(self, week, slug, kind):
        pantry.log("drop", week=week, recipe=slug, reason_shown="x",
                   reason_kind=kind, candidate=False)

    def kept(self, week, slug):
        pantry.log("feedback_applied", recipe=slug, outcome="kept", by="Sam", week=week)


# --------------------------------------------------------------------------- #

class TestWhichReasonsLand(Logged):
    def test_a_reason_that_gets_dropped_scores_lower_than_one_that_does_not(self):
        """The product claim, testable for the first time. If `stale` and `plain`
        score the same, the reason is not doing any work."""
        self.propose("2026-06-01", added=[("a", "stale", False, "beef"),
                                          ("b", "plain", False, "pork")])
        self.propose("2026-06-08", added=[("c", "stale", False, "beef"),
                                          ("d", "plain", False, "pork")])
        self.drop("2026-06-01", "b", "plain")
        self.drop("2026-06-08", "d", "plain")
        by_kind = {r["kind"]: r for r in review.reasons()}
        self.assertEqual(by_kind["stale"]["accept_rate"], 100)
        self.assertEqual(by_kind["plain"]["accept_rate"], 0)

    def test_worst_first(self):
        self.propose("2026-06-01", added=[("a", "stale", False, "beef"),
                                          ("b", "plain", False, "pork")])
        self.drop("2026-06-01", "b", "plain")
        self.assertEqual(review.reasons()[0]["kind"], "plain")

    def test_dropped_and_kept_are_different_denominators(self):
        """A meal neither dropped nor cooked is not evidence either way, and
        rolling them into one number invents a verdict for a week nobody
        finished."""
        self.propose("2026-06-01", added=[("a", "stale", False, "beef"),
                                          ("b", "stale", False, "pork"),
                                          ("c", "stale", False, "fish")])
        self.kept("2026-06-01", "a")
        row = {r["kind"]: r for r in review.reasons()}["stale"]
        self.assertEqual((row["offered"], row["dropped"], row["kept"]), (3, 0, 1))
        self.assertEqual(row["accept_rate"], 100)

    def test_the_same_recipe_twice_is_two_offers(self):
        """The right denominator for an accept rate is offers, not recipes."""
        self.propose("2026-06-01", added=[("a", "stale", False, "beef")])
        self.propose("2026-06-08", added=[("a", "stale", False, "beef")])
        self.assertEqual({r["kind"]: r for r in review.reasons()}["stale"]["offered"], 2)

    def test_a_drop_with_no_recorded_kind_is_joined_back_to_its_proposal(self):
        """Older log lines predate `reason_kind` on a drop. The log is
        append-only, so the join is how history stays readable."""
        self.propose("2026-06-01", added=[("a", "yield", False, "beef")])
        pantry.log("drop", week="2026-06-01", recipe="a", reason_shown="x")
        self.assertEqual({r["kind"]: r for r in review.reasons()}["yield"]["dropped"], 1)

    def test_an_empty_log_is_empty_not_broken(self):
        self.assertEqual(review.reasons(), [])
        self.assertEqual(review.breadth(), [])
        self.assertEqual(review.summary()["distinct_recipes"], 0)


class TestBreadth(Logged):
    def test_new_recipes_are_what_counts_not_meals_served(self):
        """Five dinners a week is five dinners a week whether they are the same
        five every time or never the same twice."""
        for week in ("2026-06-01", "2026-06-08", "2026-06-15"):
            self.propose(week, added=[("a", "stale", False, "beef"),
                                      ("b", "stale", False, "pork")])
        rows = review.breadth()
        self.assertEqual([r["offered"] for r in rows], [2, 2, 2])
        self.assertEqual([r["new"] for r in rows], [2, 0, 0])
        self.assertEqual(rows[-1]["distinct_so_far"], 2)

    def test_a_widening_repertoire_shows_as_widening(self):
        self.propose("2026-06-01", added=[("a", "stale", False, "beef")])
        self.propose("2026-06-08", added=[("b", "stale", False, "pork")])
        self.propose("2026-06-15", added=[("c", "stale", False, "fish")])
        self.assertEqual([r["distinct_so_far"] for r in review.breadth()], [1, 2, 3])

    def test_it_groups_by_the_week_planned_not_the_day_it_ran(self):
        """Proposing next week on a Sunday is a normal thing to do, and grouping
        by wall clock files it under the wrong one."""
        self.propose("2026-06-01", added=[("a", "stale", False, "beef")])
        self.propose("2026-06-08", added=[("b", "stale", False, "pork")])
        self.assertEqual([r["week"] for r in review.breadth()],
                         ["2026-06-01", "2026-06-08"])

    def test_protein_and_cuisine_spread_are_counted(self):
        self.propose("2026-06-01", added=[("a", "stale", False, "beef"),
                                          ("b", "stale", False, "pork"),
                                          ("c", "stale", False, "beef")])
        self.assertEqual(review.breadth()[0]["proteins"], 2)


class TestThePlannerComparison(Logged):
    def test_the_two_implementations_are_compared_on_real_weeks(self):
        self.propose("2026-06-01", planner="ranker",
                     added=[("a", "stale", False, "beef"), ("b", "plain", False, "pork")])
        self.propose("2026-06-08", planner="model",
                     added=[("c", "model", False, "beef"), ("d", "model", False, "pork")])
        self.drop("2026-06-01", "b", "plain")
        by = {r["planner"]: r for r in review.planners()["by_planner"]}
        self.assertEqual(by["ranker"]["accept_rate"], 50)
        self.assertEqual(by["model"]["accept_rate"], 100)

    def test_a_quiet_fallback_is_visible(self):
        """A model that degraded to the ranker for a month is the real failure,
        and this is where it would show."""
        self.propose("2026-06-01", planner="ranker", asked="model",
                     fallback="API error 529: overloaded",
                     added=[("a", "stale", False, "beef")])
        out = review.planners()
        self.assertEqual(out["asked_for_a_model"], 1)
        self.assertEqual(out["fell_back"], 1)
        self.assertIn("529", out["why"][0][0])

    def test_no_model_week_yet_is_itself_the_answer(self):
        self.propose("2026-06-01", added=[("a", "stale", False, "beef")])
        by = {r["planner"] for r in review.planners()["by_planner"]}
        self.assertEqual(by, {"ranker"})


class TestAcquisitionShowsUp(Logged):
    def test_searching_and_pasting_are_told_apart(self):
        pantry.log("acquired", recipe="a", source="https://x.test/a/", found_by="acquire")
        pantry.log("acquired", recipe="b", source="https://y.test/b/", found_by="paste")
        out = review.acquisition()
        self.assertEqual(out["total"], 2)
        self.assertEqual(out["by_route"], {"acquire": 1, "paste": 1})

    def test_nothing_acquired_has_been_proven_yet(self):
        pantry.log("acquired", recipe="a", source="https://x.test/a/", found_by="acquire")
        self.assertEqual(review.acquisition()["promoted"], 0)


class TestTheLogStaysTrue(Logged):
    def test_a_payload_may_not_overwrite_the_decision_type(self):
        """`kind` is an easy key to reach for, and the drop route reached for it.
        Silently replacing the decision type would corrupt every count read back
        off the log."""
        with self.assertRaises(ValueError):
            pantry.log("drop", kind="stale")
        with self.assertRaises(ValueError):
            pantry.log("drop", at="whenever")

    def test_the_kind_a_meal_was_offered_under_is_recoverable(self):
        self.propose("2026-06-01", added=[("chili", "stale", False, "beef")])
        self.assertEqual(review.kind_of("chili"), "stale")
        self.assertEqual(review.kind_of("never-offered"), "")

    def test_the_most_recent_offer_wins(self):
        self.propose("2026-06-01", added=[("chili", "stale", False, "beef")])
        self.propose("2026-06-08", added=[("chili", "protein", False, "beef")])
        self.assertEqual(review.kind_of("chili"), "protein")


class TestTheRankerRecordsItsKinds(Logged):
    """End to end: the ranker has to actually put kinds in the log, or every
    number above is computed off an empty column."""

    def test_a_real_proposal_carries_reason_kinds(self):
        pantry.propose(5, planner="ranker", week="2026-06-01")
        kinds = {o["kind"] for o in review.offers()}
        self.assertTrue(kinds)
        self.assertNotIn("", kinds)

    def test_the_kinds_are_the_ones_review_can_explain(self):
        pantry.propose(5, planner="ranker", week="2026-06-01")
        for offer in review.offers():
            self.assertIn(offer["kind"], review.KIND_MEANING, offer)


if __name__ == "__main__":
    unittest.main(verbosity=2)
