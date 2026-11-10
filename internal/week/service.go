// Package week implements transactional current-week planning and grocery recomputation.
package week

import (
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mbcoward3/grocery-router/internal/settings"
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
// Construct it with invalid(), which returns the value form: callers match it with
// errors.As against a week.ValidationError value, so a *ValidationError would not match.
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

// CurrentSunday returns the calendar date, in now's own zone, of the Sunday beginning now's week.
// It works on the calendar date alone so a DST change at midnight cannot move it a day.
func CurrentSunday(now time.Time) string {
	year, month, day := now.Date()
	return time.Date(year, month, day-int(now.Weekday()), 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}

// SundayIn returns the Sunday beginning the week that contains now in location.
func SundayIn(now time.Time, location *time.Location) string {
	return CurrentSunday(now.In(location))
}

// nextBoundary returns the first instant strictly after after at which location's clock reads
// startDay at atMinutes past midnight. Days are counted on the calendar, so a daylight-saving
// change during the week still lands on local wall-clock time.
func nextBoundary(after time.Time, startDay time.Weekday, atMinutes int, location *time.Location) time.Time {
	local := after.In(location)
	year, month, day := local.Date()
	day += (int(startDay) - int(local.Weekday()) + 7) % 7
	for {
		boundary := wallClock(year, month, day, atMinutes, location)
		if boundary.After(after) {
			return boundary
		}
		day += 7
	}
}

// wallClock returns the instant at which location's clock reads minutes past midnight on the
// given date. For a time skipped by a daylight-saving gap, time.Date returns an instant before
// the gap, possibly on the previous day; this moves it to the matching instant after the gap,
// as PostgreSQL does, so a boundary never falls before its own day has begun.
func wallClock(year int, month time.Month, day, minutes int, location *time.Location) time.Time {
	instant := time.Date(year, month, day, 0, minutes, 0, 0, location)
	want := time.Date(year, month, day, 0, minutes, 0, 0, time.UTC)
	gotYear, gotMonth, gotDay := instant.Date()
	hour, minute, second := instant.Clock()
	if got := time.Date(gotYear, gotMonth, gotDay, hour, minute, second, 0, time.UTC); got.Before(want) {
		instant = instant.Add(want.Sub(got))
	}
	return instant
}

// location loads the household's time zone.
func (service *Household) location(ctx context.Context, queries *store.Queries) (*time.Location, error) {
	zone, err := queries.GetHouseholdTimezone(ctx, service.householdID)
	if errors.Is(err, sql.ErrNoRows) {
		// A household without a row has no weeks. UTC is the zone used before households had one.
		zone = "UTC"
	} else if err != nil {
		return nil, fmt.Errorf("load household time zone: %w", err)
	}
	location, err := settings.ValidateTimezone(zone)
	if err != nil {
		return nil, fmt.Errorf("household time zone: %w", err)
	}
	return location, nil
}

// closeExpired closes the household's open week when its automatic close time is at or before
// now. Every read or change of the current week or History runs it first; no background job
// closes weeks.
func (service *Household) closeExpired(ctx context.Context, queries *store.Queries, now time.Time) error {
	return queries.CloseExpiredWeek(ctx, store.CloseExpiredWeekParams{HouseholdID: service.householdID, Now: now})
}

// openWeek closes an expired week, then returns the household's open week.
func (service *Household) openWeek(ctx context.Context, queries *store.Queries, now time.Time) (store.Week, error) {
	if err := service.closeExpired(ctx, queries, now); err != nil {
		return store.Week{}, err
	}
	weekRow, err := queries.GetOpenWeek(ctx, service.householdID)
	if errors.Is(err, sql.ErrNoRows) {
		return store.Week{}, ErrNoCurrentWeek
	}
	return weekRow, err
}

// Current returns the explicitly generated current week.
func (service *Household) Current(ctx context.Context, now time.Time) (View, error) {
	queries := store.New(service.db)
	weekRow, err := service.openWeek(ctx, queries, now)
	if err != nil {
		return View{}, err
	}
	return loadView(ctx, queries, service.householdID, weekRow)
}

// PastWeeks returns closed weeks, newest first.
func (service *Household) PastWeeks(ctx context.Context, now time.Time) ([]store.ListPastWeeksRow, error) {
	queries := store.New(service.db)
	if err := service.closeExpired(ctx, queries, now); err != nil {
		return nil, err
	}
	return queries.ListPastWeeks(ctx, service.householdID)
}

// HistoricalView is the final recipe pool and grocery checklist retained for an earlier week.
type HistoricalView struct {
	View
	Checklist Checklist
}

// PastWeek returns a read-only earlier week with its retained checklist state.
func (service *Household) PastWeek(ctx context.Context, now time.Time, weekID int64) (HistoricalView, error) {
	queries := store.New(service.db)
	if err := service.closeExpired(ctx, queries, now); err != nil {
		return HistoricalView{}, err
	}
	weekRow, err := queries.GetPastWeek(ctx, store.GetPastWeekParams{HouseholdID: service.householdID, ID: weekID})
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
	checklist, err := loadChecklist(ctx, queries, service.householdID, weekRow)
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
	// A new week can collide with a week another request opened at the same moment, or with a
	// closed week that has the same start date. A second attempt tells them apart: it finds the
	// other request's open week and regenerates its pool, as if the requests had run in turn.
	view, err := service.generate(ctx, now, count)
	var conflict weekConflict
	if !errors.As(err, &conflict) {
		return view, err
	}
	view, err = service.generate(ctx, now, count)
	if errors.As(err, &conflict) && conflict.constraint == "weeks_household_start_key" {
		return View{}, invalid("This week already ended and is in History. You can plan the next week from %s.",
			conflict.nextStart.Format("Monday, January 2"))
	}
	return view, err
}

// uniqueViolation is PostgreSQL's SQLSTATE for a unique constraint violation.
const uniqueViolation = "23505"

// weekConflict is a new week rejected by a uniqueness constraint.
type weekConflict struct {
	constraint string
	// nextStart is when the household's next week begins, in its own zone.
	nextStart time.Time
	err       error
}

func (conflict weekConflict) Error() string { return conflict.err.Error() }
func (conflict weekConflict) Unwrap() error { return conflict.err }

func (service *Household) generate(ctx context.Context, now time.Time, count int) (View, error) {
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
		weekRow, err := service.getOrOpenWeek(ctx, queries, now)
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
		weekRow, err := service.openWeek(ctx, queries, now)
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

// getOrOpenWeek returns the open week, after closing an expired one, or opens a new week. A new
// week starts on the local date of the most recent Sunday midnight and closes itself at the next.
func (service *Household) getOrOpenWeek(ctx context.Context, queries *store.Queries, now time.Time) (store.Week, error) {
	householdID := service.householdID
	weekRow, err := service.openWeek(ctx, queries, now)
	if err == nil {
		if _, listErr := queries.GetShoppingListByWeek(ctx, store.GetShoppingListByWeekParams{HouseholdID: householdID, WeekID: weekRow.ID}); errors.Is(listErr, sql.ErrNoRows) {
			_, listErr = queries.CreateShoppingList(ctx, store.CreateShoppingListParams{HouseholdID: householdID, WeekID: weekRow.ID})
			return weekRow, listErr
		} else if listErr != nil {
			return store.Week{}, listErr
		}
		return weekRow, nil
	}
	if !errors.Is(err, ErrNoCurrentWeek) {
		return store.Week{}, err
	}
	location, err := service.location(ctx, queries)
	if err != nil {
		return store.Week{}, err
	}
	autoClose := nextBoundary(now, time.Sunday, 0, location)
	weekRow, err = queries.CreateWeek(ctx, store.CreateWeekParams{
		HouseholdID: householdID,
		StartsOn:    SundayIn(now, location),
		OpenedAt:    now,
		AutoCloseAt: sql.NullTime{Time: autoClose, Valid: true},
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return store.Week{}, weekConflict{constraint: pgErr.ConstraintName, nextStart: autoClose.In(location), err: err}
	}
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
