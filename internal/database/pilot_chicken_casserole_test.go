package database_test

import (
	"database/sql"
	"testing"
)

// This is a proposed, deliberately unverified pilot from the authoritative linked source
// checked on 2026-08-17. It proves the schema can represent source measurements, package
// sizes, optional ingredients, preparation, presence-only shopping, and review flags without
// making the draft selectable.
func TestChickenAndBiscuitsPilotFitsCorpusSchema(t *testing.T) {
	db := migratedDB(t)

	pantry := mustInsertID(t, db, "INSERT INTO store_sections (key, name) VALUES ('pantry', 'Pantry')")
	dairy := mustInsertID(t, db, "INSERT INTO store_sections (key, name) VALUES ('dairy', 'Dairy')")
	frozen := mustInsertID(t, db, "INSERT INTO store_sections (key, name) VALUES ('frozen', 'Frozen')")
	deli := mustInsertID(t, db, "INSERT INTO store_sections (key, name) VALUES ('deli', 'Deli')")
	produce := mustInsertID(t, db, "INSERT INTO store_sections (key, name) VALUES ('produce', 'Produce')")
	spices := mustInsertID(t, db, "INSERT INTO store_sections (key, name) VALUES ('spices', 'Spices')")

	each := createUnit(t, db, "each", "each", "each", "count", 1, 1)
	cup := createUnit(t, db, "cup", "cup", "cup", "volume", 48, 1)
	tsp := createUnit(t, db, "tsp", "teaspoon", "tsp", "volume", 1, 1)
	oz := createUnit(t, db, "oz", "ounce", "oz", "mass", 1, 1)
	_ = each // package counts use package_type rather than the count unit.

	creamSoup := createItem(t, db, "cream-of-chicken-soup", "Cream of Chicken Soup", pantry, "counted")
	milk := createItem(t, db, "milk", "Milk", dairy, "measured")
	garlicPowder := createItem(t, db, "garlic-powder", "Garlic Powder", spices, "presence-only")
	rotisserieSeasoning := createItem(t, db, "rotisserie-seasoning", "Rotisserie Seasoning", spices, "presence-only")
	blackPepper := createItem(t, db, "black-pepper", "Black Pepper", spices, "presence-only")
	biscuits := createItem(t, db, "refrigerated-biscuits", "Refrigerated Biscuits", dairy, "counted")
	peasCarrots := createItem(t, db, "frozen-peas-and-carrots", "Frozen Peas and Carrots", frozen, "measured")
	cheddar := createItem(t, db, "shredded-cheddar-cheese", "Shredded Cheddar Cheese", dairy, "measured")
	rotisserieChicken := createItem(t, db, "rotisserie-chicken", "Rotisserie Chicken", deli, "presence-only")
	greenOnion := createItem(t, db, "green-onion", "Green Onion", produce, "measured")

	recipeID := mustInsertID(t, db, `INSERT INTO recipes
		(key, name, yield_text, hands_on_min_minutes, hands_on_max_minutes,
		 unattended_min_minutes, unattended_max_minutes)
		VALUES ('chicken-and-biscuits-casserole', 'Chicken and Biscuits Casserole',
		 '6 servings', 15, 15, 40, 40)`)
	mustExec(t, db, `INSERT INTO recipe_sources
		(recipe_id, relationship, attribution, url, checked_on, is_primary, position)
		VALUES (?, 'source', 'The Country Cook',
		 'https://www.thecountrycook.net/chicken-and-biscuits-casserole/',
		 '2026-08-17', 1, 0)`, recipeID)
	sectionID := mustInsertID(t, db, `INSERT INTO recipe_ingredient_sections
		(recipe_id, name, position) VALUES (?, 'Ingredients', 0)`, recipeID)

	insertPackageIngredient(t, db, sectionID, creamSoup, 0,
		"2 (10.5 ounce) cans cream of chicken soup", 2, "can", 21, 2, oz, false, sql.NullString{})
	insertMeasuredIngredient(t, db, sectionID, milk, 1, "1 cup milk", 1, 1, cup, false, nil)
	insertMeasuredIngredient(t, db, sectionID, garlicPowder, 2, "1 teaspoon garlic powder", 1, 1, tsp, false, nil)
	insertMeasuredIngredient(t, db, sectionID, rotisserieSeasoning, 3, "1/2 teaspoon rotisserie seasoning", 1, 2, tsp, false, nil)
	insertMeasuredIngredient(t, db, sectionID, blackPepper, 4, "1/2 teaspoon black pepper", 1, 2, tsp, false, nil)
	insertPackageIngredient(t, db, sectionID, biscuits, 5,
		"12 ounce can refrigerated biscuits (or two smaller 6 ounce cans)", 1, "can", 12, 1, oz, false,
		nullString("v1 default: one 12 ounce can"))
	insertMeasuredIngredient(t, db, sectionID, peasCarrots, 6,
		"1 cup frozen peas and carrots (allow to thaw slightly)", 1, 1, cup, false, strPtr("allow to thaw slightly"))
	insertMeasuredIngredient(t, db, sectionID, cheddar, 7,
		"1 cup shredded cheddar cheese", 1, 1, cup, false, strPtr("shredded"))
	insertMeasuredIngredient(t, db, sectionID, rotisserieChicken, 8,
		"2 cups cooked chicken (shredded or diced)", 2, 1, cup, false, strPtr("shredded or diced"))
	insertMeasuredIngredient(t, db, sectionID, greenOnion, 9,
		"1/4 cup sliced green onion (optional)", 1, 4, cup, true, strPtr("sliced"))

	methodID := mustInsertID(t, db, `INSERT INTO recipe_instruction_sections
		(recipe_id, name, position) VALUES (?, 'Method', 0)`, recipeID)
	steps := []string{
		"Heat the oven to 375°F and coat a 9×13-inch baking dish with nonstick cooking spray.",
		"Whisk the cream of chicken soup, milk, garlic powder, rotisserie seasoning, and black pepper until mostly smooth.",
		"Separate the biscuits, cut each into quarters, and stir them into the soup mixture.",
		"Fold in the peas and carrots, cheddar, and chicken, then spread the mixture in the prepared dish.",
		"Bake uncovered on the middle rack for 35–45 minutes, until the biscuits are golden and the filling bubbles at the edges.",
		"If the center needs more time, cover the top with foil and continue baking. Cool briefly, top with optional green onion, and serve.",
	}
	for i, step := range steps {
		mustExec(t, db, `INSERT INTO recipe_steps (section_id, position, instruction) VALUES (?, ?, ?)`, methodID, i, step)
	}
	mustExec(t, db, `INSERT INTO recipe_review_flags (recipe_id, field_path, kind, note)
		VALUES (?, 'ingredients[5]', 'conflict-resolved', 'Selected the source-listed 12 ounce biscuit can instead of two 6 ounce cans')`, recipeID)
	mustExec(t, db, `INSERT INTO recipe_review_flags (recipe_id, field_path, kind, note)
		VALUES (?, 'ingredients[8].grocery_item', 'conflict-resolved', 'Mapped cooked chicken to the author-recommended rotisserie chicken; shopping quantity remains presence-only')`, recipeID)
	mustExec(t, db, `INSERT INTO recipe_review_flags (recipe_id, field_path, kind, note)
		VALUES (?, 'steps', 'rewritten', 'Condensed source instructions without changing the method')`, recipeID)

	mustExec(t, db, "UPDATE recipes SET status = 'reviewable' WHERE id = ?", recipeID)
	var ingredients, stepsCount, openFlags int
	mustScan(t, db, "SELECT count(*) FROM recipe_ingredients WHERE section_id = ?", []any{sectionID}, &ingredients)
	mustScan(t, db, "SELECT count(*) FROM recipe_steps WHERE section_id = ?", []any{methodID}, &stepsCount)
	mustScan(t, db, "SELECT count(*) FROM recipe_review_flags WHERE recipe_id = ? AND approved = 0", []any{recipeID}, &openFlags)
	if ingredients != 10 || stepsCount != 6 || openFlags != 3 {
		t.Fatalf("pilot counts: ingredients=%d steps=%d flags=%d", ingredients, stepsCount, openFlags)
	}

	_, err := db.Exec("UPDATE recipes SET status = 'verified', verified_at = '2026-08-17T00:00:00Z' WHERE id = ?", recipeID)
	assertErrorContains(t, err, "unapproved review flags")
}

func createUnit(t *testing.T, db *sql.DB, key, name, symbol, dimension string, numerator, denominator int64) int64 {
	t.Helper()
	return mustInsertID(t, db, `INSERT INTO units
		(key, name, symbol, dimension, to_base_numerator, to_base_denominator)
		VALUES (?, ?, ?, ?, ?, ?)`, key, name, symbol, dimension, numerator, denominator)
}

func createItem(t *testing.T, db *sql.DB, key, name string, sectionID int64, mode string) int64 {
	t.Helper()
	return mustInsertID(t, db, `INSERT INTO grocery_items
		(key, name, store_section_id, shopping_mode) VALUES (?, ?, ?, ?)`, key, name, sectionID, mode)
}

func insertMeasuredIngredient(t *testing.T, db *sql.DB, sectionID, itemID int64, position int,
	source string, numerator, denominator, unitID int64, optional bool, preparation *string,
) {
	t.Helper()
	mustExec(t, db, `INSERT INTO recipe_ingredients
		(section_id, grocery_item_id, position, source_text, quantity_kind,
		 amount_min_numerator, amount_min_denominator, unit_id, preparation, is_optional)
		VALUES (?, ?, ?, ?, 'exact', ?, ?, ?, ?, ?)`,
		sectionID, itemID, position, source, numerator, denominator, unitID, preparation, boolInt(optional))
}

func insertPackageIngredient(t *testing.T, db *sql.DB, sectionID, itemID int64, position int,
	source string, count int64, packageType string, sizeNumerator, sizeDenominator, sizeUnitID int64,
	optional bool, note sql.NullString,
) {
	t.Helper()
	mustExec(t, db, `INSERT INTO recipe_ingredients
		(section_id, grocery_item_id, position, source_text, quantity_kind,
		 amount_min_numerator, amount_min_denominator, package_type,
		 package_size_numerator, package_size_denominator, package_size_unit_id,
		 is_optional, display_note)
		VALUES (?, ?, ?, ?, 'exact', ?, 1, ?, ?, ?, ?, ?, ?)`,
		sectionID, itemID, position, source, count, packageType, sizeNumerator, sizeDenominator,
		sizeUnitID, boolInt(optional), note)
}

func mustScan(t *testing.T, db *sql.DB, query string, args []any, destination ...any) {
	t.Helper()
	if err := db.QueryRow(query, args...).Scan(destination...); err != nil {
		t.Fatal(err)
	}
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func strPtr(value string) *string { return &value }

func nullString(value string) sql.NullString { return sql.NullString{String: value, Valid: true} }
