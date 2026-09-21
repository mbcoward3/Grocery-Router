// Package testdatabase creates isolated PostgreSQL schemas for integration tests.
package testdatabase

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/database"
)

var sequence atomic.Uint64

// Open creates, migrates, and returns a database pool isolated by a temporary schema.
func Open(t testing.TB) *sql.DB {
	t.Helper()
	db, _ := OpenWithURL(t)
	return db
}

// OpenWithURL also returns the isolated connection URL for CLI tests.
func OpenWithURL(t testing.TB) (*sql.DB, string) {
	t.Helper()
	baseURL := os.Getenv("GROCERY_ROUTER_TEST_DATABASE_URL")
	if baseURL == "" {
		baseURL = "postgres://grocery_router:grocery_router@localhost:5432/grocery_router?sslmode=disable"
	}
	admin, err := database.Open(baseURL)
	if err != nil {
		t.Fatalf("open test postgres (set GROCERY_ROUTER_TEST_DATABASE_URL): %v", err)
	}
	schema := fmt.Sprintf("test_%d_%d", time.Now().UnixNano(), sequence.Add(1))
	if _, err := admin.Exec(`CREATE SCHEMA "` + schema + `"`); err != nil {
		admin.Close()
		t.Fatalf("create test schema: %v", err)
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse test database URL: %v", err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	db, err := database.Open(parsed.String())
	if err != nil {
		t.Fatalf("open isolated test schema: %v", err)
	}
	if err := database.Migrate(context.Background(), db); err != nil {
		db.Close()
		t.Fatalf("migrate test schema: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		_, _ = admin.Exec(`DROP SCHEMA IF EXISTS "` + schema + `" CASCADE`)
		admin.Close()
	})
	return db, parsed.String()
}
