"""Regression tests for the deterministic core.

The sixteen cases in `TestSixteenRegressions` come straight off the "rows that earned
their place" notes at `items.md:150-182`. Every one of them records a bug that actually
happened in this project. They are the reason the item table has the shape it has, and a
change that breaks one of them is a change that reintroduces a bug somebody already paid
for.

Run: `python3 -m unittest discover -s tests -v`
"""

from __future__ import annotations

import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from gr import repo as R                          # noqa: E402
from gr import shoplist as S                      # noqa: E402
from gr import units as U                         # noqa: E402
from gr.parse import parse_line                   # noqa: E402
from gr.recipes import AE, PER_PORTION, PORTIONS, UNKNOWN, multiplier_for, parse_yield  # noqa: E402

REPO = R.load(ROOT)
TABLE = REPO.items


def resolve(text):
    """Return `(canonical_or_None, match_kind_or_refusal)` for one ingredient line."""
    line = parse_line(text, TABLE)
    if line.resolved:
        return line.match.item.canonical, line.match.kind
    return None, "refused"


class TestSixteenRegressions(unittest.TestCase):
    """The sixteen cases from `items.md:150-182`. All sixteen must pass."""

    # -- the onion powder family: a mis-merge is worse than an unknown line ----

    def test_01_onion_powder_is_not_an_onion(self):
        # Thirteen lines of the corpus once bought a fresh onion for a teaspoon of spice.
        self.assertEqual(resolve("1/2 tsp onion powder"), ("onion_powder", "exact"))

    def test_02_dried_thyme_is_not_fresh_thyme(self):
        # `dried` and `fresh` name which aisle you walk to. Neither is a noise word.
        self.assertEqual(resolve("1 teaspoon dried thyme"), ("dried_thyme", "exact"))

    def test_03_green_onion_is_its_own_item(self):
        self.assertEqual(resolve("1/4 cup sliced green onion"), ("green_onion", "exact"))

    def test_04_garlic_salt_and_garlic_powder_stay_apart(self):
        self.assertEqual(resolve("1 tsp garlic salt")[0], "garlic_salt")
        self.assertEqual(resolve("1 tsp garlic powder")[0], "garlic_powder")
        self.assertEqual(resolve("3 cloves garlic")[0], "garlic")

    def test_05_dried_parsley_is_not_fresh_parsley(self):
        self.assertEqual(resolve("½ teaspoon dried parsley")[0], "dried_parsley")
        self.assertEqual(resolve("3 Tbsp fresh parsley")[0], "parsley")

    # -- the partial-match rule: leftovers must all be noise -------------------

    def test_06_partial_match_when_every_leftover_is_noise(self):
        self.assertEqual(resolve("1 medium onion, chopped"), ("onion", "partial"))

    def test_07_salt_to_taste_parses_and_routes_to_staples(self):
        item, kind = resolve("salt, to taste")
        self.assertEqual(item, "salt")
        self.assertTrue(TABLE.items["salt"].staple,
                        "salt must be a staple, or `salt, to taste` reaches the list as "
                        "a thing to buy")

    def test_08_pickle_juice_merges_into_the_jar_you_already_buy(self):
        self.assertEqual(resolve("1-2 tsp pickle juice")[0], "pickles")

    # -- refusals: an unknown line is the mechanism working --------------------

    def test_09_ambiguous_or_is_refused(self):
        self.assertEqual(resolve("Butter or oil"), (None, "refused"))

    def test_10_two_ingredients_on_one_line_is_refused(self):
        self.assertEqual(
            resolve("2 tsp thyme and rosemary, freshly chopped"), (None, "refused"))

    # -- hard grammar the sources actually contain -----------------------------

    def test_11_leading_adjectives_and_trailing_prep(self):
        self.assertEqual(
            resolve("2 lb boneless, skinless chicken thighs, cubed")[0], "chicken_thigh")

    def test_12_declared_accepts_wins(self):
        line = parse_line(
            "1 cup shredded provolone or mozzarella cheese    accepts: mozzarella", TABLE)
        self.assertEqual(line.match.item.canonical, "mozzarella")
        self.assertEqual(line.match.kind, "accepts",
                         "`accepts` is declared by the household, never inferred")

    # -- each_equiv: the reason the column exists ------------------------------

    def test_13_bell_pepper_conversion_makes_two_meals_comparable(self):
        # `3 cups bell peppers` and `1 green + 1 red bell pepper` are incomparable
        # without a conversion, and aggregate to 5 with one.
        graph = TABLE.items["bell_pepper"].graph
        self.assertAlmostEqual(U.convert(3, "cup", "ea", graph), 3.0)

    def test_14_garlic_chains_two_clauses_to_reach_one_head(self):
        # `1 head = 10 cloves; 1 clove = 1 tsp` is what turns a tablespoon of chopped
        # garlic and four whole cloves into *buy one head*.
        graph = TABLE.items["garlic"].graph
        self.assertAlmostEqual(U.convert(1, "tbsp", "clove", graph), 3.0)
        self.assertAlmostEqual(U.convert(10, "clove", "head", graph), 1.0)
        # items.md's own worked example: 4 cloves + 1 tbsp chopped + 1 clove, across
        # three recipes and three units, must come out as *buy one head*.
        total = (U.convert(4, "clove", "head", graph)
                 + U.convert(1, "tbsp", "head", graph)
                 + U.convert(1, "clove", "head", graph))
        self.assertEqual(U.tidy(total, "head", graph), (1, "head"))
        # And thirteen cloves is genuinely two heads. Rounding up is not over-buying.
        self.assertEqual(U.tidy(U.convert(13, "clove", "head", graph), "head", graph),
                         (2, "head"))

    def test_15_canned_carrot_is_not_a_produce_carrot(self):
        # Live bug, fixed today: `sliced carrots` sat in the `carrot` row, so a 14.5 oz
        # can in chicken-and-dumplings was bought as fresh produce.
        item, _ = resolve("1 14.5 oz can sliced carrots")
        self.assertEqual(item, "canned_carrot")
        self.assertNotEqual(TABLE.items[item].aisle, "produce")
        self.assertEqual(resolve("4 large carrots")[0], "carrot",
                         "a fresh carrot must still be a fresh carrot")

    def test_16_the_two_remaining_audit_misses_now_have_rows(self):
        self.assertEqual(resolve("1 cup frozen peas and carrots")[0],
                         "frozen_peas_and_carrots")
        self.assertEqual(resolve("1/2 pound ground mild Italian sausage")[0],
                         "ground_italian_sausage")


class TestGrammar(unittest.TestCase):
    """Shapes the recipe files contain that the sixteen do not cover."""

    def test_container_with_a_stated_size(self):
        line = parse_line("- 1 10.75 oz can cream of chicken soup", TABLE)
        self.assertEqual(line.match.item.canonical, "cream_of_chicken_soup")
        self.assertEqual((line.parsed.qty, line.parsed.unit), (1.0, "can"))

    def test_count_of_containers_with_a_stated_size(self):
        line = parse_line("- 2 16.3 oz. tubes refrigerated biscuits cut into quarters", TABLE)
        self.assertEqual(line.match.item.canonical, "refrigerated_biscuits")
        self.assertEqual((line.parsed.qty, line.parsed.unit), (2.0, "tube"))

    def test_size_in_parentheses(self):
        line = parse_line("- 1 (10-ounce) can red enchilada sauce", TABLE)
        self.assertEqual(line.match.item.canonical, "enchilada_sauce")
        self.assertEqual((line.parsed.qty, line.parsed.unit), (1.0, "can"))

    def test_unicode_fractions(self):
        self.assertEqual(parse_line("- ¼ cup all-purpose flour", TABLE).parsed.qty, 0.25)
        self.assertEqual(parse_line("- 1½ tablespoons tomato paste", TABLE).parsed.qty, 1.5)
        self.assertEqual(parse_line("- 1 ½ tsp white vinegar", TABLE).parsed.qty, 1.5)

    def test_a_range_takes_the_high_end(self):
        # Under-buying sends somebody back to the shop.
        self.assertEqual(parse_line("- 4-5 pound chuck roast", TABLE).parsed.qty, 5.0)

    def test_container_with_no_number(self):
        line = parse_line("- Tube of biscuits", TABLE)
        self.assertEqual(line.match.item.canonical, "refrigerated_biscuits")
        self.assertEqual((line.parsed.qty, line.parsed.unit), (1.0, "tube"))

    def test_trailing_unit_word_is_read_as_the_unit(self):
        line = parse_line("- 3 garlic cloves (minced)", TABLE)
        self.assertEqual(line.match.item.canonical, "garlic")
        self.assertEqual((line.parsed.qty, line.parsed.unit), (3.0, "clove"))

    def test_a_synonym_ending_in_a_unit_word_still_wins(self):
        # `pepperoncini pepper slices` is the item's own name, not a count of slices.
        line = parse_line("- 8 oz pepperoncini pepper slices, plus extra for serving", TABLE)
        self.assertEqual(line.match.item.canonical, "pepperoncini")
        self.assertEqual(line.parsed.unit, "oz")

    def test_a_declared_or_in_the_table_is_not_ambiguous(self):
        # `pasta sauce or marinara sauce` is one synonym, written that way by the source.
        self.assertEqual(resolve("2 cups pasta sauce or marinara sauce")[0], "marinara")

    def test_malformed_line_is_refused_not_guessed(self):
        self.assertEqual(resolve("2 cups or chicken broth"), (None, "refused"))

    def test_a_line_with_no_quantity_keeps_its_item(self):
        line = parse_line("- Onion    <!-- quantity not stated -->", TABLE)
        self.assertEqual(line.match.item.canonical, "onion")
        self.assertIsNone(line.parsed.qty)

    def test_refused_lines_keep_their_raw_text(self):
        line = parse_line("- Butter or oil", TABLE)
        self.assertFalse(line.resolved)
        self.assertIn("Butter or oil", line.raw)


class TestYieldAndScaling(unittest.TestCase):

    def test_the_four_yield_shapes(self):
        self.assertEqual(parse_yield("6 AE (source: 6)").shape, AE)
        self.assertEqual(parse_yield("8 enchiladas (source) [recovered]").shape, PORTIONS)
        self.assertEqual(parse_yield("per portion — no batch exists").shape, PER_PORTION)
        self.assertEqual(parse_yield("unknown (not stated in source)").shape, UNKNOWN)

    def test_ae_scales_by_target_over_yield(self):
        recipe = REPO.recipes["beef-stew-with-carrots-and-potatoes"]     # 6 AE
        scale = multiplier_for(recipe, 2.5, {})
        self.assertAlmostEqual(scale.multiplier, 2.5 / 6)
        self.assertTrue(scale.scaled)

    def test_per_portion_never_multiplies_the_batch_convenience(self):
        # The bug this replaces: `2 lb ground beef` × 2.5 AE ordered five pounds of beef
        # for burger night. corpus.md calls that 2 lb "a convenience, not a batch".
        recipe = REPO.recipes["hamburgers"]
        scale = multiplier_for(recipe, 2.5, {})
        self.assertEqual(scale.multiplier, 1.0)
        self.assertFalse(scale.scaled)
        self.assertIn("not scaled", scale.note)

    def test_unknown_yield_is_never_scaled_and_says_so(self):
        scale = multiplier_for(REPO.recipes["zuppa-toscana"], 2.5, {})
        self.assertEqual(scale.multiplier, 1.0)
        self.assertFalse(scale.scaled)
        self.assertIn("not scaled", scale.note)

    def test_portions_without_a_conversion_are_not_scaled(self):
        scale = multiplier_for(REPO.recipes["enchiladas"], 2.5, {})
        self.assertFalse(scale.scaled)
        self.assertIn("enchilada", scale.note)

    def test_portions_with_a_conversion_do_scale(self):
        # One number from the household closes the recipe forever.
        scale = multiplier_for(REPO.recipes["enchiladas"], 2.5, {"enchilada": 2.0})
        self.assertTrue(scale.scaled)
        self.assertAlmostEqual(scale.multiplier, 2.5 / (8 / 2.0))

    def test_guests_raise_the_target(self):
        self.assertEqual(REPO.target_ae(0), REPO.household.base_ae)
        self.assertEqual(REPO.target_ae(3), REPO.household.base_ae + 3)


class TestAggregation(unittest.TestCase):

    def _list_for(self, slugs):
        meals = [S.MealPlan(slug=s, title=s, reason_kind="plain", reason="",
                            yield_raw="", scale=None) for s in slugs]
        return S.build(REPO, meals, guests=0)

    def test_one_item_is_always_one_line(self):
        # The old aggregation keyed on (item, unit) and printed pickles three times.
        result = self._list_for(["tuna-melt"])
        names = [l.item for l in result.buy] + [l.item for l in result.staples]
        self.assertEqual(len(names), len(set(names)),
                         "an item printed twice reads as two things to buy")

    def test_pickles_merge_across_units_into_a_single_line(self):
        # Report §3.3: the old list printed `pickles` three times — bare, `2 tbsp`,
        # `1 tsp` — which a shopper reads as three jars. One item, one line.
        result = self._list_for(["tuna-melt"])
        pickles = [l for l in result.buy if l.item == "pickles"]
        self.assertEqual(len(pickles), 1)
        self.assertEqual(pickles[0].unit, "tbsp")

    def test_unaddable_units_stay_on_one_line_and_are_named(self):
        # A can of biscuits and a tube of biscuits are the same item in two units that
        # no `each_equiv` clause bridges. That is one line saying both, never two lines.
        result = self._list_for(["chicken-and-biscuits-casserole", "chicken-and-dumplings"])
        biscuits = [l for l in result.buy if l.item == "refrigerated_biscuits"]
        self.assertEqual(len(biscuits), 1)
        self.assertIn("+", biscuits[0].quantity_text())
        self.assertTrue(any("could not be added" in f for f in biscuits[0].flags))

    def test_garlic_aggregates_across_meals_into_one_head(self):
        result = self._list_for(["beef-stew-with-carrots-and-potatoes", "zuppa-toscana"])
        garlic = next(l for l in result.buy if l.item == "garlic")
        self.assertEqual(len(garlic.sources), 2)
        self.assertFalse(garlic.stranded)

    def test_staples_are_routed_not_dropped(self):
        result = self._list_for(["easy-salmon-dinner"])
        staple_names = [l.item for l in result.staples]
        self.assertIn("salt", staple_names)
        self.assertNotIn("salt", [l.item for l in result.buy])

    def test_unknown_lines_are_printed_never_dropped(self):
        result = self._list_for(["tuna-melt"])
        raws = [u.raw for u in result.unknown]
        self.assertTrue(any("Butter or oil" in r for r in raws))
        for unknown in result.unknown:
            self.assertTrue(unknown.reason, "an unknown line must say why it is unknown")

    def test_provenance_survives_aggregation(self):
        result = self._list_for(["chili", "tacos"])
        beef = next(l for l in result.buy if l.item == "ground_beef")
        self.assertEqual(sorted(beef.sources), ["chili", "tacos"])

    def test_a_line_from_one_meal_is_stranded(self):
        result = self._list_for(["easy-salmon-dinner", "chili"])
        salmon = next(l for l in result.buy if l.item == "salmon_fillet")
        self.assertTrue(salmon.stranded)

    def test_hamburgers_do_not_order_five_pounds_of_beef(self):
        # The regression that report §3.1 caught, asserted end to end.
        result = self._list_for(["hamburgers"])
        beef = next(l for l in result.buy if l.item == "ground_beef")
        self.assertEqual(beef.qty, 2, "the recipe says 2 lb and nobody scaled it")
        self.assertEqual(beef.unit, "lb")
        self.assertTrue(any("unscaled" in f for f in beef.flags))

    def test_canned_carrots_do_not_reach_the_produce_aisle(self):
        result = self._list_for(["chicken-and-dumplings"])
        carrot = next(l for l in result.buy if l.item == "canned_carrot")
        self.assertNotEqual(carrot.aisle, "produce")


class TestRepositoryData(unittest.TestCase):

    def test_every_corpus_slug_names_a_recipe_file(self):
        missing = [r.title for r in REPO.missing_recipe_files()]
        self.assertEqual(missing, [],
                         "the Slug column is the join; a row with no file is a silent "
                         "failure waiting to happen")

    def test_the_corpus_is_the_size_it_says_it_is(self):
        self.assertEqual(len(REPO.corpus), 24)
        self.assertEqual(len(REPO.candidates), 3)

    def test_the_household_is_read_from_profile_md(self):
        self.assertEqual(REPO.household.base_ae, 2.5)
        self.assertIn("peanut", REPO.household.allergens)

    def test_both_portion_conversions_are_open_and_none_is_invented(self):
        self.assertEqual(REPO.household.portion_conversions, {})
        self.assertEqual(sorted(REPO.household.open_conversions),
                         ["enchilada", "slider"])

    def test_sides_is_empty_and_that_is_recorded_not_filled(self):
        self.assertEqual(REPO.sides, [],
                         "a seeded side would make the tool look finished and every "
                         "list wrong in a new way")

    def test_resolution_rate_holds(self):
        """The audit the project runs on itself. Regressions show up here first."""
        total = resolved = 0
        for recipe in REPO.recipes.values():
            for line in recipe.lines:
                total += 1
                resolved += 1 if line.resolved else 0
        rate = resolved / total
        self.assertGreaterEqual(
            rate, 0.97,
            f"resolution fell to {rate:.1%} ({resolved}/{total} lines)")


if __name__ == "__main__":
    unittest.main(verbosity=2)
