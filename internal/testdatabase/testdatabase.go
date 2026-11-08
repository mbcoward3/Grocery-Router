// Package testdatabase creates isolated PostgreSQL databases for integration tests.
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

// Open creates, migrates, and returns a pool for a fresh temporary database.
func Open(t testing.TB) *sql.DB {
	t.Helper()
	db, _ := OpenWithURL(t)
	return db
}

// OpenWithURL also returns the isolated connection URL for CLI tests.
//
// Each test gets its own database rather than its own schema because migrations create fixed
// named schemas (such as ingest) that a search_path cannot isolate.
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
	name := fmt.Sprintf("test_%d_%d_%d", time.Now().UnixNano(), os.Getpid(), sequence.Add(1))
	if _, err := admin.Exec(`CREATE DATABASE "` + name + `" TEMPLATE template0`); err != nil {
		admin.Close()
		t.Fatalf("create test database: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(`DROP DATABASE IF EXISTS "` + name + `" WITH (FORCE)`)
		admin.Close()
	})

	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse test database URL: %v", err)
	}
	parsed.Path = "/" + name
	isolatedURL := parsed.String()
	db, err := database.Open(isolatedURL)
	if err != nil {
		t.Fatalf("open isolated test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(context.Background(), db); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return db, isolatedURL
}
