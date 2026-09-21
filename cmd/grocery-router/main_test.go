package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/testdatabase"
)

func TestRunUsesDatabaseEnvironment(t *testing.T) {
	_, databaseURL := testdatabase.OpenWithURL(t)
	t.Setenv("GROCERY_ROUTER_DATABASE_URL", databaseURL)

	if err := run([]string{"migrate"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run migrate: %v", err)
	}
}

func TestRunUsesRepositoryRelativeDefaults(t *testing.T) {
	root := filepath.Join("..", "..")
	if err := run([]string{"corpus-audit", "--root", root}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run corpus-audit: %v", err)
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

func TestApplicationHandlerServesSPAAndAPI(t *testing.T) {
	webRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(webRoot, "index.html"), []byte("application shell"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webRoot, "asset.js"), []byte("asset"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}
	api := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte("api"))
	})
	handler, err := applicationHandler(api, webRoot)
	if err != nil {
		t.Fatalf("applicationHandler: %v", err)
	}

	for path, expected := range map[string]string{
		"/api/week/current": "api",
		"/asset.js":         "asset",
		"/groceries":        "application shell",
		"/healthz":          "ok\n",
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Errorf("GET %s status = %d", path, response.Code)
		}
		if body := response.Body.String(); body != expected {
			t.Errorf("GET %s body = %q, want %q", path, body, expected)
		}
	}
}

func TestApplicationHandlerRejectsMissingWebRoot(t *testing.T) {
	_, err := applicationHandler(http.NotFoundHandler(), filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("applicationHandler unexpectedly accepted a missing web root")
	}
}
