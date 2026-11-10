// Package week implements transactional plan management and grocery recomputation. A plan is
// the meals and groceries between two shopping trips; it is stored as a week for history (D051).
package week

import (
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/mbcoward3/grocery-router/internal/settings"
	"github.com/mbcoward3/grocery-router/internal/store"
)

var (
	// ErrNoCurrentWeek means the household has no open plan.
	ErrNoCurrentWeek = errors.New("no current plan")
	// ErrOccurrence means a recipe occurrence does not belong to the current week.
	ErrOccurrence = errors.New("week recipe occurrence not found")
	// ErrPastWeek means the requested plan is not in History.
	ErrPastWeek = errors.New("past plan not found")
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

// View is a plan and its stable recipe occurrences. Week.EndsOn is NULL while the plan is open.
type View struct {
	Week    store.Week
	Recipes []store.ListWeekRecipesRow
}

// zoneLocation resolves a stored household time zone.
func zoneLocation(zone string) (*time.Location, error) {
	location, err := settings.ValidateTimezone(zone)
	if err != nil {
		return nil, fmt.Errorf("household time zone: %w", err)
	}
	return location, nil
}

// openWeek returns the household's open plan.
func (service *Household) openWeek(ctx context.Context, queries *store.Queries) (store.Week, error) {
	weekRow, err := queries.GetOpenWeek(ctx, service.householdID)
	if errors.Is(err, sql.ErrNoRows) {
		return store.Week{}, ErrNoCurrentWeek
	}
	return weekRow, err
}

// lockOpenWeek returns the household's open plan locked for a change until the transaction ends.
// Every change to the open plan or its list takes it first, so it never writes into a plan that a
// concurrent Close or Reopen has just closed: that either waits for the change, or closes first,
// and then this finds no open plan (or the newly open one).
func (service *Household) lockOpenWeek(ctx context.Context, queries *store.Queries) (store.Week, error) {
	weekRow, err := queries.LockOpenWeek(ctx, service.householdID)
	if errors.Is(err, sql.ErrNoRows) {
		return store.Week{}, ErrNoCurrentWeek
	}
	return weekRow, err
}

// Current returns the household's open plan.
func (service *Household) Current(ctx context.Context) (View, error) {
	queries := store.New(service.db)
	weekRow, err := service.openWeek(ctx, queries)
	if err != nil {
		return View{}, err
	}
	return service.loadView(ctx, queries, weekRow)
}

// PastWeeks returns closed plans, newest first. Each carries the ends_on stored when it closed.
func (service *Household) PastWeeks(ctx context.Context) ([]store.ListPastWeeksRow, error) {
	return store.New(service.db).ListPastWeeks(ctx, service.householdID)
}

// HistoricalView is the recipe pool and grocery checklist of a plan in History.
type HistoricalView struct {
	View
	Checklist Checklist
}

// PastWeek returns a plan in History with its retained checklist state.
func (service *Household) PastWeek(ctx context.Context, weekID int64) (HistoricalView, error) {
	queries := store.New(service.db)
	weekRow, err := queries.GetPastWeek(ctx, store.GetPastWeekParams{HouseholdID: service.householdID, ID: weekID})
	if err != nil {
		return HistoricalView{}, pastWeekError(err)
	}
	view, err := service.loadView(ctx, queries, weekRow)
	if err != nil {
		return HistoricalView{}, err
	}
	checklist, err := loadChecklist(ctx, queries, service.householdID, weekRow)
	if err != nil {
		return HistoricalView{}, err
	}
	return HistoricalView{View: view, Checklist: checklist}, nil
}

// pastWeekError reports a plan missing from History as ErrPastWeek.
func pastWeekError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPastWeek
	}
	return err
}

// Generate fills the open plan with count unique verified recipes, replacing its meals, or opens
// a new plan starting today when none is open.
func (service *Household) Generate(ctx context.Context, now time.Time, count int) (View, error) {
	if count <= 0 {
		return View{}, invalid("recipe count must be positive")
	}
	var result View
	err := service.transact(ctx, func(queries *store.Queries) error {
		location, err := service.lockPlans(ctx, queries)
		if err != nil {
			return err
		}
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
		weekRow, err := service.getOrOpenWeek(ctx, queries, now, location)
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
		result, err = service.loadView(ctx, queries, weekRow)
		return err
	})
	return result, err
}

// Add appends one specific verified recipe. Duplicate recipes are intentionally allowed.
func (service *Household) Add(ctx context.Context, recipeID int64) (View, error) {
	return service.mutateCurrent(ctx, func(queries *store.Queries, weekRow store.Week) error {
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
func (service *Household) Remove(ctx context.Context, occurrenceID int64) (View, error) {
	return service.mutateCurrent(ctx, func(queries *store.Queries, weekRow store.Week) error {
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
func (service *Household) Swap(ctx context.Context, occurrenceID, recipeID int64) (View, error) {
	return service.mutateCurrent(ctx, func(queries *store.Queries, weekRow store.Week) error {
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
func (service *Household) RandomSwap(ctx context.Context, occurrenceID int64) (View, error) {
	return service.mutateCurrent(ctx, func(queries *store.Queries, weekRow store.Week) error {
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
	mutation func(*store.Queries, store.Week) error,
) (View, error) {
	var result View
	err := service.transact(ctx, func(queries *store.Queries) error {
		weekRow, err := service.lockOpenWeek(ctx, queries)
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
		result, err = service.loadView(ctx, queries, weekRow)
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

// getOrOpenWeek returns the open plan, or opens a new one starting on today's local date. The
// caller holds lockPlans.
func (service *Household) getOrOpenWeek(ctx context.Context, queries *store.Queries, now time.Time, location *time.Location) (store.Week, error) {
	householdID := service.householdID
	weekRow, err := service.lockOpenWeek(ctx, queries)
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
	weekRow, err = queries.CreateWeek(ctx, store.CreateWeekParams{
		HouseholdID: householdID, StartsOn: localDate(now, location), OpenedAt: now,
	})
	if err != nil {
		return store.Week{}, err
	}
	_, err = queries.CreateShoppingList(ctx, store.CreateShoppingListParams{HouseholdID: householdID, WeekID: weekRow.ID})
	return weekRow, err
}

func (service *Household) loadView(ctx context.Context, queries *store.Queries, weekRow store.Week) (View, error) {
	recipes, err := queries.ListWeekRecipes(ctx, store.ListWeekRecipesParams{HouseholdID: service.householdID, WeekID: weekRow.ID})
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
