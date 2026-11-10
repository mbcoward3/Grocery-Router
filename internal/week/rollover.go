package week

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/mbcoward3/grocery-router/internal/settings"
	"github.com/mbcoward3/grocery-router/internal/store"
)

// rules are a household's rollover settings, resolved for computing week boundaries.
type rules struct {
	location *time.Location
	startDay time.Weekday
	minutes  int
	manual   bool
}

// defaultRules are the rollover settings a household starts with: Sunday 12:00 AM UTC,
// automatic.
var defaultRules = rules{location: time.UTC, startDay: time.Sunday}

func rulesFrom(zone string, startDay, minutes int, mode string) (rules, error) {
	location, err := settings.ValidateTimezone(zone)
	if err != nil {
		return rules{}, fmt.Errorf("household time zone: %w", err)
	}
	return rules{location: location, startDay: time.Weekday(startDay), minutes: minutes, manual: mode == settings.Manual}, nil
}

// opening returns the start date and automatic close time of a week opened at now. An automatic
// week starts on the local date of the most recent rollover at or before now and closes at the
// next one; a manual week starts today and has no close time.
func (rules rules) opening(now time.Time) (string, sql.NullTime) {
	if rules.manual {
		return localDate(now, rules.location), sql.NullTime{}
	}
	return previousBoundaryDate(now, rules.startDay, rules.minutes, rules.location), rules.autoClose(now)
}

// autoClose returns when a week open at now closes itself: the next rollover strictly after now,
// so it is never already due, or NULL in manual mode.
func (rules rules) autoClose(now time.Time) sql.NullTime {
	if rules.manual {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: nextBoundary(now, rules.startDay, rules.minutes, rules.location), Valid: true}
}

// previousBoundaryDate returns the local calendar date of the most recent instant at or before
// now at which location's clock reads startDay at atMinutes past midnight.
func previousBoundaryDate(now time.Time, startDay time.Weekday, atMinutes int, location *time.Location) string {
	local := now.In(location)
	year, month, day := local.Date()
	day -= (int(local.Weekday()) - int(startDay) + 7) % 7
	if wallClock(year, month, day, atMinutes, location).After(now) {
		day -= 7
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Format(time.DateOnly)
}

// localDate returns the calendar date of instant in location.
func localDate(instant time.Time, location *time.Location) string {
	return instant.In(location).Format(time.DateOnly)
}

// endsOn returns the last local date of a week: the date of the instant one second before it
// closed, or before it closes itself while open. A week that rolls over at Sunday 12:00 AM ends on
// Saturday; one that rolls over at Friday 6:00 PM ends on Friday. An open manual week has no end.
func endsOn(closedAt, autoCloseAt sql.NullTime, location *time.Location) sql.NullString {
	end := closedAt
	if !end.Valid {
		end = autoCloseAt
	}
	if !end.Valid {
		return sql.NullString{}
	}
	return sql.NullString{String: localDate(end.Time.Add(-time.Second), location), Valid: true}
}

// weekRules loads the household's rollover settings and holds them until the transaction ends,
// so a settings change waits for a week being opened and then moves its close time.
func (service *Household) weekRules(ctx context.Context, queries *store.Queries) (rules, error) {
	row, err := queries.GetWeekRules(ctx, service.householdID)
	if errors.Is(err, sql.ErrNoRows) {
		// A household without a row has no weeks; opening one fails on its foreign key.
		return defaultRules, nil
	}
	if err != nil {
		return rules{}, fmt.Errorf("load household rollover settings: %w", err)
	}
	return rulesFrom(row.Timezone, int(row.WeekStartDay), int(row.RolloverMinutes), row.RolloverMode)
}

// SaveSettings validates and stores a settings change. When it changes the time zone, rollover
// day, time, or mode, the open week's automatic close time moves in the same transaction: to
// the next rollover after now under the new settings, or to none in manual mode. Saving never
// closes a week; a week whose close time had already passed closes first, as on any read.
func (service *Household) SaveSettings(ctx context.Context, now time.Time, change settings.Change) (settings.Settings, error) {
	if err := change.Validate(); err != nil {
		return settings.Settings{}, err
	}
	var saved settings.Settings
	err := service.transact(ctx, func(queries *store.Queries) error {
		row, err := queries.LockHouseholdSettings(ctx, service.householdID)
		if err != nil {
			return err
		}
		before := settings.FromRow(store.GetHouseholdSettingsRow(row))
		if err := service.closeExpired(ctx, queries, now); err != nil {
			return err
		}
		saved, err = settings.Update(ctx, queries, service.householdID, change)
		if err != nil {
			return err
		}
		if saved.SameRollover(before) {
			return nil
		}
		after, err := rulesFrom(saved.Timezone, saved.WeekStartDay, saved.RolloverMinutes, saved.RolloverMode)
		if err != nil {
			return err
		}
		return queries.SetOpenWeekAutoClose(ctx, store.SetOpenWeekAutoCloseParams{
			HouseholdID: service.householdID, AutoCloseAt: after.autoClose(now),
		})
	})
	if err != nil {
		return settings.Settings{}, err
	}
	return saved, nil
}
