package main

import (
	"io"
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
	} {
		if err := run(args, io.Discard, io.Discard); err == nil {
			t.Fatalf("%v unexpectedly succeeded", args)
		}
		if got := zone(); got != "America/Chicago" {
			t.Fatalf("%v changed zone to %q", args, got)
		}
	}
}
