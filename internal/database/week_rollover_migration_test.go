package database_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/mbcoward3/grocery-router/internal/database"
	"github.com/mbcoward3/grocery-router/internal/tenant"
)

func TestWeekRolloverSettingsMigration(t *testing.T) {
	ctx := context.Background()
	db := migratedDB(t)
	provider, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("migrations"))
	if err != nil {
		t.Fatal(err)
	}
	// 00013 drops the rollover settings again; this checks 00012 on its own.
	if _, err := provider.DownTo(ctx, 12); err != nil {
		t.Fatal(err)
	}

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
	mustExec(t, db, `INSERT INTO weeks (household_id, starts_on, opened_at, closed_at, ends_on)
		VALUES ($1, '2026-08-21', now(), now(), '2026-08-27')`, tenant.CowardHouseholdID)
	mustExec(t, db, "INSERT INTO weeks (household_id, starts_on, opened_at) VALUES ($1, '2026-08-21', now())", tenant.CowardHouseholdID)

	// A closed week has its last date and an open one has none.
	for _, statement := range []string{
		`INSERT INTO weeks (household_id, starts_on, opened_at, closed_at) VALUES ($1, '2026-08-14', now(), now())`,
		`UPDATE weeks SET ends_on = '2026-08-27' WHERE household_id = $1 AND closed_at IS NULL`,
	} {
		_, err := db.Exec(statement, tenant.CowardHouseholdID)
		assertErrorContains(t, err, "weeks_ends_on_when_closed")
	}
	_, err = db.Exec(`UPDATE weeks SET ends_on = '2026-8-27' WHERE household_id = $1 AND closed_at IS NOT NULL`, tenant.CowardHouseholdID)
	assertErrorContains(t, err, "violates check constraint")

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
	// A week closed before ends_on existed: 2026-08-16 05:00 UTC is Sunday midnight in Chicago.
	mustExec(t, db, "UPDATE households SET timezone = 'America/Chicago' WHERE id = $1", tenant.CowardHouseholdID)
	closedID := mustInsertID(t, db, `INSERT INTO weeks (household_id, starts_on, opened_at, closed_at)
		VALUES ($1, '2026-08-09', '2026-08-09T05:00:00Z', '2026-08-16T05:00:00Z')`, tenant.CowardHouseholdID)
	var columns int
	if err := db.QueryRow(`SELECT count(*) FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'households'
		AND column_name IN ('week_start_day', 'rollover_minutes', 'rollover_mode')`).Scan(&columns); err != nil || columns != 0 {
		t.Fatalf("rollover columns after rollback = %d, %v", columns, err)
	}

	if _, err := provider.UpTo(ctx, 12); err != nil {
		t.Fatalf("reapply week rollover settings: %v", err)
	}
	// The backfill uses the household's zone: the week's last local day is Saturday.
	var endsOn sql.NullString
	if err := db.QueryRow("SELECT ends_on FROM weeks WHERE id = $1", closedID).Scan(&endsOn); err != nil || endsOn.String != "2026-08-15" {
		t.Fatalf("backfilled ends_on = %v, %v; want 2026-08-15", endsOn, err)
	}
	var openEndsOn sql.NullString
	if err := db.QueryRow("SELECT ends_on FROM weeks WHERE household_id = $1 AND closed_at IS NULL", tenant.CowardHouseholdID).Scan(&openEndsOn); err != nil || openEndsOn.Valid {
		t.Fatalf("open week ends_on = %v, %v; want NULL", openEndsOn, err)
	}
	mustExec(t, db, `INSERT INTO weeks (household_id, starts_on, opened_at, closed_at, ends_on)
		VALUES ($1, '2026-08-16', now(), now(), '2026-08-22')`, tenant.CowardHouseholdID)
}

// Plans are open-ended (00013): the rollover settings and automatic close time go, an open plan
// stays open, and the migration rolls back and reapplies.
func TestOpenEndedPlansMigration(t *testing.T) {
	ctx := context.Background()
	db := migratedDB(t)
	provider, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.DownTo(ctx, 12); err != nil {
		t.Fatal(err)
	}
	// An open week whose automatic close time had passed but was not yet noticed.
	openID := mustInsertID(t, db, `INSERT INTO weeks (household_id, starts_on, opened_at, auto_close_at)
		VALUES ($1, '2026-08-16', now() - interval '10 days', now() - interval '3 days')`, tenant.CowardHouseholdID)
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	removed := func() int {
		t.Helper()
		var columns int
		if err := db.QueryRow(`SELECT count(*) FROM information_schema.columns
			WHERE table_schema = current_schema() AND (
				(table_name = 'households' AND column_name IN ('week_start_day', 'rollover_minutes', 'rollover_mode'))
				OR (table_name = 'weeks' AND column_name = 'auto_close_at'))`).Scan(&columns); err != nil {
			t.Fatal(err)
		}
		return columns
	}
	if got := removed(); got != 0 {
		t.Fatalf("rollover columns after 00013 = %d, want 0", got)
	}
	var open bool
	if err := db.QueryRow("SELECT closed_at IS NULL FROM weeks WHERE id = $1", openID).Scan(&open); err != nil || !open {
		t.Fatalf("open week after 00013: open %t, %v", open, err)
	}

	if _, err := provider.DownTo(ctx, 12); err != nil {
		t.Fatalf("roll back open-ended plans: %v", err)
	}
	if got := removed(); got != 4 {
		t.Fatalf("rollover columns after rollback = %d, want 4", got)
	}
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("reapply open-ended plans: %v", err)
	}
}
