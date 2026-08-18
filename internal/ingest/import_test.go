package ingest_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/database"
	"github.com/mbcoward3/grocery-router/internal/ingest"
)

func approvedDocument(t *testing.T) ingest.Document {
	t.Helper()
	path := filepath.Join("..", "..", "corpus", "recipes", "chicken-and-biscuits-casserole.md")
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	document, err := ingest.ParseDocument(file)
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func TestParseApprovedRecipe(t *testing.T) {
	document := approvedDocument(t)
	if document.Key != "chicken-and-biscuits-casserole" || document.Status != "verified" {
		t.Fatalf("parsed recipe = %q, %q", document.Key, document.Status)
	}
	if got := len(document.IngredientSections[0].Ingredients); got != 10 {
		t.Fatalf("ingredients = %d, want 10", got)
	}
	chicken := document.IngredientSections[0].Ingredients[8]
	if chicken.GroceryItem.Name != "Cooked Chicken" || chicken.Quantity.Amount != "2" || chicken.Note != "suggestion: rotisserie chicken" {
		t.Fatalf("cooked chicken parsed as %#v", chicken)
	}
}

func TestStrictFrontMatterRejectsUnknownField(t *testing.T) {
	path := filepath.Join("..", "..", "corpus", "recipes", "chicken-and-biscuits-casserole.md")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	contents = []byte(strings.Replace(string(contents), "format_version: 1", "format_version: 1\nunknown_field: nope", 1))
	_, err = ingest.ParseDocument(strings.NewReader(string(contents)))
	if err == nil || !strings.Contains(err.Error(), "field unknown_field not found") {
		t.Fatalf("error = %v, want strict YAML failure", err)
	}
}

func TestImportApprovedCorpus(t *testing.T) {
	db := openMigratedDB(t)
	document := approvedDocument(t)
	if err := ingest.Import(context.Background(), db, []ingest.Document{document}); err != nil {
		t.Fatal(err)
	}

	var recipes, ingredients, steps, unapproved int
	mustCount(t, db, "SELECT count(*) FROM recipes WHERE status = 'verified'", &recipes)
	mustCount(t, db, "SELECT count(*) FROM recipe_ingredients", &ingredients)
	mustCount(t, db, "SELECT count(*) FROM recipe_steps", &steps)
	mustCount(t, db, "SELECT count(*) FROM recipe_review_flags WHERE approved = 0", &unapproved)
	if recipes != 1 || ingredients != 10 || steps != 6 || unapproved != 0 {
		t.Fatalf("import counts: recipes=%d ingredients=%d steps=%d unapproved=%d", recipes, ingredients, steps, unapproved)
	}

	var item, section, mode, note string
	if err := db.QueryRow(`
		SELECT gi.name, ss.name, gi.shopping_mode, ri.display_note
		FROM recipe_ingredients ri
		JOIN grocery_items gi ON gi.id = ri.grocery_item_id
		JOIN store_sections ss ON ss.id = gi.store_section_id
		WHERE gi.key = 'cooked-chicken'
	`).Scan(&item, &section, &mode, &note); err != nil {
		t.Fatal(err)
	}
	if item != "Cooked Chicken" || section != "Meat" || mode != "measured" || note != "suggestion: rotisserie chicken" {
		t.Fatalf("cooked chicken = %q, %q, %q, %q", item, section, mode, note)
	}

	if err := ingest.Import(context.Background(), db, []ingest.Document{document}); err == nil {
		t.Fatal("second import unexpectedly succeeded")
	}
}

func TestImportRollsBackCompleteSet(t *testing.T) {
	db := openMigratedDB(t)
	document := approvedDocument(t)
	err := ingest.Import(context.Background(), db, []ingest.Document{document, document})
	if err == nil {
		t.Fatal("duplicate document import unexpectedly succeeded")
	}
	var recipes int
	mustCount(t, db, "SELECT count(*) FROM recipes", &recipes)
	if recipes != 0 {
		t.Fatalf("failed import left %d recipes", recipes)
	}
}

func openMigratedDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:ingest-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := database.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return db
}

func mustCount(t *testing.T, db *sql.DB, query string, destination *int) {
	t.Helper()
	if err := db.QueryRow(query).Scan(destination); err != nil {
		t.Fatal(err)
	}
}
