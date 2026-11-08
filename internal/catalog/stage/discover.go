package stage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/mbcoward3/grocery-router/internal/catalog/discover"
	"github.com/mbcoward3/grocery-router/internal/catalog/registry"
	"github.com/mbcoward3/grocery-router/internal/store"
)

// DiscoverOptions scopes a discovery run.
type DiscoverOptions struct {
	// Site limits the run to one site ID; empty means every fetchable site.
	Site string
}

// SiteDiscovery summarizes one site's discovery.
type SiteDiscovery struct {
	SiteID string
	Report discover.Report
	// New and Known count stored entries; Conflicts are URLs owned by
	// another site; Unstorable fail the schema's URL check.
	New, Known, Conflicts, Unstorable int
}

// DiscoverSummary is the result of a discovery run.
type DiscoverSummary struct {
	RunID   int64
	Sites   []SiteDiscovery
	Skipped map[string]string
}

// registryFilter adapts a registry site's include/exclude patterns.
type registryFilter struct{ site registry.Site }

func (filter registryFilter) Allow(page *url.URL) bool { return filter.site.AllowsPage(page) }

// Discover walks each fetchable site's registered sitemaps and queues the
// pages it finds in ingest.source_pages. Known pages keep their state.
func Discover(ctx context.Context, config Config, options DiscoverOptions) (DiscoverSummary, error) {
	var summary DiscoverSummary
	err := withLock(ctx, config.DB, lockDiscover, func() error {
		sites, skipped, err := fetchableSites(ctx, config.DB, options.Site)
		if err != nil {
			return err
		}
		summary.Skipped = skipped
		current, err := startRun(ctx, config.DB, "catalog-discover", options.Site, map[string]any{"site": options.Site})
		if err != nil {
			return err
		}
		summary.RunID = current.id
		runErr := discoverSites(ctx, config, current, sites, &summary)
		return current.finish(ctx, runErr)
	})
	return summary, err
}

func discoverSites(ctx context.Context, config Config, current *run, sites []registry.Site, summary *DiscoverSummary) error {
	client, err := config.client(current.id)
	if err != nil {
		return err
	}
	queries := store.New(config.DB)
	for _, site := range sites {
		config.progress("discover %s: %d registered sitemaps", site.ID, len(site.Sitemaps))
		report, err := discover.Discover(ctx, client, discover.Site{
			Site: acquireSite(site), Sitemaps: site.Sitemaps, Filter: registryFilter{site: site},
		})
		if err != nil {
			return fmt.Errorf("discover %s: %w", site.ID, err)
		}
		result := SiteDiscovery{SiteID: site.ID, Report: report}
		for _, entry := range report.Entries {
			stored := storableURL(entry.URL)
			via := storableURL(entry.Sitemap)
			if stored == "" || via == "" {
				result.Unstorable++
				continue
			}
			row, err := queries.UpsertIngestSourcePage(ctx, store.UpsertIngestSourcePageParams{
				SiteID: site.ID, Url: stored, DiscoveredVia: via,
				FirstRunID: sql.NullInt64{Int64: current.id, Valid: true}, SitemapLastmod: parseLastMod(entry.LastMod),
			})
			switch {
			case errors.Is(err, sql.ErrNoRows):
				result.Conflicts++
			case err != nil:
				return fmt.Errorf("queue %s: %w", stored, err)
			case row.Inserted:
				result.New++
			default:
				result.Known++
			}
		}
		for name, value := range map[string]int{
			"entries": len(report.Entries), "new": result.New, "known": result.Known,
			"conflicts": result.Conflicts, "unstorable": result.Unstorable,
			"unregistered_host": report.UnregisteredHost, "invalid": report.Invalid,
			"excluded": report.Excluded, "duplicates": report.Duplicates, "cycles": report.Cycles,
			"too_deep": report.TooDeep, "sitemaps": len(report.Files),
		} {
			current.counters.Add(name, int64(value))
		}
		if report.Paused != nil {
			current.counters.Add("paused_sites", 1)
			config.progress("discover %s: site paused (%s: %s)", site.ID, report.Paused.Reason, report.Paused.Detail)
		}
		config.progress("discover %s: %d pages (%d new), %d unregistered-host entries ignored",
			site.ID, len(report.Entries), result.New, report.UnregisteredHost)
		summary.Sites = append(summary.Sites, result)
		if err := current.queries.UpdateIngestRunCounters(ctx, store.UpdateIngestRunCountersParams{
			Counters: current.counters.json(), ID: current.id,
		}); err != nil {
			return fmt.Errorf("update run counters: %w", err)
		}
	}
	return nil
}

// parseLastMod accepts the W3C datetime forms sitemaps use; anything else is
// left empty rather than guessed.
func parseLastMod(raw string) sql.NullTime {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04Z07:00", "2006-01-02T15:04:05", "2006-01-02"} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return sql.NullTime{Time: parsed, Valid: true}
		}
	}
	return sql.NullTime{}
}
