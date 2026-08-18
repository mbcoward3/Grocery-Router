package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mbcoward3/grocery-router/internal/database"
	"github.com/mbcoward3/grocery-router/internal/trueup"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: grocery-router <migrate|trueup-inventory>")
	}

	switch args[0] {
	case "migrate":
		flags := flag.NewFlagSet("migrate", flag.ContinueOnError)
		databasePath := flags.String("database", "data/grocery-router.db", "SQLite database path")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		return migrate(*databasePath)
	case "trueup-inventory":
		flags := flag.NewFlagSet("trueup-inventory", flag.ContinueOnError)
		root := flags.String("root", ".", "repository root")
		inventoryPath := flags.String("inventory", "trueup/recipes.csv", "inventory path relative to root")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		return auditInventory(*root, *inventoryPath)
	default:
		return fmt.Errorf("unknown command %q; usage: grocery-router <migrate|trueup-inventory>", args[0])
	}
}

func migrate(path string) error {
	if directory := filepath.Dir(path); directory != "." {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return fmt.Errorf("create database directory: %w", err)
		}
	}
	db, err := database.Open(path)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := database.Migrate(context.Background(), db); err != nil {
		return err
	}
	fmt.Printf("migrated %s\n", path)
	return nil
}

func auditInventory(root, relativePath string) error {
	file, err := os.Open(filepath.Join(root, filepath.FromSlash(relativePath)))
	if err != nil {
		return fmt.Errorf("open inventory: %w", err)
	}
	defer file.Close()
	rows, err := trueup.ReadInventory(root, file)
	if err != nil {
		return err
	}
	fmt.Printf("inventory valid: %d PDF recipes\n", len(rows))
	return nil
}
