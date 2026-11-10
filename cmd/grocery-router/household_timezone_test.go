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
	if err := run([]string{"household-timezone", "America/Chicago", "--database-url", databaseURL}, io.Discard, io.Discard); err != nil {
		t.Fatalf("household-timezone: %v", err)
	}
	if got := zone(); got != "America/Chicago" {
		t.Fatalf("zone = %q, want America/Chicago", got)
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
