package ingest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mbcoward3/grocery-router/internal/store"
)

// ReadDirectory parses every Markdown recipe in a bootstrap directory in stable filename
// order. Any invalid file rejects the complete set before database writes begin.
func ReadDirectory(path string) ([]Document, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read corpus directory: %w", err)
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, fmt.Errorf("corpus directory %s contains no Markdown recipes", path)
	}

	documents := make([]Document, 0, len(names))
	seen := make(map[string]string, len(names))
	for _, name := range names {
		file, err := os.Open(filepath.Join(path, name))
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", name, err)
		}
		document, parseErr := ParseDocument(file)
		file.Close()
		if parseErr != nil {
			return nil, fmt.Errorf("%s: %w", name, parseErr)
		}
		if previous, ok := seen[document.Key]; ok {
			return nil, fmt.Errorf("%s and %s use duplicate recipe key %q", previous, name, document.Key)
		}
		seen[document.Key] = name
		documents = append(documents, document)
	}
	return documents, nil
}

// Import inserts a complete approved document set into an empty migrated corpus. It is
// intentionally not an upsert path: bootstrap import either commits every recipe or none.
func Import(ctx context.Context, db *sql.DB, householdID string, documents []Document) error {
	if len(documents) == 0 {
		return fmt.Errorf("no recipe documents to import")
	}
	for _, document := range documents {
		if err := document.Validate(); err != nil {
			return err
		}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin corpus import: %w", err)
	}
	defer tx.Rollback()

	var existing int
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM recipes WHERE household_id = $1", householdID).Scan(&existing); err != nil {
		return fmt.Errorf("count existing recipes: %w", err)
	}
	if existing != 0 {
		return fmt.Errorf("corpus import requires an empty recipe table, found %d", existing)
	}

	queries := store.New(tx)
	cache := newContentCache()
	for _, document := range documents {
		if err := importDocument(ctx, queries, householdID, document, cache); err != nil {
			return fmt.Errorf("import recipe %s: %w", document.Key, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit corpus import: %w", err)
	}
	return nil
}

// contentCache remembers the store sections, grocery items, and units one transaction has
// already resolved, so a recipe set reads each only once.
type contentCache struct {
	sections map[string]store.StoreSection
	items    map[string]store.GroceryItem
	units    map[string]store.Unit
}

func newContentCache() *contentCache {
	return &contentCache{
		sections: make(map[string]store.StoreSection),
		items:    make(map[string]store.GroceryItem),
		units:    make(map[string]store.Unit),
	}
}

func importDocument(ctx context.Context, queries *store.Queries, householdID string, document Document, cache *contentCache) error {
	content := document.Content()
	recipe, err := queries.CreateDraftRecipe(ctx, store.CreateDraftRecipeParams{
		HouseholdID:          householdID,
		Key:                  document.Key,
		Name:                 content.Name,
		ImageUrl:             nullString(content.ImageURL),
		YieldText:            nullString(content.Yield),
		HandsOnMinMinutes:    durationMinimum(content.HandsOn),
		HandsOnMaxMinutes:    durationMaximum(content.HandsOn),
		UnattendedMinMinutes: durationMinimum(content.Unattended),
		UnattendedMaxMinutes: durationMaximum(content.Unattended),
	})
	if err != nil {
		return err
	}
	if err := insertRecipeContent(ctx, queries, householdID, recipe.ID, content, cache); err != nil {
		return err
	}
	for _, decision := range document.Review {
		flag, err := queries.CreateReviewFlag(ctx, store.CreateReviewFlagParams{
			RecipeID:  recipe.ID,
			FieldPath: decision.Field,
			Kind:      decision.Kind,
			Note:      decision.Note,
		})
		if err != nil {
			return err
		}
		if decision.Approved {
			if _, err := queries.ApproveReviewFlag(ctx, flag.ID); err != nil {
				return err
			}
		}
	}
	_, err = verifyDraftRecipe(ctx, queries, householdID, recipe.ID, document.ApprovedOn+"T00:00:00Z", "")
	return err
}

// insertRecipeContent writes a draft recipe's primary source, ingredient sections, and
// instruction sections. The recipe row itself already holds the content's metadata. Both the
// bootstrap import and an in-app edit use it.
func insertRecipeContent(ctx context.Context, queries *store.Queries, householdID string, recipeID int64, content RecipeContent, cache *contentCache) error {
	if _, err := queries.CreateRecipeSource(ctx, store.CreateRecipeSourceParams{
		RecipeID:     recipeID,
		Relationship: content.Source.Relationship,
		Attribution:  content.Source.Attribution,
		Url:          nullString(content.Source.URL),
		CheckedOn:    nullString(content.Source.CheckedOn),
		IsPrimary:    1,
		Position:     0,
	}); err != nil {
		return err
	}

	for sectionPosition, ingredientSection := range content.IngredientSections {
		section, err := queries.CreateIngredientSection(ctx, store.CreateIngredientSectionParams{
			HouseholdID: householdID,
			RecipeID:    recipeID,
			Name:        ingredientSection.Name,
			Position:    int64(sectionPosition),
		})
		if err != nil {
			return err
		}
		for ingredientPosition, ingredient := range ingredientSection.Ingredients {
			item, err := ensureGroceryItem(ctx, queries, householdID, ingredient.GroceryItem, cache.sections, cache.items)
			if err != nil {
				return err
			}
			params, err := ingredientParams(ctx, queries, householdID, ingredient, section.ID, item.ID, int64(ingredientPosition), cache.units)
			if err != nil {
				return err
			}
			if _, err := queries.CreateRecipeIngredient(ctx, params); err != nil {
				return err
			}
		}
	}

	for sectionPosition, instructionSection := range content.InstructionSections {
		section, err := queries.CreateInstructionSection(ctx, store.CreateInstructionSectionParams{
			RecipeID: recipeID,
			Name:     instructionSection.Name,
			Position: int64(sectionPosition),
		})
		if err != nil {
			return err
		}
		for stepPosition, instruction := range instructionSection.Steps {
			if _, err := queries.CreateRecipeStep(ctx, store.CreateRecipeStepParams{
				SectionID:   section.ID,
				Position:    int64(stepPosition),
				Instruction: instruction,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

// verifyDraftRecipe moves a draft recipe through reviewable to verified. verifiedBy is the
// approving member's user ID, or empty for the bootstrap import.
func verifyDraftRecipe(ctx context.Context, queries *store.Queries, householdID string, recipeID int64, verifiedAt, verifiedBy string) (store.Recipe, error) {
	if _, err := queries.MarkRecipeReviewable(ctx, store.MarkRecipeReviewableParams{HouseholdID: householdID, ID: recipeID}); err != nil {
		return store.Recipe{}, err
	}
	return queries.VerifyRecipe(ctx, store.VerifyRecipeParams{
		HouseholdID: householdID, ID: recipeID,
		VerifiedAt: sql.NullString{String: verifiedAt, Valid: true},
		VerifiedBy: sql.NullString{String: verifiedBy, Valid: verifiedBy != ""},
	})
}

func ensureGroceryItem(
	ctx context.Context,
	queries *store.Queries,
	householdID string,
	wanted GroceryItem,
	sections map[string]store.StoreSection,
	items map[string]store.GroceryItem,
) (store.GroceryItem, error) {
	section, ok := sections[wanted.StoreSection.Key]
	if !ok {
		var err error
		section, err = queries.GetStoreSectionByKey(ctx, store.GetStoreSectionByKeyParams{HouseholdID: householdID, Key: wanted.StoreSection.Key})
		if errors.Is(err, sql.ErrNoRows) {
			section, err = queries.CreateBootstrapStoreSection(ctx, store.CreateBootstrapStoreSectionParams{
				HouseholdID: householdID, Key: wanted.StoreSection.Key, Name: wanted.StoreSection.Name,
			})
		}
		if err != nil {
			return store.GroceryItem{}, err
		}
		if section.Name != wanted.StoreSection.Name {
			return store.GroceryItem{}, fmt.Errorf("store section %q name is %q, document wants %q", section.Key, section.Name, wanted.StoreSection.Name)
		}
		sections[wanted.StoreSection.Key] = section
	}

	item, ok := items[wanted.Key]
	if !ok {
		var err error
		item, err = queries.GetGroceryItemByKey(ctx, store.GetGroceryItemByKeyParams{HouseholdID: householdID, Key: wanted.Key})
		if errors.Is(err, sql.ErrNoRows) {
			item, err = queries.CreateGroceryItem(ctx, store.CreateGroceryItemParams{
				HouseholdID: householdID, Key: wanted.Key, Name: wanted.Name, StoreSectionID: section.ID, ShoppingMode: wanted.ShoppingMode,
			})
		}
		if err != nil {
			return store.GroceryItem{}, err
		}
		items[wanted.Key] = item
	}
	if item.Name != wanted.Name || item.StoreSectionID != section.ID || item.ShoppingMode != wanted.ShoppingMode {
		return store.GroceryItem{}, fmt.Errorf("grocery item %q conflicts with an earlier approved definition", wanted.Key)
	}
	return item, nil
}

func ingredientParams(
	ctx context.Context,
	queries *store.Queries,
	householdID string,
	ingredient Ingredient,
	sectionID, itemID, position int64,
	units map[string]store.Unit,
) (store.CreateRecipeIngredientParams, error) {
	rationals, err := ingredient.Quantity.rationals()
	if err != nil {
		return store.CreateRecipeIngredientParams{}, err
	}
	params := store.CreateRecipeIngredientParams{
		HouseholdID:          householdID,
		SectionID:            sectionID,
		GroceryItemID:        sql.NullInt64{Int64: itemID, Valid: true},
		Position:             position,
		SourceText:           ingredient.SourceText,
		QuantityKind:         ingredient.Quantity.Kind,
		Preparation:          nullString(ingredient.Preparation),
		IsOptional:           boolInt(ingredient.Optional),
		IncludeOnGroceryList: boolInt(!ingredient.NonShopping),
		DisplayNote:          nullString(ingredient.Note),
	}
	if rationals.Minimum != nil {
		params.AmountMinNumerator, params.AmountMinDenominator, err = nullRational(rationals.Minimum)
		if err != nil {
			return params, err
		}
	}
	if rationals.Maximum != nil {
		params.AmountMaxNumerator, params.AmountMaxDenominator, err = nullRational(rationals.Maximum)
		if err != nil {
			return params, err
		}
	}
	if ingredient.Quantity.Unit != "" {
		unit, err := ensureUnit(ctx, queries, ingredient.Quantity.Unit, units)
		if err != nil {
			return params, err
		}
		params.UnitID = sql.NullInt64{Int64: unit.ID, Valid: true}
	}
	if ingredient.Quantity.Package != nil {
		params.PackageType = nullString(ingredient.Quantity.Package.Type)
		if rationals.PackageSize != nil {
			params.PackageSizeNumerator, params.PackageSizeDenominator, err = nullRational(rationals.PackageSize)
			if err != nil {
				return params, err
			}
			unit, err := ensureUnit(ctx, queries, ingredient.Quantity.Package.Unit, units)
			if err != nil {
				return params, err
			}
			params.PackageSizeUnitID = sql.NullInt64{Int64: unit.ID, Valid: true}
		}
	}
	return params, nil
}

func ensureUnit(ctx context.Context, queries *store.Queries, key string, units map[string]store.Unit) (store.Unit, error) {
	if unit, ok := units[key]; ok {
		return unit, nil
	}
	unit, err := queries.GetUnitByKey(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.Unit{}, fmt.Errorf("unknown exact unit %q", key)
		}
		return store.Unit{}, err
	}
	units[key] = unit
	return unit, nil
}

func nullRational(value *big.Rat) (sql.NullInt64, sql.NullInt64, error) {
	if !value.Num().IsInt64() || !value.Denom().IsInt64() {
		return sql.NullInt64{}, sql.NullInt64{}, fmt.Errorf("quantity %s exceeds 64-bit integer range", value.RatString())
	}
	return sql.NullInt64{Int64: value.Num().Int64(), Valid: true},
		sql.NullInt64{Int64: value.Denom().Int64(), Valid: true}, nil
}

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func durationMinimum(value *Duration) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: value.Min, Valid: true}
}

func durationMaximum(value *Duration) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: value.Max, Valid: true}
}

func boolInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}
