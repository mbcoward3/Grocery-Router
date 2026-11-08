package main

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/testdatabase"
)

// The network commands are covered offline in internal/catalog/stage; these
// commands never touch the network.
func TestCatalogOfflineCommands(t *testing.T) {
	db, databaseURL := testdatabase.OpenWithURL(t)
	root := filepath.Join("..", "..")
	cli := func(args ...string) error {
		return run(append(args, "--database-url", databaseURL), io.Discard, io.Discard)
	}
	if err := cli("catalog-sites", "--root", root); err != nil {
		t.Fatalf("catalog-sites: %v", err)
	}
	if err := cli("catalog-report"); err != nil {
		t.Fatalf("catalog-report: %v", err)
	}
	if err := cli("catalog-fetch", "--replay"); err != nil {
		t.Fatalf("catalog-fetch --replay: %v", err)
	}
	if err := cli("catalog-fetch", "--limit", "-1"); err == nil {
		t.Fatal("negative --limit accepted")
	}

	if _, err := db.Exec(`INSERT INTO ingest.site_pauses (site_id, reason, detail) VALUES ('budget-bytes', 'operator', 'test pause')`); err != nil {
		t.Fatal(err)
	}
	if err := cli("catalog-site-resume", "budget-bytes"); err == nil {
		t.Fatal("resume without --note accepted")
	}
	if err := cli("catalog-site-resume", "budget-bytes", "--note", "checked by hand"); err != nil {
		t.Fatalf("catalog-site-resume: %v", err)
	}
	var open int
	if err := db.QueryRow(`SELECT count(*) FROM ingest.site_pauses WHERE resumed_at IS NULL`).Scan(&open); err != nil || open != 0 {
		t.Fatalf("open pauses = %d (%v)", open, err)
	}
	if err := cli("catalog-site-resume", "no-such-site", "--note", "x"); err == nil {
		t.Fatal("resume of unknown site accepted")
	}
}
