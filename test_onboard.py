#!/usr/bin/env python3
"""Tests for the ingredient grammar and the never-invent rules.

    python3 test_onboard.py

The cases are the ones docs/step2-design.md 2 names as known-hard, plus the
ones the twenty-three real recipes turned up. Standard library only.
"""

import unittest

from onboard import (
    ensure_yield_column,
    infer_passive,
    infer_cuisine,
    infer_protein,
    parse_block,
    parse_ingredient,
    scan_peanut,
    slugify,
)


def p(line):
    return parse_ingredient(line)


class TestGrammar(unittest.TestCase):
    def test_qty_unit_item_note(self):
        r = p("3 cups bell peppers, sliced")
        self.assertEqual((r["qty"], r["unit"], r["item"], r["note"]),
                         ("3", "cup", "bell peppers", "sliced"))

    def test_unit_omitted_means_each(self):
        r = p("1 large white onion, sliced")
        self.assertEqual((r["qty"], r["unit"], r["item"]), ("1", "each", "large white onion"))

    def test_no_quantity_is_flagged_not_guessed(self):
        r = p("Onion")
        self.assertIsNone(r["qty"])
        self.assertIn("no-quantity", r["flags"])
        self.assertEqual(r["item"], "Onion")

    def test_nested_quantity_can_is_the_unit(self):
        r = p("1 (14.5 oz) can beef broth")
        self.assertEqual((r["qty"], r["unit"], r["size"], r["item"]),
                         ("1", "can", "14.5 oz", "beef broth"))

    def test_nested_quantity_without_parens(self):
        r = p("1 10.75 oz can cream of chicken soup")
        self.assertEqual((r["qty"], r["unit"], r["size"], r["item"]),
                         ("1", "can", "10.75 oz", "cream of chicken soup"))

    def test_nested_quantity_plural_container(self):
        r = p("2 13 oz cans cooked and shredded chicken")
        self.assertEqual((r["qty"], r["unit"], r["size"]), ("2", "can", "13 oz"))

    def test_bare_size_is_not_eaten_when_it_is_the_measure(self):
        # `8 oz pepperoncini` - the oz IS the quantity, there is no container
        r = p("8 oz pepperoncini pepper slices, plus extra for serving")
        self.assertEqual((r["qty"], r["unit"], r["size"]), ("8", "oz", None))
        self.assertIn("second-unmeasured-quantity", r["flags"])

    def test_packaging_defined_unit(self):
        r = p("1 envelope Italian salad dressing mix")
        self.assertEqual((r["qty"], r["unit"], r["item"]),
                         ("1", "envelope", "Italian salad dressing mix"))

    def test_quantity_as_a_source_object(self):
        r = p("juice of 1 lemon")
        self.assertIn("quantity-as-source-object", r["flags"])
        self.assertEqual(r["raw"], "juice of 1 lemon")

    def test_to_taste_is_kept_and_flagged(self):
        r = p("salt, to taste")
        self.assertIsNone(r["qty"])
        self.assertIn("to-taste", r["flags"])
        self.assertIn("likely-staple", r["flags"])

    def test_vulgar_fractions(self):
        self.assertEqual(p("⅔ cup water")["qty"], "2/3")
        self.assertEqual(p("½ teaspoon onion powder")["unit"], "tsp")

    def test_ranges(self):
        self.assertEqual(p("4-5 medium Russet potatoes, peeled")["qty"], "4-5")
        self.assertEqual(p("1-2 tsp yellow mustard")["qty"], "1-2")

    def test_no_space_between_number_and_unit(self):
        r = p("2lb ground beef")
        self.assertEqual((r["qty"], r["unit"], r["item"]), ("2", "lb", "ground beef"))

    def test_bullet_markers_are_stripped_but_the_line_is_kept(self):
        r = p("● 1 lb ground beef")
        self.assertEqual(r["item"], "ground beef")
        # the bullet is markup; everything after it survives verbatim
        self.assertEqual(r["raw"], "1 lb ground beef")

    def test_an_odd_item_name_is_asked_about_not_resolved(self):
        r = p("Soup sauce")
        self.assertIn("ambiguous-item", r["flags"])
        self.assertEqual(r["item"], "Soup sauce")

    def test_a_normal_two_word_item_is_not_flagged(self):
        self.assertNotIn("ambiguous-item", p("Marinara sauce")["flags"])
        self.assertNotIn("ambiguous-item", p("1 oz pkt La Preferida Taco Seasoning")["flags"])

    def test_a_comma_inside_the_item_is_not_the_note(self):
        r = p("2 lb boneless, skinless chicken thighs, cubed")
        self.assertEqual(r["item"], "boneless, skinless chicken thighs")
        self.assertEqual(r["note"], "cubed")

    def test_a_comma_inside_brackets_is_not_a_split(self):
        r = p("8 (8-inch) tortillas (We much prefer flour tortillas in this "
              "recipe, but corn are more traditional.)")
        self.assertIsNone(r["note"])
        self.assertTrue(r["item"].startswith("tortillas ("))

    def test_of_belongs_to_the_unit(self):
        self.assertEqual(p("2 tbsp of chili powder")["item"], "chili powder")
        self.assertEqual(p("Jar of Pepperocinis")["item"], "Pepperocinis")

    def test_hyphenated_size_in_parens(self):
        r = p("1 (10-ounce) can red enchilada sauce")
        self.assertEqual((r["qty"], r["unit"], r["item"]),
                         ("1", "can", "red enchilada sauce"))


class TestNeverInvent(unittest.TestCase):
    def test_missing_yield_stays_unknown(self):
        rec = parse_block("# Hamburgers\n\n2lb ground beef\nBuns\n", "text")
        self.assertIsNone(rec["yield"])
        self.assertEqual(rec["yield_note"], "not stated in source")

    def test_a_bang_note_marks_the_capture_partial(self):
        rec = parse_block(
            "# Chili\n\n## Ingredients\n\n- 1 lb ground hamburger\n\n"
            "## Capture notes\n\n- ! the list may be cut off at the bottom\n",
            "screenshot")
        self.assertEqual(rec["status"], "partial")
        self.assertEqual(len(rec["questions"]), 1)

    def test_a_question_note_asks_without_claiming_content_is_missing(self):
        rec = parse_block(
            "# Chili\n\n## Ingredients\n\n- 1 lb ground hamburger\n\n"
            "## Capture notes\n\n- ? servings are not shown\n"
            "- the site was julieseatsandtreats.com\n", "screenshot")
        self.assertEqual(rec["status"], "complete")
        self.assertEqual(len(rec["questions"]), 1)

    def test_an_ingredient_sub_group_is_kept(self):
        rec = parse_block(
            "# Meatball subs\n\n## Ingredients\n\n- 4 hoagie rolls\n\n"
            "### Meatballs\n\n- 1 pound lean ground beef\n", "screenshot")
        self.assertIsNone(rec["ingredients"][0]["group"])
        self.assertEqual(rec["ingredients"][1]["group"], "Meatballs")

    def test_clean_capture_is_complete(self):
        rec = parse_block("# Chili\n\n## Ingredients\n\n- 1 lb ground hamburger\n",
                          "screenshot")
        self.assertEqual(rec["status"], "complete")

    def test_no_ingredients_is_failed_not_empty_success(self):
        rec = parse_block("# Mystery\n", "screenshot")
        self.assertEqual(rec["status"], "failed")
        self.assertTrue(rec["questions"])


class TestInference(unittest.TestCase):
    def test_protein_from_ingredients(self):
        ings = [p("1 lb ground beef"), p("1 onion")]
        self.assertEqual(infer_protein(ings)[0], "beef")

    def test_a_broth_is_not_a_protein(self):
        ings = [p("1/2 pound ground mild Italian sausage"), p("4 cups chicken broth")]
        self.assertEqual(infer_protein(ings)[0], "pork")
        self.assertIsNone(infer_protein([p("4 cups chicken broth")])[0])

    def test_protein_unknown_rather_than_guessed(self):
        ings = [p("1 onion"), p("2 cups rice")]
        self.assertIsNone(infer_protein(ings)[0])

    def test_one_marker_does_not_name_a_cuisine(self):
        self.assertIsNone(infer_cuisine([p("1 cup soy sauce")])[0])
        self.assertIsNone(infer_cuisine([p("1-2 tbsp chopped pepperoncini")])[0])

    def test_two_markers_do(self):
        cuisine, evidence = infer_cuisine(
            [p("1 oz pkt La Preferida Taco Seasoning"), p("Flour tortillas")])
        self.assertEqual(cuisine, "Tex-Mex")
        self.assertEqual(len(evidence), 2)

    def test_peanut_ingredient_is_flagged(self):
        verdict, _ = scan_peanut([p("2 tbsp peanut butter")])
        self.assertEqual(verdict, "CONTAINS PEANUT")

    def test_bought_sauce_is_a_check_not_a_pass(self):
        verdict, evidence = scan_peanut([p("1/2 cup teriyaki sauce")])
        self.assertEqual(verdict, "check label")
        self.assertTrue(evidence)

    def test_clean_recipe_says_none_seen(self):
        self.assertEqual(scan_peanut([p("1 lb ground beef")])[0], "none seen")


class TestPassive(unittest.TestCase):
    def test_a_dutch_oven_is_not_an_oven(self):
        self.assertEqual(
            infer_passive("In a dutch oven over medium heat add hamburger, "
                          "then simmer for 20 minutes"), "simmer")

    def test_a_real_oven_is(self):
        self.assertEqual(infer_passive("Bake at 350F for 20 minutes"), "oven")

    def test_slow_cooker_wins(self):
        self.assertEqual(
            infer_passive("Brown the roast, then cook in your slow cooker on "
                          "low for 8 hours"), "slow cooker")

    def test_nothing_claimed_without_a_method_word(self):
        self.assertIsNone(infer_passive("Combine tuna and mayo in a bowl."))


class TestCorpus(unittest.TestCase):
    def test_yield_column_is_added_once(self):
        lines = [
            "| Recipe | Protein | Cuisine | Active | Passive | Last cooked | Notes |",
            "|---|---|---|---|---|---|---|",
            "| Chili | beef | American | low | simmer | | |",
        ]
        out, header, pos = ensure_yield_column(lines, 0, [
            "Recipe", "Protein", "Cuisine", "Active", "Passive", "Last cooked", "Notes"])
        self.assertEqual(header[pos], "Yield")
        self.assertEqual(pos, 3)
        again, header2, pos2 = ensure_yield_column(out, 0, header)
        self.assertEqual(again, out)
        self.assertEqual(pos2, 3)

    def test_slug(self):
        self.assertEqual(slugify("Beef dip Sammies"), "beef-dip-sammies")


if __name__ == "__main__":
    unittest.main(verbosity=2)
