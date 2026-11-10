package main

import (
	"io"
	"strings"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/tenant"
	"github.com/mbcoward3/grocery-router/internal/testdatabase"
)

func TestHouseholdTimezoneCommand(t *testing.T) {
	db, databaseURL := testdatabase.OpenWithURL(t)
	zone := func() string {
		t.Helper()
		var value string
		if err := db.QueryRow("SELECT timezone FROM households WHERE id = $1", tenant.CowardHouseholdID).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	if got := zone(); got != "UTC" {
		t.Fatalf("default zone = %q, want UTC", got)
	}
	// An open automatic week whose close time is not a Chicago Sunday midnight.
	var weekID int64
	if err := db.QueryRow(`INSERT INTO weeks (household_id, starts_on, opened_at, auto_close_at)
		VALUES ($1, '2026-08-16', now() - interval '1 day', now() + interval '1 hour') RETURNING id`,
		tenant.CowardHouseholdID).Scan(&weekID); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"household-timezone", "America/Chicago", "--database-url", databaseURL}, io.Discard, io.Discard); err != nil {
		t.Fatalf("household-timezone: %v", err)
	}
	if got := zone(); got != "America/Chicago" {
		t.Fatalf("zone = %q, want America/Chicago", got)
	}
	// Like the Settings screen, the command keeps the week open and moves its close time to the
	// next Sunday 12:00 AM in the new zone.
	var open, future, withinWeek bool
	var weekday int
	var clock string
	if err := db.QueryRow(`SELECT closed_at IS NULL, auto_close_at > now(), auto_close_at <= now() + interval '7 days',
		CAST(EXTRACT(DOW FROM auto_close_at AT TIME ZONE 'America/Chicago') AS INTEGER),
		to_char(auto_close_at AT TIME ZONE 'America/Chicago', 'HH24:MI')
		FROM weeks WHERE id = $1`, weekID).Scan(&open, &future, &withinWeek, &weekday, &clock); err != nil {
		t.Fatal(err)
	}
	if !open || !future || !withinWeek || weekday != 0 || clock != "00:00" {
		t.Fatalf("open week after zone change: open %t, future %t, within a week %t, weekday %d, clock %s",
			open, future, withinWeek, weekday, clock)
	}
	for _, args := range [][]string{
		{"household-timezone", "Not/AZone", "--database-url", databaseURL},
		{"household-timezone", "Local", "--database-url", databaseURL},
		{"household-timezone", "UTC", "--household", "c7c7d0df-c81c-41d9-b95b-c131bdd605bd", "--database-url", databaseURL},
		{"household-timezone", "UTC", "--household", "coward", "--database-url", databaseURL},
	} {
		if err := run(args, io.Discard, io.Discard); err == nil {
			t.Fatalf("%v unexpectedly succeeded", args)
		}
		if got := zone(); got != "America/Chicago" {
			t.Fatalf("%v changed zone to %q", args, got)
		}
	}
}

func TestHouseholdTimezoneRejectsInputBeforeDatabase(t *testing.T) {
	// The unreachable database URL proves both checks fail before any connection.
	const databaseURL = "postgres://nobody@127.0.0.1:1/none?sslmode=disable"
	for args, want := range map[[2]string]string{
		{"Not/AZone", tenant.CowardHouseholdID}: "unknown time zone",
		{"America/Chicago", "coward"}:           "is not a UUID",
	} {
		err := run([]string{"household-timezone", args[0], "--household", args[1], "--database-url", databaseURL}, io.Discard, io.Discard)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("household-timezone %v error = %v, want %q", args, err, want)
		}
	}
}
