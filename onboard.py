#!/usr/bin/env python3
"""Onboard one raw recipe into the two stores that Step 2 needs.

A raw recipe arrives as one of three things: a URL, a block of loose text the
household typed, or a screenshot of somebody else's app. This turns any of them
into

    recipes/<slug>.md   the content   - ingredients, quantities, yield, source
    a row in corpus.md  the index     - protein, cuisine, yield, active, passive

which are deliberately separate stores (docs/step2-design.md 1). No dependencies
beyond the standard library.

    ./onboard.py --url https://natashaskitchen.com/meatloaf-recipe/
    ./onboard.py --text sources/inputs/text/tacos.txt
    ./onboard.py --transcript sources/inputs/transcripts/chili.md
    ./onboard.py --batch sources/inputs --report docs/onboarding-run.md

The rule the whole thing is built around: **never invent**. A quantity that is
not stated is recorded as not stated. A yield the source never gives is
`unknown`. A screenshot that cannot be read is reported unread. Every guess the
tool does make is labelled as one, and everything it cannot answer comes out as
a question for the household rather than as a plausible number.
"""

import argparse
import json
import os
import re
import sys
import time
import unicodedata
import urllib.error
import urllib.request
from datetime import date
from pathlib import Path

ROOT = Path(__file__).resolve().parent

# A default user agent gets 403 from a good share of recipe sites. Sending a
# real browser string is not a trick, it is the minimum to be served at all;
# when it still fails we say so rather than filing an empty recipe.
USER_AGENT = (
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 "
    "(KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
)

# ---------------------------------------------------------------------------
# ingredient grammar
# ---------------------------------------------------------------------------

UNITS = {
    "cup": "cup", "cups": "cup", "c": "cup",
    "teaspoon": "tsp", "teaspoons": "tsp", "tsp": "tsp", "tsps": "tsp",
    "tablespoon": "tbsp", "tablespoons": "tbsp", "tbsp": "tbsp",
    "tbsps": "tbsp", "tbs": "tbsp",
    "ounce": "oz", "ounces": "oz", "oz": "oz", "oz.": "oz",
    "pound": "lb", "pounds": "lb", "lb": "lb", "lbs": "lb",
    "gram": "g", "grams": "g", "g": "g", "kg": "kg",
    "ml": "ml", "l": "l", "liter": "l", "liters": "l",
    "quart": "quart", "quarts": "quart", "pint": "pint", "pints": "pint",
    "clove": "clove", "cloves": "clove",
    "slice": "slice", "slices": "slice",
    "stick": "stick", "sticks": "stick",
    "rib": "rib", "ribs": "rib",
    "sprig": "sprig", "sprigs": "sprig",
    "bunch": "bunch", "bunches": "bunch",
    "head": "head", "heads": "head",
    "pinch": "pinch", "dash": "dash",
    "fillet": "fillet", "fillets": "fillet",
    "loaf": "loaf", "loaves": "loaf",
    # packaging-defined units: the size is on the package, not in the line
    "can": "can", "cans": "can",
    "jar": "jar", "jars": "jar",
    "package": "package", "packages": "package", "pkg": "package",
    "pkt": "packet", "packet": "packet", "packets": "packet",
    "envelope": "envelope", "envelopes": "envelope",
    "tube": "tube", "tubes": "tube",
    "box": "box", "boxes": "box",
    "bag": "bag", "bags": "bag",
    "brick": "brick", "bricks": "brick",
    "container": "container", "containers": "container",
}

# Units whose real size lives on the packaging. `1 (14.5 oz) can beef broth` is
# one can, and the 14.5 oz is the can's size - two different numbers that a
# single qty field would flatten.
PACKAGE_UNITS = {
    "can", "jar", "package", "packet", "envelope", "tube", "box", "bag",
    "brick", "container",
}

VULGAR = {
    "¼": "1/4", "½": "1/2", "¾": "3/4",
    "⅓": "1/3", "⅔": "2/3",
    "⅛": "1/8", "⅜": "3/8", "⅝": "5/8", "⅞": "7/8",
    "⅕": "1/5", "⅖": "2/5", "⅗": "3/5", "⅘": "4/5",
    "⅙": "1/6", "⅚": "5/6",
}

NUM = r"\d+(?:\.\d+)?(?:\s*/\s*\d+)?"
# 2, 2.5, 1/2, 1 1/2, 4-5, 1 to 2
QTY_RE = re.compile(
    rf"^(?P<qty>{NUM}(?:\s+\d+\s*/\s*\d+)?(?:\s*(?:-|–|to)\s*{NUM})?)\s*"
)
SIZE_PAREN_RE = re.compile(
    r"^\((?P<size>[\d./\s-]+\s*(?:oz|ounce|ounces|lb|pound|pounds|g|gram|grams|"
    r"ml|l|inch|inches|-ounce|-oz)[^)]*)\)\s*",
    re.I,
)
SIZE_BARE_RE = re.compile(
    r"^(?P<size>\d+(?:\.\d+)?\s*(?:oz\.?|ounce|ounces|lb|pound|pounds|g|"
    r"gram|grams|ml|l))\s+(?=\w)",
    re.I,
)
BULLET_RE = re.compile(r"^\s*(?:[-*•●▪–‐⁃]|\d+\.)\s+")

# Words that name a kind of thing rather than a thing. An item made only of
# these names nothing you can buy - `Soup sauce` - and is a typo or a shorthand
# the household can resolve in two seconds. Ask; do not resolve it to the
# nearest plausible product.
CATEGORY_WORDS = {
    "soup", "sauce", "mix", "seasoning", "broth", "stock", "dressing",
    "powder", "paste", "juice", "oil", "spice", "blend",
}

STAPLE_HINTS = {
    "salt", "pepper", "black pepper", "kosher salt", "garlic salt", "water",
    "olive oil", "vegetable oil", "oil", "flour", "sugar", "butter",
}

PEANUT_TERMS = [
    "peanut", "peanuts", "peanut butter", "peanut oil", "peanut sauce",
    "groundnut", "satay",
]
# Bought sauces are where a peanut allergen hides without saying so on the
# recipe page. Surface the risk, do not filter silently (profile.md).
PEANUT_RISK_TERMS = [
    "teriyaki", "stir fry sauce", "stir-fry sauce", "hoisin", "pad thai",
    "sesame sauce", "thai", "satay",
]

PROTEIN_MARKERS = [
    ("beef", ["ground beef", "ground hamburger", "hamburger", "chuck roast",
              "beef chuck", "chuck", "beef roast", "stew meat", "stew beef",
              "beef stew meat",
              "steak", "brisket", "meatball", "ground chuck", "corned beef",
              "roast beef", "lean ground beef"]),
    ("chicken", ["chicken breast", "chicken thigh", "chicken thighs",
                 "shredded chicken", "rotisserie chicken", "whole chicken",
                 "chicken tenders", "cooked and shredded chicken", "chicken"]),
    ("pork", ["italian sausage", "ground sausage", "sausage", "bacon",
              "pork loin", "pork tenderloin", "pork shoulder", "ham",
              "pork chops", "pancetta", "prosciutto"]),
    ("fish", ["salmon", "tuna", "cod", "tilapia", "shrimp", "halibut",
              "yellowfin"]),
]

CUISINE_MARKERS = [
    ("Tex-Mex", ["taco seasoning", "enchilada sauce", "tortilla", "salsa",
                 "taco sauce", "refried beans", "rotel"]),
    ("Italian-American", ["marinara", "italian sausage", "provolone",
                          "italian seasoning", "pasta sauce", "giardiniera",
                          "pepperoncini", "mozzarella", "parmesan"]),
    ("Japanese-ish", ["teriyaki", "mirin", "sake"]),
    ("Chinese-ish", ["stir fry", "stir-fry", "hoisin", "oyster sauce",
                     "soy sauce"]),
]

# Passive time is unattended time. Only claim it when the source says a method
# that is unattended; do not turn a cook time into passive time by assumption -
# a 40-minute stir fry is 40 minutes of standing at the stove.
PASSIVE_METHODS = [
    ("slow cooker", [r"slow cooker", r"crock ?pot"]),
    # `dutch oven` is a pot on a stove, not an oven. Matching it as one turned
    # a simmered chili into a baked one.
    ("oven", [r"\bbakes?\b", r"\bbaking\b", r"(?<!dutch )\boven\b",
              r"\broast(?:ed|ing)?\b", r"\bbroil(?:er|ed)?\b"]),
    ("simmer", [r"\bsimmers?\b", r"\bbraises?\b", r"\bstews?\b"]),
    ("marinate", [r"\bmarinat", r"\bmarinade\b", r"\brefrigerate\b"]),
]


def normalize_fractions(text):
    for ch, rep in VULGAR.items():
        text = text.replace(ch, rep)
    return text


def parse_ingredient(raw):
    """Split one ingredient line into (qty, unit, size, item, note).

    The raw line is always kept. Everything derived from it is best effort and
    is allowed to come back empty; a line the parser cannot read is surfaced,
    never dropped (docs/step2-design.md 2).
    """
    # The bullet glyph is markup, not content; everything after it is kept
    # exactly as written, including the site's own spelling and punctuation.
    out = {
        "raw": BULLET_RE.sub("", raw.strip()).strip(),
        "qty": None,
        "unit": None,
        "size": None,
        "item": None,
        "note": None,
        "flags": [],
    }
    text = normalize_fractions(BULLET_RE.sub("", raw.strip()))
    text = text.replace(" ", " ").strip()
    if not text:
        out["flags"].append("empty")
        return out

    # The note is what follows the LAST comma that is not inside brackets.
    # Splitting on the first comma instead loses half the item on
    # `2 lb boneless, skinless chicken thighs, cubed`, and splitting inside
    # brackets shreds `8 (8-inch) tortillas (we prefer flour, but corn ...)`.
    depth, cut = 0, None
    for i, ch in enumerate(text):
        if ch in "([":
            depth += 1
        elif ch in ")]":
            depth = max(0, depth - 1)
        elif ch == "," and depth == 0:
            cut = i
    if cut is not None:
        out["note"] = text[cut + 1:].strip() or None
        text = text[:cut].strip()

    m = QTY_RE.match(text)
    if m:
        out["qty"] = re.sub(r"\s*(?:-|–)\s*", "-", m.group("qty").strip())
        text = text[m.end():]
    else:
        out["flags"].append("no-quantity")

    # a package size, either parenthesised or bare: `(14.5 oz) can`, `10.75 oz can`
    m = SIZE_PAREN_RE.match(text)
    if m:
        out["size"] = re.sub(r"\s+", " ", m.group("size")).strip()
        text = text[m.end():]
    else:
        m = SIZE_BARE_RE.match(text)
        if m:
            candidate = text[m.end():]
            first = candidate.split()[0].lower().strip(".,") if candidate.split() else ""
            if UNITS.get(first) in PACKAGE_UNITS:
                out["size"] = re.sub(r"\s+", " ", m.group("size")).strip()
                text = candidate

    words = text.split()
    if words:
        head = words[0].lower().strip(".")
        if head in UNITS:
            out["unit"] = UNITS[head]
            text = " ".join(words[1:])
        elif out["qty"] is not None:
            # `2lb ground beef` - number and unit run together
            m = re.match(r"^([a-z]+)(?=\b)", head)
            if m and m.group(1) in UNITS and len(head) == len(m.group(1)):
                out["unit"] = UNITS[m.group(1)]
                text = " ".join(words[1:])

    if out["qty"] is not None and out["unit"] is None:
        out["unit"] = "each"

    # `2lb ground beef`: qty regex stops at the digits, unit is glued on
    if out["qty"] and out["unit"] == "each" and text:
        m = re.match(r"^(lb|lbs|oz|g|kg|ml|l)\b\.?\s*", text, re.I)
        if m:
            out["unit"] = UNITS[m.group(1).lower()]
            text = text[m.end():]

    # `2 tbsp of chili powder`, `Jar of Pepperocinis`: the `of` belongs to the
    # unit, not to the item name.
    text = re.sub(r"^of\s+", "", text.strip(), flags=re.I)
    out["item"] = text.strip() or None
    if out["item"] is None:
        out["flags"].append("no-item")
    if out["size"]:
        out["flags"].append("packaged-size")
    if out["item"] and out["item"].lower() in STAPLE_HINTS:
        out["flags"].append("likely-staple")
    if out["note"] and re.search(r"\bto taste\b", out["note"], re.I):
        out["flags"].append("to-taste")
    if out["note"] and re.search(r"\bplus (?:extra|more)\b", out["note"], re.I):
        out["flags"].append("second-unmeasured-quantity")
    if re.search(r"^juice of\b", out["raw"], re.I):
        out["flags"].append("quantity-as-source-object")
    if out["item"]:
        words = [w for w in re.findall(r"[a-z]+", out["item"].lower())]
        if words and len(words) > 1 and all(w in CATEGORY_WORDS for w in words):
            out["flags"].append("ambiguous-item")
    return out


def scan_peanut(ingredients):
    """Return (verdict, evidence). Flag, never filter."""
    hits, risks = [], []
    for ing in ingredients:
        low = ing["raw"].lower()
        for term in PEANUT_TERMS:
            if re.search(rf"\b{re.escape(term)}\b", low):
                hits.append(ing["raw"])
                break
        else:
            for term in PEANUT_RISK_TERMS:
                if term in low:
                    risks.append(ing["raw"])
                    break
    if hits:
        return "CONTAINS PEANUT", hits
    if risks:
        return "check label", risks
    return "none seen", []


# A broth, a soup base or a seasoning names an animal without being one. Left
# in, `4 cups chicken broth` makes the zuppa toscana a chicken recipe.
NOT_THE_PROTEIN = re.compile(
    r"\b(broth|stock|bouillon|soup|seasoning|powder|bacon bits|base)\b", re.I)


def infer_protein(ingredients):
    for protein, markers in PROTEIN_MARKERS:
        for ing in ingredients:
            low = (ing["item"] or ing["raw"]).lower()
            if NOT_THE_PROTEIN.search(low):
                continue
            for marker in markers:
                if marker in low:
                    return protein, ing["raw"]
    return None, None


def infer_cuisine(ingredients):
    scores = {}
    evidence = {}
    for cuisine, markers in CUISINE_MARKERS:
        for ing in ingredients:
            low = ing["raw"].lower()
            for marker in markers:
                if marker in low:
                    scores[cuisine] = scores.get(cuisine, 0) + 1
                    evidence.setdefault(cuisine, []).append(ing["raw"])
    if not scores:
        return None, []
    # ties go to the earlier, more specific marker set: `teriyaki` should beat
    # `soy sauce`, which half of Asian cooking contains
    order = {c: i for i, (c, _) in enumerate(CUISINE_MARKERS)}
    best = max(scores, key=lambda c: (scores[c], -order[c]))
    # One marker is not evidence of a cuisine. Soy sauce does not make a recipe
    # Chinese and a jar of pepperoncini does not make a tuna melt Italian; both
    # of those were wrong before this line existed.
    if scores[best] < 2:
        return None, []
    return best, evidence[best]


def infer_passive(text):
    """Name an unattended method if the source says one. No time is invented."""
    low = text.lower()
    for label, markers in PASSIVE_METHODS:
        for marker in markers:
            if re.search(marker, low):
                return label
    return None


# ---------------------------------------------------------------------------
# input: a URL
# ---------------------------------------------------------------------------


def fetch(url, retries=3):
    """Fetch a page. Returns (html, error). A block is an error, not a blank."""
    last = None
    for attempt in range(retries):
        req = urllib.request.Request(url, headers={
            "User-Agent": USER_AGENT,
            "Accept": "text/html,application/xhtml+xml",
            "Accept-Language": "en-US,en;q=0.9",
        })
        try:
            with urllib.request.urlopen(req, timeout=45) as resp:
                charset = resp.headers.get_content_charset() or "utf-8"
                return resp.read().decode(charset, errors="replace"), None
        except urllib.error.HTTPError as e:
            last = f"HTTP {e.code} {e.reason}"
            if e.code in (403, 401, 429):
                return None, (
                    f"{last} - the site refused an automated fetch. "
                    "Not retried further; this needs a hand capture."
                )
        except urllib.error.URLError as e:
            last = f"network error: {e.reason}"
        except Exception as e:  # noqa: BLE001 - report anything, invent nothing
            last = f"{type(e).__name__}: {e}"
        time.sleep(2 ** attempt)
    return None, last


LDJSON_RE = re.compile(
    r"<script[^>]+type=[\"']application/ld\+json[\"'][^>]*>(.*?)</script>",
    re.S | re.I,
)


def _walk_jsonld(node):
    if isinstance(node, list):
        for item in node:
            yield from _walk_jsonld(item)
    elif isinstance(node, dict):
        yield node
        for key in ("@graph", "mainEntity", "itemListElement"):
            if key in node:
                yield from _walk_jsonld(node[key])


def extract_recipe_jsonld(html):
    """Pull the schema.org Recipe block. Structured data or nothing.

    Guessing a recipe out of page prose is exactly the invention this tool is
    not allowed to do, so there is no HTML-scraping fallback: no structured
    data means the page needs a human, and the report says so.
    """
    for blob in LDJSON_RE.findall(html):
        blob = blob.strip()
        try:
            data = json.loads(blob)
        except json.JSONDecodeError:
            try:
                data = json.loads(re.sub(r",\s*([\]}])", r"\1", blob))
            except json.JSONDecodeError:
                continue
        for node in _walk_jsonld(data):
            types = node.get("@type")
            types = [types] if isinstance(types, str) else (types or [])
            if any(str(t).lower() == "recipe" for t in types):
                return node
    return None


def _text(value):
    if value is None:
        return None
    if isinstance(value, list):
        parts = [_text(v) for v in value]
        return " ".join(p for p in parts if p) or None
    if isinstance(value, dict):
        return _text(value.get("text") or value.get("name"))
    return str(value).strip() or None


def iso_duration_to_text(value):
    if not value or not isinstance(value, str):
        return None
    m = re.match(r"^P(?:(\d+)D)?T?(?:(\d+)H)?(?:(\d+)M)?$", value.strip())
    if not m:
        return value.strip()
    days, hours, mins = (int(g) if g else 0 for g in m.groups())
    hours += days * 24
    if not hours and not mins:
        return None
    parts = []
    if hours:
        parts.append(f"{hours} hr")
    if mins:
        parts.append(f"{mins} min")
    return " ".join(parts)


def from_url(url, corpus_title=None, title=None):
    html, error = fetch(url)
    rec = {
        "title": title or corpus_title or url,
        "corpus_title": corpus_title,
        "source": url,
        "modality": "url",
        "ingredients": [],
        "yield": None,
        "yield_note": None,
        "times": [],
        "passive": None,
        "questions": [],
        "capture_notes": [],
        "status": "failed",
        "instructions_text": "",
    }
    if error:
        rec["capture_notes"].append(f"! fetch failed: {error}")
        rec["questions"].append(
            f"{rec['title']}: the page could not be fetched ({error}). "
            "Can you paste the ingredient list, or send a screenshot?"
        )
        return rec

    node = extract_recipe_jsonld(html)
    if not node:
        rec["capture_notes"].append(
            "! page fetched but carries no schema.org Recipe data; nothing was "
            "read off it (guessing from page prose is not allowed)"
        )
        rec["questions"].append(
            f"{rec['title']}: fetched, but the page has no machine-readable "
            "recipe. Can you paste the ingredients or send a screenshot?"
        )
        return rec

    if not title:
        rec["title"] = _text(node.get("name")) or rec["title"]
    raw_ings = node.get("recipeIngredient") or node.get("ingredients") or []
    if isinstance(raw_ings, str):
        raw_ings = [raw_ings]
    rec["ingredients"] = [parse_ingredient(line) for line in raw_ings if str(line).strip()]

    y = node.get("recipeYield")
    y = y[0] if isinstance(y, list) and y else y
    if y is not None:
        y = str(y).strip()
        m = re.search(r"\d+", y)
        if m:
            rec["yield"] = f"{m.group(0)} AE"
            rec["yield_note"] = f"source: {y}"
        else:
            rec["yield_note"] = f"source says: {y}"

    for label, key in (("prep", "prepTime"), ("cook", "cookTime"),
                       ("total", "totalTime")):
        pretty = iso_duration_to_text(node.get(key))
        if pretty:
            rec["times"].append(f"{label} {pretty}")

    instructions = _text(node.get("recipeInstructions")) or ""
    rec["instructions_text"] = instructions
    rec["passive"] = infer_passive(instructions + " " + (rec["title"] or ""))
    rec["status"] = "complete" if rec["ingredients"] else "failed"
    if not rec["ingredients"]:
        rec["capture_notes"].append("! recipe data found but it listed no ingredients")
        rec["questions"].append(f"{rec['title']}: no ingredient list in the page data.")
    return rec


# ---------------------------------------------------------------------------
# input: loose text, and screenshot transcripts (same parser)
# ---------------------------------------------------------------------------

HEADER_RE = re.compile(r"^(source|yield|images|modality|corpus|times|slug)\s*:\s*(.*)$", re.I)


def parse_block(text, modality, path=None):
    """Parse a typed block or a screenshot transcript.

    Both are the same shape: an optional key: value header, a title, a list of
    ingredient lines, and optional `## Capture notes` - one line per thing the
    capture could not settle. Those notes become the household's questions.
    """
    rec = {
        "title": None,
        "corpus_title": None,
        "source": None,
        "modality": modality,
        "ingredients": [],
        "yield": None,
        "yield_note": None,
        "times": [],
        "passive": None,
        "questions": [],
        "capture_notes": [],
        "status": "failed",
        "instructions_text": "",
        "images": None,
        "slug": None,
    }
    section = "head"
    body_lines = []
    group = None
    for line in text.splitlines():
        stripped = line.strip()
        if not stripped:
            continue
        low = stripped.lower()
        if low.startswith("### "):
            # a sub-list inside the ingredients, e.g. `### Meatballs`
            group = stripped[4:].strip()
            continue
        if low.startswith("## "):
            group = None
            name = low[3:].strip()
            section = {
                "ingredients": "ingredients",
                "capture notes": "notes",
                "notes": "notes",
                "method": "method",
                "instructions": "method",
            }.get(name, "other")
            continue
        if stripped.startswith("# "):
            rec["title"] = stripped[2:].strip()
            continue
        m = HEADER_RE.match(stripped)
        if m and section in ("head", "ingredients") and not BULLET_RE.match(line):
            key, value = m.group(1).lower(), m.group(2).strip()
            if key == "source":
                rec["source"] = value
            elif key == "yield":
                if value.lower() in ("unknown", "not stated", ""):
                    rec["yield_note"] = "not stated in source"
                else:
                    rec["yield"] = value
                    rec["yield_note"] = "source"
            elif key == "images":
                rec["images"] = value
            elif key == "corpus":
                rec["corpus_title"] = value
            elif key == "slug":
                rec["slug"] = value
            elif key == "times":
                rec["times"] = [v.strip() for v in value.split(",") if v.strip()]
            continue
        if section == "notes":
            rec["capture_notes"].append(BULLET_RE.sub("", stripped).strip())
            continue
        if section == "method":
            rec["instructions_text"] += " " + stripped
            continue
        body_lines.append((group, stripped))

    if rec["title"] is None and body_lines:
        # a bare typed block: the first line that is not a bullet is the title
        first = body_lines[0][1]
        if not BULLET_RE.match(first):
            rec["title"] = first
            body_lines = body_lines[1:]
        else:
            rec["title"] = path.stem.replace("-", " ").capitalize() if path else "untitled"

    rec["ingredients"] = []
    for group_name, line in body_lines:
        ing = parse_ingredient(line)
        ing["group"] = group_name
        rec["ingredients"].append(ing)
    rec["passive"] = infer_passive(rec["instructions_text"])
    if not rec["ingredients"]:
        rec["status"] = "failed"
        rec["questions"].append(
            f"{rec['title']}: no ingredients were captured at all.")
    else:
        # `!` on a note means content may be missing from the capture. That is
        # the difference between a recipe that is short a line and one that is
        # merely short an answer, and only the transcriber can tell them apart.
        rec["status"] = "partial" if any(
            n.startswith("!") for n in rec["capture_notes"]) else "complete"
    for note in rec["capture_notes"]:
        if note[:1] in ("!", "?"):
            rec["questions"].append(f"{rec['title']}: {note[1:].strip()}")
    for ing in rec["ingredients"]:
        if "ambiguous-item" in ing["flags"]:
            rec["questions"].append(
                f"{rec['title']}: \"{ing['raw']}\" names two kinds of thing and "
                "no product - typo, or shorthand for something specific?")
    if rec["yield"] is None and rec["yield_note"] is None:
        rec["yield_note"] = "not stated in source"
    return rec


def from_text(path, modality="text"):
    rec = parse_block(Path(path).read_text(), modality, Path(path))
    rec.setdefault("source", None)
    if not rec["source"]:
        rec["source"] = f"household notes ({Path(path).name})"
    return rec


def from_transcript(path):
    rec = parse_block(Path(path).read_text(), "screenshot", Path(path))
    if not rec["source"]:
        rec["source"] = "screenshot, source not shown in the image"
    return rec


# ---------------------------------------------------------------------------
# input: a screenshot
# ---------------------------------------------------------------------------

VISION_PROMPT = """Transcribe this recipe screenshot. Rules:

- Copy every ingredient line VERBATIM. Do not normalise, convert or reorder.
- If the list is cut off at the top or bottom of the image, say so in a note.
- If any text is unreadable, write [unreadable] rather than guessing.
- Never add an ingredient, a quantity or a yield the image does not show.

Reply in exactly this format:

# <recipe title as shown>
source: <site or app shown in the image, or `not shown`>
yield: <servings if the image states them, else `unknown`>

## Ingredients

- <verbatim line>

## Capture notes

- ! <content that may be missing: the list is cut off, two screenshots may not
  join, text is unreadable>
- ? <something the household has to answer: no servings shown, no can size>
- <anything else worth recording that needs no answer>

Prefix every note. `!` means the capture may be short a line, `?` means it is
short an answer, and an unprefixed note is just provenance.
"""


def from_image(path, api_key=None):
    """Read a screenshot.

    Reading pixels needs a vision model. With ANTHROPIC_API_KEY set this calls
    one and then parses its transcription through the ordinary text path. With
    no key the tool does not pretend: it writes the transcription request and
    reports the recipe as unread, which is the honest outcome and the one the
    household can act on.
    """
    import base64

    path = Path(path)
    if not api_key:
        return {
            "title": path.stem.replace("-", " ").capitalize(),
            "corpus_title": None,
            "source": f"screenshot {path.name}",
            "modality": "screenshot",
            "ingredients": [],
            "yield": None,
            "yield_note": "not read",
            "times": [],
            "passive": None,
            "questions": [
                f"{path.name}: not read. No vision model was available "
                "(ANTHROPIC_API_KEY unset), so nothing was extracted. "
                "Transcribe it with --transcript, or set a key and re-run."
            ],
            "capture_notes": ["screenshot not read: no vision model available"],
            "status": "failed",
            "instructions_text": "",
        }

    media = {".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
             ".webp": "image/webp", ".gif": "image/gif"}[path.suffix.lower()]
    payload = json.dumps({
        "model": os.environ.get("PANTRY_MODEL", "claude-sonnet-5"),
        "max_tokens": 2000,
        "messages": [{"role": "user", "content": [
            {"type": "image", "source": {
                "type": "base64", "media_type": media,
                "data": base64.b64encode(path.read_bytes()).decode()}},
            {"type": "text", "text": VISION_PROMPT},
        ]}],
    }).encode()
    req = urllib.request.Request(
        "https://api.anthropic.com/v1/messages", data=payload,
        headers={"content-type": "application/json",
                 "anthropic-version": "2023-06-01", "x-api-key": api_key})
    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            body = json.load(resp)
    except Exception as e:  # noqa: BLE001
        rec = parse_block("", "screenshot", path)
        rec["title"] = path.stem.replace("-", " ").capitalize()
        rec["capture_notes"].append(f"vision call failed: {type(e).__name__}: {e}")
        rec["questions"] = [f"{path.name}: not read ({type(e).__name__})."]
        return rec
    text = "".join(b.get("text", "") for b in body.get("content", []))
    rec = parse_block(text, "screenshot", path)
    if not rec["source"]:
        rec["source"] = f"screenshot {path.name}"
    return rec


# ---------------------------------------------------------------------------
# output: recipes/<slug>.md
# ---------------------------------------------------------------------------


def slugify(title):
    text = unicodedata.normalize("NFKD", title)
    text = text.encode("ascii", "ignore").decode().lower()
    text = re.sub(r"[^a-z0-9]+", "-", text).strip("-")
    return text or "recipe"


def render_recipe(rec):
    lines = [f"# {rec['title']}", ""]
    fields = [
        ("source", rec.get("source") or "unknown"),
        ("modality", rec["modality"]),
        ("yield", rec["yield"] or "unknown"),
    ]
    if rec.get("yield_note") and not rec["yield"]:
        fields[-1] = ("yield", f"unknown ({rec['yield_note']})")
    elif rec.get("yield_note"):
        fields[-1] = ("yield", f"{rec['yield']} ({rec['yield_note']})")
    if rec.get("times"):
        fields.append(("times", ", ".join(rec["times"]) + " (source)"))
    fields.append(("active", "not stated in source"))
    fields.append((
        "passive",
        f"{rec['passive']} (method word in the source's own steps; unverified)"
        if rec.get("passive") and not rec.get("times")
        else (f"{rec['passive']} (method word in the source's own steps; unverified)" if rec.get("passive")
              else "not stated in source"),
    ))
    verdict, evidence = scan_peanut(rec["ingredients"])
    fields.append(("peanut", verdict + (f" - {evidence[0]}" if evidence else "")))
    fields.append(("status", rec["status"]))
    if rec.get("images"):
        fields.append(("images", rec["images"]))
    width = max(len(k) for k, _ in fields) + 2
    for key, value in fields:
        lines.append(f"{(key + ':').ljust(width)}{value}")

    lines += ["", "## Ingredients", ""]
    if rec["ingredients"]:
        group = None
        for ing in rec["ingredients"]:
            if ing.get("group") != group:
                group = ing.get("group")
                if group:
                    lines += ["", f"### {group}", ""]
            suffix = ""
            if "no-quantity" in ing["flags"]:
                suffix = "    <!-- quantity not stated -->"
            elif "ambiguous-item" in ing["flags"]:
                suffix = "    <!-- ambiguous item, see open questions -->"
            lines.append(f"- {ing['raw']}{suffix}")
    else:
        lines.append("*none captured - see open questions*")

    method = " ".join((rec.get("instructions_text") or "").split())
    if method:
        lines += ["", "## Method", "",
                  "*Not parsed. Here so the cook can read the file, and so the "
                  "`passive` line above can be checked against it.*", ""]
        line = ""
        for word in method.split():
            if len(line) + len(word) + 1 > 88:
                lines.append(line)
                line = word
            else:
                line = f"{line} {word}".strip()
        if line:
            lines.append(line)

    open_notes = [n for n in rec["capture_notes"] if n[:1] in ("!", "?")]
    plain_notes = [n for n in rec["capture_notes"] if n[:1] not in ("!", "?")]
    if open_notes:
        lines += ["", "## Open questions", ""]
        for note in open_notes:
            prefix = "**content may be missing** - " if note.startswith("!") else ""
            lines.append(f"- {prefix}{note[1:].strip()}")
    if plain_notes:
        lines += ["", "## Capture notes", ""]
        for note in plain_notes:
            lines.append(f"- {note}")
    lines.append("")
    return "\n".join(lines)


# ---------------------------------------------------------------------------
# output: the corpus row
# ---------------------------------------------------------------------------

CORPUS_COLUMNS = ["Recipe", "Protein", "Cuisine", "Yield", "Active", "Passive",
                  "Last cooked", "Notes"]


def _norm_title(title):
    return re.sub(r"[^a-z0-9]+", " ", (title or "").lower()).strip()


def load_corpus(path):
    text = Path(path).read_text()
    lines = text.splitlines()
    header_idx = None
    for i, line in enumerate(lines):
        cells = [c.strip() for c in line.strip().strip("|").split("|")]
        if line.strip().startswith("|") and cells and cells[0].lower() == "recipe":
            header_idx = i
            break
    if header_idx is None:
        raise SystemExit(f"{path}: no recipe table found")
    header = [c.strip() for c in lines[header_idx].strip().strip("|").split("|")]
    return lines, header_idx, header


def ensure_yield_column(lines, header_idx, header):
    """Move `yield` into the corpus (docs/step2-design.md 1). Idempotent."""
    if any(h.lower() == "yield" for h in header):
        return lines, header, header.index(next(h for h in header if h.lower() == "yield"))
    pos = header.index("Cuisine") + 1 if "Cuisine" in header else 1
    header = header[:pos] + ["Yield"] + header[pos:]
    out = list(lines)
    i = header_idx
    while i < len(out) and out[i].strip().startswith("|"):
        cells = [c.strip() for c in out[i].strip().strip("|").split("|")]
        if i == header_idx:
            cells = header
        elif set("".join(cells)) <= set("-: "):
            cells = cells[:pos] + ["---"] + cells[pos:]
        else:
            cells = cells[:pos] + [""] + cells[pos:]
        out[i] = "| " + " | ".join(cells) + " |"
        i += 1
    return out, header, pos


def known_items(path):
    """Every canonical name and synonym in items.md, normalised."""
    known = set()
    if not Path(path).exists():
        return known
    for line in Path(path).read_text().splitlines():
        if not line.strip().startswith("|"):
            continue
        cells = [c.strip() for c in line.strip().strip("|").split("|")]
        if len(cells) < 5 or cells[0].lower() == "canonical":
            continue
        if set("".join(cells)) <= set("-: "):
            continue
        known.add(_norm_title(cells[0].replace("_", " ")))
        for syn in cells[4].split(","):
            if syn.strip():
                known.add(_norm_title(syn))
    return known


def upsert_corpus(path, rec, dry_run=False):
    """Add or complete this recipe's index row. Never overwrites a human value.

    The 23 in this batch already have corpus rows. The only field onboarding
    can add is yield; where the tool's inference disagrees with what is already
    there, it reports the disagreement and leaves the existing value alone -
    those columns were set by a person and this is not a person.
    """
    lines, header_idx, header = load_corpus(path)
    lines, header, ycol = ensure_yield_column(lines, header_idx, header)
    idx = {h.lower(): i for i, h in enumerate(header)}

    protein, p_evidence = infer_protein(rec["ingredients"])
    cuisine, c_evidence = infer_cuisine(rec["ingredients"])
    target = _norm_title(rec.get("corpus_title") or rec["title"])
    result = {"action": "none", "disagreements": [], "row": None,
              "inferred": {"protein": protein, "cuisine": cuisine},
              "existing": {"protein": None, "cuisine": None}}

    row_idx = None
    i = header_idx + 1
    while i < len(lines) and lines[i].strip().startswith("|"):
        cells = [c.strip() for c in lines[i].strip().strip("|").split("|")]
        if cells and not set("".join(cells)) <= set("-: "):
            if _norm_title(cells[0]) == target:
                row_idx = i
                break
        i += 1

    if row_idx is None:
        cells = [""] * len(header)
        cells[0] = rec.get("corpus_title") or rec["title"]
        if protein:
            cells[idx["protein"]] = f"{protein} (inferred)"
        if cuisine:
            cells[idx["cuisine"]] = f"{cuisine} (inferred)"
        cells[ycol] = rec["yield"] or "unknown"
        if "notes" in idx:
            cells[idx["notes"]] = f"onboarded from {rec['modality']}"
        lines.insert(header_idx + 1, "| " + " | ".join(cells) + " |")
        result["action"] = "added"
        result["row"] = lines[header_idx + 1]
    else:
        cells = [c.strip() for c in lines[row_idx].strip().strip("|").split("|")]
        while len(cells) < len(header):
            cells.append("")
        before = list(cells)
        cells[ycol] = rec["yield"] or "unknown"
        result["existing"] = {"protein": cells[idx["protein"]] or None,
                              "cuisine": cells[idx["cuisine"]] or None}
        existing_protein = cells[idx["protein"]].lower()
        if protein and existing_protein and protein != existing_protein:
            result["disagreements"].append(
                f"protein: corpus says '{cells[idx['protein']]}', ingredients "
                f"suggest '{protein}' ({p_evidence})")
        existing_cuisine = cells[idx["cuisine"]].lower()
        if cuisine and existing_cuisine and cuisine.lower() != existing_cuisine:
            result["disagreements"].append(
                f"cuisine: corpus says '{cells[idx['cuisine']]}', ingredients "
                f"suggest '{cuisine}' ({', '.join(c_evidence[:2])})")
        if not cells[idx["protein"]] and protein:
            cells[idx["protein"]] = f"{protein} (inferred)"
        if not cells[idx["cuisine"]] and cuisine:
            cells[idx["cuisine"]] = f"{cuisine} (inferred)"
        lines[row_idx] = "| " + " | ".join(cells) + " |"
        result["action"] = "updated" if cells != before else "unchanged"
        result["row"] = lines[row_idx]

    if not dry_run:
        Path(path).write_text("\n".join(lines) + "\n")
    return result


# ---------------------------------------------------------------------------
# driver
# ---------------------------------------------------------------------------


def qty_stats(rec):
    total = len(rec["ingredients"])
    with_qty = sum(1 for i in rec["ingredients"] if i["qty"] is not None)
    return with_qty, total


def onboard(rec, recipes_dir, corpus_path, dry_run=False):
    # Yield is asked for every time it is missing, whatever the modality. The
    # planner needs it for leftovers and only the household can supply it.
    if not rec["yield"] and rec["ingredients"]:
        rec["questions"].append(
            f"{rec['title']}: how many adults does this feed? The source never "
            "states servings.")
        rec["capture_notes"].append(
            "? how many adults does this feed? The source never states servings.")
    slug = rec.get("slug") or slugify(rec.get("corpus_title") or rec["title"])
    rec["slug"] = slug
    path = Path(recipes_dir) / f"{slug}.md"
    body = render_recipe(rec)
    if not dry_run:
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(body)
    corpus = upsert_corpus(corpus_path, rec, dry_run=dry_run)
    with_qty, total = qty_stats(rec)
    verdict, evidence = scan_peanut(rec["ingredients"])
    return {
        "title": rec["title"],
        "corpus_title": rec.get("corpus_title") or rec["title"],
        "slug": slug,
        "file": str(path.relative_to(ROOT)) if path.is_absolute() else str(path),
        "modality": rec["modality"],
        "status": rec["status"],
        "source": rec.get("source"),
        "yield": rec["yield"] or "unknown",
        "yield_note": rec.get("yield_note"),
        "ingredients": total,
        "with_quantity": with_qty,
        "items": [i["item"] for i in rec["ingredients"] if i["item"]],
        "peanut": verdict,
        "peanut_evidence": evidence,
        "questions": rec["questions"],
        "corpus": corpus,
    }


def sync_existing(recipes_dir, corpus_path, dry_run=False):
    """Carry the yield out of hand-written recipe files into the corpus index.

    Only fills the yield of a recipe that already has a corpus row. A file with
    no row is left alone and reported: membership in the corpus is earned by
    being cooked and liked ( 4), and having a file is not that.
    """
    results = []
    for path in sorted(Path(recipes_dir).glob("*.md")):
        text = path.read_text()
        title = next((l[2:].strip() for l in text.splitlines()
                      if l.startswith("# ")), path.stem)
        m = re.search(r"^yield:\s*(.+)$", text, re.M)
        y = m.group(1).strip() if m else None
        y = None if not y or y.lower().startswith("unknown") else y.split(" (")[0]
        lines, header_idx, header = load_corpus(corpus_path)
        rows = {}
        for line in lines[header_idx + 1:]:
            if not line.strip().startswith("|"):
                continue
            name = line.strip().strip("|").split("|")[0].strip()
            if name and not set(name) <= set("-: "):
                rows[_norm_title(name)] = name
                rows[slugify(name)] = name
        # a file's own title is the source's ("Meatloaf Recipe"), which is not
        # the corpus row's ("Meatloaf"); the slug is what actually ties them
        row = rows.get(_norm_title(title)) or rows.get(path.stem)
        if not row:
            results.append((path.name, title, y, "no corpus row - left alone"))
            continue
        rec = {"title": title, "corpus_title": row, "yield": y,
               "ingredients": [], "modality": "existing file"}
        out = upsert_corpus(corpus_path, rec, dry_run=dry_run)
        results.append((path.name, title, y, out["action"]))
    return results


def run_batch(directory, recipes_dir, corpus_path, dry_run=False):
    """Process a directory of inputs: urls.tsv, text/*.txt, transcripts/*.md."""
    directory = Path(directory)
    results = []
    urls = directory / "urls.tsv"
    if urls.exists():
        for line in urls.read_text().splitlines():
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            parts = [p.strip() for p in line.split("\t") if p.strip()]
            corpus_title, url = (parts + [None])[:2] if len(parts) > 1 else (None, parts[0])
            print(f"  url        {corpus_title or url}", file=sys.stderr)
            rec = from_url(url, corpus_title=corpus_title)
            results.append(onboard(rec, recipes_dir, corpus_path, dry_run))
    for path in sorted((directory / "text").glob("*.txt")):
        print(f"  text       {path.name}", file=sys.stderr)
        results.append(onboard(from_text(path), recipes_dir, corpus_path, dry_run))
    for path in sorted((directory / "transcripts").glob("*.md")):
        print(f"  screenshot {path.name}", file=sys.stderr)
        results.append(onboard(from_transcript(path), recipes_dir, corpus_path, dry_run))
    for path in sorted((directory / "screenshots").glob("*.png")):
        print(f"  screenshot {path.name}", file=sys.stderr)
        rec = from_image(path, os.environ.get("ANTHROPIC_API_KEY"))
        results.append(onboard(rec, recipes_dir, corpus_path, dry_run))
    return results


def summarize(results):
    by_modality = {}
    for r in results:
        m = by_modality.setdefault(r["modality"], {"n": 0, "complete": 0,
                                                   "partial": 0, "failed": 0,
                                                   "lines": 0, "with_qty": 0,
                                                   "yield": 0})
        m["n"] += 1
        m[r["status"]] += 1
        m["lines"] += r["ingredients"]
        m["with_qty"] += r["with_quantity"]
        if r["yield"] != "unknown":
            m["yield"] += 1
    return by_modality


def render_report(results):
    by = summarize(results)
    out = [
        "# Onboarding run",
        "",
        f"*Generated by `onboard.py` on {date.today().isoformat()}. "
        "Regenerating overwrites this file.*",
        "",
        "## Per modality",
        "",
        "| Modality | Recipes | Complete | Partial | Failed | Ingredient lines "
        "| With a stated quantity | Yield stated |",
        "|---|---|---|---|---|---|---|---|",
    ]
    for modality in ("url", "text", "screenshot"):
        m = by.get(modality)
        if not m:
            continue
        pct = f"{100 * m['with_qty'] // m['lines']}%" if m["lines"] else "-"
        out.append(
            f"| {modality} | {m['n']} | {m['complete']} | {m['partial']} | "
            f"{m['failed']} | {m['lines']} | {m['with_qty']} ({pct}) | "
            f"{m['yield']}/{m['n']} |")
    total = len(results)
    complete = sum(1 for r in results if r["status"] == "complete")
    partial = sum(1 for r in results if r["status"] == "partial")
    failed = sum(1 for r in results if r["status"] == "failed")
    out += ["", f"**{total} processed - {complete} complete, {partial} partial, "
            f"{failed} failed.**", "", "## Per recipe", "",
            "| Recipe | Modality | Status | Lines | With qty | Yield | Peanut | "
            "Open questions |", "|---|---|---|---|---|---|---|---|"]
    for r in sorted(results, key=lambda r: (r["modality"], r["corpus_title"])):
        out.append(
            f"| [{r['corpus_title']}]({r['file']}) | {r['modality']} | "
            f"{r['status']} | {r['ingredients']} | {r['with_quantity']} | "
            f"{r['yield']} | {r['peanut']} | {len(r['questions'])} |")

    questions = [(r["corpus_title"], q) for r in results for q in r["questions"]]
    out += ["", "## Questions for the household", ""]
    if questions:
        out.append(f"{len(questions)} question(s). Each one is something the "
                   "tool refused to guess.")
        out.append("")
        for title, q in questions:
            out.append(f"- **{title}** - {q.split(': ', 1)[-1]}")
    else:
        out.append("None.")

    dis = [(r["corpus_title"], d) for r in results for d in r["corpus"]["disagreements"]]
    out += ["", "## Corpus disagreements", "",
            "*Inference vs. what is already in `corpus.md`. The existing value "
            "wins; this is a list for a human to settle.*", ""]
    if dis:
        for title, d in dis:
            out.append(f"- **{title}** - {d}")
    else:
        out.append("None.")

    out += ["", "## Inference vs. the corpus", "",
            "*`protein` and `cuisine` are read off the ingredients. The corpus "
            "value always wins - these columns were set by a person. This is "
            "how often the tool would have agreed with one.*", "",
            "| Field | Agreed | Could not infer | Disagreed |", "|---|---|---|---|"]
    for field in ("protein", "cuisine"):
        agreed = none = dis = 0
        misses = []
        for r in results:
            guess = r["corpus"]["inferred"][field]
            actual = (r["corpus"]["existing"][field] or "").lower()
            if not guess:
                none += 1
                misses.append(r["corpus_title"])
            elif actual and guess.lower() != actual:
                dis += 1
            else:
                agreed += 1
        out.append(f"| {field} | {agreed} | {none} | {dis} |")
        if misses:
            out.append(f"| | *{', '.join(misses)}* | | |")

    known = known_items(ROOT / "items.md")
    unknown = sorted({
        item for r in results for item in r["items"]
        if _norm_title(item) not in known
    })
    out += ["", "## Items not yet in `items.md`", "",
            "*Step 2 normalises against `items.md` ( 3), which was seeded from "
            "four recipes. These are the item names these recipes use that it "
            "does not know yet. An unknown item is not an error - it defaults "
            "to `aisle: other, staple: no` and gets reported - but the table "
            "has to grow by this much before a list is right.*", "",
            f"{len(unknown)} unrecognised item name(s):", ""]
    out.append(", ".join(f"`{i}`" for i in unknown) if unknown else "None.")

    peanut = [r for r in results if r["peanut"] != "none seen"]
    out += ["", "## Peanut", ""]
    if peanut:
        for r in peanut:
            out.append(f"- **{r['corpus_title']}** - {r['peanut']}: "
                       f"{'; '.join(r['peanut_evidence'][:3])}")
    else:
        out.append("No peanut ingredient found in any recipe, and no bought "
                   "sauce that could carry one.")
    out.append("")
    return "\n".join(out)


def main():
    p = argparse.ArgumentParser(
        description="Onboard a raw recipe into recipes/ and corpus.md.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__)
    src = p.add_mutually_exclusive_group(required=True)
    src.add_argument("--url", help="fetch and read a recipe page")
    src.add_argument("--text", type=Path, help="a typed block of loose text")
    src.add_argument("--transcript", type=Path,
                     help="a verbatim transcription of a screenshot")
    src.add_argument("--image", type=Path, help="a screenshot (needs a vision model)")
    src.add_argument("--batch", type=Path, help="a directory of inputs")
    src.add_argument("--sync", action="store_true",
                     help="copy yields out of existing recipes/ files into the corpus")
    p.add_argument("--corpus-row", help="the title of the corpus row this belongs to")
    p.add_argument("--title", help="override the recipe title")
    p.add_argument("--recipes", type=Path, default=ROOT / "recipes")
    p.add_argument("--corpus", type=Path, default=ROOT / "corpus.md")
    p.add_argument("--report", type=Path, help="write a run report here")
    p.add_argument("--dry-run", action="store_true",
                   help="parse and report, write nothing")
    p.add_argument("--json", action="store_true", help="print the result as JSON")
    args = p.parse_args()

    if args.sync:
        for name, title, y, action in sync_existing(args.recipes, args.corpus,
                                                    args.dry_run):
            print(f"{action:28} {name:38} yield {y or 'unknown'}")
        return

    if args.batch:
        results = run_batch(args.batch, args.recipes, args.corpus, args.dry_run)
    else:
        if args.url:
            rec = from_url(args.url, corpus_title=args.corpus_row, title=args.title)
        elif args.text:
            rec = from_text(args.text)
        elif args.transcript:
            rec = from_transcript(args.transcript)
        else:
            rec = from_image(args.image, os.environ.get("ANTHROPIC_API_KEY"))
        if args.corpus_row:
            rec["corpus_title"] = args.corpus_row
        if args.title:
            rec["title"] = args.title
        results = [onboard(rec, args.recipes, args.corpus, args.dry_run)]

    if args.json:
        print(json.dumps(results, indent=2))
    else:
        for r in results:
            print(f"{r['status']:9} {r['modality']:11} {r['file']:45} "
                  f"{r['with_quantity']}/{r['ingredients']} lines with a "
                  f"quantity, yield {r['yield']}")
            for q in r["questions"]:
                print(f"    ? {q}")
            for d in r["corpus"]["disagreements"]:
                print(f"    ! corpus {d}")

    if args.report:
        text = render_report(results)
        if args.dry_run:
            print(text)
        else:
            args.report.write_text(text)
            print(f"\nreport: {args.report}", file=sys.stderr)


if __name__ == "__main__":
    main()
