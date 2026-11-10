package week

import (
	"context"

	"github.com/mbcoward3/grocery-router/internal/store"
)

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
