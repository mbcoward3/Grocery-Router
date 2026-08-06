#!/usr/bin/env python3
"""Tests for the store adapter and SKU matching.

    python3 test_store.py

**No credentials and no network.** Matching is pure and takes products as an
argument; the Kroger client is exercised through a stubbed `urlopen`. That is not
only convenience — the cases that matter here are *the store is down*, *the match
is wrong*, and *the token expired mid-list*, and none of those can be summoned on
demand from a real API.

The governing rule, stated once: **a match that is not confident is not made.** A
grocery list is read by a person who can see that `onion` came out wrong. A cart
is a box that arrives.
"""

import io
import json
import os
import unittest
import urllib.error
import urllib.request

import adapters
from adapters import kroger, match
from adapters import Product


def line(canonical, qty=None, unit=None, accepts=(), merged_into=""):
    """A stand-in for `shop.Line` with only the fields matching reads."""
    import shop
    return shop.Line(canonical=canonical, qty=qty, unit=unit,
                     accepts=set(accepts), merged_into=merged_into)


def product(name, price=None, promo=None, size="", brand="", sku="1"):
    return Product(sku=sku, name=name, brand=brand, size=size, price=price, promo=promo)


# --------------------------------------------------------------------------- #

class TestTheFloor(unittest.TestCase):
    """The rule the whole file exists for."""

    def test_onion_powder_does_not_resolve_to_onion(self):
        """The receipt. Thirteen lines resolved this way and put a fresh onion in
        the cart for a teaspoon of spice — and that was the *cheap* version of
        this mistake, because a human read the list first."""
        got = match.best(line("onion_powder"), [product("Yellow Onion", 0.99)])
        self.assertFalse(got.ok)
        self.assertIn("powder", got.why)

    def test_onion_does_not_resolve_to_onion_soup_mix(self):
        """The same mismatch pointing the other way. Every word of the item is
        present and the product is still mostly other things."""
        got = match.best(line("onion"), [product("French Onion Soup Mix", 1.29)])
        self.assertFalse(got.ok)

    def test_dried_thyme_does_not_resolve_to_fresh_thyme(self):
        got = match.best(line("dried_thyme"), [product("Fresh Thyme Bunch", 2.49)])
        self.assertFalse(got.ok)

    def test_a_real_match_is_made(self):
        got = match.best(line("onion"), [product("Yellow Onion", 0.99)])
        self.assertTrue(got.ok)
        self.assertEqual(got.product.name, "Yellow Onion")

    def test_store_branding_is_not_held_against_a_product(self):
        got = match.best(line("ground_beef"),
                         [product("Kroger Brand Fresh Ground Beef", 5.99, brand="Kroger")])
        self.assertTrue(got.ok, got.why)

    def test_a_refusal_says_what_was_on_the_shelf(self):
        """Blank refusals are useless to somebody standing in an aisle."""
        got = match.best(line("onion_powder"),
                         [product("Yellow Onion"), product("Red Onion")])
        self.assertFalse(got.ok)
        self.assertEqual(len(got.alternatives), 2)
        self.assertIn("Yellow Onion", got.why)

    def test_nothing_on_the_shelf_is_a_refusal_not_a_crash(self):
        got = match.best(line("saffron"), [])
        self.assertFalse(got.ok)


class TestWhatTheHouseholdAllows(unittest.TestCase):
    def test_an_accepts_tolerance_widens_the_match(self):
        """`accepts:` is the household's own permission to substitute, stated on
        the recipe line. It is the only thing that widens identity here."""
        got = match.best(line("beef_broth", accepts=["beef_stock"]),
                         [product("Beef Stock", 2.49)])
        self.assertTrue(got.ok)
        self.assertIn("accepted as", got.why)

    def test_without_the_tolerance_the_same_product_is_refused(self):
        got = match.best(line("beef_broth"), [product("Beef Stock", 2.49)])
        self.assertFalse(got.ok)


class TestPackSizing(unittest.TestCase):
    """Proposal §7. Whole packages only — you cannot buy 0.3 of a can."""

    def test_it_buys_enough(self):
        self.assertEqual(match.pack_count(3, "lb", "1 lb"), 3)
        self.assertEqual(match.pack_count(20, "oz", "15.5 oz"), 2)

    def test_it_does_not_over_buy_on_an_exact_fit(self):
        self.assertEqual(match.pack_count(2, "lb", "2 lb"), 1)

    def test_an_unreadable_size_buys_one_and_the_caller_says_so(self):
        """Being short is recoverable. Silently buying six is not."""
        self.assertEqual(match.pack_count(3, "lb", "family size"), 1)
        self.assertEqual(match.pack_count(3, "lb", ""), 1)

    def test_units_that_do_not_compare_buy_one(self):
        self.assertEqual(match.pack_count(2, "cup", "15.5 oz"), 1)


class TestTheCart(unittest.TestCase):
    def shelf(self, **by_term):
        return lambda term: by_term.get(term, [])

    def test_it_matches_what_it_can_and_refuses_the_rest(self):
        cart = match.plan_cart(
            [line("onion", 2, "ea"), line("onion_powder", 1, "tsp")],
            self.shelf(onion=[product("Yellow Onion", 0.99)],
                       **{"onion powder": [product("Yellow Onion", 0.99)]}))
        self.assertEqual(len(cart.lines), 1)
        self.assertEqual(len(cart.unmatched), 1)

    def test_the_total_counts_packages_not_lines(self):
        cart = match.plan_cart([line("ground_beef", 3, "lb")],
                               self.shelf(**{"ground beef":
                                             [product("Ground Beef", 5.00, size="1 lb")]}))
        self.assertEqual(cart.lines[0][2], 3)
        self.assertAlmostEqual(cart.total, 15.00)

    def test_a_sale_price_wins(self):
        cart = match.plan_cart([line("onion", 1, "ea")],
                               self.shelf(onion=[product("Yellow Onion", 2.00, promo=1.00)]))
        self.assertAlmostEqual(cart.total, 1.00)
        self.assertTrue(cart.lines[0][1].on_sale)

    def test_a_merged_line_is_not_bought_twice(self):
        cart = match.plan_cart(
            [line("onion", 2, "ea"), line("white_onion", 1, "ea", merged_into="onion")],
            self.shelf(onion=[product("Yellow Onion", 0.99)]))
        self.assertEqual(len(cart.lines), 1)

    def test_a_store_that_is_down_becomes_unmatched_not_an_exception(self):
        """Degrades, never blocks — the same posture `prep.py` takes. A list that
        raises is worse than a list with no prices on it."""
        def down(term):
            raise adapters.StoreUnavailable("HTTP 503")
        cart = match.plan_cart([line("onion", 1, "ea")], down)
        self.assertEqual(len(cart.unmatched), 1)
        self.assertIn("503", cart.unmatched[0][1])

    def test_nothing_is_ever_submitted(self):
        """Decision 4. Not a promise — there is no credential here that could."""
        cart = match.plan_cart([line("onion", 1, "ea")],
                               self.shelf(onion=[product("Yellow Onion", 0.99)]))
        self.assertFalse(cart.submitted)
        self.assertIn("has been ordered", match.report(cart))

    def test_the_report_leads_with_what_you_are_buying_yourself(self):
        cart = match.plan_cart([line("saffron", 1, "g")], self.shelf())
        out = match.report(cart)
        self.assertIn("Not matched", out)
        self.assertIn("saffron", out)


class TestSelectingAStore(unittest.TestCase):
    def setUp(self):
        self._env = {k: os.environ.get(k) for k in
                     ("KROGER_CLIENT_ID", "KROGER_CLIENT_SECRET")}
        for k in self._env:
            os.environ.pop(k, None)

    def tearDown(self):
        for k, v in self._env.items():
            os.environ.pop(k, None) if v is None else os.environ.__setitem__(k, v)

    def test_no_credentials_is_a_supported_configuration(self):
        """Not a stub to be replaced later — it is what runs in CI and in the
        hosted demo, and the reason neither needs a credential."""
        got = adapters.store()
        self.assertEqual(got.name, "none")
        self.assertEqual(got.search("onion"), [])
        self.assertEqual(got.promotions(["onion"]), [])

    def test_credentials_select_kroger(self):
        os.environ["KROGER_CLIENT_ID"] = "id"
        os.environ["KROGER_CLIENT_SECRET"] = "secret"
        self.assertEqual(adapters.store().name, "kroger")


class TestTheKrogerClient(unittest.TestCase):
    """Through a stubbed `urlopen`. The interesting cases are the failures."""

    def setUp(self):
        kroger._token = None
        self._open = urllib.request.urlopen
        self.addCleanup(setattr, urllib.request, "urlopen", self._open)
        self._env = {k: os.environ.get(k) for k in
                     ("KROGER_CLIENT_ID", "KROGER_CLIENT_SECRET", "KROGER_LOCATION_ID")}
        os.environ["KROGER_CLIENT_ID"] = "id"
        os.environ["KROGER_CLIENT_SECRET"] = "secret"
        self.addCleanup(self.restore)

    def restore(self):
        kroger._token = None
        for k, v in self._env.items():
            os.environ.pop(k, None) if v is None else os.environ.__setitem__(k, v)

    def wire(self, *responses):
        self.calls = []
        queue = list(responses)

        def fake(req, timeout=None):
            self.calls.append(getattr(req, "full_url", str(req)))
            item = queue.pop(0) if queue else {}
            if isinstance(item, Exception):
                raise item
            return io.BytesIO(json.dumps(item).encode())
        urllib.request.urlopen = fake

    TOKEN = {"access_token": "tok", "expires_in": 1800}
    PRODUCTS = {"data": [{
        "productId": "0001111041700", "description": "Yellow Onion", "brand": "Kroger",
        "items": [{"size": "1 lb", "price": {"regular": 1.29, "promo": 0.99},
                   "inventory": {"stockLevel": "HIGH"}}],
        "aisleLocations": [{"description": "Produce"}]}]}

    def test_a_product_is_read_off_the_response(self):
        self.wire(self.TOKEN, self.PRODUCTS)
        got = kroger.Kroger().search("onion")
        self.assertEqual(len(got), 1)
        self.assertEqual(got[0].sku, "0001111041700")
        self.assertEqual(got[0].size, "1 lb")
        self.assertAlmostEqual(got[0].best_price, 0.99)
        self.assertTrue(got[0].on_sale)

    def test_the_token_is_reused_across_searches(self):
        """A shopping list is twenty-odd searches. Minting a token for each
        would be rude and slow."""
        self.wire(self.TOKEN, self.PRODUCTS, self.PRODUCTS)
        store = kroger.Kroger()
        store.search("onion")
        store.search("beef")
        self.assertEqual(sum(1 for c in self.calls if "token" in c), 1)

    def test_a_field_the_store_does_not_send_stays_empty(self):
        """Never filled with a plausible default. A made-up size is worse than a
        missing one when the next step is buying it."""
        self.wire(self.TOKEN, {"data": [{"productId": "9", "description": "Thing",
                                         "items": [{}]}]})
        got = kroger.Kroger().search("thing")[0]
        self.assertEqual(got.size, "")
        self.assertIsNone(got.price)
        self.assertIsNone(got.best_price)

    def test_being_down_raises_one_type(self):
        self.wire(self.TOKEN, urllib.error.URLError("connection refused"))
        with self.assertRaises(adapters.StoreUnavailable):
            kroger.Kroger().search("onion")

    def test_a_rejected_credential_says_so(self):
        self.wire(urllib.error.HTTPError("u", 401, "Unauthorized", {}, None))
        with self.assertRaises(adapters.StoreUnavailable) as e:
            kroger.Kroger().search("onion")
        self.assertIn("401", str(e.exception))

    def test_promotions_skip_a_term_that_fails_rather_than_giving_up(self):
        """The briefing degrades. One unreachable term must not cost the card."""
        self.wire(self.TOKEN, urllib.error.URLError("boom"), self.PRODUCTS)
        got = kroger.Kroger().promotions(["beef", "onion"])
        self.assertEqual(len(got), 1)

    def test_only_products_actually_on_sale_are_promotions(self):
        flat = {"data": [{"productId": "2", "description": "Onion",
                          "items": [{"price": {"regular": 1.29}}]}]}
        self.wire(self.TOKEN, flat)
        self.assertEqual(kroger.Kroger().promotions(["onion"]), [])


if __name__ == "__main__":
    unittest.main(verbosity=2)
