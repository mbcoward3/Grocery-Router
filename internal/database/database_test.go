package database_test

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/testdatabase"
)

func migratedDB(t *testing.T) *sql.DB {
	t.Helper()
	return testdatabase.Open(t)
}

func TestMigrateEmptyDatabase(t *testing.T) {
	db := migratedDB(t)

	for _, table := range []string{
		"recipes", "recipe_sources", "recipe_ingredient_sections", "recipe_ingredients",
		"recipe_instruction_sections", "recipe_steps", "recipe_review_flags",
		"store_sections", "grocery_items", "units", "weeks", "week_recipes",
		"shopping_lists", "shopping_lines", "shopping_line_contributions",
		"app_users", "user_identities", "households", "household_memberships", "auth_sessions",
	} {
		var count int
		err := db.QueryRow("SELECT count(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = $1", table).Scan(&count)
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

	recipeID := mustInsertID(t, db, "INSERT INTO recipes (household_id, key, name) VALUES ('c0a7a2d8-669b-4e47-91c1-4d9a32f339d5', 'test-soup', 'Test Soup')")

	// No source, ingredient, instruction, or reviewable transition may be skipped.
	_, err := db.Exec("UPDATE recipes SET status = 'verified', verified_at = $1 WHERE id = $2", "2026-08-17T00:00:00Z", recipeID)
	assertErrorContains(t, err, "recipe must be reviewable")

	mustExec(t, db, `INSERT INTO recipe_sources
		(recipe_id, relationship, attribution, is_primary, position)
		VALUES ($1, 'source', 'Recipes.pdf', 1, 0)`, recipeID)
	sectionID := mustInsertID(t, db, `INSERT INTO recipe_ingredient_sections
		(recipe_id, name, position) VALUES ($1, 'Ingredients', 0)`, recipeID)
	ingredientID := mustInsertID(t, db, `INSERT INTO recipe_ingredients
		(section_id, position, source_text, quantity_kind)
		VALUES ($1, 0, 'salt to taste', 'unspecified')`, sectionID)
	instructionSectionID := mustInsertID(t, db, `INSERT INTO recipe_instruction_sections
		(recipe_id, name, position) VALUES ($1, 'Method', 0)`, recipeID)
	mustExec(t, db, `INSERT INTO recipe_steps (section_id, position, instruction)
		VALUES ($1, 0, 'Season the soup.')`, instructionSectionID)
	flagID := mustInsertID(t, db, `INSERT INTO recipe_review_flags
		(recipe_id, field_path, kind, note) VALUES ($1, 'steps[0]', 'backfilled', 'Drafted from household description')`, recipeID)
	mustExec(t, db, "UPDATE recipes SET status = 'reviewable' WHERE id = $1", recipeID)

	_, err = db.Exec("UPDATE recipes SET status = 'verified', verified_at = $1 WHERE id = $2", "2026-08-17T00:00:00Z", recipeID)
	assertErrorContains(t, err, "unmapped ingredient")

	storeSectionID := mustInsertID(t, db, "INSERT INTO store_sections (household_id, key, name) VALUES ('c0a7a2d8-669b-4e47-91c1-4d9a32f339d5', 'spices', 'Spices')")
	groceryItemID := mustInsertID(t, db, `INSERT INTO grocery_items
		(household_id, key, name, store_section_id, shopping_mode) VALUES ('c0a7a2d8-669b-4e47-91c1-4d9a32f339d5', 'salt', 'Salt', $1, 'presence-only')`, storeSectionID)
	mustExec(t, db, "UPDATE recipe_ingredients SET grocery_item_id = $1 WHERE id = $2", groceryItemID, ingredientID)

	_, err = db.Exec("UPDATE recipes SET status = 'verified', verified_at = $1 WHERE id = $2", "2026-08-17T00:00:00Z", recipeID)
	assertErrorContains(t, err, "unapproved review flags")

	mustExec(t, db, "UPDATE recipe_review_flags SET approved = 1 WHERE id = $1", flagID)
	mustExec(t, db, "UPDATE recipes SET status = 'verified', verified_at = $1 WHERE id = $2", "2026-08-17T00:00:00Z", recipeID)

	var status string
	if err := db.QueryRow("SELECT status FROM recipes WHERE id = $1", recipeID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "verified" {
		t.Fatalf("status = %q, want verified", status)
	}

	_, err = db.Exec("UPDATE recipe_ingredients SET source_text = 'more salt' WHERE id = $1", ingredientID)
	assertErrorContains(t, err, "return recipe to draft")
	_, err = db.Exec("UPDATE recipes SET name = 'Changed' WHERE id = $1", recipeID)
	assertErrorContains(t, err, "return recipe to draft")

	mustExec(t, db, "UPDATE recipes SET status = 'draft', verified_at = NULL WHERE id = $1", recipeID)
	mustExec(t, db, "UPDATE recipe_ingredients SET source_text = 'more salt' WHERE id = $1", ingredientID)
}

func TestWeekDateAndVerifiedRecipeConstraints(t *testing.T) {
	db := migratedDB(t)

	_, err := db.Exec("INSERT INTO weeks (household_id, starts_on) VALUES ('c0a7a2d8-669b-4e47-91c1-4d9a32f339d5', '2026-08-17')")
	assertErrorContains(t, err, "violates check constraint")
	_, err = db.Exec("INSERT INTO weeks (household_id, starts_on) VALUES ('c0a7a2d8-669b-4e47-91c1-4d9a32f339d5', '2026-99-99')")
	assertErrorContains(t, err, "date/time field value out of range")
	weekID := mustInsertID(t, db, "INSERT INTO weeks (household_id, starts_on) VALUES ('c0a7a2d8-669b-4e47-91c1-4d9a32f339d5', '2026-08-16')")
	recipeID := mustInsertID(t, db, "INSERT INTO recipes (household_id, key, name) VALUES ('c0a7a2d8-669b-4e47-91c1-4d9a32f339d5', 'draft', 'Draft')")

	_, err = db.Exec("INSERT INTO week_recipes (household_id, week_id, recipe_id, position) VALUES ('c0a7a2d8-669b-4e47-91c1-4d9a32f339d5', $1, $2, 0)", weekID, recipeID)
	assertErrorContains(t, err, "verified recipe")
}

func TestQuantityAndPackageConstraints(t *testing.T) {
	db := migratedDB(t)
	recipeID := mustInsertID(t, db, "INSERT INTO recipes (household_id, key, name) VALUES ('c0a7a2d8-669b-4e47-91c1-4d9a32f339d5', 'test', 'Test')")
	sectionID := mustInsertID(t, db, "INSERT INTO recipe_ingredient_sections (recipe_id, name, position) VALUES ($1, 'Ingredients', 0)", recipeID)
	var unitID int64
	if err := db.QueryRow("SELECT id FROM units WHERE key = 'oz'").Scan(&unitID); err != nil {
		t.Fatal(err)
	}

	_, err := db.Exec(`INSERT INTO recipe_ingredients
		(section_id, position, source_text, quantity_kind, amount_min_numerator,
		 amount_min_denominator, unit_id)
		VALUES ($1, 0, '1 oz cheese', 'exact', 1, 0, $2)`, sectionID, unitID)
	assertErrorContains(t, err, "violates check constraint")

	_, err = db.Exec(`INSERT INTO recipe_ingredients
		(section_id, position, source_text, quantity_kind, amount_min_numerator,
		 amount_min_denominator, package_type, package_size_numerator)
		VALUES ($1, 0, 'one 14.5 oz can', 'exact', 1, 1, 'can', 29)`, sectionID)
	assertErrorContains(t, err, "violates check constraint")

	mustExec(t, db, `INSERT INTO recipe_ingredients
		(section_id, position, source_text, quantity_kind, amount_min_numerator,
		 amount_min_denominator, package_type, package_size_numerator,
		 package_size_denominator, package_size_unit_id)
		VALUES ($1, 0, 'one 14.5 oz can', 'exact', 1, 1, 'can', 29, 2, $2)`, sectionID, unitID)
}

func TestHouseholdConstraintsRejectCrossTenantRelationships(t *testing.T) {
	db := migratedDB(t)
	const coward = "c0a7a2d8-669b-4e47-91c1-4d9a32f339d5"
	var other string
	if err := db.QueryRow("INSERT INTO households (name) VALUES ('Other') RETURNING id::text").Scan(&other); err != nil {
		t.Fatal(err)
	}
	sectionID := mustInsertID(t, db, "INSERT INTO store_sections (household_id, key, name) VALUES ($1, 'produce', 'Produce')", coward)
	_, err := db.Exec(`INSERT INTO grocery_items
		(household_id, key, name, store_section_id, shopping_mode)
		VALUES ($1, 'apple', 'Apple', $2, 'counted')`, other, sectionID)
	assertErrorContains(t, err, "grocery_items_household_section_fkey")

	otherWeekID := mustInsertID(t, db, "INSERT INTO weeks (household_id, starts_on) VALUES ($1, '2026-08-16')", other)
	_, err = db.Exec("INSERT INTO shopping_lists (household_id, week_id) VALUES ($1, $2)", coward, otherWeekID)
	assertErrorContains(t, err, "shopping_lists_household_week_fkey")
}

func TestHouseholdCannotLoseLastOwner(t *testing.T) {
	db := migratedDB(t)
	const coward = "c0a7a2d8-669b-4e47-91c1-4d9a32f339d5"
	var userID string
	if err := db.QueryRow(`INSERT INTO app_users (primary_email, display_name)
		VALUES ('owner@example.com', 'Owner') RETURNING id::text`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	mustExec(t, db, "INSERT INTO household_memberships (household_id, user_id, role) VALUES ($1, $2, 'owner')", coward, userID)
	_, err := db.Exec("DELETE FROM household_memberships WHERE household_id = $1 AND user_id = $2", coward, userID)
	assertErrorContains(t, err, "retain at least one owner")
}

func mustInsertID(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(query+" RETURNING id", args...).Scan(&id); err != nil {
		t.Fatalf("execute %q: %v", query, err)
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
