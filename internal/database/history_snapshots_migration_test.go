package database_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pressly/goose/v3"

	"github.com/mbcoward3/grocery-router/internal/database"
	"github.com/mbcoward3/grocery-router/internal/ingest"
	"github.com/mbcoward3/grocery-router/internal/tenant"
	"github.com/mbcoward3/grocery-router/internal/week"
)

type firstPicker struct{}

func (firstPicker) IntN(int) (int, error) { return 0, nil }

// plannedWeek imports the corpus and generates a plan with groceries.
func plannedWeek(t *testing.T, db *sql.DB) week.View {
	t.Helper()
	documents, err := ingest.ReadDirectory(filepath.Join("..", "..", "corpus", "recipes"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ingest.Import(context.Background(), db, tenant.CowardHouseholdID, documents); err != nil {
		t.Fatal(err)
	}
	service := week.NewService(db, firstPicker{}).ForHousehold(tenant.CowardHouseholdID)
	view, err := service.Generate(context.Background(), time.Date(2026, time.August, 19, 12, 0, 0, 0, time.UTC), 3)
	if err != nil {
		t.Fatal(err)
	}
	return view
}

// Generating and swapping write the recipe's values onto the occurrence and its contributions.
func TestPlanWritesRecipeSnapshots(t *testing.T) {
	db := migratedDB(t)
	view := plannedWeek(t, db)
	assertSnapshotsMatchLiveRows(t, db)

	var replacement int64
	if err := db.QueryRow(`SELECT id FROM recipes WHERE status = 'verified'
		AND id NOT IN (SELECT recipe_id FROM week_recipes) ORDER BY id LIMIT 1`).Scan(&replacement); err != nil {
		t.Fatal(err)
	}
	service := week.NewService(db, firstPicker{}).ForHousehold(tenant.CowardHouseholdID)
	if _, err := service.Swap(context.Background(), view.Recipes[0].ID, replacement); err != nil {
		t.Fatal(err)
	}
	var snapshotted int64
	if err := db.QueryRow("SELECT recipe_id FROM week_recipes WHERE id = $1", view.Recipes[0].ID).Scan(&snapshotted); err != nil || snapshotted != replacement {
		t.Fatalf("swapped occurrence recipe = %d, %v; want %d", snapshotted, err, replacement)
	}
	assertSnapshotsMatchLiveRows(t, db)
}

// assertSnapshotsMatchLiveRows checks that every occurrence and contribution carries its recipe's
// current values.
func assertSnapshotsMatchLiveRows(t *testing.T, db *sql.DB) {
	t.Helper()
	var occurrences, mismatchedOccurrences int
	if err := db.QueryRow(`
		SELECT count(*), count(*) FILTER (WHERE
			wr.recipe_name IS DISTINCT FROM r.name OR wr.recipe_key IS DISTINCT FROM r.key
			OR wr.image_url IS DISTINCT FROM r.image_url OR wr.yield_text IS DISTINCT FROM r.yield_text
			OR wr.hands_on_min_minutes IS DISTINCT FROM r.hands_on_min_minutes
			OR wr.hands_on_max_minutes IS DISTINCT FROM r.hands_on_max_minutes
			OR wr.unattended_min_minutes IS DISTINCT FROM r.unattended_min_minutes
			OR wr.unattended_max_minutes IS DISTINCT FROM r.unattended_max_minutes)
		FROM week_recipes wr JOIN recipes r ON r.id = wr.recipe_id`).Scan(&occurrences, &mismatchedOccurrences); err != nil {
		t.Fatal(err)
	}
	if occurrences == 0 || mismatchedOccurrences != 0 {
		t.Fatalf("occurrence snapshots: %d of %d differ from their recipes", mismatchedOccurrences, occurrences)
	}
	var contributions, mismatchedContributions int
	if err := db.QueryRow(`
		SELECT count(*), count(*) FILTER (WHERE
			c.recipe_id IS DISTINCT FROM r.id OR c.recipe_name IS DISTINCT FROM r.name
			OR c.source_text IS DISTINCT FROM ri.source_text OR c.preparation IS DISTINCT FROM ri.preparation)
		FROM shopping_line_contributions c
		JOIN week_recipes wr ON wr.id = c.week_recipe_id
		JOIN recipes r ON r.id = wr.recipe_id
		JOIN recipe_ingredients ri ON ri.id = c.recipe_ingredient_id`).Scan(&contributions, &mismatchedContributions); err != nil {
		t.Fatal(err)
	}
	if contributions == 0 || mismatchedContributions != 0 {
		t.Fatalf("contribution snapshots: %d of %d differ from their recipes", mismatchedContributions, contributions)
	}
}

// The migration backfills snapshots for plans made before it, and rolls back even after an
// ingredient behind a retained contribution was deleted.
func TestHistorySnapshotsMigrationBackfillsAndRollsBack(t *testing.T) {
	ctx := context.Background()
	db := migratedDB(t)
	plannedWeek(t, db)
	provider, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.DownTo(ctx, 15); err != nil {
		t.Fatalf("roll back history snapshots: %v", err)
	}
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("reapply history snapshots: %v", err)
	}
	assertSnapshotsMatchLiveRows(t, db)

	// Detach one contribution by deleting its ingredient, then roll back and reapply again.
	var ingredient, recipe int64
	if err := db.QueryRow(`SELECT c.recipe_ingredient_id, c.recipe_id FROM shopping_line_contributions c
		ORDER BY c.id LIMIT 1`).Scan(&ingredient, &recipe); err != nil {
		t.Fatal(err)
	}
	mustExec(t, db, "UPDATE recipes SET status = 'draft', verified_at = NULL WHERE id = $1", recipe)
	mustExec(t, db, "DELETE FROM recipe_ingredients WHERE id = $1", ingredient)
	var detached int
	if err := db.QueryRow("SELECT count(*) FROM shopping_line_contributions WHERE recipe_ingredient_id IS NULL").Scan(&detached); err != nil || detached == 0 {
		t.Fatalf("detached contributions = %d, %v", detached, err)
	}
	if _, err := provider.DownTo(ctx, 15); err != nil {
		t.Fatalf("roll back history snapshots with a detached contribution: %v", err)
	}
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("reapply history snapshots after rollback: %v", err)
	}
	var nullIngredients int
	if err := db.QueryRow("SELECT count(*) FROM shopping_line_contributions WHERE recipe_ingredient_id IS NULL").Scan(&nullIngredients); err != nil || nullIngredients != 0 {
		t.Fatalf("contributions without an ingredient after rollback = %d, %v", nullIngredients, err)
	}
}

// The matching check still guards every change except detaching a deleted ingredient.
func TestContributionMatchingCheckStillRejects(t *testing.T) {
	db := migratedDB(t)
	plannedWeek(t, db)
	var id, line, occurrence, ingredient int64
	if err := db.QueryRow(`SELECT id, shopping_line_id, week_recipe_id, recipe_ingredient_id
		FROM shopping_line_contributions ORDER BY id LIMIT 1`).Scan(&id, &line, &occurrence, &ingredient); err != nil {
		t.Fatal(err)
	}
	var otherIngredient int64
	if err := db.QueryRow(`SELECT ri.id FROM recipe_ingredients ri
		JOIN recipe_ingredient_sections ris ON ris.id = ri.section_id
		WHERE ris.recipe_id <> (SELECT recipe_id FROM week_recipes WHERE id = $1)
		ORDER BY ri.id LIMIT 1`, occurrence).Scan(&otherIngredient); err != nil {
		t.Fatal(err)
	}
	var otherLine int64
	if err := db.QueryRow(`SELECT id FROM shopping_lines WHERE origin = 'generated' AND id <> $1
		ORDER BY id LIMIT 1`, line).Scan(&otherLine); err != nil {
		t.Fatal(err)
	}
	const mismatch = "contribution must match its line, week occurrence, and recipe ingredient"

	_, err := db.Exec("UPDATE shopping_line_contributions SET recipe_ingredient_id = $1 WHERE id = $2", otherIngredient, id)
	assertErrorContains(t, err, mismatch)
	_, err = db.Exec("UPDATE shopping_line_contributions SET shopping_line_id = $1 WHERE id = $2", otherLine, id)
	assertErrorContains(t, err, mismatch)
	// Detaching while moving the row is not the exempt change.
	_, err = db.Exec("UPDATE shopping_line_contributions SET recipe_ingredient_id = NULL, shopping_line_id = $1 WHERE id = $2", otherLine, id)
	assertErrorContains(t, err, mismatch)
	_, err = db.Exec(`INSERT INTO shopping_line_contributions (
			household_id, shopping_line_id, week_recipe_id, recipe_ingredient_id, quantity_kind, is_optional,
			recipe_id, recipe_name, source_text)
		SELECT household_id, shopping_line_id, week_recipe_id, NULL, 'unspecified', 0, recipe_id, recipe_name, source_text
		FROM shopping_line_contributions WHERE id = $1`, id)
	assertErrorContains(t, err, mismatch)

	// Detaching in place is the one exempt change.
	mustExec(t, db, "UPDATE shopping_line_contributions SET recipe_ingredient_id = NULL WHERE id = $1", id)
	// A detached row cannot be pointed at an ingredient again unless it matches.
	_, err = db.Exec("UPDATE shopping_line_contributions SET recipe_ingredient_id = $1 WHERE id = $2", otherIngredient, id)
	assertErrorContains(t, err, mismatch)
}
