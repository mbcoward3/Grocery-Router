package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"

	"github.com/mbcoward3/grocery-router/internal/catalog/stage"
	"github.com/mbcoward3/grocery-router/internal/database"
)

// catalogFetchCommands is embedded into cli so the slice 2 acquisition commands live in this file.
type catalogFetchCommands struct {
	CatalogDiscover   catalogDiscoverCommand   `cmd:"" name:"catalog-discover" help:"Queue pages from the registered sitemaps of fetchable catalog sites."`
	CatalogFetch      catalogFetchCommand      `cmd:"" name:"catalog-fetch" help:"Fetch queued catalog pages and extract JSON-LD Recipe nodes (resumable)."`
	CatalogSiteResume catalogSiteResumeCommand `cmd:"" name:"catalog-site-resume" help:"Resume a paused catalog site with an operator note."`
	CatalogReport     catalogReportCommand     `cmd:"" name:"catalog-report" help:"Report catalog acquisition metrics per site."`
}

// CatalogNetworkConfig names the operator contact sent in the user agent.
type CatalogNetworkConfig struct {
	Contact string `help:"Public support address named in the user agent." default:"support@matthewcoward.com" env:"GROCERY_ROUTER_CATALOG_CONTACT"`
}

type catalogDiscoverCommand struct {
	DatabaseConfig
	CatalogNetworkConfig
	Site string `help:"Only discover this site ID."`
}

func (command *catalogDiscoverCommand) Run() error {
	return withCatalogDatabase(command.DatabaseURL, func(ctx context.Context, config stage.Config) error {
		config.Contact = command.Contact
		summary, err := stage.Discover(ctx, config, stage.DiscoverOptions{Site: command.Site})
		printSkipped(summary.Skipped)
		for _, site := range summary.Sites {
			report := site.Report
			fmt.Printf("%s: %d pages (%d new, %d known, %d owned by another site); %d sitemap files; ignored %d unregistered-host, %d excluded, %d invalid, %d duplicate entries; %d cycles, %d too deep\n",
				site.SiteID, len(report.Entries), site.New, site.Known, site.Conflicts, len(report.Files),
				report.UnregisteredHost, report.Excluded, report.Invalid, report.Duplicates, report.Cycles, report.TooDeep)
			if report.Paused != nil {
				fmt.Printf("%s: PAUSED (%s): %s\n", site.SiteID, report.Paused.Reason, report.Paused.Detail)
			}
		}
		if err != nil {
			return err
		}
		fmt.Printf("discover run %d complete\n", summary.RunID)
		return nil
	})
}

type catalogFetchCommand struct {
	DatabaseConfig
	CatalogNetworkConfig
	Site   string `help:"Only fetch this site ID."`
	Limit  int    `help:"Maximum pages to attempt per site (0 = no limit)." default:"0"`
	Replay bool   `help:"Re-extract source nodes from stored evidence without network access."`
}

func (command *catalogFetchCommand) Run() error {
	if command.Limit < 0 {
		return fmt.Errorf("--limit must not be negative")
	}
	return withCatalogDatabase(command.DatabaseURL, func(ctx context.Context, config stage.Config) error {
		if command.Replay {
			summary, err := stage.Replay(ctx, config, stage.ReplayOptions{Site: command.Site})
			fmt.Printf("replay run %d: %d evidence re-extracted, %d matched, %d backfilled, %d mismatched, %d source nodes\n",
				summary.RunID, summary.Evidence, summary.Matched, summary.Backfilled, len(summary.Mismatched), summary.Nodes)
			return err
		}
		config.Contact = command.Contact
		summary, err := stage.Fetch(ctx, config, stage.FetchOptions{Site: command.Site, Limit: command.Limit})
		printSkipped(summary.Skipped)
		for site, pause := range summary.Paused {
			fmt.Printf("%s: PAUSED (%s): %s\n", site, pause.Reason, pause.Detail)
		}
		names := make([]string, 0, len(summary.Counters))
		for name := range summary.Counters {
			names = append(names, name)
		}
		sort.Strings(names)
		parts := make([]string, 0, len(names))
		for _, name := range names {
			parts = append(parts, fmt.Sprintf("%s=%d", name, summary.Counters[name]))
		}
		fmt.Printf("fetch run %d: %s\n", summary.RunID, strings.Join(parts, " "))
		return err
	})
}

type catalogSiteResumeCommand struct {
	DatabaseConfig
	Site string `arg:"" help:"Site ID to resume."`
	Note string `required:"" help:"Why the site may be fetched again (recorded with the pause)."`
}

func (command *catalogSiteResumeCommand) Run() error {
	return withCatalogDatabase(command.DatabaseURL, func(ctx context.Context, config stage.Config) error {
		resumed, err := stage.Resume(ctx, config.DB, command.Site, command.Note)
		if err != nil {
			return err
		}
		if resumed {
			fmt.Printf("%s resumed\n", command.Site)
		} else {
			fmt.Printf("%s was not paused\n", command.Site)
		}
		return nil
	})
}

type catalogReportCommand struct {
	DatabaseConfig
}

func (command *catalogReportCommand) Run() error {
	return withCatalogDatabase(command.DatabaseURL, func(ctx context.Context, config stage.Config) error {
		reports, err := stage.Report(ctx, config.DB)
		if err != nil {
			return err
		}
		return stage.WriteReport(os.Stdout, reports)
	})
}

// withCatalogDatabase migrates and opens the database and cancels the
// context on interrupt so a stage can record an interrupted run.
func withCatalogDatabase(databaseURL string, fn func(context.Context, stage.Config) error) error {
	if err := migrate(databaseURL); err != nil {
		return err
	}
	db, err := database.Open(databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return fn(ctx, stage.Config{
		DB:       db,
		Progress: func(format string, args ...any) { fmt.Fprintf(os.Stderr, format+"\n", args...) },
	})
}

func printSkipped(skipped map[string]string) {
	ids := make([]string, 0, len(skipped))
	for id := range skipped {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		fmt.Printf("%s: skipped (%s)\n", id, skipped[id])
	}
}
