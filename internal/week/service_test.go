package week_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/ingest"
	"github.com/mbcoward3/grocery-router/internal/store"
	"github.com/mbcoward3/grocery-router/internal/testdatabase"
	"github.com/mbcoward3/grocery-router/internal/week"
)

type zeroPicker struct{}

func (zeroPicker) IntN(int) (int, error) { return 0, nil }

func TestCurrentWeekRequiresExplicitGeneration(t *testing.T) {
	service, _ := testService(t)
	_, err := service.Current(context.Background())
	if !errors.Is(err, week.ErrNoCurrentWeek) {
		t.Fatalf("Current error = %v, want ErrNoCurrentWeek", err)
	}
}

func TestGenerateIsUniqueAndFailureIsTransactional(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()

	view, err := service.Generate(ctx, testNow(), 4)
	if err != nil {
		t.Fatal(err)
	}
	if view.Week.StartsOn != "2026-08-19" || len(view.Recipes) != 4 {
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
	checklist, err := service.Checklist(ctx)
	if err != nil || len(checklist.Lines) == 0 {
		t.Fatalf("Checklist = %#v, %v", checklist, err)
	}
	contributions, err := service.Contributions(ctx, checklist.Lines[0].ID)
	if err != nil || len(contributions) == 0 {
		t.Fatalf("Contributions = %#v, %v", contributions, err)
	}

	if _, err := service.Generate(ctx, testNow(), 100); err == nil {
		t.Fatal("oversized generation unexpectedly succeeded")
	}
	current, err := service.Current(ctx)
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
	view, err = service.Add(ctx, view.Recipes[0].RecipeID)
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
	if err := service.SetLineCompleted(ctx, lineID, true); err != nil {
		t.Fatal(err)
	}
	if err := service.SetLineOverride(ctx, lineID, "buy two large containers"); err != nil {
		t.Fatal(err)
	}
	manual, err := service.AddManualLine(ctx, "Birthday candles")
	if err != nil {
		t.Fatal(err)
	}

	// Recompute preserves generated state by aggregation key and leaves manual lines alone.
	view, err = service.Add(ctx, view.Recipes[0].RecipeID)
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
	view, err = service.Remove(ctx, removedID)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Recipes) != 4 {
		t.Fatalf("remove left %d recipes, want 4", len(view.Recipes))
	}
	assertAllShoppingContributionsPresent(t, db, view.Week.ID)

	oldRecipe := view.Recipes[0].RecipeID
	view, err = service.RandomSwap(ctx, view.Recipes[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.Recipes[0].RecipeID == oldRecipe {
		t.Fatal("random swap retained the same recipe")
	}

	target := view.Recipes[1].RecipeID
	view, err = service.Swap(ctx, view.Recipes[0].ID, target)
	if err != nil {
		t.Fatal(err)
	}
	if view.Recipes[0].RecipeID != target {
		t.Fatal("specific swap did not permit the selected duplicate")
	}
}

func testService(t *testing.T) (*week.Household, *sql.DB) {
	t.Helper()
	db := testdatabase.Open(t)
	documents, err := ingest.ReadDirectory(filepath.Join("..", "..", "corpus", "recipes"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ingest.Import(context.Background(), db, "c0a7a2d8-669b-4e47-91c1-4d9a32f339d5", documents); err != nil {
		t.Fatal(err)
	}
	return week.NewService(db, zeroPicker{}).ForHousehold("c0a7a2d8-669b-4e47-91c1-4d9a32f339d5"), db
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
	exact, err := service.AddManualLine(ctx, "  MILK  ")
	if err != nil || exact.StoreSectionID != dairy || exact.GroceryItemID.Int64 != milk || exact.Origin != "manual" {
		t.Fatalf("exact match = %#v, %v", exact, err)
	}
	picked, err := service.AddManualLine(ctx, "mil", week.ManualLineOptions{GroceryItemID: milk})
	if err != nil || picked.StoreSectionID != dairy {
		t.Fatalf("picked match = %#v, %v", picked, err)
	}
	unknown, err := service.AddManualLine(ctx, "Birthday candles")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.SetLineSection(ctx, unknown.ID, baking); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AddManualLine(ctx, "Bad item", week.ManualLineOptions{GroceryItemID: -1}); err == nil {
		t.Fatal("invalid item accepted")
	}
	if err := service.SetLineSection(ctx, unknown.ID, -1); err == nil {
		t.Fatal("invalid section accepted")
	}
	if _, err := service.Generate(ctx, testNow(), 3); err != nil {
		t.Fatal(err)
	}
	checklist, err := service.Checklist(ctx)
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

func TestUnknownManualItemRemembersChosenSection(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	const household = "c0a7a2d8-669b-4e47-91c1-4d9a32f339d5"
	if _, err := service.Generate(ctx, testNow(), 2); err != nil {
		t.Fatal(err)
	}
	var dairy, other int64
	if err := db.QueryRow(`SELECT id FROM store_sections WHERE household_id = $1 AND key = 'dairy'`, household).Scan(&dairy); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT id FROM store_sections WHERE household_id = $1 AND key = 'other'`, household).Scan(&other); err != nil {
		t.Fatal(err)
	}

	// First time: unknown, lands in Other, then the user places it in Dairy.
	first, err := service.AddManualLine(ctx, "Cottage cheese")
	if err != nil || first.StoreSectionID != other {
		t.Fatalf("first add = %#v, %v", first, err)
	}
	if err := service.SetLineSection(ctx, first.ID, dairy); err != nil {
		t.Fatal(err)
	}

	// Searching again offers it in Dairy, ahead of looser matches.
	matches, err := service.SearchItems(ctx, "cottage")
	if err != nil || len(matches) == 0 || matches[0].Name != "Cottage cheese" || matches[0].SectionID != dairy || matches[0].GroceryItemID != 0 {
		t.Fatalf("search = %#v, %v", matches, err)
	}

	// Typing it again goes straight to Dairy.
	second, err := service.AddManualLine(ctx, "  cottage CHEESE ")
	if err != nil || second.StoreSectionID != dairy {
		t.Fatalf("second add = %#v, %v", second, err)
	}

	// Choosing a section while adding is remembered too; moving back to Other forgets it.
	candles, err := service.AddManualLine(ctx, "Birthday candles", week.ManualLineOptions{StoreSectionID: dairy})
	if err != nil || candles.StoreSectionID != dairy {
		t.Fatalf("explicit add = %#v, %v", candles, err)
	}
	if err := service.SetLineSection(ctx, candles.ID, other); err != nil {
		t.Fatal(err)
	}
	if matches, err := service.SearchItems(ctx, "birthday"); err != nil || len(matches) != 0 {
		t.Fatalf("forgotten search = %#v, %v", matches, err)
	}

	// A known item is not remembered by name: moving it saves the household's placement of
	// the item itself (D053), which suggestions then follow.
	milk, err := service.AddManualLine(ctx, "milk")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.SetLineSection(ctx, milk.ID, other); err != nil {
		t.Fatal(err)
	}
	var remembered int
	if err := db.QueryRow(`SELECT count(*) FROM remembered_item_sections WHERE household_id = $1`, household).Scan(&remembered); err != nil || remembered != 1 {
		t.Fatalf("remembered rows = %d, %v", remembered, err)
	}
	if matches, err := service.SearchItems(ctx, "milk"); err != nil || len(matches) == 0 || matches[0].Name != "Milk" || matches[0].SectionID != other {
		t.Fatalf("known search = %#v, %v", matches, err)
	}
}

// Moving a generated line saves where the household's store keeps that item: the item's lines
// move now, later plans and manual adds use the placement, and choosing the reviewed section
// again clears it (D053).
func TestKnownItemSectionApplies(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	const household = "c0a7a2d8-669b-4e47-91c1-4d9a32f339d5"
	if _, err := service.Generate(ctx, testNow(), 3); err != nil {
		t.Fatal(err)
	}
	checklist, err := service.Checklist(ctx)
	if err != nil || len(checklist.Lines) == 0 {
		t.Fatalf("checklist = %#v, %v", checklist, err)
	}
	line := checklist.Lines[0]
	reviewed := line.StoreSectionID
	var other int64
	if err := db.QueryRow(`SELECT id FROM store_sections WHERE household_id = $1 AND key = 'other'`, household).Scan(&other); err != nil {
		t.Fatal(err)
	}
	if reviewed == other {
		t.Fatal("test needs a line outside Other")
	}
	// The item's generated row, found by item because a rebuild replaces rows.
	generatedRow := func() store.ListShoppingLinesRow {
		t.Helper()
		current, err := service.Checklist(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, candidate := range current.Lines {
			if candidate.Origin == "generated" && candidate.GroceryItemID == line.GroceryItemID {
				return candidate
			}
		}
		t.Fatalf("generated row for item %v missing", line.GroceryItemID)
		return store.ListShoppingLinesRow{}
	}
	sectionOf := func(lineID int64) int64 {
		t.Helper()
		current, err := service.Checklist(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, candidate := range current.Lines {
			if candidate.ID == lineID {
				return candidate.StoreSectionID
			}
		}
		t.Fatalf("line %d missing", lineID)
		return 0
	}
	if err := service.SetLineCompleted(ctx, line.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := service.SetLineSection(ctx, line.ID, other); err != nil {
		t.Fatal(err)
	}
	if got := sectionOf(line.ID); got != other {
		t.Fatalf("moved line section = %d, want %d", got, other)
	}
	// A rebuild of the list keeps the placement and the checked state.
	if _, err := service.Add(ctx, firstRecipe(t, db)); err != nil {
		t.Fatal(err)
	}
	if row := generatedRow(); row.StoreSectionID != other || row.IsCompleted != 1 {
		t.Fatalf("row after rebuild = section %d, completed %d; want %d and checked", row.StoreSectionID, row.IsCompleted, other)
	}
	manual, err := service.AddManualLine(ctx, line.DisplayName)
	if err != nil || manual.StoreSectionID != other {
		t.Fatalf("manual add of a placed item = %#v, %v", manual, err)
	}

	// Choosing the reviewed section again clears the placement.
	if err := service.SetLineSection(ctx, generatedRow().ID, reviewed); err != nil {
		t.Fatal(err)
	}
	var overrides int
	if err := db.QueryRow(`SELECT count(*) FROM household_item_sections WHERE household_id = $1`, household).Scan(&overrides); err != nil || overrides != 0 {
		t.Fatalf("overrides after reset = %d, %v", overrides, err)
	}
	if got := generatedRow().StoreSectionID; got != reviewed {
		t.Fatalf("reset line section = %d, want %d", got, reviewed)
	}
	if got := sectionOf(manual.ID); got != reviewed {
		t.Fatalf("reset manual line section = %d, want %d", got, reviewed)
	}
}

func firstRecipe(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(`SELECT id FROM recipes WHERE status = 'verified' ORDER BY id DESC LIMIT 1`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestChecklistHasOneRowPerItemAndUpgradesOldLists(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	view, err := service.Generate(ctx, testNow(), 1)
	if err != nil {
		t.Fatal(err)
	}
	// Two measured teaspoons and one "generous" pepper used to show as two Black Pepper rows.
	for _, key := range []string{"tuna-melt", "beef-pot-roast", "beef-stew-with-carrots-and-potatoes"} {
		var recipeID int64
		if err := db.QueryRow(`SELECT id FROM recipes WHERE key = $1`, key).Scan(&recipeID); err != nil {
			t.Fatal(err)
		}
		if _, err := service.Add(ctx, recipeID); err != nil {
			t.Fatal(err)
		}
	}
	var rows, items int
	if err := db.QueryRow(`SELECT count(*), count(DISTINCT grocery_item_id) FROM shopping_lines WHERE origin = 'generated'`).Scan(&rows, &items); err != nil {
		t.Fatal(err)
	}
	if rows != items {
		t.Fatalf("generated rows = %d for %d items, want one row per item", rows, items)
	}
	pepper := func() (int64, string, int64, bool) {
		checklist, err := service.Checklist(ctx)
		if err != nil {
			t.Fatal(err)
		}
		found := 0
		var id, completed int64
		var kind string
		for _, line := range checklist.Lines {
			if line.DisplayName == "Black Pepper" {
				found++
				id, kind, completed = line.ID, line.QuantityKind, line.IsCompleted
			}
		}
		if found != 1 {
			t.Fatalf("Black Pepper rows = %d, want 1", found)
		}
		return id, kind, completed, checklist.Unspecified[id]
	}
	id, kind, _, unspecified := pepper()
	if kind != "exact" || !unspecified {
		t.Fatalf("Black Pepper kind = %q, unspecified = %v; want an exact amount plus more", kind, unspecified)
	}
	assertAllShoppingContributionsPresent(t, db, view.Week.ID)

	// A list built before one row per item had keys like item:<id>:dimension:volume. Reading the
	// current week rebuilds it once and keeps what was already checked.
	if _, err := db.Exec(`UPDATE shopping_lines SET aggregation_key = aggregation_key || ':dimension:volume', is_completed = 1 WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE shopping_lists SET aggregation_version = 1`); err != nil {
		t.Fatal(err)
	}
	if _, _, completed, _ := pepper(); completed != 1 {
		t.Fatal("upgrading an old list lost the checked state")
	}
	var version int64
	var key string
	if err := db.QueryRow(`SELECT aggregation_version FROM shopping_lists`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT aggregation_key FROM shopping_lines WHERE display_name = 'Black Pepper'`).Scan(&key); err != nil {
		t.Fatal(err)
	}
	if version != 2 || strings.Count(key, ":") != 1 {
		t.Fatalf("after upgrade version = %d, key = %q", version, key)
	}
	assertAllShoppingContributionsPresent(t, db, view.Week.ID)
}
