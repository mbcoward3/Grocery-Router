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

	"github.com/mbcoward3/grocery-router/internal/httpapi"
	"github.com/mbcoward3/grocery-router/internal/ingest"
	"github.com/mbcoward3/grocery-router/internal/testdatabase"
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

func TestRecipeDetailAndGroceryAPI(t *testing.T) {
	handler := testHandler(t)
	generated := request(t, handler, http.MethodPost, "/api/week/current/generate", `{"recipeCount":2}`)
	if generated.Code != http.StatusOK {
		t.Fatalf("generate status = %d, body %s", generated.Code, generated.Body.String())
	}
	firstRecipe := decodeObject(t, generated)["recipes"].([]any)[0].(map[string]any)["recipe"].(map[string]any)
	recipeID := int64(firstRecipe["id"].(float64))
	detail := request(t, handler, http.MethodGet, fmt.Sprintf("/api/recipes/%d", recipeID), "")
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status = %d, body %s", detail.Code, detail.Body.String())
	}
	detailBody := decodeObject(t, detail)
	if detailBody["name"] == "" || len(detailBody["ingredientSections"].([]any)) == 0 || len(detailBody["instructionSections"].([]any)) == 0 {
		t.Fatalf("incomplete recipe detail = %#v", detailBody)
	}

	groceries := request(t, handler, http.MethodGet, "/api/week/current/groceries", "")
	groceryBody := decodeObject(t, groceries)
	lines := groceryBody["lines"].([]any)
	if groceries.Code != http.StatusOK || len(lines) == 0 {
		t.Fatalf("groceries status = %d, body %s", groceries.Code, groceries.Body.String())
	}
	line := lines[0].(map[string]any)
	lineID := int64(line["id"].(float64))
	contributions := request(t, handler, http.MethodGet,
		fmt.Sprintf("/api/week/current/groceries/%d/contributions", lineID), "")
	if contributions.Code != http.StatusOK || len(decodeObject(t, contributions)["contributions"].([]any)) == 0 {
		t.Fatalf("contributions status = %d, body %s", contributions.Code, contributions.Body.String())
	}
	completed := request(t, handler, http.MethodPatch,
		fmt.Sprintf("/api/week/current/groceries/%d", lineID), `{"completed":true}`)
	if completed.Code != http.StatusOK {
		t.Fatalf("complete status = %d, body %s", completed.Code, completed.Body.String())
	}
	added := request(t, handler, http.MethodPost, "/api/week/current/groceries", `{"name":"Paper towels"}`)
	if added.Code != http.StatusOK || len(decodeObject(t, added)["lines"].([]any)) != len(lines)+1 {
		t.Fatalf("manual add status = %d, body %s", added.Code, added.Body.String())
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
	db := testdatabase.Open(t)
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
