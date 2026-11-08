package registry

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/store"
	"github.com/mbcoward3/grocery-router/internal/testdatabase"
)

func TestSyncRoundTripsAndRetiresRemovedSites(t *testing.T) {
	ctx := context.Background()
	db := testdatabase.Open(t)
	queries := store.New(db)

	first, err := Parse([]byte(validRegistry))
	if err != nil {
		t.Fatal(err)
	}
	result, err := Sync(ctx, db, first)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if result != (SyncResult{Listed: 2, Active: 1}) {
		t.Fatalf("result = %+v", result)
	}
	stored, err := ReadStored(ctx, queries)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stored, first) {
		t.Fatalf("stored registry differs:\n got %+v\nwant %+v", stored, first)
	}

	// Give the soon-to-be-removed site history that must survive deactivation.
	page, err := queries.UpsertIngestSourcePage(ctx, store.UpsertIngestSourcePageParams{
		SiteID: "other-kitchen", Url: "https://other.example.org/soup/", DiscoveredVia: "https://other.example.org/sitemap.xml",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := queries.PauseIngestSite(ctx, store.PauseIngestSiteParams{SiteID: "example-kitchen", Reason: "bot_challenge", Detail: "challenge page"}); err != nil {
		t.Fatal(err)
	}

	// Remove other-kitchen and move one of example-kitchen's hosts to a new site.
	second := strings.Replace(validRegistry, "hosts: [www.example.com, example.com]", "hosts: [www.example.com]", 1)
	second = second[:strings.Index(second, "  - id: other-kitchen")] + `  - id: new-kitchen
    name: New Kitchen
    hosts: [example.com]
    sitemaps: []
    measurement_convention: us
    active: false
`
	registry, err := Parse([]byte(second))
	if err != nil {
		t.Fatal(err)
	}
	result, err = Sync(ctx, db, registry)
	if err != nil {
		t.Fatalf("second Sync: %v", err)
	}
	if result != (SyncResult{Listed: 2, Active: 1, Retired: 1}) {
		t.Fatalf("second result = %+v", result)
	}

	retired, err := queries.GetIngestSite(ctx, "other-kitchen")
	if err != nil {
		t.Fatal(err)
	}
	if retired.Active || !retired.RemovedAt.Valid {
		t.Fatalf("removed site = %+v, want inactive and removed", retired)
	}
	if _, err := queries.GetIngestSourcePage(ctx, page.ID); err != nil {
		t.Fatalf("removed site's page was lost: %v", err)
	}
	owner, err := queries.GetIngestSiteByHost(ctx, "example.com")
	if err != nil || owner.ID != "new-kitchen" {
		t.Fatalf("example.com owner = %q, %v", owner.ID, err)
	}
	if _, err := queries.GetIngestSiteByHost(ctx, "other.example.org"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("removed site's host still resolves: %v", err)
	}
	if _, err := queries.GetOpenIngestSitePause(ctx, "example-kitchen"); err != nil {
		t.Fatalf("sync cleared a pause: %v", err)
	}
	stored, err = ReadStored(ctx, queries)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stored, registry) {
		t.Fatalf("stored registry differs after resync:\n got %+v\nwant %+v", stored, registry)
	}

	// Listing a removed site again restores it; a third sync with the same file changes nothing.
	if _, err := Sync(ctx, db, first); err != nil {
		t.Fatal(err)
	}
	restored, err := queries.GetIngestSite(ctx, "other-kitchen")
	if err != nil || restored.RemovedAt.Valid {
		t.Fatalf("relisted site = %+v, %v", restored, err)
	}
}
