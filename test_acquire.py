#!/usr/bin/env python3
"""Tests for acquisition, and for the door it writes through.

    python3 test_acquire.py

**No network.** Search and fetch are both stubbed, so these run in CI, in a
sandbox, and on a plane. That is not only convenience: the interesting cases here
are a search returning cake, a page with no recipe data on it, and a recipe with
peanuts in it, and waiting for the open web to serve those up on demand is not a
test.

Several of the fixtures are verbatim from a live run against the household's own
sources, and are labelled where they are. The blueberry muffin and the Thai curry
are real things nine recipe sites handed back when asked for `fish`.
"""

import json
import shutil
import tempfile
import unittest
from pathlib import Path

import acquire
import onboard
import pantry

REAL = Path(__file__).resolve().parent


def ingredients(*raw):
    return [onboard.parse_ingredient(line) for line in raw]


def recipe(title, *raw, yield_="4 AE", times=("prep 10 min",), status="complete",
           source="https://thecountrycook.net/x/"):
    return {"title": title, "source": source, "modality": "url",
            "ingredients": ingredients(*raw), "yield": yield_, "yield_note": None,
            "times": list(times), "passive": "oven", "questions": [],
            "capture_notes": [], "status": status, "instructions_text": ""}


# Real captures, trimmed. These are what the sources actually returned.
COD = recipe("Cod Fish in Tomato Sauce", "1.5 lb cod fillets", "1 can diced tomatoes",
             "2 Tbsp olive oil", "1 packet italian seasoning",
             source="https://thecountrycook.net/cod-in-tomato-sauce/")
MUFFIN = recipe("Blueberry Banana Muffins", "2 cups flour", "1 cup blueberries",
                "2 bananas", "1/2 cup sugar",
                source="https://thecountrycook.net/blueberry-banana-muffins/")
CURRY = recipe("Thai Massaman Beef Curry", "2 lb beef chuck", "1 can coconut milk",
               "½ cup roasted unsalted peanuts, coarsely chopped",
               source="https://chefjeanpierre.com/massaman-curry/")
TACOS = recipe("Ground Beef Tacos", "1 lb ground beef", "1 packet taco seasoning",
               "8 tortillas", source="https://lilluna.com/ground-beef-tacos/")
NO_DATA = recipe("15+ Sweet Peach Recipes For Summer", status="failed", yield_=None,
                 source="https://southernbite.com/sweet-peach-recipes/")
SLOW = recipe("Crock Pot Pork Chops", "4 pork chops", "1 can cream of mushroom soup",
              "1 packet onion soup mix", times=("prep 5 min", "cook 6 hr"),
              source="https://thecountrycook.net/crock-pot-pork-chops/")
FUSSY = recipe("Swordfish Milanese", "2 swordfish steaks", "1 cup breadcrumbs",
               "2 eggs", times=("prep 45 min",),
               source="https://chefjeanpierre.com/swordfish-milanese/")


class Isolated(unittest.TestCase):
    def setUp(self):
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
        pantry._FILE_INDEX = None
        shutil.rmtree(self.tmp, ignore_errors=True)

    def stub(self, hits, pages):
        """Point search and fetch at fixtures. Returns a call log."""
        calls = {"search": [], "fetch": []}
        # Captured *before* the reassignment below. Reading them back off the
        # module afterwards restores the stub over the stub and leaks it into
        # every later test in the run, which is a fun afternoon.
        self.addCleanup(setattr, acquire, "search", acquire.search)
        self.addCleanup(setattr, onboard, "from_url", onboard.from_url)
        self.addCleanup(setattr, acquire, "PAUSE", acquire.PAUSE)

        def search(query, host, limit=5, **kw):
            calls["search"].append((query, host))
            return [{"title": r["title"], "url": r["source"], "host": host}
                    for r in hits.get(host, [])]

        def from_url(url, corpus_title=None, title=None):
            calls["fetch"].append(url)
            return dict(pages[url])

        acquire.search = search
        onboard.from_url = from_url
        acquire.PAUSE = 0
        return calls


# --------------------------------------------------------------------------- #

class TestWhereItLooks(Isolated):
    def test_the_sources_are_read_off_the_corpus(self):
        """Not a constant. A hardcoded list would be me deciding whose food this
        household likes, which the corpus already decides better."""
        hosts = acquire.sources()
        self.assertIn("thecountrycook.net", hosts)
        self.assertIn("natashaskitchen.com", hosts)

    def test_promoting_from_a_new_site_widens_the_search_surface(self):
        self.assertNotIn("example-kitchen.com", acquire.sources())
        pantry.CORPUS.write_text(
            pantry.CORPUS.read_text().replace("julieseatsandtreats.com",
                                              "example-kitchen.com"))
        self.assertIn("example-kitchen.com", acquire.sources())

    def test_a_corpus_naming_no_domains_searches_nothing(self):
        pantry.CORPUS.write_text("# Corpus\n\n| Recipe |\n| --- |\n| Chili |\n")
        pantry.CANDIDATES.write_text("# Candidates\n\n| Recipe |\n| --- |\n")
        self.assertEqual(acquire.sources(), [])
        self.assertEqual(acquire.acquire([]), [])


class TestTheQuery(unittest.TestCase):
    """`searchable` exists because searching nine sites for `chinese-ish`
    returns nothing anywhere."""

    def test_the_hedge_is_dropped(self):
        self.assertEqual(acquire.searchable("Chinese-ish"), "chinese")
        self.assertEqual(acquire.searchable("Japanese-ish"), "japanese")

    def test_fish_is_not_shortened_to_f(self):
        """It was. The query went out as a single letter, to nine sites."""
        self.assertEqual(acquire.searchable("fish"), "fish")
        self.assertEqual(acquire.searchable("Danish"), "danish")

    def test_a_compound_keeps_the_half_a_search_box_knows(self):
        self.assertEqual(acquire.searchable("Italian-American"), "italian")
        self.assertEqual(acquire.searchable("Tex-Mex"), "tex mex")


class TestTheGap(Isolated):
    def make(self, *specs):
        return [pantry.Meal(slug=f"m{i}", title=f"M{i}", protein=p, active=a)
                for i, (p, a) in enumerate(specs)]

    def test_a_protein_the_corpus_has_and_the_week_does_not(self):
        kinds = {(g.kind, g.query) for g in acquire.gaps(self.make(("beef", "low")))}
        self.assertIn(("protein", "pork"), kinds)
        self.assertIn(("protein", "fish"), kinds)

    def test_it_never_asks_for_a_protein_the_household_has_never_bought(self):
        queries = {g.query for g in acquire.gaps(self.make(("beef", "low")))}
        self.assertNotIn("lamb", queries)
        self.assertNotIn("duck", queries)

    def test_a_thin_low_active_week_asks_for_a_slow_cooker(self):
        gaps = acquire.gaps(self.make(("beef", "high"), ("pork", "high"),
                                      ("chicken", "med"), ("fish", "med")))
        self.assertIn("effort", {g.kind for g in gaps})

    def test_a_covered_week_still_widens_the_corpus(self):
        """The planner prompt is explicit that at this corpus size acquisition is
        part of the job every week, not an occasional flourish. A week that wants
        nothing is still a corpus that needs widening."""
        gaps = acquire.gaps(self.make(("beef", "low"), ("pork", "low"),
                                      ("chicken", "low"), ("fish", "low")))
        self.assertEqual(gaps[0].kind, "breadth")
        self.assertIn("thinnest protein", gaps[0].why)

    def test_breadth_widens_protein_and_never_cuisine_on_its_own(self):
        """`profile.md` records the cuisine narrowness as measured fact and then
        says nobody has been asked whether they want it widened. Acting on an
        unanswered question is the failure that profile is written to prevent."""
        gaps = acquire.gaps(self.make(("beef", "low"), ("pork", "low"),
                                      ("chicken", "low"), ("fish", "low")))
        self.assertNotEqual(gaps[0].kind, "cuisine")


class TestJudgingOnePage(Isolated):
    def known(self):
        return {r["slug"] for r in pantry.load_corpus()} | \
               {r["slug"] for r in pantry.load_candidates()}

    def test_a_peanut_recipe_is_refused(self):
        """Verbatim from a live run: nine sources were asked for `fish` and one
        returned a Thai curry with half a cup of roasted peanuts."""
        v = acquire.assess(CURRY, acquire.Gap("protein", "beef", ""), self.known())
        self.assertFalse(v.ok)
        self.assertIn("peanut", v.refusals[0])

    def test_a_muffin_is_refused_for_not_being_dinner(self):
        """Also from the live run. Nothing about a muffin is unsafe; it is just
        not dinner, and every hard constraint passed it."""
        v = acquire.assess(MUFFIN, acquire.Gap("breadth", "fish", ""), self.known())
        self.assertFalse(v.ok)
        self.assertIn("mains-only", v.refusals[0])

    def test_the_wrong_protein_is_refused_rather_than_scored_down(self):
        """A beef taco is a fine recipe and a wrong answer to 'the week has no
        fish'. Scored, it wins whenever nothing better turns up."""
        v = acquire.assess(TACOS, acquire.Gap("protein", "fish", ""), self.known())
        self.assertFalse(v.ok)
        self.assertIn("gap asked for fish", v.refusals[0])

    def test_a_page_with_no_recipe_data_is_refused(self):
        v = acquire.assess(NO_DATA, acquire.Gap("protein", "fish", ""), self.known())
        self.assertFalse(v.ok)
        self.assertIn("no machine-readable recipe", v.refusals[0])

    def test_something_already_known_is_refused(self):
        v = acquire.assess(recipe("Chili", "1 lb ground beef"),
                           acquire.Gap("protein", "beef", ""), self.known())
        self.assertFalse(v.ok)
        self.assertIn("already in the corpus", v.refusals[0])

    def test_a_good_fish_recipe_passes(self):
        v = acquire.assess(COD, acquire.Gap("breadth", "fish", ""), self.known())
        self.assertTrue(v.ok, v.refusals)
        self.assertEqual(v.protein, "fish")

    def test_active_is_read_off_the_source_and_says_so(self):
        v = acquire.assess(SLOW, acquire.Gap("protein", "pork", ""), self.known())
        self.assertEqual(v.active, "low")
        self.assertIn("prep 5 min", v.active_basis)

    def test_a_source_that_states_no_prep_time_gets_no_rating(self):
        """Every effort rating in the corpus is already an unverified guess.
        Adding one with no page behind it makes that worse."""
        v = acquire.assess(recipe("Quiet Dish", "1 lb cod fillets", times=()),
                           acquire.Gap("breadth", "fish", ""), self.known())
        self.assertEqual(v.active, "")

    def test_a_fussy_recipe_is_kept_but_loses(self):
        """Not a refusal: the profile keeps one or two weekend nights open for
        something longer and nicer."""
        fussy = acquire.assess(FUSSY, acquire.Gap("breadth", "fish", ""), self.known())
        easy = acquire.assess(COD, acquire.Gap("breadth", "fish", ""), self.known())
        self.assertTrue(fussy.ok)
        self.assertLess(fussy.score, easy.score)

    def test_the_shortcut_shape_scores(self):
        """`profile.md`: a scratch-everything proposal is wrong for this
        household regardless of how good the recipe is."""
        v = acquire.assess(SLOW, acquire.Gap("protein", "pork", ""), self.known())
        self.assertTrue(any("shortcut" in f for f in v.fits))


class TestTheRun(Isolated):
    def gap_week(self):
        return [pantry.Meal(slug="a", title="A", protein="beef", active="low"),
                pantry.Meal(slug="b", title="B", protein="chicken", active="low")]

    def test_it_lands_a_recipe_nobody_had_bookmarked(self):
        self.stub({"thecountrycook.net": [COD]}, {COD["source"]: COD})
        got = acquire.acquire(self.gap_week(), want=1)
        self.assertEqual(len(got), 1)
        rows = {r["slug"]: r for r in pantry.load_candidates()}
        self.assertIn("cod-fish-in-tomato-sauce", rows)

    def test_the_candidate_says_where_it_came_from(self):
        self.stub({"thecountrycook.net": [COD]}, {COD["source"]: COD})
        acquire.acquire(self.gap_week(), want=1)
        row = {r["slug"]: r for r in pantry.load_candidates()}["cod-fish-in-tomato-sauce"]
        self.assertEqual(row["source"], COD["source"])

    def test_the_capture_is_complete_enough_for_the_shopping_list(self):
        """`docs/brief-next.md` §2: done means the capture is complete enough for
        the list. An acquisition that cannot be shopped for is not one."""
        self.stub({"thecountrycook.net": [COD]}, {COD["source"]: COD})
        acquire.acquire(self.gap_week(), want=1)
        path = pantry.recipe_file("cod-fish-in-tomato-sauce")
        self.assertTrue(path.exists())
        body = path.read_text()
        self.assertIn("cod fillets", body)
        self.assertIn(COD["source"], body)
        self.assertIn("peanut:", body)

    def test_the_reason_says_why_it_reached(self):
        self.stub({"thecountrycook.net": [COD]}, {COD["source"]: COD})
        got = acquire.acquire(self.gap_week(), want=1)
        reason = got[0].reason()
        self.assertTrue(reason.startswith("new here —"))
        self.assertIn("fish", reason)

    def test_a_candidate_never_claims_membership_it_does_not_have(self):
        self.stub({"thecountrycook.net": [COD]}, {COD["source"]: COD})
        got = acquire.acquire(self.gap_week(), want=1)
        self.assertTrue(got[0].reason().startswith("new here"))
        row = {r["slug"]: r for r in pantry.load_candidates()}["cod-fish-in-tomato-sauce"]
        self.assertEqual(row["outcome"], "untested")

    def test_nothing_lands_in_the_corpus(self):
        """Membership is earned. Acquisition is the one path most likely to
        forget that, since it is the one that adds recipes."""
        before = len(pantry.load_corpus())
        self.stub({"thecountrycook.net": [COD]}, {COD["source"]: COD})
        acquire.acquire(self.gap_week(), want=1)
        self.assertEqual(len(pantry.load_corpus()), before)

    def test_a_dry_run_writes_nothing(self):
        before = pantry.CANDIDATES.read_text()
        self.stub({"thecountrycook.net": [COD]}, {COD["source"]: COD})
        got = acquire.acquire(self.gap_week(), want=1, dry_run=True)
        self.assertEqual(len(got), 1)
        self.assertEqual(pantry.CANDIDATES.read_text(), before)

    def test_it_picks_the_better_of_two(self):
        self.stub({"thecountrycook.net": [COD, FUSSY]},
                  {COD["source"]: COD, FUSSY["source"]: FUSSY})
        got = acquire.acquire(self.gap_week(), want=1)
        self.assertEqual(got[0].rec["title"], COD["title"])

    def test_a_source_with_no_search_api_is_skipped_not_scraped(self):
        """tasty.co really does 404 on `/wp-json`. A site that has not published
        a search API has not agreed to be searched by a program, and guessing its
        URL structure is the same move `onboard.from_url` already refuses when it
        declines to read a recipe off page prose."""
        self.stub({h: [COD] for h in acquire.sources() if h != "tasty.co"},
                  {COD["source"]: COD})
        inner = acquire.search

        def search(query, host, limit=5, **kw):
            if host == "tasty.co":
                raise acquire.Unavailable("HTTP 404")
            return inner(query, host, limit, **kw)
        acquire.search = search
        self.assertEqual(len(acquire.acquire(self.gap_week(), want=1)), 1)

    def test_the_fetch_budget_is_capped(self):
        many = [recipe(f"Cod Number {i}", "1 lb cod fillets",
                       source=f"https://thecountrycook.net/{i}/") for i in range(40)]
        calls = self.stub({"thecountrycook.net": many},
                          {r["source"]: r for r in many})
        acquire.acquire(self.gap_week(), want=1)
        self.assertLessEqual(len(calls["fetch"]), acquire.MAX_FETCHES)

    def test_the_run_is_recorded(self):
        self.stub({"thecountrycook.net": [COD]}, {COD["source"]: COD})
        acquire.acquire(self.gap_week(), want=1)
        got = pantry.decisions({"acquired"})
        self.assertEqual(len(got), 1)
        self.assertEqual(got[0]["source"], COD["source"])
        self.assertEqual(got[0]["found_by"], "acquire")


class TestPastingALink(Isolated):
    """§3: adding a recipe meant running a CLI with a URL, which for a tool whose
    thesis is closing the gap between the 15 you reach for and the 60 you like is
    close to fatal."""

    def page(self, rec):
        self.addCleanup(setattr, onboard, "from_url", onboard.from_url)
        onboard.from_url = lambda url, **kw: dict(rec)

    def test_a_pasted_link_lands_a_candidate(self):
        self.page(COD)
        found = acquire.from_url(COD["source"])
        self.assertEqual(found.rec["title"], COD["title"])
        self.assertIn("cod-fish-in-tomato-sauce",
                      {r["slug"] for r in pantry.load_candidates()})

    def test_it_goes_through_the_same_door_acquisition_uses(self):
        """A recipe somebody pasted and a recipe the tool found are the same
        recipe and get the same row. Two write paths would drift."""
        self.page(COD)
        acquire.from_url(COD["source"])
        row = {r["slug"]: r for r in pantry.load_candidates()}["cod-fish-in-tomato-sauce"]
        self.assertEqual(row["source"], COD["source"])
        self.assertEqual(row["outcome"], "untested")
        self.assertTrue(pantry.recipe_file("cod-fish-in-tomato-sauce").exists())

    def test_a_hard_constraint_still_applies_to_a_recipe_a_person_chose(self):
        """An allergy is not a preference, and a human typing the URL is not
        evidence against it."""
        self.page(CURRY)
        with self.assertRaises(acquire.Unavailable) as e:
            acquire.from_url(CURRY["source"])
        self.assertIn("peanut", str(e.exception))

    def test_a_page_with_no_recipe_data_is_refused_with_a_readable_reason(self):
        self.page(NO_DATA)
        with self.assertRaises(acquire.Unavailable) as e:
            acquire.from_url(NO_DATA["source"])
        self.assertIn("no machine-readable recipe", str(e.exception))

    def test_relevance_is_not_applied_to_something_chosen(self):
        """The muffin filter exists because full-text search is noisy. A person
        pasting a link is not noise, and applying it there would refuse a
        vegetarian main somebody deliberately went and found."""
        veg = recipe("Mushroom Barley Risotto", "1 cup pearl barley",
                     "8 oz mushrooms", "1 can vegetable broth",
                     source="https://lilluna.com/mushroom-barley-risotto/")
        self.page(veg)
        found = acquire.from_url(veg["source"])
        self.assertEqual(found.rec["title"], veg["title"])

    def test_the_same_thing_twice_says_so(self):
        self.page(COD)
        acquire.from_url(COD["source"])
        with self.assertRaises(acquire.Unavailable) as e:
            acquire.from_url(COD["source"])
        self.assertIn("already", str(e.exception))

    def test_it_is_recorded_as_pasted_not_as_found(self):
        """The decision log has to be able to tell the two apart, or `how much of
        the corpus did the tool grow` stops being answerable."""
        self.page(COD)
        acquire.from_url(COD["source"])
        self.assertEqual(pantry.decisions({"acquired"})[0]["found_by"], "paste")


class TestTheWriteDoor(Isolated):
    """`pantry.add_candidate` — what `upsert_corpus` refuses to be."""

    def test_a_candidate_with_no_source_is_refused(self):
        with self.assertRaises(pantry.RuleViolation) as e:
            pantry.add_candidate("Something", source="")
        self.assertIn("invention", str(e.exception))

    def test_a_corpus_recipe_may_not_be_re_added_as_a_gamble(self):
        with self.assertRaises(pantry.RuleViolation) as e:
            pantry.add_candidate("Chili", source="https://example.org/x")
        self.assertIn("already in the corpus", str(e.exception))

    def test_adding_the_same_thing_twice_is_a_no_op(self):
        self.assertTrue(pantry.add_candidate("New Thing", source="https://x.test/a"))
        self.assertFalse(pantry.add_candidate("New Thing", source="https://x.test/a"))
        self.assertEqual(sum(1 for r in pantry.load_candidates()
                             if r["slug"] == "new-thing"), 1)

    def test_the_source_column_is_migrated_in_place(self):
        """These files are hand-edited and diffed by eye. A migration that
        reformats the table it touched is one nobody can review."""
        pantry.add_candidate("New Thing", source="https://x.test/a")
        rows = [l for l in pantry.CANDIDATES.read_text().splitlines()
                if l.startswith("|")]
        self.assertTrue(rows[0].rstrip().endswith("| Source |"))
        self.assertEqual(rows[1].count("|"), rows[0].count("|"))
        for row in rows[2:]:
            self.assertEqual(row.count("|"), rows[0].count("|"))

    def test_the_existing_rows_keep_their_values(self):
        before = {r["slug"]: r.get("outcome") for r in pantry.load_candidates()}
        pantry.add_candidate("New Thing", source="https://x.test/a")
        after = {r["slug"]: r.get("outcome") for r in pantry.load_candidates()}
        for slug, outcome in before.items():
            self.assertEqual(after[slug], outcome)


if __name__ == "__main__":
    unittest.main(verbosity=2)
