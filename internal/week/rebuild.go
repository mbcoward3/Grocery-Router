package week

import (
	"context"
	"database/sql"
	"strings"

	"github.com/mbcoward3/grocery-router/internal/grocery"
	"github.com/mbcoward3/grocery-router/internal/store"
)

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
