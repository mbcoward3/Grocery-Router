package acquire_test

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/catalog/acquire"
	"github.com/mbcoward3/grocery-router/internal/catalog/clock"
	"github.com/mbcoward3/grocery-router/internal/catalog/fetchtest"
	"github.com/mbcoward3/grocery-router/internal/catalog/ratelimit"
	"github.com/mbcoward3/grocery-router/internal/catalog/robots"
)

const (
	www = "www.a.test"
	cdn = "cdn.a.test"
)

var site = acquire.Site{ID: "a", Hosts: []string{www, cdn}}

type harness struct {
	client   *acquire.Client
	mux      *fetchtest.Mux
	network  *fetchtest.Network
	clock    *clock.Fake
	recorder *acquire.MemoryRecorder
	pauses   *acquire.MemoryPauses
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	mux := fetchtest.NewMux()
	network := fetchtest.New(t, mux, www, cdn)
	clk := clock.NewFake(time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC))
	h := &harness{
		mux: mux, network: network, clock: clk,
		recorder: &acquire.MemoryRecorder{}, pauses: acquire.NewMemoryPauses(),
	}
	h.client = &acquire.Client{
		Fetcher: network.Client(t), Limiter: ratelimit.NewMemory(clk), Robots: robots.NewMemoryCache(),
		Pauses: h.pauses, Recorder: h.recorder, Clock: clk,
	}
	return h
}

func (h *harness) robots(host string, status int, body string) {
	h.mux.Handle(host, "/robots.txt", fetchtest.Text(status, "text/plain", body))
}

func (h *harness) outcomes() []string {
	var out []string
	for _, attempt := range h.recorder.Attempts() {
		out = append(out, string(attempt.Kind)+":"+string(attempt.Outcome))
	}
	return out
}

const html = "text/html; charset=utf-8"

func TestGetPageObeysRobotsCrawlDelayAndCache(t *testing.T) {
	h := newHarness(t)
	h.robots(www, 200, "User-agent: GroceryRouter\nCrawl-delay: 8\nDisallow: /private\n")
	h.mux.Handle(www, "/soup", fetchtest.Text(200, html, "<html>soup</html>"))
	h.mux.Handle(www, "/stew", fetchtest.Text(200, html, "<html>stew</html>"))
	ctx := context.Background()

	result, err := h.client.GetPage(ctx, site, 1, "https://www.a.test/soup")
	if err != nil || result.Outcome != acquire.OutcomeFetched || string(result.Response.Body) != "<html>soup</html>" {
		t.Fatalf("soup: %v %+v", err, result)
	}
	if got := h.clock.Sleeps(); !slices.Equal(got, []time.Duration{8 * time.Second}) {
		t.Fatalf("sleeps = %v, want crawl-delay spacing after robots", got)
	}
	if _, err := h.client.GetPage(ctx, site, 1, "https://www.a.test/stew"); err != nil {
		t.Fatal(err)
	}
	if hits := h.mux.Hits(www, "/robots.txt"); hits != 1 {
		t.Fatalf("robots fetched %d times within TTL", hits)
	}
	result, _ = h.client.GetPage(ctx, site, 1, "https://www.a.test/private/x")
	if result.Outcome != acquire.OutcomeDisallowed || h.mux.Hits(www, "/private/x") != 0 {
		t.Fatalf("private: %+v", result.Outcome)
	}
	h.clock.Advance(robots.TTL)
	if _, err := h.client.GetPage(ctx, site, 1, "https://www.a.test/soup"); err != nil {
		t.Fatal(err)
	}
	if hits := h.mux.Hits(www, "/robots.txt"); hits != 2 {
		t.Fatalf("robots not refreshed after TTL: %d", hits)
	}
	want := []string{"robots:fetched", "page:fetched", "page:fetched", "robots:fetched", "page:fetched"}
	if got := h.outcomes(); !slices.Equal(got, want) {
		t.Fatalf("attempts = %v", got)
	}
}

func TestRobotsUnavailablePausesSite(t *testing.T) {
	for name, setup := range map[string]func(*harness){
		"429":     func(h *harness) { h.robots(www, 429, "") },
		"503":     func(h *harness) { h.robots(www, 503, "") },
		"network": func(h *harness) { h.network.Resolver.Set(www) },
	} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			setup(h)
			h.mux.Handle(www, "/soup", fetchtest.Text(200, html, "soup"))
			result, err := h.client.GetPage(context.Background(), site, 1, "https://www.a.test/soup")
			if err != nil || result.Outcome != acquire.OutcomePaused || result.Pause == nil ||
				result.Pause.Reason != acquire.PauseRobotsUnavailable {
				t.Fatalf("result = %v %+v", err, result)
			}
			if h.mux.Hits(www, "/soup") != 0 {
				t.Fatal("page fetched despite unavailable robots")
			}
			// The pause persists: later fetches make no request at all.
			robotsHits := h.mux.Hits(www, "/robots.txt")
			result, _ = h.client.GetSitemap(context.Background(), site, "https://www.a.test/sitemap.xml")
			if result.Outcome != acquire.OutcomePaused || h.mux.Hits(www, "/robots.txt") != robotsHits {
				t.Fatalf("paused site was fetched: %+v", result)
			}
			if _, err := h.pauses.Resume(context.Background(), site.ID, "test"); err != nil {
				t.Fatal(err)
			}
			h.robots(www, 200, "")
			h.network.Resolver.Set(www, fetchtest.PublicIP)
			result, _ = h.client.GetPage(context.Background(), site, 1, "https://www.a.test/soup")
			if result.Outcome != acquire.OutcomeFetched {
				t.Fatalf("after resume: %+v", result)
			}
		})
	}
}

func TestRobots4xxMeansUnrestricted(t *testing.T) {
	h := newHarness(t)
	h.robots(www, 404, "")
	h.mux.Handle(www, "/soup", fetchtest.Text(200, html, "soup"))
	result, err := h.client.GetPage(context.Background(), site, 1, "https://www.a.test/soup")
	if err != nil || result.Outcome != acquire.OutcomeFetched {
		t.Fatalf("%v %+v", err, result)
	}
}

func TestRobotsCheckedPerActualHostOnRedirect(t *testing.T) {
	h := newHarness(t)
	h.robots(www, 200, "")
	h.robots(cdn, 200, "User-agent: *\nDisallow: /\n")
	h.mux.Handle(www, "/soup", fetchtest.Redirect(301, "https://cdn.a.test/soup"))
	h.mux.Handle(cdn, "/soup", fetchtest.Text(200, html, "soup"))
	result, _ := h.client.GetPage(context.Background(), site, 1, "https://www.a.test/soup")
	if result.Outcome != acquire.OutcomeDisallowed || h.mux.Hits(cdn, "/robots.txt") != 1 || h.mux.Hits(cdn, "/soup") != 0 {
		t.Fatalf("result = %+v", result)
	}
}

func TestRetriesTimeoutsAnd5xxTwiceWithBackoff(t *testing.T) {
	h := newHarness(t)
	h.robots(www, 404, "")
	h.mux.Handle(www, "/down", fetchtest.Text(502, html, "bad gateway"))
	result, err := h.client.GetPage(context.Background(), site, 1, "https://www.a.test/down")
	if err != nil || result.Outcome != acquire.OutcomeFailed || result.Attempts != 3 || h.mux.Hits(www, "/down") != 3 {
		t.Fatalf("%v %+v hits=%d", err, result, h.mux.Hits(www, "/down"))
	}
	sleeps := h.clock.Sleeps()
	if !slices.Contains(sleeps, 10*time.Second) || !slices.Contains(sleeps, 30*time.Second) {
		t.Fatalf("backoff sleeps = %v", sleeps)
	}
	if paused, _, _ := h.pauses.ActivePause(context.Background(), site.ID); paused.SiteID != "" {
		t.Fatal("5xx paused the site")
	}

	calls := 0
	h.mux.Handle(www, "/flaky", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			fetchtest.Text(503, html, "busy")(w, r)
			return
		}
		fetchtest.Text(200, html, "ok")(w, r)
	})
	result, _ = h.client.GetPage(context.Background(), site, 1, "https://www.a.test/flaky")
	if result.Outcome != acquire.OutcomeFetched || result.Attempts != 2 {
		t.Fatalf("flaky: %+v", result)
	}
}

func TestBlockingResponsesPauseSite(t *testing.T) {
	challengeHeader := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cf-Mitigated", "challenge")
		w.Header().Set("Content-Type", html)
		w.WriteHeader(503)
	}
	cases := []struct {
		name    string
		handler http.HandlerFunc
		sitemap bool
		reason  acquire.PauseReason
	}{
		{"403 page", fetchtest.Text(403, html, "no"), false, acquire.PauseForbidden},
		{"429 page", fetchtest.Text(429, html, "slow down"), false, acquire.PauseTooManyRequests},
		{"403 sitemap", fetchtest.Text(403, "text/xml", "no"), true, acquire.PauseForbidden},
		{"challenge header", challengeHeader, false, acquire.PauseBotChallenge},
		{"challenge page 200", fetchtest.Text(200, html, "<html><title>Just a moment...</title></html>"), false, acquire.PauseBotChallenge},
		{"challenge 429 sitemap", fetchtest.Text(429, html, "<script>window._cf_chl_opt={}</script>"), true, acquire.PauseBotChallenge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			h.robots(www, 200, "")
			h.mux.Handle(www, "/x", tc.handler)
			get := func(ctx context.Context, site acquire.Site, raw string) (acquire.Result, error) {
				return h.client.GetPage(ctx, site, 1, raw)
			}
			if tc.sitemap {
				get = h.client.GetSitemap
			}
			result, err := get(context.Background(), site, "https://www.a.test/x")
			if err != nil || result.Outcome != acquire.OutcomePaused || result.Pause == nil || result.Pause.Reason != tc.reason {
				t.Fatalf("%v %+v", err, result)
			}
			if result.Attempts != 1 || h.mux.Hits(www, "/x") != 1 {
				t.Fatalf("blocking response retried: attempts=%d", result.Attempts)
			}
			stored, ok, _ := h.pauses.ActivePause(context.Background(), site.ID)
			if !ok || stored.Reason != tc.reason || stored.Status == 0 || stored.Detail == "" || stored.AttemptID == 0 {
				t.Fatalf("stored pause = %+v", stored)
			}
		})
	}
}

func TestOtherOutcomesAreFinal(t *testing.T) {
	h := newHarness(t)
	h.robots(www, 200, "")
	h.mux.Handle(www, "/gone", fetchtest.Text(404, html, "gone"))
	h.mux.Handle(www, "/json", fetchtest.Text(200, "application/json", "{}"))
	h.mux.Handle(www, "/off", fetchtest.Redirect(302, "https://elsewhere.test/"))
	for path, want := range map[string]acquire.Outcome{
		"/gone": acquire.OutcomeFailed, "/json": acquire.OutcomeRejected, "/off": acquire.OutcomeRejected,
	} {
		result, err := h.client.GetPage(context.Background(), site, 1, "https://www.a.test"+path)
		if err != nil || result.Outcome != want || result.Attempts != 1 {
			t.Errorf("%s: %v %+v", path, err, result)
		}
	}
	if _, ok, _ := h.pauses.ActivePause(context.Background(), site.ID); ok {
		t.Fatal("final failures paused the site")
	}
	for _, attempt := range h.recorder.Attempts() {
		if attempt.SiteID != site.ID || attempt.StartedAt.IsZero() || attempt.URL == "" {
			t.Errorf("incomplete attempt %+v", attempt)
		}
	}
}

func TestIsBotChallengeIgnoresInjectedScripts(t *testing.T) {
	body := []byte(`<html><title>Soup</title><script src="/cdn-cgi/challenge-platform/scripts/jsd/main.js"></script></html>`)
	if acquire.IsBotChallenge(http.Header{}, body) {
		t.Fatal("ordinary Cloudflare page treated as challenge")
	}
}
