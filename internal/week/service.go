// Package week implements transactional current-week planning and grocery recomputation.
package week

import (
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"sort"
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
	// ErrPastWeek means the requested week is not an earlier generated week.
	ErrPastWeek = errors.New("past week not found")
)

// ValidationError is a user-facing rejection whose message is safe to return to clients.
type ValidationError struct {
	Message string
}

func (err ValidationError) Error() string {
	return err.Message
}

func invalid(format string, args ...any) error {
	return ValidationError{Message: fmt.Sprintf(format, args...)}
}

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

// Service provides household-scoped week operations.
type Service struct {
	db     *sql.DB
	picker Picker
}

// Household owns week mutations. Every mutation and grocery rebuild uses one transaction.
type Household struct {
	db          *sql.DB
	picker      Picker
	householdID string
}

// NewService constructs a week service. A nil picker uses secure randomness.
func NewService(db *sql.DB, picker Picker) *Service {
	if picker == nil {
		picker = securePicker{}
	}
	return &Service{db: db, picker: picker}
}

// ForHousehold scopes all week operations to the requested household.
func (service *Service) ForHousehold(householdID string) *Household {
	if householdID == "" {
		panic("week service requires an explicit household ID")
	}
	return &Household{db: service.db, picker: service.picker, householdID: householdID}
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
func (service *Household) Current(ctx context.Context, now time.Time) (View, error) {
	queries := store.New(service.db)
	weekRow, err := queries.GetWeekByStart(ctx, store.GetWeekByStartParams{HouseholdID: service.householdID, StartsOn: CurrentSunday(now)})
	if errors.Is(err, sql.ErrNoRows) {
		return View{}, ErrNoCurrentWeek
	}
	if err != nil {
		return View{}, err
	}
	return loadView(ctx, queries, service.householdID, weekRow)
}

// PastWeeks returns earlier generated weeks, newest first.
func (service *Household) PastWeeks(ctx context.Context, now time.Time) ([]store.ListPastWeeksRow, error) {
	return store.New(service.db).ListPastWeeks(ctx, store.ListPastWeeksParams{HouseholdID: service.householdID, StartsOn: CurrentSunday(now)})
}

// HistoricalView is the final recipe pool and grocery checklist retained for an earlier week.
type HistoricalView struct {
	View
	Checklist Checklist
}

// PastWeek returns a read-only earlier week with its retained checklist state.
func (service *Household) PastWeek(ctx context.Context, now time.Time, weekID int64) (HistoricalView, error) {
	queries := store.New(service.db)
	weekRow, err := queries.GetPastWeek(ctx, store.GetPastWeekParams{HouseholdID: service.householdID, ID: weekID, StartsOn: CurrentSunday(now)})
	if errors.Is(err, sql.ErrNoRows) {
		return HistoricalView{}, ErrPastWeek
	}
	if err != nil {
		return HistoricalView{}, err
	}
	view, err := loadView(ctx, queries, service.householdID, weekRow)
	if err != nil {
		return HistoricalView{}, err
	}
	checklist, err := loadChecklist(ctx, queries, service.householdID, weekRow.ID)
	if err != nil {
		return HistoricalView{}, err
	}
	return HistoricalView{View: view, Checklist: checklist}, nil
}

// Generate creates or fully regenerates the current week with unique verified recipes.
func (service *Household) Generate(ctx context.Context, now time.Time, count int) (View, error) {
	if count <= 0 {
		return View{}, invalid("recipe count must be positive")
	}
	var result View
	err := service.transact(ctx, func(queries *store.Queries) error {
		verified, err := queries.ListVerifiedRecipes(ctx, service.householdID)
		if err != nil {
			return err
		}
		if count > len(verified) {
			return invalid("requested %d recipes but only %d verified recipes are available", count, len(verified))
		}
		if err := shuffle(service.picker, verified); err != nil {
			return err
		}
		weekRow, err := getOrCreateWeek(ctx, queries, service.householdID, CurrentSunday(now))
		if err != nil {
			return err
		}
		if err := queries.DeleteWeekRecipes(ctx, store.DeleteWeekRecipesParams{HouseholdID: service.householdID, WeekID: weekRow.ID}); err != nil {
			return err
		}
		for position := 0; position < count; position++ {
			if _, err := queries.CreateWeekRecipe(ctx, store.CreateWeekRecipeParams{
				HouseholdID: service.householdID, WeekID: weekRow.ID, RecipeID: verified[position].ID, Position: int64(position),
			}); err != nil {
				return err
			}
		}
		if err := recompute(ctx, queries, service.householdID, weekRow.ID); err != nil {
			return err
		}
		result, err = loadView(ctx, queries, service.householdID, weekRow)
		return err
	})
	return result, err
}

// Add appends one specific verified recipe. Duplicate recipes are intentionally allowed.
func (service *Household) Add(ctx context.Context, now time.Time, recipeID int64) (View, error) {
	return service.mutateCurrent(ctx, now, func(queries *store.Queries, weekRow store.Week) error {
		if err := requireVerifiedRecipe(ctx, queries, service.householdID, recipeID); err != nil {
			return err
		}
		position, err := queries.NextWeekRecipePosition(ctx, store.NextWeekRecipePositionParams{HouseholdID: service.householdID, WeekID: weekRow.ID})
		if err != nil {
			return err
		}
		_, err = queries.CreateWeekRecipe(ctx, store.CreateWeekRecipeParams{
			HouseholdID: service.householdID, WeekID: weekRow.ID, RecipeID: recipeID, Position: position,
		})
		return err
	})
}

// Remove deletes one occurrence, not every occurrence of its recipe.
func (service *Household) Remove(ctx context.Context, now time.Time, occurrenceID int64) (View, error) {
	return service.mutateCurrent(ctx, now, func(queries *store.Queries, weekRow store.Week) error {
		occurrence, err := queries.GetWeekRecipe(ctx, store.GetWeekRecipeParams{HouseholdID: service.householdID, ID: occurrenceID})
		if errors.Is(err, sql.ErrNoRows) || (err == nil && occurrence.WeekID != weekRow.ID) {
			return ErrOccurrence
		}
		if err != nil {
			return err
		}
		rows, err := queries.DeleteWeekRecipe(ctx, store.DeleteWeekRecipeParams{HouseholdID: service.householdID, ID: occurrenceID})
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
func (service *Household) Swap(ctx context.Context, now time.Time, occurrenceID, recipeID int64) (View, error) {
	return service.mutateCurrent(ctx, now, func(queries *store.Queries, weekRow store.Week) error {
		occurrence, err := queries.GetWeekRecipe(ctx, store.GetWeekRecipeParams{HouseholdID: service.householdID, ID: occurrenceID})
		if errors.Is(err, sql.ErrNoRows) || (err == nil && occurrence.WeekID != weekRow.ID) {
			return ErrOccurrence
		}
		if err != nil {
			return err
		}
		if err := requireVerifiedRecipe(ctx, queries, service.householdID, recipeID); err != nil {
			return err
		}
		_, err = queries.UpdateWeekRecipeRecipe(ctx, store.UpdateWeekRecipeRecipeParams{
			HouseholdID: service.householdID, RecipeID: recipeID, ID: occurrenceID,
		})
		return err
	})
}

// RandomSwap prefers a verified recipe absent from the pool, then falls back to any recipe
// different from the replaced occurrence.
func (service *Household) RandomSwap(ctx context.Context, now time.Time, occurrenceID int64) (View, error) {
	return service.mutateCurrent(ctx, now, func(queries *store.Queries, weekRow store.Week) error {
		occurrence, err := queries.GetWeekRecipe(ctx, store.GetWeekRecipeParams{HouseholdID: service.householdID, ID: occurrenceID})
		if errors.Is(err, sql.ErrNoRows) || (err == nil && occurrence.WeekID != weekRow.ID) {
			return ErrOccurrence
		}
		if err != nil {
			return err
		}
		verified, err := queries.ListVerifiedRecipes(ctx, service.householdID)
		if err != nil {
			return err
		}
		selected, err := queries.ListWeekRecipeIDs(ctx, store.ListWeekRecipeIDsParams{HouseholdID: service.householdID, WeekID: weekRow.ID})
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
			return invalid("no replacement recipe is available")
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
			HouseholdID: service.householdID, RecipeID: replacement, ID: occurrenceID,
		})
		return err
	})
}

func requireVerifiedRecipe(ctx context.Context, queries *store.Queries, householdID string, recipeID int64) error {
	_, err := queries.GetVerifiedRecipe(ctx, store.GetVerifiedRecipeParams{HouseholdID: householdID, ID: recipeID})
	if errors.Is(err, sql.ErrNoRows) {
		return invalid("recipe not found")
	}
	return err
}

func (service *Household) mutateCurrent(
	ctx context.Context,
	now time.Time,
	mutation func(*store.Queries, store.Week) error,
) (View, error) {
	var result View
	err := service.transact(ctx, func(queries *store.Queries) error {
		weekRow, err := queries.GetWeekByStart(ctx, store.GetWeekByStartParams{HouseholdID: service.householdID, StartsOn: CurrentSunday(now)})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoCurrentWeek
		}
		if err != nil {
			return err
		}
		if err := mutation(queries, weekRow); err != nil {
			return err
		}
		if err := recompute(ctx, queries, service.householdID, weekRow.ID); err != nil {
			return err
		}
		if err := queries.TouchWeek(ctx, store.TouchWeekParams{HouseholdID: service.householdID, ID: weekRow.ID}); err != nil {
			return err
		}
		result, err = loadView(ctx, queries, service.householdID, weekRow)
		return err
	})
	return result, err
}

func (service *Household) transact(ctx context.Context, operation func(*store.Queries) error) error {
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

func getOrCreateWeek(ctx context.Context, queries *store.Queries, householdID, startsOn string) (store.Week, error) {
	weekRow, err := queries.GetWeekByStart(ctx, store.GetWeekByStartParams{HouseholdID: householdID, StartsOn: startsOn})
	if err == nil {
		if _, listErr := queries.GetShoppingListByWeek(ctx, store.GetShoppingListByWeekParams{HouseholdID: householdID, WeekID: weekRow.ID}); errors.Is(listErr, sql.ErrNoRows) {
			_, listErr = queries.CreateShoppingList(ctx, store.CreateShoppingListParams{HouseholdID: householdID, WeekID: weekRow.ID})
			return weekRow, listErr
		} else if listErr != nil {
			return store.Week{}, listErr
		}
		return weekRow, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return store.Week{}, err
	}
	weekRow, err = queries.CreateWeek(ctx, store.CreateWeekParams{HouseholdID: householdID, StartsOn: startsOn})
	if err != nil {
		return store.Week{}, err
	}
	_, err = queries.CreateShoppingList(ctx, store.CreateShoppingListParams{HouseholdID: householdID, WeekID: weekRow.ID})
	return weekRow, err
}

func loadView(ctx context.Context, queries *store.Queries, householdID string, weekRow store.Week) (View, error) {
	recipes, err := queries.ListWeekRecipes(ctx, store.ListWeekRecipesParams{HouseholdID: householdID, WeekID: weekRow.ID})
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

func recompute(ctx context.Context, queries *store.Queries, householdID string, weekID int64) error {
	list, err := queries.GetShoppingListByWeek(ctx, store.GetShoppingListByWeekParams{HouseholdID: householdID, WeekID: weekID})
	if err != nil {
		return err
	}
	states, err := queries.ListGeneratedShoppingLineStates(ctx, store.ListGeneratedShoppingLineStatesParams{HouseholdID: householdID, ShoppingListID: list.ID})
	if err != nil {
		return err
	}
	carried := carryStates(states)
	requirementRows, err := queries.ListWeekIngredientRequirements(ctx, store.ListWeekIngredientRequirementsParams{HouseholdID: householdID, WeekID: weekID})
	if err != nil {
		return err
	}
	requirements := make([]grocery.Requirement, 0, len(requirementRows))
	for _, row := range requirementRows {
		requirements = append(requirements, requirementFromRow(row))
	}
	aggregated, err := grocery.Aggregate(requirements)
	if err != nil {
		return err
	}
	lines := grocery.Group(aggregated)
	if err := queries.DeleteGeneratedShoppingLines(ctx, store.DeleteGeneratedShoppingLinesParams{HouseholdID: householdID, ShoppingListID: list.ID}); err != nil {
		return err
	}
	for position, line := range lines {
		created, err := queries.CreateGeneratedShoppingLine(ctx, generatedLineParams(householdID, list.ID, int64(position), line, carried(line)))
		if err != nil {
			return err
		}
		for index, quantity := range line.Quantities {
			if index == 0 {
				continue // The row's own columns hold its first quantity.
			}
			if err := queries.CreateShoppingLineQuantity(ctx, quantityParams(householdID, created.ID, int64(index), quantity)); err != nil {
				return err
			}
		}
		for _, contribution := range line.Contributions {
			if _, err := queries.CreateShoppingLineContribution(ctx, contributionParams(householdID, created.ID, contribution)); err != nil {
				return err
			}
		}
	}
	_, err = queries.SetShoppingListAggregationVersion(ctx, store.SetShoppingListAggregationVersionParams{
		HouseholdID: householdID, ID: list.ID, Version: aggregationVersion,
	})
	return err
}

// aggregationVersion 2 builds one checklist row per grocery item.
const aggregationVersion = 2

// carryStates keeps checked, removed, and adjusted state across rebuilds by row key. A row from
// before one-row-per-item (keys "item:<id>:...") passes its state to its item's new row: checked
// or removed only if every old row for the item was, and an adjustment only from a single row.
func carryStates(states []store.ListGeneratedShoppingLineStatesRow) func(grocery.ItemLine) store.ListGeneratedShoppingLineStatesRow {
	byKey := make(map[string]store.ListGeneratedShoppingLineStatesRow, len(states))
	legacy := make(map[string][]store.ListGeneratedShoppingLineStatesRow)
	for _, state := range states {
		if !state.AggregationKey.Valid {
			continue
		}
		key := state.AggregationKey.String
		byKey[key] = state
		if parts := strings.SplitN(key, ":", 3); len(parts) == 3 && parts[0] == "item" {
			itemKey := parts[0] + ":" + parts[1]
			legacy[itemKey] = append(legacy[itemKey], state)
		}
	}
	return func(line grocery.ItemLine) store.ListGeneratedShoppingLineStatesRow {
		if state, ok := byKey[line.Key]; ok {
			return state
		}
		old := legacy[line.Key]
		if len(old) == 0 {
			return store.ListGeneratedShoppingLineStatesRow{}
		}
		merged := store.ListGeneratedShoppingLineStatesRow{IsRemoved: 1, IsCompleted: 1}
		for _, state := range old {
			merged.IsRemoved &= state.IsRemoved
			merged.IsCompleted &= state.IsCompleted
		}
		if len(old) == 1 {
			merged.OverrideText = old[0].OverrideText
		}
		return merged
	}
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
	householdID string, listID, position int64,
	line grocery.ItemLine,
	state store.ListGeneratedShoppingLineStatesRow,
) store.CreateGeneratedShoppingLineParams {
	params := store.CreateGeneratedShoppingLineParams{
		HouseholdID: householdID, ShoppingListID: listID,
		GroceryItemID:  sql.NullInt64{Int64: line.GroceryItemID, Valid: true},
		StoreSectionID: line.StoreSectionID,
		AggregationKey: sql.NullString{String: line.Key, Valid: true},
		DisplayName:    line.GroceryItemName, QuantityKind: "unspecified",
		IsOptional: boolInt(line.Optional), IsRemoved: state.IsRemoved,
		IsCompleted: state.IsCompleted, DisplayPosition: position, OverrideText: state.OverrideText,
	}
	switch {
	case line.Presence:
		params.QuantityKind = "presence"
	case len(line.Quantities) > 0:
		first := line.Quantities[0]
		params.QuantityKind = first.Kind
		params.AmountMinNumerator, params.AmountMinDenominator = nullRational(first.Minimum)
		params.AmountMaxNumerator, params.AmountMaxDenominator = nullRational(first.Maximum)
		params.UnitID, params.PackageType, params.PackageSizeNumerator, params.PackageSizeDenominator, params.PackageSizeUnitID = unitAndPackage(first.Unit, first.Package)
	}
	return params
}

func quantityParams(householdID string, lineID, position int64, quantity grocery.Quantity) store.CreateShoppingLineQuantityParams {
	params := store.CreateShoppingLineQuantityParams{
		HouseholdID: householdID, ShoppingLineID: lineID, Position: position, QuantityKind: quantity.Kind,
		AmountMinNumerator: quantity.Minimum.Numerator, AmountMinDenominator: quantity.Minimum.Denominator,
	}
	params.AmountMaxNumerator, params.AmountMaxDenominator = nullRational(quantity.Maximum)
	params.UnitID, params.PackageType, params.PackageSizeNumerator, params.PackageSizeDenominator, params.PackageSizeUnitID = unitAndPackage(quantity.Unit, quantity.Package)
	return params
}

func unitAndPackage(unit *grocery.Unit, pkg *grocery.Package) (unitID sql.NullInt64, packageType sql.NullString, sizeNumerator, sizeDenominator, sizeUnitID sql.NullInt64) {
	if unit != nil {
		unitID = sql.NullInt64{Int64: unit.ID, Valid: true}
	}
	if pkg != nil {
		packageType = sql.NullString{String: pkg.Type, Valid: true}
		sizeNumerator, sizeDenominator = nullRational(pkg.Size)
		if pkg.SizeUnit != nil {
			sizeUnitID = sql.NullInt64{Int64: pkg.SizeUnit.ID, Valid: true}
		}
	}
	return unitID, packageType, sizeNumerator, sizeDenominator, sizeUnitID
}

func contributionParams(householdID string, lineID int64, contribution grocery.Contribution) store.CreateShoppingLineContributionParams {
	params := store.CreateShoppingLineContributionParams{
		HouseholdID: householdID, ShoppingLineID: lineID, WeekRecipeID: contribution.WeekRecipeID,
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

// AddManualLine adds a week-only item, resolving known names deterministically.
func (service *Household) AddManualLine(ctx context.Context, now time.Time, name string, choices ...ManualLineOptions) (store.ShoppingLine, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return store.ShoppingLine{}, invalid("manual line name is required")
	}
	var result store.ShoppingLine
	err := service.transact(ctx, func(queries *store.Queries) error {
		weekRow, err := queries.GetWeekByStart(ctx, store.GetWeekByStartParams{HouseholdID: service.householdID, StartsOn: CurrentSunday(now)})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoCurrentWeek
		}
		if err != nil {
			return err
		}
		list, err := queries.GetShoppingListByWeek(ctx, store.GetShoppingListByWeekParams{HouseholdID: service.householdID, WeekID: weekRow.ID})
		if err != nil {
			return err
		}
		section, err := queries.GetOtherStoreSection(ctx, service.householdID)
		if err != nil {
			return err
		}
		var itemID sql.NullInt64
		var options ManualLineOptions
		if len(choices) > 0 {
			options = choices[0]
		}
		if options.GroceryItemID < 0 || options.StoreSectionID < 0 {
			return invalid("item and section IDs must be positive")
		}
		items, err := queries.ListGroceryItems(ctx, service.householdID)
		if err != nil {
			return err
		}
		var matches []store.ListGroceryItemsRow
		for _, item := range items {
			if options.GroceryItemID != 0 {
				if item.ID == options.GroceryItemID {
					matches = append(matches, item)
				}
			} else if normalizeItem(name) == normalizeItem(item.Name) || normalizeItem(name) == normalizeItem(strings.ReplaceAll(item.Key, "-", " ")) {
				matches = append(matches, item)
			}
		}
		if options.GroceryItemID != 0 && len(matches) != 1 {
			return invalid("grocery item not found")
		}
		if len(matches) == 1 {
			itemID = sql.NullInt64{Int64: matches[0].ID, Valid: true}
			section.ID = matches[0].StoreSectionID
			name = matches[0].Name
		}
		switch {
		case options.StoreSectionID != 0:
			if err := requireSection(ctx, queries, service.householdID, options.StoreSectionID); err != nil {
				return err
			}
			if !itemID.Valid {
				if err := rememberSection(ctx, queries, service.householdID, name, options.StoreSectionID, section.ID); err != nil {
					return err
				}
			}
			section.ID = options.StoreSectionID
		case !itemID.Valid:
			// An unknown name reuses the section the household chose for it last time.
			remembered, err := queries.GetRememberedItemSection(ctx, store.GetRememberedItemSectionParams{HouseholdID: service.householdID, NameKey: normalizeItem(name)})
			if err == nil {
				section.ID = remembered
			} else if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}
		position, err := queries.NextManualShoppingLinePosition(ctx, store.NextManualShoppingLinePositionParams{HouseholdID: service.householdID, ShoppingListID: list.ID})
		if err != nil {
			return err
		}
		result, err = queries.CreateManualShoppingLine(ctx, store.CreateManualShoppingLineParams{
			HouseholdID: service.householdID, ShoppingListID: list.ID, StoreSectionID: section.ID,
			DisplayName: name, DisplayPosition: position, GroceryItemID: itemID,
		})
		return err
	})
	return result, err
}

// Checklist is the current week's store-section-ordered grocery list.
type Checklist struct {
	List        store.ShoppingList
	Lines       []store.ListShoppingLinesRow
	RecipeNames map[int64][]string
	// Quantities holds each row's second and later amounts, in order.
	Quantities map[int64][]store.ListShoppingLineQuantitiesRow
	// Unspecified marks rows with a requirement that gave no amount ("to taste").
	Unspecified map[int64]bool
}

// Checklist returns the current week's compact grocery lines in store-section order.
func (service *Household) Checklist(ctx context.Context, now time.Time) (Checklist, error) {
	queries := store.New(service.db)
	weekRow, err := queries.GetWeekByStart(ctx, store.GetWeekByStartParams{HouseholdID: service.householdID, StartsOn: CurrentSunday(now)})
	if errors.Is(err, sql.ErrNoRows) {
		return Checklist{}, ErrNoCurrentWeek
	}
	if err != nil {
		return Checklist{}, err
	}
	if err := service.upgradeList(ctx, weekRow.ID); err != nil {
		return Checklist{}, err
	}
	return loadChecklist(ctx, queries, service.householdID, weekRow.ID)
}

// upgradeList rebuilds a current week whose list predates one row per item. Past weeks are never
// rebuilt: history shows the list as it was. Only one concurrent reader performs the rebuild.
func (service *Household) upgradeList(ctx context.Context, weekID int64) error {
	list, err := store.New(service.db).GetShoppingListByWeek(ctx, store.GetShoppingListByWeekParams{HouseholdID: service.householdID, WeekID: weekID})
	if err != nil || list.AggregationVersion >= aggregationVersion {
		return err
	}
	return service.transact(ctx, func(queries *store.Queries) error {
		claimed, err := queries.SetShoppingListAggregationVersion(ctx, store.SetShoppingListAggregationVersionParams{
			HouseholdID: service.householdID, ID: list.ID, Version: aggregationVersion,
		})
		if err != nil || claimed == 0 {
			return err
		}
		return recompute(ctx, queries, service.householdID, weekID)
	})
}

func loadChecklist(ctx context.Context, queries *store.Queries, householdID string, weekID int64) (Checklist, error) {
	list, err := queries.GetShoppingListByWeek(ctx, store.GetShoppingListByWeekParams{HouseholdID: householdID, WeekID: weekID})
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
	return Checklist{List: list, Lines: lines, RecipeNames: recipeNames, Quantities: quantities, Unspecified: unspecified}, nil
}

// Contributions returns the recipe traces behind a current generated line.
func (service *Household) Contributions(
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
		return nil, invalid("shopping line not found")
	}
	return store.New(service.db).ListShoppingLineContributions(ctx, store.ListShoppingLineContributionsParams{HouseholdID: service.householdID, ShoppingLineID: lineID})
}

// SetLineRemoved removes or restores one current-week line without deleting its provenance.
func (service *Household) SetLineRemoved(ctx context.Context, now time.Time, lineID int64, removed bool) error {
	return service.updateLine(ctx, func(queries *store.Queries) (int64, error) {
		return queries.SetShoppingLineRemoved(ctx, store.SetShoppingLineRemovedParams{
			HouseholdID: service.householdID, Removed: boolInt(removed), LineID: lineID, StartsOn: CurrentSunday(now),
		})
	})
}

// SetLineCompleted updates one current-week checklist item's completion state.
func (service *Household) SetLineCompleted(ctx context.Context, now time.Time, lineID int64, completed bool) error {
	return service.updateLine(ctx, func(queries *store.Queries) (int64, error) {
		return queries.SetShoppingLineCompleted(ctx, store.SetShoppingLineCompletedParams{
			HouseholdID: service.householdID, Completed: boolInt(completed), LineID: lineID, StartsOn: CurrentSunday(now),
		})
	})
}

// SetLineOverride stores a week-only display override on a generated line.
func (service *Household) SetLineOverride(ctx context.Context, now time.Time, lineID int64, text string) error {
	text = strings.TrimSpace(text)
	return service.updateLine(ctx, func(queries *store.Queries) (int64, error) {
		return queries.SetShoppingLineOverride(ctx, store.SetShoppingLineOverrideParams{
			HouseholdID: service.householdID, OverrideText: sql.NullString{String: text, Valid: text != ""},
			LineID: lineID, StartsOn: CurrentSunday(now),
		})
	})
}

func (service *Household) updateLine(ctx context.Context, update func(*store.Queries) (int64, error)) error {
	return service.transact(ctx, func(queries *store.Queries) error {
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

// ManualLineOptions optionally identifies a known item and a household section.
type ManualLineOptions struct {
	GroceryItemID  int64
	StoreSectionID int64
}

func normalizeItem(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

// SetLineSection changes only a manual line in the current week.
func (service *Household) SetLineSection(ctx context.Context, now time.Time, lineID, sectionID int64) error {
	return service.transact(ctx, func(queries *store.Queries) error {
		if err := requireSection(ctx, queries, service.householdID, sectionID); err != nil {
			return err
		}
		line, err := queries.SetManualLineSection(ctx, store.SetManualLineSectionParams{
			HouseholdID: service.householdID, LineID: lineID, StoreSectionID: sectionID, StartsOn: CurrentSunday(now),
		})
		if errors.Is(err, sql.ErrNoRows) {
			return invalid("shopping line not found or operation is not supported")
		}
		if err != nil {
			return err
		}
		// Known grocery items keep their reviewed corpus section; only unknown names are remembered.
		if line.GroceryItemID.Valid {
			return nil
		}
		other, err := queries.GetOtherStoreSection(ctx, service.householdID)
		if err != nil {
			return err
		}
		return rememberSection(ctx, queries, service.householdID, line.DisplayName, sectionID, other.ID)
	})
}

func requireSection(ctx context.Context, queries *store.Queries, householdID string, sectionID int64) error {
	sections, err := queries.ListStoreSections(ctx, householdID)
	if err != nil {
		return err
	}
	for _, section := range sections {
		if section.ID == sectionID {
			return nil
		}
	}
	return invalid("store section not found")
}

// rememberSection saves the household's placement of an unknown item name. Placing it in
// Other is the default, so it clears the memory instead of storing it.
func rememberSection(ctx context.Context, queries *store.Queries, householdID, name string, sectionID, otherID int64) error {
	key := normalizeItem(name)
	if sectionID == otherID {
		return queries.ForgetItemSection(ctx, store.ForgetItemSectionParams{HouseholdID: householdID, NameKey: key})
	}
	return queries.RememberItemSection(ctx, store.RememberItemSectionParams{
		HouseholdID: householdID, NameKey: key, DisplayName: strings.TrimSpace(name), StoreSectionID: sectionID,
	})
}

// ItemMatch is one add-item suggestion: a known grocery item, or an unknown name the
// household has placed before (GroceryItemID is zero).
type ItemMatch struct {
	GroceryItemID int64
	Name          string
	SectionID     int64
	Section       string
}

// SearchItems returns up to twelve suggestions, best match first: exact, prefix, word
// prefix, then substring, alphabetical within each tier.
func (service *Household) SearchItems(ctx context.Context, term string) ([]ItemMatch, error) {
	term = normalizeItem(term)
	if term == "" {
		return []ItemMatch{}, nil
	}
	queries := store.New(service.db)
	items, err := queries.ListGroceryItems(ctx, service.householdID)
	if err != nil {
		return nil, err
	}
	remembered, err := queries.ListRememberedItemSections(ctx, service.householdID)
	if err != nil {
		return nil, err
	}
	type ranked struct {
		match ItemMatch
		rank  int
	}
	var found []ranked
	known := make(map[string]bool, len(items))
	for _, item := range items {
		known[normalizeItem(item.Name)] = true
		rank := matchRank(term, normalizeItem(item.Name))
		if keyRank := matchRank(term, strings.ReplaceAll(item.Key, "-", " ")); keyRank < rank {
			rank = keyRank
		}
		if rank < noMatch {
			found = append(found, ranked{ItemMatch{item.ID, item.Name, item.StoreSectionID, item.StoreSectionName}, rank})
		}
	}
	for _, entry := range remembered {
		if known[entry.NameKey] {
			continue
		}
		if rank := matchRank(term, entry.NameKey); rank < noMatch {
			found = append(found, ranked{ItemMatch{0, entry.DisplayName, entry.StoreSectionID, entry.StoreSectionName}, rank})
		}
	}
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].rank != found[j].rank {
			return found[i].rank < found[j].rank
		}
		return strings.ToLower(found[i].match.Name) < strings.ToLower(found[j].match.Name)
	})
	result := make([]ItemMatch, 0, min(len(found), 12))
	for _, entry := range found {
		if len(result) == 12 {
			break
		}
		result = append(result, entry.match)
	}
	return result, nil
}

const noMatch = 4

func matchRank(term, candidate string) int {
	switch {
	case candidate == term:
		return 0
	case strings.HasPrefix(candidate, term):
		return 1
	case strings.Contains(candidate, " "+term):
		return 2
	case strings.Contains(candidate, term):
		return 3
	default:
		return noMatch
	}
}

// SetSectionOrder validates and saves a complete household order, or resets the default.
func (service *Household) SetSectionOrder(ctx context.Context, ids []int64, reset bool) error {
	return service.transact(ctx, func(queries *store.Queries) error {
		if _, err := queries.LockSectionOrder(ctx, service.householdID); err != nil {
			return err
		}
		sections, err := queries.ListStoreSections(ctx, service.householdID)
		if err != nil {
			return err
		}
		if reset {
			if len(ids) != 0 {
				return invalid("reset cannot include section IDs")
			}
			if err := queries.ResetSectionPositions(ctx, service.householdID); err != nil {
				return err
			}
		} else {
			if len(ids) != len(sections) {
				return invalid("provide every household section exactly once; reload and try again")
			}
			allowed := make(map[int64]bool, len(sections))
			for _, section := range sections {
				allowed[section.ID] = true
			}
			for position, id := range ids {
				if !allowed[id] {
					return invalid("unknown or duplicate section")
				}
				delete(allowed, id)
				if err := queries.SetSectionPosition(ctx, store.SetSectionPositionParams{HouseholdID: service.householdID, ID: id, DisplayPosition: int64(position)}); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
