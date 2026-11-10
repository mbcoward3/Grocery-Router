package database_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/pressly/goose/v3"

	"github.com/mbcoward3/grocery-router/internal/database"
	"github.com/mbcoward3/grocery-router/internal/tenant"
)

func TestOpenAndClosedWeeksMigrationBackfillsInHouseholdZone(t *testing.T) {
	ctx := context.Background()
	db := migratedDB(t)
	provider, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.DownTo(ctx, 10); err != nil {
		t.Fatalf("roll back open and closed weeks: %v", err)
	}

	chicago, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatal(err)
	}
	sundayOf := func(now time.Time) time.Time {
		year, month, day := now.Date()
		return time.Date(year, month, day-int(now.Weekday()), 0, 0, 0, 0, time.UTC)
	}
	const dateOnly = "2006-01-02"
	current := sundayOf(time.Now().In(chicago))
	previous := current.AddDate(0, 0, -7)
	const springForward = "2026-03-08"

	mustExec(t, db, "UPDATE households SET timezone = 'America/Chicago' WHERE id = $1", tenant.CowardHouseholdID)
	insertWeek := func(household, startsOn string) int64 {
		t.Helper()
		return mustInsertID(t, db, "INSERT INTO weeks (household_id, starts_on) VALUES ($1, $2)", household, startsOn)
	}
	springID := insertWeek(tenant.CowardHouseholdID, springForward)
	previousID := insertWeek(tenant.CowardHouseholdID, previous.Format(dateOnly))
	currentID := insertWeek(tenant.CowardHouseholdID, current.Format(dateOnly))

	// A household whose zone changed after it planned a later Sunday has two unexpired weeks.
	var other string
	if err := db.QueryRow("INSERT INTO households (name) VALUES ('Other') RETURNING id::text").Scan(&other); err != nil {
		t.Fatal(err)
	}
	otherCurrent := sundayOf(time.Now().UTC())
	otherCurrentID := insertWeek(other, otherCurrent.Format(dateOnly))
	otherNextID := insertWeek(other, otherCurrent.AddDate(0, 0, 7).Format(dateOnly))

	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("reapply open and closed weeks: %v", err)
	}

	type lifecycle struct {
		opened, autoClose time.Time
		closed            sql.NullTime
	}
	load := func(id int64) lifecycle {
		t.Helper()
		var row lifecycle
		if err := db.QueryRow("SELECT opened_at, auto_close_at, closed_at FROM weeks WHERE id = $1", id).
			Scan(&row.opened, &row.autoClose, &row.closed); err != nil {
			t.Fatal(err)
		}
		return row
	}
	localMidnight := func(date time.Time) time.Time {
		return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, chicago)
	}

	// The spring-forward week is an hour short in UTC but runs midnight to midnight locally.
	spring := load(springID)
	if !spring.opened.Equal(time.Date(2026, time.March, 8, 6, 0, 0, 0, time.UTC)) ||
		!spring.autoClose.Equal(time.Date(2026, time.March, 15, 5, 0, 0, 0, time.UTC)) ||
		!spring.closed.Valid || !spring.closed.Time.Equal(spring.autoClose) {
		t.Fatalf("spring-forward week = %+v", spring)
	}
	previousWeek := load(previousID)
	if !previousWeek.opened.Equal(localMidnight(previous)) || !previousWeek.autoClose.Equal(localMidnight(current)) ||
		!previousWeek.closed.Valid || !previousWeek.closed.Time.Equal(previousWeek.autoClose) {
		t.Fatalf("previous week = %+v", previousWeek)
	}
	currentWeek := load(currentID)
	if !currentWeek.opened.Equal(localMidnight(current)) || !currentWeek.autoClose.Equal(localMidnight(current.AddDate(0, 0, 7))) ||
		currentWeek.closed.Valid {
		t.Fatalf("current week = %+v", currentWeek)
	}
	var open int64
	if err := db.QueryRow("SELECT count(*) FROM weeks WHERE household_id = $1 AND closed_at IS NULL", tenant.CowardHouseholdID).Scan(&open); err != nil || open != 1 {
		t.Fatalf("open weeks = %d, %v; want 1", open, err)
	}

	if !load(otherCurrentID).closed.Valid || load(otherNextID).closed.Valid {
		t.Fatalf("other household weeks = %+v, %+v; want only the later one open", load(otherCurrentID), load(otherNextID))
	}

	_, err = db.Exec("INSERT INTO weeks (household_id, starts_on, opened_at) VALUES ($1, '2020-01-05', now())", tenant.CowardHouseholdID)
	assertErrorContains(t, err, "weeks_one_open_per_household")
	_, err = db.Exec(`INSERT INTO weeks (household_id, starts_on, opened_at, closed_at)
		VALUES ($1, '2020-01-05', now(), now() - interval '1 day')`, tenant.CowardHouseholdID)
	assertErrorContains(t, err, "weeks_closed_after_opened")

	// With backfilled and closed weeks present, the migration rolls back and reapplies.
	if _, err := provider.DownTo(ctx, 10); err != nil {
		t.Fatalf("roll back open and closed weeks with data: %v", err)
	}
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("reapply open and closed weeks with data: %v", err)
	}
	if err := db.QueryRow("SELECT count(*) FROM weeks WHERE household_id = $1 AND closed_at IS NULL", tenant.CowardHouseholdID).Scan(&open); err != nil || open != 1 {
		t.Fatalf("open weeks after reapplying = %d, %v; want 1", open, err)
	}
}
