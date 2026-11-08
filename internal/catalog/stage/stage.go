// Package stage runs the slice 2 acquisition stages against PostgreSQL:
// discovery, fetching, extraction replay, site resume, and reporting
// (CATALOG_INGEST_SPEC.md §4 slice 2, §7). Each stage holds a PostgreSQL
// advisory lock and records an ingest.runs row.
package stage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/mbcoward3/grocery-router/internal/catalog/acquire"
	"github.com/mbcoward3/grocery-router/internal/catalog/clock"
	"github.com/mbcoward3/grocery-router/internal/catalog/ingeststore"
	"github.com/mbcoward3/grocery-router/internal/catalog/ratelimit"
	"github.com/mbcoward3/grocery-router/internal/catalog/registry"
	"github.com/mbcoward3/grocery-router/internal/catalog/robots"
	"github.com/mbcoward3/grocery-router/internal/catalog/safefetch"
	"github.com/mbcoward3/grocery-router/internal/store"
)

// Advisory lock keys, one per stage. Replay shares the fetch lock because both
// write source nodes.
const (
	lockDiscover int64 = 0x6772_6361_0001
	lockFetch    int64 = 0x6772_6361_0002
	lockResume   int64 = 0x6772_6361_0003
)

// ErrStageBusy means another process holds the stage's advisory lock.
var ErrStageBusy = errors.New("another run of this stage holds its lock")

// Config wires a stage to its database and, in tests, to fakes.
type Config struct {
	DB *sql.DB
	// Contact is the public support address named in the user agent.
	Contact string
	// Fetcher defaults to a safefetch client naming Contact.
	Fetcher acquire.Fetcher
	// Limiter defaults to the PostgreSQL host rate limiter.
	Limiter ratelimit.Limiter
	// Clock defaults to the wall clock.
	Clock clock.Clock
	// Backoff overrides acquire.DefaultBackoff.
	Backoff []time.Duration
	// Progress, when set, receives one line per notable event.
	Progress func(format string, args ...any)
}

func (config Config) progress(format string, args ...any) {
	if config.Progress != nil {
		config.Progress(format, args...)
	}
}

func (config Config) client(runID int64) (*acquire.Client, error) {
	fetcher := config.Fetcher
	if fetcher == nil {
		safe, err := safefetch.New(config.Contact)
		if err != nil {
			return nil, err
		}
		fetcher = safe
	}
	limiter := config.Limiter
	if limiter == nil {
		limiter = ingeststore.NewLimiter(config.DB)
	}
	clk := config.Clock
	if clk == nil {
		clk = clock.Real{}
	}
	queries := store.New(config.DB)
	return &acquire.Client{
		Fetcher: fetcher, Limiter: limiter, Robots: robots.NewMemoryCache(),
		Pauses:   &ingeststore.Pauses{Queries: queries, RunID: runID},
		Recorder: &ingeststore.Recorder{Queries: queries, RunID: runID},
		Clock:    clk, Backoff: config.Backoff,
	}, nil
}

// withLock runs fn while holding a session-level advisory lock on one
// dedicated connection.
func withLock(ctx context.Context, db *sql.DB, key int64, fn func() error) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("open lock connection: %w", err)
	}
	defer conn.Close()
	queries := store.New(conn)
	acquired, err := queries.TryIngestStageLock(ctx, key)
	if err != nil {
		return fmt.Errorf("take stage lock: %w", err)
	}
	if !acquired {
		return ErrStageBusy
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		_, _ = queries.ReleaseIngestStageLock(unlockCtx, key)
	}()
	return fn()
}

// Counters accumulate a run's numbers; they are stored on its runs row.
type Counters struct {
	mu     sync.Mutex
	values map[string]int64
}

// Add increments name by delta.
func (counters *Counters) Add(name string, delta int64) {
	counters.mu.Lock()
	defer counters.mu.Unlock()
	if counters.values == nil {
		counters.values = map[string]int64{}
	}
	counters.values[name] += delta
}

// Get returns name's value.
func (counters *Counters) Get(name string) int64 {
	counters.mu.Lock()
	defer counters.mu.Unlock()
	return counters.values[name]
}

// Map returns a copy of every counter.
func (counters *Counters) Map() map[string]int64 {
	counters.mu.Lock()
	defer counters.mu.Unlock()
	copied := make(map[string]int64, len(counters.values))
	for key, value := range counters.values {
		copied[key] = value
	}
	return copied
}

func (counters *Counters) json() json.RawMessage {
	encoded, err := json.Marshal(counters.Map())
	if err != nil || len(counters.Map()) == 0 {
		return json.RawMessage("{}")
	}
	return encoded
}

// run is one ingest.runs row.
type run struct {
	queries  *store.Queries
	id       int64
	counters *Counters
}

func startRun(ctx context.Context, db *sql.DB, command, siteID string, parameters map[string]any) (*run, error) {
	queries := store.New(db)
	// The lock is held, so any running row for this command is from a process
	// that ended without finishing.
	running, err := queries.ListRunningIngestRuns(ctx)
	if err != nil {
		return nil, fmt.Errorf("list running runs: %w", err)
	}
	for _, stale := range running {
		if stale.Command != command {
			continue
		}
		if _, err := queries.FinishIngestRun(ctx, store.FinishIngestRunParams{
			State: "interrupted", Counters: stale.Counters, ID: stale.ID,
			Error: sql.NullString{String: "process ended before the run finished", Valid: true},
		}); err != nil {
			return nil, fmt.Errorf("close stale run %d: %w", stale.ID, err)
		}
	}
	encoded, err := json.Marshal(parameters)
	if err != nil {
		return nil, fmt.Errorf("encode run parameters: %w", err)
	}
	row, err := queries.StartIngestRun(ctx, store.StartIngestRunParams{
		Command: command, SiteID: sql.NullString{String: siteID, Valid: siteID != ""}, Parameters: encoded,
	})
	if err != nil {
		return nil, fmt.Errorf("start run: %w", err)
	}
	return &run{queries: queries, id: row.ID, counters: &Counters{}}, nil
}

// finish closes the run as completed, interrupted (context ended), or failed.
func (current *run) finish(ctx context.Context, runErr error) error {
	state := "completed"
	var message sql.NullString
	switch {
	case runErr == nil:
	case ctx.Err() != nil:
		state, message = "interrupted", sql.NullString{String: runErr.Error(), Valid: true}
	default:
		state, message = "failed", sql.NullString{String: runErr.Error(), Valid: true}
	}
	finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if _, err := current.queries.FinishIngestRun(finishCtx, store.FinishIngestRunParams{
		State: state, Counters: current.counters.json(), Error: message, ID: current.id,
	}); err != nil {
		return errors.Join(runErr, fmt.Errorf("finish run %d: %w", current.id, err))
	}
	return runErr
}

// fetchableSites returns the registry sites that are active, eligible, and
// not paused, optionally limited to one site. A named site that is not
// fetchable is returned in skipped with the reason.
func fetchableSites(ctx context.Context, db *sql.DB, only string) ([]registry.Site, map[string]string, error) {
	queries := store.New(db)
	listed, err := registry.ReadStored(ctx, queries)
	if err != nil {
		return nil, nil, err
	}
	fetchable, err := queries.ListFetchableIngestSites(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("list fetchable sites: %w", err)
	}
	allowed := map[string]bool{}
	for _, site := range fetchable {
		allowed[site.ID] = true
	}
	skipped := map[string]string{}
	if only != "" {
		site, ok := listed.Site(only)
		if !ok {
			return nil, nil, fmt.Errorf("site %q is not in the synced registry (run catalog-sites)", only)
		}
		if !allowed[only] {
			skipped[only] = notFetchableReason(ctx, queries, site)
			return nil, skipped, nil
		}
		return []registry.Site{site}, skipped, nil
	}
	var sites []registry.Site
	for _, site := range listed.Sites {
		if allowed[site.ID] {
			sites = append(sites, site)
		} else if site.Active {
			skipped[site.ID] = notFetchableReason(ctx, queries, site)
		}
	}
	return sites, skipped, nil
}

func notFetchableReason(ctx context.Context, queries *store.Queries, site registry.Site) string {
	if !site.Active {
		return "inactive"
	}
	if site.Eligibility == nil {
		return "no eligibility record"
	}
	if pause, err := queries.GetOpenIngestSitePause(ctx, site.ID); err == nil {
		return fmt.Sprintf("paused (%s: %s) until catalog-site-resume", pause.Reason, pause.Detail)
	}
	return "not fetchable"
}

func acquireSite(site registry.Site) acquire.Site {
	return acquire.Site{ID: site.ID, Hosts: site.Hosts, MinInterval: site.MinInterval}
}

// storableURL returns raw in the canonical stored form, or "" if it cannot
// be stored (the schema requires a lowercase host and a path).
func storableURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	parsed.Host = safefetch.NormalizeHost(parsed.Host)
	if parsed.Path == "" && parsed.RawPath == "" {
		parsed.Path = "/"
	}
	normalized := parsed.String()
	if !ingeststore.StorableURL(normalized) {
		return ""
	}
	return normalized
}

func truncate(value string, limit int) string {
	value = ingeststore.CleanText(strings.TrimSpace(value))
	if len(value) <= limit {
		return value
	}
	return strings.ToValidUTF8(value[:limit], "")
}
