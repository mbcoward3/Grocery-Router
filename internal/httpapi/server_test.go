package httpapi_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/mbcoward3/grocery-router/internal/httpapi"
	"github.com/mbcoward3/grocery-router/internal/ingest"
	"github.com/mbcoward3/grocery-router/internal/testdatabase"
	"github.com/mbcoward3/grocery-router/internal/week"
)

type zeroPicker struct{}

func (zeroPicker) IntN(int) (int, error) { return 0, nil }

func TestWeekAPIEmptyGenerateAndMutate(t *testing.T) {
	handler := testHandler(t)

	empty := request(t, handler, http.MethodGet, "/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/week/current", "")
	if empty.Code != http.StatusNotFound {
		t.Fatalf("empty week status = %d, body %s", empty.Code, empty.Body.String())
	}
	assertErrorCode(t, empty, "no_current_week")

	generated := request(t, handler, http.MethodPost, "/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/week/current/generate", `{"recipeCount":3}`)
	if generated.Code != http.StatusOK {
		t.Fatalf("generate status = %d, body %s", generated.Code, generated.Body.String())
	}
	weekBody := decodeObject(t, generated)
	// A plan starts on the day it is made and has no end while it is current.
	if weekBody["startsOn"] != "2026-08-19" {
		t.Fatalf("startsOn = %#v", weekBody["startsOn"])
	}
	if endsOn, ok := weekBody["endsOn"]; !ok || endsOn != nil {
		t.Fatalf("endsOn = %#v, want null", endsOn)
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

	added := request(t, handler, http.MethodPost, "/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/week/current/recipes",
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
		fmt.Sprintf("/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/week/current/recipes/%d", occurrenceID), "")
	if removed.Code != http.StatusOK || len(decodeObject(t, removed)["recipes"].([]any)) != 3 {
		t.Fatalf("remove status = %d, body %s", removed.Code, removed.Body.String())
	}
}

func TestRecipeDetailAndGroceryAPI(t *testing.T) {
	handler := testHandler(t)
	generated := request(t, handler, http.MethodPost, "/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/week/current/generate", `{"recipeCount":2}`)
	if generated.Code != http.StatusOK {
		t.Fatalf("generate status = %d, body %s", generated.Code, generated.Body.String())
	}
	firstRecipe := decodeObject(t, generated)["recipes"].([]any)[0].(map[string]any)["recipe"].(map[string]any)
	recipeID := int64(firstRecipe["id"].(float64))
	detail := request(t, handler, http.MethodGet, fmt.Sprintf("/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/recipes/%d", recipeID), "")
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status = %d, body %s", detail.Code, detail.Body.String())
	}
	detailBody := decodeObject(t, detail)
	if detailBody["name"] == "" || len(detailBody["ingredientSections"].([]any)) == 0 || len(detailBody["instructionSections"].([]any)) == 0 {
		t.Fatalf("incomplete recipe detail = %#v", detailBody)
	}

	groceries := request(t, handler, http.MethodGet, "/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/week/current/groceries", "")
	groceryBody := decodeObject(t, groceries)
	lines := groceryBody["lines"].([]any)
	if groceries.Code != http.StatusOK || len(lines) == 0 {
		t.Fatalf("groceries status = %d, body %s", groceries.Code, groceries.Body.String())
	}
	line := lines[0].(map[string]any)
	if recipeNames, ok := line["recipeNames"].([]any); !ok || len(recipeNames) == 0 {
		t.Fatalf("grocery line does not expose recipe names: %#v", line)
	}
	lineID := int64(line["id"].(float64))
	contributions := request(t, handler, http.MethodGet,
		fmt.Sprintf("/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/week/current/groceries/%d/contributions", lineID), "")
	if contributions.Code != http.StatusOK || len(decodeObject(t, contributions)["contributions"].([]any)) == 0 {
		t.Fatalf("contributions status = %d, body %s", contributions.Code, contributions.Body.String())
	}
	completed := request(t, handler, http.MethodPatch,
		fmt.Sprintf("/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/week/current/groceries/%d", lineID), `{"completed":true}`)
	if completed.Code != http.StatusOK {
		t.Fatalf("complete status = %d, body %s", completed.Code, completed.Body.String())
	}
	added := request(t, handler, http.MethodPost, "/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/week/current/groceries", `{"name":"Paper towels"}`)
	addedLines := decodeObject(t, added)["lines"].([]any)
	if added.Code != http.StatusOK || len(addedLines) != len(lines)+1 {
		t.Fatalf("manual add status = %d, body %s", added.Code, added.Body.String())
	}
	var manualLine map[string]any
	for _, value := range addedLines {
		candidate := value.(map[string]any)
		if candidate["name"] == "Paper towels" {
			manualLine = candidate
			break
		}
	}
	if manualLine == nil {
		t.Fatal("manual grocery line was not returned")
	}
	if recipeNames, ok := manualLine["recipeNames"].([]any); !ok || len(recipeNames) != 0 {
		t.Fatalf("manual grocery line recipe names = %#v", manualLine["recipeNames"])
	}
	linePath := fmt.Sprintf("/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/week/current/groceries/%d", lineID)
	removed := request(t, handler, http.MethodDelete, linePath, "")
	if removed.Code != http.StatusOK || !lineRemoved(t, removed, lineID) {
		t.Fatalf("remove status = %d, body %s", removed.Code, removed.Body.String())
	}
	if invalid := requestExpecting(t, handler, nonconformingRequest, http.MethodPatch, linePath, `{"removed":true}`); invalid.Code != http.StatusBadRequest {
		t.Fatalf("removing through PATCH status = %d, want 400", invalid.Code)
	}
	restored := request(t, handler, http.MethodPatch, linePath, `{"removed":false}`)
	if restored.Code != http.StatusOK || lineRemoved(t, restored, lineID) {
		t.Fatalf("restore status = %d, body %s", restored.Code, restored.Body.String())
	}
}

func lineRemoved(t *testing.T, response *httptest.ResponseRecorder, lineID int64) bool {
	t.Helper()
	for _, value := range decodeObject(t, response)["lines"].([]any) {
		line := value.(map[string]any)
		if int64(line["id"].(float64)) == lineID {
			return line["removed"].(bool)
		}
	}
	t.Fatalf("line %d missing from response", lineID)
	return false
}

func TestWeekHistoryListsAndReturnsRetainedState(t *testing.T) {
	db := testDB(t)
	service := week.NewService(db, zeroPicker{}).ForHousehold("c0a7a2d8-669b-4e47-91c1-4d9a32f339d5")
	past := time.Date(2026, time.August, 12, 12, 0, 0, 0, time.Local)
	view, err := service.Generate(context.Background(), past, 2)
	if err != nil {
		t.Fatal(err)
	}
	checklist, err := service.Checklist(context.Background())
	if err != nil || len(checklist.Lines) == 0 {
		t.Fatalf("past checklist: %v, lines = %d", err, len(checklist.Lines))
	}
	if len(checklist.Lines) < 3 {
		t.Fatalf("past checklist needs representative lines, got %d", len(checklist.Lines))
	}
	if err := service.SetLineCompleted(context.Background(), checklist.Lines[0].ID, true); err != nil {
		t.Fatal(err)
	}
	if err := service.SetLineOverride(context.Background(), checklist.Lines[1].ID, "2 bags"); err != nil {
		t.Fatal(err)
	}
	if err := service.SetLineRemoved(context.Background(), checklist.Lines[2].ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AddManualLine(context.Background(), "Paper towels"); err != nil {
		t.Fatal(err)
	}
	if err := service.Close(context.Background(), past.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	handler := handlerForDB(t, db)
	history := request(t, handler, http.MethodGet, "/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/weeks/history", "")
	if history.Code != http.StatusOK {
		t.Fatalf("history status = %d, body %s", history.Code, history.Body.String())
	}
	weeks := decodeObject(t, history)["weeks"].([]any)
	if len(weeks) != 1 {
		t.Fatalf("history weeks = %d", len(weeks))
	}
	summary := weeks[0].(map[string]any)
	if summary["startsOn"] != "2026-08-12" || summary["endsOn"] != "2026-08-12" || summary["recipeCount"] != float64(2) || summary["completedCount"] != float64(1) {
		t.Fatalf("history summary = %#v", summary)
	}

	detail := request(t, handler, http.MethodGet, fmt.Sprintf("/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/weeks/history/%d", view.Week.ID), "")
	if detail.Code != http.StatusOK {
		t.Fatalf("history detail status = %d, body %s", detail.Code, detail.Body.String())
	}
	body := decodeObject(t, detail)
	if pastWeek := body["week"].(map[string]any); len(pastWeek["recipes"].([]any)) != 2 || pastWeek["endsOn"] != "2026-08-12" {
		t.Fatalf("history detail week = %#v", body["week"])
	}
	lines := body["groceries"].(map[string]any)["lines"].([]any)
	var foundCompleted, foundAdjusted, foundRemoved, foundManual bool
	for _, value := range lines {
		line := value.(map[string]any)
		foundCompleted = foundCompleted || line["completed"] == true
		foundAdjusted = foundAdjusted || line["quantity"] == "2 bags"
		foundRemoved = foundRemoved || line["removed"] == true
		foundManual = foundManual || line["origin"] == "manual"
	}
	if !foundCompleted || !foundAdjusted || !foundRemoved || !foundManual {
		t.Fatalf("history groceries did not retain all states: %#v", lines)
	}
}

func TestCloseCurrentWeekAPI(t *testing.T) {
	handler := testHandler(t)
	const root = "/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5"

	missing := request(t, handler, http.MethodPost, root+"/week/current/close", "")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("close without a week status = %d, body %s", missing.Code, missing.Body.String())
	}
	assertErrorCode(t, missing, "no_current_week")

	generated := request(t, handler, http.MethodPost, root+"/week/current/generate", `{"recipeCount":2}`)
	if generated.Code != http.StatusOK {
		t.Fatalf("generate status = %d, body %s", generated.Code, generated.Body.String())
	}
	weekID := int64(decodeObject(t, generated)["id"].(float64))
	groceries := request(t, handler, http.MethodGet, root+"/week/current/groceries", "")
	lineID := int64(decodeObject(t, groceries)["lines"].([]any)[0].(map[string]any)["id"].(float64))
	if completed := request(t, handler, http.MethodPatch, fmt.Sprintf("%s/week/current/groceries/%d", root, lineID), `{"completed":true}`); completed.Code != http.StatusOK {
		t.Fatalf("complete status = %d, body %s", completed.Code, completed.Body.String())
	}

	closed := request(t, handler, http.MethodPost, root+"/week/current/close", "")
	if closed.Code != http.StatusNoContent || closed.Body.Len() != 0 {
		t.Fatalf("close status = %d, body %q", closed.Code, closed.Body.String())
	}
	current := request(t, handler, http.MethodGet, root+"/week/current", "")
	if current.Code != http.StatusNotFound {
		t.Fatalf("current after close status = %d, body %s", current.Code, current.Body.String())
	}
	assertErrorCode(t, current, "no_current_week")

	history := request(t, handler, http.MethodGet, root+"/weeks/history", "")
	weeks := decodeObject(t, history)["weeks"].([]any)
	if history.Code != http.StatusOK || len(weeks) != 1 || weeks[0].(map[string]any)["id"] != float64(weekID) ||
		weeks[0].(map[string]any)["completedCount"] != float64(1) {
		t.Fatalf("history after close status = %d, body %s", history.Code, history.Body.String())
	}
	detail := request(t, handler, http.MethodGet, fmt.Sprintf("%s/weeks/history/%d", root, weekID), "")
	if detail.Code != http.StatusOK || !lineCompleted(t, detail, lineID) {
		t.Fatalf("closed week detail status = %d, body %s", detail.Code, detail.Body.String())
	}

	again := request(t, handler, http.MethodPost, root+"/week/current/close", "")
	if again.Code != http.StatusNotFound {
		t.Fatalf("second close status = %d, body %s", again.Code, again.Body.String())
	}
	assertErrorCode(t, again, "no_current_week")

	// Reopening makes the closed plan current again, with its checklist as it was, and moves
	// the plan that was current to History.
	next := request(t, handler, http.MethodPost, root+"/week/current/generate", `{"recipeCount":1}`)
	if next.Code != http.StatusOK {
		t.Fatalf("generate next status = %d, body %s", next.Code, next.Body.String())
	}
	nextID := decodeObject(t, next)["id"]
	reopened := request(t, handler, http.MethodPost, fmt.Sprintf("%s/weeks/history/%d/reopen", root, weekID), "")
	reopenedBody := decodeObject(t, reopened)
	if reopened.Code != http.StatusOK || reopenedBody["id"] != float64(weekID) || reopenedBody["endsOn"] != nil {
		t.Fatalf("reopen status = %d, body %s", reopened.Code, reopened.Body.String())
	}
	if current := request(t, handler, http.MethodGet, root+"/week/current", ""); decodeObject(t, current)["id"] != float64(weekID) {
		t.Fatalf("current after reopen = %s", current.Body.String())
	}
	if groceries := request(t, handler, http.MethodGet, root+"/week/current/groceries", ""); !strings.Contains(groceries.Body.String(), `"completed":true`) {
		t.Fatalf("reopened groceries lost their state: %s", groceries.Body.String())
	}
	history = request(t, handler, http.MethodGet, root+"/weeks/history", "")
	weeks = decodeObject(t, history)["weeks"].([]any)
	if len(weeks) != 1 || weeks[0].(map[string]any)["id"] != nextID {
		t.Fatalf("history after reopen = %s", history.Body.String())
	}
	// The current plan is not in History, so it cannot be reopened.
	notPast := request(t, handler, http.MethodPost, fmt.Sprintf("%s/weeks/history/%d/reopen", root, weekID), "")
	if notPast.Code != http.StatusNotFound {
		t.Fatalf("reopen current plan status = %d, body %s", notPast.Code, notPast.Body.String())
	}
	assertErrorCode(t, notPast, "past_week_not_found")
}

func lineCompleted(t *testing.T, response *httptest.ResponseRecorder, lineID int64) bool {
	t.Helper()
	for _, value := range decodeObject(t, response)["groceries"].(map[string]any)["lines"].([]any) {
		line := value.(map[string]any)
		if int64(line["id"].(float64)) == lineID {
			return line["completed"] == true
		}
	}
	return false
}

func TestRecipesAPIAndRequestValidation(t *testing.T) {
	handler := testHandler(t)

	response := request(t, handler, http.MethodGet, "/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/recipes", "")
	if response.Code != http.StatusOK {
		t.Fatalf("recipes status = %d, body %s", response.Code, response.Body.String())
	}
	if recipes := decodeObject(t, response)["recipes"].([]any); len(recipes) != 24 {
		t.Fatalf("verified recipes = %d", len(recipes))
	}

	invalid := requestExpecting(t, handler, nonconformingRequest, http.MethodPost, "/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/week/current/generate", `{"recipeCount":3,"surprise":true}`)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid request status = %d, body %s", invalid.Code, invalid.Body.String())
	}
	assertErrorCode(t, invalid, "invalid_json")

	wrongMethod := requestExpecting(t, handler, undocumentedOperation, http.MethodDelete, "/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/week/current/generate", "")
	if wrongMethod.Code != http.StatusMethodNotAllowed {
		t.Fatalf("wrong method status = %d", wrongMethod.Code)
	}
}

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	return handlerForDB(t, testDB(t))
}

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	db := testdatabase.Open(t)
	documents, err := ingest.ReadDirectory(filepath.Join("..", "..", "corpus", "recipes"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ingest.Import(context.Background(), db, "c0a7a2d8-669b-4e47-91c1-4d9a32f339d5", documents); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestRequestsUseAuthorizedPathHousehold(t *testing.T) {
	db := testDB(t)
	const householdA = "c0a7a2d8-669b-4e47-91c1-4d9a32f339d5"
	const householdB = "711e58fa-2c03-44cc-83aa-432eab3158fb"
	const missing = "c7c7d0df-c81c-41d9-b95b-c131bdd605bd"
	statements := []struct {
		query string
		args  []any
	}{
		{"INSERT INTO households (id, name) VALUES ($1, 'Second household')", []any{householdB}},
		{"INSERT INTO store_sections (household_id, key, name) VALUES ($1, 'other', 'Other')", []any{householdB}},
		{"INSERT INTO grocery_items (household_id, key, name, store_section_id, shopping_mode) SELECT $1, 'second-item', 'Second item', id, 'counted' FROM store_sections WHERE household_id = $1 AND key = 'other'", []any{householdB}},
		{"INSERT INTO recipes (household_id, key, name) VALUES ($1, 'second-recipe', 'Second recipe')", []any{householdB}},
		{"INSERT INTO recipe_sources (recipe_id, relationship, attribution, is_primary, position) SELECT id, 'source', 'Household', 1, 0 FROM recipes WHERE household_id = $1 AND key = 'second-recipe'", []any{householdB}},
		{"INSERT INTO recipe_ingredient_sections (household_id, recipe_id, name, position) SELECT $1, id, 'Ingredients', 0 FROM recipes WHERE household_id = $1 AND key = 'second-recipe'", []any{householdB}},
		{"INSERT INTO recipe_ingredients (household_id, section_id, grocery_item_id, position, source_text, quantity_kind) SELECT $1, ris.id, gi.id, 0, 'Second item', 'unspecified' FROM recipe_ingredient_sections ris JOIN grocery_items gi ON gi.household_id = $1 AND gi.key = 'second-item' WHERE ris.household_id = $1", []any{householdB}},
		{"INSERT INTO recipe_instruction_sections (recipe_id, name, position) SELECT id, 'Steps', 0 FROM recipes WHERE household_id = $1 AND key = 'second-recipe'", []any{householdB}},
		{"INSERT INTO recipe_steps (section_id, position, instruction) SELECT ris.id, 0, 'Prepare item.' FROM recipe_instruction_sections ris JOIN recipes r ON r.id = ris.recipe_id WHERE r.household_id = $1", []any{householdB}},
		{"UPDATE recipes SET status = 'reviewable' WHERE household_id = $1 AND key = 'second-recipe'", []any{householdB}},
		{"UPDATE recipes SET status = 'verified', verified_at = '2026-08-01' WHERE household_id = $1 AND key = 'second-recipe'", []any{householdB}},
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement.query, statement.args...); err != nil {
			t.Fatalf("seed second household: %v: %s", err, statement.query)
		}
	}
	handler := contractHandler(t, httpapi.New(db, week.NewService(db, zeroPicker{}), func() time.Time {
		return time.Date(2026, time.August, 19, 12, 0, 0, 0, time.UTC)
	}, func(_ context.Context, id string) bool { return id == householdB }).Handler())

	path := func(id, suffix string) string { return "/api/v2/households/" + id + suffix }
	for _, id := range []string{householdA, missing} {
		response := request(t, handler, http.MethodGet, path(id, "/recipes"), "")
		if response.Code != http.StatusNotFound {
			t.Fatalf("household %s recipes status = %d", id, response.Code)
		}
		assertErrorCode(t, response, "not_found")
	}
	recipes := request(t, handler, http.MethodGet, path(householdB, "/recipes"), "")
	if recipes.Code != http.StatusOK {
		t.Fatalf("second household recipes status = %d, body %s", recipes.Code, recipes.Body.String())
	}
	listed, ok := decodeObject(t, recipes)["recipes"].([]any)
	if !ok || len(listed) != 1 || listed[0].(map[string]any)["name"] != "Second recipe" {
		t.Fatalf("second household recipe list = %s", recipes.Body.String())
	}
	generated := request(t, handler, http.MethodPost, path(householdB, "/week/current/generate"), `{"recipeCount":1}`)
	if generated.Code != http.StatusOK {
		t.Fatalf("second household generate status = %d, body %s", generated.Code, generated.Body.String())
	}
	var countA, countB int
	if err := db.QueryRow("SELECT count(*) FROM weeks WHERE household_id = $1", householdA).Scan(&countA); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM weeks WHERE household_id = $1", householdB).Scan(&countB); err != nil {
		t.Fatal(err)
	}
	if countA != 0 || countB != 1 {
		t.Fatalf("week counts: first household %d, second household %d", countA, countB)
	}

	var recipeA int64
	if err := db.QueryRow("SELECT id FROM recipes WHERE household_id = $1 AND status = 'verified' ORDER BY id LIMIT 1", householdA).Scan(&recipeA); err != nil {
		t.Fatal(err)
	}
	assertRecipeNotFound := func(name string, response *httptest.ResponseRecorder) {
		t.Helper()
		if response.Code != http.StatusUnprocessableEntity {
			t.Fatalf("%s status = %d, body %s", name, response.Code, response.Body.String())
		}
		assertErrorCode(t, response, "week_operation_failed")
		if message := decodeObject(t, response)["error"].(map[string]any)["message"]; message != "recipe not found" {
			t.Fatalf("%s message = %#v", name, message)
		}
	}
	assertRecipeNotFound("cross-household add", request(t, handler, http.MethodPost, path(householdB, "/week/current/recipes"),
		fmt.Sprintf(`{"recipeId":%d}`, recipeA)))
	occurrences := decodeObject(t, generated)["recipes"].([]any)
	if len(occurrences) != 1 {
		t.Fatalf("second household occurrences = %#v", occurrences)
	}
	occurrenceID := int64(occurrences[0].(map[string]any)["id"].(float64))
	assertRecipeNotFound("missing recipe swap", request(t, handler, http.MethodPut,
		path(householdB, fmt.Sprintf("/week/current/recipes/%d", occurrenceID)), `{"recipeId":999999}`))
}

func handlerForDB(t *testing.T, db *sql.DB) http.Handler {
	t.Helper()
	service := week.NewService(db, zeroPicker{})
	now := func() time.Time {
		return time.Date(2026, time.August, 19, 12, 0, 0, 0, time.Local)
	}
	return contractHandler(t, httpapi.New(db, service, now, func(_ context.Context, id string) bool {
		return id == "c0a7a2d8-669b-4e47-91c1-4d9a32f339d5"
	}).Handler())
}

// request sends a request that must conform to api/openapi.yaml.
func request(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return requestExpecting(t, handler, conformingRequest, method, path, body)
}

// requestExpecting sends a request with an explicit contract expectation.
func requestExpecting(t *testing.T, handler http.Handler, expectation contractExpectation, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	req := withContractExpectation(httptest.NewRequest(method, path, strings.NewReader(body)), expectation)
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

func TestShoppingDiscoveryAndOrderAPI(t *testing.T) {
	handler := testHandler(t)
	const root = "/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5"
	matches := request(t, handler, http.MethodGet, root+"/grocery-items?q=mil", "")
	if matches.Code != http.StatusOK || len(decodeObject(t, matches)["items"].([]any)) == 0 {
		t.Fatalf("matches: %s", matches.Body)
	}
	sections := request(t, handler, http.MethodGet, root+"/store-sections", "")
	rows := decodeObject(t, sections)["sections"].([]any)
	if sections.Code != http.StatusOK || rows[0].(map[string]any)["name"] != "Produce" {
		t.Fatalf("default sections: %s", sections.Body)
	}
	var ids []int64
	for i := len(rows) - 1; i >= 0; i-- {
		ids = append(ids, int64(rows[i].(map[string]any)["id"].(float64)))
	}
	body, err := json.Marshal(map[string]any{"sectionIds": ids})
	if err != nil {
		t.Fatal(err)
	}
	saved := request(t, handler, http.MethodPut, root+"/store-sections", string(body))
	if saved.Code != http.StatusOK || int64(decodeObject(t, saved)["sections"].([]any)[0].(map[string]any)["id"].(float64)) != ids[0] {
		t.Fatalf("save order: %s", saved.Body)
	}
	invalid := requestExpecting(t, handler, nonconformingRequest, http.MethodPut, root+"/store-sections", `{"sectionIds":[-1]}`)
	if invalid.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid order status: %d", invalid.Code)
	}
	inaccessible := request(t, handler, http.MethodGet, "/api/v2/households/00000000-0000-0000-0000-000000000000/grocery-items?q=mil", "")
	if inaccessible.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant search status: %d", inaccessible.Code)
	}
	assertErrorCode(t, inaccessible, "not_found")
	reset := request(t, handler, http.MethodPut, root+"/store-sections", `{"reset":true}`)
	if reset.Code != http.StatusOK || decodeObject(t, reset)["sections"].([]any)[0].(map[string]any)["name"] != "Produce" {
		t.Fatalf("reset: %s", reset.Body)
	}
}

func TestInternalErrorsReturnGenericBody(t *testing.T) {
	db := testDB(t)
	handler := handlerForDB(t, db)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	response := request(t, handler, http.MethodGet, "/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/week/current", "")
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("closed database status = %d, body %s", response.Code, response.Body.String())
	}
	assertErrorCode(t, response, "internal_error")
	body := response.Body.String()
	for _, leak := range []string{"sql", "pq", "pgx", "SQLSTATE"} {
		if strings.Contains(body, leak) {
			t.Fatalf("internal error body leaks %q: %s", leak, body)
		}
	}
}

func TestGenerateBeyondCorpusIsValidationError(t *testing.T) {
	handler := testHandler(t)
	response := request(t, handler, http.MethodPost, "/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/week/current/generate", `{"recipeCount":25}`)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("oversized generate status = %d, body %s", response.Code, response.Body.String())
	}
	assertErrorCode(t, response, "week_operation_failed")
	message := decodeObject(t, response)["error"].(map[string]any)["message"]
	if message != "requested 25 recipes but only 24 verified recipes are available" {
		t.Fatalf("oversized generate message = %#v", message)
	}
}

func TestAddMissingRecipeIsValidationError(t *testing.T) {
	handler := testHandler(t)
	const root = "/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5"
	if generated := request(t, handler, http.MethodPost, root+"/week/current/generate", `{"recipeCount":1}`); generated.Code != http.StatusOK {
		t.Fatalf("generate status = %d, body %s", generated.Code, generated.Body.String())
	}
	response := request(t, handler, http.MethodPost, root+"/week/current/recipes", `{"recipeId":999999}`)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("missing recipe status = %d, body %s", response.Code, response.Body.String())
	}
	assertErrorCode(t, response, "week_operation_failed")
	message := decodeObject(t, response)["error"].(map[string]any)["message"]
	if message != "recipe not found" {
		t.Fatalf("missing recipe message = %#v", message)
	}
}

func TestSwapAndRandomSwapAPI(t *testing.T) {
	handler := testHandler(t)
	const root = "/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5"
	generated := request(t, handler, http.MethodPost, root+"/week/current/generate", `{"recipeCount":2}`)
	if generated.Code != http.StatusOK {
		t.Fatalf("generate status = %d, body %s", generated.Code, generated.Body.String())
	}
	occurrences := decodeObject(t, generated)["recipes"].([]any)
	first := occurrences[0].(map[string]any)
	second := occurrences[1].(map[string]any)
	occurrenceID := int64(first["id"].(float64))
	secondRecipeID := int64(second["recipe"].(map[string]any)["id"].(float64))

	swapped := request(t, handler, http.MethodPut, fmt.Sprintf("%s/week/current/recipes/%d", root, occurrenceID),
		fmt.Sprintf(`{"recipeId":%d}`, secondRecipeID))
	if swapped.Code != http.StatusOK {
		t.Fatalf("swap status = %d, body %s", swapped.Code, swapped.Body.String())
	}
	swappedFirst := decodeObject(t, swapped)["recipes"].([]any)[0].(map[string]any)
	if int64(swappedFirst["recipe"].(map[string]any)["id"].(float64)) != secondRecipeID {
		t.Fatalf("swapped occurrence = %#v", swappedFirst)
	}

	swappedID := int64(swappedFirst["id"].(float64))
	random := request(t, handler, http.MethodPost, fmt.Sprintf("%s/week/current/recipes/%d/random-swap", root, swappedID), "")
	if random.Code != http.StatusOK {
		t.Fatalf("random swap status = %d, body %s", random.Code, random.Body.String())
	}
	randomRecipes := decodeObject(t, random)["recipes"].([]any)
	if len(randomRecipes) != 2 {
		t.Fatalf("recipes after random swap = %d", len(randomRecipes))
	}
	randomFirst := randomRecipes[0].(map[string]any)
	if int64(randomFirst["id"].(float64)) != swappedID {
		t.Fatalf("random swap moved the occurrence: %#v", randomFirst)
	}
	if int64(randomFirst["recipe"].(map[string]any)["id"].(float64)) == secondRecipeID {
		t.Fatalf("random swap kept recipe %d: %#v", secondRecipeID, randomFirst)
	}

	missing := request(t, handler, http.MethodPost, root+"/week/current/recipes/999999/random-swap", "")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing occurrence status = %d, body %s", missing.Code, missing.Body.String())
	}
	assertErrorCode(t, missing, "occurrence_not_found")
}

func TestGroceriesWithoutCurrentWeek(t *testing.T) {
	handler := testHandler(t)
	const groceries = "/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/week/current/groceries"
	for _, tc := range []struct{ method, body string }{
		{http.MethodGet, ""},
		{http.MethodPost, `{"name":"Paper towels"}`},
	} {
		response := request(t, handler, tc.method, groceries, tc.body)
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s groceries without a week status = %d, body %s", tc.method, response.Code, response.Body.String())
		}
		assertErrorCode(t, response, "no_current_week")
	}
}

// A plan is dated in the household's zone: Sunday 01:00 UTC is still Saturday in Chicago.
func TestPlanDatesFollowHouseholdTimezone(t *testing.T) {
	const household = "c0a7a2d8-669b-4e47-91c1-4d9a32f339d5"
	db := testDB(t)
	handler := contractHandler(t, httpapi.New(db, week.NewService(db, zeroPicker{}), func() time.Time {
		return time.Date(2026, time.August, 16, 1, 0, 0, 0, time.UTC)
	}, func(_ context.Context, id string) bool { return id == household }).Handler())
	path := func(suffix string) string { return "/api/v2/households/" + household + suffix }

	for _, zone := range []struct{ name, date string }{{"America/Chicago", "2026-08-15"}, {"UTC", "2026-08-16"}} {
		if _, err := db.Exec("UPDATE households SET timezone = $1 WHERE id = $2", zone.name, household); err != nil {
			t.Fatal(err)
		}
		generated := request(t, handler, http.MethodPost, path("/week/current/generate"), `{"recipeCount":1}`)
		if generated.Code != http.StatusOK || decodeObject(t, generated)["startsOn"] != zone.date {
			t.Fatalf("%s generate status = %d, body %s", zone.name, generated.Code, generated.Body.String())
		}
		if closed := request(t, handler, http.MethodPost, path("/week/current/close"), ""); closed.Code != http.StatusNoContent {
			t.Fatalf("%s close status = %d", zone.name, closed.Code)
		}
		history := request(t, handler, http.MethodGet, path("/weeks/history"), "")
		latest := decodeObject(t, history)["weeks"].([]any)[0].(map[string]any)
		if latest["startsOn"] != zone.date || latest["endsOn"] != zone.date {
			t.Fatalf("%s history = %s", zone.name, history.Body.String())
		}
	}
}

func TestInvalidPathIDsAndBlankManualLine(t *testing.T) {
	handler := testHandler(t)
	const root = "/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5"
	for _, call := range []struct{ method, path string }{
		{http.MethodGet, root + "/recipes/abc"},
		{http.MethodGet, root + "/weeks/history/abc"},
		{http.MethodDelete, root + "/week/current/groceries/abc"},
		{http.MethodGet, root + "/week/current/groceries/abc/contributions"},
	} {
		response := requestExpecting(t, handler, nonconformingRequest, call.method, call.path, "")
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s %s status = %d, body %s", call.method, call.path, response.Code, response.Body.String())
		}
		assertErrorCode(t, response, "invalid_request")
	}

	if generated := request(t, handler, http.MethodPost, root+"/week/current/generate", `{"recipeCount":1}`); generated.Code != http.StatusOK {
		t.Fatalf("generate status = %d, body %s", generated.Code, generated.Body.String())
	}
	blank := request(t, handler, http.MethodPost, root+"/week/current/groceries", `{"name":" "}`)
	if blank.Code != http.StatusUnprocessableEntity {
		t.Fatalf("blank manual line status = %d, body %s", blank.Code, blank.Body.String())
	}
	assertErrorCode(t, blank, "grocery_operation_failed")
}
