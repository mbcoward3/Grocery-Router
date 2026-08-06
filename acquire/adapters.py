"""How to search a recipe site, one strategy per class.

Acquisition started against `/wp-json/wp/v2/search`, which nine of the household's
eleven sources happen to expose. That was the right first move and the wrong
shape to stop at: it made "can this site be searched" a property of the site's CMS
rather than of this code, and it left `tasty.co` permanently unreachable because
it is not WordPress.

**So searching is an interface with several implementations, tried in order, and
the first one that answers for a host is remembered.** Adding a site that needs
something new is a class here, not an edit to the pipeline.

The order is not arbitrary. It runs from *the site told us how* to *we worked it
out*, and it stops at the first thing that works:

1. `WordPressSearch` - the documented REST search route.
2. `WordPressPosts` - the same REST API where only the posts route is exposed.
3. `SearchAction` - the site's own `schema.org/SearchAction`, published in its
   homepage JSON-LD specifically so that programs know how to search it.
4. `HtmlSearch` - the site's search page, read as HTML.
5. `Sitemap` - no search at all; match the query against the URLs the site
   publishes for crawlers.

**Only the last two are scraping, and both are fenced.** `robots.txt` is checked
before any request and a disallowed path is not fetched - that is stdlib
`urllib.robotparser`, not a promise in a comment. Requests to one host are spaced
out. Nothing here parses a recipe: an adapter's whole job is to *propose URLs*,
and `onboard.from_url` then refuses anything without machine-readable
schema.org recipe data. So a sloppy scrape costs a wasted fetch and can never
cost a wrong ingredient - which is what makes the loose strategies safe to have
at all.

Standard library only.
"""

from __future__ import annotations

import json
import re
import time
import urllib.error
import urllib.parse
import urllib.request
import urllib.robotparser
import xml.etree.ElementTree as ET
from html.parser import HTMLParser

USER_AGENT = "Mozilla/5.0 (compatible; pantry-router/0.1; +household meal planner)"
TIMEOUT = 15
# Per host, not global. Two sites can be searched back to back; one site is not
# hit twice in a second.
COURTESY = 1.0

_last_hit: dict[str, float] = {}
_robots: dict[str, urllib.robotparser.RobotFileParser | None] = {}
_chosen: dict[str, "Adapter | None"] = {}


class Unavailable(Exception):
    """This adapter cannot answer for this host. The next one is tried."""


class Disallowed(Unavailable):
    """`robots.txt` says no. **Not** an error to route around."""


# --------------------------------------------------------------------------- #
# Fetching, politely
# --------------------------------------------------------------------------- #

def robots(host: str):
    """The host's `robots.txt`, parsed and cached. `None` if it has none.

    A host that does not publish one has not refused anything, so a missing file
    is permission - that is what the standard says and inventing a stricter rule
    would just make the tool useless against half the web. A host that publishes
    one is obeyed.
    """
    if host in _robots:
        return _robots[host]
    parser = urllib.robotparser.RobotFileParser()
    parser.set_url(f"https://{host}/robots.txt")
    # **Fetched here rather than by `parser.read()`, and the difference is not
    # cosmetic.** `read()` calls `urlopen` with Python's default user agent,
    # which most of these sites' CDNs answer with a 403 - and CPython reads a 403
    # on `robots.txt` as *disallow everything*. Nine sources that had been
    # working went unreachable in one commit because of it, all of them
    # incorrectly: their `robots.txt` allows what is being asked for, and none of
    # them had said no to anything.
    try:
        req = urllib.request.Request(f"https://{host}/robots.txt",
                                     headers={"User-Agent": USER_AGENT})
        with urllib.request.urlopen(req, timeout=TIMEOUT) as resp:
            parser.parse(resp.read().decode("utf-8", "replace").splitlines())
    except urllib.error.HTTPError as e:
        # A real 401 or 403 on the file itself is still a refusal - it just has
        # to be one the site made to *us*, asked properly, rather than one it
        # made to a user agent we are not.
        if e.code in (401, 403):
            parser.disallow_all = True
        else:
            parser = None
    except Exception:
        parser = None
    _robots[host] = parser
    return parser


def allowed(url: str) -> bool:
    host = urllib.parse.urlparse(url).netloc
    parser = robots(host)
    if parser is None:
        return True
    try:
        return parser.can_fetch(USER_AGENT, url)
    except Exception:
        return True


def get(url: str, timeout: int = TIMEOUT) -> str:
    """One HTTP GET, after `robots.txt` and after waiting our turn."""
    if not allowed(url):
        raise Disallowed(f"robots.txt disallows {url}")
    host = urllib.parse.urlparse(url).netloc
    wait = COURTESY - (time.monotonic() - _last_hit.get(host, 0.0))
    if wait > 0:
        time.sleep(wait)
    _last_hit[host] = time.monotonic()
    req = urllib.request.Request(url, headers={"User-Agent": USER_AGENT})
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as e:
        raise Unavailable(f"HTTP {e.code}") from e
    except (urllib.error.URLError, TimeoutError, OSError) as e:
        raise Unavailable(str(getattr(e, "reason", e))) from e


# --------------------------------------------------------------------------- #
# What a hit looks like, and what is obviously not one
# --------------------------------------------------------------------------- #

# Paths that are never a recipe. Cheap to exclude and they otherwise dominate
# any link-scraping result: a listing page has one link to each recipe and forty
# to the site's own furniture.
NOT_A_RECIPE = re.compile(
    r"/(category|categories|tag|tags|author|page|about|contact|privacy|terms|"
    r"shop|store|subscribe|newsletter|search|feed|comment|wp-|cdn-cgi|"
    r"cookbook|web-stories|amp)(/|$)", re.I)

RECIPE_HINT = re.compile(r"/(recipe|recipes)/", re.I)


def plausible(url: str, host: str) -> bool:
    """Could this URL be a recipe page on this host?

    Deliberately permissive. Everything that survives is fetched and handed to
    `onboard.from_url`, which refuses anything without schema.org recipe data -
    so the cost of a false positive here is one wasted request, and the cost of
    a false negative is a recipe the household never sees. Guessing loose and
    verifying hard is the right way round.
    """
    parts = urllib.parse.urlparse(url)
    if parts.scheme not in ("http", "https"):
        return False
    if parts.netloc.removeprefix("www.") != host.removeprefix("www."):
        return False
    path = parts.path
    if not path or path == "/":
        return False
    if NOT_A_RECIPE.search(path):
        return False
    return path.count("/") <= 4


def hit(title: str, url: str, host: str) -> dict:
    return {"title": re.sub(r"\s+", " ", title).strip(), "url": url, "host": host}


# --------------------------------------------------------------------------- #
# The strategies
# --------------------------------------------------------------------------- #

class Adapter:
    """One way to ask a site what it has. Subclasses implement `search`."""

    name = "adapter"
    scrapes = False

    def search(self, query: str, host: str, limit: int) -> list[dict]:
        raise NotImplementedError

    def __repr__(self):
        return f"<{self.name}>"


class WordPressSearch(Adapter):
    """`/wp-json/wp/v2/search` — the documented REST search route.

    Nine of the household's eleven sources answer here. It returns real posts
    with real permalinks and cannot return a page that does not exist, which is
    why it stays first even though the later adapters are more general.
    """

    name = "wp-search"

    def search(self, query, host, limit):
        url = (f"https://{host}/wp-json/wp/v2/search?"
               + urllib.parse.urlencode({"search": query, "per_page": limit,
                                         "subtype": "post"}))
        try:
            data = json.loads(get(url))
        except json.JSONDecodeError as e:
            raise Unavailable(f"not JSON: {e}") from e
        if not isinstance(data, list):
            raise Unavailable("no result list")
        return [hit(str(h.get("title", "")), str(h["url"]), host)
                for h in data if isinstance(h, dict) and h.get("url")]


class WordPressPosts(Adapter):
    """`/wp-json/wp/v2/posts?search=` — the same API with the search route off.

    A real configuration, not a hypothetical: the search route is a separate
    controller and plenty of sites disable it while leaving posts readable.
    """

    name = "wp-posts"

    def search(self, query, host, limit):
        url = (f"https://{host}/wp-json/wp/v2/posts?"
               + urllib.parse.urlencode({"search": query, "per_page": limit,
                                         "_fields": "link,title"}))
        try:
            data = json.loads(get(url))
        except json.JSONDecodeError as e:
            raise Unavailable(f"not JSON: {e}") from e
        if not isinstance(data, list):
            raise Unavailable("no result list")
        out = []
        for post in data:
            if not isinstance(post, dict) or not post.get("link"):
                continue
            title = post.get("title")
            if isinstance(title, dict):
                title = title.get("rendered", "")
            out.append(hit(str(title or post["link"]), str(post["link"]), host))
        return out


class _LinkReader(HTMLParser):
    """Collect `(href, text)` pairs. Enough HTML parsing for a results page and
    no more - the recipe itself is never read this way."""

    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.links: list[tuple[str, str]] = []
        self._href = None
        self._text: list[str] = []
        self._jsonld: list[str] = []
        self._in_ld = False

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if tag == "a" and attrs.get("href"):
            self._href, self._text = attrs["href"], []
        elif tag == "script" and "ld+json" in (attrs.get("type") or ""):
            self._in_ld = True

    def handle_endtag(self, tag):
        if tag == "a" and self._href is not None:
            self.links.append((self._href, "".join(self._text)))
            self._href, self._text = None, []
        elif tag == "script":
            self._in_ld = False

    def handle_data(self, data):
        if self._href is not None:
            self._text.append(data)
        if self._in_ld:
            self._jsonld.append(data)


class SearchAction(Adapter):
    """The site's own `schema.org/SearchAction`, read off its homepage.

    A site that publishes a `SearchAction` has stated, in machine-readable form,
    how it wants to be searched. Using it is the opposite of guessing - it is the
    same posture as reading a recipe out of JSON-LD instead of off the page, and
    it is why this sits above the two scraping strategies rather than beside them.
    """

    name = "search-action"
    scrapes = True

    def template(self, host: str) -> str:
        reader = _LinkReader()
        reader.feed(get(f"https://{host}/"))
        for blob in reader._jsonld:
            try:
                data = json.loads(blob)
            except json.JSONDecodeError:
                continue
            for node in data if isinstance(data, list) else [data]:
                action = (node or {}).get("potentialAction") if isinstance(node, dict) else None
                for act in (action if isinstance(action, list) else [action]):
                    if not isinstance(act, dict) or "SearchAction" not in str(act.get("@type")):
                        continue
                    target = act.get("target")
                    if isinstance(target, dict):
                        target = target.get("urlTemplate")
                    if isinstance(target, str) and "{" in target:
                        return target
        raise Unavailable("no SearchAction published")

    def search(self, query, host, limit):
        target = self.template(host)
        url = re.sub(r"\{[^}]*\}", urllib.parse.quote_plus(query), target, count=1)
        return _links_from(get(url), host, limit)


class HtmlSearch(Adapter):
    """The site's search page, read as HTML.

    Genuine scraping, and the point at which this stops being something the site
    published for us. It is fenced by `robots.txt` and by the courtesy delay, it
    reads links and nothing else, and every link it proposes still has to survive
    `onboard.from_url`. `?s=` is WordPress's own convention and covers most of
    what is left after the REST adapters.
    """

    name = "html-search"
    scrapes = True
    PATHS = ["/?s={q}", "/search?q={q}", "/search/{q}"]

    def search(self, query, host, limit):
        last = None
        for path in self.PATHS:
            url = f"https://{host}" + path.format(q=urllib.parse.quote_plus(query))
            try:
                found = _links_from(get(url), host, limit)
            except Unavailable as exc:
                last = exc
                continue
            if found:
                return found
        if last:
            raise last
        return []


class Sitemap(Adapter):
    """No search at all: match the query against the URLs the site publishes.

    The last resort, and the one that works on a site with no search of any kind.
    A sitemap exists so crawlers can find pages, so reading one is the most
    clearly-invited request in this file - but it matches on the URL slug alone,
    which is why it sits at the bottom rather than the top.
    """

    name = "sitemap"
    scrapes = True
    INDEXES = ["/sitemap.xml", "/sitemap_index.xml", "/wp-sitemap.xml",
               "/post-sitemap.xml"]

    def urls(self, host: str, depth: int = 0) -> list[str]:
        for path in (self.INDEXES if depth == 0 else []):
            try:
                return self._read(f"https://{host}{path}", host)
            except Unavailable:
                continue
        raise Unavailable("no sitemap")

    def _read(self, url: str, host: str, depth: int = 0) -> list[str]:
        try:
            root = ET.fromstring(get(url))
        except ET.ParseError as e:
            raise Unavailable(f"sitemap did not parse: {e}") from e
        locs = [e.text.strip() for e in root.iter() if
                e.tag.endswith("loc") and e.text]
        # A sitemap index points at more sitemaps. One level down only: these
        # files run to tens of thousands of URLs and this is a recipe lookup,
        # not a crawl.
        if depth == 0 and locs and all(l.endswith(".xml") for l in locs[:3]):
            out: list[str] = []
            for child in locs[:4]:
                try:
                    out += self._read(child, host, depth + 1)
                except Unavailable:
                    continue
            return out
        return locs

    def search(self, query, host, limit):
        terms = [t for t in re.split(r"\W+", query.lower()) if len(t) > 2]
        if not terms:
            raise Unavailable("query too short to match a slug")
        out = []
        for url in self.urls(host):
            if not plausible(url, host):
                continue
            slug = urllib.parse.urlparse(url).path.lower()
            if all(t in slug for t in terms):
                out.append(hit(slug.strip("/").split("/")[-1].replace("-", " "),
                               url, host))
                if len(out) >= limit:
                    break
        return out


def _links_from(html: str, host: str, limit: int) -> list[dict]:
    reader = _LinkReader()
    reader.feed(html)
    seen, out = set(), []
    for href, text in reader.links:
        url = urllib.parse.urljoin(f"https://{host}/", href).split("#")[0]
        if url in seen or not plausible(url, host):
            continue
        seen.add(url)
        out.append(hit(text or url, url, host))
        if len(out) >= limit * 3:      # trimmed by the caller after ranking
            break
    # A results page links each recipe once and its own furniture many times.
    # Anything under /recipe/ is a stronger signal than anything else here.
    out.sort(key=lambda h: (0 if RECIPE_HINT.search(h["url"]) else 1))
    return out[:limit]


# --------------------------------------------------------------------------- #
# Choosing one
# --------------------------------------------------------------------------- #

ADAPTERS: list[Adapter] = [WordPressSearch(), WordPressPosts(), SearchAction(),
                           HtmlSearch(), Sitemap()]


def enabled(allow_scraping: bool = True) -> list[Adapter]:
    return [a for a in ADAPTERS if allow_scraping or not a.scrapes]


def search(query: str, host: str, limit: int = 5, allow_scraping: bool = True,
           log=lambda *_: None) -> list[dict]:
    """Search one host with whichever strategy answers, remembering which did.

    The choice is cached per host for the life of the process, so a site that
    needed the sitemap once does not walk the whole ladder again on the next gap.
    A host where every adapter fails is remembered as `None` and skipped, which
    is what stops a dead source costing five requests every week.
    """
    if host in _chosen:
        chosen = _chosen[host]
        if chosen is None:
            raise Unavailable("no strategy works for this host")
        return chosen.search(query, host, limit)

    last: Exception = Unavailable("no adapters enabled")
    for adapter in enabled(allow_scraping):
        try:
            found = adapter.search(query, host, limit)
        except Unavailable as exc:
            last = exc
            log(f"      {host}: {adapter.name} — {exc}")
            continue
        # An adapter that answers but finds nothing has still answered. Caching
        # it is right: the query was the problem, not the strategy, and the next
        # query deserves the same route rather than a fresh walk down the ladder.
        _chosen[host] = adapter
        log(f"      {host}: via {adapter.name}, {len(found)} hit(s)")
        return found
    _chosen[host] = None
    raise last


def strategy(host: str) -> str:
    """Which adapter is being used for a host, for the CLI's `--sources`."""
    chosen = _chosen.get(host, "?")
    if chosen == "?":
        return "not probed"
    return chosen.name if chosen else "unreachable"


def forget():
    """Drop every cache. Tests, and anything that changes the source list."""
    _chosen.clear()
    _robots.clear()
    _last_hit.clear()
