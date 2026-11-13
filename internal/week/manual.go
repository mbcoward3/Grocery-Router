package week

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"strings"

	"github.com/mbcoward3/grocery-router/internal/store"
)

// AddManualLine adds a week-only item, resolving known names deterministically.
func (service *Household) AddManualLine(ctx context.Context, name string, choices ...ManualLineOptions) (store.ShoppingLine, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return store.ShoppingLine{}, invalid("manual line name is required")
	}
	var result store.ShoppingLine
	err := service.transact(ctx, func(queries *store.Queries) error {
		weekRow, err := service.lockOpenWeek(ctx, queries)
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

// ManualLineOptions optionally identifies a known item and a household section.
type ManualLineOptions struct {
	GroceryItemID  int64
	StoreSectionID int64
}

func normalizeItem(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

// SetLineSection moves a line on the current plan's list to another section. For a known grocery
// item, generated or manual, it saves the household's placement of that item (D053), so every
// later list uses it, and moves all of the item's lines now. Choosing the reviewed section again
// clears the placement. For an unknown manual name it remembers the section for that name (D045).
func (service *Household) SetLineSection(ctx context.Context, lineID, sectionID int64) error {
	return service.transact(ctx, func(queries *store.Queries) error {
		if err := requireSection(ctx, queries, service.householdID, sectionID); err != nil {
			return err
		}
		// Without an open plan the line below is not found.
		if _, err := service.lockOpenWeek(ctx, queries); err != nil && !errors.Is(err, ErrNoCurrentWeek) {
			return err
		}
		line, err := queries.GetOpenShoppingLine(ctx, store.GetOpenShoppingLineParams{HouseholdID: service.householdID, LineID: lineID})
		if errors.Is(err, sql.ErrNoRows) {
			return invalid("shopping line not found or operation is not supported")
		}
		if err != nil {
			return err
		}
		if line.GroceryItemID.Valid {
			return service.placeItem(ctx, queries, line.GroceryItemID.Int64, sectionID)
		}
		if line.Origin != "manual" {
			return invalid("shopping line not found or operation is not supported")
		}
		if _, err := queries.SetManualLineSection(ctx, store.SetManualLineSectionParams{
			HouseholdID: service.householdID, LineID: lineID, StoreSectionID: sectionID,
		}); err != nil {
			return err
		}
		other, err := queries.GetOtherStoreSection(ctx, service.householdID)
		if err != nil {
			return err
		}
		return rememberSection(ctx, queries, service.householdID, line.DisplayName, sectionID, other.ID)
	})
}

// placeItem saves where the household's store keeps a known item and moves its current lines.
func (service *Household) placeItem(ctx context.Context, queries *store.Queries, itemID, sectionID int64) error {
	reviewed, err := queries.GetGroceryItemSection(ctx, store.GetGroceryItemSectionParams{HouseholdID: service.householdID, ID: itemID})
	if err != nil {
		return err
	}
	if sectionID == reviewed {
		err = queries.ForgetItemSectionOverride(ctx, store.ForgetItemSectionOverrideParams{HouseholdID: service.householdID, GroceryItemID: itemID})
	} else {
		err = queries.SetItemSection(ctx, store.SetItemSectionParams{HouseholdID: service.householdID, GroceryItemID: itemID, StoreSectionID: sectionID})
	}
	if err != nil {
		return err
	}
	return queries.SetItemLinesSection(ctx, store.SetItemLinesSectionParams{
		HouseholdID: service.householdID, GroceryItemID: sql.NullInt64{Int64: itemID, Valid: true}, StoreSectionID: sectionID,
	})
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
