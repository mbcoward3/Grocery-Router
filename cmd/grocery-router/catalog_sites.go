package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/mbcoward3/grocery-router/internal/catalog/registry"
	"github.com/mbcoward3/grocery-router/internal/database"
)

// catalogSitesCommands is embedded into cli so the catalog-sites command lives in this file.
type catalogSitesCommands struct {
	CatalogSites catalogSitesCommand `cmd:"" name:"catalog-sites" help:"Validate catalog/sites.yaml and sync it into the ingest schema."`
}

type catalogSitesCommand struct {
	DatabaseConfig
	Root      string `help:"Repository root." default:"." env:"GROCERY_ROUTER_ROOT" type:"path"`
	Registry  string `help:"Site registry path, relative to root." default:"catalog/sites.yaml" env:"GROCERY_ROUTER_CATALOG_SITES"`
	CheckOnly bool   `name:"check" help:"Validate the registry without touching the database."`
}

func (command *catalogSitesCommand) Run() error {
	// Validation happens before any database or network access.
	sites, err := registry.ReadFile(filepath.Join(command.Root, filepath.FromSlash(command.Registry)))
	if err != nil {
		return err
	}
	if command.CheckOnly {
		fmt.Printf("site registry valid: %d sites\n", len(sites.Sites))
		return nil
	}
	if err := migrate(command.DatabaseURL); err != nil {
		return err
	}
	db, err := database.Open(command.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	result, err := registry.Sync(context.Background(), db, sites)
	if err != nil {
		return err
	}
	fmt.Printf("site registry synced: %d sites (%d active), %d removed sites deactivated\n",
		result.Listed, result.Active, result.Retired)
	return nil
}
