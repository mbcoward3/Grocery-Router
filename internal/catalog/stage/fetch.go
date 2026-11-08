package stage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/mbcoward3/grocery-router/internal/catalog/acquire"
	"github.com/mbcoward3/grocery-router/internal/catalog/evidence"
	"github.com/mbcoward3/grocery-router/internal/catalog/extract"
	"github.com/mbcoward3/grocery-router/internal/catalog/ingeststore"
	"github.com/mbcoward3/grocery-router/internal/catalog/registry"
	"github.com/mbcoward3/grocery-router/internal/store"
)

// FetchOptions scopes a fetch run.
type FetchOptions struct {
	// Site limits the run to one site ID; empty means every fetchable site.
	Site string
	// Limit caps pages attempted per site; 0 means no cap.
	Limit int
}

// FetchSummary is the result of a fetch run.
type FetchSummary struct {
	RunID    int64
	Counters map[string]int64
	Paused   map[string]acquire.Pause
	Skipped  map[string]string
}

const pageBatch = 100

// Fetch fetches queued pages for each fetchable site, stores evidence with
// its JSON-LD blocks and source nodes, and advances each page's queue state.
// Fetched pages are never fetched again, so an interrupted run resumes where
// it stopped. A pause stops the site for this and later runs.
func Fetch(ctx context.Context, config Config, options FetchOptions) (FetchSummary, error) {
	summary := FetchSummary{Paused: map[string]acquire.Pause{}}
	err := withLock(ctx, config.DB, lockFetch, func() error {
		sites, skipped, err := fetchableSites(ctx, config.DB, options.Site)
		if err != nil {
			return err
		}
		summary.Skipped = skipped
		current, err := startRun(ctx, config.DB, "catalog-fetch", options.Site,
			map[string]any{"site": options.Site, "limit": options.Limit})
		if err != nil {
			return err
		}
		summary.RunID = current.id
		runErr := fetchSites(ctx, config, current, sites, options.Limit, &summary)
		summary.Counters = current.counters.Map()
		return current.finish(ctx, runErr)
	})
	return summary, err
}

func fetchSites(ctx context.Context, config Config, current *run, sites []registry.Site, limit int, summary *FetchSummary) error {
	client, err := config.client(current.id)
	if err != nil {
		return err
	}
	queries := store.New(config.DB)
	for _, site := range sites {
		attempted := 0
	pages:
		for limit == 0 || attempted < limit {
			batch := pageBatch
			if limit > 0 {
				batch = min(batch, limit-attempted)
			}
			due, err := queries.ListDueIngestSourcePages(ctx, store.ListDueIngestSourcePagesParams{SiteID: site.ID, RowLimit: int32(batch)})
			if err != nil {
				return fmt.Errorf("list queued pages for %s: %w", site.ID, err)
			}
			if len(due) == 0 {
				break
			}
			for _, page := range due {
				attempted++
				pause, err := fetchPage(ctx, config, client, queries, current, site, page)
				if err != nil {
					return err
				}
				if pause != nil {
					summary.Paused[site.ID] = *pause
					current.counters.Add("paused_sites", 1)
					config.progress("fetch %s: site paused (%s: %s)", site.ID, pause.Reason, pause.Detail)
					break pages
				}
			}
			if err := queries.UpdateIngestRunCounters(ctx, store.UpdateIngestRunCountersParams{
				Counters: current.counters.json(), ID: current.id,
			}); err != nil {
				return fmt.Errorf("update run counters: %w", err)
			}
		}
		config.progress("fetch %s: %d pages attempted", site.ID, attempted)
	}
	return nil
}

// fetchPage fetches and settles one queued page. It returns the site's pause
// when the site is (or becomes) paused; the page then stays queued.
func fetchPage(ctx context.Context, config Config, client *acquire.Client, queries *store.Queries, current *run, site registry.Site, page store.IngestSourcePage) (*acquire.Pause, error) {
	result, err := client.GetPage(ctx, acquireSite(site), page.ID, page.Url)
	if err != nil {
		return nil, err
	}
	fail := func(counter, reason string) error {
		current.counters.Add(counter, 1)
		config.progress("fetch %s: %s %s", site.ID, counter, page.Url)
		if err := queries.MarkIngestSourcePageFailed(ctx, store.MarkIngestSourcePageFailedParams{
			StateReason: sql.NullString{String: truncate(reason, 1000), Valid: true}, ID: page.ID,
		}); err != nil {
			return fmt.Errorf("mark page %d failed: %w", page.ID, err)
		}
		return nil
	}
	errText := func() string {
		if result.Err != nil {
			return result.Err.Error()
		}
		return string(result.Outcome)
	}
	switch result.Outcome {
	case acquire.OutcomePaused:
		return result.Pause, nil
	case acquire.OutcomeDisallowed:
		return nil, fail("disallowed", "robots_disallowed")
	case acquire.OutcomeRejected:
		return nil, fail("rejected", "rejected: "+errText())
	case acquire.OutcomeFetched:
	default:
		return nil, fail("failed", errText())
	}

	record, err := evidence.New(site.ID, result.Response)
	if err != nil {
		return nil, fail("failed", err.Error())
	}
	record.FetchAttemptID, record.SourcePageID = result.AttemptID, page.ID
	if !ingeststore.StorableURL(record.FinalURL) || !ingeststore.StorableURL(record.RequestedURL) {
		return nil, fail("rejected", "rejected: final URL cannot be stored: "+record.FinalURL)
	}
	// Extraction runs on the stored, compressed form so replay sees exactly
	// what this run saw.
	extraction, err := extract.FromEvidence(record, site.Hosts)
	if err != nil {
		return nil, fail("failed", "extract: "+err.Error())
	}
	if len(extraction.CanonicalLinks) > 0 {
		record.CanonicalLink = extraction.CanonicalLinks[0]
	}
	if _, err := ingeststore.SaveFetchedPage(ctx, config.DB, record, extraction); err != nil {
		if ctx.Err() != nil {
			return nil, err
		}
		return nil, fail("store_failed", "store: "+err.Error())
	}
	current.counters.Add("fetched", 1)
	current.counters.Add("source_nodes", int64(len(extraction.Nodes)))
	current.counters.Add("jsonld_warnings", int64(len(extraction.Warnings)))
	if extraction.Outcome == extract.OutcomeNoRecipe {
		current.counters.Add("no_recipe", 1)
	}
	return nil, nil
}
