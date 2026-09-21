// Command grocery-router manages the local corpus and PostgreSQL database.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/mbcoward3/grocery-router/internal/database"
	"github.com/mbcoward3/grocery-router/internal/httpapi"
	"github.com/mbcoward3/grocery-router/internal/ingest"
	"github.com/mbcoward3/grocery-router/internal/trueup"
	"github.com/mbcoward3/grocery-router/internal/week"
)

type RepositoryPaths struct {
	Root      string `help:"Repository root." default:"." env:"GROCERY_ROUTER_ROOT" type:"path"`
	Corpus    string `help:"Approved Markdown corpus directory, relative to root." default:"corpus/recipes" env:"GROCERY_ROUTER_CORPUS"`
	Inventory string `help:"Archived inventory path, relative to root." default:"archive/trueup/recipes.csv" env:"GROCERY_ROUTER_INVENTORY"`
}

type DatabaseConfig struct {
	DatabaseURL string `name:"database-url" help:"PostgreSQL connection URL." default:"postgres://grocery_router:grocery_router@localhost:5432/grocery_router?sslmode=disable" env:"GROCERY_ROUTER_DATABASE_URL"`
}

type corpusAuditCommand struct {
	RepositoryPaths
}

func (command *corpusAuditCommand) Run() error {
	return auditCorpus(command.Root, command.Corpus, command.Inventory)
}

type corpusIngestCommand struct {
	RepositoryPaths
	DatabaseConfig
}

func (command *corpusIngestCommand) Run() error {
	return ingestCorpus(command.DatabaseURL, command.Root, command.Corpus, command.Inventory)
}

type corpusRenderCommand struct {
	Root   string `help:"Repository root." default:"." env:"GROCERY_ROUTER_ROOT" type:"path"`
	Corpus string `help:"Approved Markdown corpus directory, relative to root." default:"corpus/recipes" env:"GROCERY_ROUTER_CORPUS"`
}

func (command *corpusRenderCommand) Run() error {
	return renderCorpus(filepath.Join(command.Root, filepath.FromSlash(command.Corpus)))
}

type migrateCommand struct {
	DatabaseConfig
}

func (command *migrateCommand) Run() error {
	return migrate(command.DatabaseURL)
}

type bootstrapCommand struct {
	RepositoryPaths
	DatabaseConfig
}

func (command *bootstrapCommand) Run() error {
	return bootstrap(command.DatabaseURL, command.Root, command.Corpus, command.Inventory)
}

type serveCommand struct {
	DatabaseConfig
	Address string `help:"HTTP listen address." default:"127.0.0.1:8080" env:"GROCERY_ROUTER_ADDRESS"`
	WebRoot string `help:"Built web application directory. Leave empty to serve only the API." default:"web/dist" env:"GROCERY_ROUTER_WEB_ROOT" type:"path"`
}

func (command *serveCommand) Run() error {
	return serve(command.DatabaseURL, command.Address, command.WebRoot)
}

type trueupInventoryCommand struct {
	Root      string `help:"Repository root." default:"." env:"GROCERY_ROUTER_ROOT" type:"path"`
	Inventory string `help:"Archived inventory path, relative to root." default:"archive/trueup/recipes.csv" env:"GROCERY_ROUTER_INVENTORY"`
}

func (command *trueupInventoryCommand) Run() error {
	return auditInventory(command.Root, command.Inventory)
}

type cli struct {
	Bootstrap       bootstrapCommand       `cmd:"" help:"Migrate and load the approved corpus when the database is empty."`
	CorpusAudit     corpusAuditCommand     `cmd:"" help:"Validate the approved corpus against the PDF inventory."`
	CorpusIngest    corpusIngestCommand    `cmd:"" help:"Migrate a database and transactionally ingest the approved corpus."`
	CorpusRender    corpusRenderCommand    `cmd:"" help:"Regenerate checked human-readable recipe sections."`
	Migrate         migrateCommand         `cmd:"" help:"Apply all database migrations."`
	Serve           serveCommand           `cmd:"" help:"Start the local Grocery Router HTTP API."`
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
	parsedContext, err := parser.Parse(args)
	if err != nil {
		return err
	}
	if err := parsedContext.Run(); err != nil {
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
	fmt.Printf("ingested %d approved recipes\n", len(documents))
	return nil
}

func renderCorpus(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("read corpus directory: %w", err)
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		filePath := filepath.Join(path, entry.Name())
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		rendered, renderErr := ingest.RewriteReadableBody(file)
		file.Close()
		if renderErr != nil {
			return fmt.Errorf("render %s: %w", entry.Name(), renderErr)
		}
		if err := os.WriteFile(filePath, rendered, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", entry.Name(), err)
		}
		count++
	}
	fmt.Printf("rendered %d approved recipes\n", count)
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

func migrate(databaseURL string) error {
	db, err := database.Open(databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := database.Migrate(context.Background(), db); err != nil {
		return err
	}
	fmt.Println("database migrated")
	return nil
}

func bootstrap(databaseURL, root, corpusPath, inventoryPath string) error {
	if err := migrate(databaseURL); err != nil {
		return err
	}
	db, err := database.Open(databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	var count int
	if err := db.QueryRow("SELECT count(*) FROM recipes").Scan(&count); err != nil {
		return fmt.Errorf("count recipes: %w", err)
	}
	if count > 0 {
		fmt.Printf("database already bootstrapped with %d recipes\n", count)
		return nil
	}
	documents, _, err := readAuditedCorpus(root, corpusPath, inventoryPath)
	if err != nil {
		return err
	}
	if err := ingest.Import(context.Background(), db, documents); err != nil {
		return err
	}
	fmt.Printf("ingested %d approved recipes\n", len(documents))
	return nil
}

func serve(databasePath, address, webRoot string) error {
	if err := migrate(databasePath); err != nil {
		return err
	}
	db, err := database.Open(databasePath)
	if err != nil {
		return err
	}
	defer db.Close()

	weekService := week.NewService(db, nil)
	api := httpapi.New(db, weekService, nil)
	handler, err := applicationHandler(api.Handler(), webRoot)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	fmt.Printf("Grocery Router listening on http://%s\n", address)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve Grocery Router: %w", err)
	}
	return nil
}

func applicationHandler(api http.Handler, webRoot string) (http.Handler, error) {
	mux := http.NewServeMux()
	mux.Handle("/api/", api)
	mux.HandleFunc("GET /healthz", func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "text/plain; charset=utf-8")
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte("ok\n"))
	})

	if webRoot == "" {
		return mux, nil
	}
	indexPath := filepath.Join(webRoot, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return nil, fmt.Errorf("find web application index: %w", err)
	}
	files := http.FileServer(http.Dir(webRoot))
	mux.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			http.NotFound(response, request)
			return
		}
		relativePath := strings.TrimPrefix(filepath.Clean(request.URL.Path), string(filepath.Separator))
		if relativePath != "." {
			if info, err := os.Stat(filepath.Join(webRoot, relativePath)); err == nil && !info.IsDir() {
				files.ServeHTTP(response, request)
				return
			}
		}
		http.ServeFile(response, request, indexPath)
	})
	return mux, nil
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
