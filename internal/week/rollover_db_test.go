package week_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/settings"
	"github.com/mbcoward3/grocery-router/internal/week"
)

func saveSettings(t *testing.T, service *week.Household, now time.Time, change settings.Change) {
	t.Helper()
	if _, err := service.SaveSettings(context.Background(), now, change); err != nil {
		t.Fatalf("SaveSettings(%+v): %v", change, err)
	}
}

func weekCloseTimes(t *testing.T, db *sql.DB, weekID int64) (autoCloseAt, closedAt sql.NullTime) {
	t.Helper()
	if err := db.QueryRow(`SELECT auto_close_at, closed_at FROM weeks WHERE id = $1`, weekID).Scan(&autoCloseAt, &closedAt); err != nil {
		t.Fatal(err)
	}
	return autoCloseAt, closedAt
}

func testNowUTC() time.Time {
	return time.Date(2026, time.August, 19, 12, 0, 0, 0, time.UTC)
}

// After a time-zone change the computed start date can belong to a week that already closed.
// Weeks may share a start date, so the household plans a new week instead of being turned away.
func TestWeeksMayShareAStartDate(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	const household = "c0a7a2d8-669b-4e47-91c1-4d9a32f339d5"
	closedWeek, err := service.Generate(ctx, time.Date(2026, time.August, 12, 12, 0, 0, 0, time.UTC), 2)
	if err != nil || closedWeek.Week.StartsOn != "2026-08-09" {
		t.Fatalf("UTC week = %#v, %v", closedWeek.Week, err)
	}
	if _, err := service.Current(ctx, time.Date(2026, time.August, 16, 0, 0, 0, 0, time.UTC)); !errors.Is(err, week.ErrNoCurrentWeek) {
		t.Fatalf("Sunday 00:00 UTC Current error = %v, want ErrNoCurrentWeek", err)
	}

	// Saturday evening in Chicago is still the week of 2026-08-09.
	chicago := "America/Chicago"
	saveSettings(t, service, time.Date(2026, time.August, 16, 0, 10, 0, 0, time.UTC), settings.Change{Timezone: &chicago})
	saturdayEvening := time.Date(2026, time.August, 16, 0, 30, 0, 0, time.UTC)
	next, err := service.Generate(ctx, saturdayEvening, 2)
	if err != nil || next.Week.ID == closedWeek.Week.ID || next.Week.StartsOn != "2026-08-09" ||
		!next.Week.AutoCloseAt.Time.Equal(time.Date(2026, time.August, 16, 5, 0, 0, 0, time.UTC)) {
		t.Fatalf("second week of 2026-08-09 = %#v, %v", next.Week, err)
	}
	var weeks int
	if err := db.QueryRow(`SELECT count(*) FROM weeks WHERE household_id = $1 AND starts_on = '2026-08-09'`, household).Scan(&weeks); err != nil || weeks != 2 {
		t.Fatalf("weeks starting 2026-08-09 = %d, %v; want 2", weeks, err)
	}
	past, err := service.PastWeeks(ctx, saturdayEvening)
	if err != nil || len(past) != 1 || past[0].ID != closedWeek.Week.ID {
		t.Fatalf("history = %#v, %v", past, err)
	}
}

func TestFridayEveningRollover(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	chicago, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatal(err)
	}
	zone, friday, evening := "America/Chicago", int(time.Friday), 18*60
	saveSettings(t, service, time.Date(2026, time.August, 23, 12, 0, 0, 0, chicago),
		settings.Change{Timezone: &zone, WeekStartDay: &friday, RolloverMinutes: &evening})

	monday := time.Date(2026, time.August, 24, 12, 0, 0, 0, chicago)
	generated, err := service.Generate(ctx, monday, 2)
	if err != nil {
		t.Fatal(err)
	}
	if generated.Week.StartsOn != "2026-08-21" || generated.EndsOn.String != "2026-08-28" ||
		!generated.Week.AutoCloseAt.Time.Equal(time.Date(2026, time.August, 28, 18, 0, 0, 0, chicago)) {
		t.Fatalf("generated week = %#v, ends %v", generated.Week, generated.EndsOn)
	}

	current, err := service.Current(ctx, time.Date(2026, time.August, 28, 17, 59, 0, 0, chicago))
	if err != nil || current.Week.ID != generated.Week.ID {
		t.Fatalf("Friday 17:59 current = %#v, %v", current.Week, err)
	}
	fridayEvening := time.Date(2026, time.August, 28, 18, 0, 0, 0, chicago)
	if _, err := service.Current(ctx, fridayEvening); !errors.Is(err, week.ErrNoCurrentWeek) {
		t.Fatalf("Friday 18:00 Current error = %v, want ErrNoCurrentWeek", err)
	}
	// History shows the week ending on the Friday it closed, not the day before.
	past, err := service.PastWeeks(ctx, fridayEvening)
	if err != nil || len(past) != 1 || past[0].ID != generated.Week.ID || past[0].EndsOn.String != "2026-08-28" ||
		!past[0].ClosedAt.Time.Equal(fridayEvening) {
		t.Fatalf("Friday 18:00 history = %#v, %v", past, err)
	}
	detail, err := service.PastWeek(ctx, fridayEvening, generated.Week.ID)
	if err != nil || detail.EndsOn.String != "2026-08-28" {
		t.Fatalf("closed week detail = ends %v, %v", detail.EndsOn, err)
	}

	// The next week starts at that rollover.
	next, err := service.Generate(ctx, fridayEvening.Add(time.Minute), 2)
	if err != nil || next.Week.StartsOn != "2026-08-28" {
		t.Fatalf("next week = %#v, %v", next.Week, err)
	}
}

func TestManualWeekStaysOpen(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	manual := settings.Manual
	generatedAt := testNowUTC()
	saveSettings(t, service, generatedAt, settings.Change{RolloverMode: &manual})

	generated, err := service.Generate(ctx, generatedAt, 2)
	if err != nil {
		t.Fatal(err)
	}
	// A manual week starts today and has no end.
	if generated.Week.StartsOn != "2026-08-19" || generated.Week.AutoCloseAt.Valid || generated.EndsOn.Valid {
		t.Fatalf("manual week = %#v, ends %v", generated.Week, generated.EndsOn)
	}
	threeWeeksLater := generatedAt.AddDate(0, 0, 21)
	current, err := service.Current(ctx, threeWeeksLater)
	if err != nil || current.Week.ID != generated.Week.ID {
		t.Fatalf("three weeks later current = %#v, %v", current.Week, err)
	}
	if past, err := service.PastWeeks(ctx, threeWeeksLater); err != nil || len(past) != 0 {
		t.Fatalf("three weeks later history = %#v, %v", past, err)
	}
}

func TestSwitchingModeMovesTheOpenWeek(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	generated, err := service.Generate(ctx, testNowUTC(), 2)
	if err != nil {
		t.Fatal(err)
	}
	oldBoundary := time.Date(2026, time.August, 23, 0, 0, 0, 0, time.UTC)
	if autoClose, _ := weekCloseTimes(t, db, generated.Week.ID); !autoClose.Time.Equal(oldBoundary) {
		t.Fatalf("auto_close_at = %v, want %s", autoClose, oldBoundary)
	}

	// Switching to manual keeps the week open past its old boundary.
	manual, automatic := settings.Manual, settings.Automatic
	saveSettings(t, service, time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC), settings.Change{RolloverMode: &manual})
	if autoClose, closed := weekCloseTimes(t, db, generated.Week.ID); autoClose.Valid || closed.Valid {
		t.Fatalf("after manual: auto_close_at = %v, closed_at = %v", autoClose, closed)
	}
	pastOldBoundary := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	current, err := service.Current(ctx, pastOldBoundary)
	if err != nil || current.Week.ID != generated.Week.ID || current.EndsOn.Valid {
		t.Fatalf("manual week past its old boundary = %#v, ends %v, %v", current.Week, current.EndsOn, err)
	}

	// Switching back sets the next boundary after now, never one already passed.
	saveSettings(t, service, pastOldBoundary, settings.Change{RolloverMode: &automatic})
	newBoundary := time.Date(2026, time.August, 30, 0, 0, 0, 0, time.UTC)
	if autoClose, closed := weekCloseTimes(t, db, generated.Week.ID); !autoClose.Time.Equal(newBoundary) || closed.Valid {
		t.Fatalf("after automatic: auto_close_at = %v, closed_at = %v; want %s", autoClose, closed, newBoundary)
	}
	current, err = service.Current(ctx, newBoundary.Add(-time.Minute))
	if err != nil || current.Week.ID != generated.Week.ID || current.EndsOn.String != "2026-08-29" {
		t.Fatalf("before the new boundary = %#v, ends %v, %v", current.Week, current.EndsOn, err)
	}

	// Changing the day or time moves the boundary too.
	saturday, morning := int(time.Saturday), 8*60
	saveSettings(t, service, pastOldBoundary, settings.Change{WeekStartDay: &saturday, RolloverMinutes: &morning})
	saturdayMorning := time.Date(2026, time.August, 29, 8, 0, 0, 0, time.UTC)
	if autoClose, _ := weekCloseTimes(t, db, generated.Week.ID); !autoClose.Time.Equal(saturdayMorning) {
		t.Fatalf("after Saturday 08:00: auto_close_at = %v, want %s", autoClose, saturdayMorning)
	}

	// A change that leaves the rollover alone leaves the close time alone.
	count := 9
	saveSettings(t, service, time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC), settings.Change{DefaultRecipeCount: &count})
	if autoClose, _ := weekCloseTimes(t, db, generated.Week.ID); !autoClose.Time.Equal(saturdayMorning) {
		t.Fatalf("after a recipe count change: auto_close_at = %v, want %s", autoClose, saturdayMorning)
	}
	if _, err := service.Current(ctx, saturdayMorning); !errors.Is(err, week.ErrNoCurrentWeek) {
		t.Fatalf("Saturday 08:00 Current error = %v, want ErrNoCurrentWeek", err)
	}
}

func TestZoneChangeMovesTheOpenWeek(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	generated, err := service.Generate(ctx, testNowUTC(), 2)
	if err != nil {
		t.Fatal(err)
	}

	// Sunday 00:00 in Chicago is 05:00 UTC: the week stays open past UTC midnight.
	chicago := "America/Chicago"
	saveSettings(t, service, time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC), settings.Change{Timezone: &chicago})
	chicagoMidnight := time.Date(2026, time.August, 23, 5, 0, 0, 0, time.UTC)
	if autoClose, closed := weekCloseTimes(t, db, generated.Week.ID); !autoClose.Time.Equal(chicagoMidnight) || closed.Valid {
		t.Fatalf("after Chicago: auto_close_at = %v, closed_at = %v; want %s", autoClose, closed, chicagoMidnight)
	}
	current, err := service.Current(ctx, time.Date(2026, time.August, 23, 2, 0, 0, 0, time.UTC))
	if err != nil || current.Week.ID != generated.Week.ID || current.EndsOn.String != "2026-08-22" {
		t.Fatalf("after UTC midnight = %#v, ends %v, %v", current.Week, current.EndsOn, err)
	}

	// In Tokyo it is already Sunday noon, past that zone's rollover. Saving never closes the
	// week: it closes at the next Tokyo Sunday midnight instead.
	tokyo := "Asia/Tokyo"
	saveSettings(t, service, time.Date(2026, time.August, 23, 3, 0, 0, 0, time.UTC), settings.Change{Timezone: &tokyo})
	tokyoMidnight := time.Date(2026, time.August, 29, 15, 0, 0, 0, time.UTC)
	if autoClose, closed := weekCloseTimes(t, db, generated.Week.ID); !autoClose.Time.Equal(tokyoMidnight) || closed.Valid {
		t.Fatalf("after Tokyo: auto_close_at = %v, closed_at = %v; want %s", autoClose, closed, tokyoMidnight)
	}
	if current, err := service.Current(ctx, chicagoMidnight); err != nil || current.Week.ID != generated.Week.ID {
		t.Fatalf("at the old Chicago boundary = %#v, %v", current.Week, err)
	}
	if _, err := service.Current(ctx, tokyoMidnight); !errors.Is(err, week.ErrNoCurrentWeek) {
		t.Fatalf("Tokyo Sunday 00:00 Current error = %v, want ErrNoCurrentWeek", err)
	}
}

func TestSaveSettingsRejectsBeforeStoring(t *testing.T) {
	service, db := testService(t)
	generated, err := service.Generate(context.Background(), testNowUTC(), 2)
	if err != nil {
		t.Fatal(err)
	}
	day, mode := 7, settings.Manual
	_, err = service.SaveSettings(context.Background(), testNowUTC(), settings.Change{WeekStartDay: &day, RolloverMode: &mode})
	var validation settings.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("SaveSettings error = %v, want a ValidationError", err)
	}
	if autoClose, _ := weekCloseTimes(t, db, generated.Week.ID); !autoClose.Valid {
		t.Fatal("a rejected change moved the open week")
	}
	var stored string
	if err := db.QueryRow(`SELECT rollover_mode FROM households WHERE id = 'c0a7a2d8-669b-4e47-91c1-4d9a32f339d5'`).Scan(&stored); err != nil || stored != settings.Automatic {
		t.Fatalf("rollover_mode = %q, %v", stored, err)
	}
}

// A week that has ended keeps the dates it closed with: a later zone change, even one that would
// put its last instant on another local date, does not re-cut History.
func TestHistoryKeepsEndDateAfterZoneChange(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	chicago := "America/Chicago"
	saveSettings(t, service, testNowUTC(), settings.Change{Timezone: &chicago})
	generated, err := service.Generate(ctx, testNowUTC(), 2)
	if err != nil {
		t.Fatal(err)
	}
	// Chicago's Sunday midnight is 05:00 UTC; the week's last local day is Saturday.
	closedAt := time.Date(2026, time.August, 23, 5, 0, 0, 0, time.UTC)
	past, err := service.PastWeeks(ctx, closedAt)
	if err != nil || len(past) != 1 || past[0].ID != generated.Week.ID || past[0].EndsOn.String != "2026-08-22" {
		t.Fatalf("history in Chicago = %#v, %v", past, err)
	}

	// In Tokyo the same closing instant, less a second, is Sunday afternoon.
	tokyo := "Asia/Tokyo"
	saveSettings(t, service, closedAt.Add(time.Hour), settings.Change{Timezone: &tokyo})
	past, err = service.PastWeeks(ctx, closedAt.Add(time.Hour))
	if err != nil || len(past) != 1 || past[0].EndsOn.String != "2026-08-22" {
		t.Fatalf("history after moving to Tokyo = %#v, %v", past, err)
	}
	detail, err := service.PastWeek(ctx, closedAt.Add(time.Hour), generated.Week.ID)
	if err != nil || detail.EndsOn.String != "2026-08-22" {
		t.Fatalf("closed week detail after moving to Tokyo = ends %v, %v", detail.EndsOn, err)
	}
}

// Saving settings after a week's close time has passed, before anyone read it, closes the week at
// that time as any read would; it is not kept open or moved by the new settings.
func TestSaveSettingsAfterCloseTimeClosesTheWeekFirst(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	generated, err := service.Generate(ctx, testNowUTC(), 2)
	if err != nil {
		t.Fatal(err)
	}
	oldBoundary := time.Date(2026, time.August, 23, 0, 0, 0, 0, time.UTC)
	later := oldBoundary.Add(36 * time.Hour)
	manual := settings.Manual
	saveSettings(t, service, later, settings.Change{RolloverMode: &manual})

	autoClose, closed := weekCloseTimes(t, db, generated.Week.ID)
	if !closed.Valid || !closed.Time.Equal(oldBoundary) || !autoClose.Time.Equal(oldBoundary) {
		t.Fatalf("week after a late save: auto_close_at = %v, closed_at = %v; want both %s", autoClose, closed, oldBoundary)
	}
	if _, err := service.Current(ctx, later); !errors.Is(err, week.ErrNoCurrentWeek) {
		t.Fatalf("Current error = %v, want ErrNoCurrentWeek", err)
	}
	past, err := service.PastWeeks(ctx, later)
	if err != nil || len(past) != 1 || past[0].ID != generated.Week.ID || past[0].EndsOn.String != "2026-08-22" {
		t.Fatalf("history = %#v, %v", past, err)
	}
}
