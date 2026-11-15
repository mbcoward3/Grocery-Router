package ingest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mbcoward3/grocery-router/internal/store"
)

// ErrRecipeNotFound means the household has no recipe with the requested ID.
var ErrRecipeNotFound = errors.New("recipe not found")

// ItemChoice picks the grocery item for one ingredient line: an existing household item by
// ID, or a new household item. Exactly one is set.
type ItemChoice struct {
	GroceryItemID int64
	New           *NewGroceryItem
}

// NewGroceryItem describes a household grocery item an edit creates. Its key is derived from
// the name with GroceryItemKey.
type NewGroceryItem struct {
	Name           string
	StoreSectionID int64
	ShoppingMode   string
}

// RecipeEdit is a household member's replacement for a recipe's content (RECIPE_LIBRARY_SPEC
// §3.3). Content's ingredient grocery items are ignored: Items chooses each one, and
// Items[s][i] belongs to Content.IngredientSections[s].Ingredients[i]. A zero Content.Source
// keeps the recipe's current primary source.
type RecipeEdit struct {
	Content    RecipeContent
	Items      [][]ItemChoice
	VerifiedBy string
	Now        time.Time
}

// ReplaceRecipe replaces a household recipe's content inside the caller's transaction. It
// validates everything before writing; invalid content returns a *ContentError and writes
// nothing. The recipe keeps its ID and key, returns to draft, has its sources, sections,
// steps, and review flags replaced, and is verified again with verified_at = edit.Now and
// verified_by = edit.VerifiedBy: a member's save is the household's approval. Lines of a past
// plan that came from a deleted ingredient keep their snapshots (#56). The caller rebuilds the
// current plan.
func ReplaceRecipe(ctx context.Context, queries *store.Queries, householdID string, recipeID int64, edit RecipeEdit) (store.Recipe, error) {
	recipe, err := queries.LockRecipe(ctx, store.LockRecipeParams{HouseholdID: householdID, ID: recipeID})
	if errors.Is(err, sql.ErrNoRows) {
		return store.Recipe{}, ErrRecipeNotFound
	}
	if err != nil {
		return store.Recipe{}, err
	}
	content := edit.Content
	if content.Source == (Source{}) {
		if content.Source, err = primarySource(ctx, queries, householdID, recipeID); err != nil {
			return store.Recipe{}, err
		}
	}
	content, newItems, err := resolveItems(ctx, queries, householdID, content, edit.Items)
	var problems validationErrors
	var contentErr *ContentError
	if errors.As(err, &contentErr) {
		problems = contentErr.Fields
	} else if err != nil {
		return store.Recipe{}, err
	}
	// Recipe names are unique within a household.
	taken, err := queries.RecipeNameTaken(ctx, store.RecipeNameTakenParams{HouseholdID: householdID, Name: content.Name, ID: recipeID})
	if err != nil {
		return store.Recipe{}, err
	}
	if taken {
		problems = append(validationErrors{{Path: "name", Message: fmt.Sprintf("another recipe is already called %q", content.Name)}}, problems...)
	}
	if err := problems.err(); err != nil {
		return store.Recipe{}, err
	}

	if recipe.Status != "draft" {
		if _, err := queries.ReturnRecipeToDraft(ctx, store.ReturnRecipeToDraftParams{HouseholdID: householdID, ID: recipeID}); err != nil {
			return store.Recipe{}, err
		}
	}
	if _, err := queries.UpdateDraftRecipeContent(ctx, store.UpdateDraftRecipeContentParams{
		HouseholdID:          householdID,
		ID:                   recipeID,
		Name:                 content.Name,
		ImageUrl:             nullString(content.ImageURL),
		YieldText:            nullString(content.Yield),
		HandsOnMinMinutes:    durationMinimum(content.HandsOn),
		HandsOnMaxMinutes:    durationMaximum(content.HandsOn),
		UnattendedMinMinutes: durationMinimum(content.Unattended),
		UnattendedMaxMinutes: durationMaximum(content.Unattended),
	}); err != nil {
		return store.Recipe{}, err
	}
	if err := queries.DeleteRecipeSources(ctx, recipeID); err != nil {
		return store.Recipe{}, err
	}
	if err := queries.DeleteRecipeIngredientSections(ctx, store.DeleteRecipeIngredientSectionsParams{HouseholdID: householdID, RecipeID: recipeID}); err != nil {
		return store.Recipe{}, err
	}
	if err := queries.DeleteRecipeInstructionSections(ctx, recipeID); err != nil {
		return store.Recipe{}, err
	}
	if err := queries.DeleteRecipeReviewFlags(ctx, recipeID); err != nil {
		return store.Recipe{}, err
	}

	cache := newContentCache()
	for _, item := range newItems {
		created, err := queries.CreateGroceryItem(ctx, store.CreateGroceryItemParams{
			HouseholdID: householdID, Key: item.Key, Name: item.Name,
			StoreSectionID: item.sectionID, ShoppingMode: item.ShoppingMode,
		})
		if err != nil {
			return store.Recipe{}, err
		}
		cache.items[created.Key] = created
	}
	if err := insertRecipeContent(ctx, queries, householdID, recipeID, content, cache); err != nil {
		return store.Recipe{}, err
	}
	return verifyDraftRecipe(ctx, queries, householdID, recipeID, edit.Now.UTC().Format("2006-01-02T15:04:05.000Z"), edit.VerifiedBy)
}

func primarySource(ctx context.Context, queries *store.Queries, householdID string, recipeID int64) (Source, error) {
	sources, err := queries.ListRecipeSources(ctx, store.ListRecipeSourcesParams{HouseholdID: householdID, RecipeID: recipeID})
	if err != nil {
		return Source{}, err
	}
	for _, source := range sources {
		if source.IsPrimary == 1 {
			return Source{
				Relationship: source.Relationship, Attribution: source.Attribution,
				URL: source.Url.String, CheckedOn: source.CheckedOn.String,
			}, nil
		}
	}
	return Source{}, nil
}

// newItem is a grocery item an edit will create, with its resolved section.
type newItem struct {
	GroceryItem
	sectionID int64
}

// resolveItems fills every ingredient's grocery item from its ItemChoice and validates the
// result. It reads but never writes. New items are returned once each, in first-use order.
func resolveItems(ctx context.Context, queries *store.Queries, householdID string, content RecipeContent, choices [][]ItemChoice) (RecipeContent, []newItem, error) {
	if len(choices) != len(content.IngredientSections) {
		return content, nil, fmt.Errorf("item choices cover %d sections, content has %d", len(choices), len(content.IngredientSections))
	}
	existing, err := queries.ListGroceryItems(ctx, householdID)
	if err != nil {
		return content, nil, err
	}
	existingNames := make(map[string]string, len(existing)*2)
	for _, item := range existing {
		existingNames[normalizeName(item.Name)] = item.Name
		existingNames[item.Key] = item.Name
	}

	var problems validationErrors
	failed := make(map[string]bool)
	var created []newItem
	seen := make(map[string]bool)
	sections := make([]IngredientSection, len(content.IngredientSections))
	for sectionIndex, section := range content.IngredientSections {
		if len(choices[sectionIndex]) != len(section.Ingredients) {
			return content, nil, fmt.Errorf("item choices for section %d cover %d lines, content has %d",
				sectionIndex, len(choices[sectionIndex]), len(section.Ingredients))
		}
		sections[sectionIndex] = IngredientSection{Name: section.Name, Ingredients: make([]Ingredient, len(section.Ingredients))}
		for ingredientIndex, ingredient := range section.Ingredients {
			location := fmt.Sprintf("ingredient_sections[%d].ingredients[%d]", sectionIndex, ingredientIndex)
			item, newOne, problem, err := resolveItem(ctx, queries, householdID, location, choices[sectionIndex][ingredientIndex], existingNames)
			if err != nil {
				return content, nil, err
			}
			if problem != nil {
				problems.add(problem)
				failed[location] = true
			} else if newOne != nil && !seen[newOne.Key] {
				seen[newOne.Key] = true
				created = append(created, *newOne)
			}
			ingredient.GroceryItem = item
			sections[sectionIndex].Ingredients[ingredientIndex] = ingredient
		}
	}
	content.IngredientSections = sections

	var contentErr *ContentError
	if err := ValidateRecipeContent(content); errors.As(err, &contentErr) {
		for _, field := range contentErr.Fields {
			if !failed[ingredientLocation(field.Path)] {
				problems = append(problems, field)
			}
		}
	} else if err != nil {
		return content, nil, err
	}
	return content, created, problems.err()
}

// resolveItem returns a line's grocery item, the new item it creates (if any), or a problem.
func resolveItem(
	ctx context.Context, queries *store.Queries, householdID, location string,
	choice ItemChoice, existingNames map[string]string,
) (GroceryItem, *newItem, *FieldError, error) {
	switch {
	case (choice.GroceryItemID != 0) == (choice.New != nil):
		return GroceryItem{}, nil, fieldError(location+".grocery_item", "choose a grocery item or describe a new one"), nil
	case choice.New == nil:
		row, err := queries.GetGroceryItemWithSection(ctx, store.GetGroceryItemWithSectionParams{HouseholdID: householdID, ID: choice.GroceryItemID})
		if errors.Is(err, sql.ErrNoRows) {
			return GroceryItem{}, nil, fieldError(location+".grocery_item_id", "grocery item not found"), nil
		}
		if err != nil {
			return GroceryItem{}, nil, nil, err
		}
		return GroceryItem{
			Key: row.Key, Name: row.Name, ShoppingMode: row.ShoppingMode,
			StoreSection: StoreSection{Key: row.StoreSectionKey, Name: row.StoreSectionName},
		}, nil, nil, nil
	}

	path := location + ".new_grocery_item"
	name := strings.Join(strings.Fields(choice.New.Name), " ")
	key := GroceryItemKey(name)
	switch {
	case name == "":
		return GroceryItem{}, nil, fieldError(path+".name", "new item name is required"), nil
	case key == "":
		return GroceryItem{}, nil, fieldError(path+".name", "new item name needs a letter or digit"), nil
	}
	if match, ok := existingNames[normalizeName(name)]; ok {
		return GroceryItem{}, nil, fieldError(path+".name", "%q already exists; pick the existing item", match), nil
	}
	if match, ok := existingNames[key]; ok {
		return GroceryItem{}, nil, fieldError(path+".name", "%q is too close to the existing item %q; pick it instead", name, match), nil
	}
	if choice.New.ShoppingMode != "measured" && choice.New.ShoppingMode != "counted" && choice.New.ShoppingMode != "presence-only" {
		return GroceryItem{}, nil, fieldError(path+".shopping_mode", "shopping mode must be measured, counted, or presence-only"), nil
	}
	section, err := queries.GetStoreSection(ctx, store.GetStoreSectionParams{HouseholdID: householdID, ID: choice.New.StoreSectionID})
	if errors.Is(err, sql.ErrNoRows) {
		return GroceryItem{}, nil, fieldError(path+".store_section_id", "store section not found"), nil
	}
	if err != nil {
		return GroceryItem{}, nil, nil, err
	}
	item := GroceryItem{
		Key: key, Name: name, ShoppingMode: choice.New.ShoppingMode,
		StoreSection: StoreSection{Key: section.Key, Name: section.Name},
	}
	return item, &newItem{GroceryItem: item, sectionID: section.ID}, nil, nil
}

// ingredientLocation trims a field path to its ingredient line, or returns "".
func ingredientLocation(path string) string {
	marker := ".ingredients["
	start := strings.Index(path, marker)
	if start < 0 {
		return ""
	}
	end := strings.Index(path[start+len(marker):], "]")
	if end < 0 {
		return ""
	}
	return path[:start+len(marker)+end+1]
}

func normalizeName(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

// GroceryItemKey derives a grocery item key from its name with the corpus key rule: lowercase
// ASCII letters and digits in hyphen-separated runs ("Wine & Beer" becomes "wine-and-beer").
// Apostrophes are dropped. It returns "" when the name has no letter or digit.
func GroceryItemKey(name string) string {
	var key strings.Builder
	pendingHyphen := false
	for _, r := range strings.ToLower(strings.ReplaceAll(name, "&", " and ")) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			if pendingHyphen && key.Len() > 0 {
				key.WriteByte('-')
			}
			pendingHyphen = false
			key.WriteRune(r)
		case r == '\'' || r == '’':
		default:
			pendingHyphen = true
		}
	}
	return key.String()
}
