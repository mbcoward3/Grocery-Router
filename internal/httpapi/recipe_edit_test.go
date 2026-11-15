package httpapi_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/httpapi"
	"github.com/mbcoward3/grocery-router/internal/ingest"
	"github.com/mbcoward3/grocery-router/internal/week"
)

// editClock is when the edits in these tests are saved.
var editClock = time.Date(2026, time.August, 19, 12, 0, 0, 0, time.UTC)

func seedUser(t *testing.T, db *sql.DB, displayName string) string {
	t.Helper()
	var id string
	if err := db.QueryRow(`INSERT INTO app_users (primary_email, display_name) VALUES ($1, $2) RETURNING id::text`,
		strings.ToLower(strings.ReplaceAll(displayName, " ", "."))+"@example.com", displayName).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// editHandler serves the API as the given signed-in user.
func editHandler(t *testing.T, db *sql.DB, userID string) http.Handler {
	t.Helper()
	return contractHandler(t, httpapi.New(db, week.NewService(db, zeroPicker{}), func() time.Time { return editClock },
		func(_ context.Context, id string) bool { return id == snapshotHousehold },
		httpapi.WithCurrentUser(func(context.Context) (string, bool) { return userID, true }),
	).Handler())
}

func recipePath(recipeID int64) string {
	return fmt.Sprintf("/api/v2/households/%s/recipes/%d", snapshotHousehold, recipeID)
}

func groceriesPath() string {
	return "/api/v2/households/" + snapshotHousehold + "/week/current/groceries"
}

func recipeDetail(t *testing.T, handler http.Handler, recipeID int64) map[string]any {
	t.Helper()
	response := request(t, handler, http.MethodGet, recipePath(recipeID), "")
	if response.Code != http.StatusOK {
		t.Fatalf("recipe detail status = %d, body %s", response.Code, response.Body.String())
	}
	return decodeObject(t, response)
}

// editBody turns a RecipeDetail response into the RecipeEdit request that saves it unchanged.
func editBody(detail map[string]any) map[string]any {
	sections := []any{}
	for _, value := range detail["ingredientSections"].([]any) {
		section := value.(map[string]any)
		lines := []any{}
		for _, value := range section["ingredients"].([]any) {
			ingredient := value.(map[string]any)
			lines = append(lines, map[string]any{
				"sourceText": ingredient["sourceText"], "quantity": ingredient["quantity"],
				"preparation": ingredient["preparation"], "optional": ingredient["optional"],
				"displayNote": ingredient["displayNote"], "onGroceryList": ingredient["onGroceryList"],
				"groceryItemId": ingredient["groceryItemId"],
			})
		}
		sections = append(sections, map[string]any{"name": section["name"], "ingredients": lines})
	}
	instructions := []any{}
	for _, value := range detail["instructionSections"].([]any) {
		section := value.(map[string]any)
		steps := []any{}
		for _, step := range section["steps"].([]any) {
			steps = append(steps, map[string]any{"instruction": step.(map[string]any)["instruction"]})
		}
		instructions = append(instructions, map[string]any{"name": section["name"], "steps": steps})
	}
	return map[string]any{
		"name": detail["name"], "imageUrl": detail["imageUrl"], "yield": detail["yield"],
		"handsOn": detail["handsOn"], "unattended": detail["unattended"],
		"ingredientSections": sections, "instructionSections": instructions,
	}
}

func line(body map[string]any, section, index int) map[string]any {
	return body["ingredientSections"].([]any)[section].(map[string]any)["ingredients"].([]any)[index].(map[string]any)
}

func putRecipe(t *testing.T, handler http.Handler, recipeID int64, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return request(t, handler, http.MethodPut, recipePath(recipeID), string(encoded))
}

// currentPlan generates a one-recipe current plan and returns its recipe's ID.
func currentPlan(t *testing.T, handler http.Handler) int64 {
	t.Helper()
	generated := request(t, handler, http.MethodPost, "/api/v2/households/"+snapshotHousehold+"/week/current/generate", `{"recipeCount":1}`)
	if generated.Code != http.StatusOK {
		t.Fatalf("generate status = %d, body %s", generated.Code, generated.Body.String())
	}
	recipe := decodeObject(t, generated)["recipes"].([]any)[0].(map[string]any)["recipe"].(map[string]any)
	return int64(recipe["id"].(float64))
}

// uniqueMeasuredLine finds an exact, unit-measured line on the grocery list whose grocery item
// no other line of the recipe uses, so its list row shows only that line's amount.
func uniqueMeasuredLine(t *testing.T, detail map[string]any) (section, index int, itemName string) {
	t.Helper()
	uses := map[float64]int{}
	for _, value := range detail["ingredientSections"].([]any) {
		for _, ingredient := range value.(map[string]any)["ingredients"].([]any) {
			if id, ok := ingredient.(map[string]any)["groceryItemId"].(float64); ok {
				uses[id]++
			}
		}
	}
	for sectionIndex, value := range detail["ingredientSections"].([]any) {
		for lineIndex, value := range value.(map[string]any)["ingredients"].([]any) {
			ingredient := value.(map[string]any)
			quantity := ingredient["quantity"].(map[string]any)
			if quantity["kind"] == "exact" && quantity["unit"] != nil && ingredient["onGroceryList"] == true &&
				ingredient["optional"] == false && uses[ingredient["groceryItemId"].(float64)] == 1 {
				return sectionIndex, lineIndex, ingredient["shoppingItem"].(string)
			}
		}
	}
	t.Fatalf("recipe %v has no uniquely measured line", detail["name"])
	return 0, 0, ""
}

func groceryLineNamed(t *testing.T, response *httptest.ResponseRecorder, name string) map[string]any {
	t.Helper()
	for _, value := range decodeObject(t, response)["lines"].([]any) {
		if line := value.(map[string]any); line["name"] == name {
			return line
		}
	}
	t.Fatalf("no grocery line %q in %s", name, response.Body.String())
	return nil
}

func TestEditRecipeQuantityRebuildsCurrentPlan(t *testing.T) {
	db := testDB(t)
	userID := seedUser(t, db, "Pat Example")
	handler := editHandler(t, db, userID)
	recipeID := currentPlan(t, handler)
	before := recipeDetail(t, handler, recipeID)
	section, index, itemName := uniqueMeasuredLine(t, before)

	itemLine := groceryLineNamed(t, request(t, handler, http.MethodGet, groceriesPath(), ""), itemName)
	lineID := int64(itemLine["id"].(float64))
	checked := request(t, handler, http.MethodPatch, fmt.Sprintf("%s/%d", groceriesPath(), lineID), `{"completed":true}`)
	if checked.Code != http.StatusOK {
		t.Fatalf("check status = %d, body %s", checked.Code, checked.Body.String())
	}

	body := editBody(before)
	line(body, section, index)["quantity"].(map[string]any)["amount"] = "7"
	saved := putRecipe(t, handler, recipeID, body)
	if saved.Code != http.StatusOK {
		t.Fatalf("edit status = %d, body %s", saved.Code, saved.Body.String())
	}
	after := decodeObject(t, saved)
	if after["id"] != before["id"] || after["key"] != before["key"] {
		t.Fatalf("edit changed identity: id %v→%v, key %v→%v", before["id"], after["id"], before["key"], after["key"])
	}
	if amount := line(editBody(after), section, index)["quantity"].(map[string]any)["amount"]; amount != "7" {
		t.Fatalf("saved amount = %v, want 7", amount)
	}
	if after["verifiedAt"] != "2026-08-19T12:00:00.000Z" || after["verifiedBy"] != "Pat Example" {
		t.Fatalf("approval = %v by %v, want the save's time and member", after["verifiedAt"], after["verifiedBy"])
	}
	var verifiedBy string
	var numerator int64
	if err := db.QueryRow(`SELECT verified_by::text FROM recipes WHERE id = $1`, recipeID).Scan(&verifiedBy); err != nil || verifiedBy != userID {
		t.Fatalf("stored verified_by = %q (%v), want %q", verifiedBy, err, userID)
	}
	if err := db.QueryRow(`
		SELECT ri.amount_min_numerator FROM recipe_ingredients ri
		JOIN recipe_ingredient_sections ris ON ris.id = ri.section_id
		WHERE ris.recipe_id = $1 AND ris.position = $2 AND ri.position = $3`, recipeID, section, index).Scan(&numerator); err != nil || numerator != 7 {
		t.Fatalf("stored amount = %d (%v), want 7", numerator, err)
	}

	rebuilt := groceryLineNamed(t, request(t, handler, http.MethodGet, groceriesPath(), ""), itemName)
	if quantity, _ := rebuilt["quantity"].(string); !strings.HasPrefix(quantity, "7 ") {
		t.Fatalf("current list quantity = %q, want the edited 7", quantity)
	}
	if rebuilt["completed"] != true {
		t.Fatalf("checked row lost its check: %#v", rebuilt)
	}

	// Deleting the approving member keeps the recipe verified and forgets who approved it.
	if _, err := db.Exec(`DELETE FROM app_users WHERE id = $1`, userID); err != nil {
		t.Fatalf("delete approving member: %v", err)
	}
	if detail := recipeDetail(t, handler, recipeID); detail["verifiedBy"] != nil {
		t.Fatalf("verifiedBy after deleting the member = %v", detail["verifiedBy"])
	}
}

func TestEditRecipeLeavesPastPlanHistoryUnchanged(t *testing.T) {
	db := testDB(t)
	handler := editHandler(t, db, seedUser(t, db, "Pat Example"))
	weekID := closedPlan(t, db)
	recorded := bytes.Clone(historyDetail(t, handler, weekID))
	contribution := firstPastContribution(t, db, weekID)

	before := recipeDetail(t, handler, contribution.recipeID)
	body := editBody(before)
	body["name"] = fmt.Sprintf("%v (edited)", before["name"])
	var removed bool
	for sectionIndex, value := range before["ingredientSections"].([]any) {
		ingredients := value.(map[string]any)["ingredients"].([]any)
		for lineIndex, ingredient := range ingredients {
			if int64(ingredient.(map[string]any)["id"].(float64)) == contribution.ingredientID {
				section := body["ingredientSections"].([]any)[sectionIndex].(map[string]any)
				section["ingredients"] = slices.Delete(section["ingredients"].([]any), lineIndex, lineIndex+1)
				removed = true
			}
		}
	}
	if !removed {
		t.Fatalf("past plan's ingredient %d not in recipe %d", contribution.ingredientID, contribution.recipeID)
	}
	if saved := putRecipe(t, handler, contribution.recipeID, body); saved.Code != http.StatusOK {
		t.Fatalf("edit status = %d, body %s", saved.Code, saved.Body.String())
	}
	if after := historyDetail(t, handler, weekID); !bytes.Equal(after, recorded) {
		t.Fatalf("history changed after editing a recipe:\nbefore %s\nafter  %s", recorded, after)
	}
}

// editableTables are every table an edit may write. Rows are compared whole, including audit
// timestamps, so a rejected edit must leave each one exactly as it was.
var editableTables = []string{
	"recipes", "recipe_sources", "recipe_ingredient_sections", "recipe_ingredients",
	"recipe_instruction_sections", "recipe_steps", "recipe_review_flags", "grocery_items",
	"weeks", "week_recipes", "shopping_lines", "shopping_line_quantities", "shopping_line_contributions",
}

func tableSnapshot(t *testing.T, db *sql.DB) string {
	t.Helper()
	var snapshot strings.Builder
	for _, table := range editableTables {
		var rows sql.NullString
		if err := db.QueryRow(`SELECT string_agg(to_jsonb(t)::text, E'\n' ORDER BY to_jsonb(t)::text) FROM ` + table + ` t`).Scan(&rows); err != nil {
			t.Fatalf("snapshot %s: %v", table, err)
		}
		snapshot.WriteString(table + "\n" + rows.String + "\n")
	}
	return snapshot.String()
}

func TestInvalidRecipeEditsChangeNothing(t *testing.T) {
	db := testDB(t)
	handler := editHandler(t, db, seedUser(t, db, "Pat Example"))
	recipeID := currentPlan(t, handler)
	detail := recipeDetail(t, handler, recipeID)
	section, index, _ := uniqueMeasuredLine(t, detail)
	linePath := fmt.Sprintf("ingredientSections[%d].ingredients[%d]", section, index)
	var otherRecipeName string
	if err := db.QueryRow(`SELECT name FROM recipes WHERE id <> $1 ORDER BY id LIMIT 1`, recipeID).Scan(&otherRecipeName); err != nil {
		t.Fatal(err)
	}
	// An item of the household's under a differently cased name is still that item.
	existingName := detail["ingredientSections"].([]any)[0].(map[string]any)["ingredients"].([]any)[0].(map[string]any)["shoppingItem"].(string)

	cases := []struct {
		name   string
		mutate func(body map[string]any)
		path   string
	}{
		{"empty name", func(body map[string]any) { body["name"] = " " }, "name"},
		{"another recipe's name", func(body map[string]any) { body["name"] = otherRecipeName }, "name"},
		{"no ingredient sections", func(body map[string]any) { body["ingredientSections"] = []any{} }, "ingredientSections"},
		{"unknown unit", func(body map[string]any) {
			line(body, section, index)["quantity"].(map[string]any)["unit"] = "pinch"
		}, linePath + ".quantity"},
		{"bad quantity", func(body map[string]any) {
			line(body, section, index)["quantity"].(map[string]any)["amount"] = "a lot"
		}, linePath + ".quantity"},
		{"duplicate new item name", func(body map[string]any) {
			edited := line(body, section, index)
			delete(edited, "groceryItemId")
			edited["newGroceryItem"] = map[string]any{"name": strings.ToUpper(existingName), "storeSectionId": 1, "shoppingMode": "measured"}
		}, linePath + ".newGroceryItem.name"},
		{"unknown grocery item", func(body map[string]any) { line(body, section, index)["groceryItemId"] = 999999 }, linePath + ".groceryItemId"},
		{"one-ended duration", func(body map[string]any) {
			body["handsOn"] = map[string]any{"minimumMinutes": 5, "maximumMinutes": nil}
		}, "handsOn"},
		{"blank step", func(body map[string]any) {
			body["instructionSections"].([]any)[0].(map[string]any)["steps"].([]any)[0].(map[string]any)["instruction"] = ""
		}, "instructionSections[0].steps[0]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := tableSnapshot(t, db)
			body := editBody(detail)
			tc.mutate(body)
			response := putRecipe(t, handler, recipeID, body)
			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, body %s", response.Code, response.Body.String())
			}
			assertErrorCode(t, response, "invalid_recipe")
			var paths []string
			for _, field := range decodeObject(t, response)["fields"].([]any) {
				paths = append(paths, field.(map[string]any)["path"].(string))
			}
			if !slices.Contains(paths, tc.path) {
				t.Fatalf("field paths = %v, want %q", paths, tc.path)
			}
			if after := tableSnapshot(t, db); after != before {
				t.Fatalf("a rejected edit changed the database")
			}
		})
	}
}

func TestEditAnotherHouseholdsRecipeIsNotFound(t *testing.T) {
	db := testDB(t)
	handler := editHandler(t, db, seedUser(t, db, "Pat Example"))
	var other string
	if err := db.QueryRow(`INSERT INTO households (name) VALUES ('Other') RETURNING id::text`).Scan(&other); err != nil {
		t.Fatal(err)
	}
	documents, err := ingest.ReadDirectory(filepath.Join("..", "..", "corpus", "recipes"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ingest.Import(context.Background(), db, other, documents[:1]); err != nil {
		t.Fatal(err)
	}
	var otherRecipe int64
	if err := db.QueryRow(`SELECT id FROM recipes WHERE household_id = $1`, other).Scan(&otherRecipe); err != nil {
		t.Fatal(err)
	}
	var ownRecipe int64
	if err := db.QueryRow(`SELECT id FROM recipes WHERE household_id = $1 AND key = $2`, snapshotHousehold, documents[0].Key).Scan(&ownRecipe); err != nil {
		t.Fatal(err)
	}
	body := editBody(recipeDetail(t, handler, ownRecipe))
	before := tableSnapshot(t, db)
	for _, recipeID := range []int64{otherRecipe, 999999} {
		response := putRecipe(t, handler, recipeID, body)
		if response.Code != http.StatusNotFound {
			t.Fatalf("recipe %d status = %d, body %s", recipeID, response.Code, response.Body.String())
		}
		assertErrorCode(t, response, "recipe_not_found")
	}
	if tableSnapshot(t, db) != before {
		t.Fatal("editing another household's recipe changed the database")
	}
}

func TestEditRecipeCreatesOneNewGroceryItem(t *testing.T) {
	db := testDB(t)
	handler := editHandler(t, db, seedUser(t, db, "Pat Example"))
	recipeID := currentPlan(t, handler)
	detail := recipeDetail(t, handler, recipeID)
	var sectionID int64
	if err := db.QueryRow(`SELECT id FROM store_sections WHERE household_id = $1 AND key = 'spices'`, snapshotHousehold).Scan(&sectionID); err != nil {
		t.Fatal(err)
	}

	body := editBody(detail)
	section := body["ingredientSections"].([]any)[0].(map[string]any)
	newLine := func(sourceText string) map[string]any {
		return map[string]any{
			"sourceText": sourceText, "preparation": nil, "optional": false, "displayNote": nil, "onGroceryList": true,
			"quantity":       map[string]any{"kind": "exact", "amount": "1", "maximum": nil, "unit": "tsp", "package": nil},
			"newGroceryItem": map[string]any{"name": "Grains of  Paradise", "storeSectionId": sectionID, "shoppingMode": "measured"},
		}
	}
	section["ingredients"] = append(section["ingredients"].([]any), newLine("1 tsp grains of paradise"), newLine("1 tsp more grains of paradise"))
	saved := putRecipe(t, handler, recipeID, body)
	if saved.Code != http.StatusOK {
		t.Fatalf("edit status = %d, body %s", saved.Code, saved.Body.String())
	}

	var count int
	var key string
	var itemSection int64
	if err := db.QueryRow(`SELECT count(*), min(key), min(store_section_id) FROM grocery_items WHERE household_id = $1 AND name = 'Grains of Paradise'`,
		snapshotHousehold).Scan(&count, &key, &itemSection); err != nil {
		t.Fatal(err)
	}
	if count != 1 || key != "grains-of-paradise" || itemSection != sectionID {
		t.Fatalf("new items = %d (key %q, section %d), want one grains-of-paradise in section %d", count, key, itemSection, sectionID)
	}
	listed := groceryLineNamed(t, request(t, handler, http.MethodGet, groceriesPath(), ""), "Grains of Paradise")
	if int64(listed["sectionId"].(float64)) != sectionID || listed["quantity"] != "2 tsp" {
		t.Fatalf("new item's grocery line = %#v, want 2 tsp in section %d", listed, sectionID)
	}
}

func TestEditRecipeRequiresSignedInMember(t *testing.T) {
	db := testDB(t)
	handler := handlerForDB(t, db)
	var recipeID int64
	if err := db.QueryRow(`SELECT id FROM recipes ORDER BY id LIMIT 1`).Scan(&recipeID); err != nil {
		t.Fatal(err)
	}
	body := editBody(recipeDetail(t, handler, recipeID))
	response := putRecipe(t, handler, recipeID, body)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body %s", response.Code, response.Body.String())
	}
}
