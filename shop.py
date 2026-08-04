#!/usr/bin/env python3
"""Turn a chosen week of meals into a grocery list. Step 2.

Deterministic end to end - there is no model in this path and there must never be
one. Parsing a recipe is code. See docs/step2-design.md.

    ./shop.py --week crock-pot-italian-beef,sausage-and-peppers
    ./shop.py --week chicken-noodle-soup:whole-young-chicken --ae 4
    ./shop.py --audit          # parse every recipe, report what the tables don't know

The pipeline, one function per stage:

    load -> resolve -> parse -> scale -> normalize -> convert
         -> aggregate -> consolidate -> link -> classify -> emit

Nothing is ever silently dropped. A line the parser cannot read comes out as raw
text with a flag, because someone getting home without the chuck roast is a worse
failure than an ugly list.
"""

from __future__ import annotations

import argparse
import json
import math
import re
import sys
from collections import defaultdict
from dataclasses import dataclass, field
from pathlib import Path

ROOT = Path(__file__).resolve().parent
RECIPES = ROOT / "recipes"
ITEMS = ROOT / "items.md"

# Household default, from profile.md: 2 adults, a 3-year-old, a 1-year-old.
DEFAULT_AE = 2.5


# --------------------------------------------------------------------------- #
# Units
# --------------------------------------------------------------------------- #

VOLUME = {"tsp": 1.0, "tbsp": 3.0, "floz": 6.0, "cup": 48.0, "pint": 96.0, "quart": 192.0}
WEIGHT = {"oz": 1.0, "lb": 16.0}

# Units that name a thing you pick up rather than a measure. These round up to
# whole numbers, because half a can is not a thing you can buy.
COUNTABLE = {
    "ea", "can", "package", "envelope", "tube", "jar", "bunch", "head", "clove",
    "slice", "sprig", "rib", "packet", "stick", "loaf", "bottle", "box", "bag",
    "container", "pkg", "roll", "fillet", "breast", "tortilla", "bun",
}

UNIT_ALIASES = {
    "teaspoon": "tsp", "teaspoons": "tsp", "tsp.": "tsp", "t": "tsp",
    "tablespoon": "tbsp", "tablespoons": "tbsp", "tbsp.": "tbsp", "tbs": "tbsp", "tb": "tbsp",
    "cups": "cup", "c": "cup", "c.": "cup",
    "fluid ounce": "floz", "fl oz": "floz",
    "ounce": "oz", "ounces": "oz", "oz.": "oz",
    "pound": "lb", "pounds": "lb", "lbs": "lb", "lb.": "lb", "#": "lb",
    "pints": "pint", "quarts": "quart",
    "cans": "can", "packages": "package", "envelopes": "envelope", "tubes": "tube",
    "jars": "jar", "bunches": "bunch", "heads": "head", "cloves": "clove",
    "slices": "slice", "sprigs": "sprig", "ribs": "rib", "packets": "packet",
    "sticks": "stick", "loaves": "loaf", "bottles": "bottle", "boxes": "box",
    "bags": "bag", "containers": "container", "pkgs": "pkg", "rolls": "roll",
    "fillets": "fillet", "breasts": "breast", "tortillas": "tortilla", "buns": "bun",
    "each": "ea", "pkt": "packet", "pkts": "packet", "pkg": "package",
}

# Words that describe what you do to an ingredient, not what it is. A trailing
# clause made only of these is prep, and prep is not part of the item name.
PREP = {
    "peeled", "chopped", "minced", "sliced", "diced", "cubed", "quartered", "halved",
    "shredded", "grated", "melted", "softened", "drained", "rinsed", "trimmed",
    "thinly", "finely", "roughly", "freshly", "coarsely", "lightly", "well",
    "cut", "into", "and", "or", "in", "to", "for", "with", "then", "plus", "extra",
    "serving", "taste", "half", "large", "small", "pieces", "chunks", "optional",
    "undrained", "seeded", "stemmed", "torn", "beaten", "room", "temperature",
    "mild", "hot", "spicy", "sweet", "on", "top", "more", "additional", "whole",
}

WEIGHT_UNITS = set(WEIGHT) | {"ounce", "ounces", "pound", "pounds", "lbs", "gram", "grams", "g", "kg"}

ALL_UNITS = set(VOLUME) | set(WEIGHT) | COUNTABLE | set(UNIT_ALIASES)

FRACTIONS = {
    "½": 0.5, "⅓": 1 / 3, "⅔": 2 / 3, "¼": 0.25, "¾": 0.75,
    "⅛": 0.125, "⅜": 0.375, "⅝": 0.625, "⅞": 0.875, "⅕": 0.2, "⅖": 0.4,
}

# Adjectives that describe a size but are not a unit. "2 medium yellow onions"
# has to normalize to the same canonical item as "1 onion".
SIZE_WORDS = {"small", "medium", "med", "large", "lg", "sm", "big", "extra-large", "jumbo"}


def norm_unit(tok: str) -> str | None:
    t = tok.lower().strip().rstrip(",")
    t = UNIT_ALIASES.get(t, t)
    return t if t in ALL_UNITS or t in VOLUME or t in WEIGHT or t in COUNTABLE else None


def parse_number(tok: str) -> float | None:
    """One quantity token. Handles 2, 2.5, 1/2, 1 1/2, 1½, ½, and 4-5 (takes the
    high end - buying short is worse than buying over)."""
    tok = tok.strip()
    if not tok:
        return None
    for ch, val in FRACTIONS.items():
        if tok.endswith(ch):
            head = tok[: -len(ch)]
            base = parse_number(head) if head else 0.0
            return (base or 0.0) + val
        if tok == ch:
            return val
    m = re.fullmatch(r"(\d+(?:\.\d+)?)\s*[-–—]\s*(\d+(?:\.\d+)?)", tok)
    if m:
        return float(m.group(2))
    m = re.fullmatch(r"(\d+)\s*/\s*(\d+)", tok)
    if m:
        return float(m.group(1)) / float(m.group(2))
    m = re.fullmatch(r"(\d+)\s+(\d+)\s*/\s*(\d+)", tok)
    if m:
        return float(m.group(1)) + float(m.group(2)) / float(m.group(3))
    try:
        return float(tok)
    except ValueError:
        return None


# --------------------------------------------------------------------------- #
# Data
# --------------------------------------------------------------------------- #

@dataclass
class Ingredient:
    raw: str
    qty: float | None = None
    unit: str | None = None
    item: str = ""
    note: str = ""
    pack: tuple[float, str] | None = None
    accepts: list[str] = field(default_factory=list)
    may_come_from: str = ""
    section: str = ""
    parsed: bool = True


@dataclass
class Variant:
    name: str
    meta: dict = field(default_factory=dict)
    adds: list[Ingredient] = field(default_factory=list)
    replaces: list[str] = field(default_factory=list)
    produces: list[str] = field(default_factory=list)


@dataclass
class Recipe:
    slug: str
    title: str
    meta: dict = field(default_factory=dict)
    ingredients: list[Ingredient] = field(default_factory=list)
    variants: list[Variant] = field(default_factory=list)
    produces: list[str] = field(default_factory=list)


@dataclass
class Item:
    canonical: str
    aisle: str = "other"
    staple: bool = False
    each_equiv: str = ""
    synonyms: list[str] = field(default_factory=list)
    guessed: bool = False


# --------------------------------------------------------------------------- #
# Stage: load
# --------------------------------------------------------------------------- #

COMMENT = re.compile(r"<!--.*?-->", re.S)
META_LINE = re.compile(r"^([a-z_]+):\s*(.*)$")


def _strip_comments(text: str) -> str:
    return COMMENT.sub("", text)


def load_recipe(slug: str) -> Recipe:
    path = RECIPES / f"{slug}.md"
    if not path.exists():
        raise FileNotFoundError(f"no recipe file for {slug!r} (looked in {path})")
    lines = _strip_comments(path.read_text(encoding="utf-8")).splitlines()

    recipe = Recipe(slug=slug, title=slug.replace("-", " ").title())
    section = None          # None | "meta" | "ingredients" | "variants" | other
    sub = ""
    variant: Variant | None = None
    last_add: Ingredient | None = None

    for raw in lines:
        line = raw.rstrip()
        stripped = line.strip()

        if stripped.startswith("# "):
            recipe.title = stripped[2:].strip()
            section = "meta"
            continue
        if stripped.startswith("## "):
            head = stripped[3:].strip().lower()
            section = {"ingredients": "ingredients", "variants": "variants"}.get(head, "skip")
            sub = ""
            variant = None
            continue
        if stripped.startswith("### "):
            name = stripped[4:].strip()
            if section == "variants":
                variant = Variant(name=name)
                recipe.variants.append(variant)
                last_add = None
            elif section == "ingredients":
                sub = name
            continue

        if section == "meta":
            m = META_LINE.match(stripped)
            if m:
                key, val = m.group(1), m.group(2).strip()
                if key == "produces":
                    recipe.produces.append(val)
                else:
                    recipe.meta[key] = val
            continue

        if section == "ingredients" and stripped.startswith("- "):
            ing = parse_ingredient(stripped[2:])
            ing.section = sub
            recipe.ingredients.append(ing)
            continue

        if section == "variants" and variant is not None:
            if stripped.startswith("+ "):
                last_add = parse_ingredient(stripped[2:])
                variant.adds.append(last_add)
                continue
            m = META_LINE.match(stripped)
            if m:
                key, val = m.group(1), m.group(2).strip()
                indented = line[:1].isspace()
                if key == "replaces":
                    variant.replaces.append(val)
                elif key == "produces":
                    variant.produces.append(val)
                elif not indented:
                    variant.meta[key] = val
                continue

    return recipe


# --------------------------------------------------------------------------- #
# Stage: parse
# --------------------------------------------------------------------------- #

TRAILING_ACCEPTS = re.compile(r"\s{2,}accepts:\s*(.+?)\s*$")
TRAILING_FROM = re.compile(r"\s{2,}may come from:\s*(.+?)\s*$")
PAREN_PACK = re.compile(r"\((\d+(?:\.\d+)?)[\s-]*(ounce|ounces|oz|pound|pounds|lb|lbs)\.?\)", re.I)
GLUED_UNIT = re.compile(r"(?<=\d)(lbs?|oz|g|kg|ml|l)\b", re.I)
TRAILING_PACK = re.compile(r"\s+(\d+(?:\.\d+)?)\s*(oz|ounce|ounces|lb|lbs|pound|pounds)\.?\s*(?:can|cans|jar|jars|package|packages)?\s*$", re.I)
JUICE_OF = re.compile(r"^juice of\s+(.+)$", re.I)


def _split_note(text: str) -> tuple[str, str]:
    """Split on the LAST top-level comma. `2 lb boneless, skinless chicken thighs,
    cubed` must not become an ingredient called `boneless`."""
    depth = 0
    idx = -1
    for i, ch in enumerate(text):
        if ch == "(":
            depth += 1
        elif ch == ")":
            depth = max(0, depth - 1)
        elif ch == "," and depth == 0:
            idx = i
    if idx == -1:
        return text.strip(), ""
    return text[:idx].strip(), text[idx + 1:].strip()


def parse_ingredient(text: str) -> Ingredient:
    raw = text.strip()
    body = raw

    accepts: list[str] = []
    m = TRAILING_ACCEPTS.search(body)
    if m:
        accepts = [a.strip() for a in m.group(1).split(",") if a.strip()]
        body = body[: m.start()]

    may_from = ""
    m = TRAILING_FROM.search(body)
    if m:
        may_from = m.group(1).strip()
        body = body[: m.start()]

    body = GLUED_UNIT.sub(r" \1", body.strip())   # "2lb ground beef"
    ing = Ingredient(raw=raw, accepts=accepts, may_come_from=may_from)

    m = JUICE_OF.match(body)
    if m:
        rest = m.group(1)
        tokens = rest.split()
        qty = parse_number(tokens[0]) if tokens else None
        ing.qty = qty if qty is not None else 1.0
        ing.item = " ".join(tokens[1:]) if qty is not None else rest
        ing.note = "juice of"
        return ing

    pack = None
    m = PAREN_PACK.search(body)
    if m:
        pack = (float(m.group(1)), "oz" if m.group(2).lower().startswith(("o",)) else "lb")
        body = (body[: m.start()] + " " + body[m.end():]).strip()

    tokens = body.split()
    if not tokens:
        ing.parsed = False
        return ing

    # leading quantity, possibly "1 1/2"
    qty = parse_number(tokens[0])
    if qty is not None and len(tokens) > 1:
        # "1 1/2 tsp" and "1 ½ tsp" are one quantity written as two tokens.
        second_is_fraction = re.fullmatch(r"\d+\s*/\s*\d+", tokens[1]) or tokens[1] in FRACTIONS
        if second_is_fraction:
            joined = parse_number(f"{tokens[0]} {tokens[1]}") if "/" in tokens[1] else qty + FRACTIONS[tokens[1]]
            if joined is not None:
                qty = joined
                tokens = tokens[1:]
    if qty is not None:
        tokens = tokens[1:]
    ing.qty = qty

    # "12 ounce can refrigerated biscuits" / "1 10.75 oz can ..." -> the weight is
    # a pack size and the container is the unit.
    if tokens:
        u0 = norm_unit(tokens[0])
        if u0 in WEIGHT and len(tokens) > 1 and norm_unit(tokens[1]) in COUNTABLE and qty is not None:
            pack = pack or (qty, u0)
            ing.qty = 1.0
            ing.unit = norm_unit(tokens[1])
            tokens = tokens[2:]
        elif u0 is not None and len(tokens) > 1:
            # Never eat the only word on the line. `- buns` is an item, not a unit.
            ing.unit = u0
            tokens = tokens[1:]

    if ing.unit is None and tokens:
        n2 = parse_number(tokens[0])
        if n2 is not None and len(tokens) > 1 and norm_unit(tokens[1]) in WEIGHT:
            u = norm_unit(tokens[1])
            if len(tokens) > 2 and norm_unit(tokens[2]) in COUNTABLE:
                pack = pack or (n2, u)
                ing.unit = norm_unit(tokens[2])
                tokens = tokens[3:]

    # "Tube of biscuits", "1 rib of celery", "2 packets of ranch seasoning" - the
    # unit has been taken, and the joining word is not part of the item.
    while tokens and tokens[0].lower() in {"of", "or"}:
        tokens = tokens[1:]

    rest = " ".join(tokens)
    m = TRAILING_PACK.search(rest)
    if m and ing.unit in COUNTABLE:
        pack = pack or (float(m.group(1)), "oz" if m.group(2).lower().startswith("o") else "lb")
        rest = rest[: m.start()].strip()

    item, note = _split_note(rest)

    # The last-comma rule protects "boneless, skinless chicken thighs" but leaves
    # "Russet potatoes, peeled". Peel off trailing clauses that are pure prep.
    while "," in item:
        head, _, tail = item.rpartition(",")
        tail_words = [w.strip("().").lower() for w in tail.split() if w.strip("().")]
        if tail_words and all(w in PREP for w in tail_words):
            note = (tail.strip() + ("; " + note if note else "")).strip()
            item = head.strip()
        else:
            break

    words = item.split()
    while words and words[0].lower().strip(",") in SIZE_WORDS:
        words = words[1:]
    item = " ".join(words)

    ing.item = item.strip()
    ing.note = note
    ing.pack = pack
    if not ing.item:
        ing.parsed = False
    return ing


# --------------------------------------------------------------------------- #
# Stage: resolve (apply a variant)
# --------------------------------------------------------------------------- #

def slugify(text: str) -> str:
    return re.sub(r"[^a-z0-9]+", "-", text.lower()).strip("-")


def resolve(recipe: Recipe, variant_name: str | None) -> tuple[list[Ingredient], Variant | None]:
    """Apply the chosen variant's `+` and `replaces:` lines. A recipe with no
    variants passes its base list straight through - that is the point."""
    if not recipe.variants:
        return list(recipe.ingredients), None

    chosen = recipe.variants[0]
    if variant_name:
        want = slugify(variant_name)
        for v in recipe.variants:
            if slugify(v.name) == want or slugify(v.name).startswith(want):
                chosen = v
                break
        else:
            raise SystemExit(
                f"{recipe.slug}: no variant matching {variant_name!r}. "
                f"Have: {', '.join(slugify(v.name) for v in recipe.variants)}"
            )

    out = []
    dropped = {r.strip().lower() for r in chosen.replaces}
    for ing in recipe.ingredients:
        if ing.raw.strip().lower() in dropped or ing.item.lower() in dropped:
            continue
        if any(d in ing.raw.strip().lower() for d in dropped):
            continue
        out.append(ing)
    out.extend(chosen.adds)
    return out, chosen


# --------------------------------------------------------------------------- #
# Stage: scale
# --------------------------------------------------------------------------- #

YIELD_AE = re.compile(r"^(\d+(?:\.\d+)?)\s*AE\b")
YIELD_THINGS = re.compile(r"^(\d+(?:\.\d+)?)\s+([A-Za-z][A-Za-z ]*?)(?:\s*\(|$)")
PORTION_RATE = re.compile(r"(\d+(?:\.\d+)?)\s+per adult", re.I)


def scale(recipe: Recipe, ae: float) -> tuple[float, str]:
    """Return (multiplier, why). Whole batches only - you do not buy 0.3 of a roast.

    The three yield shapes of docs/step2-design.md §2.5, plus `unknown`, which
    scales x1 and says so rather than guessing."""
    y = recipe.meta.get("yield", "").strip()

    if y.startswith("per portion"):
        return 1.0, "scales with headcount; quantities as recorded, per-portion amount not stated"

    m = YIELD_AE.match(y)
    if m:
        served = float(m.group(1))
        mult = max(1, math.ceil(ae / served - 1e-9))
        return float(mult), f"serves {served:g} AE, week needs {ae:g}"

    m = YIELD_THINGS.match(y)
    if m and not y.startswith("unknown"):
        count, thing = float(m.group(1)), m.group(2).strip()
        rate = PORTION_RATE.search(recipe.meta.get("portion", ""))
        if rate:
            need = ae * float(rate.group(1))
            mult = max(1, math.ceil(need / count - 1e-9))
            return float(mult), f"makes {count:g} {thing}, need {need:g}"
        return 1.0, f"makes {count:g} {thing} — how many {thing} is one adult? not stated, so not scaled"

    return 1.0, "yield unknown, not scaled"


# --------------------------------------------------------------------------- #
# Stage: normalize (items.md)
# --------------------------------------------------------------------------- #

def load_items() -> tuple[dict[str, Item], dict[str, str]]:
    items: dict[str, Item] = {}
    index: dict[str, str] = {}
    if not ITEMS.exists():
        return items, index
    for line in ITEMS.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line.startswith("|") or line.startswith("|---") or "canonical" in line[:20]:
            continue
        cells = [c.strip() for c in line.strip("|").split("|")]
        if len(cells) < 5:
            continue
        canonical, aisle, staple, each_equiv, syns = cells[:5]
        if not canonical or canonical.startswith("-"):
            continue
        guessed = aisle.endswith("?")
        it = Item(
            canonical=canonical,
            aisle=aisle.rstrip("?"),
            staple=staple.lower().startswith("y"),
            each_equiv=each_equiv,
            synonyms=[s.strip() for s in syns.split(",") if s.strip()],
            guessed=guessed,
        )
        items[canonical] = it
        for key in [canonical, canonical.replace("_", " ")] + it.synonyms:
            index[key.lower()] = canonical
    return items, index


def split_compound(ing: Ingredient, index: dict[str, str]) -> list[Ingredient] | None:
    """`2 tsp thyme and rosemary, freshly chopped` is two items sharing one
    quantity. Matching it to `thyme` and moving on drops the rosemary silently,
    which is the one thing this pipeline may never do.

    Both halves are emitted, and neither carries a quantity — splitting `2 tsp`
    between them would be a guess, and `some rosemary` is what you can actually
    act on at the shelf."""
    if " and " not in ing.item.lower():
        return None
    parts = [p.strip() for p in re.split(r"\s+and\s+", ing.item, flags=re.I) if p.strip()]
    if len(parts) < 2:
        return None
    out = []
    seen = set()
    for part in parts:
        probe = Ingredient(raw=ing.raw, item=part, note=ing.note)
        canonical = normalize(probe, index)
        if canonical is None or canonical in seen:
            return None
        seen.add(canonical)
        out.append(Ingredient(raw=ing.raw, qty=None, unit=None, item=part,
                              note=(ing.note + " [quantity shared with the rest of the line]").strip(),
                              accepts=ing.accepts, section=ing.section))
    return out


# Note what is NOT here: `fresh` and `dried`. They look like noise and are not -
# fresh thyme is in the produce aisle and dried thyme is in the spice rack, and
# letting the probe discard the word merges them. Every real pair gets a row.
STOPWORDS = {"chopped", "minced", "sliced", "diced", "shredded", "ground",
             "peeled", "frozen", "canned", "boneless", "skinless", "cooked", "raw",
             "lean", "grated", "melted", "softened", "of", "the", "a"}


def normalize(ing: Ingredient, index: dict[str, str]) -> str | None:
    """Item name -> canonical, via items.md synonyms. Longest match wins; a
    miss returns None and is reported rather than dropped."""
    name = ing.item.lower().strip().rstrip(".")
    while "(" in name and ")" in name:                 # ((shredded or diced))
        stripped_once = re.sub(r"\([^()]*\)", " ", name)
        if stripped_once == name:
            break
        name = stripped_once
    # Sources write unbalanced parens - `1 med onion ((1 cup), finely chopped)`.
    # Whatever follows a stray bracket is commentary; the name ends there.
    name = re.split(r"[()]", name)[0]
    # `shredded provolone or mozzarella cheese` names one item and one tolerance.
    # The head is what goes in the cart; the tail is what `accepts:` is for, and
    # inferring a swap from the word "or" is exactly what §2.3 forbids.
    name = re.split(r"\bor\b", name)[0]
    name = re.sub(r"\s+", " ", name).strip()
    if not name:
        return None
    if name in index:
        return index[name]
    singular = re.sub(r"(?<=[a-z])s$", "", name)
    if singular in index:
        return index[singular]
    words = [w for w in re.split(r"[\s,]+", name) if w and w not in STOPWORDS]
    for n in range(len(words), 0, -1):
        for start in range(0, len(words) - n + 1):
            probe = " ".join(words[start:start + n])
            leftover = words[:start] + words[start + n:]
            # A partial match is only safe when everything it leaves behind is
            # noise. Without this, `onion powder` matches `onion` and a teaspoon
            # of spice becomes a fresh onion in the cart - a silent mis-merge,
            # which is worse than an unknown line, because an unknown line is
            # printed and a mis-merge is not.
            if any(w not in STOPWORDS and w not in PREP and norm_unit(w) is None
                   for w in leftover):
                continue
            if probe in index:
                return index[probe]
            probe_s = re.sub(r"(?<=[a-z])s$", "", probe)
            if probe_s in index:
                return index[probe_s]
    return None


# --------------------------------------------------------------------------- #
# Stage: convert
# --------------------------------------------------------------------------- #

EACH_EQUIV = re.compile(
    r"^\s*(\d+(?:\.\d+)?)?\s*([A-Za-z]+)\s*=\s*(\d+(?:\.\d+)?)\s*([A-Za-z]+)", re.I
)


def unit_graph(item: Item | None) -> dict[str, dict[str, float]]:
    """Per-item conversion graph. Standard families plus whatever `each_equiv`
    bridges. This is the whole of unit reconciliation, and it is per-item on
    purpose: 1 onion is a cup only because someone wrote that down for onions."""
    g: dict[str, dict[str, float]] = defaultdict(dict)

    def edge(a, b, factor):        # 1 a = factor b
        g[a][b] = factor
        g[b][a] = 1.0 / factor

    for fam in (VOLUME, WEIGHT):
        units = list(fam)
        for i in range(len(units)):
            for j in range(i + 1, len(units)):
                edge(units[i], units[j], fam[units[i]] / fam[units[j]])

    for clause in (item.each_equiv.split(";") if item and item.each_equiv else []):
        m = EACH_EQUIV.match(clause)
        if not m:
            continue
        lhs_n = float(m.group(1) or 1)
        lhs_u = norm_unit(m.group(2)) or m.group(2).lower()
        rhs_n, rhs_u = float(m.group(3)), norm_unit(m.group(4)) or m.group(4).lower()
        if lhs_n:
            edge(lhs_u, rhs_u, rhs_n / lhs_n)
    return g


def convert(qty: float, frm: str, to: str, graph) -> float | None:
    if frm == to:
        return qty
    seen = {frm: 1.0}
    queue = [frm]
    while queue:
        node = queue.pop(0)
        for nxt, factor in graph.get(node, {}).items():
            if nxt not in seen:
                seen[nxt] = seen[node] * factor
                queue.append(nxt)
    return qty * seen[to] if to in seen else None


# --------------------------------------------------------------------------- #
# Stage: aggregate
# --------------------------------------------------------------------------- #

@dataclass
class Line:
    canonical: str
    qty: float | None
    unit: str | None
    sources: list[str] = field(default_factory=list)
    raws: list[str] = field(default_factory=list)
    accepts: set[str] = field(default_factory=set)
    packs: list[tuple[float, str]] = field(default_factory=list)
    split: list[tuple[float, str]] = field(default_factory=list)
    from_produce: str = ""
    merged_into: str = ""


def preferred_unit(item: Item | None, units: list[str | None]) -> str | None:
    """The unit the list should say. If each_equiv names a countable left-hand
    side, use it - "5 bell peppers" beats "5 cups bell pepper" at the store."""
    if item and item.each_equiv:
        m = EACH_EQUIV.match(item.each_equiv.split(";")[0])
        if m:
            lhs = norm_unit(m.group(2)) or m.group(2).lower()
            # Anything that isn't a measure is a thing you pick up: head, lime, stick.
            if lhs not in VOLUME and lhs not in WEIGHT:
                return lhs
    counts = defaultdict(int)
    for u in units:
        counts[u] += 1
    return max(counts, key=lambda u: (counts[u], u is not None)) if counts else None


def aggregate(entries, items) -> tuple[list[Line], list[tuple[str, str]]]:
    """Sum across recipes, keeping provenance. Round after aggregating, never
    before: 1.5 peppers + 1.5 peppers is 3 peppers, not 4."""
    by_item: dict[str, list] = defaultdict(list)
    unknown: list[tuple[str, str]] = []
    for canonical, ing, mult, slug in entries:
        if canonical is None:
            unknown.append((slug, ing.raw))
            continue
        by_item[canonical].append((ing, mult, slug))

    lines = []
    for canonical, group in by_item.items():
        item = items.get(canonical)
        graph = unit_graph(item)
        units = [ing.unit if ing.unit else "ea" for ing, _, _ in group]
        target = preferred_unit(item, units)

        total = 0.0
        leftovers: dict[str, float] = defaultdict(float)
        any_qty = False
        line = Line(canonical=canonical, qty=None, unit=target)
        for ing, mult, slug in group:
            line.sources.append(slug)
            line.raws.append(f"{slug}: {ing.raw}")
            line.accepts.update(a.lower() for a in ing.accepts)
            if ing.pack:
                line.packs.append(ing.pack)
            if ing.may_come_from:
                line.from_produce = ing.may_come_from
            if ing.qty is None:
                continue
            any_qty = True
            u = ing.unit or "ea"
            got = convert(ing.qty * mult, u, target, graph) if target else None
            if got is None:
                leftovers[u] += ing.qty * mult
            else:
                total += got

        if any_qty:
            line.qty = round(total, 3) if total else None
            line.split = [(round(v, 3), u) for u, v in leftovers.items()]
        lines.append(line)

    lines.sort(key=lambda l: l.canonical)
    return lines, unknown


def round_out(qty: float | None, unit: str | None) -> str:
    if qty is None:
        return "some"
    # Anything that isn't a measure is a thing you pick up whole: round up. Half a
    # can, half a lime and half a head of garlic are not purchasable.
    if unit is None or (unit not in VOLUME and unit not in WEIGHT):
        return f"{math.ceil(qty - 1e-9):g}"
    if abs(qty - round(qty)) < 0.02:
        return f"{round(qty):g}"
    return f"{qty:.2f}".rstrip("0").rstrip(".")


def plural(word: str, n: float) -> str:
    # Canonical names and units are singular by convention, so an existing -s is
    # left alone rather than turned into "tortillases".
    if n <= 1 or not word or word.endswith("s"):
        return word
    if word.endswith(("x", "ch", "sh")):
        return word + "es"
    if word.endswith("o") and not word.endswith(("oo", "eo")):
        return word + "es"
    if word.endswith("y") and word[-2:-1] not in "aeiou":
        return word[:-1] + "ies"
    return word + "s"


def display(line: "Line") -> tuple[str, str]:
    """(quantity phrase, item name). Kept together because pluralising one
    depends on the other."""
    qty_text = round_out(line.qty, line.unit)
    # "some" is what an unmeasured line prints, and unmeasured lines are usually
    # mass nouns - "some rosemary", not "some rosemaries".
    n = float(qty_text) if re.fullmatch(r"[\d.]+", qty_text) else 0.0
    name = line.canonical.replace("_", " ")
    if line.qty is None and not line.split:
        # No source line gave a quantity. Naming a unit would imply one.
        return "some", name
    # "2 limes lime" - when the unit and the item are the same word, say it once.
    if line.unit and line.unit == name.split()[-1]:
        return qty_text, plural(name, n)
    if line.unit in (None, "ea"):
        head = qty_text
        name = plural(name, n)
    else:
        head = f"{qty_text} {plural(line.unit, n) if line.unit not in VOLUME and line.unit not in WEIGHT else line.unit}"
    return head, name


# --------------------------------------------------------------------------- #
# Stages: consolidate, link
# --------------------------------------------------------------------------- #

def consolidate(lines: list[Line], index: dict[str, str]) -> list[tuple[str, str]]:
    """Merge an item into one already in the week when the recipe declared it
    would do. Tolerance is declared, never inferred - a model reasoning that
    cheese is cheese will eventually swap the ingredient that was the point."""
    present = {l.canonical: l for l in lines}
    merges = []
    for line in lines:
        if not line.accepts:
            continue
        for alt in sorted(line.accepts):
            target = index.get(alt.lower())
            if target and target != line.canonical and target in present and not present[target].merged_into:
                line.merged_into = target
                merges.append((line.canonical, target))
                break
    return merges


def link(lines: list[Line], week: list[tuple[Recipe, Variant | None]]) -> list[str]:
    """Mark lines that another meal in the week produces. The fallback is always
    bought - the link saves you using the item, never buying it."""
    producers = []
    for recipe, variant in week:
        for text in recipe.produces + (variant.produces if variant else []):
            producers.append((recipe.slug, text))
    def key(s):
        return re.sub(r"[^a-z0-9]+", " ", s.lower()).strip()

    notes = []
    for line in lines:
        if not line.from_produce:
            continue
        for slug, text in producers:
            if key(slug) in key(line.from_produce) or key(line.canonical) in key(text):
                notes.append(
                    f"{line.canonical.replace('_', ' ')} — {slug} produces {text}. "
                    f"Buy it anyway; if you cook that one first this is spare."
                )
                break
    return notes


# --------------------------------------------------------------------------- #
# Stage: emit
# --------------------------------------------------------------------------- #

AISLE_ORDER = ["produce", "meat", "seafood", "dairy", "bread", "frozen", "pantry", "other"]


def emit(week, lines, unknown, merges, links, ae, scales, items) -> str:
    out = []
    names = ", ".join(r.title + (f" ({v.name})" if v else "") for r, v in week)
    out.append(f"# Grocery list — {len(week)} meals, {ae:g} AE")
    out.append("")
    out.append(names)
    out.append("")

    shown = [l for l in lines if not l.merged_into]
    by_aisle = defaultdict(list)
    staples = []
    for line in shown:
        item = items.get(line.canonical)
        if item and item.staple:
            staples.append(line)
        else:
            by_aisle[item.aisle if item else "other"].append(line)

    for aisle in AISLE_ORDER:
        group = by_aisle.get(aisle)
        if not group:
            continue
        out.append(f"## {aisle.title()}")
        out.append("")
        for line in group:
            head, name = display(line)
            splits = list(line.split)
            if line.qty is None and splits:
                # One recipe measured it and another didn't. Lead with the number
                # that exists rather than "some + 8 sprigs" — the unmeasured line
                # is covered by buying the bunch either way.
                q, u = max(splits, key=lambda s: s[0])
                splits.remove((q, u))
                head = f"{round_out(q, u)}{'' if u == 'ea' else ' ' + plural(u, q)}"
            extra = "".join(f" + {round_out(q, u)}{'' if u == 'ea' else ' ' + u}"
                            for q, u in splits)
            pack = ""
            if line.packs and line.unit in COUNTABLE:
                sizes = sorted({f"{p[0]:g} {p[1]}" for p in line.packs})
                pack = f" ({', '.join(sizes)})"
            src = ", ".join(sorted(set(s.replace("-", " ") for s in line.sources)))
            out.append(f"- **{head}{extra} {name}**{pack}  — {src}")
        out.append("")

    if staples:
        out.append("## Probably have — check before you go")
        out.append("")
        for line in sorted(staples, key=lambda l: l.canonical):
            out.append(f"- {line.canonical.replace('_', ' ')}")
        out.append("")

    if merges:
        out.append("## Merged")
        out.append("")
        for a, b in merges:
            out.append(f"- {a.replace('_',' ')} → buying {b.replace('_',' ')} instead "
                       f"(the recipe says it accepts it)  [keep separate: edit items.md]")
        out.append("")

    if links:
        out.append("## Comes from another meal")
        out.append("")
        for note in links:
            out.append(f"- {note}")
        out.append("")

    flagged = [s for s in scales if "unknown" in s[1] or "not stated" in s[1]]
    if flagged:
        out.append("## Not scaled — the recipe doesn't know how much it makes")
        out.append("")
        for slug, why in flagged:
            out.append(f"- **{slug.replace('-', ' ')}** — {why}")
        out.append("")

    if unknown:
        out.append("## Not recognised — buy these by reading the line")
        out.append("")
        out.append("*`items.md` has no row for these, so they could not be merged, converted "
                   "or shelved. Nothing is dropped; adding a row is the fix.*")
        out.append("")
        for slug, raw in unknown:
            out.append(f"- {raw}  — {slug.replace('-', ' ')}")
        out.append("")

    guessed = sorted({l.canonical for l in shown
                      if items.get(l.canonical) and items[l.canonical].guessed})
    if guessed:
        out.append(f"*{len(guessed)} item(s) on this list have a machine-guessed aisle "
                   f"(marked `?` in items.md): {', '.join(g.replace('_',' ') for g in guessed)}. "
                   f"A wrong aisle costs you a few steps, never the wrong item.*")
        out.append("")

    out.append("---")
    out.append("")
    out.append("**Sides are not included.** The corpus is mains-only — sides get cooked here "
               "and never got written down, so this list is systematically short on "
               "vegetables. That is a known gap, not an oversight.")
    return "\n".join(out)


def coupling_report(lines: list[Line], items: dict[str, Item]) -> str:
    """Staples are excluded. Salt is in every meal and shares nothing meaningful;
    counting it makes the week look coupled when it isn't."""
    out = ["", "## What is shared, and what is stranded", ""]
    real = [l for l in lines
            if not l.merged_into and not (items.get(l.canonical) and items[l.canonical].staple)]
    multi = [l for l in real if len(set(l.sources)) > 1]
    single = [l for l in real if len(set(l.sources)) == 1]
    if multi:
        for line in sorted(multi, key=lambda l: -len(set(l.sources))):
            src = ", ".join(sorted(set(s.replace('-', ' ') for s in line.sources)))
            out.append(f"- **{line.canonical.replace('_', ' ')}** → {src}")
    else:
        out.append("- Nothing in this week shares an ingredient with anything else.")
    out.append("")
    out.append(f"*{len(single)} item(s) come from exactly one meal. Skip that meal and they "
               f"are stranded — this is the honest answer to \"what breaks if a night falls "
               f"through\", computed rather than guessed.*")
    return "\n".join(out)


# --------------------------------------------------------------------------- #
# Driver
# --------------------------------------------------------------------------- #

def build(week_spec: list[str], ae: float):
    items, index = load_items()
    week = []
    entries = []
    scales = []
    for spec in week_spec:
        slug, _, variant_name = spec.partition(":")
        recipe = load_recipe(slug.strip())
        ingredients, variant = resolve(recipe, variant_name.strip() or None)
        mult, why = scale(recipe, ae)
        scales.append((slug, why))
        week.append((recipe, variant))
        for ing in ingredients:
            if not ing.parsed:
                entries.append((None, ing, mult, slug))
                continue
            parts = split_compound(ing, index)
            for piece in (parts or [ing]):
                entries.append((normalize(piece, index), piece, mult, slug))

    lines, unknown = aggregate(entries, items)
    merges = consolidate(lines, index)
    links = link(lines, week)
    return week, lines, unknown, merges, links, scales, items


def audit():
    items, index = load_items()
    misses = defaultdict(list)
    failures = []
    total = 0
    for path in sorted(RECIPES.glob("*.md")):
        recipe = load_recipe(path.stem)
        ingredients, _ = resolve(recipe, None)
        for v in recipe.variants:
            ingredients = ingredients + v.adds
        for ing in ingredients:
            total += 1
            if not ing.parsed:
                failures.append((path.stem, ing.raw))
                continue
            if split_compound(ing, index):
                continue
            if normalize(ing, index) is None:
                misses[ing.item.lower()].append(path.stem)
    print(f"{len(list(RECIPES.glob('*.md')))} recipes, {total} ingredient lines")
    print(f"{len(failures)} unparseable, {sum(len(v) for v in misses.values())} lines "
          f"with no items.md row ({len(misses)} distinct names)")
    print()
    for name, slugs in sorted(misses.items(), key=lambda kv: (-len(kv[1]), kv[0])):
        print(f"{len(slugs):3d}  {name}   [{', '.join(sorted(set(slugs)))}]")
    if failures:
        print("\nUNPARSEABLE:")
        for slug, raw in failures:
            print(f"  {slug}: {raw}")


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--week", help="comma-separated recipe slugs, optionally slug:variant")
    ap.add_argument("--ae", type=float, default=DEFAULT_AE, help=f"adult-equivalents (default {DEFAULT_AE})")
    ap.add_argument("--guests", type=float, default=0.0, help="extra AE for guests")
    ap.add_argument("--json", action="store_true")
    ap.add_argument("--no-coupling", action="store_true")
    ap.add_argument("--audit", action="store_true", help="parse every recipe, report gaps")
    args = ap.parse_args(argv)

    if args.audit:
        audit()
        return 0
    if not args.week:
        ap.error("--week is required (or --audit)")

    ae = args.ae + args.guests
    specs = [s.strip() for s in args.week.split(",") if s.strip()]
    week, lines, unknown, merges, links, scales, items = build(specs, ae)

    if args.json:
        print(json.dumps({
            "ae": ae,
            "meals": [{"slug": r.slug, "title": r.title, "variant": v.name if v else None}
                      for r, v in week],
            "lines": [{"item": l.canonical, "qty": l.qty, "unit": l.unit,
                       "display": round_out(l.qty, l.unit),
                       "sources": sorted(set(l.sources)),
                       "merged_into": l.merged_into or None,
                       "raw": l.raws} for l in lines],
            "unknown": [{"recipe": s, "line": r} for s, r in unknown],
            "scaling": [{"recipe": s, "why": w} for s, w in scales],
        }, indent=2))
        return 0

    print(emit(week, lines, unknown, merges, links, ae, scales, items))
    if not args.no_coupling:
        print(coupling_report(lines, items))
    return 0


if __name__ == "__main__":
    sys.exit(main())
