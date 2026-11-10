package settings_test

import (
	"context"
	"errors"
	"testing"
	_ "time/tzdata"

	"github.com/mbcoward3/grocery-router/internal/settings"
	"github.com/mbcoward3/grocery-router/internal/store"
)

func TestValidateTimezone(t *testing.T) {
	for _, name := range []string{"America/Chicago", "UTC"} {
		location, err := settings.ValidateTimezone(name)
		if err != nil {
			t.Fatalf("ValidateTimezone(%q): %v", name, err)
		}
		if location.String() != name {
			t.Fatalf("ValidateTimezone(%q) = %q", name, location)
		}
	}
	for _, name := range []string{"", "Local", "Not/AZone"} {
		if _, err := settings.ValidateTimezone(name); err == nil {
			t.Fatalf("ValidateTimezone(%q) unexpectedly succeeded", name)
		}
	}
}

type fakeQueries struct {
	stored  store.GetHouseholdSettingsRow
	updates []store.UpdateHouseholdSettingsParams
}

func (queries *fakeQueries) GetHouseholdSettings(context.Context, string) (store.GetHouseholdSettingsRow, error) {
	return queries.stored, nil
}

func (queries *fakeQueries) UpdateHouseholdSettings(_ context.Context, arg store.UpdateHouseholdSettingsParams) (store.UpdateHouseholdSettingsRow, error) {
	queries.updates = append(queries.updates, arg)
	if arg.Timezone.Valid {
		queries.stored.Timezone = arg.Timezone.String
	}
	if arg.DefaultRecipeCount.Valid {
		queries.stored.DefaultRecipeCount = arg.DefaultRecipeCount.Int32
	}
	return store.UpdateHouseholdSettingsRow(queries.stored), nil
}

func TestSettingsValidate(t *testing.T) {
	for _, valid := range []settings.Settings{
		{Timezone: "UTC", DefaultRecipeCount: 1},
		{Timezone: "America/Chicago", DefaultRecipeCount: 21},
	} {
		if err := valid.Validate(); err != nil {
			t.Fatalf("Validate(%+v): %v", valid, err)
		}
	}
	for _, invalid := range []settings.Settings{
		{Timezone: "Not/AZone", DefaultRecipeCount: 5},
		{Timezone: "", DefaultRecipeCount: 5},
		{Timezone: "UTC", DefaultRecipeCount: 0},
		{Timezone: "UTC", DefaultRecipeCount: 22},
	} {
		var validation settings.ValidationError
		if err := invalid.Validate(); !errors.As(err, &validation) || validation.Message == "" {
			t.Fatalf("Validate(%+v) = %v, want a ValidationError", invalid, err)
		}
	}
}

func TestUpdateStoresOnlyValidNamedSettings(t *testing.T) {
	ctx := context.Background()
	queries := &fakeQueries{stored: store.GetHouseholdSettingsRow{Timezone: "UTC", DefaultRecipeCount: 5}}
	loaded, err := settings.Load(ctx, queries, "household")
	if err != nil || loaded != (settings.Settings{Timezone: "UTC", DefaultRecipeCount: 5}) {
		t.Fatalf("Load = %+v, %v", loaded, err)
	}

	badZone, zero, empty := "Not/AZone", 0, settings.Change{}
	for _, change := range []settings.Change{{Timezone: &badZone}, {DefaultRecipeCount: &zero}, empty} {
		var validation settings.ValidationError
		if _, err := settings.Update(ctx, queries, "household", change); !errors.As(err, &validation) {
			t.Fatalf("Update(%+v) error = %v, want a ValidationError", change, err)
		}
	}
	if len(queries.updates) != 0 {
		t.Fatalf("invalid changes reached the database: %+v", queries.updates)
	}

	count := 7
	updated, err := settings.Update(ctx, queries, "household", settings.Change{DefaultRecipeCount: &count})
	if err != nil || updated != (settings.Settings{Timezone: "UTC", DefaultRecipeCount: 7}) {
		t.Fatalf("Update count = %+v, %v", updated, err)
	}
	if params := queries.updates[0]; params.Timezone.Valid || params.HouseholdID != "household" {
		t.Fatalf("count-only update params = %+v", params)
	}
}
