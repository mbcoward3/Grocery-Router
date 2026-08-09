"""Reading the household's markdown files.

**The markdown files are the state.** There is no database and no hidden cache. The
household corrects a file and the next run reads the correction — `profile.md` calls that
the trust mechanism, and it only works if nothing here keeps a second copy of anything.
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field
from pathlib import Path

from .items import ItemTable, load_items, singular
from .recipes import Recipe, Yield, load_recipes, parse_yield


@dataclass
class CorpusRow:
    title: str
    slug: str
    protein: str
    cuisine: str
    yield_raw: str
    active: str
    passive: str
    last_cooked: str
    notes: str
    untried: bool = False       # True for a candidates.md row
    yield_: Yield = field(default_factory=lambda: Yield("unknown"))

    @property
    def label(self) -> str:
        return f"{self.title} [candidate]" if self.untried else self.title


@dataclass
class Household:
    base_ae: float = 2.5
    members: list[str] = field(default_factory=list)
    allergens: list[str] = field(default_factory=list)
    portion_conversions: dict[str, float] = field(default_factory=dict)
    open_conversions: list[str] = field(default_factory=list)


def _table_rows(text: str) -> list[list[str]]:
    """Every pipe-table row in a markdown file, header and rule lines removed."""
    rows = []
    for line in text.splitlines():
        line = line.strip()
        if not line.startswith("|"):
            continue
        cells = [c.strip() for c in line.strip("|").split("|")]
        if not cells or all(set(c) <= {"-", " ", ":"} for c in cells):
            continue
        rows.append(cells)
    return rows


def _load_rows(path: Path, untried: bool) -> list[CorpusRow]:
    if not path.exists():
        return []
    rows = _table_rows(path.read_text(encoding="utf-8"))
    out = []
    for cells in rows:
        if len(cells) < 6 or cells[0].lower() == "recipe":
            continue
        cells = cells + [""] * (9 - len(cells))
        row = CorpusRow(
            title=cells[0], slug=cells[1], protein=cells[2], cuisine=cells[3],
            yield_raw=cells[4], active=cells[5], passive=cells[6],
            last_cooked=cells[7] if not untried else "",
            notes=cells[8], untried=untried,
        )
        row.yield_ = parse_yield(row.yield_raw)
        out.append(row)
    return out


def _load_sides(path: Path) -> list[CorpusRow]:
    """`sides.md` has its own column shape, and today it has no rows at all.

    That emptiness is the file's whole point, so this reads whatever is there and never
    invents a row to fill it. Columns: Side | Goes with | Season | Active | Passive |
    Last served | Notes.
    """
    if not path.exists():
        return []
    out = []
    for cells in _table_rows(path.read_text(encoding="utf-8")):
        if not cells[0] or cells[0].lower() == "side":
            continue
        cells = cells + [""] * (7 - len(cells))
        out.append(CorpusRow(
            title=cells[0], slug=_slugify(cells[0]), protein="", cuisine=cells[1],
            yield_raw="unknown", active=cells[3], passive=cells[4],
            last_cooked=cells[5], notes=cells[6],
        ))
    return out


def _slugify(title: str) -> str:
    return re.sub(r"[^a-z0-9]+", "-", title.lower()).strip("-")


_BASE_AE = re.compile(r"base\s*~?\s*([\d.]+)\s*ae", re.I)
_CONVERSION_HINT = re.compile(r"^\|?\s*([a-z ]+?)\s*\|\s*([\d.]*)\s*\|", re.I)


def load_household(profile_path: Path) -> Household:
    """Read the parts of `profile.md` that code acts on.

    Everything else in that file is prose for the planner and is passed through
    untouched. Code reads four things: the household's base appetite, who lives here, the
    allergens, and the portion conversions.
    """
    house = Household()
    if not profile_path.exists():
        return house
    text = profile_path.read_text(encoding="utf-8")

    m = _BASE_AE.search(text)
    if m:
        house.base_ae = float(m.group(1))

    if re.search(r"allergies:\s*\*?\*?\s*peanut", text, re.I):
        house.allergens = ["peanut"]

    section = ""
    for block in re.split(r"^## ", text, flags=re.M):
        if block.lower().startswith("portion conversions"):
            section = block
        if block.lower().startswith("members"):
            house.members = [
                line.strip("- ").strip()
                for line in block.splitlines()
                if line.strip().startswith("- ")
            ]

    for cells in _table_rows(section):
        if len(cells) < 2 or cells[0].lower().startswith("portion"):
            continue
        noun = singular(cells[0].strip().lower())
        value = cells[1].strip()
        if value:
            try:
                house.portion_conversions[noun] = float(value)
                continue
            except ValueError:
                pass
        house.open_conversions.append(noun)

    return house


@dataclass
class Repo:
    root: Path
    items: ItemTable
    recipes: dict[str, Recipe]
    corpus: list[CorpusRow]
    candidates: list[CorpusRow]
    sides: list[CorpusRow]
    household: Household

    @property
    def all_rows(self) -> list[CorpusRow]:
        return self.corpus + self.candidates

    def row(self, slug: str) -> CorpusRow | None:
        for r in self.all_rows:
            if r.slug == slug:
                return r
        return None

    def missing_recipe_files(self) -> list[CorpusRow]:
        """Rows whose `Slug` names no file. Reported, never guessed at."""
        return [r for r in self.all_rows if r.slug not in self.recipes]

    def target_ae(self, guests: int = 0) -> float:
        return self.household.base_ae + max(0, guests)


def load(root: Path | str = ".") -> Repo:
    root = Path(root)
    items = load_items(root / "items.md")
    return Repo(
        root=root,
        items=items,
        recipes=load_recipes(root / "recipes", items),
        corpus=_load_rows(root / "corpus.md", untried=False),
        candidates=_load_rows(root / "candidates.md", untried=True),
        sides=_load_sides(root / "sides.md"),
        household=load_household(root / "profile.md"),
    )
