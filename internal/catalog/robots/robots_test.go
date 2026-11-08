package robots_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/catalog/robots"
)

const fixture = "\xef\xbb\xbf# comment\n" + `
User-agent: *
Disallow: /

User-agent: OtherBot
User-agent: groceryrouter/2.0
Disallow: /private
Allow: /private/ok
Disallow: /*.pdf$
Disallow: /search?
Crawl-delay: 7.5

Sitemap: https://www.a.test/sitemap.xml

User-agent: GroceryRouter
Disallow: /drafts/
`

func policyFor(t *testing.T, body string) robots.Policy {
	t.Helper()
	rules, delay := robots.Parse([]byte(body), "GroceryRouter")
	return robots.Policy{Host: "www.a.test", Mode: robots.ModeRules, Rules: rules, CrawlDelay: delay}
}

func allowed(t *testing.T, policy robots.Policy, raw string) bool {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return policy.Allowed(parsed)
}

func TestParseUsesOurGroupsAndCrawlDelay(t *testing.T) {
	policy := policyFor(t, fixture)
	if policy.CrawlDelay != 7500*time.Millisecond {
		t.Errorf("crawl delay = %v", policy.CrawlDelay)
	}
	for raw, want := range map[string]bool{
		"https://www.a.test/recipes/soup":    true, // "*" group is not ours
		"https://www.a.test/private/x":       false,
		"https://www.a.test/private/ok/x":    true, // longer allow wins
		"https://www.a.test/files/a.pdf":     false,
		"https://www.a.test/files/a.pdf?x=1": true, // $ anchors the end
		"https://www.a.test/search?q=soup":   false,
		"https://www.a.test/search":          true,
		"https://www.a.test/drafts/one":      false, // groups for our token combine
		"https://www.a.test/robots.txt":      true,
	} {
		if got := allowed(t, policy, raw); got != want {
			t.Errorf("%s allowed = %v, want %v", raw, got, want)
		}
	}
}

func TestParseFallsBackToStarGroup(t *testing.T) {
	policy := policyFor(t, "User-agent: *\nCrawl-delay: 10\nDisallow: /tmp\nAllow: /tmp\n")
	if policy.CrawlDelay != 10*time.Second {
		t.Errorf("crawl delay = %v", policy.CrawlDelay)
	}
	if !allowed(t, policy, "https://www.a.test/tmp/x") {
		t.Error("equal-length allow should win")
	}
	empty := policyFor(t, "User-agent: *\nDisallow:\n")
	if !allowed(t, empty, "https://www.a.test/anything") {
		t.Error("empty disallow should allow all")
	}
}

func TestModes(t *testing.T) {
	target, _ := url.Parse("https://www.a.test/x")
	if !(robots.Policy{Mode: robots.ModeAllowAll}).Allowed(target) {
		t.Error("allow_all refused")
	}
	if (robots.Policy{Mode: robots.ModeDisallowAll}).Allowed(target) {
		t.Error("disallow_all allowed")
	}
	if (robots.Policy{}).Allowed(target) {
		t.Error("zero policy allowed")
	}
}

func TestFresh(t *testing.T) {
	start := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	policy := robots.Policy{FetchedAt: start}
	if !policy.Fresh(start.Add(robots.TTL-time.Second)) || policy.Fresh(start.Add(robots.TTL)) {
		t.Error("freshness boundary wrong")
	}
}
