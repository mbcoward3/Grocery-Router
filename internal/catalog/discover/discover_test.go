package discover_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/catalog/acquire"
	"github.com/mbcoward3/grocery-router/internal/catalog/clock"
	"github.com/mbcoward3/grocery-router/internal/catalog/discover"
	"github.com/mbcoward3/grocery-router/internal/catalog/fetchtest"
	"github.com/mbcoward3/grocery-router/internal/catalog/ratelimit"
	"github.com/mbcoward3/grocery-router/internal/catalog/robots"
	"github.com/mbcoward3/grocery-router/internal/catalog/safefetch"
)

type fakeGetter struct {
	files    map[string]acquire.Result
	requests []string
}

func (getter *fakeGetter) GetSitemap(_ context.Context, _ acquire.Site, raw string) (acquire.Result, error) {
	getter.requests = append(getter.requests, raw)
	if result, ok := getter.files[raw]; ok {
		return result, nil
	}
	return acquire.Result{Outcome: acquire.OutcomeFailed, Err: errors.New("HTTP 404")}, nil
}

func fetched(body string) acquire.Result {
	return acquire.Result{Outcome: acquire.OutcomeFetched, Response: safefetch.Response{Body: []byte(body)}}
}

func index(locs ...string) string {
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?><sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for _, loc := range locs {
		fmt.Fprintf(&builder, "<sitemap><loc>%s</loc></sitemap>", loc)
	}
	builder.WriteString("</sitemapindex>")
	return builder.String()
}

func urlset(locs ...string) string {
	var builder strings.Builder
	builder.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for _, loc := range locs {
		fmt.Fprintf(&builder, "<url><loc>\n  %s\n</loc><lastmod>2026-01-02</lastmod></url>", loc)
	}
	builder.WriteString("</urlset>")
	return builder.String()
}

var baseSite = acquire.Site{ID: "a", Hosts: []string{"www.a.test", "cdn.a.test"}}

func TestDiscoverWalksIndexesWithLimitsAndFilters(t *testing.T) {
	filter, err := discover.NewPatternFilter([]string{`^/recipes/`}, []string{`/recipes/roundup-`})
	if err != nil {
		t.Fatal(err)
	}
	getter := &fakeGetter{files: map[string]acquire.Result{
		"https://www.a.test/sitemap.xml": fetched(index(
			"https://www.a.test/l2.xml",
			"https://WWW.a.test/sitemap.xml", // cycle back to the root
			"https://other.test/sitemap.xml", // unregistered host
		)),
		"https://www.a.test/l2.xml": fetched(index("https://cdn.a.test/l3.xml", "https://www.a.test/l2.xml")),
		"https://cdn.a.test/l3.xml": fetched(index("https://www.a.test/l4.xml") + ""),
		"https://www.a.test/posts.xml": fetched(urlset(
			"https://www.a.test/recipes/soup",
			"https://www.a.test/recipes/soup", // duplicate
			"https://cdn.a.test/recipes/stew",
			"https://www.a.test/recipes/roundup-best-soups",
			"https://www.a.test/about",
			"https://partner.test/recipes/pie",
			"http://www.a.test/recipes/insecure",
		)),
	}}
	site := discover.Site{Site: baseSite, Filter: filter, Sitemaps: []string{
		"https://www.a.test/sitemap.xml", "https://www.a.test/posts.xml", "https://www.a.test/sitemap.xml",
	}}
	report, err := discover.Discover(context.Background(), getter, site)
	if err != nil {
		t.Fatal(err)
	}
	var urls []string
	for _, entry := range report.Entries {
		urls = append(urls, entry.URL)
	}
	if strings.Join(urls, " ") != "https://www.a.test/recipes/soup https://cdn.a.test/recipes/stew" {
		t.Fatalf("entries = %v", urls)
	}
	if report.Entries[0].LastMod != "2026-01-02" || report.Entries[0].Sitemap != "https://www.a.test/posts.xml" {
		t.Errorf("entry = %+v", report.Entries[0])
	}
	if report.Cycles != 2 || report.TooDeep != 1 || report.UnregisteredHost != 2 || report.Excluded != 2 ||
		report.Duplicates != 1 || report.Invalid != 1 {
		t.Errorf("counts = %+v", report)
	}
	for _, request := range getter.requests {
		if strings.Contains(request, "l4") || strings.Contains(request, "other.test") {
			t.Errorf("fetched %s", request)
		}
	}
	if len(getter.requests) != 4 {
		t.Errorf("requests = %v", getter.requests)
	}
}

func TestDiscoverStopsWhenSitePauses(t *testing.T) {
	pause := &acquire.Pause{SiteID: "a", Reason: acquire.PauseTooManyRequests}
	getter := &fakeGetter{files: map[string]acquire.Result{
		"https://www.a.test/one.xml": {Outcome: acquire.OutcomePaused, Pause: pause, Err: errors.New("HTTP 429")},
		"https://www.a.test/two.xml": fetched(urlset("https://www.a.test/recipes/x")),
	}}
	report, err := discover.Discover(context.Background(), getter, discover.Site{
		Site: baseSite, Sitemaps: []string{"https://www.a.test/one.xml", "https://www.a.test/two.xml"},
	})
	if err != nil || report.Paused != pause || len(getter.requests) != 1 || len(report.Entries) != 0 {
		t.Fatalf("%v %+v %v", err, report, getter.requests)
	}
}

func TestParseLimits(t *testing.T) {
	locs := make([]string, discover.MaxURLs)
	for i := range locs {
		locs[i] = fmt.Sprintf("https://www.a.test/r/%d", i)
	}
	if parsed, err := discover.Parse([]byte(urlset(locs...))); err != nil || len(parsed.Locs) != discover.MaxURLs {
		t.Fatalf("at limit: %v %d", err, len(parsed.Locs))
	}
	_, err := discover.Parse([]byte(urlset(append(locs, "https://www.a.test/r/extra")...)))
	if !errors.Is(err, discover.ErrTooManyURLs) {
		t.Fatalf("over URL limit: %v", err)
	}

	// A small gzip file that decodes past 50 MiB is refused.
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, _ = writer.Write([]byte(`<urlset><url><loc>`))
	_, _ = writer.Write(bytes.Repeat([]byte("a"), discover.MaxDecodedBytes))
	_, _ = writer.Write([]byte(`</loc></url></urlset>`))
	_ = writer.Close()
	if _, err := discover.Parse(compressed.Bytes()); err == nil || !strings.Contains(err.Error(), "decoded bytes") {
		t.Fatalf("gzip bomb: %v", err)
	}

	for _, bad := range []string{"<html></html>", "not xml", ""} {
		if _, err := discover.Parse([]byte(bad)); err == nil {
			t.Errorf("Parse(%q) accepted", bad)
		}
	}
}

func TestDiscoverOverNetworkFetchesSitemapsOnly(t *testing.T) {
	mux := fetchtest.NewMux()
	network := fetchtest.New(t, mux, "www.a.test", "cdn.a.test")
	mux.Handle("www.a.test", "/robots.txt", fetchtest.Text(200, "text/plain", "User-agent: *\nAllow: /\n"))
	var gz bytes.Buffer
	writer := gzip.NewWriter(&gz)
	_, _ = writer.Write([]byte(urlset("https://www.a.test/recipes/soup")))
	_ = writer.Close()
	mux.Handle("www.a.test", "/sitemap.xml", fetchtest.Text(200, "application/xml", index("https://www.a.test/posts.xml.gz", "https://www.a.test/huge.xml")))
	mux.Handle("www.a.test", "/posts.xml.gz", fetchtest.Text(200, "application/gzip", gz.String()))
	mux.Handle("www.a.test", "/huge.xml", fetchtest.Text(200, "application/xml", strings.Repeat(" ", int(safefetch.SitemapProfile.MaxBodyBytes)+1)))
	mux.Handle("www.a.test", "/recipes/soup", fetchtest.Text(200, "text/html", `<a href="/recipes/linked">x</a>`))
	clk := clock.NewFake(time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC))
	client := &acquire.Client{
		Fetcher: network.Client(t), Limiter: ratelimit.NewMemory(clk), Robots: robots.NewMemoryCache(),
		Pauses: acquire.NewMemoryPauses(), Recorder: &acquire.MemoryRecorder{}, Clock: clk,
	}
	report, err := discover.Discover(context.Background(), client, discover.Site{Site: baseSite, Sitemaps: []string{"https://www.a.test/sitemap.xml"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Entries) != 1 || report.Entries[0].URL != "https://www.a.test/recipes/soup" {
		t.Fatalf("entries = %+v", report.Entries)
	}
	if mux.Hits("www.a.test", "/recipes/soup") != 0 || mux.Hits("www.a.test", "/recipes/linked") != 0 {
		t.Fatal("discovery fetched page bodies")
	}
	huge := report.Files[len(report.Files)-1]
	if huge.URL != "https://www.a.test/huge.xml" || huge.Outcome != acquire.OutcomeRejected {
		t.Fatalf("oversize sitemap = %+v", huge)
	}
}
