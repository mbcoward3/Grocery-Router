// Package settings loads, validates, and changes household-configurable settings.
package settings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/mbcoward3/grocery-router/internal/store"
)

// The bounds of a household's default recipe count; the database enforces the same range.
const (
	MinRecipeCount = 1
	MaxRecipeCount = 21
)

// The rollover modes. In automatic mode the open week closes itself at the household's rollover
// day and time; in manual mode it stays open until someone starts the next week.
const (
	Automatic = "automatic"
	Manual    = "manual"
)

// minutesPerDay bounds the rollover time; the database enforces the same range.
const minutesPerDay = 24 * 60

// Settings are the preferences a household edits in the app.
type Settings struct {
	Timezone           string
	DefaultRecipeCount int
	// WeekStartDay is the local day the week rolls over on, 0 (Sunday) to 6, as time.Weekday.
	WeekStartDay int
	// RolloverMinutes is the local time of day the week rolls over, in minutes after midnight.
	RolloverMinutes int
	RolloverMode    string
}

// SameRollover reports whether settings and other close an open week at the same moment: the
// same time zone, rollover day and time, and mode.
func (settings Settings) SameRollover(other Settings) bool {
	return settings.Timezone == other.Timezone && settings.WeekStartDay == other.WeekStartDay &&
		settings.RolloverMinutes == other.RolloverMinutes && settings.RolloverMode == other.RolloverMode
}

// Change names the settings to change; a nil field keeps the stored value.
type Change struct {
	Timezone           *string
	DefaultRecipeCount *int
	WeekStartDay       *int
	RolloverMinutes    *int
	RolloverMode       *string
}

// ValidationError is a user-facing rejection whose message is safe to return to clients.
type ValidationError struct {
	Message string
}

func (err ValidationError) Error() string {
	return err.Message
}

// Queries is the part of store.Queries that settings use, so validation is testable without
// PostgreSQL.
type Queries interface {
	GetHouseholdSettings(ctx context.Context, householdID string) (store.GetHouseholdSettingsRow, error)
	UpdateHouseholdSettings(ctx context.Context, arg store.UpdateHouseholdSettingsParams) (store.UpdateHouseholdSettingsRow, error)
}

// ValidateTimezone returns the IANA location for name. It rejects the empty string and "Local",
// which time.LoadLocation would otherwise resolve to UTC and the server's own zone.
func ValidateTimezone(name string) (*time.Location, error) {
	if name == "" || name == "Local" {
		return nil, errors.New("time zone must be an IANA zone name such as America/Chicago")
	}
	location, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("unknown time zone %q", name)
	}
	return location, nil
}

var clockPattern = regexp.MustCompile(`^([01][0-9]|2[0-3]):([0-5][0-9])$`)

// ParseClock reads a 24-hour "HH:MM" time of day as minutes after midnight. It reports a
// ValidationError for any other form.
func ParseClock(value string) (int, error) {
	parts := clockPattern.FindStringSubmatch(value)
	if parts == nil {
		return 0, ValidationError{Message: fmt.Sprintf("rollover time must be HH:MM on a 24-hour clock, not %q", value)}
	}
	hours, _ := strconv.Atoi(parts[1])
	minutes, _ := strconv.Atoi(parts[2])
	return hours*60 + minutes, nil
}

// FormatClock writes minutes after midnight as a 24-hour "HH:MM" time of day.
func FormatClock(minutes int) string {
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}

// Validate reports the first invalid setting as a ValidationError.
func (settings Settings) Validate() error {
	return Change{
		Timezone: &settings.Timezone, DefaultRecipeCount: &settings.DefaultRecipeCount,
		WeekStartDay: &settings.WeekStartDay, RolloverMinutes: &settings.RolloverMinutes,
		RolloverMode: &settings.RolloverMode,
	}.Validate()
}

// Validate checks only the settings the change names. It reports a ValidationError when the
// change names none or names an invalid value.
func (change Change) Validate() error {
	if change.Timezone == nil && change.DefaultRecipeCount == nil && change.WeekStartDay == nil &&
		change.RolloverMinutes == nil && change.RolloverMode == nil {
		return ValidationError{Message: "provide at least one setting"}
	}
	if change.Timezone != nil {
		if _, err := ValidateTimezone(*change.Timezone); err != nil {
			return ValidationError{Message: err.Error()}
		}
	}
	if change.DefaultRecipeCount != nil {
		count := *change.DefaultRecipeCount
		if count < MinRecipeCount || count > MaxRecipeCount {
			return ValidationError{Message: fmt.Sprintf("recipes per week must be between %d and %d, not %d",
				MinRecipeCount, MaxRecipeCount, count)}
		}
	}
	if change.WeekStartDay != nil {
		if day := *change.WeekStartDay; day < int(time.Sunday) || day > int(time.Saturday) {
			return ValidationError{Message: fmt.Sprintf("week start day must be between 0 (Sunday) and 6 (Saturday), not %d", day)}
		}
	}
	if change.RolloverMinutes != nil {
		if minutes := *change.RolloverMinutes; minutes < 0 || minutes >= minutesPerDay {
			return ValidationError{Message: fmt.Sprintf("rollover time must be between 00:00 and 23:59, not %d minutes after midnight", minutes)}
		}
	}
	if change.RolloverMode != nil {
		if mode := *change.RolloverMode; mode != Automatic && mode != Manual {
			return ValidationError{Message: fmt.Sprintf("rollover mode must be %s or %s, not %q", Automatic, Manual, mode)}
		}
	}
	return nil
}

// FromRow converts a stored settings row.
func FromRow(row store.GetHouseholdSettingsRow) Settings {
	return Settings{
		Timezone: row.Timezone, DefaultRecipeCount: int(row.DefaultRecipeCount),
		WeekStartDay: int(row.WeekStartDay), RolloverMinutes: int(row.RolloverMinutes), RolloverMode: row.RolloverMode,
	}
}

// Load returns the household's settings. A missing household is sql.ErrNoRows.
func Load(ctx context.Context, queries Queries, householdID string) (Settings, error) {
	row, err := queries.GetHouseholdSettings(ctx, householdID)
	if err != nil {
		return Settings{}, err
	}
	return FromRow(row), nil
}

// Update validates change, stores the settings it names, and returns the household's full
// settings. Nothing is stored when the change is invalid. A missing household is sql.ErrNoRows.
// It changes only the household row; week.Household.SaveSettings also moves the open week.
func Update(ctx context.Context, queries Queries, householdID string, change Change) (Settings, error) {
	if err := change.Validate(); err != nil {
		return Settings{}, err
	}
	params := store.UpdateHouseholdSettingsParams{HouseholdID: householdID}
	if change.Timezone != nil {
		params.Timezone = sql.NullString{String: *change.Timezone, Valid: true}
	}
	// Validate bounded every number below, so each conversion is exact.
	if change.DefaultRecipeCount != nil {
		params.DefaultRecipeCount = sql.NullInt32{Int32: int32(*change.DefaultRecipeCount), Valid: true}
	}
	if change.WeekStartDay != nil {
		params.WeekStartDay = sql.NullInt16{Int16: int16(*change.WeekStartDay), Valid: true}
	}
	if change.RolloverMinutes != nil {
		params.RolloverMinutes = sql.NullInt16{Int16: int16(*change.RolloverMinutes), Valid: true}
	}
	if change.RolloverMode != nil {
		params.RolloverMode = sql.NullString{String: *change.RolloverMode, Valid: true}
	}
	row, err := queries.UpdateHouseholdSettings(ctx, params)
	if err != nil {
		return Settings{}, err
	}
	return FromRow(store.GetHouseholdSettingsRow(row)), nil
}
