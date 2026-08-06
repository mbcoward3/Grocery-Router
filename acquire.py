#!/usr/bin/env python3
"""Acquisition: find a recipe nobody bookmarked.

    ./acquire.py                      # read this week's gaps, fill the biggest
    ./acquire.py --protein pork       # go after one thing
    ./acquire.py --dry-run            # search, capture, judge, write nothing

`docs/pantry-router-proposal.md` calls acquisition half the job, and until now it
was the half that did not exist: `candidates.md` had three rows and a human typed
all three. The tool had never searched for a recipe, never read a page it was not
handed, and never judged fit against the profile.

**The bar, and everything here is built around it: never invent a recipe.** Every
candidate this produces resolves to a page somebody can go and read. Nothing is
written from a title, a memory or a guess - `1 lb chicken breast because soup
usually has chicken` is the failure this project exists to avoid, and the
acquisition path is where that failure would be easiest to commit.

**Where it searches, and why that is the interesting decision.** Not a search
engine: the sites the household already cooks from, read off the `Notes` and
`Source` columns of `corpus.md` and `candidates.md`. Nine of the eleven expose
the WordPress REST search API, which is a documented public endpoint returning
real posts - no key, no dependency, no scraping, and no way to surface a page
that does not exist.

That is a smaller search surface than the open web and it is the right one. The
corpus is the household's own expression of what it wants, and the domains in it
are that same signal about *sources* - these are the kitchens whose food they
already cook and whose shortcut-heavy style `profile.md` documents. It also grows
on its own: cook a recipe from a new site, promote it, and that site is in the
search surface next week. Widening beyond it is a decision for the household,
not a default.

**No model runs here**, and that is a choice rather than an omission. Every fit
signal that matters is computable - peanut off the capture, active time off the
source's stated prep, protein and cuisine off the ingredients, and whether the
household already has it. What a model would add is taste, and the household
supplies that by cooking the thing. The one place a model would genuinely help is
the query, and a bad query costs a wasted search rather than a wrong claim.

Standard library only.
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass, field

import onboard
import pantry

USER_AGENT = "Mozilla/5.0 (compatible; pantry-router/0.1; +household meal planner)"
TIMEOUT = 15
# A courtesy pause between requests to one host. These are small independent
# recipe sites, not an API anybody is being paid for.
PAUSE = 0.7
# Ceilings, so a bad query cannot turn into a crawl. Acquisition adds one or two
# recipes a week; it has no reason to make fifty requests to find them.
MAX_HITS_PER_SOURCE = 5
MAX_FETCHES = 12


class Unavailable(Exception):
    """A source could not be searched. Never fatal - the next one is tried."""


# --------------------------------------------------------------------------- #
# Where we are allowed to look
# --------------------------------------------------------------------------- #

DOMAIN = re.compile(r"\b((?:[a-z0-9-]+\.)+(?:com|net|org|co|us|kitchen|recipes))\b", re.I)
# Read off the corpus, so the household's sources are the search surface. Bare
# hosts in a Notes cell are what this file actually contains, e.g.
# `thecountrycook.net`, alongside full URLs in a Source cell.
NOT_A_SOURCE = {"schema.org", "example.com"}


def sources() -> list[str]:
    """The domains this household already cooks from, most-used first.

    Deliberately not a constant. A hardcoded list would go stale the first time
    someone promotes a recipe from a new site, and it would also be *me* deciding
    whose food this household likes, which is exactly the judgment the corpus is
    already making better.
    """
    counts: dict[str, int] = {}
    for path in (pantry.CORPUS, pantry.CANDIDATES):
        if not path.exists():
            continue
        for host in DOMAIN.findall(path.read_text(encoding="utf-8")):
            host = host.lower().removeprefix("www.")
            if host in NOT_A_SOURCE:
                continue
            counts[host] = counts.get(host, 0) + 1
    return [h for h, _ in sorted(counts.items(), key=lambda kv: (-kv[1], kv[0]))]


def _get(url: str, timeout: int = TIMEOUT) -> str:
    req = urllib.request.Request(url, headers={"User-Agent": USER_AGENT})
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as e:
        raise Unavailable(f"HTTP {e.code}") from e
    except (urllib.error.URLError, TimeoutError, OSError) as e:
        raise Unavailable(str(getattr(e, "reason", e))) from e


def search(query: str, host: str, limit: int = MAX_HITS_PER_SOURCE) -> list[dict]:
    """Search one source. Returns `[{title, url, host}]`.

    The WordPress REST search endpoint, which most of these sites are built on.
    A source without it is skipped rather than scraped: a site that has not
    published a search API has not agreed to be searched by a program, and
    guessing its URL structure would be the same class of move as guessing a
    recipe from page prose - which `onboard.from_url` already refuses to do.
    """
    url = (f"https://{host}/wp-json/wp/v2/search?"
           + urllib.parse.urlencode({"search": query, "per_page": limit,
                                     "subtype": "post"}))
    body = _get(url)
    try:
        hits = json.loads(body)
    except json.JSONDecodeError as e:
        raise Unavailable(f"search returned something that was not JSON: {e}") from e
    if not isinstance(hits, list):
        raise Unavailable("search returned no result list")
    out = []
    for hit in hits:
        if isinstance(hit, dict) and hit.get("url") and hit.get("title"):
            out.append({"title": str(hit["title"]).strip(),
                        "url": str(hit["url"]), "host": host})
    return out


# --------------------------------------------------------------------------- #
# What the week is short of
# --------------------------------------------------------------------------- #

@dataclass
class Gap:
    """Something this week does not have, and the search that would fill it."""
    kind: str
    query: str
    why: str


def searchable(term: str) -> str:
    """Turn a corpus label into something a recipe site would recognise.

    The corpus writes cuisine the way the household thinks about it, hedges and
    all - `Chinese-ish`, `Japanese-ish`, `Italian-American`. Those are honest
    labels and useless queries: searching nine sites for `chinese-ish` returns
    nothing anywhere, which is how this was found. The hedge is dropped for the
    search and kept everywhere else, because the label is right about the recipe
    and only wrong about the search box.
    """
    term = term.strip().lower()
    # Only the hyphenated hedge. Without the hyphen this eats the `ish` off
    # `fish`, turns the query into `f`, and searches nine recipe sites for a
    # single letter — which is exactly what it did.
    term = re.sub(r"-ish$", "", term)
    term = term.split("-american")[0]
    return term.replace("-", " ").strip()


def gaps(meals: list, corpus: list[dict] | None = None) -> list[Gap]:
    """Read the week for what is missing, best-first.

    Deterministic, and computed off the same fields the ranker scores on, so
    acquisition and retrieval are answering the same question about the same
    week rather than two different ones.

    **Protein first, then effort, then cuisine.** That order is the profile's:
    protein variety is what it names as good planning, the active-time ceiling
    is a hard constraint, and cuisine is recorded as a *fact about the corpus*
    that nobody has yet called a problem - so widening it is offered last and
    never on its own initiative.
    """
    corpus = corpus if corpus is not None else pantry.load_corpus()
    have_protein = {(m.protein or "").lower() for m in meals if m.protein}
    have_cuisine = {(m.cuisine or "").lower() for m in meals if m.cuisine}
    known_protein = {(r.get("protein") or "").lower() for r in corpus if r.get("protein")}

    out: list[Gap] = []
    # A protein the household cooks and this week does not have. Read off the
    # corpus rather than a list of proteins I think exist, so it can never ask
    # for something they have never once bought.
    for protein in sorted(known_protein - have_protein):
        out.append(Gap("protein", searchable(protein),
                       f"no {protein} in the week, and it is in the corpus"))

    low = sum(1 for m in meals if pantry.ACTIVE_RANK.get(m.active, 1) == 0)
    if low < max(2, len(meals) // 2):
        out.append(Gap("effort", "slow cooker",
                       f"only {low} low-active night(s); hard nights here are "
                       f"unpredictable, so the week needs more it cannot break"))

    # **A well-covered week is the normal case, not a reason to stop.** The
    # planner prompt is explicit that at this corpus size acquisition is part of
    # the job every week and not an occasional flourish - a corpus of 24 cannot
    # fill a year of weeks, so a week that happens to want nothing is still a
    # week the corpus needs widening in.
    #
    # Widened on **protein**, at the thinnest one the household already cooks.
    # Not on cuisine: `profile.md` records the narrowness as a measured fact and
    # then says plainly that nobody has been asked whether they want it widened,
    # and acting on an unanswered question is the exact failure this profile is
    # written to prevent.
    if not out:
        counts = {p: sum(1 for r in corpus if (r.get("protein") or "").lower() == p)
                  for p in known_protein}
        if counts:
            thinnest = min(sorted(counts), key=lambda p: counts[p])
            out.append(Gap("breadth", searchable(thinnest),
                           f"the week is covered, but {thinnest} is the thinnest "
                           f"protein in the corpus at {counts[thinnest]} recipe(s)"))

    for cuisine in sorted({(r.get("cuisine") or "").lower()
                           for r in corpus if r.get("cuisine")} - have_cuisine):
        out.append(Gap("cuisine", searchable(cuisine), f"no {cuisine} in the week"))
    return out


# --------------------------------------------------------------------------- #
# Judging one page
# --------------------------------------------------------------------------- #

# From `profile.md`: *"Cooks with shortcuts on purpose. Seasoning packets, canned
# soup, refrigerated biscuits, blocks of cream cheese recur across the corpus. A
# scratch-everything proposal is wrong for this household regardless of how good
# the recipe is."* Traced to the ingredient lists of six typed-out recipes, so it
# is a measured claim and this is it applied rather than restated.
SHORTCUT_MARKERS = [
    "seasoning packet", "packet", "soup mix", "canned", "can of", "cream of",
    "refrigerated", "biscuit", "cream cheese", "frozen", "jarred", "jar of",
    "rotisserie", "boxed", "mix",
]

# The weeknight ceiling is 20-30 minutes *active*. Sources state prep time, which
# is the closest honest proxy for hands-on and is what the household's own corpus
# ratings were guessed from anyway - except those were guessed from nothing and
# this is read off the page. Marked inferred either way.
PREP_TO_ACTIVE = [(15, "low"), (35, "med")]


def active_from(rec: dict) -> tuple[str, str]:
    """`(active, basis)` from the source's stated prep time, or `("", "")`.

    Returns empty rather than guessing when the source is silent. Every effort
    rating in `corpus.md` is the system's unverified guess and `profile.md` says
    so; adding another guess with no page behind it would make that worse, and
    the ranker already treats an unknown active as `med` rather than as an
    extreme.
    """
    for entry in rec.get("times") or []:
        m = re.match(r"prep\s+(\d+)\s*min", entry.strip(), re.I)
        if not m:
            continue
        mins = int(m.group(1))
        for ceiling, rank in PREP_TO_ACTIVE:
            if mins <= ceiling:
                return rank, f"source says prep {mins} min"
        return "high", f"source says prep {mins} min"
    return "", ""


@dataclass
class Verdict:
    """Why a page is or is not worth proposing. `ok` is the only gate; the rest
    is what gets shown to a person who wants to disagree with it."""
    ok: bool
    refusals: list[str] = field(default_factory=list)
    fits: list[str] = field(default_factory=list)
    score: float = 0.0
    active: str = ""
    active_basis: str = ""
    protein: str = ""
    cuisine: str = ""


def assess(rec: dict, gap: Gap | None = None, known: set[str] | None = None) -> Verdict:
    """Judge one captured page against the profile. Deterministic.

    Refusals are hard and each one is a rule from `profile.md` or from this
    project's own list of what it has already got wrong. Fits are soft and only
    order the shortlist.
    """
    known = known or set()
    v = Verdict(ok=False)
    title = rec.get("title") or ""
    sl = pantry.slug(title)

    # A capture with no ingredients is not a recipe. `onboard.from_url` refuses
    # to guess from page prose, so this is that refusal arriving here as a fact
    # rather than being second-guessed.
    if rec.get("status") != "complete" or not rec.get("ingredients"):
        v.refusals.append("the page carries no machine-readable recipe")
        return v
    if sl in known:
        v.refusals.append("already in the corpus or the candidates")
        return v

    verdict, evidence = onboard.scan_peanut(rec["ingredients"])
    if verdict == "CONTAINS PEANUT":
        v.refusals.append(f"contains peanut ({evidence[0]}) — a hard constraint")
        return v

    v.protein = onboard.infer_protein(rec["ingredients"])[0] or ""
    v.cuisine = onboard.infer_cuisine(rec["ingredients"])[0] or ""
    v.active, v.active_basis = active_from(rec)

    # **The search is full-text, so it returns cake.** Asked for `fish`, the nine
    # sources came back with blueberry muffins, a coffee cake, a fruit salad and
    # a clafoutis - every one of which passed the hard constraints, because
    # nothing about a muffin is unsafe. It is just not dinner.
    #
    # The corpus is mains-only and this is a dinner planner, so a capture with no
    # identifiable protein is refused. That is a blunt instrument and it is worth
    # saying what it costs: a genuinely vegetarian main would be refused too. The
    # profile already records that the corpus is mains-only *and* that vegetables
    # are a recording artifact rather than an absence - so a vegetarian dinner is
    # a real gap here, and it is `docs/brief-next.md` §6's to close, not
    # something to paper over by letting dessert through in the meantime.
    if not v.protein:
        v.refusals.append("no identifiable protein — reads as a side or a sweet, "
                          "and the corpus is mains-only")
        return v

    # Relevance is a refusal, not a score. A beef taco is a fine recipe and a
    # wrong answer to "the week has no fish", and scoring it merely means it wins
    # whenever nothing better turned up - which is how a search for one thing
    # quietly lands another.
    if gap and gap.kind in ("protein", "breadth") and v.protein != gap.query:
        v.refusals.append(f"{v.protein or 'no protein'}, but the gap asked for "
                          f"{gap.query}")
        return v

    if v.active == "high":
        # Not a refusal: `profile.md` keeps one or two weekend nights open for
        # something longer and nicer. It just loses every head-to-head against a
        # weeknight-shaped recipe, which is what the score does.
        v.score -= 12
        v.fits.append("weekend-shaped — more hands-on than a weeknight allows")
    elif v.active == "low":
        v.score += 14
        v.fits.append(f"low active ({v.active_basis})")
    elif v.active == "med":
        v.score += 4

    if gap and gap.kind in ("protein", "breadth"):
        v.score += 20
        v.fits.append(f"{v.protein}, which the week is missing")
    if gap and gap.kind == "cuisine" and searchable(v.cuisine) == gap.query:
        v.score += 12
        v.fits.append(f"{v.cuisine}, which the week is missing")

    raw = " ".join(i["raw"].lower() for i in rec["ingredients"])
    shortcuts = sorted({m for m in SHORTCUT_MARKERS if m in raw})
    if shortcuts:
        v.score += 8
        v.fits.append(f"uses the shortcuts this household cooks with ({shortcuts[0]})")

    if rec.get("yield"):
        v.score += 4
    else:
        # Not disqualifying, but it is the question `onboard` asks every time and
        # the planner needs it to reason about leftovers.
        v.fits.append("yield not stated by the source — someone will have to answer that")

    if verdict == "check label":
        # Flag, never filter. The profile is explicit that trace risk is
        # acceptable and that this filters the recipe rather than the pantry.
        v.fits.append(f"check the label on {evidence[0]} — bought sauces can carry peanut")

    v.ok = True
    return v


# --------------------------------------------------------------------------- #
# The run
# --------------------------------------------------------------------------- #

@dataclass
class Found:
    rec: dict
    verdict: Verdict
    gap: Gap
    hit: dict

    def reason(self) -> str:
        """The reason for the reach, in one sentence.

        Traceable by construction: the gap comes off this week, the fits come off
        the captured ingredients, and neither is a sentence about this household
        that nothing supports. A candidate has never been cooked here and can make
        no claim about it - so the reason is about the shape of the week and the
        content of the page, which is the only honest ground it has.
        """
        why = self.verdict.fits[0] if self.verdict.fits else "fills a gap in the week"
        return f"new here — {self.gap.why}; {why}"


def look(gap: Gap, hosts: list[str], known: set[str], budget: list[int],
         log=lambda *_: None) -> list[Found]:
    """Search every source for one gap, capture what comes back, judge it."""
    out: list[Found] = []
    for host in hosts:
        if budget[0] <= 0:
            break
        try:
            hits = search(gap.query, host)
        except Unavailable as exc:
            log(f"    {host}: no search API ({exc})")
            continue
        if not hits:
            # Worth a line. A silent run that searched nine sites and found
            # nothing looks identical to one that never searched at all, and the
            # first is a query problem while the second is a bug.
            log(f"    {host}: no results for {gap.query!r}")
        time.sleep(PAUSE)
        for hit in hits:
            if budget[0] <= 0:
                break
            if pantry.slug(hit["title"]) in known:
                continue
            budget[0] -= 1
            rec = onboard.from_url(hit["url"])
            time.sleep(PAUSE)
            verdict = assess(rec, gap, known)
            state = "ok" if verdict.ok else f"no — {verdict.refusals[0]}"
            log(f"    {hit['title'][:52]:54} {state}")
            if verdict.ok:
                out.append(Found(rec, verdict, gap, hit))
    return out


def acquire(week_meals: list, want: int = 1, gap_filter: str | None = None,
            dry_run: bool = False, log=lambda *_: None) -> list[Found]:
    """Fill this week's gaps with recipes nobody had bookmarked.

    Writes through `pantry.add_candidate`, which is the only door into
    `candidates.md` and which refuses a candidate with no source. Returns what
    was landed, best first.
    """
    hosts = sources()
    if not hosts:
        log("no sources: the corpus names no domains to search")
        return []
    known = {r["slug"] for r in pantry.load_corpus()} | \
            {r["slug"] for r in pantry.load_candidates()}

    wanted = gaps(week_meals)
    if gap_filter:
        wanted = [g for g in wanted if g.query == gap_filter.lower()] or \
                 [Gap("protein", gap_filter.lower(), f"asked for {gap_filter}")]
    if not wanted:
        log("no gaps: this week is not short of anything the corpus knows about")
        return []

    log(f"searching {len(hosts)} source(s) the household already cooks from")
    budget = [MAX_FETCHES]
    landed: list[Found] = []
    for gap in wanted:
        if len(landed) >= want or budget[0] <= 0:
            break
        log(f"\n  gap: {gap.why}\n  query: {gap.query!r}")
        found = look(gap, hosts, known, budget, log)
        found.sort(key=lambda f: -f.verdict.score)
        for cand in found:
            if len(landed) >= want:
                break
            title = cand.rec["title"]
            if dry_run:
                log(f"  would add: {title}")
                landed.append(cand)
                known.add(pantry.slug(title))
                continue
            cand.rec["slug"] = pantry.slug(title)
            (pantry.ROOT / "recipes").mkdir(parents=True, exist_ok=True)
            pantry.recipe_file(cand.rec["slug"]).write_text(
                onboard.render_recipe(cand.rec), encoding="utf-8")
            pantry._FILE_INDEX = None
            added = pantry.add_candidate(
                title, source=cand.rec["source"], protein=cand.verdict.protein,
                cuisine=cand.verdict.cuisine, yield_=cand.rec.get("yield") or "unknown",
                active=cand.verdict.active, passive=cand.rec.get("passive") or "—",
                proposed=f"wk of {pantry.monday()}", found_by="acquire")
            if added:
                log(f"  added: {title}  ({cand.rec['source']})")
                landed.append(cand)
                known.add(pantry.slug(title))
    return landed


def main():
    p = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    p.add_argument("--protein", help="go after one thing rather than this week's gaps")
    p.add_argument("--want", type=int, default=1, help="how many to land (default: 1)")
    p.add_argument("--dry-run", action="store_true",
                   help="search, capture and judge, but write nothing")
    p.add_argument("--sources", action="store_true",
                   help="list the domains that would be searched, and stop")
    args = p.parse_args()

    if args.sources:
        for host in sources():
            print(f"  {host}")
        return 0

    week = pantry.read_week(pantry.monday())
    meals = week.meals if week else []
    if not meals:
        print("note: no week planned yet — searching against an empty week, so every "
              "protein in the corpus reads as a gap\n", file=sys.stderr)

    found = acquire(meals, want=args.want, gap_filter=args.protein,
                    dry_run=args.dry_run, log=lambda s="": print(s, file=sys.stderr))
    if not found:
        print("\nnothing landed.", file=sys.stderr)
        return 1
    print()
    for f in found:
        print(f"{f.rec['title']}")
        print(f"  source:  {f.rec['source']}")
        print(f"  reason:  {f.reason()}")
        print(f"  capture: {len(f.rec['ingredients'])} ingredients, "
              f"yield {f.rec.get('yield') or 'unknown'}, "
              f"active {f.verdict.active or 'not stated'}")
        for question in f.rec.get("questions", []):
            print(f"  ask:     {question}")
    if args.dry_run:
        print("\n(dry run — nothing was written)", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())
