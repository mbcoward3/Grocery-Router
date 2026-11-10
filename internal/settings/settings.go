// Package settings loads, validates, and changes household-configurable settings.
package settings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/mbcoward3/grocery-router/internal/store"
)

// The bounds of a household's default recipe count; the database enforces the same range.
const (
	MinRecipeCount = 1
	MaxRecipeCount = 21
)

// Settings are the preferences a household edits in the app.
type Settings struct {
	Timezone           string
	DefaultRecipeCount int
}

// Change names the settings to change; a nil field keeps the stored value.
type Change struct {
	Timezone           *string
	DefaultRecipeCount *int
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

// Validate reports the first invalid setting as a ValidationError.
func (settings Settings) Validate() error {
	return Change{Timezone: &settings.Timezone, DefaultRecipeCount: &settings.DefaultRecipeCount}.Validate()
}

// Validate checks only the settings the change names. It reports a ValidationError when the
// change names none or names an invalid value.
func (change Change) Validate() error {
	if change.Timezone == nil && change.DefaultRecipeCount == nil {
		return ValidationError{Message: "provide timezone, defaultRecipeCount, or both"}
	}
	if change.Timezone != nil {
		if _, err := ValidateTimezone(*change.Timezone); err != nil {
			return ValidationError{Message: err.Error()}
		}
	}
	if change.DefaultRecipeCount != nil {
		count := *change.DefaultRecipeCount
		if count < MinRecipeCount || count > MaxRecipeCount {
			return ValidationError{Message: fmt.Sprintf("recipes per plan must be between %d and %d, not %d",
				MinRecipeCount, MaxRecipeCount, count)}
		}
	}
	return nil
}

// Load returns the household's settings. A missing household is sql.ErrNoRows.
func Load(ctx context.Context, queries Queries, householdID string) (Settings, error) {
	row, err := queries.GetHouseholdSettings(ctx, householdID)
	if err != nil {
		return Settings{}, err
	}
	return Settings{Timezone: row.Timezone, DefaultRecipeCount: int(row.DefaultRecipeCount)}, nil
}

// Update validates change, stores the settings it names, and returns the household's full
// settings. Nothing is stored when the change is invalid. A missing household is sql.ErrNoRows.
func Update(ctx context.Context, queries Queries, householdID string, change Change) (Settings, error) {
	if err := change.Validate(); err != nil {
		return Settings{}, err
	}
	params := store.UpdateHouseholdSettingsParams{HouseholdID: householdID}
	if change.Timezone != nil {
		params.Timezone = sql.NullString{String: *change.Timezone, Valid: true}
	}
	if change.DefaultRecipeCount != nil {
		// Validate bounded the count to 1-21, so the conversion is exact.
		params.DefaultRecipeCount = sql.NullInt32{Int32: int32(*change.DefaultRecipeCount), Valid: true}
	}
	row, err := queries.UpdateHouseholdSettings(ctx, params)
	if err != nil {
		return Settings{}, err
	}
	return Settings{Timezone: row.Timezone, DefaultRecipeCount: int(row.DefaultRecipeCount)}, nil
}
