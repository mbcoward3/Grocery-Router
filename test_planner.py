#!/usr/bin/env python3
"""Tests for the model planner.

    python3 test_planner.py

**No key and no network.** Every test here drives `planner/model.py` through its
`client` seam with a canned reply, which is the only way to test the paths that
matter: what happens when the model names a recipe that does not exist, claims a
recency no date supports, proposes something the household is allergic to, or
does not answer at all. Those are the paths a real key would exercise least and
that have to work most.

The rule the whole file is checking, stated once: **the model selects and
explains, and everything else is read off the corpus.** A pick that cannot be
resolved to a real row is dropped rather than repaired, and the ranker finishes
the week.
"""

import datetime as dt
import json
import os
import shutil
import tempfile
import unittest
from pathlib import Path

import pantry
import planner
from planner import constraints
from planner import model as model_planner

REAL = Path(__file__).resolve().parent


def reply(meals, note="1 candidate, because the corpus is small.", **extra):
    """A model reply in the shape the contract asks for: prose, then the block.

    The prose is deliberately not empty. Taking the *last* json block is a real
    behaviour and a reply with only a block would not test it.
    """
    body = {"note": note, "coupling": "", "gaps": "", "meals": meals}
    body.update(extra)
    return ("Here is the week.\n\n"
            "    Something                6 AE   low active   [because]\n\n"
            "Effort mix: plenty of low-active nights.\n\n"
            "```json\n" + json.dumps(body, indent=2) + "\n```\n")


def pick(slug, reason="the only fish this week, and it has been out of rotation"):
    return {"slug": slug, "reason": reason}


class Isolated(unittest.TestCase):
    """A scratch copy of the household's files, and a pinned planner selection."""

    def setUp(self):
        self._env = {k: os.environ.get(k) for k in
                     ("PANTRY_PLANNER", "ANTHROPIC_API_KEY", "PANTRY_MODEL")}
        os.environ["PANTRY_PLANNER"] = "ranker"
        os.environ.pop("ANTHROPIC_API_KEY", None)

        self.tmp = Path(tempfile.mkdtemp())
        for name in ("corpus.md", "candidates.md", "profile.md"):
            shutil.copy(REAL / name, self.tmp / name)
        shutil.copytree(REAL / "recipes", self.tmp / "recipes")
        self._saved = {k: getattr(pantry, k) for k in
                       ("ROOT", "CORPUS", "CANDIDATES", "PROFILE", "WEEKS", "CACHE",
                        "DECISIONS")}
        pantry.ROOT = self.tmp
        pantry.CORPUS = self.tmp / "corpus.md"
        pantry.CANDIDATES = self.tmp / "candidates.md"
        pantry.PROFILE = self.tmp / "profile.md"
        pantry.WEEKS = self.tmp / "weeks"
        pantry.CACHE = self.tmp / ".cache"
        pantry.DECISIONS = self.tmp / "decisions.jsonl"
        pantry._FILE_INDEX = None

    def tearDown(self):
        for k, v in self._saved.items():
            setattr(pantry, k, v)
        for k, v in self._env.items():
            if v is None:
                os.environ.pop(k, None)
            else:
                os.environ[k] = v
        pantry._FILE_INDEX = None
        shutil.rmtree(self.tmp, ignore_errors=True)

    # -- fixtures the scratch copy makes cheap ------------------------------- #

    def make_peanut(self, slug="chili"):
        """Rewrite one capture's recorded verdict. Nothing in the real corpus
        contains peanut - `profile.md` says so - so the constraint can only be
        tested against a household that has one."""
        path = pantry.recipe_file(slug)
        text = path.read_text().replace("peanut:   none seen", "peanut:   CONTAINS PEANUT")
        path.write_text(text)
        return slug

    def make_high_active(self, n=4):
        """Flip the first `n` med rows to high. The corpus has no high-active
        row at all, which is why the ceiling has never been able to bite."""
        lines = pantry.CORPUS.read_text().splitlines()
        flipped = 0
        for i, line in enumerate(lines):
            if flipped < n and "| med |" in line:
                lines[i] = line.replace("| med |", "| high |", 1)
                flipped += 1
        pantry.CORPUS.write_text("\n".join(lines) + "\n")
        return [r["slug"] for r in pantry.load_corpus() if r.get("active") == "high"]

    def slugs(self, n=5):
        return [r["slug"] for r in pantry.load_corpus()][:n]


# --------------------------------------------------------------------------- #

class TestSelection(unittest.TestCase):
    """Which implementation runs, and why. No files needed."""

    def setUp(self):
        self._env = {k: os.environ.get(k) for k in
                     ("PANTRY_PLANNER", "ANTHROPIC_API_KEY")}

    def tearDown(self):
        for k, v in self._env.items():
            os.environ.pop(k, None) if v is None else os.environ.__setitem__(k, v)

    def set(self, want=None, key=None):
        os.environ.pop("PANTRY_PLANNER", None) if want is None else \
            os.environ.__setitem__("PANTRY_PLANNER", want)
        os.environ.pop("ANTHROPIC_API_KEY", None) if key is None else \
            os.environ.__setitem__("ANTHROPIC_API_KEY", key)

    def test_no_key_means_the_ranker(self):
        self.set()
        self.assertEqual(planner.which(), "ranker")

    def test_a_key_is_the_whole_configuration(self):
        """The hosted demo has no key and gets a real week; a laptop with a key
        gets the better planner. Neither needs a flag."""
        self.set(key="sk-test")
        self.assertEqual(planner.which(), "model")

    def test_the_env_can_pin_the_ranker_even_with_a_key(self):
        self.set(want="ranker", key="sk-test")
        self.assertEqual(planner.which(), "ranker")

    def test_an_explicit_argument_beats_the_env(self):
        self.set(want="ranker", key="sk-test")
        self.assertEqual(planner.which("model"), "model")

    def test_asking_for_the_model_without_a_key_still_says_model(self):
        """So that `propose()` fails loudly and logs it. Silently handing back
        the ranker every week tells a household that asked for a model nothing
        at all."""
        self.set(want="model")
        self.assertEqual(planner.which(), "model")

    def test_a_typo_falls_back_to_deciding_on_the_key(self):
        self.set(want="modle")
        self.assertEqual(planner.which(), "ranker")
        self.set(want="modle", key="sk-test")
        self.assertEqual(planner.which(), "model")


class TestWhatTheModelIsGiven(Isolated):
    """The trap this project has a receipt for: asked to reason about ingredient
    coupling from an index with no ingredients, a model invented the coupling and
    then chose a recipe because of it. The fix was never *trust the model less* -
    it was give it only what the corpus contains."""

    def test_the_catalogue_carries_no_ingredients(self):
        text = model_planner.catalogue(pantry.load_corpus(), dt.date(2026, 8, 6))
        for line in text.splitlines():
            self.assertNotIn("lb ", line)
            self.assertNotIn("tsp", line)
            self.assertNotIn("clove", line)

    def test_a_missing_date_reads_unknown_not_blank_and_not_zero(self):
        text = model_planner.catalogue(pantry.load_corpus(), dt.date(2026, 8, 6))
        rows = [l for l in text.splitlines() if l.startswith("| ") and "---" not in l][1:]
        self.assertTrue(rows)
        for row in rows:
            self.assertTrue(row.rstrip().endswith("unknown |"), row)

    def test_a_date_is_turned_into_a_number_here_not_there(self):
        """Date arithmetic is not asked for. The column is computed so that a
        recency claim can be checked against something real."""
        pantry.record_cooked("chili", "2026-06-06")
        text = model_planner.catalogue(pantry.load_corpus(), dt.date(2026, 8, 6))
        line = [l for l in text.splitlines() if l.startswith("| chili |")][0]
        self.assertTrue(line.rstrip().endswith("| 61 |"), line)

    def test_the_prompt_carries_the_profile_and_both_catalogues(self):
        prompt = model_planner.build_prompt(
            pantry.load_corpus(), pantry.load_candidates(),
            pantry.PROFILE.read_text(), 5, 0.0, "normal", [], set(),
            dt.date(2026, 8, 6))
        self.assertIn("Catalogue — corpus", prompt)
        self.assertIn("Catalogue — candidates", prompt)
        self.assertIn("Hard constraints", prompt)          # from profile.md
        self.assertIn("slug from the catalogue", prompt)   # the contract


class TestTheModelMayNotInvent(Isolated):
    """Every one of these is the same failure in a different coat: a plausible
    value where there should have been a gap."""

    def plan(self, meals, **kw):
        return model_planner.plan(client=lambda p: reply(meals), **kw)

    def test_a_recipe_that_does_not_exist_is_dropped(self):
        """`1 lb chicken breast because soup usually has chicken` is the failure
        this project exists to avoid. A name is not a recipe."""
        got = self.plan([pick("thai-peanut-noodles"), pick(self.slugs()[0])])
        self.assertEqual([m.slug for m in got.meals], [self.slugs()[0]])
        self.assertIn("thai-peanut-noodles", " ".join(got.dropped))

    def test_a_near_miss_is_not_resolved_to_its_neighbour(self):
        """`onion powder` resolving to `onion` across thirteen lines put a fresh
        onion in the cart for a teaspoon of spice. A silent mis-merge beats a
        loud gap, and that is backwards."""
        got = self.plan([pick("chili-con-carne")])       # the corpus has `chili`
        self.assertEqual(got.meals, [])
        self.assertIn("chili-con-carne", " ".join(got.dropped))

    def test_a_recency_claim_with_no_date_behind_it_is_dropped(self):
        got = self.plan([pick("chili", "you haven't made this since March")])
        self.assertEqual(got.meals, [])
        self.assertIn("recency", " ".join(got.dropped))

    def test_the_same_claim_is_fine_once_a_date_supports_it(self):
        pantry.record_cooked("chili", "2025-03-04")
        got = model_planner.plan(today=dt.date(2026, 8, 6),
                                 client=lambda p: reply(
                                     [pick("chili", "you haven't made this since March")]))
        self.assertEqual([m.slug for m in got.meals], ["chili"])

    def test_a_pick_with_no_reason_is_dropped(self):
        """The reason is the product. A pick without one is a suggestion, and
        the ranker writes better ones."""
        got = self.plan([{"slug": "chili", "reason": "  "}])
        self.assertEqual(got.meals, [])
        self.assertIn("no reason", " ".join(got.dropped))

    def test_facts_come_off_the_corpus_row_not_the_reply(self):
        got = model_planner.plan(client=lambda p: reply(
            [{"slug": "chili", "reason": "the only beef this week",
              "protein": "tofu", "yield": "40 AE", "active": "high"}]))
        meal = got.meals[0]
        self.assertEqual(meal.protein, "beef")
        self.assertEqual(meal.yield_, "4 AE")
        self.assertEqual(meal.active, "low")

    def test_membership_is_read_here_never_claimed_there(self):
        """A candidate inherited a corpus recipe's reason once and claimed
        membership it did not have. The flag is not the model's to set."""
        cand = pantry.load_candidates()[0]["slug"]
        got = model_planner.plan(client=lambda p: reply(
            [{"slug": cand, "reason": "worth a try", "candidate": False}]))
        self.assertTrue(got.meals[0].candidate)

    def test_a_flopped_candidate_is_not_in_the_catalogue_at_all(self):
        cand = pantry.load_candidates()[0]["slug"]
        pantry.record_flop(cand, "2026-08-01", "nobody ate it")
        got = model_planner.plan(client=lambda p: reply([pick(cand)]))
        self.assertEqual(got.meals, [])


class TestHardConstraints(Isolated):
    """`profile.md`: *"Not preferences. The planner must never violate these."*
    Until now nothing checked."""

    def test_a_peanut_recipe_never_reaches_the_week(self):
        self.make_peanut("chili")
        got = model_planner.plan(client=lambda p: reply([pick("chili")]))
        self.assertEqual(got.meals, [])
        self.assertIn("peanut", " ".join(got.dropped))

    def test_the_check_reads_the_capture_rather_than_scanning_again(self):
        self.assertEqual(constraints.peanut_verdict("chili"), "none seen")
        self.make_peanut("chili")
        self.assertEqual(constraints.peanut_verdict("chili"), "contains peanut")

    def test_check_label_is_not_a_violation(self):
        """The profile is explicit: trace risk and shared facilities are
        acceptable, and this filters the recipe rather than the pantry. The
        teriyaki is in the corpus because this household has eaten it."""
        path = pantry.recipe_file("3-ingredient-teriyaki-chicken")
        path.write_text(path.read_text().replace("peanut:   none seen",
                                                 "peanut:   check label"))
        got = model_planner.plan(client=lambda p: reply(
            [pick("3-ingredient-teriyaki-chicken")]))
        self.assertEqual(len(got.meals), 1)

    def test_an_unscanned_recipe_is_reported_rather_than_blocked(self):
        """Unknown is not the extreme. A recipe with no last-cooked date once
        scored as maximally stale for exactly this reason."""
        self.assertEqual(constraints.peanut_verdict("sausage-and-peppers"), "")
        got = model_planner.plan(client=lambda p: reply([pick("sausage-and-peppers")]))
        self.assertEqual(len(got.meals), 1)
        self.assertIn("no peanut scan on record", " ".join(got.warnings))

    def test_too_many_high_active_cooks_is_surfaced(self):
        highs = self.make_high_active(4)
        got = model_planner.plan(nights=4, client=lambda p: reply(
            [pick(s, "a reason") for s in highs]))
        self.assertEqual(len(got.meals), 4)
        self.assertIn("weeknight ceiling", " ".join(got.warnings))

    def test_two_high_active_cooks_is_the_weekend_and_is_fine(self):
        highs = self.make_high_active(2)
        got = model_planner.plan(nights=5, client=lambda p: reply(
            [pick(s, "a reason") for s in highs]))
        self.assertEqual(constraints.check_week(got.meals), [])

    def test_the_ceiling_is_not_enforced_by_deleting_a_meal(self):
        """Which of four high-active cooks is the wrong one is the household's
        call. Dropping the fourth silently would hide that the week is
        unrunnable, which is the more expensive failure."""
        highs = self.make_high_active(4)
        got = model_planner.plan(nights=4, client=lambda p: reply(
            [pick(s, "a reason") for s in highs]))
        self.assertEqual(len(got.meals), len(highs))


class TestTheWeekIsAlwaysWhole(Isolated):
    """Dropping a bad pick costs a good reason. It must never cost a dinner."""

    def use(self, client, **kw):
        return pantry.propose(planner="model", client=client, **kw)

    def test_a_short_model_week_is_topped_up_by_the_ranker(self):
        got = self.use(lambda p: reply([pick(self.slugs()[0])]), nights=5)
        self.assertEqual(len(got), 5)
        self.assertEqual(len({m.slug for m in got}), 5)

    def test_the_top_up_is_recorded_as_such(self):
        self.use(lambda p: reply([pick(self.slugs()[0])]), nights=5)
        last = pantry.last_proposal()
        self.assertEqual(last["planner"], "model")
        self.assertEqual(len(last["topped_up"]), 4)

    def test_kept_meals_are_left_alone(self):
        first = pantry.rank(3, 0, "normal")
        got = self.use(lambda p: reply([pick(self.slugs(6)[5])]), nights=5, keep=first)
        self.assertEqual([m.slug for m in got[:3]], [m.slug for m in first])

    def test_a_declined_meal_does_not_come_back_through_the_model(self):
        dropped = self.slugs()[0]
        got = self.use(lambda p: reply([pick(dropped), pick(self.slugs()[1])]),
                       nights=2, avoid={dropped})
        self.assertNotIn(dropped, [m.slug for m in got])

    def test_more_picks_than_nights_are_truncated(self):
        got = model_planner.plan(nights=2, client=lambda p: reply(
            [pick(s, "a reason") for s in self.slugs(5)]))
        self.assertEqual(len(got.meals), 2)

    def test_the_same_slug_twice_lands_once(self):
        got = model_planner.plan(nights=3, client=lambda p: reply(
            [pick("chili"), pick("chili"), pick(self.slugs()[0])]))
        self.assertEqual(len([m for m in got.meals if m.slug == "chili"]), 1)


class TestFallingBackIsNeverSilent(Isolated):
    """Every failure between here and a parsed week ends the same way. What must
    not happen is that it ends quietly."""

    def fails(self, client, nights=5):
        got = pantry.propose(nights, planner="model", client=client)
        self.assertEqual(len(got), nights)
        return pantry.last_proposal()

    def test_no_key_falls_back_and_says_so(self):
        os.environ.pop("ANTHROPIC_API_KEY", None)
        last = self.fails(None)                # no stub: the real client, no key
        self.assertEqual(last["planner"], "ranker")
        self.assertEqual(last["asked"], "model")
        self.assertIn("ANTHROPIC_API_KEY", last["fallback"])

    def test_an_api_error_falls_back_and_says_so(self):
        def boom(prompt):
            raise model_planner.PlannerUnavailable("API error 529: overloaded")
        last = self.fails(boom)
        self.assertEqual(last["planner"], "ranker")
        self.assertIn("529", last["fallback"])

    def test_a_reply_with_no_json_block_falls_back(self):
        last = self.fails(lambda p: "Here is a lovely week, in prose only.")
        self.assertIn("no ```json block", last["fallback"])

    def test_a_malformed_json_block_falls_back(self):
        last = self.fails(lambda p: "```json\n{\"meals\": [oops}\n```")
        self.assertIn("did not parse", last["fallback"])

    def test_a_block_with_no_meals_array_falls_back(self):
        last = self.fails(lambda p: "```json\n{\"note\": \"hi\"}\n```")
        self.assertIn("`meals` array", last["fallback"])

    def test_a_week_of_pure_invention_falls_back_and_the_drops_are_kept(self):
        last = self.fails(lambda p: reply([pick("thai-peanut-noodles"),
                                           pick("kung-pao-anything")]))
        self.assertEqual(last["planner"], "ranker")
        self.assertIn("nothing that survived validation", last["fallback"])
        self.assertEqual(len(last["dropped"]), 2)

    def test_the_ranker_path_logs_no_apology(self):
        got = pantry.propose(5, planner="ranker")
        last = pantry.last_proposal()
        self.assertEqual(len(got), 5)
        self.assertEqual(last["planner"], "ranker")
        self.assertNotIn("fallback", last)

    def test_the_illustrative_block_is_not_the_answer(self):
        """A model that explains its output format before producing it emits two
        blocks. The last one is the week."""
        text = ("```json\n{\"meals\": [{\"slug\": \"<slug>\", \"reason\": \"<why>\"}]}\n```\n"
                + reply([pick("chili")]))
        got = model_planner.plan(client=lambda p: text)
        self.assertEqual([m.slug for m in got.meals], ["chili"])


class TestTheDecisionLog(Isolated):
    """`decisions.jsonl` is what lets a planner change be replayed against real
    history instead of argued about. A second planner that did not appear in it
    would have broken that."""

    def test_the_log_records_which_planner_produced_the_week(self):
        pantry.propose(3, planner="model", client=lambda p: reply(
            [pick(s, "a reason") for s in self.slugs(3)]))
        last = pantry.last_proposal()
        self.assertEqual(last["planner"], "model")
        self.assertEqual(len(last["added"]), 3)

    def test_the_models_own_notes_survive_into_the_log(self):
        pantry.propose(3, planner="model", client=lambda p: reply(
            [pick(s, "a reason") for s in self.slugs(3)],
            note="0 candidates: the corpus is wide enough this week",
            coupling="the stew and the pot roast share the carrots",
            gaps="no last-cooked dates, so nothing is surfaced on recency"))
        last = pantry.last_proposal()
        self.assertIn("0 candidates", last["note"])
        self.assertIn("carrots", last["coupling"])
        self.assertIn("last-cooked", last["gaps"])

    def test_one_proposal_is_one_log_entry(self):
        """`rank()` deliberately does not log. Two entries for one proposal
        would corrupt every count read back off the log - including the accept
        rate on the session's metrics strip."""
        pantry.propose(5, planner="model", client=lambda p: reply([pick("chili")]))
        self.assertEqual(len(pantry.decisions({"proposed"})), 1)


if __name__ == "__main__":
    unittest.main(verbosity=2)
