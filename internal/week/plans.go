package week

import (
	"context"
	"time"

	"github.com/mbcoward3/grocery-router/internal/store"
)

// localDate returns the calendar date of instant in location.
func localDate(instant time.Time, location *time.Location) string {
	return instant.In(location).Format(time.DateOnly)
}

// lockPlans takes the household row before a plan is opened, closed, or reopened, so those run
// one at a time and a household never has two open plans, and returns its time zone. Changes
// inside the open plan take only the plan row (lockOpenWeek) and reference the household with a
// key-share lock, which this does not block, so no lock cycle forms.
func (service *Household) lockPlans(ctx context.Context, queries *store.Queries) (*time.Location, error) {
	zone, err := queries.LockHouseholdPlans(ctx, service.householdID)
	if err != nil {
		return nil, err
	}
	return zoneLocation(zone)
}

// closeOpen closes the open plan, if any, at now, ending it on today's local date. It reports
// whether a plan was open.
func (service *Household) closeOpen(ctx context.Context, queries *store.Queries, now time.Time, location *time.Location) (bool, error) {
	closed, err := queries.CloseOpenWeek(ctx, store.CloseOpenWeekParams{
		HouseholdID: service.householdID, ClosedAt: now, EndsOn: localDate(now, location),
	})
	return closed > 0, err
}

// Close ends the household's open plan now and moves it to History with its meals and checklist
// as they are. It never starts the next plan; the next one exists once someone presses Generate
// (D051). With no open plan it returns ErrNoCurrentWeek.
func (service *Household) Close(ctx context.Context, now time.Time) error {
	return service.transact(ctx, func(queries *store.Queries) error {
		location, err := service.lockPlans(ctx, queries)
		if err != nil {
			return err
		}
		closed, err := service.closeOpen(ctx, queries, now, location)
		if err == nil && !closed {
			return ErrNoCurrentWeek
		}
		return err
	})
}

// Reopen makes a past plan the open one again, with its meals and checklist as they were, so the
// household can keep shopping or cooking from it. The plan that was open, if any, closes now and
// moves to History. A plan that is not in History returns ErrPastWeek.
func (service *Household) Reopen(ctx context.Context, now time.Time, weekID int64) (View, error) {
	var result View
	err := service.transact(ctx, func(queries *store.Queries) error {
		location, err := service.lockPlans(ctx, queries)
		if err != nil {
			return err
		}
		if _, err := queries.GetPastWeek(ctx, store.GetPastWeekParams{HouseholdID: service.householdID, ID: weekID}); err != nil {
			return pastWeekError(err)
		}
		if _, err := service.closeOpen(ctx, queries, now, location); err != nil {
			return err
		}
		if _, err := queries.ReopenWeek(ctx, store.ReopenWeekParams{HouseholdID: service.householdID, ID: weekID}); err != nil {
			return err
		}
		weekRow, err := queries.GetOpenWeek(ctx, service.householdID)
		if err != nil {
			return err
		}
		result, err = service.loadView(ctx, queries, weekRow)
		return err
	})
	return result, err
}
