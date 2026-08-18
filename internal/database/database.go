package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

// migrations is the single authoritative migration set used by the application, tests,
// and the Goose-compatible migration runner.
//
//go:embed migrations/*.sql
var migrations embed.FS

// Open opens SQLite with the connection invariants Grocery Router relies on. Foreign keys
// are enabled in the DSN so the setting applies to every pooled connection, not merely the
// first one returned by database/sql.
func Open(dataSourceName string) (*sql.DB, error) {
	dsn := dataSourceName
	separator := "?"
	for _, b := range dsn {
		if b == '?' {
			separator = "&"
			break
		}
	}
	dsn += separator + "_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return db, nil
}

// Migrate applies all embedded Goose migrations in order.
func Migrate(ctx context.Context, db *sql.DB) error {
	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.UpContext(ctx, db, "migrations"); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
