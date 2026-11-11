package week

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/mbcoward3/grocery-router/internal/store"
)

// Checklist is the current week's store-section-ordered grocery list.
type Checklist struct {
	// StartsOn is the plan's start date.
	StartsOn    string
	List        store.ShoppingList
	Lines       []store.ListShoppingLinesRow
	RecipeNames map[int64][]string
	// Quantities holds each row's second and later amounts, in order.
	Quantities map[int64][]store.ListShoppingLineQuantitiesRow
	// Unspecified marks rows with a requirement that gave no amount ("to taste").
	Unspecified map[int64]bool
}

// Checklist returns the current week's compact grocery lines in store-section order.
func (service *Household) Checklist(ctx context.Context) (Checklist, error) {
	queries := store.New(service.db)
	weekRow, err := service.openWeek(ctx, queries)
	if err != nil {
		return Checklist{}, err
	}
	if err := service.upgradeList(ctx, weekRow.ID); err != nil {
		return Checklist{}, err
	}
	return loadChecklist(ctx, queries, service.householdID, weekRow)
}

func loadChecklist(ctx context.Context, queries *store.Queries, householdID string, weekRow store.Week) (Checklist, error) {
	list, err := queries.GetShoppingListByWeek(ctx, store.GetShoppingListByWeekParams{HouseholdID: householdID, WeekID: weekRow.ID})
	if err != nil {
		return Checklist{}, err
	}
	lines, err := queries.ListShoppingLines(ctx, store.ListShoppingLinesParams{HouseholdID: householdID, ShoppingListID: list.ID})
	if err != nil {
		return Checklist{}, err
	}
	associations, err := queries.ListShoppingLineRecipeNames(ctx, store.ListShoppingLineRecipeNamesParams{
		HouseholdID: householdID, ShoppingListID: list.ID,
	})
	if err != nil {
		return Checklist{}, err
	}
	recipeNames := make(map[int64][]string)
	for _, association := range associations {
		recipeNames[association.ShoppingLineID] = append(recipeNames[association.ShoppingLineID], association.RecipeName)
	}
	quantityRows, err := queries.ListShoppingLineQuantities(ctx, store.ListShoppingLineQuantitiesParams{HouseholdID: householdID, ShoppingListID: list.ID})
	if err != nil {
		return Checklist{}, err
	}
	quantities := make(map[int64][]store.ListShoppingLineQuantitiesRow)
	for _, row := range quantityRows {
		quantities[row.ShoppingLineID] = append(quantities[row.ShoppingLineID], row)
	}
	unspecifiedIDs, err := queries.ListShoppingLinesWithUnspecified(ctx, store.ListShoppingLinesWithUnspecifiedParams{HouseholdID: householdID, ShoppingListID: list.ID})
	if err != nil {
		return Checklist{}, err
	}
	unspecified := make(map[int64]bool, len(unspecifiedIDs))
	for _, id := range unspecifiedIDs {
		unspecified[id] = true
	}
	return Checklist{
		StartsOn: weekRow.StartsOn, List: list, Lines: lines, RecipeNames: recipeNames, Quantities: quantities, Unspecified: unspecified,
	}, nil
}

// Contributions returns the recipe traces behind a current generated line.
func (service *Household) Contributions(
	ctx context.Context,
	lineID int64,
) ([]store.ListShoppingLineContributionsRow, error) {
	checklist, err := service.Checklist(ctx)
	if err != nil {
		return nil, err
	}
	found := false
	for _, line := range checklist.Lines {
		if line.ID == lineID {
			found = true
			break
		}
	}
	if !found {
		return nil, invalid("shopping line not found")
	}
	return store.New(service.db).ListShoppingLineContributions(ctx, store.ListShoppingLineContributionsParams{HouseholdID: service.householdID, ShoppingLineID: lineID})
}

// SetLineRemoved removes or restores one current-week line without deleting its provenance.
func (service *Household) SetLineRemoved(ctx context.Context, lineID int64, removed bool) error {
	return service.updateLine(ctx, func(queries *store.Queries) (int64, error) {
		return queries.SetShoppingLineRemoved(ctx, store.SetShoppingLineRemovedParams{
			HouseholdID: service.householdID, Removed: boolInt(removed), LineID: lineID,
		})
	})
}

// SetLineCompleted updates one current-week checklist item's completion state.
func (service *Household) SetLineCompleted(ctx context.Context, lineID int64, completed bool) error {
	return service.updateLine(ctx, func(queries *store.Queries) (int64, error) {
		return queries.SetShoppingLineCompleted(ctx, store.SetShoppingLineCompletedParams{
			HouseholdID: service.householdID, Completed: boolInt(completed), LineID: lineID,
		})
	})
}

// SetLineOverride stores a week-only display override on a generated line.
func (service *Household) SetLineOverride(ctx context.Context, lineID int64, text string) error {
	text = strings.TrimSpace(text)
	return service.updateLine(ctx, func(queries *store.Queries) (int64, error) {
		return queries.SetShoppingLineOverride(ctx, store.SetShoppingLineOverrideParams{
			HouseholdID: service.householdID, OverrideText: sql.NullString{String: text, Valid: text != ""},
			LineID: lineID,
		})
	})
}

func (service *Household) updateLine(ctx context.Context, update func(*store.Queries) (int64, error)) error {
	return service.transact(ctx, func(queries *store.Queries) error {
		// Without an open week the update finds no line.
		if _, err := service.lockOpenWeek(ctx, queries); err != nil && !errors.Is(err, ErrNoCurrentWeek) {
			return err
		}
		rows, err := update(queries)
		if err != nil {
			return err
		}
		if rows != 1 {
			return invalid("shopping line not found or operation is not supported")
		}
		return nil
	})
}
