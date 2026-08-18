// Command grocery-router manages the local corpus and SQLite database.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/mbcoward3/grocery-router/internal/database"
	"github.com/mbcoward3/grocery-router/internal/ingest"
	"github.com/mbcoward3/grocery-router/internal/trueup"
)

type RepositoryPaths struct {
	Root      string `help:"Repository root." default:"." env:"GROCERY_ROUTER_ROOT" type:"path"`
	Corpus    string `help:"Approved Markdown corpus directory, relative to root." default:"corpus/recipes" env:"GROCERY_ROUTER_CORPUS" type:"path"`
	Inventory string `help:"Inventory path, relative to root." default:"trueup/recipes.csv" env:"GROCERY_ROUTER_INVENTORY" type:"path"`
}

type DatabasePath struct {
	Database string `help:"SQLite database path." default:"data/grocery-router.db" env:"GROCERY_ROUTER_DATABASE" type:"path"`
}

type corpusAuditCommand struct {
	RepositoryPaths
}

func (command *corpusAuditCommand) Run() error {
	return auditCorpus(command.Root, command.Corpus, command.Inventory)
}

type corpusIngestCommand struct {
	RepositoryPaths
	DatabasePath
}

func (command *corpusIngestCommand) Run() error {
	return ingestCorpus(command.Database, command.Root, command.Corpus, command.Inventory)
}

type migrateCommand struct {
	DatabasePath
}

func (command *migrateCommand) Run() error {
	return migrate(command.Database)
}

type trueupInventoryCommand struct {
	Root      string `help:"Repository root." default:"." env:"GROCERY_ROUTER_ROOT" type:"path"`
	Inventory string `help:"Inventory path, relative to root." default:"trueup/recipes.csv" env:"GROCERY_ROUTER_INVENTORY" type:"path"`
}

func (command *trueupInventoryCommand) Run() error {
	return auditInventory(command.Root, command.Inventory)
}

type cli struct {
	CorpusAudit     corpusAuditCommand     `cmd:"" help:"Validate the approved corpus against the PDF inventory."`
	CorpusIngest    corpusIngestCommand    `cmd:"" help:"Migrate a database and transactionally ingest the approved corpus."`
	Migrate         migrateCommand         `cmd:"" help:"Apply all database migrations."`
	TrueupInventory trueupInventoryCommand `cmd:"" help:"Validate the PDF recipe inventory and its evidence paths."`
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	configuration := &cli{}
	parser, err := kong.New(
		configuration,
		kong.Name("grocery-router"),
		kong.Description("Build and validate the Grocery Router corpus and database."),
		kong.UsageOnError(),
		kong.Writers(stdout, stderr),
	)
	if err != nil {
		return fmt.Errorf("configure CLI: %w", err)
	}
	context, err := parser.Parse(args)
	if err != nil {
		return err
	}
	if err := context.Run(); err != nil {
		return err
	}
	return nil
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
