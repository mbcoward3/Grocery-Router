"""Recipe files, their yields, and the multiplier each yield shape earns.

A recipe file is markdown with a small header block and an `## Ingredients` section. Only
those two are read. The `## Method` section exists so a person can cook from the file and
is never parsed.

Scaling is **per recipe, never per week**. The Italian beef serves 8 and needs no
multiplier; the salmon serves 2 and needs one. A single week-level factor gets both wrong.
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field
from pathlib import Path

from . import parse as P
from .items import ItemTable, singular

# --- yield shapes ----------------------------------------------------------

AE = "ae"                 # a batch stated in adult-equivalents
PORTIONS = "portions"     # a batch stated in countable portions — "8 enchiladas"
PER_PORTION = "per_portion"   # no batch exists — burgers, tacos, a BLT
UNKNOWN = "unknown"       # a genuine batch dish whose source never said


@dataclass
class Yield:
    shape: str
    amount: float | None = None
    noun: str = ""
    raw: str = ""


@dataclass
class Recipe:
    slug: str
    title: str
    path: Path
    fields: dict[str, str] = field(default_factory=dict)
    lines: list[P.Line] = field(default_factory=list)
    yield_: Yield = field(default_factory=lambda: Yield(UNKNOWN))

    @property
    def unscaled_by_data(self) -> bool:
        """The recipe file itself says its amounts must pass through as written."""
        return self.fields.get("scaling", "").strip().lower().startswith("unscaled")


_HEADER = re.compile(r"^([a-z_]+):\s*(.*)$")
_AE_RE = re.compile(r"^(\d+(?:\.\d+)?)\s*ae\b")
_PORTIONS_RE = re.compile(r"^(\d+(?:\.\d+)?)\s+([a-z]+)")


def parse_yield(raw: str) -> Yield:
    """Read the `yield:` header into one of the four shapes.

    `unknown` is a real value and not a gap to be filled. The old repo scored a recipe
    with no last-cooked date as *maximally* stale — unknown treated as an endpoint — and
    yield must not repeat that. Here `unknown` means one thing: a batch dish whose source
    never stated servings.
    """
    text = (raw or "").strip()
    low = text.lower()
    if not low or low.startswith("unknown"):
        return Yield(UNKNOWN, raw=text)
    if low.startswith("per portion"):
        return Yield(PER_PORTION, raw=text)
    m = _AE_RE.match(low)
    if m:
        return Yield(AE, amount=float(m.group(1)), raw=text)
    m = _PORTIONS_RE.match(low)
    if m:
        return Yield(PORTIONS, amount=float(m.group(1)),
                     noun=singular(m.group(2)), raw=text)
    return Yield(UNKNOWN, raw=text)


def load_recipe(path: Path, table: ItemTable) -> Recipe:
    """Read one `recipes/<slug>.md`."""
    text = path.read_text(encoding="utf-8")
    lines = text.splitlines()

    title = path.stem.replace("-", " ").title()
    fields: dict[str, str] = {}
    ingredient_lines: list[P.Line] = []
    in_ingredients = False
    seen_ingredients = False

    for line in lines:
        stripped = line.strip()

        if stripped.startswith("# "):
            title = stripped[2:].strip()
            continue

        if stripped.startswith("## "):
            heading = stripped[3:].strip().lower()
            in_ingredients = heading.startswith("ingredients")
            if in_ingredients:
                seen_ingredients = True
            continue

        if stripped.startswith("### "):
            # "### For the sandwich" is still the ingredient list.
            continue

        if in_ingredients:
            if stripped.startswith("- "):
                ingredient_lines.append(P.parse_line(stripped, table))
            continue

        if not seen_ingredients:
            m = _HEADER.match(stripped)
            if m:
                fields.setdefault(m.group(1), m.group(2).strip())

    recipe = Recipe(slug=path.stem, title=title, path=path,
                    fields=fields, lines=ingredient_lines)
    recipe.yield_ = parse_yield(fields.get("yield", ""))
    return recipe


def load_recipes(directory: Path, table: ItemTable) -> dict[str, Recipe]:
    return {p.stem: load_recipe(p, table) for p in sorted(Path(directory).glob("*.md"))}


# --- scaling ---------------------------------------------------------------

@dataclass
class Scale:
    multiplier: float
    scaled: bool
    note: str = ""


def multiplier_for(recipe: Recipe, target_ae: float,
                   conversions: dict[str, float]) -> Scale:
    """The multiplier for one recipe, by yield shape.

    | Yield shape                          | Multiplier                        |
    |--------------------------------------|-----------------------------------|
    | `AE(n)`                              | `target_AE / n`                   |
    | `Portions(count, noun)` + conversion | `target_AE / (count / per_adult)` |
    | `Portions(count, noun)` no conversion| ×1, and the list says so          |
    | `PerPortion`                         | ×1, and the list says so          |
    | `Unknown`                            | ×1, and the list says so          |

    **`PerPortion` is ×1 here, and that is a deliberate correction.** The rule used to
    read *multiply the per-portion amounts by `target_AE`*, but no per-portion amount is
    recorded anywhere. What the files actually hold is a batch convenience — `2 lb ground
    beef` for hamburgers, which `corpus.md` calls "a convenience, not a batch" — and
    multiplying it by a household of 2.5 ordered five pounds of beef for burger night.
    Until somebody states how much beef is one patty, the line passes through as written
    and the list says it was not scaled. The recipe files carry `scaling: unscaled` so
    the decision is visible in the data and not only in this docstring.
    """
    y = recipe.yield_

    if recipe.unscaled_by_data:
        return Scale(1.0, False,
                     "not scaled — no per-person amount is recorded, so the recipe's own "
                     "amounts pass through as written")

    if y.shape == AE and y.amount:
        return Scale(target_ae / y.amount, True, "")

    if y.shape == PORTIONS and y.amount:
        per_adult = conversions.get(y.noun)
        if per_adult:
            batch_ae = y.amount / per_adult
            if batch_ae > 0:
                return Scale(target_ae / batch_ae, True, "")
        return Scale(1.0, False,
                     f"not scaled — the source says {y.raw}, and nobody has said how many "
                     f"{y.noun}s is one adult. Answer it once in profile.md and this "
                     f"recipe scales forever.")

    if y.shape == PER_PORTION:
        return Scale(1.0, False,
                     "not scaled — a per-portion dish has no batch, and no per-person "
                     "amount is recorded")

    return Scale(1.0, False,
                 "not scaled — the source never stated how many this feeds")
