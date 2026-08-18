package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunUsesDatabaseEnvironment(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "configured.db")
	t.Setenv("GROCERY_ROUTER_DATABASE", databasePath)

	if err := run([]string{"migrate"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run migrate: %v", err)
	}
	if _, err := os.Stat(databasePath); err != nil {
		t.Fatalf("stat configured database: %v", err)
	}
}

func TestRunReportsUnknownCommand(t *testing.T) {
	err := run([]string{"unknown"}, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("run unknown command unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "unexpected argument unknown") {
		t.Fatalf("unexpected error: %v", err)
	}
}
