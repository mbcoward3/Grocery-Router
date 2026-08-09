"""Turning a week of meals into a shopping list.

**No model produces a line of this file's output.** Every quantity, every conversion,
every merge and every aisle here is arithmetic on the recipe files. A model picks which
meals go in the week; from that point on it is not consulted, cannot be, and the planner
process it runs in has no file access at all.

Three channels come out, and a line is always in exactly one of them:

- **buy** — a resolved, non-staple item with a quantity
- **probably have** — a resolved item flagged `staple` in `items.md`. Routed, not dropped:
  `salt, to taste` must not reach the list as a thing to buy and must not vanish either.
- **unknown** — a line the parser refused or `items.md` has no row for. Printed verbatim
  with the meal it came from. An unknown line is the mechanism working; a mis-merge is
  the mechanism failing silently.
"""

from __future__ import annotations

from dataclasses import dataclass, field

from . import units as U
from .recipes import Recipe, Scale, multiplier_for
from .repo import Repo


@dataclass
class Contribution:
    meal_slug: str
    meal_title: str
    qty: float | None
    unit: str | None
    raw: str
    scaled: bool


@dataclass
class ListLine:
    item: str
    aisle: str
    qty: float | None
    unit: str | None
    sources: list[str]                      # meal titles, in week order
    extras: list[str] = field(default_factory=list)   # amounts no conversion could add
    flags: list[str] = field(default_factory=list)
    contributions: list[Contribution] = field(default_factory=list)

    @property
    def stranded(self) -> bool:
        return len(self.sources) == 1

    def quantity_text(self) -> str:
        """The amount, as a shopper reads it.

        A line always names one thing. Where two amounts could not be added, they are
        both on this one line — never two lines that read as two products.
        """
        if self.qty is None and not self.extras:
            return ""
        head = f"{U.format_qty(self.qty)} {self.unit or ''}".strip() if self.qty is not None else ""
        parts = [p for p in [head] + self.extras if p]
        return " + ".join(parts)

    def display(self) -> str:
        qty = self.quantity_text()
        return f"{qty} {self.item}".strip() if qty else self.item


@dataclass
class UnknownLine:
    meal_slug: str
    meal_title: str
    raw: str
    reason: str


@dataclass
class MealPlan:
    slug: str
    title: str
    reason_kind: str
    reason: str
    yield_raw: str
    scale: Scale
    untried: bool = False
    dropped: str = ""

    @property
    def label(self) -> str:
        return f"{self.title} [candidate]" if self.untried else self.title

    @property
    def multiplier_text(self) -> str:
        return f"×{self.scale.multiplier:.2f}".rstrip("0").rstrip(".")


@dataclass
class ShoppingList:
    buy: list[ListLine] = field(default_factory=list)
    staples: list[ListLine] = field(default_factory=list)
    unknown: list[UnknownLine] = field(default_factory=list)
    missing_recipes: list[str] = field(default_factory=list)

    def by_aisle(self) -> dict[str, list[ListLine]]:
        """Grouped for walking. Aisle order is a considered guess and nothing more."""
        order = ["produce", "meat", "seafood", "dairy", "bread", "frozen", "pantry"]
        groups: dict[str, list[ListLine]] = {}
        for line in self.buy:
            groups.setdefault(line.aisle, []).append(line)
        return {a: groups[a] for a in order if a in groups} | {
            a: v for a, v in sorted(groups.items()) if a not in order
        }


def _reconcile(item, contributions: list[Contribution]) -> ListLine:
    """Add one item's amounts together, across whatever units the recipes used.

    The old aggregation keyed on `(item, unit)` and printed `pickles` three times — bare,
    `2 tbsp`, `1 tsp` — which a shopper reads as three jars. Aggregation reconciles units
    per item instead, using that item's own `each_equiv` chain, and **emits exactly one
    line per item**. Where no conversion exists the line still stays one line and names
    the amount it could not add.

    Rounding happens once, here, after everything is summed. 1.5 peppers plus 1.5 peppers
    is 3 peppers, not 4.
    """
    graph = item.graph
    measured = [c for c in contributions if c.qty is not None]

    line = ListLine(
        item=item.canonical, aisle=item.aisle, qty=None, unit=None,
        sources=[], contributions=contributions,
    )
    for c in contributions:
        if c.meal_title not in line.sources:
            line.sources.append(c.meal_title)

    if not measured:
        line.flags.append("quantity not stated in the recipe")
        return line

    # Pick the unit that the most contributions can convert into. Where a count unit is
    # reachable, prefer it — a shopper buys heads of garlic, not teaspoons of it.
    candidates: list[str] = []
    for c in measured:
        for unit in U.reachable(c.unit, graph):
            if unit not in candidates:
                candidates.append(unit)

    def score(unit: str) -> tuple[int, int, int]:
        convertible = sum(
            1 for c in measured if U.convert(c.qty, c.unit, unit, graph) is not None)
        native = sum(1 for c in measured if c.unit == unit)
        return (convertible, 1 if U.is_count(unit) else 0, native)

    target = max(candidates, key=score) if candidates else measured[0].unit

    total = 0.0
    leftovers: dict[str, float] = {}
    for c in measured:
        converted = U.convert(c.qty, c.unit, target, graph)
        if converted is None:
            leftovers[c.unit or "ea"] = leftovers.get(c.unit or "ea", 0.0) + c.qty
        else:
            total += converted

    qty, unit = U.tidy(total, target, graph)
    line.qty, line.unit = qty, unit
    for unit_name, amount in leftovers.items():
        amount, unit_name = U.tidy(amount, unit_name, graph)
        line.extras.append(f"{U.format_qty(amount)} {unit_name}")
    if leftovers:
        line.flags.append(
            "units could not be added together — no conversion is recorded in items.md")
    if any(not c.scaled for c in contributions):
        line.flags.append("includes an unscaled meal")
    return line


def build(repo: Repo, meals: list[MealPlan], guests: int = 0) -> ShoppingList:
    """Build the whole list from a week's meals."""
    target_ae = repo.target_ae(guests)
    result = ShoppingList()
    gathered: dict[str, list[Contribution]] = {}

    for meal in meals:
        recipe: Recipe | None = repo.recipes.get(meal.slug)
        if recipe is None:
            result.missing_recipes.append(meal.slug)
            continue

        meal.scale = multiplier_for(recipe, target_ae, repo.household.portion_conversions)
        meal.yield_raw = recipe.yield_.raw or "unknown"

        for line in recipe.lines:
            if not line.resolved:
                reason = line.refusal or "no items.md row"
                result.unknown.append(UnknownLine(
                    meal_slug=meal.slug, meal_title=meal.label,
                    raw=line.raw.lstrip("- ").strip(), reason=reason))
                continue

            qty = line.parsed.qty
            if qty is not None:
                qty = qty * meal.scale.multiplier

            gathered.setdefault(line.match.item.canonical, []).append(Contribution(
                meal_slug=meal.slug, meal_title=meal.label, qty=qty,
                unit=line.parsed.unit, raw=line.raw.lstrip("- ").strip(),
                scaled=meal.scale.scaled,
            ))

    for canonical, contributions in gathered.items():
        item = repo.items.items[canonical]
        line = _reconcile(item, contributions)
        (result.staples if item.staple else result.buy).append(line)

    result.buy.sort(key=lambda l: (l.aisle, l.item))
    result.staples.sort(key=lambda l: l.item)
    return result
