"""The item table, and the rule that decides what an ingredient name means.

`items.md` is the normalization target. This module reads it and answers one question:
*what canonical item does this string name?* The answer is an Item or it is nothing.

**The mis-merge rule is the whole point of this file.** A partial match is accepted only
when every word it leaves behind is noise. `onion powder` once matched `onion` and put a
fresh onion in the cart for a teaspoon of spice, silently, across thirteen lines; `dried
thyme` matched fresh thyme across five more. Neither `powder` nor `dried` is noise — they
name which aisle you walk to. So a leftover word that is not on the noise list refuses the
match, and the line goes to the unknown channel where somebody reads it.

**A mis-merge is worse than an unknown line**, because an unknown line gets printed.
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field
from itertools import combinations
from pathlib import Path

from . import units as U

# Words that may be left over by a partial match without changing what you buy.
#
# Read the exclusions as carefully as the entries. `dried`, `fresh`, `powder`, `salt`,
# `seed`, `mix`, `sauce`, `juice`, `soup`, `ground`, `and` and `or` are all deliberately
# absent. Each one names a different thing in a different aisle, and each one has already
# caused a real mis-merge in this project or would.
NOISE = {
    "large", "medium", "med", "small", "extra", "jumbo", "whole", "half", "halved",
    "chopped", "minced", "sliced", "diced", "cubed", "shredded", "grated", "crushed",
    "peeled", "trimmed", "drained", "undrained", "rinsed", "seeded", "stemmed",
    "thinly", "finely", "coarsely", "roughly", "freshly", "lightly", "generous",
    "optional", "divided", "softened", "melted", "cooled", "warm", "warmed",
    "cut", "into", "chunks", "pieces", "quarters", "quartered", "wedges",
    "uncooked", "raw", "boneless", "skinless", "lean", "marbled", "well",
    "of", "the", "a", "an", "on", "top", "for", "serving", "garnish", "to", "taste",
    "thick", "thin", "ripe", "packed", "heaping", "level", "about", "approx",
    "plus", "more", "extra-virgin", "good", "quality", "your", "favorite", "favourite",
    "inch", "inches", "long", "wide", "size", "sized", "pieces", "strips",
}

# Extra words tolerated only when deciding whether a trailing comma clause is an aside.
# `and` belongs here and nowhere else: "peeled and thinly sliced" is a preparation note,
# while "thyme and rosemary" is two ingredients on one line and must be refused.
_CLAUSE_EXTRA = {"and", "or", "use", "packs", "grease", "baking", "sheet", "pan",
                 "needed", "if", "using", "desired", "taste", "top", "with", "in",
                 "then", "fat", "excess", "off", "ends", "stems", "core", "skin",
                 "seeds", "juices", "room", "temperature"}

_PUNCT = re.compile(r"[^\w\s&®'À-ɏ]", re.UNICODE)
_WS = re.compile(r"\s+")


def normalize(text: str) -> str:
    """Lower-case, de-punctuate and collapse a name so two spellings compare equal.

    Applied identically to the table and to the query, so it can never introduce a
    difference that only one side sees.
    """
    text = text.replace("_", " ").replace("-", " ").lower()
    text = _PUNCT.sub(" ", text)
    return _WS.sub(" ", text).strip()


def singular(word: str) -> str:
    """A deliberately small de-pluraliser. It only has to handle grocery nouns."""
    if len(word) > 3 and word.endswith("ies"):
        return word[:-3] + "y"
    if len(word) > 3 and word.endswith("oes"):
        return word[:-2]
    if len(word) > 3 and word.endswith("ses"):
        return word[:-2]
    if len(word) > 2 and word.endswith("s") and not word.endswith("ss"):
        return word[:-1]
    return word


@dataclass
class Item:
    canonical: str
    aisle: str
    staple: bool
    synonyms: list[str] = field(default_factory=list)
    each_equiv_raw: str = ""
    graph: dict = field(default_factory=dict)


@dataclass
class Match:
    item: Item
    kind: str          # exact | accepts | partial
    matched: str       # the string that actually hit the table


class ItemTable:
    def __init__(self, items: list[Item]):
        self.items = {i.canonical: i for i in items}
        self.index: dict[str, str] = {}
        for item in items:
            self._add(normalize(item.canonical), item.canonical)
            for syn in item.synonyms:
                self._add(normalize(syn), item.canonical)

    def _add(self, key: str, canonical: str) -> None:
        if key and key not in self.index:
            self.index[key] = canonical

    def lookup(self, phrase: str) -> str | None:
        """Exact table hit for a phrase, trying its singular form too."""
        key = normalize(phrase)
        if not key:
            return None
        if key in self.index:
            return self.index[key]
        words = key.split()
        if words:
            depluralised = " ".join(words[:-1] + [singular(words[-1])])
            if depluralised in self.index:
                return self.index[depluralised]
        return None

    def resolve(self, name: str, accepts: list[str] | None = None) -> Match | None:
        """Resolve an ingredient name to an Item, or return None and let it be printed.

        Order matters. A declared `accepts:` wins, because the household stated it. Then
        an exact hit. Only then the partial probe, under the mis-merge rule.
        """
        for alt in accepts or []:
            canonical = self.lookup(alt)
            if canonical:
                return Match(self.items[canonical], "accepts", normalize(alt))

        canonical = self.lookup(name)
        if canonical:
            return Match(self.items[canonical], "exact", normalize(name))

        return self._partial(name)

    def _partial(self, name: str) -> Match | None:
        """Probe sub-phrases, longest first, keeping only those that leave pure noise.

        Sub-phrases keep word order but need not be contiguous, so `fresh chopped
        parsley` can reach the `fresh parsley` synonym. That widens what resolves and
        does not weaken the rule: the leftover test still runs on every candidate, so
        `onion powder` still cannot reach `onion`.

        Two different items matching at the same length is an ambiguity, and an ambiguity
        is refused rather than guessed.
        """
        words = normalize(name).split()
        if not words:
            return None
        # Long names are almost always prose the parser should refuse anyway, and the
        # subset probe is exponential. Cap it rather than hang.
        if len(words) > 9:
            words = words[:9]

        for size in range(len(words), 0, -1):
            hits: dict[str, str] = {}
            for combo in combinations(range(len(words)), size):
                phrase = " ".join(words[i] for i in combo)
                canonical = self.lookup(phrase)
                if not canonical:
                    continue
                leftover = [w for i, w in enumerate(words) if i not in combo]
                if all(w in NOISE or w.isdigit() for w in leftover):
                    hits.setdefault(canonical, phrase)
            if len(hits) == 1:
                canonical, phrase = next(iter(hits.items()))
                return Match(self.items[canonical], "partial", phrase)
            if len(hits) > 1:
                return None      # ambiguous: refuse, never pick
        return None


# --- reading items.md ------------------------------------------------------

_EQUIV_SIDE = re.compile(r"^\s*(\d+(?:\.\d+)?)?\s*([A-Za-z.]+)")


def parse_each_equiv(raw: str, item_names: set[str]) -> list[tuple[float, str, float, str]]:
    """Read the `each_equiv` cell into conversion clauses.

    Clauses are separated by `;` and each is `<qty> <unit> = <qty> <unit>`. A side that
    names the item itself — `1 lemon = 3 tbsp` — means one of the thing you buy, so it
    reads as `ea`. Trailing descriptive words are ignored: in `1 ea = 1 cup sliced`,
    `sliced` says which cup, not which unit.
    """
    clauses = []
    for clause in raw.split(";"):
        if "=" not in clause:
            continue
        left, right = clause.split("=", 1)
        parsed = []
        for side in (left, right):
            m = _EQUIV_SIDE.match(side.strip())
            if not m:
                parsed = []
                break
            qty = float(m.group(1)) if m.group(1) else 1.0
            token = m.group(2)
            unit = U.normalize_unit(token)
            if unit is None:
                unit = "ea" if normalize(token) in item_names else None
            if unit is None:
                parsed = []
                break
            parsed.append((qty, unit))
        if len(parsed) == 2:
            clauses.append((parsed[0][0], parsed[0][1], parsed[1][0], parsed[1][1]))
    return clauses


def load_items(path) -> ItemTable:
    """Read `items.md`. The pipe table is the data; everything else is prose for people."""
    rows = []
    for line in Path(path).read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line.startswith("|"):
            continue
        cells = [c.strip() for c in line.strip("|").split("|")]
        if len(cells) < 5:
            continue
        canonical = cells[0]
        if canonical in ("canonical", "---") or set(canonical) <= {"-", " "}:
            continue
        synonyms = [s.strip() for s in cells[4].split(",") if s.strip()]
        rows.append(Item(
            canonical=canonical,
            aisle=cells[1],
            staple=cells[2].lower().startswith("y"),
            synonyms=synonyms,
            each_equiv_raw=cells[3],
        ))

    names = set()
    for item in rows:
        names.add(normalize(item.canonical))
        for syn in item.synonyms:
            names.add(normalize(syn))

    for item in rows:
        clauses = parse_each_equiv(item.each_equiv_raw, names)
        item.graph = U.build_graph(clauses)

    return ItemTable(rows)
