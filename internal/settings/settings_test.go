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
	if arg.WeekStartDay.Valid {
		queries.stored.WeekStartDay = arg.WeekStartDay.Int16
	}
	if arg.RolloverMinutes.Valid {
		queries.stored.RolloverMinutes = arg.RolloverMinutes.Int16
	}
	if arg.RolloverMode.Valid {
		queries.stored.RolloverMode = arg.RolloverMode.String
	}
	return store.UpdateHouseholdSettingsRow(queries.stored), nil
}

func TestSettingsValidate(t *testing.T) {
	for _, valid := range []settings.Settings{
		{Timezone: "UTC", DefaultRecipeCount: 1, RolloverMode: settings.Automatic},
		{Timezone: "America/Chicago", DefaultRecipeCount: 21, WeekStartDay: 6, RolloverMinutes: 1439, RolloverMode: settings.Manual},
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
		{Timezone: "UTC", DefaultRecipeCount: 5, WeekStartDay: 7, RolloverMode: settings.Automatic},
		{Timezone: "UTC", DefaultRecipeCount: 5, RolloverMinutes: 1440, RolloverMode: settings.Automatic},
		{Timezone: "UTC", DefaultRecipeCount: 5, RolloverMode: ""},
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
	if params := queries.updates[0]; params.Timezone.Valid || params.WeekStartDay.Valid || params.RolloverMinutes.Valid ||
		params.RolloverMode.Valid || params.HouseholdID != "household" {
		t.Fatalf("count-only update params = %+v", params)
	}

	friday, evening, manual := 5, 18*60, settings.Manual
	updated, err = settings.Update(ctx, queries, "household", settings.Change{WeekStartDay: &friday, RolloverMinutes: &evening, RolloverMode: &manual})
	want := settings.Settings{Timezone: "UTC", DefaultRecipeCount: 7, WeekStartDay: 5, RolloverMinutes: 18 * 60, RolloverMode: settings.Manual}
	if err != nil || updated != want {
		t.Fatalf("Update rollover = %+v, %v; want %+v", updated, err, want)
	}
	if params := queries.updates[1]; params.Timezone.Valid || params.DefaultRecipeCount.Valid {
		t.Fatalf("rollover-only update params = %+v", params)
	}
}

func TestClock(t *testing.T) {
	for value, want := range map[string]int{"00:00": 0, "06:05": 365, "18:00": 1080, "23:59": 1439} {
		got, err := settings.ParseClock(value)
		if err != nil || got != want {
			t.Fatalf("ParseClock(%q) = %d, %v; want %d", value, got, err, want)
		}
		if formatted := settings.FormatClock(want); formatted != value {
			t.Fatalf("FormatClock(%d) = %q, want %q", want, formatted, value)
		}
	}
	for _, value := range []string{"", "24:00", "6:00", "18:60", "6pm", "18:00:00", " 18:00"} {
		var validation settings.ValidationError
		if _, err := settings.ParseClock(value); !errors.As(err, &validation) {
			t.Fatalf("ParseClock(%q) error = %v, want a ValidationError", value, err)
		}
	}
}

func TestSameRollover(t *testing.T) {
	base := settings.Settings{Timezone: "UTC", DefaultRecipeCount: 5, RolloverMode: settings.Automatic}
	count := base
	count.DefaultRecipeCount = 9
	if !base.SameRollover(count) {
		t.Fatal("a recipe count change moved the rollover")
	}
	for _, change := range []func(*settings.Settings){
		func(value *settings.Settings) { value.Timezone = "America/Chicago" },
		func(value *settings.Settings) { value.WeekStartDay = 5 },
		func(value *settings.Settings) { value.RolloverMinutes = 1 },
		func(value *settings.Settings) { value.RolloverMode = settings.Manual },
	} {
		other := base
		change(&other)
		if base.SameRollover(other) {
			t.Fatalf("SameRollover(%+v, %+v) = true", base, other)
		}
	}
}

// TestValidationMessages pins the messages web/src/features/settings/settings-page.tsx relies on:
// invalidField moves focus to the time zone when a message mentions "time zone", to the week
// start day for "week start day", to the rollover time for "rollover time", to the mode for
// "rollover mode", and to the recipe count otherwise. Change both together.
func TestValidationMessages(t *testing.T) {
	zone := func(name string) settings.Change { return settings.Change{Timezone: &name} }
	count := func(value int) settings.Change { return settings.Change{DefaultRecipeCount: &value} }
	for _, tc := range []struct {
		change settings.Change
		want   string
	}{
		{zone(""), "time zone must be an IANA zone name such as America/Chicago"},
		{zone("Local"), "time zone must be an IANA zone name such as America/Chicago"},
		{zone("Not/AZone"), `unknown time zone "Not/AZone"`},
		{count(0), "recipes per week must be between 1 and 21, not 0"},
		{count(22), "recipes per week must be between 1 and 21, not 22"},
		{settings.Change{WeekStartDay: ptr(7)}, "week start day must be between 0 (Sunday) and 6 (Saturday), not 7"},
		{settings.Change{RolloverMinutes: ptr(1440)}, "rollover time must be between 00:00 and 23:59, not 1440 minutes after midnight"},
		{settings.Change{RolloverMode: ptr("weekly")}, `rollover mode must be automatic or manual, not "weekly"`},
		{settings.Change{}, "provide at least one setting"},
	} {
		var validation settings.ValidationError
		if err := tc.change.Validate(); !errors.As(err, &validation) || validation.Message != tc.want {
			t.Fatalf("Validate(%+v) = %v, want ValidationError %q", tc.change, err, tc.want)
		}
	}
	var validation settings.ValidationError
	if _, err := settings.ParseClock("6pm"); !errors.As(err, &validation) ||
		validation.Message != `rollover time must be HH:MM on a 24-hour clock, not "6pm"` {
		t.Fatalf("ParseClock(6pm) = %v", err)
	}
}

func ptr[T any](value T) *T { return &value }
