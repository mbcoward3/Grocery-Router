package stage

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/mbcoward3/grocery-router/internal/catalog/ingeststore"
	"github.com/mbcoward3/grocery-router/internal/store"
)

// Resume closes a site's open pause with an operator note
// (catalog-site-resume). It reports whether a pause was open.
func Resume(ctx context.Context, db *sql.DB, siteID, note string) (bool, error) {
	var resumed bool
	err := withLock(ctx, db, lockResume, func() error {
		queries := store.New(db)
		if _, err := queries.GetIngestSite(ctx, siteID); err != nil {
			return fmt.Errorf("site %q is not in the synced registry: %w", siteID, err)
		}
		var err error
		resumed, err = (&ingeststore.Pauses{Queries: queries}).Resume(ctx, siteID, note)
		return err
	})
	return resumed, err
}

// SiteReport holds acquisition metrics for one site.
type SiteReport struct {
	SiteID      string
	Active      bool
	Discovered  int64
	Queued      int64
	Fetched     int64
	NoRecipe    int64
	Failed      int64
	SourceNodes int64
	// DistinctNodes counts distinct content digests among source nodes.
	DistinctNodes int64
	// Pause is the open pause, empty when the site is not paused.
	Pause string
}

// Report returns acquisition metrics for every listed site in registry order.
func Report(ctx context.Context, db *sql.DB) ([]SiteReport, error) {
	queries := store.New(db)
	sites, err := queries.ListIngestSites(ctx)
	if err != nil {
		return nil, fmt.Errorf("list sites: %w", err)
	}
	pages, err := queries.CountIngestSourcePagesByState(ctx)
	if err != nil {
		return nil, fmt.Errorf("count pages: %w", err)
	}
	noRecipe, err := queries.CountIngestNoRecipePagesBySite(ctx)
	if err != nil {
		return nil, fmt.Errorf("count no_recipe pages: %w", err)
	}
	nodes, err := queries.CountIngestSourceNodesBySite(ctx)
	if err != nil {
		return nil, fmt.Errorf("count source nodes: %w", err)
	}
	pauses, err := queries.ListOpenIngestSitePauses(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pauses: %w", err)
	}
	reports := make([]SiteReport, 0, len(sites))
	index := map[string]int{}
	for _, site := range sites {
		if site.RemovedAt.Valid {
			continue
		}
		index[site.ID] = len(reports)
		reports = append(reports, SiteReport{SiteID: site.ID, Active: site.Active})
	}
	for _, row := range pages {
		position, ok := index[row.SiteID]
		if !ok {
			continue
		}
		report := &reports[position]
		report.Discovered += row.PageCount
		switch row.State {
		case "queued":
			report.Queued += row.PageCount
		case "fetched":
			report.Fetched += row.PageCount
		case "failed":
			report.Failed += row.PageCount
		}
	}
	for _, row := range noRecipe {
		if position, ok := index[row.SiteID]; ok {
			reports[position].NoRecipe = row.PageCount
		}
	}
	for _, row := range nodes {
		if position, ok := index[row.SiteID]; ok {
			reports[position].SourceNodes, reports[position].DistinctNodes = row.NodeCount, row.DistinctDigestCount
		}
	}
	for _, pause := range pauses {
		if position, ok := index[pause.SiteID]; ok {
			reports[position].Pause = fmt.Sprintf("%s since %s: %s", pause.Reason, pause.PausedAt.UTC().Format("2006-01-02 15:04Z"), pause.Detail)
		}
	}
	return reports, nil
}

// WriteReport prints reports as a table followed by any pauses.
func WriteReport(out io.Writer, reports []SiteReport) error {
	table := tabwriter.NewWriter(out, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(table, "site\tactive\tdiscovered\tqueued\tfetched\tno_recipe\tfailed\tsource_nodes\tdistinct_nodes\tpaused\t")
	var pauses []string
	for _, report := range reports {
		paused := "no"
		if report.Pause != "" {
			paused = "yes"
			pauses = append(pauses, report.SiteID+": "+report.Pause)
		}
		fmt.Fprintf(table, "%s\t%t\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s\t\n", report.SiteID, report.Active,
			report.Discovered, report.Queued, report.Fetched, report.NoRecipe, report.Failed,
			report.SourceNodes, report.DistinctNodes, paused)
	}
	if err := table.Flush(); err != nil {
		return err
	}
	if len(pauses) > 0 {
		fmt.Fprintf(out, "\npaused sites (resume with catalog-site-resume):\n  %s\n", strings.Join(pauses, "\n  "))
	}
	return nil
}
