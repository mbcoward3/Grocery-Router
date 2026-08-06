"""Canonical item → a product somebody would actually buy.

The hard half of the Kroger step, and the one with the sharpest failure mode. A
grocery list is read by a person who can see that `onion` came out wrong; a cart
is a box that arrives. So this is deterministic, it has no model anywhere near
it, and its governing rule is the one the whole project keeps arriving at from
different directions:

> **A match that is not confident is not made.**

`onion powder` resolving to `onion` across thirteen lines put a fresh onion in the
cart for a teaspoon of spice. That mistake was *free* - it cost one wasted
vegetable in a list a human reads before shopping. The same mistake against a
cart costs money, on an item nobody chose. So the floor is set where a partial
match is refused, `None` comes back with a sentence saying what was wanted, and
the household settles it in the aisle.

**A gap in a cart is a smaller failure than a stranger's guess in it.** Every
other rule here is a consequence of that one.

Standard library only. Nothing in this file performs I/O.
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field

# Words a store puts in a product name that say nothing about what it is. Kept
# out of scoring so `Kroger® Brand Large Yellow Onion, 1 lb` and `onion` are not
# penalised for the store's own copywriting.
NOISE = {
    "kroger", "brand", "private", "selection", "simple", "truth", "organic",
    "fresh", "natural", "premium", "quality", "classic", "original", "value",
    "great", "grade", "usda", "choice", "family", "size", "pack", "count", "ct",
    "each", "per", "lb", "oz", "fl", "the", "and", "with", "of", "a", "in",
}

# The line's own words that must survive into the product name. A quantity or a
# unit is not identity - `2 lb ground beef` and `1 lb ground beef` are the same
# item at different amounts - but `ground` and `beef` both are.
STRIP_UNITS = re.compile(
    r"^\s*[\d./\s-]+\s*(lb|lbs|oz|ounce|ounces|g|kg|ml|l|cup|cups|can|cans|"
    r"package|packages|jar|jars|bunch|head|clove|cloves|ea|each)?\b", re.I)

# **The floor.** Every word of the item has to appear in the product name, and at
# least this share of the product's own meaningful words has to be accounted for.
# Below it there is no match, by design.
MIN_COVERAGE = 0.5


@dataclass
class Match:
    """A product, or an honest refusal. Never a nearest guess."""
    item: str
    product: object | None = None
    qty: int = 1
    why: str = ""
    confidence: float = 0.0
    alternatives: list = field(default_factory=list)

    @property
    def ok(self) -> bool:
        return self.product is not None


def words(text: str) -> list[str]:
    text = STRIP_UNITS.sub("", (text or "").lower())
    text = re.sub(r"[^a-z0-9 ]+", " ", text)
    return [w for w in text.split() if w and w not in NOISE]


def canonical_words(canonical: str) -> list[str]:
    """`items.md` writes canonical names with underscores: `bell_pepper`."""
    return [w for w in re.split(r"[_\s]+", (canonical or "").lower()) if w]


def score(item_words: list[str], product_name: str) -> tuple[float, str]:
    """`(confidence, why)` for one candidate. Zero means refuse.

    **Every word of the item must be present.** This is the rule that would have
    caught `onion powder` → `onion`: the product `Yellow Onion` does not contain
    `powder`, so it scores zero rather than scoring well on one word out of two.
    A partial match is only ever accepted when the words it leaves behind are
    noise, which `NOISE` is the list of.
    """
    prod = words(product_name)
    if not prod or not item_words:
        return 0.0, "nothing to compare"
    missing = [w for w in item_words if w not in prod]
    if missing:
        return 0.0, f"the product does not say {', '.join(missing)}"
    # How much of the *product* the item accounts for. `onion` against
    # `French Onion Soup Mix` matches every item word and still leaves `soup`
    # and `mix` unexplained - the direction of the mismatch that a one-way check
    # misses entirely.
    extra = [w for w in prod if w not in item_words]
    coverage = len(item_words) / max(1, len(item_words) + len(extra))
    if coverage < MIN_COVERAGE:
        return 0.0, f"the product is mostly other things: {', '.join(extra[:3])}"
    return coverage, ("exact" if not extra else f"close — also says {', '.join(extra[:2])}")


def pack_count(needed_qty: float | None, needed_unit: str | None, size: str) -> int:
    """How many packages cover the amount wanted. Whole packages only.

    §7 of the proposal, and where the `size` string finally earns its keep. When
    the store's size cannot be read, or the units do not compare, this returns 1
    and the caller says so - **buying one and being short is recoverable, and
    silently buying six is not.**
    """
    if not needed_qty or not needed_unit:
        return 1
    m = re.search(r"(\d+(?:\.\d+)?)\s*(oz|ounce|ounces|lb|lbs|pound|pounds|g|ml|l|ct|count)",
                  (size or "").lower())
    if not m:
        return 1
    have, unit = float(m.group(1)), m.group(2)
    scale = {"oz": 1.0, "ounce": 1.0, "ounces": 1.0,
             "lb": 16.0, "lbs": 16.0, "pound": 16.0, "pounds": 16.0}
    want_unit = needed_unit.lower()
    if unit not in scale or want_unit not in scale:
        return 1
    want_oz = needed_qty * scale[want_unit]
    have_oz = have * scale[unit]
    if have_oz <= 0:
        return 1
    import math
    return max(1, math.ceil(want_oz / have_oz - 1e-9))


def best(line, products: list) -> Match:
    """The product for one shopping-list line, or a refusal with a reason.

    `line` is a `shop.Line` - it carries the canonical name, the amount, and the
    `accepts:` tolerances the household stated on the recipe. Those tolerances
    are the household's own permission to substitute, so they widen what counts
    as the item and nothing else does.
    """
    names = [line.canonical] + sorted(getattr(line, "accepts", set()) or [])
    best_match = Match(item=line.canonical, why="nothing on the shelf matched")

    for name in names:
        item_words = canonical_words(name)
        for product in products:
            confidence, why = score(item_words, f"{product.name} {product.brand}")
            if confidence <= 0:
                continue
            # Ties break toward the cheaper product. Not a preference - a tie
            # here means the names are equally good and there is nothing else to
            # decide on, and the household's own corpus is full of shortcuts and
            # store brands rather than premium ones.
            price = product.best_price if product.best_price is not None else float("inf")
            better = (confidence > best_match.confidence
                      or (confidence == best_match.confidence and best_match.product
                          and price < (best_match.product.best_price or float("inf"))))
            if better:
                best_match = Match(item=line.canonical, product=product,
                                   qty=pack_count(line.qty, line.unit, product.size),
                                   why=why + ("" if name == line.canonical
                                              else f" (accepted as {name})"),
                                   confidence=confidence)

    if not best_match.ok:
        # What *was* on the shelf, so the refusal is useful rather than blank.
        # Somebody standing in the aisle can act on "we found these three".
        best_match.alternatives = [p.name for p in products[:3]]
        if products:
            best_match.why = (f"nothing matched {line.canonical} closely enough — "
                              f"found {', '.join(best_match.alternatives)}")
    return best_match


def plan_cart(lines: list, products_for, store_name: str = "") -> "object":
    """What would go in the cart, and what could not. **Never submits.**

    `products_for` is a callable taking a search term and returning products -
    the seam that keeps this file free of I/O and lets every case here be tested
    without a credential.

    Staples are skipped rather than matched. `items.md` marks them because the
    household almost certainly has salt, and putting salt in a cart every week is
    the tool being confidently unhelpful.
    """
    from . import Cart, StoreUnavailable

    cart = Cart(store=store_name)
    for line in lines:
        if getattr(line, "merged_into", ""):
            continue
        try:
            products = products_for(line.canonical.replace("_", " "))
        except StoreUnavailable as exc:
            cart.unmatched.append((line, f"the store could not be reached: {exc}"))
            continue
        match = best(line, products)
        if not match.ok:
            cart.unmatched.append((line, match.why))
            continue
        cart.lines.append((line, match.product, match.qty))
        price = match.product.best_price
        if price is not None:
            cart.total += price * match.qty
    return cart


def report(cart) -> str:
    """The cart as markdown, for a person to check before they buy anything.

    The unmatched section is not an appendix. It is the part somebody has to read
    - every line in it is a thing they will otherwise get home without.
    """
    out = [f"## Cart — {cart.store or 'no store configured'}", ""]
    if not cart.lines and not cart.unmatched:
        out.append("_Nothing to match._")
        return "\n".join(out)

    for line, product, qty in cart.lines:
        price = product.best_price
        money = f"${price * qty:.2f}" if price is not None else "no price"
        sale = " **on sale**" if product.on_sale else ""
        out.append(f"- {qty} × {product.name}"
                   + (f" ({product.size})" if product.size else "")
                   + f" — {money}{sale}  ·  for {line.canonical.replace('_', ' ')}")
    if cart.total:
        out += ["", f"**{len(cart.lines)} item(s), about ${cart.total:.2f}.** "
                    f"Prices are what the store published, and are not a quote."]
    if cart.unmatched:
        out += ["", "### Not matched — you are buying these yourself", "",
                "*Nothing was guessed. A wrong item in a cart costs money on "
                "something nobody chose, so an uncertain match is refused rather "
                "than made.*", ""]
        for line, why in cart.unmatched:
            out.append(f"- **{line.canonical.replace('_', ' ')}** — {why}")
    out += ["", "---", "",
            "**Nothing here has been ordered.** The tool fills a cart and a person "
            "submits it — that is a decision recorded in `docs/architecture.md`, and "
            "the credentials this runs on cannot submit anything even if it tried."]
    return "\n".join(out)
