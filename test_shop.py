#!/usr/bin/env python3
"""Tests for the grocery list generator.

    python3 test_shop.py

Three kinds of case: the ingredient lines docs/step2-design.md §2 names as
known-hard, the arithmetic the whole pipeline exists to get right, and the
week of 2 August, which is the acceptance fixture. Standard library only.
"""

import unittest

import household
import shop
from shop import (
    Ingredient,
    aggregate,
    build,
    display,
    load_items,
    load_recipe,
    normalize,
    parse_ingredient,
    resolve,
    scale,
    split_compound,
)

# These tests run against the real household's files on purpose: the point of
# most of them is that all 27 captures parse and every ingredient line resolves.
HH = household.here()

ITEMS, INDEX = load_items(HH)


def p(line):
    return parse_ingredient(line)


def canonical(line):
    return normalize(p(line), INDEX)


class TestGrammar(unittest.TestCase):
    """The six lines §2 calls known-hard, plus what twenty-seven real files added."""

    def test_qty_unit_item_note(self):
        r = p("3 cups bell peppers, sliced")
        self.assertEqual((r.qty, r.unit, r.item, r.note), (3.0, "cup", "bell peppers", "sliced"))

    def test_nested_quantity_the_can_is_the_unit(self):
        r = p("1 (14.5 oz) can beef broth")
        self.assertEqual((r.qty, r.unit, r.item), (1.0, "can", "beef broth"))
        self.assertEqual(r.pack, (14.5, "oz"))

    def test_hyphenated_pack_size(self):
        r = p("1 (10-ounce) can red enchilada sauce")
        self.assertEqual((r.qty, r.unit, r.item, r.pack), (1.0, "can", "red enchilada sauce", (10.0, "oz")))

    def test_packaging_defined_unit(self):
        r = p("1 envelope Italian salad dressing mix")
        self.assertEqual((r.qty, r.unit, r.item), (1.0, "envelope", "Italian salad dressing mix"))

    def test_quantity_as_a_source_object(self):
        r = p("juice of 1 lemon")
        self.assertEqual((r.qty, r.item, r.note), (1.0, "lemon", "juice of"))

    def test_no_quantity_and_a_staple(self):
        r = p("salt, to taste")
        self.assertIsNone(r.qty)
        self.assertEqual(r.item, "salt")
        self.assertTrue(ITEMS[normalize(r, INDEX)].staple)

    def test_two_quantities_one_unmeasured(self):
        r = p("8 oz pepperoncini pepper slices, plus extra for serving")
        self.assertEqual((r.qty, r.unit, r.item), (8.0, "oz", "pepperoncini pepper slices"))

    def test_note_splits_on_the_last_comma(self):
        r = p("2 lb boneless, skinless chicken thighs, cubed")
        self.assertEqual(r.item, "boneless, skinless chicken thighs")
        self.assertEqual(r.note, "cubed")

    def test_trailing_prep_clause_is_not_part_of_the_name(self):
        r = p("4-5 medium Russet potatoes, peeled, thinly sliced and quartered")
        self.assertEqual(r.item, "Russet potatoes")
        self.assertEqual(r.qty, 5.0, "a range takes the high end - buying short is worse")

    def test_glued_unit(self):
        self.assertEqual((p("2lb ground beef").qty, p("2lb ground beef").unit), (2.0, "lb"))

    def test_unicode_and_mixed_fractions(self):
        self.assertEqual(p("½ cup brown sugar").qty, 0.5)
        self.assertEqual(p("1 ½ tsp white vinegar").qty, 1.5)
        self.assertEqual(p("1½ tablespoons tomato paste").qty, 1.5)
        self.assertEqual(p("3/4 cup ketchup").qty, 0.75)

    def test_weight_before_a_container_is_a_pack_size(self):
        r = p("12 ounce can refrigerated biscuits")
        self.assertEqual((r.qty, r.unit, r.pack), (1.0, "can", (12.0, "oz")))

    def test_unit_word_alone_on_the_line_is_the_item(self):
        r = p("buns")
        self.assertEqual(r.item, "buns", "a lone unit word names the thing, not the unit")
        self.assertIsNone(r.unit)

    def test_unit_of_item(self):
        self.assertEqual(p("1 rib of celery, thinly sliced").item, "celery")
        self.assertEqual(p("2 packets of ranch seasoning").item, "ranch seasoning")

    def test_size_words_are_not_units(self):
        self.assertEqual(p("1 med onion ((1 cup), finely chopped)").item.lower()[:5], "onion")
        self.assertEqual(normalize(p("2  medium yellow onions, (cut into 1-inch chunks)"), INDEX), "onion")

    def test_every_corpus_line_parses(self):
        """265 lines across 27 files. A line that cannot be read must be surfaced,
        so an unparsed line is a bug rather than a shrug."""
        bad = []
        for path in sorted(HH.recipes.glob("*.md")):
            recipe = load_recipe(HH, path.stem)
            for ing in recipe.ingredients + [a for v in recipe.variants for a in v.adds]:
                if not ing.parsed:
                    bad.append(f"{path.stem}: {ing.raw}")
        self.assertEqual(bad, [])


class TestNormalize(unittest.TestCase):
    def test_synonyms_merge(self):
        for line in ["1 green bell pepper, sliced", "3 cups bell peppers, sliced"]:
            self.assertEqual(canonical(line), "bell_pepper")

    def test_a_partial_match_may_not_leave_a_real_word_behind(self):
        """`onion powder` matching `onion` put a fresh onion in the cart for a
        teaspoon of spice, silently, across thirteen lines. A mis-merge is worse
        than an unknown line, because an unknown line gets printed."""
        self.assertEqual(canonical("1/2 teaspoon onion powder"), "onion_powder")
        self.assertEqual(canonical("1 tsp garlic salt"), "garlic_salt")
        self.assertEqual(canonical("1/4 cup sliced green onion"), "green_onion")
        self.assertEqual(canonical("Onion Soup Mix (Lipton's)"), "onion_soup_mix")

    def test_unknown_item_returns_none_rather_than_guessing(self):
        self.assertIsNone(normalize(Ingredient(raw="x", item="dragonfruit"), INDEX))

    def test_or_names_one_item_and_one_tolerance(self):
        self.assertEqual(canonical("1 cup shredded provolone or mozzarella cheese"), "provolone")

    def test_unbalanced_parens_end_the_name(self):
        self.assertEqual(canonical("2 cups cooked chicken ((shredded or diced))"), "cooked_chicken")

    def test_every_corpus_line_is_known(self):
        """items.md is complete for the current corpus. This is allowed to fail
        when a recipe is added - the fix is a row, and ./shop.py --audit names it."""
        misses = []
        for path in sorted(HH.recipes.glob("*.md")):
            recipe = load_recipe(HH, path.stem)
            for ing in recipe.ingredients + [a for v in recipe.variants for a in v.adds]:
                if split_compound(ing, INDEX):
                    continue
                if normalize(ing, INDEX) is None:
                    misses.append(f"{path.stem}: {ing.item}")
        self.assertEqual(misses, [])


class TestCompound(unittest.TestCase):
    def test_one_line_two_items(self):
        parts = split_compound(p("2 tsp thyme and rosemary, freshly chopped"), INDEX)
        self.assertEqual([normalize(x, INDEX) for x in parts], ["thyme", "rosemary"])

    def test_neither_half_carries_the_shared_quantity(self):
        parts = split_compound(p("2 tsp thyme and rosemary, freshly chopped"), INDEX)
        self.assertTrue(all(x.qty is None for x in parts),
                        "splitting 2 tsp between them would be a guess")

    def test_not_a_compound_when_a_half_is_unknown(self):
        self.assertIsNone(split_compound(p("1 cup peas and dragonfruit"), INDEX))


class TestScale(unittest.TestCase):
    """The four yield shapes of §2.5."""

    def test_ae_yield_scales_in_whole_batches(self):
        salmon = load_recipe(HH, "parchment-garlic-butter-salmon")
        self.assertEqual(scale(salmon, 2.5)[0], 3.0, "serves 1, week needs 2.5")
        beef = load_recipe(HH, "crock-pot-italian-beef")
        self.assertEqual(scale(beef, 2.5)[0], 1.0, "serves 8 already; never scale down")

    def test_portion_count_without_a_rate_is_not_scaled_and_says_so(self):
        mult, why = scale(load_recipe(HH, "sliders"), 2.5)
        self.assertEqual(mult, 1.0)
        self.assertIn("how many", why)

    def test_per_portion_is_not_a_batch(self):
        mult, why = scale(load_recipe(HH, "blt"), 2.5)
        self.assertEqual(mult, 1.0)
        self.assertIn("headcount", why)

    def test_unknown_yield_scales_by_one_and_flags(self):
        mult, why = scale(load_recipe(HH, "chicken-chili"), 2.5)
        self.assertEqual(mult, 1.0)
        self.assertIn("unknown", why)

    def test_guests_push_a_recipe_to_a_second_batch(self):
        self.assertEqual(scale(load_recipe(HH, "sausage-and-peppers"), 4.0)[0], 1.0)
        self.assertEqual(scale(load_recipe(HH, "sausage-and-peppers"), 5.0)[0], 2.0)


class TestVariants(unittest.TestCase):
    def test_first_variant_is_the_default(self):
        soup = load_recipe(HH, "chicken-noodle-soup")
        _, chosen = resolve(soup, None)
        self.assertEqual(chosen.name, "Rotisserie")

    def test_adds_land_on_the_list(self):
        ings, _ = resolve(load_recipe(HH, "chicken-noodle-soup"), "rotisserie")
        self.assertIn("rotisserie_chicken", [normalize(i, INDEX) for i in ings])

    def test_replaces_removes_the_base_line(self):
        ings, _ = resolve(load_recipe(HH, "chicken-noodle-soup"), "whole-young-chicken")
        got = [normalize(i, INDEX) for i in ings]
        self.assertIn("whole_chicken", got)
        self.assertNotIn("chicken_broth", got, "the bird is boiled for its stock")

    def test_a_recipe_without_variants_passes_through_unchanged(self):
        recipe = load_recipe(HH, "meatloaf")
        ings, chosen = resolve(recipe, None)
        self.assertIsNone(chosen)
        self.assertEqual(len(ings), len(recipe.ingredients))

    def test_unknown_variant_name_is_an_error_not_a_silent_default(self):
        with self.assertRaises(SystemExit):
            resolve(load_recipe(HH, "chicken-noodle-soup"), "sous-vide")


class TestAggregate(unittest.TestCase):
    def test_cups_and_counts_add_up(self):
        """3 cups bell peppers + 1 green + 1 red = 5, not three incomparable lines."""
        entries = [
            ("bell_pepper", p("3 cups bell peppers, sliced"), 1.0, "fajitas"),
            ("bell_pepper", p("1 green bell pepper, sliced"), 1.0, "sausage"),
            ("bell_pepper", p("1 red bell pepper, sliced"), 1.0, "sausage"),
        ]
        lines, _ = aggregate(entries, ITEMS)
        self.assertEqual(display(lines[0]), ("5", "bell peppers"))

    def test_chained_equivalences(self):
        """4 cloves + 1 tbsp chopped + 3 cloves has to come out as one head."""
        entries = [
            ("garlic", p("4 cloves garlic, minced"), 1.0, "fajitas"),
            ("garlic", p("1 tbsp garlic, chopped"), 1.0, "sausage"),
            ("garlic", p("1 clove garlic, minced"), 3.0, "salmon"),
        ]
        lines, _ = aggregate(entries, ITEMS)
        self.assertEqual(display(lines[0]), ("1 head", "garlic"))

    def test_round_after_aggregating_never_before(self):
        entries = [("bell_pepper", p("1.5 bell peppers"), 1.0, "a"),
                   ("bell_pepper", p("1.5 bell peppers"), 1.0, "b")]
        lines, _ = aggregate(entries, ITEMS)
        self.assertEqual(display(lines[0])[0], "3", "1.5 + 1.5 is 3 peppers, not 4")

    def test_countables_round_up(self):
        entries = [("lime", p("3 tbsp lime juice"), 1.0, "fajitas")]
        lines, _ = aggregate(entries, ITEMS)
        self.assertEqual(display(lines[0]), ("2", "limes"), "you cannot buy 1.5 limes")

    def test_unknown_item_is_reported_not_dropped(self):
        entries = [(None, p("1 cup dragonfruit"), 1.0, "nowhere")]
        lines, unknown = aggregate(entries, ITEMS)
        self.assertEqual(lines, [])
        self.assertEqual(unknown, [("nowhere", "1 cup dragonfruit")])

    def test_provenance_survives(self):
        entries = [("onion", p("1 large white onion, sliced"), 1.0, "fajitas"),
                   ("onion", p("1 cup onion, sliced"), 1.0, "sausage")]
        lines, _ = aggregate(entries, ITEMS)
        self.assertEqual(sorted(set(lines[0].sources)), ["fajitas", "sausage"])


class TestConsolidate(unittest.TestCase):
    def test_a_declared_tolerance_merges_when_the_week_already_has_the_substitute(self):
        from shop import consolidate, Line
        prov = Line(canonical="provolone", qty=1, unit="cup", accepts={"mozzarella"})
        mozz = Line(canonical="mozzarella", qty=2, unit="cup")
        merges = consolidate([prov, mozz], INDEX)
        self.assertEqual(merges, [("provolone", "mozzarella")])
        self.assertEqual(prov.merged_into, "mozzarella")

    def test_nothing_merges_when_the_substitute_is_not_in_the_week(self):
        from shop import consolidate, Line
        prov = Line(canonical="provolone", qty=1, unit="cup", accepts={"mozzarella"})
        self.assertEqual(consolidate([prov], INDEX), [])

    def test_tolerance_is_never_inferred(self):
        from shop import consolidate, Line
        a = Line(canonical="cheddar", qty=1, unit="cup")
        b = Line(canonical="mozzarella", qty=1, unit="cup")
        self.assertEqual(consolidate([a, b], INDEX), [],
                         "cheese is not interchangeable with cheese unless a recipe said so")


class TestLink(unittest.TestCase):
    def test_a_produced_item_is_noted_and_still_bought(self):
        from shop import link, Line
        soup = load_recipe(HH, "chicken-noodle-soup")
        _, whole = resolve(soup, "whole-young-chicken")
        line = Line(canonical="chicken_broth", qty=6, unit="cup",
                    from_produce="chicken noodle soup, whole-bird variant")
        notes = link([line], [(soup, whole)])
        self.assertEqual(len(notes), 1)
        self.assertIn("Buy it anyway", notes[0],
                      "the link saves you using the item, never buying it")

    def test_no_producer_in_the_week_means_no_note(self):
        from shop import link, Line
        line = Line(canonical="chicken_broth", qty=6, unit="cup", from_produce="something else")
        self.assertEqual(link([line], [(load_recipe(HH, "meatloaf"), None)]), [])

    def test_the_corpus_has_exactly_one_producer(self):
        """§2.4 first claimed three pairs and two were invented. This pins the
        real number so the claim cannot drift back."""
        producers = []
        for path in sorted(HH.recipes.glob("*.md")):
            recipe = load_recipe(HH, path.stem)
            for v in recipe.variants:
                producers.extend((path.stem, t) for t in v.produces)
            producers.extend((path.stem, t) for t in recipe.produces
                             if "no recipe in the corpus consumes" not in t)
        self.assertEqual([slug for slug, _ in producers], ["chicken-noodle-soup"])


class TestAugustFixture(unittest.TestCase):
    """The week of 2 August. The correct answer was worked out by hand before any
    of this code existed, which is what makes it a fixture and not a snapshot."""

    WEEK = ["crock-pot-italian-beef", "sausage-and-peppers",
            "sheet-pan-chicken-fajitas", "parchment-garlic-butter-salmon"]

    EXPECTED = {
        "bell_pepper": "5",          # 3 cups + 1 green + 1 red
        "garlic": "1 head",          # 4 cloves + 1 tbsp + 3 cloves
        "onion": "2",                # 1 large + 1 cup
        "chuck_roast": "3 lb",
        "chicken_breast": "1 lb",
        "salmon_fillet": "18 oz",    # 6 oz x3, because it serves 1
        "potato": "3",               # 1 x3
        "butter": "2 sticks",        # 3 tbsp x3 = 9 tbsp
        "lime": "2",                 # 3 tbsp
        "italian_sausage": "2",
        "tortilla": "8",
        "beef_broth": "1 can",
    }

    def setUp(self):
        self.week, self.lines, self.unknown, self.merges, self.links, self.scales, self.items = \
            build(HH, self.WEEK, 2.5)
        self.by_item = {l.canonical: l for l in self.lines}

    def test_quantities(self):
        for item, expected in self.EXPECTED.items():
            with self.subTest(item=item):
                self.assertIn(item, self.by_item)
                self.assertEqual(display(self.by_item[item])[0], expected)

    def test_nothing_unrecognised(self):
        self.assertEqual(self.unknown, [])

    def test_staples_do_not_reach_the_buy_list(self):
        for name in ("salt", "pepper", "olive_oil", "vegetable_oil"):
            self.assertTrue(self.items[name].staple)

    def test_coupling_is_computed_not_claimed(self):
        shared = {l.canonical for l in self.lines
                  if len(set(l.sources)) > 1 and not self.items[l.canonical].staple}
        self.assertEqual(shared, {"bell_pepper", "onion", "garlic"})

    def test_the_salmon_is_stranded(self):
        self.assertEqual(set(self.by_item["salmon_fillet"].sources),
                         {"parchment-garlic-butter-salmon"})


if __name__ == "__main__":
    unittest.main(verbosity=2)
