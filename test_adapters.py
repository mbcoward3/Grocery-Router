#!/usr/bin/env python3
"""Tests for the search adapters.

    python3 test_adapters.py

**No network.** `adapters.get` is the one seam and every test replaces it with a
dict of canned responses, so the ladder, the caching, the `robots.txt` handling
and the link filtering are all exercised without a socket.

The fixtures are trimmed from what the household's real sources return. Three of
them — `southernbite.com`, `spendwithpennies.com`, `onceuponachef.com` — really
do `Disallow: /wp-json/`, which is how the fallback path came to exist.
"""

import json
import unittest

from acquire import adapters

ROBOTS_OPEN = "User-agent: *\nDisallow: /wp-admin/\n"
ROBOTS_NO_JSON = "User-agent: *\nDisallow: /wp-admin/\nDisallow: /wp-json/\n"
ROBOTS_CLOSED = "User-agent: *\nDisallow: /\n"

WP_SEARCH = json.dumps([
    {"title": "Crock Pot BBQ Pork Chops", "url": "https://x.test/crock-pot-bbq-pork-chops/"},
    {"title": "Baked Pork Chops", "url": "https://x.test/baked-pork-chops/"},
])
WP_POSTS = json.dumps([
    {"link": "https://x.test/pork-loin/", "title": {"rendered": "Pork Loin"}},
])
SEARCH_PAGE = """
<html><body>
  <a href="/category/dinner/">Dinner</a>
  <a href="/recipe/pork-chops/">Pork Chops</a>
  <a href="/pork-tenderloin/">Pork Tenderloin</a>
  <a href="/tag/pork/">pork</a>
  <a href="https://elsewhere.test/pork/">Somebody else</a>
  <a href="/privacy/">Privacy</a>
</body></html>
"""
HOMEPAGE_WITH_ACTION = """
<html><head><script type="application/ld+json">
{"@type":"WebSite","potentialAction":{"@type":"SearchAction",
 "target":{"urlTemplate":"https://x.test/find?query={search_term_string}"},
 "query-input":"required name=search_term_string"}}
</script></head><body></body></html>
"""
SITEMAP_INDEX = """<?xml version="1.0"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap><loc>https://x.test/post-sitemap1.xml</loc></sitemap>
</sitemapindex>"""
SITEMAP_POSTS = """<?xml version="1.0"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://x.test/pork-chops-in-gravy/</loc></url>
  <url><loc>https://x.test/beef-stew/</loc></url>
  <url><loc>https://x.test/category/pork/</loc></url>
</urlset>"""


class Wired(unittest.TestCase):
    """Replace the one HTTP call with a lookup table."""

    def setUp(self):
        adapters.forget()
        self._get = adapters.get
        self._courtesy = adapters.COURTESY
        adapters.COURTESY = 0
        self.addCleanup(setattr, adapters, "get", self._get)
        self.addCleanup(setattr, adapters, "COURTESY", self._courtesy)
        self.addCleanup(adapters.forget)
        self.fetched = []

    def wire(self, pages, robots_txt=ROBOTS_OPEN):
        def get(url, timeout=adapters.TIMEOUT):
            if not adapters.allowed(url):
                raise adapters.Disallowed(f"robots.txt disallows {url}")
            self.fetched.append(url)
            # Longest prefix wins. `https://x.test/` would otherwise shadow
            # every more specific URL under it, and the homepage would answer
            # for the search page.
            for prefix, body in sorted(pages.items(), key=lambda kv: -len(kv[0])):
                if url.startswith(prefix):
                    if isinstance(body, Exception):
                        raise body
                    return body
            raise adapters.Unavailable("HTTP 404")

        adapters.get = get
        # `robots` uses urllib directly rather than `get`, so it is stubbed
        # separately - which is the point: it must not be able to route around
        # its own permission check.
        parser = None
        if robots_txt is not None:
            import urllib.robotparser
            parser = urllib.robotparser.RobotFileParser()
            parser.parse(robots_txt.splitlines())
        adapters._robots["x.test"] = parser


# --------------------------------------------------------------------------- #

class TestTheLadder(Wired):
    def test_the_documented_route_is_tried_first(self):
        self.wire({"https://x.test/wp-json/wp/v2/search": WP_SEARCH})
        hits = adapters.search("pork chops", "x.test")
        self.assertEqual(len(hits), 2)
        self.assertEqual(adapters.strategy("x.test"), "wp-search")

    def test_it_falls_through_to_the_posts_route(self):
        """A real configuration: the search controller is separate and plenty of
        sites turn it off while leaving posts readable."""
        self.wire({"https://x.test/wp-json/wp/v2/posts": WP_POSTS})
        hits = adapters.search("pork", "x.test")
        self.assertEqual([h["title"] for h in hits], ["Pork Loin"])
        self.assertEqual(adapters.strategy("x.test"), "wp-posts")

    def test_it_uses_the_search_action_the_site_published(self):
        """A site that publishes a SearchAction has said in machine-readable form
        how it wants to be searched. That beats guessing."""
        self.wire({"https://x.test/": HOMEPAGE_WITH_ACTION,
                   "https://x.test/find?query=pork": SEARCH_PAGE})
        hits = adapters.search("pork", "x.test")
        self.assertEqual(adapters.strategy("x.test"), "search-action")
        self.assertTrue(hits)

    def test_it_scrapes_the_search_page_when_nothing_else_answers(self):
        self.wire({"https://x.test/?s=pork": SEARCH_PAGE})
        hits = adapters.search("pork", "x.test")
        self.assertEqual(adapters.strategy("x.test"), "html-search")
        self.assertTrue(any("pork-chops" in h["url"] for h in hits))

    def test_the_sitemap_is_the_last_resort(self):
        self.wire({"https://x.test/sitemap.xml": SITEMAP_INDEX,
                   "https://x.test/post-sitemap1.xml": SITEMAP_POSTS})
        hits = adapters.search("pork", "x.test")
        self.assertEqual(adapters.strategy("x.test"), "sitemap")
        self.assertEqual([h["url"] for h in hits],
                         ["https://x.test/pork-chops-in-gravy/"])

    def test_a_host_where_nothing_answers_is_remembered_as_dead(self):
        """So a source that has gone away does not cost five requests a week."""
        self.wire({})
        with self.assertRaises(adapters.Unavailable):
            adapters.search("pork", "x.test")
        self.assertEqual(adapters.strategy("x.test"), "unreachable")
        before = len(self.fetched)
        with self.assertRaises(adapters.Unavailable):
            adapters.search("beef", "x.test")
        self.assertEqual(len(self.fetched), before)

    def test_the_working_strategy_is_remembered(self):
        self.wire({"https://x.test/wp-json/wp/v2/search": WP_SEARCH})
        adapters.search("pork", "x.test")
        first = len(self.fetched)
        adapters.search("beef", "x.test")
        self.assertEqual(len(self.fetched), first + 1)   # one call, no re-probe

    def test_an_adapter_that_answers_with_nothing_still_counts_as_answering(self):
        """The query was the problem, not the strategy. Walking the whole ladder
        again would be four wasted requests per empty search."""
        self.wire({"https://x.test/wp-json/wp/v2/search": "[]"})
        self.assertEqual(adapters.search("kangaroo", "x.test"), [])
        self.assertEqual(adapters.strategy("x.test"), "wp-search")


class TestRobots(Wired):
    def test_a_disallowed_path_is_not_fetched(self):
        """Three of the household's real sources Disallow: /wp-json/. The first
        implementation hit it anyway."""
        self.wire({"https://x.test/wp-json/wp/v2/search": WP_SEARCH,
                   "https://x.test/?s=pork": SEARCH_PAGE},
                  robots_txt=ROBOTS_NO_JSON)
        adapters.search("pork", "x.test")
        self.assertEqual(adapters.strategy("x.test"), "html-search")
        self.assertFalse(any("wp-json" in u for u in self.fetched))

    def test_a_site_that_refuses_everything_is_left_alone(self):
        self.wire({"https://x.test/wp-json/wp/v2/search": WP_SEARCH},
                  robots_txt=ROBOTS_CLOSED)
        with self.assertRaises(adapters.Unavailable):
            adapters.search("pork", "x.test")
        self.assertEqual(self.fetched, [])

    def test_no_robots_file_is_permission(self):
        """What the standard says. Inventing a stricter rule would make the tool
        useless against half the web and would not protect anyone."""
        self.wire({"https://x.test/wp-json/wp/v2/search": WP_SEARCH},
                  robots_txt=None)
        self.assertEqual(len(adapters.search("pork", "x.test")), 2)


class TestScrapingIsOptional(Wired):
    def test_scraping_can_be_turned_off_in_one_place(self):
        self.wire({"https://x.test/?s=pork": SEARCH_PAGE})
        with self.assertRaises(adapters.Unavailable):
            adapters.search("pork", "x.test", allow_scraping=False)

    def test_the_published_routes_still_work_with_scraping_off(self):
        self.wire({"https://x.test/wp-json/wp/v2/search": WP_SEARCH})
        self.assertEqual(len(adapters.search("pork", "x.test",
                                             allow_scraping=False)), 2)


class TestWhatCountsAsALink(unittest.TestCase):
    """A results page links each recipe once and its own furniture many times."""

    def test_the_sites_own_furniture_is_excluded(self):
        for path in ("/category/dinner/", "/tag/pork/", "/page/2/", "/privacy/",
                     "/author/matt/", "/feed/", "/search/?s=x"):
            self.assertFalse(adapters.plausible(f"https://x.test{path}", "x.test"),
                             path)

    def test_another_site_is_never_proposed(self):
        self.assertFalse(adapters.plausible("https://elsewhere.test/pork/", "x.test"))

    def test_a_recipe_url_survives(self):
        self.assertTrue(adapters.plausible("https://x.test/pork-chops/", "x.test"))
        self.assertTrue(adapters.plausible("https://x.test/recipe/pork-chops/",
                                           "x.test"))

    def test_www_is_not_a_different_site(self):
        self.assertTrue(adapters.plausible("https://www.x.test/pork/", "x.test"))

    def test_a_recipe_path_outranks_a_bare_one(self):
        hits = adapters._links_from(SEARCH_PAGE, "x.test", 5)
        self.assertIn("/recipe/", hits[0]["url"])

    def test_it_is_permissive_on_purpose(self):
        """Everything surviving here is fetched and handed to `onboard.from_url`,
        which refuses anything with no schema.org recipe data. A false positive
        costs one request; a false negative costs a recipe nobody sees."""
        self.assertTrue(adapters.plausible("https://x.test/some-post/", "x.test"))


if __name__ == "__main__":
    unittest.main(verbosity=2)
