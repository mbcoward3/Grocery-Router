package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/database"
	"github.com/mbcoward3/grocery-router/internal/httpapi"
	"github.com/mbcoward3/grocery-router/internal/ingest"
	"github.com/mbcoward3/grocery-router/internal/week"
)

type zeroPicker struct{}

func (zeroPicker) IntN(int) (int, error) { return 0, nil }

func TestWeekAPIEmptyGenerateAndMutate(t *testing.T) {
	handler := testHandler(t)

	empty := request(t, handler, http.MethodGet, "/api/week/current", "")
	if empty.Code != http.StatusNotFound {
		t.Fatalf("empty week status = %d, body %s", empty.Code, empty.Body.String())
	}
	assertErrorCode(t, empty, "no_current_week")

	generated := request(t, handler, http.MethodPost, "/api/week/current/generate", `{"recipeCount":3}`)
	if generated.Code != http.StatusOK {
		t.Fatalf("generate status = %d, body %s", generated.Code, generated.Body.String())
	}
	weekBody := decodeObject(t, generated)
	if weekBody["startsOn"] != "2026-08-16" {
		t.Fatalf("startsOn = %#v", weekBody["startsOn"])
	}
	recipes := weekBody["recipes"].([]any)
	if len(recipes) != 3 {
		t.Fatalf("generated recipes = %d", len(recipes))
	}
	first := recipes[0].(map[string]any)
	firstRecipe := first["recipe"].(map[string]any)
	if firstRecipe["name"] == "" || firstRecipe["handsOn"] == nil {
		t.Fatalf("recipe summary = %#v", firstRecipe)
	}

	added := request(t, handler, http.MethodPost, "/api/week/current/recipes",
		fmt.Sprintf(`{"recipeId":%.0f}`, firstRecipe["id"].(float64)))
	if added.Code != http.StatusOK {
		t.Fatalf("add status = %d, body %s", added.Code, added.Body.String())
	}
	addedRecipes := decodeObject(t, added)["recipes"].([]any)
	if len(addedRecipes) != 4 {
		t.Fatalf("recipes after add = %d", len(addedRecipes))
	}

	occurrenceID := int64(addedRecipes[1].(map[string]any)["id"].(float64))
	removed := request(t, handler, http.MethodDelete,
		fmt.Sprintf("/api/week/current/recipes/%d", occurrenceID), "")
	if removed.Code != http.StatusOK || len(decodeObject(t, removed)["recipes"].([]any)) != 3 {
		t.Fatalf("remove status = %d, body %s", removed.Code, removed.Body.String())
	}
}

func TestRecipesAPIAndRequestValidation(t *testing.T) {
	handler := testHandler(t)

	response := request(t, handler, http.MethodGet, "/api/recipes", "")
	if response.Code != http.StatusOK {
		t.Fatalf("recipes status = %d, body %s", response.Code, response.Body.String())
	}
	if recipes := decodeObject(t, response)["recipes"].([]any); len(recipes) != 24 {
		t.Fatalf("verified recipes = %d", len(recipes))
	}

	invalid := request(t, handler, http.MethodPost, "/api/week/current/generate", `{"recipeCount":3,"surprise":true}`)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid request status = %d, body %s", invalid.Code, invalid.Body.String())
	}
	assertErrorCode(t, invalid, "invalid_json")

	wrongMethod := request(t, handler, http.MethodDelete, "/api/week/current/generate", "")
	if wrongMethod.Code != http.StatusMethodNotAllowed {
		t.Fatalf("wrong method status = %d", wrongMethod.Code)
	}
}

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	dsn := fmt.Sprintf("file:httpapi-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := database.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	documents, err := ingest.ReadDirectory(filepath.Join("..", "..", "corpus", "recipes"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ingest.Import(context.Background(), db, documents); err != nil {
		t.Fatal(err)
	}
	service := week.NewService(db, zeroPicker{})
	now := func() time.Time {
		return time.Date(2026, time.August, 19, 12, 0, 0, 0, time.Local)
	}
	return httpapi.New(db, service, now).Handler()
}

func request(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	handler.ServeHTTP(recorder, req)
	return recorder
}

func decodeObject(t *testing.T, response *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %q: %v", response.Body.String(), err)
	}
	return body
}

func assertErrorCode(t *testing.T, response *httptest.ResponseRecorder, want string) {
	t.Helper()
	body := decodeObject(t, response)
	errorBody := body["error"].(map[string]any)
	if errorBody["code"] != want {
		t.Fatalf("error code = %#v, want %q", errorBody["code"], want)
	}
}
