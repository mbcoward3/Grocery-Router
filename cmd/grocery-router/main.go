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
	"github.com/mbcoward3/grocery-router/internal/auth"
	"github.com/mbcoward3/grocery-router/internal/catalog"
	"github.com/mbcoward3/grocery-router/internal/catalogcandidate"
	"github.com/mbcoward3/grocery-router/internal/database"
	"github.com/mbcoward3/grocery-router/internal/httpapi"
	"github.com/mbcoward3/grocery-router/internal/ingest"
	"github.com/mbcoward3/grocery-router/internal/recipeintel"
	"github.com/mbcoward3/grocery-router/internal/recipepilot"
	"github.com/mbcoward3/grocery-router/internal/tenant"
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
	Address        string `help:"HTTP listen address." default:"127.0.0.1:8080" env:"GROCERY_ROUTER_ADDRESS"`
	WebRoot        string `help:"Built web application directory. Leave empty to serve only the API." default:"web/dist" env:"GROCERY_ROUTER_WEB_ROOT" type:"path"`
	Origin         string `help:"Canonical application origin." env:"GROCERY_ROUTER_AUTH_ORIGIN"`
	OIDCIssuer     string `help:"Trusted OIDC issuer." default:"https://accounts.google.com" env:"GROCERY_ROUTER_AUTH_OIDC_ISSUER"`
	OIDCClientID   string `help:"Google OIDC client ID." env:"GROCERY_ROUTER_AUTH_GOOGLE_CLIENT_ID"`
	OIDCSecret     string `help:"Google OIDC client secret." env:"GROCERY_ROUTER_AUTH_GOOGLE_CLIENT_SECRET"`
	SessionSecret  string `help:"Base64url application session secret." env:"GROCERY_ROUTER_AUTH_SESSION_SECRET"`
	BootstrapUsers string `help:"JSON bootstrap owner allowlist." env:"GROCERY_ROUTER_AUTH_BOOTSTRAP_USERS"`
	HouseholdName  string `help:"Seeded household name." env:"GROCERY_ROUTER_AUTH_BOOTSTRAP_HOUSEHOLD"`
}

func (command *serveCommand) Run() error {
	config, err := auth.ParseConfig(auth.RawConfig{
		Origin: command.Origin, Issuer: command.OIDCIssuer, ClientID: command.OIDCClientID,
		ClientSecret: command.OIDCSecret, SessionSecret: command.SessionSecret,
		AllowedEmails: command.BootstrapUsers, HouseholdName: command.HouseholdName,
	})
	if err != nil {
		return fmt.Errorf("validate authentication configuration: %w", err)
	}
	return serve(command.DatabaseURL, command.Address, command.WebRoot, config)
}

type catalogAuditCommand struct {
	Root    string `help:"Repository root." default:"." env:"GROCERY_ROUTER_ROOT" type:"path"`
	Release string `help:"Catalog release manifest, relative to root." default:"catalog/releases/initial-review-2026-09-29.yaml"`
}

func (command *catalogAuditCommand) Run() error {
	manifest, _, err := catalog.LoadReviewManifest(command.Root, command.Release)
	if err != nil {
		return err
	}
	fmt.Printf("catalog release valid: %d %s recipes in %s\n", len(manifest.Recipes), manifest.Status, manifest.ReleaseID)
	return nil
}

type catalogPublishCommand struct {
	Root    string `help:"Repository root." default:"." env:"GROCERY_ROUTER_ROOT" type:"path"`
	Release string `help:"Review release manifest, relative to root." required:""`
	DatabaseConfig
}

func (command *catalogPublishCommand) Run() error {
	if err := migrate(command.DatabaseURL); err != nil {
		return err
	}
	manifest, data, err := catalog.LoadReviewManifest(command.Root, command.Release)
	if err != nil {
		return err
	}
	db, err := database.Open(command.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	result, err := catalog.NewService(db).PublishReviewManifest(context.Background(), command.Root, manifest, data)
	if err != nil {
		return err
	}
	if result.Applied {
		fmt.Printf("published %d reviewable catalog recipes from %s\n", len(manifest.Recipes), manifest.ReleaseID)
	} else {
		fmt.Printf("catalog release %s already applied exactly\n", manifest.ReleaseID)
	}
	return nil
}

type catalogAgentInputsCommand struct {
	Root      string `help:"Repository root." default:"." env:"GROCERY_ROUTER_ROOT" type:"path"`
	SourceRun string `help:"Completed source-pilot run, relative to root." required:""`
	Corpus    string `help:"Approved household corpus used for grocery match references." default:"corpus/recipes"`
	Output    string `help:"Ignored Pi input directory, relative to root." default:".local-run/catalog-agent/inputs"`
}

func (command *catalogAgentInputsCommand) Run() error {
	documents, err := ingest.ReadDirectory(filepath.Join(command.Root, filepath.FromSlash(command.Corpus)))
	if err != nil {
		return err
	}
	count, err := catalogcandidate.WriteAgentPackets(
		filepath.Join(command.Root, filepath.FromSlash(command.SourceRun)),
		filepath.Join(command.Root, filepath.FromSlash(command.Output)), documents,
	)
	if err == nil {
		fmt.Printf("wrote %d catalog Pi input packets\n", count)
	}
	return err
}

type catalogCandidateBuildCommand struct {
	Root       string `help:"Repository root." default:"." env:"GROCERY_ROUTER_ROOT" type:"path"`
	SourceRun  string `help:"Completed source-pilot run, relative to root." required:""`
	Proposals  string `help:"Pi proposal directory, relative to root." required:""`
	Output     string `help:"Catalog candidate output directory, relative to root." default:"catalog/candidates"`
	Provider   string `help:"Pi proposal provider provenance." default:"openai-codex"`
	Model      string `help:"Pi proposal model provenance." default:"gpt-5.5"`
	PromptHash string `help:"SHA-256 of the exact Pi prompt." required:""`
	RunAt      string `help:"UTC RFC3339 time of the Pi proposal run." required:""`
}

func (command *catalogCandidateBuildCommand) Run() error {
	runAt, err := time.Parse(time.RFC3339, command.RunAt)
	if err != nil {
		return fmt.Errorf("parse --run-at: %w", err)
	}
	report, err := catalogcandidate.BuildBatch(catalogcandidate.BuildOptions{
		SourceRunDir: filepath.Join(command.Root, filepath.FromSlash(command.SourceRun)),
		ProposalDir:  filepath.Join(command.Root, filepath.FromSlash(command.Proposals)),
		OutputDir:    filepath.Join(command.Root, filepath.FromSlash(command.Output)),
		Agent: catalogcandidate.AgentProvenance{
			Kind: "pi-proposed", Provider: command.Provider, Model: command.Model,
			PromptSHA256: command.PromptHash, RunAt: runAt,
		},
	})
	if report.IndexPath != "" {
		fmt.Printf("built %d catalog candidates; review: %s\n", len(report.Candidates), report.IndexPath)
	}
	return err
}

type recipeSourcePilotCommand struct {
	Root     string `help:"Repository root." default:"." env:"GROCERY_ROUTER_ROOT" type:"path"`
	Manifest string `help:"Pilot source manifest, relative to root." default:"pilot/recipe-sources.yaml"`
	Output   string `help:"Run output root, relative to root." default:".local-run/recipe-source-pilot"`
}

func (command *recipeSourcePilotCommand) Run() error {
	runner := recipepilot.Runner{Fetcher: recipepilot.NewSafeFetcher()}
	report, err := runner.Run(
		context.Background(),
		filepath.Join(command.Root, filepath.FromSlash(command.Manifest)),
		filepath.Join(command.Root, filepath.FromSlash(command.Output)),
	)
	if report.Directory != "" {
		fmt.Printf("recipe source pilot: %d succeeded, %d failed; output: %s\n", report.Index.Succeeded, report.Index.Failed, report.Directory)
	}
	return err
}

type recipeIntelligencePilotCommand struct {
	Root       string `help:"Repository root." default:"." env:"GROCERY_ROUTER_ROOT" type:"path"`
	Catalog    string `help:"Versioned assessment catalog, relative to root." default:"pilot/recipe-intelligence.yaml"`
	SourceRun  string `help:"Source-pilot run directory or output root, relative to root." default:".local-run/recipe-source-pilot"`
	Corpus     string `help:"Approved corpus directory, relative to root." default:"corpus/recipes"`
	Output     string `help:"Assessment output root, relative to root." default:".local-run/recipe-intelligence"`
	APIKey     string `help:"TypeSafe API key." env:"TYPESAFE_API_KEY" hidden:""`
	Endpoint   string `help:"TypeSafe System One endpoint." default:"https://api.typesafe.ai/v1/systemone" env:"TYPESAFE_API_ENDPOINT"`
	MaxRetries int    `help:"Retries for rate limiting or service overload." default:"3"`
}

func (command *recipeIntelligencePilotCommand) Run() error {
	client := recipeintel.HTTPClient{Endpoint: command.Endpoint, APIKey: command.APIKey, MaxRetries: command.MaxRetries}
	runner := recipeintel.Runner{Evaluator: client}
	report, err := runner.Run(context.Background(), recipeintel.RunConfig{
		CatalogPath: filepath.Join(command.Root, filepath.FromSlash(command.Catalog)),
		SourcePath:  filepath.Join(command.Root, filepath.FromSlash(command.SourceRun)),
		CorpusPath:  filepath.Join(command.Root, filepath.FromSlash(command.Corpus)),
		OutputRoot:  filepath.Join(command.Root, filepath.FromSlash(command.Output)),
	})
	if report.Directory != "" {
		fmt.Printf("recipe intelligence pilot: %d succeeded, %d failed; %d input tokens; output: %s\n", report.Index.Succeeded, report.Index.Failed, report.Index.InputTokens, report.Directory)
	}
	return err
}

type trueupInventoryCommand struct {
	Root      string `help:"Repository root." default:"." env:"GROCERY_ROUTER_ROOT" type:"path"`
	Inventory string `help:"Archived inventory path, relative to root." default:"archive/trueup/recipes.csv" env:"GROCERY_ROUTER_INVENTORY"`
}

func (command *trueupInventoryCommand) Run() error {
	return auditInventory(command.Root, command.Inventory)
}

type cli struct {
	Bootstrap               bootstrapCommand               `cmd:"" help:"Migrate and load the approved corpus when the database is empty."`
	CatalogAgentInputs      catalogAgentInputsCommand      `cmd:"" help:"Write compact source packets for isolated Pi standardization workers."`
	CatalogAudit            catalogAuditCommand            `cmd:"" help:"Audit a catalog release and every referenced digest."`
	CatalogCandidateBuild   catalogCandidateBuildCommand   `cmd:"" help:"Build strict review candidates from source evidence and Pi proposals."`
	CatalogPublish          catalogPublishCommand          `cmd:"" help:"Transactionally publish a catalog review release to PostgreSQL."`
	CorpusAudit             corpusAuditCommand             `cmd:"" help:"Validate the approved corpus against the PDF inventory."`
	CorpusIngest            corpusIngestCommand            `cmd:"" help:"Migrate a database and transactionally ingest the approved corpus."`
	CorpusRender            corpusRenderCommand            `cmd:"" help:"Regenerate checked human-readable recipe sections."`
	Migrate                 migrateCommand                 `cmd:"" help:"Apply all database migrations."`
	RecipeIntelligencePilot recipeIntelligencePilotCommand `cmd:"" help:"Assess source and approved recipes with the import-time Jev catalog."`
	RecipeSourcePilot       recipeSourcePilotCommand       `cmd:"" help:"Fetch manifested public recipe JSON-LD into inspectable pilot output."`
	Serve                   serveCommand                   `cmd:"" help:"Start the local Grocery Router HTTP API."`
	TrueupInventory         trueupInventoryCommand         `cmd:"" help:"Validate the PDF recipe inventory and its evidence paths."`
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
	if err := ingest.Import(context.Background(), db, "c0a7a2d8-669b-4e47-91c1-4d9a32f339d5", documents); err != nil {
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
	if err := db.QueryRow("SELECT count(*) FROM recipes WHERE household_id = $1", tenant.CowardHouseholdID).Scan(&count); err != nil {
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
	if err := ingest.Import(context.Background(), db, "c0a7a2d8-669b-4e47-91c1-4d9a32f339d5", documents); err != nil {
		return err
	}
	fmt.Printf("ingested %d approved recipes\n", len(documents))
	return nil
}

func serve(databasePath, address, webRoot string, authConfig auth.Config) error {
	if err := migrate(databasePath); err != nil {
		return err
	}
	db, err := database.Open(databasePath)
	if err != nil {
		return err
	}
	defer db.Close()

	weekService := week.NewService(db, nil, tenant.CowardHouseholdID)
	api := httpapi.New(db, weekService, nil, tenant.CowardHouseholdID, auth.AuthorizeHousehold)
	provider, err := auth.NewGoogleProvider(context.Background(), authConfig)
	if err != nil {
		return err
	}
	authService := auth.NewService(db, authConfig)
	authHandler := auth.NewHTTPHandler(authConfig, authService, provider)
	apiMux := http.NewServeMux()
	authHandler.Register(apiMux)
	apiMux.Handle("/api/v2/catalog/", authHandler.RequireSession(api.Handler()))
	apiMux.Handle("/api/v2/households/", authHandler.RequireSession(api.Handler()))
	handler, err := applicationHandler(apiMux, webRoot)
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
		return securityHeaders(mux), nil
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
			staticPath := filepath.Join(webRoot, relativePath)
			if info, err := os.Stat(staticPath); err == nil {
				if !info.IsDir() {
					files.ServeHTTP(response, request)
					return
				}
				if _, err := os.Stat(filepath.Join(staticPath, "index.html")); err == nil {
					files.ServeHTTP(response, request)
					return
				}
			}
		}
		http.ServeFile(response, request, indexPath)
	})
	return securityHeaders(mux), nil
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' https: data:; style-src 'self' 'unsafe-inline'; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self' https://accounts.google.com")
		response.Header().Set("Referrer-Policy", "no-referrer")
		response.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(response, request)
	})
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
