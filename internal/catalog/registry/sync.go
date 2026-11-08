package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mbcoward3/grocery-router/internal/store"
)

// SyncResult summarizes one registry sync.
type SyncResult struct {
	Listed  int // sites in the registry
	Active  int // listed sites marked active
	Retired int // previously listed sites no longer in the registry, now inactive
}

// Sync writes a validated registry into ingest.sites in one transaction. Sites missing from
// the registry are deactivated, never deleted, so their pages, evidence, and pauses remain.
// Pauses are never changed by a sync.
func Sync(ctx context.Context, db *sql.DB, registry Registry) (SyncResult, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return SyncResult{}, fmt.Errorf("begin registry sync: %w", err)
	}
	defer tx.Rollback()
	// Serialize concurrent syncs; the lock is released at commit or rollback.
	if _, err := tx.ExecContext(ctx, "LOCK TABLE ingest.sites IN SHARE ROW EXCLUSIVE MODE"); err != nil {
		return SyncResult{}, fmt.Errorf("lock site registry: %w", err)
	}
	queries := store.New(tx)

	existing, err := queries.ListIngestSites(ctx)
	if err != nil {
		return SyncResult{}, fmt.Errorf("list stored sites: %w", err)
	}
	// Hosts are re-inserted below, so a host may move between sites in one sync.
	if err := queries.DeleteAllIngestSiteHosts(ctx); err != nil {
		return SyncResult{}, fmt.Errorf("clear site hosts: %w", err)
	}

	result := SyncResult{Listed: len(registry.Sites)}
	listed := make(map[string]bool, len(registry.Sites))
	for position, site := range registry.Sites {
		listed[site.ID] = true
		if site.Active {
			result.Active++
		}
		params, err := siteParams(site, position)
		if err != nil {
			return SyncResult{}, err
		}
		if err := queries.UpsertIngestSite(ctx, params); err != nil {
			return SyncResult{}, fmt.Errorf("store site %s: %w", site.ID, err)
		}
		for hostPosition, host := range site.Hosts {
			if err := queries.InsertIngestSiteHost(ctx, store.InsertIngestSiteHostParams{
				Host: host, SiteID: site.ID, Position: int32(hostPosition),
			}); err != nil {
				return SyncResult{}, fmt.Errorf("store host %s for site %s: %w", host, site.ID, err)
			}
		}
		if err := queries.DeleteIngestSiteSitemaps(ctx, site.ID); err != nil {
			return SyncResult{}, fmt.Errorf("clear sitemaps for site %s: %w", site.ID, err)
		}
		for sitemapPosition, sitemap := range site.Sitemaps {
			if err := queries.InsertIngestSiteSitemap(ctx, store.InsertIngestSiteSitemapParams{
				SiteID: site.ID, Position: int32(sitemapPosition), Url: sitemap,
			}); err != nil {
				return SyncResult{}, fmt.Errorf("store sitemap %s for site %s: %w", sitemap, site.ID, err)
			}
		}
	}
	for _, stored := range existing {
		if listed[stored.ID] {
			continue
		}
		retired, err := queries.RetireIngestSite(ctx, stored.ID)
		if err != nil {
			return SyncResult{}, fmt.Errorf("retire site %s: %w", stored.ID, err)
		}
		result.Retired += int(retired)
	}
	if err := tx.Commit(); err != nil {
		return SyncResult{}, fmt.Errorf("commit registry sync: %w", err)
	}
	return result, nil
}

func siteParams(site Site, position int) (store.UpsertIngestSiteParams, error) {
	include, err := json.Marshal(nonNil(site.Include))
	if err != nil {
		return store.UpsertIngestSiteParams{}, fmt.Errorf("encode include patterns for %s: %w", site.ID, err)
	}
	exclude, err := json.Marshal(nonNil(site.Exclude))
	if err != nil {
		return store.UpsertIngestSiteParams{}, fmt.Errorf("encode exclude patterns for %s: %w", site.ID, err)
	}
	params := store.UpsertIngestSiteParams{
		ID:                    site.ID,
		AttributionName:       site.Name,
		RegistryPosition:      int32(position),
		IncludePatterns:       include,
		ExcludePatterns:       exclude,
		MeasurementConvention: site.MeasurementConvention,
		Active:                site.Active,
	}
	if site.MinInterval > 0 {
		params.MinIntervalMs = sql.NullInt32{Int32: int32(site.MinInterval / time.Millisecond), Valid: true}
	}
	if site.Eligibility != nil {
		params.EligibilityIntendedUse = sql.NullString{String: site.Eligibility.IntendedUse, Valid: true}
		params.EligibilityReviewedOn = sql.NullTime{Time: site.Eligibility.ReviewedOn, Valid: true}
		params.EligibilityNotes = site.Eligibility.Notes
	}
	return params, nil
}

// ReadStored returns the sites the registry currently lists, as synced into ingest.sites, in
// registry order. Fetch stages read sites from the database, not from the YAML file.
func ReadStored(ctx context.Context, queries store.Querier) (Registry, error) {
	sites, err := queries.ListIngestSites(ctx)
	if err != nil {
		return Registry{}, fmt.Errorf("list stored sites: %w", err)
	}
	hosts, err := queries.ListIngestSiteHosts(ctx)
	if err != nil {
		return Registry{}, fmt.Errorf("list stored site hosts: %w", err)
	}
	sitemaps, err := queries.ListIngestSiteSitemaps(ctx)
	if err != nil {
		return Registry{}, fmt.Errorf("list stored sitemaps: %w", err)
	}
	hostsBySite := map[string][]string{}
	for _, host := range hosts {
		hostsBySite[host.SiteID] = append(hostsBySite[host.SiteID], host.Host)
	}
	sitemapsBySite := map[string][]string{}
	for _, sitemap := range sitemaps {
		sitemapsBySite[sitemap.SiteID] = append(sitemapsBySite[sitemap.SiteID], sitemap.Url)
	}

	var registry Registry
	for _, stored := range sites {
		if stored.RemovedAt.Valid {
			continue
		}
		site := Site{
			ID:                    stored.ID,
			Name:                  stored.AttributionName,
			Hosts:                 nonNil(hostsBySite[stored.ID]),
			Sitemaps:              nonNil(sitemapsBySite[stored.ID]),
			MeasurementConvention: stored.MeasurementConvention,
			Active:                stored.Active,
		}
		if err := json.Unmarshal(stored.IncludePatterns, &site.Include); err != nil {
			return Registry{}, fmt.Errorf("decode include patterns for %s: %w", stored.ID, err)
		}
		if err := json.Unmarshal(stored.ExcludePatterns, &site.Exclude); err != nil {
			return Registry{}, fmt.Errorf("decode exclude patterns for %s: %w", stored.ID, err)
		}
		if site.include, err = compilePatterns(site.Include); err != nil {
			return Registry{}, fmt.Errorf("site %s include: %w", stored.ID, err)
		}
		if site.exclude, err = compilePatterns(site.Exclude); err != nil {
			return Registry{}, fmt.Errorf("site %s exclude: %w", stored.ID, err)
		}
		if stored.MinIntervalMs.Valid {
			site.MinInterval = time.Duration(stored.MinIntervalMs.Int32) * time.Millisecond
		}
		if stored.EligibilityIntendedUse.Valid {
			site.Eligibility = &Eligibility{
				IntendedUse: stored.EligibilityIntendedUse.String,
				ReviewedOn:  stored.EligibilityReviewedOn.Time,
				Notes:       stored.EligibilityNotes,
			}
		}
		registry.Sites = append(registry.Sites, site)
	}
	return registry, nil
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
