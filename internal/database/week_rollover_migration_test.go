package database_test

import (
	"context"
	"os"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/mbcoward3/grocery-router/internal/database"
	"github.com/mbcoward3/grocery-router/internal/tenant"
)

func TestWeekRolloverSettingsMigration(t *testing.T) {
	ctx := context.Background()
	db := migratedDB(t)

	var day, minutes int
	var mode string
	if err := db.QueryRow("SELECT week_start_day, rollover_minutes, rollover_mode FROM households WHERE id = $1", tenant.CowardHouseholdID).
		Scan(&day, &minutes, &mode); err != nil {
		t.Fatal(err)
	}
	if day != 0 || minutes != 0 || mode != "automatic" {
		t.Fatalf("default rollover = %d, %d, %q; want Sunday 00:00 automatic", day, minutes, mode)
	}
	for _, statement := range []string{
		"UPDATE households SET week_start_day = 7 WHERE id = $1",
		"UPDATE households SET week_start_day = -1 WHERE id = $1",
		"UPDATE households SET rollover_minutes = 1440 WHERE id = $1",
		"UPDATE households SET rollover_mode = 'weekly' WHERE id = $1",
	} {
		_, err := db.Exec(statement, tenant.CowardHouseholdID)
		assertErrorContains(t, err, "violates check constraint")
	}
	mustExec(t, db, "UPDATE households SET week_start_day = 6, rollover_minutes = 1439, rollover_mode = 'manual' WHERE id = $1", tenant.CowardHouseholdID)

	// Weeks may start on any day, and two weeks may share a start date.
	mustExec(t, db, `INSERT INTO weeks (household_id, starts_on, opened_at, closed_at)
		VALUES ($1, '2026-08-21', now(), now())`, tenant.CowardHouseholdID)
	mustExec(t, db, "INSERT INTO weeks (household_id, starts_on, opened_at) VALUES ($1, '2026-08-21', now())", tenant.CowardHouseholdID)

	provider, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("migrations"))
	if err != nil {
		t.Fatal(err)
	}
	// Rolling back refuses to drop weeks the earlier schema cannot hold.
	if _, err := provider.DownTo(ctx, 11); err == nil {
		t.Fatal("rolled back with weeks that share a Friday start date")
	}
	mustExec(t, db, "DELETE FROM weeks WHERE household_id = $1", tenant.CowardHouseholdID)
	mustExec(t, db, "INSERT INTO weeks (household_id, starts_on, opened_at) VALUES ($1, '2026-08-16', now())", tenant.CowardHouseholdID)
	if _, err := provider.DownTo(ctx, 11); err != nil {
		t.Fatalf("roll back week rollover settings: %v", err)
	}
	_, err = db.Exec(`INSERT INTO weeks (household_id, starts_on, opened_at, closed_at)
		VALUES ($1, '2026-08-21', now(), now())`, tenant.CowardHouseholdID)
	assertErrorContains(t, err, "violates check constraint")
	_, err = db.Exec(`INSERT INTO weeks (household_id, starts_on, opened_at, closed_at)
		VALUES ($1, '2026-08-16', now(), now())`, tenant.CowardHouseholdID)
	assertErrorContains(t, err, "weeks_household_start_key")
	var columns int
	if err := db.QueryRow(`SELECT count(*) FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'households'
		AND column_name IN ('week_start_day', 'rollover_minutes', 'rollover_mode')`).Scan(&columns); err != nil || columns != 0 {
		t.Fatalf("rollover columns after rollback = %d, %v", columns, err)
	}

	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("reapply week rollover settings: %v", err)
	}
	mustExec(t, db, `INSERT INTO weeks (household_id, starts_on, opened_at, closed_at)
		VALUES ($1, '2026-08-16', now(), now())`, tenant.CowardHouseholdID)
}
