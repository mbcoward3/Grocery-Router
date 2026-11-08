package database_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/mbcoward3/grocery-router/internal/database"
	"github.com/mbcoward3/grocery-router/internal/testdatabase"
)

var ingestTables = []string{
	"sites", "site_hosts", "site_sitemaps", "site_pauses", "host_rate_limits", "runs",
	"source_pages", "fetch_attempts", "evidence", "evidence_jsonld_blocks", "source_nodes",
}

func TestCatalogIngestMigrationRollsBackAndReapplies(t *testing.T) {
	ctx := context.Background()
	db := testdatabase.Open(t)
	assertIngestTables(t, db)

	// The embedded set is the checked-in directory; read it from disk to run Down.
	provider, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.DownTo(ctx, 3); err != nil {
		t.Fatalf("roll back 00004: %v", err)
	}
	var schemas int
	if err := db.QueryRow("SELECT count(*) FROM information_schema.schemata WHERE schema_name = 'ingest'").Scan(&schemas); err != nil {
		t.Fatal(err)
	}
	if schemas != 0 {
		t.Fatal("ingest schema remains after rollback")
	}
	var recipes int
	if err := db.QueryRow("SELECT count(*) FROM information_schema.tables WHERE table_name = 'recipes'").Scan(&recipes); err != nil || recipes != 1 {
		t.Fatalf("rollback touched household tables: recipes tables = %d, %v", recipes, err)
	}
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("reapply 00004: %v", err)
	}
	assertIngestTables(t, db)
}

func TestIngestSchemaHasNoHouseholdColumns(t *testing.T) {
	db := testdatabase.Open(t)
	var columns int
	err := db.QueryRow(`SELECT count(*) FROM information_schema.columns
		WHERE table_schema = 'ingest' AND column_name = 'household_id'`).Scan(&columns)
	if err != nil {
		t.Fatal(err)
	}
	if columns != 0 {
		t.Fatalf("ingest schema has %d household_id columns, want 0 (invariant 1)", columns)
	}
}

func assertIngestTables(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, table := range ingestTables {
		var count int
		err := db.QueryRow(`SELECT count(*) FROM information_schema.tables
			WHERE table_schema = 'ingest' AND table_name = $1`, table).Scan(&count)
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("ingest.%s missing", table)
		}
	}
}
