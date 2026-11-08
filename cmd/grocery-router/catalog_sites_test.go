package main

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/testdatabase"
)

func TestCatalogSitesSyncsRepositoryRegistry(t *testing.T) {
	db, databaseURL := testdatabase.OpenWithURL(t)
	root := filepath.Join("..", "..")
	if err := run([]string{"catalog-sites", "--check", "--root", root}, io.Discard, io.Discard); err != nil {
		t.Fatalf("catalog-sites --check: %v", err)
	}
	for range 2 {
		if err := run([]string{"catalog-sites", "--root", root, "--database-url", databaseURL}, io.Discard, io.Discard); err != nil {
			t.Fatalf("catalog-sites: %v", err)
		}
	}
	var sites, active int
	if err := db.QueryRow("SELECT count(*), count(*) FILTER (WHERE active) FROM ingest.sites").Scan(&sites, &active); err != nil {
		t.Fatal(err)
	}
	if sites != 13 || active != 4 {
		t.Fatalf("synced %d sites (%d active), want 13 (4 active)", sites, active)
	}
}
