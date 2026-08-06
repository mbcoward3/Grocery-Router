"""Stores, behind one interface, so a second one is a new file.

`docs/architecture.md` reserved this directory and left two questions open on
purpose: *how Kroger is actually talked to*, and *what the product does when the
SKU match is wrong or the API is down*. This package answers both, and the
answers are the design rather than a footnote to it.

---

## How the store is talked to

Kroger publishes a developer API. Client credentials get a token for product and
location lookups; a **user** authorization is separately required to touch a
cart. That split is not an inconvenience, it is the decision this project already
made saying itself back: decision 4, *the tool fills a cart and a human submits
it.* There is no credential anywhere in this codebase that could spend money
unattended, and that is a property of Kroger's own auth model rather than of our
good intentions.

Configured by environment, never by a file in the repo:

    KROGER_CLIENT_ID, KROGER_CLIENT_SECRET     product search and prices
    KROGER_LOCATION_ID                          which store's shelf and prices
    KROGER_USER_TOKEN                           optional; required only to write a cart

**No credentials is the normal case**, exactly as no API key is for the planner.
Unconfigured, `store()` returns `NoStore`, every price comes back absent and
labelled absent, and the shopping list is the same list it has always been.

## What happens when it is wrong or down

Two different failures, and conflating them would be the mistake.

**Down** is easy and the posture is already written: *degrades, never blocks* -
the same rule `prep.py` follows. A timeout, a 500, an expired token and a missing
credential all end the same way: no prices, a line saying why, and a grocery list
that still gets somebody to the shop. Nothing in Step 2 depends on a network.

**Wrong is the dangerous one**, and it gets the stricter rule: **a match that is
not confident is not made.** `onion powder` resolving to `onion` put a fresh onion
in the cart for a teaspoon of spice, and that was a *free* mistake - it cost one
wasted vegetable in a list a human reads. The same class of error against a cart
costs money, on an item nobody chose, in a box that arrives. So the matcher
returns `None` and says what it was looking for, and the household resolves it in
the aisle. **A gap in a cart is a smaller failure than a stranger's guess in it.**

That asymmetry is why matching is deterministic and lives in `adapters/match.py`
with no model anywhere near it, and why the confidence floor is set where a
partial match is refused rather than accepted.

Standard library only.
"""

from __future__ import annotations

import os
from dataclasses import dataclass, field


@dataclass
class Product:
    """One thing on a shelf, as a store describes it."""
    sku: str
    name: str
    brand: str = ""
    size: str = ""              # the store's own text, e.g. "15.5 oz"
    price: float | None = None
    promo: float | None = None  # promotional price, when there is one
    aisle: str = ""
    stock: str = ""             # store's own words; "" when it does not say

    @property
    def best_price(self) -> float | None:
        if self.promo is not None and self.price is not None:
            return min(self.promo, self.price)
        return self.promo if self.promo is not None else self.price

    @property
    def on_sale(self) -> bool:
        return (self.promo is not None and self.price is not None
                and self.promo < self.price)


@dataclass
class Cart:
    """What would be added, and what could not be. Never submitted from here."""
    lines: list = field(default_factory=list)      # (Line, Product, qty)
    unmatched: list = field(default_factory=list)  # (Line, why)
    total: float = 0.0
    store: str = ""
    submitted: bool = False                        # never True. Decision 4.


class StoreUnavailable(Exception):
    """The store could not be reached, or is not configured.

    One type for every failure between here and a price, because every one has
    the same consequence: a list with no prices on it, which is the list this
    project has always produced.
    """


class Store:
    """What a store has to be able to do. Two methods and neither one buys."""

    name = "store"
    configured = False

    def search(self, term: str, limit: int = 8) -> list[Product]:
        raise NotImplementedError

    def promotions(self, terms: list[str]) -> list[Product]:
        """Whatever of `terms` is on sale. Used by the Step 0 briefing."""
        raise NotImplementedError


class NoStore(Store):
    """The current state, and a supported one.

    Not a stub to be replaced later - it is what runs with no credentials, in
    CI, and in the hosted demo, and it is the reason none of those need any. It
    returns nothing rather than raising, because *no prices* is a normal answer
    and callers should not have to guard a search that cannot fail.
    """

    name = "none"
    configured = False

    def search(self, term, limit=8):
        return []

    def promotions(self, terms):
        return []


def store() -> Store:
    """The configured store, or `NoStore`.

    Selection by credential rather than by flag, the same way the planner picks
    itself: a household that has set one gets prices, and one that has not gets
    the tool it already had, with no configuration to discover.
    """
    from . import kroger
    if kroger.configured():
        return kroger.Kroger()
    return NoStore()
