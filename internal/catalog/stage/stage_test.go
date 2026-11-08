package stage_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/catalog/acquire"
	"github.com/mbcoward3/grocery-router/internal/catalog/clock"
	"github.com/mbcoward3/grocery-router/internal/catalog/fetchtest"
	"github.com/mbcoward3/grocery-router/internal/catalog/ingeststore"
	"github.com/mbcoward3/grocery-router/internal/catalog/ratelimit"
	"github.com/mbcoward3/grocery-router/internal/catalog/registry"
	"github.com/mbcoward3/grocery-router/internal/catalog/robots"
	"github.com/mbcoward3/grocery-router/internal/catalog/stage"
	"github.com/mbcoward3/grocery-router/internal/database"
	"github.com/mbcoward3/grocery-router/internal/testdatabase"
)

const host = "www.a.test"

const siteRegistry = `version: 1
sites:
  - id: site-a
    name: Site A
    hosts: [www.a.test]
    sitemaps: [https://www.a.test/sitemap.xml]
    include: ['^/recipes/']
    measurement_convention: us
    active: true
    eligibility:
      intended_use: private household use
      reviewed_on: 2026-11-08
      notes: test fixture
`

type harness struct {
	db      *sql.DB
	url     string
	mux     *fetchtest.Mux
	network *fetchtest.Network
	config  stage.Config
}

func recipePage(name string) string {
	return fmt.Sprintf(`<html><head><link rel="canonical" href="https://www.a.test/recipes/%[1]s">
<script type="application/ld+json">{"@context":"https://schema.org","@graph":[
 {"@type":"WebPage","name":"%[1]s page"},
 {"@type":"Recipe","@id":"https://www.a.test/recipes/%[1]s#recipe","name":"%[1]s","recipeIngredient":["1 onion"]}]}</script>
<script type="application/ld+json">{broken</script></head></html>`, name)
}

func newHarness(t *testing.T, pages ...string) *harness {
	t.Helper()
	db, databaseURL := testdatabase.OpenWithURL(t)
	sites, err := registry.Parse([]byte(siteRegistry))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Sync(context.Background(), db, sites); err != nil {
		t.Fatal(err)
	}
	mux := fetchtest.NewMux()
	network := fetchtest.New(t, mux, host)
	mux.Handle(host, "/robots.txt", fetchtest.Text(200, "text/plain", "User-agent: *\nDisallow: /recipes/private\n"))
	var locs []string
	for _, page := range pages {
		locs = append(locs, "<url><loc>https://www.a.test"+page+"</loc><lastmod>2026-10-01</lastmod></url>")
	}
	mux.Handle(host, "/sitemap.xml", fetchtest.Text(200, "application/xml",
		`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`+strings.Join(locs, "")+
			`<url><loc>https://elsewhere.test/recipes/x</loc></url><url><loc>https://www.a.test/about</loc></url></urlset>`))
	clk := clock.NewFake(time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC))
	return &harness{
		db: db, url: databaseURL, mux: mux, network: network,
		config: stage.Config{
			DB: db, Contact: "support@example.com", Fetcher: network.Client(t),
			Limiter: ratelimit.NewMemory(clk), Clock: clk, Backoff: []time.Duration{time.Second},
		},
	}
}

func (h *harness) count(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	if err := h.db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}

func (h *harness) discover(t *testing.T) {
	t.Helper()
	summary, err := stage.Discover(context.Background(), h.config, stage.DiscoverOptions{})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(summary.Sites) != 1 || summary.Sites[0].Report.UnregisteredHost != 1 || summary.Sites[0].Report.Excluded != 1 {
		t.Fatalf("discover summary = %+v", summary)
	}
}

func TestInterruptedFetchResumesWithoutRefetching(t *testing.T) {
	h := newHarness(t, "/recipes/one", "/recipes/two", "/recipes/three", "/recipes/no-recipe", "/recipes/gone", "/recipes/private/x")
	h.discover(t)
	if got := h.count(t, "SELECT count(*) FROM ingest.source_pages WHERE state = 'queued'"); got != 6 {
		t.Fatalf("queued = %d", got)
	}
	// Discovery is idempotent: known pages keep their state.
	h.discover(t)
	if got := h.count(t, "SELECT count(*) FROM ingest.source_pages"); got != 6 {
		t.Fatalf("pages after rediscovery = %d", got)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, name := range []string{"one", "three"} {
		h.mux.Handle(host, "/recipes/"+name, fetchtest.Text(200, "text/html", recipePage(name)))
	}
	h.mux.Handle(host, "/recipes/no-recipe", fetchtest.Text(200, "text/html", "<html>no structured data</html>"))
	h.mux.Handle(host, "/recipes/gone", fetchtest.Text(404, "text/html", "gone"))
	interrupted := true
	h.mux.Handle(host, "/recipes/two", func(w http.ResponseWriter, r *http.Request) {
		if interrupted {
			cancel() // the operator interrupts while page two is in flight
			<-r.Context().Done()
			return
		}
		fetchtest.Text(200, "text/html", recipePage("two"))(w, r)
	})

	// --limit caps pages per site.
	limited, err := stage.Fetch(context.Background(), h.config, stage.FetchOptions{Limit: 1})
	if err != nil || limited.Counters["fetched"] != 1 {
		t.Fatalf("limited fetch: %v %+v", err, limited.Counters)
	}

	_, err = stage.Fetch(ctx, h.config, stage.FetchOptions{})
	if err == nil {
		t.Fatal("interrupted fetch returned no error")
	}
	if got := h.count(t, "SELECT count(*) FROM ingest.runs WHERE command = 'catalog-fetch' AND state = 'interrupted'"); got != 1 {
		t.Fatalf("interrupted runs = %d", got)
	}
	if got := h.count(t, "SELECT count(*) FROM ingest.source_pages WHERE state = 'queued'"); got != 5 {
		t.Fatalf("queued after interruption = %d", got)
	}

	interrupted = false
	summary, err := stage.Fetch(context.Background(), h.config, stage.FetchOptions{Site: "site-a"})
	if err != nil {
		t.Fatalf("resumed fetch: %v", err)
	}
	if h.mux.Hits(host, "/recipes/one") != 1 || h.mux.Hits(host, "/recipes/two") != 2 || h.mux.Hits(host, "/recipes/three") != 1 {
		t.Fatalf("hits one=%d two=%d three=%d", h.mux.Hits(host, "/recipes/one"), h.mux.Hits(host, "/recipes/two"), h.mux.Hits(host, "/recipes/three"))
	}
	if h.mux.Hits(host, "/recipes/private/x") != 0 {
		t.Fatal("robots-disallowed page was requested")
	}
	if summary.Counters["fetched"] != 3 || summary.Counters["disallowed"] != 1 || summary.Counters["failed"] != 1 || summary.Counters["no_recipe"] != 1 {
		t.Fatalf("resumed counters = %+v", summary.Counters)
	}
	if got := h.count(t, "SELECT count(*) FROM ingest.source_pages WHERE state = 'queued'"); got != 0 {
		t.Fatalf("queued after resume = %d", got)
	}
	if got := h.count(t, "SELECT count(*) FROM ingest.source_nodes"); got != 3 {
		t.Fatalf("source nodes = %d", got)
	}
	if got := h.count(t, "SELECT count(*) FROM ingest.evidence_jsonld_blocks WHERE parse_status = 'invalid_json'"); got != 3 {
		t.Fatalf("malformed blocks = %d", got)
	}
	if got := h.count(t, "SELECT count(*) FROM ingest.evidence WHERE canonical_link IS NOT NULL"); got != 3 {
		t.Fatalf("canonical links = %d", got)
	}
	// Every network attempt is recorded; the robots refusal is not one.
	if got := h.count(t, "SELECT count(*) FROM ingest.fetch_attempts WHERE kind = 'page' AND outcome = 'success'"); got != 4 {
		t.Fatalf("successful page attempts = %d", got)
	}

	// Nothing left to do: a further run fetches nothing.
	again, err := stage.Fetch(context.Background(), h.config, stage.FetchOptions{})
	if err != nil || again.Counters["fetched"] != 0 || h.mux.Hits(host, "/recipes/one") != 1 {
		t.Fatalf("idle run: %v %+v", err, again.Counters)
	}

	reports, err := stage.Report(context.Background(), h.db)
	if err != nil {
		t.Fatal(err)
	}
	report := reports[0]
	if report.Discovered != 6 || report.Fetched != 4 || report.Failed != 2 || report.NoRecipe != 1 || report.SourceNodes != 3 || report.Pause != "" {
		t.Fatalf("report = %+v", report)
	}
	var out strings.Builder
	if err := stage.WriteReport(&out, reports); err != nil || !strings.Contains(out.String(), "site-a") {
		t.Fatalf("WriteReport: %v %q", err, out.String())
	}
}

func TestPausePersistsAcrossRunsUntilResumed(t *testing.T) {
	h := newHarness(t, "/recipes/one", "/recipes/two")
	h.discover(t)
	h.mux.Handle(host, "/recipes/one", fetchtest.Text(403, "text/html", "forbidden"))
	h.mux.Handle(host, "/recipes/two", fetchtest.Text(200, "text/html", recipePage("two")))

	summary, err := stage.Fetch(context.Background(), h.config, stage.FetchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	pause, ok := summary.Paused["site-a"]
	if !ok || pause.Reason != acquire.PauseForbidden || h.mux.Hits(host, "/recipes/two") != 0 {
		t.Fatalf("summary = %+v", summary)
	}
	if got := h.count(t, `SELECT count(*) FROM ingest.site_pauses
		WHERE site_id = 'site-a' AND resumed_at IS NULL AND reason = 'http_403' AND fetch_attempt_id IS NOT NULL AND run_id IS NOT NULL`); got != 1 {
		t.Fatalf("open pauses = %d", got)
	}

	for range 2 {
		later, err := stage.Fetch(context.Background(), h.config, stage.FetchOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(later.Skipped["site-a"], "paused") {
			t.Fatalf("skipped = %+v", later.Skipped)
		}
		discovered, err := stage.Discover(context.Background(), h.config, stage.DiscoverOptions{Site: "site-a"})
		if err != nil || len(discovered.Sites) != 0 {
			t.Fatalf("discover on paused site: %v %+v", err, discovered)
		}
	}
	if h.mux.Hits(host, "/recipes/one") != 1 || h.mux.Hits(host, "/recipes/two") != 0 {
		t.Fatal("paused site was fetched")
	}
	reports, _ := stage.Report(context.Background(), h.db)
	if !strings.Contains(reports[0].Pause, "http_403") {
		t.Fatalf("report pause = %q", reports[0].Pause)
	}

	if _, err := stage.Resume(context.Background(), h.db, "site-a", ""); err == nil {
		t.Fatal("resume without a note succeeded")
	}
	resumed, err := stage.Resume(context.Background(), h.db, "site-a", "403 was a temporary block")
	if err != nil || !resumed {
		t.Fatalf("Resume: %v %v", resumed, err)
	}
	if again, _ := stage.Resume(context.Background(), h.db, "site-a", "again"); again {
		t.Fatal("second resume reported an open pause")
	}
	h.mux.Handle(host, "/recipes/one", fetchtest.Text(200, "text/html", recipePage("one")))
	final, err := stage.Fetch(context.Background(), h.config, stage.FetchOptions{})
	if err != nil || final.Counters["fetched"] != 2 {
		t.Fatalf("after resume: %v %+v", err, final.Counters)
	}
}

func TestReplayReproducesSourceNodesWithoutNetwork(t *testing.T) {
	h := newHarness(t, "/recipes/one", "/recipes/two", "/recipes/no-recipe")
	h.discover(t)
	h.mux.Handle(host, "/recipes/one", fetchtest.Text(200, "text/html", recipePage("one")))
	h.mux.Handle(host, "/recipes/two", fetchtest.Text(200, "text/html", recipePage("two")))
	h.mux.Handle(host, "/recipes/no-recipe", fetchtest.Text(200, "text/html", "<p>none</p>"))
	if _, err := stage.Fetch(context.Background(), h.config, stage.FetchOptions{}); err != nil {
		t.Fatal(err)
	}
	h.network.Server.Close() // no network from here on
	stored := h.count(t, "SELECT count(*) FROM ingest.source_nodes")

	summary, err := stage.Replay(context.Background(), h.config, stage.ReplayOptions{})
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if summary.Evidence != 3 || summary.Matched != 3 || len(summary.Mismatched) != 0 || summary.Nodes != stored || stored != 2 {
		t.Fatalf("summary = %+v stored = %d", summary, stored)
	}
	if got := h.count(t, "SELECT count(*) FROM ingest.source_nodes"); got != stored {
		t.Fatalf("replay wrote nodes: %d", got)
	}
	if _, err := stage.Replay(context.Background(), h.config, stage.ReplayOptions{Site: "site-a"}); err != nil {
		t.Fatalf("scoped replay: %v", err)
	}
}

func TestStageLockIsExclusive(t *testing.T) {
	h := newHarness(t)
	conn, err := h.db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(context.Background(), "SELECT pg_advisory_lock($1)", int64(0x6772_6361_0002)); err != nil {
		t.Fatal(err)
	}
	if _, err := stage.Fetch(context.Background(), h.config, stage.FetchOptions{}); !errors.Is(err, stage.ErrStageBusy) {
		t.Fatalf("Fetch under held lock: %v", err)
	}
}

func TestDatabaseRateLimitIsSharedAcrossFetchers(t *testing.T) {
	db, databaseURL := testdatabase.OpenWithURL(t)
	other, err := database.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()

	var mu sync.Mutex
	type span struct{ start, end time.Time }
	var spans []span
	mux := fetchtest.NewMux()
	network := fetchtest.New(t, mux, host)
	for _, path := range []string{"/recipes/one", "/recipes/two"} {
		mux.Handle(host, path, func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			time.Sleep(100 * time.Millisecond)
			fetchtest.Text(200, "text/html", "<html></html>")(w, r)
			mu.Lock()
			spans = append(spans, span{start, time.Now()})
			mu.Unlock()
		})
	}
	robotsCache := robots.NewMemoryCache()
	_ = robotsCache.Put(context.Background(), robots.Policy{Host: host, Mode: robots.ModeAllowAll, FetchedAt: time.Now()})
	newClient := func(pool *sql.DB) *acquire.Client {
		return &acquire.Client{
			Fetcher: network.Client(t), Limiter: ingeststore.NewLimiter(pool), Robots: robotsCache,
			Pauses: acquire.NewMemoryPauses(), Recorder: &acquire.MemoryRecorder{}, Clock: clock.Real{},
		}
	}
	site := acquire.Site{ID: "site-a", Hosts: []string{host}}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for index, pool := range []*sql.DB{db, other} {
		wg.Add(1)
		go func(index int, pool *sql.DB) {
			defer wg.Done()
			path := []string{"/recipes/one", "/recipes/two"}[index]
			result, err := newClient(pool).GetPage(context.Background(), site, int64(index+1), "https://www.a.test"+path)
			if err == nil && result.Outcome != acquire.OutcomeFetched {
				err = fmt.Errorf("outcome %s: %w", result.Outcome, result.Err)
			}
			errs <- err
		}(index, pool)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(spans) != 2 {
		t.Fatalf("spans = %d", len(spans))
	}
	first, second := spans[0], spans[1]
	if second.start.Before(first.start) {
		first, second = second, first
	}
	if gap := second.start.Sub(first.end); gap < ratelimit.MinInterval-100*time.Millisecond {
		t.Fatalf("requests to one host from two fetchers were %v apart, want >= %v", gap, ratelimit.MinInterval)
	}
	var leased int
	if err := db.QueryRow("SELECT count(*) FROM ingest.host_rate_limits WHERE host = $1 AND lease_token IS NULL AND next_allowed_at > clock_timestamp()", host).Scan(&leased); err != nil || leased != 1 {
		t.Fatalf("host row after release: %d %v", leased, err)
	}
}
