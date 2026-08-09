"""Reading one ingredient line.

The output of this module and the input of the shopping list are two different types, and
that is on purpose. A line this parser refuses keeps its `raw` text, gets `parsed = None`,
and goes onto the list flagged. **It is never dropped and never guessed at.** Silently
losing a line means somebody gets home without the chuck roast.

The grammar this handles is the grammar the recipe files actually contain, which is messy
because the sources are messy:

    3 pounds boneless beef chuck, (well-marbled, cut into 1½-inch pieces)
    1 10.75 oz can cream of chicken soup
    2 16.3 oz. tubes refrigerated biscuits cut into quarters
    1 (8-ounce) package Borden® Cheese Thick Cut Shredded Four Cheese Mexican
    1-2 tsp pickle juice
    ⅓ cup reduced sodium soy sauce
    Tube of biscuits
    salt, to taste
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field

from . import items as I
from . import units as U

# --- refusal reasons -------------------------------------------------------

TWO_INGREDIENTS = "two ingredients on one line"
AMBIGUOUS_OR = "ambiguous alternative — nobody said which"
MALFORMED = "malformed line"
NO_ITEM_ROW = "no items.md row"

_FRACTIONS = {
    "½": "1/2", "⅓": "1/3", "⅔": "2/3", "¼": "1/4", "¾": "3/4",
    "⅕": "1/5", "⅖": "2/5", "⅗": "3/5", "⅘": "4/5",
    "⅙": "1/6", "⅚": "5/6", "⅛": "1/8", "⅜": "3/8", "⅝": "5/8", "⅞": "7/8",
}

_COMMENT = re.compile(r"<!--.*?-->", re.S)
_ACCEPTS = re.compile(r"\baccepts:\s*(.+)$", re.I)
# Order matters: the mixed and bare fraction forms must be tried before the plain
# integer, or `1/2 tsp` reads as a bare `1` and the rest of the line becomes the name.
_NUM = r"\d+\s+\d+/\d+|\d+/\d+|\d+(?:\.\d+)?"
_NUM_RE = re.compile(rf"^\s*({_NUM})")
_RANGE_RE = re.compile(rf"^\s*({_NUM})\s*(?:-|–|—|\bto\b)\s*({_NUM})")
_TOKEN_RE = re.compile(r"^\s*([A-Za-z.]+)")
_TRAILING_SIZE = re.compile(
    rf"[,\s]+({_NUM})\s*([A-Za-z.]+)\s*([A-Za-z.]+)?\s*$")


@dataclass
class Parsed:
    qty: float | None
    unit: str | None
    item_name: str
    note: str = ""


@dataclass
class Line:
    raw: str
    parsed: Parsed | None = None
    accepts: list[str] = field(default_factory=list)
    refusal: str = ""
    match: I.Match | None = None

    @property
    def resolved(self) -> bool:
        return self.match is not None


# --- text tidying ----------------------------------------------------------

def _expand_fractions(text: str) -> str:
    for glyph, ascii_form in _FRACTIONS.items():
        # `1½` is one and a half; a bare `½` is a half.
        text = re.sub(rf"(?<=\d){re.escape(glyph)}", f" {ascii_form}", text)
        text = text.replace(glyph, ascii_form)
    return text


def _strip_parens(text: str) -> tuple[str, str]:
    """Remove balanced parenthetical groups, keeping them as a note.

    Nested groups occur in the real files — `((baby yukons), cut in half)` — so this
    counts depth rather than matching a regex.
    """
    out, note, depth = [], [], 0
    for ch in text:
        if ch == "(":
            depth += 1
            if depth == 1:
                continue
        if ch == ")" and depth > 0:
            depth -= 1
            if depth == 0:
                note.append(" ")
                continue
        (note if depth else out).append(ch)
    return "".join(out), "".join(note).strip()


def _is_aside(clause: str) -> bool:
    """Is this trailing comma clause a preparation note rather than an ingredient?

    `, peeled and thinly sliced` is an aside. `, to taste` is an aside. A clause opening
    with `plus` or `or use` is an aside. Anything else stays, because a clause this test
    does not recognise may be a second ingredient, and dropping one of those is the
    failure the whole file guards against.
    """
    words = I.normalize(clause).split()
    if not words:
        return True
    if words[0] in {"plus", "or", "for", "about"}:
        return True
    return all(w in I.NOISE or w in I._CLAUSE_EXTRA or w.isdigit() for w in words)


def _strip_asides(text: str) -> tuple[str, str]:
    notes = []
    while "," in text:
        head, _, tail = text.rpartition(",")
        if not _is_aside(tail):
            break
        notes.insert(0, tail.strip())
        text = head
    return text.strip(" ,;"), "; ".join(n for n in notes if n)


def _to_float(text: str) -> float:
    text = text.strip()
    if " " in text:                       # "1 1/2"
        whole, frac = text.split(None, 1)
        return float(whole) + _to_float(frac)
    if "/" in text:
        num, den = text.split("/", 1)
        return float(num) / float(den)
    return float(text)


# --- the parse itself ------------------------------------------------------

def _read_quantity(text: str) -> tuple[float | None, str | None, str, str]:
    """Peel the leading quantity and unit off a line.

    Returns `(qty, unit, remainder, note)`. A range takes its **high** end: told `4-5
    potatoes`, buy five. Under-buying sends somebody back to the shop.
    """
    note = ""
    numbers: list[float] = []

    m = _RANGE_RE.match(text)
    if m:
        low, high = _to_float(m.group(1)), _to_float(m.group(2))
        numbers.append(max(low, high))
        note = f"range {m.group(1)}–{m.group(2)}, taking the high end"
        text = text[m.end():]
    else:
        while len(numbers) < 2:
            m = _NUM_RE.match(text)
            if not m:
                break
            numbers.append(_to_float(m.group(1)))
            text = text[m.end():]

    def peek(s: str) -> tuple[str | None, str]:
        m = _TOKEN_RE.match(s)
        if not m:
            return None, s
        return m.group(1), s[m.end():]

    token, rest = peek(text)
    unit = U.normalize_unit(token) if token else None

    if not numbers:
        # "Tube of biscuits", "Jar of Pepperocinis" — one of whatever it is.
        if unit and U.is_count(unit):
            rest = re.sub(r"^\s*of\b", "", rest)
            return 1.0, unit, rest.strip(), note
        return None, None, text.strip(), note

    if unit is None:
        # "8 tortillas", "2 large eggs" — a bare count of the thing itself.
        return numbers[0], "ea", text.strip(), note

    if U.is_count(unit):
        rest = re.sub(r"^\s*of\b", "", rest)
        return numbers[0], unit, rest.strip(), note

    # A measure. If a container follows, the measure is the container's size and the
    # count in front is how many you buy: `2 13 oz cans …` is two cans.
    nxt, after = peek(rest)
    nxt_unit = U.normalize_unit(nxt) if nxt else None
    if nxt_unit and U.is_count(nxt_unit):
        after = re.sub(r"^\s*of\b", "", after)
        size = numbers[-1]
        count = numbers[0] if len(numbers) > 1 else 1.0
        size_note = f"{U.format_qty(size)} {unit} each"
        return count, nxt_unit, after.strip(), "; ".join(n for n in (note, size_note) if n)

    rest = re.sub(r"^\s*of\b", "", rest)
    if len(numbers) > 1:
        # Two numbers and no container: keep the first as the count and say so.
        note = "; ".join(n for n in (note, f"second number {U.format_qty(numbers[1])} not read") if n)
    return numbers[0], unit, rest.strip(), note


def parse_line(raw: str, table: I.ItemTable) -> Line:
    """Parse one ingredient line and resolve it against `items.md`."""
    line = Line(raw=raw.strip())

    text = raw.strip()
    if text.startswith("-"):
        text = text[1:]

    notes = []

    comments = _COMMENT.findall(text)
    if comments:
        notes.append(" ".join(c.strip("<!->").strip() for c in comments))
    text = _COMMENT.sub(" ", text)

    m = _ACCEPTS.search(text)
    if m:
        line.accepts = [a.strip() for a in m.group(1).split(",") if a.strip()]
        text = text[:m.start()]

    text = _expand_fractions(text)
    text, paren_note = _strip_parens(text)
    if paren_note:
        notes.append(paren_note)
    text, aside_note = _strip_asides(text)
    if aside_note:
        notes.append(aside_note)

    text = re.sub(r"\s+", " ", text).strip(" .,;")
    if not text:
        line.refusal = MALFORMED
        return line

    qty, unit, name, qty_note = _read_quantity(text)
    if qty_note:
        notes.append(qty_note)

    # A size stated after the item — "1 can chili beans 15.5 oz".
    m = _TRAILING_SIZE.search(name)
    if m and U.normalize_unit(m.group(2)):
        tail_unit = U.normalize_unit(m.group(2))
        tail_extra = U.normalize_unit(m.group(3)) if m.group(3) else None
        if tail_extra is None or U.is_count(tail_extra):
            notes.append(f"stated size {m.group(1)} {tail_unit}")
            name = name[:m.start()].strip()

    name = name.strip(" .,;")
    if not name:
        line.refusal = MALFORMED
        return line

    normalized = I.normalize(name)
    if normalized.startswith("or ") or normalized.startswith("and "):
        line.refusal = MALFORMED
        return line

    match = table.resolve(name, line.accepts)

    if match is None:
        # Retry with a trailing unit word read as the unit: "3 garlic cloves" is three
        # cloves of garlic. Tried second, so "8 oz pepperoncini pepper slices" — whose
        # own synonym ends in `slices` — resolves as itself first.
        words = normalized.split()
        if len(words) > 1 and unit in (None, "ea") and U.is_unit(words[-1]):
            retry = table.resolve(" ".join(words[:-1]), line.accepts)
            if retry is not None:
                match, unit = retry, U.normalize_unit(words[-1])

    if match is None:
        # An unresolved line naming two things, or an unstated choice between two, is a
        # refusal with a reason. It is never resolved to one of the two.
        if " and " in f" {normalized} ":
            line.refusal = TWO_INGREDIENTS
        elif " or " in f" {normalized} ":
            line.refusal = AMBIGUOUS_OR
        else:
            line.refusal = NO_ITEM_ROW
        line.parsed = Parsed(qty, unit, name, "; ".join(notes))
        return line

    line.match = match
    line.parsed = Parsed(qty, unit, name, "; ".join(n for n in notes if n))
    return line
