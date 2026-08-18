package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mbcoward3/grocery-router/internal/database"
	"github.com/mbcoward3/grocery-router/internal/ingest"
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
		return fmt.Errorf("usage: grocery-router <corpus-audit|corpus-ingest|migrate|trueup-inventory>")
	}

	switch args[0] {
	case "corpus-audit":
		flags := flag.NewFlagSet("corpus-audit", flag.ContinueOnError)
		root := flags.String("root", ".", "repository root")
		corpusPath := flags.String("corpus", "corpus/recipes", "approved Markdown corpus directory relative to root")
		inventoryPath := flags.String("inventory", "trueup/recipes.csv", "inventory path relative to root")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		return auditCorpus(*root, *corpusPath, *inventoryPath)
	case "corpus-ingest":
		flags := flag.NewFlagSet("corpus-ingest", flag.ContinueOnError)
		databasePath := flags.String("database", "data/grocery-router.db", "SQLite database path")
		root := flags.String("root", ".", "repository root")
		corpusPath := flags.String("corpus", "corpus/recipes", "approved Markdown corpus directory relative to root")
		inventoryPath := flags.String("inventory", "trueup/recipes.csv", "inventory path relative to root")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		return ingestCorpus(*databasePath, *root, *corpusPath, *inventoryPath)
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
		return fmt.Errorf("unknown command %q; usage: grocery-router <corpus-audit|corpus-ingest|migrate|trueup-inventory>", args[0])
	}
}

func auditCorpus(root, corpusPath, inventoryPath string) error {
	documents, inventoryCount, err := readAuditedCorpus(root, corpusPath, inventoryPath)
	if err != nil {
		return err
	}
	fmt.Printf("corpus valid: %d approved of %d PDF recipes\n", len(documents), inventoryCount)
	return nil
}

func ingestCorpus(databasePath, root, corpusPath, inventoryPath string) error {
	if err := migrate(databasePath); err != nil {
		return err
	}
	documents, _, err := readAuditedCorpus(root, corpusPath, inventoryPath)
	if err != nil {
		return err
	}
	db, err := database.Open(databasePath)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := ingest.Import(context.Background(), db, documents); err != nil {
		return err
	}
	fmt.Printf("ingested %d approved recipes into %s\n", len(documents), databasePath)
	return nil
}

func readAuditedCorpus(root, corpusPath, inventoryPath string) ([]ingest.Document, int, error) {
	documents, err := ingest.ReadDirectory(filepath.Join(root, filepath.FromSlash(corpusPath)))
	if err != nil {
		return nil, 0, err
	}
	file, err := os.Open(filepath.Join(root, filepath.FromSlash(inventoryPath)))
	if err != nil {
		return nil, 0, fmt.Errorf("open inventory: %w", err)
	}
	defer file.Close()
	rows, err := trueup.ReadInventory(root, file)
	if err != nil {
		return nil, 0, err
	}
	if err := trueup.MatchApprovedCorpus(rows, documents); err != nil {
		return nil, 0, err
	}
	return documents, len(rows), nil
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
