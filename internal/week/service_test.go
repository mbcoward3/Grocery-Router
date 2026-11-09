package week_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/ingest"
	"github.com/mbcoward3/grocery-router/internal/testdatabase"
	"github.com/mbcoward3/grocery-router/internal/week"
)

type zeroPicker struct{}

func (zeroPicker) IntN(int) (int, error) { return 0, nil }

func TestCurrentWeekRequiresExplicitGeneration(t *testing.T) {
	service, _ := testService(t)
	_, err := service.Current(context.Background(), testNow())
	if !errors.Is(err, week.ErrNoCurrentWeek) {
		t.Fatalf("Current error = %v, want ErrNoCurrentWeek", err)
	}
	if got := week.CurrentSunday(testNow()); got != "2026-08-16" {
		t.Fatalf("CurrentSunday = %q", got)
	}
}

func TestGenerateIsUniqueAndFailureIsTransactional(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()

	view, err := service.Generate(ctx, testNow(), 4)
	if err != nil {
		t.Fatal(err)
	}
	if view.Week.StartsOn != "2026-08-16" || len(view.Recipes) != 4 {
		t.Fatalf("generated view = %#v", view)
	}
	seen := map[int64]bool{}
	for _, occurrence := range view.Recipes {
		if seen[occurrence.RecipeID] {
			t.Fatalf("initial generation duplicated recipe %d", occurrence.RecipeID)
		}
		seen[occurrence.RecipeID] = true
	}
	assertAllShoppingContributionsPresent(t, db, view.Week.ID)
	checklist, err := service.Checklist(ctx, testNow())
	if err != nil || len(checklist.Lines) == 0 {
		t.Fatalf("Checklist = %#v, %v", checklist, err)
	}
	contributions, err := service.Contributions(ctx, testNow(), checklist.Lines[0].ID)
	if err != nil || len(contributions) == 0 {
		t.Fatalf("Contributions = %#v, %v", contributions, err)
	}

	if _, err := service.Generate(ctx, testNow(), 100); err == nil {
		t.Fatal("oversized generation unexpectedly succeeded")
	}
	current, err := service.Current(ctx, testNow())
	if err != nil {
		t.Fatal(err)
	}
	if len(current.Recipes) != 4 {
		t.Fatalf("failed generation changed recipe count to %d", len(current.Recipes))
	}
}

func TestWeekMutationsAndGroceryState(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	view, err := service.Generate(ctx, testNow(), 3)
	if err != nil {
		t.Fatal(err)
	}

	// Manual add permits a duplicate occurrence.
	view, err = service.Add(ctx, testNow(), view.Recipes[0].RecipeID)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Recipes) != 4 || view.Recipes[0].RecipeID != view.Recipes[3].RecipeID {
		t.Fatal("manual duplicate was not retained as a separate occurrence")
	}
	assertAllShoppingContributionsPresent(t, db, view.Week.ID)

	var lineID int64
	var key, sourceBefore string
	if err := db.QueryRow(`
		SELECT sl.id, sl.aggregation_key, ri.source_text
		FROM shopping_lines sl
		JOIN shopping_line_contributions c ON c.shopping_line_id = sl.id
		JOIN recipe_ingredients ri ON ri.id = c.recipe_ingredient_id
		WHERE sl.origin = 'generated'
		ORDER BY sl.id LIMIT 1
	`).Scan(&lineID, &key, &sourceBefore); err != nil {
		t.Fatal(err)
	}
	if err := service.SetLineCompleted(ctx, testNow(), lineID, true); err != nil {
		t.Fatal(err)
	}
	if err := service.SetLineOverride(ctx, testNow(), lineID, "buy two large containers"); err != nil {
		t.Fatal(err)
	}
	manual, err := service.AddManualLine(ctx, testNow(), "Birthday candles")
	if err != nil {
		t.Fatal(err)
	}

	// Recompute preserves generated state by aggregation key and leaves manual lines alone.
	view, err = service.Add(ctx, testNow(), view.Recipes[0].RecipeID)
	if err != nil {
		t.Fatal(err)
	}
	var completed int
	var override, sourceAfter string
	if err := db.QueryRow(`SELECT is_completed, override_text FROM shopping_lines
		WHERE shopping_list_id = (SELECT id FROM shopping_lists WHERE week_id = $1)
		AND aggregation_key = $2`, view.Week.ID, key).Scan(&completed, &override); err != nil {
		t.Fatal(err)
	}
	if completed != 1 || override != "buy two large containers" {
		t.Fatalf("recomputed state = completed %d override %q", completed, override)
	}
	if err := db.QueryRow("SELECT source_text FROM recipe_ingredients WHERE source_text = $1 LIMIT 1", sourceBefore).Scan(&sourceAfter); err != nil {
		t.Fatal(err)
	}
	if sourceAfter != sourceBefore {
		t.Fatal("week override changed recipe truth")
	}
	var manualCount int
	if err := db.QueryRow("SELECT count(*) FROM shopping_lines WHERE id = $1 AND origin = 'manual'", manual.ID).Scan(&manualCount); err != nil {
		t.Fatal(err)
	}
	if manualCount != 1 {
		t.Fatal("grocery recomputation removed a manual line")
	}

	removedID := view.Recipes[1].ID
	view, err = service.Remove(ctx, testNow(), removedID)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Recipes) != 4 {
		t.Fatalf("remove left %d recipes, want 4", len(view.Recipes))
	}
	assertAllShoppingContributionsPresent(t, db, view.Week.ID)

	oldRecipe := view.Recipes[0].RecipeID
	view, err = service.RandomSwap(ctx, testNow(), view.Recipes[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.Recipes[0].RecipeID == oldRecipe {
		t.Fatal("random swap retained the same recipe")
	}

	target := view.Recipes[1].RecipeID
	view, err = service.Swap(ctx, testNow(), view.Recipes[0].ID, target)
	if err != nil {
		t.Fatal(err)
	}
	if view.Recipes[0].RecipeID != target {
		t.Fatal("specific swap did not permit the selected duplicate")
	}
}

func testService(t *testing.T) (*week.Service, *sql.DB) {
	t.Helper()
	db := testdatabase.Open(t)
	documents, err := ingest.ReadDirectory(filepath.Join("..", "..", "corpus", "recipes"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ingest.Import(context.Background(), db, "c0a7a2d8-669b-4e47-91c1-4d9a32f339d5", documents); err != nil {
		t.Fatal(err)
	}
	return week.NewService(db, zeroPicker{}, "c0a7a2d8-669b-4e47-91c1-4d9a32f339d5"), db
}

func testNow() time.Time {
	return time.Date(2026, time.August, 19, 12, 0, 0, 0, time.Local)
}

func assertAllShoppingContributionsPresent(t *testing.T, db *sql.DB, weekID int64) {
	t.Helper()
	var wanted, got int
	if err := db.QueryRow(`
		SELECT count(*)
		FROM week_recipes wr
		JOIN recipe_ingredient_sections ris ON ris.recipe_id = wr.recipe_id
		JOIN recipe_ingredients ri ON ri.section_id = ris.id
		WHERE wr.week_id = $1 AND ri.include_on_grocery_list = 1
	`, weekID).Scan(&wanted); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`
		SELECT count(*)
		FROM shopping_line_contributions c
		JOIN shopping_lines sl ON sl.id = c.shopping_line_id
		JOIN shopping_lists list ON list.id = sl.shopping_list_id
		WHERE list.week_id = $1
	`, weekID).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != wanted {
		t.Fatalf("shopping contributions = %d, want %d eligible ingredients", got, wanted)
	}
}

func TestManualPlacementAndSharedSectionOrder(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	const household = "c0a7a2d8-669b-4e47-91c1-4d9a32f339d5"
	if _, err := service.Generate(ctx, testNow(), 2); err != nil {
		t.Fatal(err)
	}
	var milk, dairy, baking int64
	if err := db.QueryRow(`SELECT id, store_section_id FROM grocery_items WHERE household_id = $1 AND key = 'milk'`, household).Scan(&milk, &dairy); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT id FROM store_sections WHERE household_id = $1 AND key = 'baking'`, household).Scan(&baking); err != nil {
		t.Fatal(err)
	}
	exact, err := service.AddManualLine(ctx, testNow(), "  MILK  ")
	if err != nil || exact.StoreSectionID != dairy || exact.GroceryItemID.Int64 != milk || exact.Origin != "manual" {
		t.Fatalf("exact match = %#v, %v", exact, err)
	}
	picked, err := service.AddManualLine(ctx, testNow(), "mil", week.ManualLineOptions{GroceryItemID: milk})
	if err != nil || picked.StoreSectionID != dairy {
		t.Fatalf("picked match = %#v, %v", picked, err)
	}
	unknown, err := service.AddManualLine(ctx, testNow(), "Birthday candles")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.SetLineSection(ctx, testNow(), unknown.ID, baking); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AddManualLine(ctx, testNow(), "Bad item", week.ManualLineOptions{GroceryItemID: -1}); err == nil {
		t.Fatal("invalid item accepted")
	}
	if err := service.SetLineSection(ctx, testNow(), unknown.ID, -1); err == nil {
		t.Fatal("invalid section accepted")
	}
	if _, err := service.Generate(ctx, testNow(), 3); err != nil {
		t.Fatal(err)
	}
	checklist, err := service.Checklist(ctx, testNow())
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, line := range checklist.Lines {
		if line.ID == unknown.ID {
			found++
			if line.StoreSectionID != baking {
				t.Fatal("manual section lost")
			}
		}
		if line.ID == exact.ID {
			found++
			if line.GroceryItemID.Int64 != milk {
				t.Fatal("manual identity lost")
			}
		}
	}
	if found != 2 {
		t.Fatal("manual lines lost on rebuild")
	}
	var traces int
	if err := db.QueryRow(`SELECT count(*) FROM shopping_line_contributions WHERE shopping_line_id IN ($1, $2)`, exact.ID, picked.ID).Scan(&traces); err != nil || traces != 0 {
		t.Fatalf("manual traces = %d, %v", traces, err)
	}
	rows, err := db.Query(`SELECT id FROM store_sections WHERE household_id = $1 ORDER BY display_position DESC, name DESC`, household)
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	if err := service.SetSectionOrder(ctx, ids, false); err != nil {
		t.Fatal(err)
	}
	if err := service.SetSectionOrder(ctx, ids[:len(ids)-1], false); err == nil {
		t.Fatal("incomplete order accepted")
	}
	duplicate := append([]int64{}, ids...)
	duplicate[0] = duplicate[1]
	if err := service.SetSectionOrder(ctx, duplicate, false); err == nil {
		t.Fatal("duplicate order accepted")
	}
	var first int64
	if err := db.QueryRow(`SELECT id FROM store_sections WHERE household_id = $1 ORDER BY display_position, name LIMIT 1`, household).Scan(&first); err != nil || first != ids[0] {
		t.Fatalf("saved order = %d, %v", first, err)
	}
	if _, err := db.Exec(`INSERT INTO store_sections (household_id, key, name) VALUES ($1, 'seasonal', 'Seasonal')`, household); err != nil {
		t.Fatal(err)
	}
	var seasonalPosition, otherPosition int64
	if err := db.QueryRow(`SELECT display_position FROM store_sections WHERE household_id = $1 AND key = 'seasonal'`, household).Scan(&seasonalPosition); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT display_position FROM store_sections WHERE household_id = $1 AND key = 'other'`, household).Scan(&otherPosition); err != nil || seasonalPosition != otherPosition-1 {
		t.Fatalf("new section placement = %d, Other %d, %v", seasonalPosition, otherPosition, err)
	}
	if err := service.SetSectionOrder(ctx, nil, true); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT id FROM store_sections WHERE household_id = $1 ORDER BY display_position, name LIMIT 1`, household).Scan(&first); err != nil {
		t.Fatal(err)
	}
	var produce int64
	if err := db.QueryRow(`SELECT id FROM store_sections WHERE household_id = $1 AND key = 'produce'`, household).Scan(&produce); err != nil || first != produce {
		t.Fatal("default did not restore Produce first")
	}
}

func TestNewSectionPlacementBeforeAnyCustomization(t *testing.T) {
	_, db := testService(t)
	const household = "c0a7a2d8-669b-4e47-91c1-4d9a32f339d5"
	if _, err := db.Exec(`INSERT INTO store_sections (household_id, key, name) VALUES ($1, 'seasonal', 'Seasonal')`, household); err != nil {
		t.Fatal(err)
	}
	var newPosition, otherPosition int64
	if err := db.QueryRow(`SELECT display_position FROM store_sections WHERE household_id = $1 AND key = 'seasonal'`, household).Scan(&newPosition); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT display_position FROM store_sections WHERE household_id = $1 AND key = 'other'`, household).Scan(&otherPosition); err != nil || newPosition != otherPosition-1 {
		t.Fatalf("new section %d, Other %d, error %v", newPosition, otherPosition, err)
	}
}
