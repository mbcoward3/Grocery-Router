package store_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/database"
	"github.com/mbcoward3/grocery-router/internal/store"
	"github.com/mbcoward3/grocery-router/internal/testdatabase"
)

func TestHostRateLimitAllowsOneLeaseAcrossProcesses(t *testing.T) {
	ctx := context.Background()
	db, databaseURL := testdatabase.OpenWithURL(t)

	// Each worker uses its own connection pool, standing in for a separate process.
	const workers = 12
	pools := make([]*sql.DB, workers)
	for index := range pools {
		pool, err := database.Open(databaseURL)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { pool.Close() })
		pools[index] = pool
	}

	for round := range 3 {
		var (
			wait    sync.WaitGroup
			start   = make(chan struct{})
			mutex   sync.Mutex
			winners []string
		)
		for _, pool := range pools {
			wait.Go(func() {
				<-start
				lease, err := store.New(pool).AcquireIngestHostSlot(ctx, store.AcquireIngestHostSlotParams{
					Host: "www.example.com", LeaseMs: 60_000,
				})
				if errors.Is(err, sql.ErrNoRows) {
					return
				}
				if err != nil {
					t.Errorf("acquire: %v", err)
					return
				}
				mutex.Lock()
				winners = append(winners, lease.LeaseToken.String)
				mutex.Unlock()
			})
		}
		close(start)
		wait.Wait()
		if len(winners) != 1 {
			t.Fatalf("round %d: %d workers acquired the host, want exactly 1", round, len(winners))
		}

		queries := store.New(db)
		released, err := queries.ReleaseIngestHostSlot(ctx, store.ReleaseIngestHostSlotParams{
			Host: "www.example.com", LeaseToken: winners[0], IntervalMs: 0,
		})
		if err != nil || released != 1 {
			t.Fatalf("release = %d, %v", released, err)
		}
		// Even with a zero interval, the 5-second floor holds the host closed.
		if _, err := queries.AcquireIngestHostSlot(ctx, store.AcquireIngestHostSlotParams{
			Host: "www.example.com", LeaseMs: 60_000,
		}); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("acquire inside the interval: err = %v, want sql.ErrNoRows", err)
		}
		state, err := queries.GetIngestHostRateLimit(ctx, "www.example.com")
		if err != nil {
			t.Fatal(err)
		}
		if wait := state.NextAllowedAt.Sub(state.DatabaseNow); wait < 4*time.Second || wait > 5*time.Second {
			t.Fatalf("next allowed in %s, want about 5s", wait)
		}
		// Stand in for the interval passing before the next round.
		mustExec(t, db, "UPDATE ingest.host_rate_limits SET next_allowed_at = clock_timestamp() - INTERVAL '1 second'")
	}
}

func TestHostRateLimitRecoversExpiredLease(t *testing.T) {
	ctx := context.Background()
	queries := store.New(testdatabase.Open(t))

	crashed, err := queries.AcquireIngestHostSlot(ctx, store.AcquireIngestHostSlotParams{Host: "example.com", LeaseMs: 1})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	recovered, err := queries.AcquireIngestHostSlot(ctx, store.AcquireIngestHostSlotParams{Host: "example.com", LeaseMs: 60_000})
	if err != nil {
		t.Fatalf("acquire after lease expiry: %v", err)
	}
	if recovered.LeaseToken == crashed.LeaseToken {
		t.Fatal("recovered lease reused the expired token")
	}
	stale, err := queries.ReleaseIngestHostSlot(ctx, store.ReleaseIngestHostSlotParams{
		Host: "example.com", LeaseToken: crashed.LeaseToken.String, IntervalMs: 5000,
	})
	if err != nil || stale != 0 {
		t.Fatalf("stale release = %d, %v; want 0 rows", stale, err)
	}
	if err := queries.DeferIngestHost(ctx, store.DeferIngestHostParams{Host: "example.com", NotBefore: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := queries.DeferIngestHost(ctx, store.DeferIngestHostParams{Host: "example.com", NotBefore: time.Now()}); err != nil {
		t.Fatal(err)
	}
	state, err := queries.GetIngestHostRateLimit(ctx, "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if state.NextAllowedAt.Before(time.Now().Add(50 * time.Minute)) {
		t.Fatalf("defer moved next_allowed_at earlier: %s", state.NextAllowedAt)
	}
}

func TestSitePausePersistsUntilExplicitResume(t *testing.T) {
	ctx := context.Background()
	db, databaseURL := testdatabase.OpenWithURL(t)
	queries := store.New(db)
	insertSite(t, queries, "example-kitchen", true)

	paused, err := queries.PauseIngestSite(ctx, store.PauseIngestSiteParams{
		SiteID: "example-kitchen", Reason: "http_429", Detail: "429 on https://www.example.com/sitemap.xml",
	})
	if err != nil || paused != 1 {
		t.Fatalf("pause = %d, %v", paused, err)
	}
	// A second pause keeps the first reason.
	paused, err = queries.PauseIngestSite(ctx, store.PauseIngestSiteParams{
		SiteID: "example-kitchen", Reason: "http_403", Detail: "403 later",
	})
	if err != nil || paused != 0 {
		t.Fatalf("second pause = %d, %v; want 0 rows", paused, err)
	}

	// A later run on a fresh connection still sees the pause.
	laterRun, err := database.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer laterRun.Close()
	later := store.New(laterRun)
	open, err := later.GetOpenIngestSitePause(ctx, "example-kitchen")
	if err != nil || open.Reason != "http_429" {
		t.Fatalf("open pause = %+v, %v", open, err)
	}
	assertFetchable(t, later, nil)

	_, err = db.Exec("DELETE FROM ingest.site_pauses WHERE id = $1", open.ID)
	assertErrorContains(t, err, "never deleted")
	_, err = db.Exec("UPDATE ingest.site_pauses SET reason = 'operator' WHERE id = $1", open.ID)
	assertErrorContains(t, err, "may only be resumed")

	resumed, err := later.ResumeIngestSite(ctx, store.ResumeIngestSiteParams{SiteID: "example-kitchen", ResumeNote: "rate limit cleared"})
	if err != nil || resumed != 1 {
		t.Fatalf("resume = %d, %v", resumed, err)
	}
	assertFetchable(t, later, []string{"example-kitchen"})
	resumed, err = later.ResumeIngestSite(ctx, store.ResumeIngestSiteParams{SiteID: "example-kitchen", ResumeNote: "again"})
	if err != nil || resumed != 0 {
		t.Fatalf("second resume = %d, %v; want 0 rows", resumed, err)
	}
	_, err = db.Exec("UPDATE ingest.site_pauses SET resume_note = 'rewritten' WHERE id = $1", open.ID)
	assertErrorContains(t, err, "already resumed")

	// History is kept, and the site can be paused again.
	paused, err = later.PauseIngestSite(ctx, store.PauseIngestSiteParams{SiteID: "example-kitchen", Reason: "operator", Detail: "manual hold"})
	if err != nil || paused != 1 {
		t.Fatalf("re-pause = %d, %v", paused, err)
	}
	var history int
	if err := db.QueryRow("SELECT count(*) FROM ingest.site_pauses").Scan(&history); err != nil || history != 2 {
		t.Fatalf("pause history = %d, %v; want 2", history, err)
	}
}

func TestEvidenceIsImmutable(t *testing.T) {
	ctx := context.Background()
	db := testdatabase.Open(t)
	queries := store.New(db)
	insertSite(t, queries, "example-kitchen", true)
	insertSite(t, queries, "other-kitchen", false)

	run, err := queries.StartIngestRun(ctx, store.StartIngestRunParams{Command: "catalog-fetch", Parameters: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	page, err := queries.UpsertIngestSourcePage(ctx, store.UpsertIngestSourcePageParams{
		SiteID: "example-kitchen", Url: "https://www.example.com/lentil-soup/",
		DiscoveredVia: "https://www.example.com/sitemap.xml", FirstRunID: sql.NullInt64{Int64: run.ID, Valid: true},
	})
	if err != nil || !page.Inserted || page.State != "queued" {
		t.Fatalf("upsert page = %+v, %v", page, err)
	}
	again, err := queries.UpsertIngestSourcePage(ctx, store.UpsertIngestSourcePageParams{
		SiteID: "example-kitchen", Url: "https://www.example.com/lentil-soup/",
		DiscoveredVia: "https://www.example.com/sitemap.xml",
	})
	if err != nil || again.Inserted || again.ID != page.ID {
		t.Fatalf("re-upsert page = %+v, %v", again, err)
	}
	due, err := queries.ListDueIngestSourcePages(ctx, store.ListDueIngestSourcePagesParams{SiteID: "example-kitchen", RowLimit: 10})
	if err != nil || len(due) != 1 {
		t.Fatalf("due pages = %d, %v", len(due), err)
	}

	now := time.Now()
	attemptID, err := queries.InsertIngestFetchAttempt(ctx, store.InsertIngestFetchAttemptParams{
		RunID: sql.NullInt64{Int64: run.ID, Valid: true}, SiteID: "example-kitchen",
		SourcePageID: sql.NullInt64{Int64: page.ID, Valid: true}, Kind: "page",
		RequestedUrl: "https://www.example.com/lentil-soup/",
		FinalUrl:     sql.NullString{String: "https://www.example.com/lentil-soup/", Valid: true},
		HttpStatus:   sql.NullInt32{Int32: 200, Valid: true}, Outcome: "success",
		StartedAt: now, FinishedAt: now.Add(time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	body := []byte("<html></html>")
	digest := sha256.Sum256(body)
	evidenceParams := store.InsertIngestEvidenceParams{
		FetchAttemptID: attemptID, SiteID: "example-kitchen", SourcePageID: page.ID,
		RequestedUrl: "https://www.example.com/lentil-soup/", FinalUrl: "https://www.example.com/lentil-soup/",
		RedirectChain: json.RawMessage(`[]`), HttpStatus: 200, FetchedAt: now, MediaType: "text/html",
		BodySha256: digest[:], BodySize: int32(len(body)), BodyEncoding: "gzip", BodyCompressed: []byte{0x1f, 0x8b},
		CanonicalLink: sql.NullString{String: "https://www.example.com/lentil-soup/", Valid: true},
	}
	// Evidence must stay on its page's site and use a valid digest.
	wrongSite := evidenceParams
	wrongSite.SiteID = "other-kitchen"
	_, err = queries.InsertIngestEvidence(ctx, wrongSite)
	assertErrorContains(t, err, "foreign key")
	badDigest := evidenceParams
	badDigest.BodySha256 = digest[:16]
	_, err = queries.InsertIngestEvidence(ctx, badDigest)
	assertErrorContains(t, err, "check constraint")

	evidenceID, err := queries.InsertIngestEvidence(ctx, evidenceParams)
	if err != nil {
		t.Fatal(err)
	}
	blockID, err := queries.InsertIngestJSONLDBlock(ctx, store.InsertIngestJSONLDBlockParams{
		EvidenceID: evidenceID, Position: 0, RawText: `{"@type":"Recipe","name":"Lentil Soup"}`, ParseStatus: "parsed",
	})
	if err != nil {
		t.Fatal(err)
	}
	nodeDigest := sha256.Sum256([]byte("normalized"))
	nodeID, err := queries.InsertIngestSourceNode(ctx, store.InsertIngestSourceNodeParams{
		SiteID: "example-kitchen", EvidenceID: evidenceID, JsonldBlockID: blockID, Location: "",
		NormalizationVersion: 1, ContentDigest: nodeDigest[:], RawNode: json.RawMessage(`{"@type":"Recipe","name":"Lentil Soup"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := queries.MarkIngestSourcePageFetched(ctx, store.MarkIngestSourcePageFetchedParams{ID: page.ID, EvidenceID: sql.NullInt64{Int64: evidenceID, Valid: true}}); err != nil {
		t.Fatalf("queue state stays mutable: %v", err)
	}

	for _, statement := range []struct{ query string }{
		{"UPDATE ingest.fetch_attempts SET outcome = 'timeout' WHERE id = $1"},
		{"DELETE FROM ingest.fetch_attempts WHERE id = $1"},
		{"UPDATE ingest.evidence SET final_url = 'https://www.example.com/other/' WHERE id = $1"},
		{"DELETE FROM ingest.evidence WHERE id = $1"},
		{"UPDATE ingest.evidence_jsonld_blocks SET raw_text = '{}' WHERE id = $1"},
		{"DELETE FROM ingest.evidence_jsonld_blocks WHERE id = $1"},
		{"UPDATE ingest.source_nodes SET location = '/0' WHERE id = $1"},
		{"DELETE FROM ingest.source_nodes WHERE id = $1"},
	} {
		id := map[string]int64{
			"fetch_attempts": attemptID, "evidence": evidenceID, "evidence_jsonld_blocks": blockID, "source_nodes": nodeID,
		}[tableName(statement.query)]
		_, err := db.Exec(statement.query, id)
		assertErrorContains(t, err, "rows are immutable")
	}

	stored, err := queries.GetIngestEvidence(ctx, evidenceID)
	if err != nil || stored.FinalUrl != "https://www.example.com/lentil-soup/" {
		t.Fatalf("stored evidence = %+v, %v", stored, err)
	}
	finished, err := queries.FinishIngestRun(ctx, store.FinishIngestRunParams{ID: run.ID, State: "completed", Counters: json.RawMessage(`{"fetched":1}`)})
	if err != nil || finished != 1 {
		t.Fatalf("finish run = %d, %v", finished, err)
	}
}

func insertSite(t *testing.T, queries *store.Queries, id string, active bool) {
	t.Helper()
	params := store.UpsertIngestSiteParams{
		ID: id, AttributionName: id, IncludePatterns: json.RawMessage(`[]`), ExcludePatterns: json.RawMessage(`[]`),
		MeasurementConvention: "us", Active: active,
	}
	if active {
		params.EligibilityIntendedUse = sql.NullString{String: "private household use", Valid: true}
		params.EligibilityReviewedOn = sql.NullTime{Time: time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC), Valid: true}
	}
	if err := queries.UpsertIngestSite(context.Background(), params); err != nil {
		t.Fatalf("insert site %s: %v", id, err)
	}
}

func assertFetchable(t *testing.T, queries *store.Queries, want []string) {
	t.Helper()
	sites, err := queries.ListFetchableIngestSites(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, site := range sites {
		got = append(got, site.ID)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("fetchable sites = %v, want %v", got, want)
	}
}

func tableName(query string) string {
	for _, field := range strings.Fields(query) {
		if name, ok := strings.CutPrefix(field, "ingest."); ok {
			return name
		}
	}
	return ""
}

func mustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want it to contain %q", err, want)
	}
}
