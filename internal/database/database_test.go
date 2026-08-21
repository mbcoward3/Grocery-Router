package database_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/database"
)

func migratedDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:test-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := database.Open(dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(context.Background(), db); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	return db
}

func TestMigrateEmptyDatabase(t *testing.T) {
	db := migratedDB(t)

	var foreignKeys int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}

	for _, table := range []string{
		"recipes", "recipe_sources", "recipe_ingredient_sections", "recipe_ingredients",
		"recipe_instruction_sections", "recipe_steps", "recipe_review_flags",
		"store_sections", "grocery_items", "units", "weeks", "week_recipes",
		"shopping_lists", "shopping_lines", "shopping_line_contributions",
	} {
		var count int
		err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&count)
		if err != nil {
			t.Fatalf("look up %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("table %s count = %d, want 1", table, count)
		}
	}
}

func TestRecipeMustPassReviewBeforeVerification(t *testing.T) {
	db := migratedDB(t)

	result, err := db.Exec("INSERT INTO recipes (key, name) VALUES ('test-soup', 'Test Soup')")
	if err != nil {
		t.Fatal(err)
	}
	recipeID, _ := result.LastInsertId()

	// No source, ingredient, instruction, or reviewable transition may be skipped.
	_, err = db.Exec("UPDATE recipes SET status = 'verified', verified_at = ? WHERE id = ?", "2026-08-17T00:00:00Z", recipeID)
	assertErrorContains(t, err, "recipe must be reviewable")

	mustExec(t, db, `INSERT INTO recipe_sources
		(recipe_id, relationship, attribution, is_primary, position)
		VALUES (?, 'source', 'Recipes.pdf', 1, 0)`, recipeID)
	sectionID := mustInsertID(t, db, `INSERT INTO recipe_ingredient_sections
		(recipe_id, name, position) VALUES (?, 'Ingredients', 0)`, recipeID)
	ingredientID := mustInsertID(t, db, `INSERT INTO recipe_ingredients
		(section_id, position, source_text, quantity_kind)
		VALUES (?, 0, 'salt to taste', 'unspecified')`, sectionID)
	instructionSectionID := mustInsertID(t, db, `INSERT INTO recipe_instruction_sections
		(recipe_id, name, position) VALUES (?, 'Method', 0)`, recipeID)
	mustExec(t, db, `INSERT INTO recipe_steps (section_id, position, instruction)
		VALUES (?, 0, 'Season the soup.')`, instructionSectionID)
	flagID := mustInsertID(t, db, `INSERT INTO recipe_review_flags
		(recipe_id, field_path, kind, note) VALUES (?, 'steps[0]', 'backfilled', 'Drafted from household description')`, recipeID)
	mustExec(t, db, "UPDATE recipes SET status = 'reviewable' WHERE id = ?", recipeID)

	_, err = db.Exec("UPDATE recipes SET status = 'verified', verified_at = ? WHERE id = ?", "2026-08-17T00:00:00Z", recipeID)
	assertErrorContains(t, err, "unmapped ingredient")

	storeSectionID := mustInsertID(t, db, "INSERT INTO store_sections (key, name) VALUES ('spices', 'Spices')")
	groceryItemID := mustInsertID(t, db, `INSERT INTO grocery_items
		(key, name, store_section_id, shopping_mode) VALUES ('salt', 'Salt', ?, 'presence-only')`, storeSectionID)
	mustExec(t, db, "UPDATE recipe_ingredients SET grocery_item_id = ? WHERE id = ?", groceryItemID, ingredientID)

	_, err = db.Exec("UPDATE recipes SET status = 'verified', verified_at = ? WHERE id = ?", "2026-08-17T00:00:00Z", recipeID)
	assertErrorContains(t, err, "unapproved review flags")

	mustExec(t, db, "UPDATE recipe_review_flags SET approved = 1 WHERE id = ?", flagID)
	mustExec(t, db, "UPDATE recipes SET status = 'verified', verified_at = ? WHERE id = ?", "2026-08-17T00:00:00Z", recipeID)

	var status string
	if err := db.QueryRow("SELECT status FROM recipes WHERE id = ?", recipeID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "verified" {
		t.Fatalf("status = %q, want verified", status)
	}

	_, err = db.Exec("UPDATE recipe_ingredients SET source_text = 'more salt' WHERE id = ?", ingredientID)
	assertErrorContains(t, err, "return recipe to draft")
	_, err = db.Exec("UPDATE recipes SET name = 'Changed' WHERE id = ?", recipeID)
	assertErrorContains(t, err, "return recipe to draft")

	mustExec(t, db, "UPDATE recipes SET status = 'draft', verified_at = NULL WHERE id = ?", recipeID)
	mustExec(t, db, "UPDATE recipe_ingredients SET source_text = 'more salt' WHERE id = ?", ingredientID)
}

func TestWeekDateAndVerifiedRecipeConstraints(t *testing.T) {
	db := migratedDB(t)

	_, err := db.Exec("INSERT INTO weeks (starts_on) VALUES ('2026-08-17')")
	assertErrorContains(t, err, "CHECK constraint failed")
	_, err = db.Exec("INSERT INTO weeks (starts_on) VALUES ('2026-99-99')")
	assertErrorContains(t, err, "CHECK constraint failed")
	weekID := mustInsertID(t, db, "INSERT INTO weeks (starts_on) VALUES ('2026-08-16')")
	recipeID := mustInsertID(t, db, "INSERT INTO recipes (key, name) VALUES ('draft', 'Draft')")

	_, err = db.Exec("INSERT INTO week_recipes (week_id, recipe_id, position) VALUES (?, ?, 0)", weekID, recipeID)
	assertErrorContains(t, err, "verified recipe")
}

func TestQuantityAndPackageConstraints(t *testing.T) {
	db := migratedDB(t)
	recipeID := mustInsertID(t, db, "INSERT INTO recipes (key, name) VALUES ('test', 'Test')")
	sectionID := mustInsertID(t, db, "INSERT INTO recipe_ingredient_sections (recipe_id, name, position) VALUES (?, 'Ingredients', 0)", recipeID)
	var unitID int64
	if err := db.QueryRow("SELECT id FROM units WHERE key = 'oz'").Scan(&unitID); err != nil {
		t.Fatal(err)
	}

	_, err := db.Exec(`INSERT INTO recipe_ingredients
		(section_id, position, source_text, quantity_kind, amount_min_numerator,
		 amount_min_denominator, unit_id)
		VALUES (?, 0, '1 oz cheese', 'exact', 1, 0, ?)`, sectionID, unitID)
	assertErrorContains(t, err, "CHECK constraint failed")

	_, err = db.Exec(`INSERT INTO recipe_ingredients
		(section_id, position, source_text, quantity_kind, amount_min_numerator,
		 amount_min_denominator, package_type, package_size_numerator)
		VALUES (?, 0, 'one 14.5 oz can', 'exact', 1, 1, 'can', 29)`, sectionID)
	assertErrorContains(t, err, "CHECK constraint failed")

	mustExec(t, db, `INSERT INTO recipe_ingredients
		(section_id, position, source_text, quantity_kind, amount_min_numerator,
		 amount_min_denominator, package_type, package_size_numerator,
		 package_size_denominator, package_size_unit_id)
		VALUES (?, 0, 'one 14.5 oz can', 'exact', 1, 1, 'can', 29, 2, ?)`, sectionID, unitID)
}

func mustInsertID(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	result, err := db.Exec(query, args...)
	if err != nil {
		t.Fatalf("execute %q: %v", query, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func mustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("execute %q: %v", query, err)
	}
}

func assertErrorContains(t *testing.T, err error, text string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q", text)
	}
	if !strings.Contains(err.Error(), text) {
		t.Fatalf("error %q does not contain %q", err, text)
	}
}
