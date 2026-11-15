package database_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/database"
	"github.com/mbcoward3/grocery-router/internal/tenant"
	"github.com/pressly/goose/v3"
)

// verifiedRecipe inserts a minimal verified recipe approved by approver ("" for none).
func verifiedRecipe(t *testing.T, db *sql.DB, key, approver string) int64 {
	t.Helper()
	section := mustInsertID(t, db, "INSERT INTO store_sections (household_id, key, name) VALUES ($1, $2, $2)", tenant.CowardHouseholdID, key)
	item := mustInsertID(t, db, `INSERT INTO grocery_items (household_id, key, name, store_section_id, shopping_mode)
		VALUES ($1, $2, $2, $3, 'counted')`, tenant.CowardHouseholdID, key, section)
	recipe := mustInsertID(t, db, "INSERT INTO recipes (household_id, key, name) VALUES ($1, $2, $2)", tenant.CowardHouseholdID, key)
	mustExec(t, db, "INSERT INTO recipe_sources (recipe_id, relationship, attribution, is_primary, position) VALUES ($1, 'source', 'Test', 1, 0)", recipe)
	ingredients := mustInsertID(t, db, "INSERT INTO recipe_ingredient_sections (household_id, recipe_id, name, position) VALUES ($1, $2, 'Ingredients', 0)", tenant.CowardHouseholdID, recipe)
	mustExec(t, db, `INSERT INTO recipe_ingredients (household_id, section_id, grocery_item_id, position, source_text, quantity_kind)
		VALUES ($1, $2, $3, 0, 'Item', 'unspecified')`, tenant.CowardHouseholdID, ingredients, item)
	steps := mustInsertID(t, db, "INSERT INTO recipe_instruction_sections (recipe_id, name, position) VALUES ($1, 'Method', 0)", recipe)
	mustExec(t, db, "INSERT INTO recipe_steps (section_id, position, instruction) VALUES ($1, 0, 'Prepare')", steps)
	mustExec(t, db, "UPDATE recipes SET status = 'reviewable' WHERE id = $1", recipe)
	mustExec(t, db, "UPDATE recipes SET status = 'verified', verified_at = '2026-11-14T00:00:00Z', verified_by = NULLIF($2, '')::uuid WHERE id = $1", recipe, approver)
	return recipe
}

func TestRecipeVerifiedByForgetsDeletedUserAndRollsBack(t *testing.T) {
	db := migratedDB(t)
	var user string
	if err := db.QueryRow("INSERT INTO app_users (primary_email, display_name) VALUES ('cook@example.com', 'Cook') RETURNING id::text").Scan(&user); err != nil {
		t.Fatal(err)
	}
	recipe := verifiedRecipe(t, db, "approved", user)

	// Only forgetting the approver is exempt; any other change to a verified recipe still needs
	// it returned to draft, including one that also clears verified_by.
	_, err := db.Exec("UPDATE recipes SET name = 'Renamed', verified_by = NULL WHERE id = $1", recipe)
	assertErrorContains(t, err, "return recipe to draft before editing")
	other := verifiedRecipe(t, db, "second-approved", user)
	_, err = db.Exec("UPDATE recipes SET verified_by = NULL, verified_at = '2026-11-15T00:00:00Z' WHERE id = $1", other)
	assertErrorContains(t, err, "return recipe to draft before editing")

	mustExec(t, db, "DELETE FROM app_users WHERE id = $1", user)
	var status string
	var approver sql.NullString
	if err := db.QueryRow("SELECT status, verified_by::text FROM recipes WHERE id = $1", recipe).Scan(&status, &approver); err != nil {
		t.Fatal(err)
	}
	if status != "verified" || approver.Valid {
		t.Fatalf("after deleting the approver: status %q, verified_by %v; want verified and NULL", status, approver)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.DownTo(context.Background(), 16); err != nil {
		t.Fatalf("roll back verified_by: %v", err)
	}
	var columns int
	if err := db.QueryRow("SELECT count(*) FROM information_schema.columns WHERE table_name = 'recipes' AND column_name = 'verified_by'").Scan(&columns); err != nil {
		t.Fatal(err)
	}
	if columns != 0 {
		t.Fatal("verified_by survived the down migration")
	}
	_, err = db.Exec("UPDATE recipes SET name = name WHERE id = $1", recipe)
	assertErrorContains(t, err, "return recipe to draft before editing")
	if err := database.Migrate(context.Background(), db); err != nil {
		t.Fatalf("reapply verified_by: %v", err)
	}
}
