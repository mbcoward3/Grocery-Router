// Package week implements transactional current-week planning and grocery recomputation.
package week

import (
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/mbcoward3/grocery-router/internal/grocery"
	"github.com/mbcoward3/grocery-router/internal/store"
)

var (
	// ErrNoCurrentWeek means the household has not explicitly generated this week's pool.
	ErrNoCurrentWeek = errors.New("current week has not been generated")
	// ErrOccurrence means a recipe occurrence does not belong to the current week.
	ErrOccurrence = errors.New("week recipe occurrence not found")
)

// Picker supplies unbiased indexes and can be replaced by deterministic tests.
type Picker interface {
	IntN(n int) (int, error)
}

type securePicker struct{}

func (securePicker) IntN(n int) (int, error) {
	if n <= 0 {
		return 0, fmt.Errorf("random bound must be positive")
	}
	value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, err
	}
	return int(value.Int64()), nil
}

// Service owns week mutations. Every mutation and grocery rebuild uses one transaction.
type Service struct {
	db     *sql.DB
	picker Picker
}

// NewService constructs a transactional week service. A nil picker uses secure randomness.
func NewService(db *sql.DB, picker Picker) *Service {
	if picker == nil {
		picker = securePicker{}
	}
	return &Service{db: db, picker: picker}
}

// View is the unordered current week and its stable recipe occurrences.
type View struct {
	Week    store.Week
	Recipes []store.ListWeekRecipesRow
}

// CurrentSunday returns the local calendar date for the Sunday beginning now's week.
func CurrentSunday(now time.Time) string {
	start := now.AddDate(0, 0, -int(now.Weekday()))
	return start.Format("2006-01-02")
}

// Current returns the explicitly generated current week.
func (service *Service) Current(ctx context.Context, now time.Time) (View, error) {
	queries := store.New(service.db)
	weekRow, err := queries.GetWeekByStart(ctx, CurrentSunday(now))
	if errors.Is(err, sql.ErrNoRows) {
		return View{}, ErrNoCurrentWeek
	}
	if err != nil {
		return View{}, err
	}
	return loadView(ctx, queries, weekRow)
}

// Generate creates or fully regenerates the current week with unique verified recipes.
func (service *Service) Generate(ctx context.Context, now time.Time, count int) (View, error) {
	if count <= 0 {
		return View{}, fmt.Errorf("recipe count must be positive")
	}
	var result View
	err := service.transact(ctx, func(queries *store.Queries) error {
		verified, err := queries.ListVerifiedRecipes(ctx)
		if err != nil {
			return err
		}
		if count > len(verified) {
			return fmt.Errorf("requested %d recipes but only %d verified recipes are available", count, len(verified))
		}
		if err := shuffle(service.picker, verified); err != nil {
			return err
		}
		weekRow, err := getOrCreateWeek(ctx, queries, CurrentSunday(now))
		if err != nil {
			return err
		}
		if err := queries.DeleteWeekRecipes(ctx, weekRow.ID); err != nil {
			return err
		}
		for position := 0; position < count; position++ {
			if _, err := queries.CreateWeekRecipe(ctx, store.CreateWeekRecipeParams{
				WeekID: weekRow.ID, RecipeID: verified[position].ID, Position: int64(position),
			}); err != nil {
				return err
			}
		}
		if err := recompute(ctx, queries, weekRow.ID); err != nil {
			return err
		}
		result, err = loadView(ctx, queries, weekRow)
		return err
	})
	return result, err
}

// Add appends one specific verified recipe. Duplicate recipes are intentionally allowed.
func (service *Service) Add(ctx context.Context, now time.Time, recipeID int64) (View, error) {
	return service.mutateCurrent(ctx, now, func(queries *store.Queries, weekRow store.Week) error {
		position, err := queries.NextWeekRecipePosition(ctx, weekRow.ID)
		if err != nil {
			return err
		}
		_, err = queries.CreateWeekRecipe(ctx, store.CreateWeekRecipeParams{
			WeekID: weekRow.ID, RecipeID: recipeID, Position: position,
		})
		return err
	})
}

// Remove deletes one occurrence, not every occurrence of its recipe.
func (service *Service) Remove(ctx context.Context, now time.Time, occurrenceID int64) (View, error) {
	return service.mutateCurrent(ctx, now, func(queries *store.Queries, weekRow store.Week) error {
		occurrence, err := queries.GetWeekRecipe(ctx, occurrenceID)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && occurrence.WeekID != weekRow.ID) {
			return ErrOccurrence
		}
		if err != nil {
			return err
		}
		rows, err := queries.DeleteWeekRecipe(ctx, occurrenceID)
		if err != nil {
			return err
		}
		if rows != 1 {
			return ErrOccurrence
		}
		return nil
	})
}

// Swap replaces one occurrence with a specific verified recipe, including a duplicate.
func (service *Service) Swap(ctx context.Context, now time.Time, occurrenceID, recipeID int64) (View, error) {
	return service.mutateCurrent(ctx, now, func(queries *store.Queries, weekRow store.Week) error {
		occurrence, err := queries.GetWeekRecipe(ctx, occurrenceID)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && occurrence.WeekID != weekRow.ID) {
			return ErrOccurrence
		}
		if err != nil {
			return err
		}
		_, err = queries.UpdateWeekRecipeRecipe(ctx, store.UpdateWeekRecipeRecipeParams{
			RecipeID: recipeID, ID: occurrenceID,
		})
		return err
	})
}

// RandomSwap prefers a verified recipe absent from the pool, then falls back to any recipe
// different from the replaced occurrence.
func (service *Service) RandomSwap(ctx context.Context, now time.Time, occurrenceID int64) (View, error) {
	return service.mutateCurrent(ctx, now, func(queries *store.Queries, weekRow store.Week) error {
		occurrence, err := queries.GetWeekRecipe(ctx, occurrenceID)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && occurrence.WeekID != weekRow.ID) {
			return ErrOccurrence
		}
		if err != nil {
			return err
		}
		verified, err := queries.ListVerifiedRecipes(ctx)
		if err != nil {
			return err
		}
		selected, err := queries.ListWeekRecipeIDs(ctx, weekRow.ID)
		if err != nil {
			return err
		}
		inPool := make(map[int64]bool, len(selected))
		for _, id := range selected {
			inPool[id] = true
		}
		candidates := make([]int64, 0, len(verified))
		for _, recipe := range verified {
			if !inPool[recipe.ID] {
				candidates = append(candidates, recipe.ID)
			}
		}
		if len(candidates) == 0 {
			for _, recipe := range verified {
				if recipe.ID != occurrence.RecipeID {
					candidates = append(candidates, recipe.ID)
				}
			}
		}
		if len(candidates) == 0 {
			return fmt.Errorf("no replacement recipe is available")
		}
		index, err := service.picker.IntN(len(candidates))
		if err != nil {
			return err
		}
		if index < 0 || index >= len(candidates) {
			return fmt.Errorf("random picker returned out-of-range index %d", index)
		}
		replacement := candidates[index]
		_, err = queries.UpdateWeekRecipeRecipe(ctx, store.UpdateWeekRecipeRecipeParams{
			RecipeID: replacement, ID: occurrenceID,
		})
		return err
	})
}

func (service *Service) mutateCurrent(
	ctx context.Context,
	now time.Time,
	mutation func(*store.Queries, store.Week) error,
) (View, error) {
	var result View
	err := service.transact(ctx, func(queries *store.Queries) error {
		weekRow, err := queries.GetWeekByStart(ctx, CurrentSunday(now))
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoCurrentWeek
		}
		if err != nil {
			return err
		}
		if err := mutation(queries, weekRow); err != nil {
			return err
		}
		if err := recompute(ctx, queries, weekRow.ID); err != nil {
			return err
		}
		if err := queries.TouchWeek(ctx, weekRow.ID); err != nil {
			return err
		}
		result, err = loadView(ctx, queries, weekRow)
		return err
	})
	return result, err
}

func (service *Service) transact(ctx context.Context, operation func(*store.Queries) error) error {
	tx, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := operation(store.New(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

func getOrCreateWeek(ctx context.Context, queries *store.Queries, startsOn string) (store.Week, error) {
	weekRow, err := queries.GetWeekByStart(ctx, startsOn)
	if err == nil {
		if _, listErr := queries.GetShoppingListByWeek(ctx, weekRow.ID); errors.Is(listErr, sql.ErrNoRows) {
			_, listErr = queries.CreateShoppingList(ctx, weekRow.ID)
			return weekRow, listErr
		} else if listErr != nil {
			return store.Week{}, listErr
		}
		return weekRow, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return store.Week{}, err
	}
	weekRow, err = queries.CreateWeek(ctx, startsOn)
	if err != nil {
		return store.Week{}, err
	}
	_, err = queries.CreateShoppingList(ctx, weekRow.ID)
	return weekRow, err
}

func loadView(ctx context.Context, queries *store.Queries, weekRow store.Week) (View, error) {
	recipes, err := queries.ListWeekRecipes(ctx, weekRow.ID)
	if err != nil {
		return View{}, err
	}
	return View{Week: weekRow, Recipes: recipes}, nil
}

func shuffle[T any](picker Picker, values []T) error {
	for index := len(values) - 1; index > 0; index-- {
		selected, err := picker.IntN(index + 1)
		if err != nil {
			return err
		}
		if selected < 0 || selected > index {
			return fmt.Errorf("random picker returned out-of-range index %d", selected)
		}
		values[index], values[selected] = values[selected], values[index]
	}
	return nil
}

func recompute(ctx context.Context, queries *store.Queries, weekID int64) error {
	list, err := queries.GetShoppingListByWeek(ctx, weekID)
	if err != nil {
		return err
	}
	states, err := queries.ListGeneratedShoppingLineStates(ctx, list.ID)
	if err != nil {
		return err
	}
	stateByKey := make(map[string]store.ListGeneratedShoppingLineStatesRow, len(states))
	for _, state := range states {
		if state.AggregationKey.Valid {
			stateByKey[state.AggregationKey.String] = state
		}
	}
	requirementRows, err := queries.ListWeekIngredientRequirements(ctx, weekID)
	if err != nil {
		return err
	}
	requirements := make([]grocery.Requirement, 0, len(requirementRows))
	for _, row := range requirementRows {
		requirements = append(requirements, requirementFromRow(row))
	}
	lines, err := grocery.Aggregate(requirements)
	if err != nil {
		return err
	}
	if err := queries.DeleteGeneratedShoppingLines(ctx, list.ID); err != nil {
		return err
	}
	for position, line := range lines {
		state := stateByKey[line.Key]
		created, err := queries.CreateGeneratedShoppingLine(ctx, generatedLineParams(list.ID, int64(position), line, state))
		if err != nil {
			return err
		}
		for _, contribution := range line.Contributions {
			if _, err := queries.CreateShoppingLineContribution(ctx, contributionParams(created.ID, contribution)); err != nil {
				return err
			}
		}
	}
	return nil
}

func requirementFromRow(row store.ListWeekIngredientRequirementsRow) grocery.Requirement {
	requirement := grocery.Requirement{
		WeekRecipeID: row.WeekRecipeID, RecipeIngredientID: row.RecipeIngredientID,
		RecipeName: row.RecipeName, GroceryItemID: row.GroceryItemID,
		GroceryItemKey: row.GroceryItemKey, GroceryItemName: row.GroceryItemName,
		ShoppingMode: row.ShoppingMode, StoreSectionID: row.StoreSectionID,
		StoreSectionName: row.StoreSectionName, QuantityKind: row.QuantityKind,
		Minimum:  rational(row.AmountMinNumerator, row.AmountMinDenominator),
		Maximum:  rational(row.AmountMaxNumerator, row.AmountMaxDenominator),
		Optional: row.IsOptional == 1,
	}
	if row.UnitID.Valid {
		requirement.Unit = &grocery.Unit{
			ID: row.UnitID.Int64, Key: row.UnitKey.String, Dimension: row.UnitDimension.String,
			ToBaseNumerator:   row.UnitToBaseNumerator.Int64,
			ToBaseDenominator: row.UnitToBaseDenominator.Int64,
		}
	}
	if row.PackageType.Valid {
		requirement.Package = &grocery.Package{Type: row.PackageType.String}
		if row.PackageSizeNumerator.Valid {
			requirement.Package.Size = rational(row.PackageSizeNumerator, row.PackageSizeDenominator)
			requirement.Package.SizeUnit = &grocery.Unit{ID: row.PackageSizeUnitID.Int64, Key: row.PackageSizeUnitKey.String}
		}
	}
	return requirement
}

func generatedLineParams(
	listID, position int64,
	line grocery.Line,
	state store.ListGeneratedShoppingLineStatesRow,
) store.CreateGeneratedShoppingLineParams {
	params := store.CreateGeneratedShoppingLineParams{
		ShoppingListID: listID,
		GroceryItemID:  sql.NullInt64{Int64: line.GroceryItemID, Valid: true},
		StoreSectionID: line.StoreSectionID,
		AggregationKey: sql.NullString{String: line.Key, Valid: true},
		DisplayName:    line.GroceryItemName, QuantityKind: line.QuantityKind,
		IsOptional: boolInt(line.Optional), IsRemoved: state.IsRemoved,
		IsCompleted: state.IsCompleted, DisplayPosition: position, OverrideText: state.OverrideText,
	}
	params.AmountMinNumerator, params.AmountMinDenominator = nullRational(line.Minimum)
	params.AmountMaxNumerator, params.AmountMaxDenominator = nullRational(line.Maximum)
	if line.Unit != nil {
		params.UnitID = sql.NullInt64{Int64: line.Unit.ID, Valid: true}
	}
	if line.Package != nil {
		params.PackageType = sql.NullString{String: line.Package.Type, Valid: true}
		params.PackageSizeNumerator, params.PackageSizeDenominator = nullRational(line.Package.Size)
		if line.Package.SizeUnit != nil {
			params.PackageSizeUnitID = sql.NullInt64{Int64: line.Package.SizeUnit.ID, Valid: true}
		}
	}
	return params
}

func contributionParams(lineID int64, contribution grocery.Contribution) store.CreateShoppingLineContributionParams {
	params := store.CreateShoppingLineContributionParams{
		ShoppingLineID: lineID, WeekRecipeID: contribution.WeekRecipeID,
		RecipeIngredientID: contribution.RecipeIngredientID,
		QuantityKind:       contribution.QuantityKind, IsOptional: boolInt(contribution.Optional),
	}
	params.AmountMinNumerator, params.AmountMinDenominator = nullRational(contribution.Minimum)
	params.AmountMaxNumerator, params.AmountMaxDenominator = nullRational(contribution.Maximum)
	if contribution.Unit != nil {
		params.UnitID = sql.NullInt64{Int64: contribution.Unit.ID, Valid: true}
	}
	if contribution.Package != nil {
		params.PackageType = sql.NullString{String: contribution.Package.Type, Valid: true}
		params.PackageSizeNumerator, params.PackageSizeDenominator = nullRational(contribution.Package.Size)
		if contribution.Package.SizeUnit != nil {
			params.PackageSizeUnitID = sql.NullInt64{Int64: contribution.Package.SizeUnit.ID, Valid: true}
		}
	}
	return params
}

func rational(numerator, denominator sql.NullInt64) *grocery.Rational {
	if !numerator.Valid {
		return nil
	}
	return &grocery.Rational{Numerator: numerator.Int64, Denominator: denominator.Int64}
}

func nullRational(value *grocery.Rational) (sql.NullInt64, sql.NullInt64) {
	if value == nil {
		return sql.NullInt64{}, sql.NullInt64{}
	}
	return sql.NullInt64{Int64: value.Numerator, Valid: true},
		sql.NullInt64{Int64: value.Denominator, Valid: true}
}

func boolInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

// AddManualLine adds a week-only arbitrary item to Other.
func (service *Service) AddManualLine(ctx context.Context, now time.Time, name string) (store.ShoppingLine, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return store.ShoppingLine{}, fmt.Errorf("manual line name is required")
	}
	var result store.ShoppingLine
	err := service.transact(ctx, func(queries *store.Queries) error {
		weekRow, err := queries.GetWeekByStart(ctx, CurrentSunday(now))
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoCurrentWeek
		}
		if err != nil {
			return err
		}
		list, err := queries.GetShoppingListByWeek(ctx, weekRow.ID)
		if err != nil {
			return err
		}
		section, err := queries.GetOtherStoreSection(ctx)
		if err != nil {
			return err
		}
		position, err := queries.NextManualShoppingLinePosition(ctx, list.ID)
		if err != nil {
			return err
		}
		result, err = queries.CreateManualShoppingLine(ctx, store.CreateManualShoppingLineParams{
			ShoppingListID: list.ID, StoreSectionID: section.ID,
			DisplayName: name, DisplayPosition: position,
		})
		return err
	})
	return result, err
}

// Checklist is the current week's store-section-ordered grocery list.
type Checklist struct {
	List  store.ShoppingList
	Lines []store.ListShoppingLinesRow
}

// Checklist returns the current week's compact grocery lines in store-section order.
func (service *Service) Checklist(ctx context.Context, now time.Time) (Checklist, error) {
	queries := store.New(service.db)
	weekRow, err := queries.GetWeekByStart(ctx, CurrentSunday(now))
	if errors.Is(err, sql.ErrNoRows) {
		return Checklist{}, ErrNoCurrentWeek
	}
	if err != nil {
		return Checklist{}, err
	}
	list, err := queries.GetShoppingListByWeek(ctx, weekRow.ID)
	if err != nil {
		return Checklist{}, err
	}
	lines, err := queries.ListShoppingLines(ctx, list.ID)
	if err != nil {
		return Checklist{}, err
	}
	return Checklist{List: list, Lines: lines}, nil
}

// Contributions returns the recipe traces behind a current generated line.
func (service *Service) Contributions(
	ctx context.Context,
	now time.Time,
	lineID int64,
) ([]store.ListShoppingLineContributionsRow, error) {
	checklist, err := service.Checklist(ctx, now)
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
		return nil, fmt.Errorf("shopping line not found")
	}
	return store.New(service.db).ListShoppingLineContributions(ctx, lineID)
}

// SetLineRemoved removes or restores one current-week line without deleting its provenance.
func (service *Service) SetLineRemoved(ctx context.Context, now time.Time, lineID int64, removed bool) error {
	return service.updateLine(ctx, func(queries *store.Queries) (int64, error) {
		return queries.SetShoppingLineRemoved(ctx, store.SetShoppingLineRemovedParams{
			Removed: boolInt(removed), LineID: lineID, StartsOn: CurrentSunday(now),
		})
	})
}

// SetLineCompleted updates one current-week checklist item's completion state.
func (service *Service) SetLineCompleted(ctx context.Context, now time.Time, lineID int64, completed bool) error {
	return service.updateLine(ctx, func(queries *store.Queries) (int64, error) {
		return queries.SetShoppingLineCompleted(ctx, store.SetShoppingLineCompletedParams{
			Completed: boolInt(completed), LineID: lineID, StartsOn: CurrentSunday(now),
		})
	})
}

// SetLineOverride stores a week-only display override on a generated line.
func (service *Service) SetLineOverride(ctx context.Context, now time.Time, lineID int64, text string) error {
	text = strings.TrimSpace(text)
	return service.updateLine(ctx, func(queries *store.Queries) (int64, error) {
		return queries.SetShoppingLineOverride(ctx, store.SetShoppingLineOverrideParams{
			OverrideText: sql.NullString{String: text, Valid: text != ""},
			LineID:       lineID, StartsOn: CurrentSunday(now),
		})
	})
}

func (service *Service) updateLine(ctx context.Context, update func(*store.Queries) (int64, error)) error {
	return service.transact(ctx, func(queries *store.Queries) error {
		rows, err := update(queries)
		if err != nil {
			return err
		}
		if rows != 1 {
			return fmt.Errorf("shopping line not found or operation is not supported")
		}
		return nil
	})
}
