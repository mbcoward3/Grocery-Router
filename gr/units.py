"""Units, and conversion between them.

Two kinds of unit live here.

**Measures** convert universally. A tablespoon is three teaspoons in every recipe and for
every item, so those factors are hard-coded. Volume and weight are separate families and
nothing bridges them here, because nothing can: a cup of flour and a cup of broth do not
weigh the same.

**Counts** convert per item, never universally. A clove is a tenth of a head of garlic and
means nothing at all applied to celery. Those factors come from the `each_equiv` column of
`items.md`, one row at a time, and `gr.items` owns reading them.

The bridge between the two families is also per item and also comes from `each_equiv` —
`1 can = 14.5 oz` for a broth, `1 ea = 2.5 oz` for a carrot.
"""

from __future__ import annotations

import math
from collections import deque

# --- unit vocabulary -------------------------------------------------------

# Every alias on the left resolves to the canonical name on the right. Anything not in
# here is not a unit, which is how the parser tells `2 cups water` from `2 large eggs`.
_ALIASES = {
    # volume
    "tsp": "tsp", "tsps": "tsp", "t": "tsp", "teaspoon": "tsp", "teaspoons": "tsp",
    "tbsp": "tbsp", "tbsps": "tbsp", "tb": "tbsp", "tbs": "tbsp",
    "tablespoon": "tbsp", "tablespoons": "tbsp",
    "cup": "cup", "cups": "cup", "c": "cup",
    "floz": "floz", "fluidounce": "floz", "fluidounces": "floz",
    "pint": "pint", "pints": "pint", "pt": "pint",
    "quart": "quart", "quarts": "quart", "qt": "quart",
    "gallon": "gallon", "gallons": "gallon", "gal": "gallon",
    "ml": "ml", "milliliter": "ml", "milliliters": "ml",
    "l": "liter", "liter": "liter", "liters": "liter", "litre": "liter",
    # weight
    "oz": "oz", "ozs": "oz", "ounce": "oz", "ounces": "oz",
    "lb": "lb", "lbs": "lb", "pound": "lb", "pounds": "lb",
    "g": "g", "gram": "g", "grams": "g",
    "kg": "kg", "kilogram": "kg", "kilograms": "kg",
    # counts and containers
    "ea": "ea", "each": "ea",
    "can": "can", "cans": "can",
    "package": "package", "packages": "package", "pkg": "package", "pkgs": "package",
    "packet": "packet", "packets": "packet", "pkt": "packet", "pkts": "packet",
    "envelope": "envelope", "envelopes": "envelope",
    "jar": "jar", "jars": "jar",
    "box": "box", "boxes": "box",
    "tube": "tube", "tubes": "tube",
    "bunch": "bunch", "bunches": "bunch",
    "head": "head", "heads": "head",
    "clove": "clove", "cloves": "clove",
    "rib": "rib", "ribs": "rib",
    "stalk": "stalk", "stalks": "stalk",
    "slice": "slice", "slices": "slice",
    "sprig": "sprig", "sprigs": "sprig",
    "stick": "stick", "sticks": "stick",
    "brick": "brick", "bricks": "brick",
    "loaf": "loaf", "loaves": "loaf",
    "bag": "bag", "bags": "bag",
    "bottle": "bottle", "bottles": "bottle",
    "fillet": "fillet", "fillets": "fillet",
}

# Units that name a container or a count. A number in front of one of these is how many
# you buy, never how much the recipe measured out.
CONTAINERS = {
    "can", "package", "packet", "envelope", "jar", "box", "tube", "bunch", "head",
    "clove", "rib", "stalk", "slice", "sprig", "stick", "brick", "loaf", "bag",
    "bottle", "fillet", "ea",
}

# Universal factors, expressed in the family's base unit.
_VOLUME = {
    "tsp": 1.0, "tbsp": 3.0, "floz": 6.0, "cup": 48.0,
    "pint": 96.0, "quart": 192.0, "gallon": 768.0,
    "ml": 0.202884, "liter": 202.884,
}
_WEIGHT = {"oz": 1.0, "lb": 16.0, "g": 0.035274, "kg": 35.274}

FAMILIES = {"volume": _VOLUME, "weight": _WEIGHT}

# The measures a shopping list is allowed to promote into. Everything else is a legal
# input unit but a poor thing to read in an aisle.
SHOPPING_UNITS = {"tsp", "tbsp", "cup", "oz", "lb"}


def normalize_unit(token: str) -> str | None:
    """Return the canonical unit name for a token, or None when it is not a unit."""
    if not token:
        return None
    key = token.strip().lower().rstrip(".").replace(" ", "").replace("-", "")
    return _ALIASES.get(key)


def is_unit(token: str) -> bool:
    return normalize_unit(token) is not None


def family_of(unit: str | None) -> str | None:
    if unit in _VOLUME:
        return "volume"
    if unit in _WEIGHT:
        return "weight"
    return None


def is_count(unit: str | None) -> bool:
    return unit in CONTAINERS


# --- conversion ------------------------------------------------------------

def build_graph(each_equiv: list[tuple[float, str, float, str]]) -> dict[str, dict[str, float]]:
    """Build one item's conversion graph.

    `each_equiv` arrives as `(left_qty, left_unit, right_qty, right_unit)` tuples read off
    the item's row — `1 head = 10 cloves` is `(1, 'head', 10, 'clove')`. Every clause
    becomes an edge in both directions, which is what lets clauses chain: `1 head = 10
    cloves` and `1 clove = 1 tsp` together answer *how many heads is a tablespoon*.

    The universal measure factors go in as edges too, so one search handles both kinds.
    """
    graph: dict[str, dict[str, float]] = {}

    def edge(a: str, b: str, factor: float) -> None:
        # factor is: 1 unit of `a` equals `factor` units of `b`.
        if factor <= 0:
            return
        graph.setdefault(a, {})[b] = factor
        graph.setdefault(b, {})[a] = 1.0 / factor

    for table in (_VOLUME, _WEIGHT):
        units = list(table)
        base = units[0]
        for unit in units[1:]:
            edge(unit, base, table[unit] / table[base])

    for lq, lu, rq, ru in each_equiv:
        if lq > 0 and rq > 0 and lu and ru:
            edge(lu, ru, rq / lq)

    return graph


def convert(qty: float, frm: str | None, to: str | None,
            graph: dict[str, dict[str, float]]) -> float | None:
    """Convert `qty` from one unit to another, or return None when nothing bridges them.

    None is a real answer and the caller must keep it. Inventing a factor is how a
    shopping list acquires a number nobody can check.
    """
    if frm == to:
        return qty
    if frm is None or to is None:
        return None
    if frm not in graph or to not in graph:
        return None

    # Breadth-first, so the shortest chain of clauses wins and the factor is stable.
    seen = {frm: 1.0}
    queue = deque([frm])
    while queue:
        node = queue.popleft()
        for nxt, factor in graph.get(node, {}).items():
            if nxt in seen:
                continue
            seen[nxt] = seen[node] * factor
            if nxt == to:
                return qty * seen[nxt]
            queue.append(nxt)
    return None


def reachable(frm: str | None, graph: dict[str, dict[str, float]]) -> set[str]:
    """Every unit `frm` can convert into, itself included."""
    if frm is None:
        return set()
    if frm not in graph:
        return {frm}
    seen = {frm}
    queue = deque([frm])
    while queue:
        node = queue.popleft()
        for nxt in graph.get(node, {}):
            if nxt not in seen:
                seen.add(nxt)
                queue.append(nxt)
    return seen


# --- presentation ----------------------------------------------------------

def tidy(qty: float, unit: str | None,
         graph: dict[str, dict[str, float]]) -> tuple[float, str | None]:
    """Round a quantity for a shopper, and promote it to a unit worth reading.

    Counts round **up**, because half a can is a whole can in the cart. The 0.15
    tolerance stops nine cloves of garlic from asking for two heads: it rounds 0.9 heads
    to one head and 1.1 heads to one head, and 1.5 to two.

    Measures round to two decimals and promote to the largest unit that leaves at least
    one of them, so 9 teaspoons prints as 3 tablespoons. It does not pretend to be
    tidier than that — `0.4 lb potato` is crude and it is also the truth.
    """
    if qty is None:
        return qty, unit

    if is_count(unit):
        rounded = math.ceil(qty - 0.15)
        return (max(1, rounded) if qty > 0 else 0), unit

    family = family_of(unit)
    if family:
        table = FAMILIES[family]
        # Promote only into units a shopping list should print. A fluid ounce is larger
        # than a tablespoon and would win on size alone, but nobody buys chopped pickles
        # by the fluid ounce. The line's own unit always stays a candidate.
        candidates = [u for u in table if u in SHOPPING_UNITS or u == unit]
        best_unit, best_qty = unit, qty
        for candidate in sorted(candidates, key=lambda u: table[u], reverse=True):
            converted = convert(qty, unit, candidate, graph)
            if converted is not None and converted >= 1.0:
                best_unit, best_qty = candidate, converted
                break
        return round(best_qty, 2), best_unit

    return round(qty, 2), unit


def format_qty(qty: float | None) -> str:
    """Print a number the way a list should: no trailing zeros, no false precision."""
    if qty is None:
        return ""
    if abs(qty - round(qty)) < 0.005:
        return str(int(round(qty)))
    return f"{qty:.2f}".rstrip("0").rstrip(".")
