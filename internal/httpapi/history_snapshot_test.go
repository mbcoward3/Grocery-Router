package httpapi_test

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/week"
)

const snapshotHousehold = "c0a7a2d8-669b-4e47-91c1-4d9a32f339d5"

// closedPlan generates a two-recipe plan and closes it, returning its week ID.
func closedPlan(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	service := week.NewService(db, zeroPicker{}).ForHousehold(snapshotHousehold)
	planned := time.Date(2026, time.August, 12, 12, 0, 0, 0, time.Local)
	view, err := service.Generate(context.Background(), planned, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Close(context.Background(), planned.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	return view.Week.ID
}

func historyDetail(t *testing.T, handler http.Handler, weekID int64) []byte {
	t.Helper()
	response := request(t, handler, http.MethodGet, fmt.Sprintf("/api/v2/households/%s/weeks/history/%d", snapshotHousehold, weekID), "")
	if response.Code != http.StatusOK {
		t.Fatalf("history detail status = %d, body %s", response.Code, response.Body.String())
	}
	return response.Body.Bytes()
}

// pastContribution is one retained contribution of a closed plan's first recipe.
type pastContribution struct {
	id, recipeID, ingredientID int64
	recipeName, sourceText     string
}

func firstPastContribution(t *testing.T, db *sql.DB, weekID int64) pastContribution {
	t.Helper()
	var contribution pastContribution
	if err := db.QueryRow(`
		SELECT c.id, c.recipe_id, c.recipe_ingredient_id, c.recipe_name, c.source_text
		FROM shopping_line_contributions c
		JOIN week_recipes wr ON wr.id = c.week_recipe_id
		WHERE wr.week_id = $1
		ORDER BY wr.position, c.id LIMIT 1`, weekID).Scan(
		&contribution.id, &contribution.recipeID, &contribution.ingredientID, &contribution.recipeName, &contribution.sourceText,
	); err != nil {
		t.Fatal(err)
	}
	return contribution
}

func returnToDraft(t *testing.T, db *sql.DB, recipeID int64) {
	t.Helper()
	if _, err := db.Exec(`UPDATE recipes SET status = 'draft', verified_at = NULL WHERE id = $1`, recipeID); err != nil {
		t.Fatal(err)
	}
}

// Deleting an ingredient that a past plan used detaches its contribution and leaves History as
// it was.
func TestHistoryKeepsDeletedIngredientContribution(t *testing.T) {
	db := testDB(t)
	weekID := closedPlan(t, db)
	contribution := firstPastContribution(t, db, weekID)

	returnToDraft(t, db, contribution.recipeID)
	if _, err := db.Exec(`DELETE FROM recipe_ingredients WHERE id = $1`, contribution.ingredientID); err != nil {
		t.Fatalf("delete ingredient used by a past plan: %v", err)
	}
	var ingredient sql.NullInt64
	var recipeName, sourceText string
	if err := db.QueryRow(`SELECT recipe_ingredient_id, recipe_name, source_text FROM shopping_line_contributions WHERE id = $1`,
		contribution.id).Scan(&ingredient, &recipeName, &sourceText); err != nil {
		t.Fatalf("retained contribution: %v", err)
	}
	if ingredient.Valid || recipeName != contribution.recipeName || sourceText != contribution.sourceText {
		t.Fatalf("retained contribution = %v, %q, %q; want detached with %q, %q",
			ingredient, recipeName, sourceText, contribution.recipeName, contribution.sourceText)
	}

	body := decodeObject(t, request(t, handlerForDB(t, db), http.MethodGet,
		fmt.Sprintf("/api/v2/households/%s/weeks/history/%d", snapshotHousehold, weekID), ""))
	var foundRecipe bool
	for _, value := range body["week"].(map[string]any)["recipes"].([]any) {
		recipe := value.(map[string]any)["recipe"].(map[string]any)
		foundRecipe = foundRecipe || recipe["name"] == contribution.recipeName
	}
	var foundLine bool
	for _, value := range body["groceries"].(map[string]any)["lines"].([]any) {
		names, _ := value.(map[string]any)["recipeNames"].([]any)
		for _, name := range names {
			foundLine = foundLine || name == contribution.recipeName
		}
	}
	if !foundRecipe || !foundLine {
		t.Fatalf("history no longer shows %q (recipes %t, grocery lines %t): %#v", contribution.recipeName, foundRecipe, foundLine, body)
	}
}

// Editing a recipe that a past plan used never changes that plan's History response.
func TestHistoryResponseUnchangedByRecipeEdits(t *testing.T) {
	db := testDB(t)
	weekID := closedPlan(t, db)
	handler := handlerForDB(t, db)
	recorded := bytes.Clone(historyDetail(t, handler, weekID))
	contribution := firstPastContribution(t, db, weekID)

	returnToDraft(t, db, contribution.recipeID)
	if _, err := db.Exec(`UPDATE recipes SET
			name = name || ' (edited)', yield_text = 'Serves 99', image_url = 'https://example.com/edited.jpg',
			hands_on_min_minutes = 1, hands_on_max_minutes = 2,
			unattended_min_minutes = 3, unattended_max_minutes = 4
		WHERE id = $1`, contribution.recipeID); err != nil {
		t.Fatalf("edit recipe: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM recipe_ingredients WHERE id = $1`, contribution.ingredientID); err != nil {
		t.Fatalf("delete ingredient: %v", err)
	}

	if after := historyDetail(t, handler, weekID); !bytes.Equal(after, recorded) {
		t.Fatalf("history changed after editing a recipe:\nbefore %s\nafter  %s", recorded, after)
	}
}
