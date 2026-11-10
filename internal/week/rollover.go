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
// next one; a week that takes over the upcoming period (see takesOver) instead starts on the date
// of that upcoming rollover and closes at the one after it. A manual week starts today and has no
// close time.
func (rules rules) opening(now time.Time, takeover bool) (string, sql.NullTime) {
	if rules.manual {
		return localDate(now, rules.location), sql.NullTime{}
	}
	if takeover {
		upcoming := rules.next(now)
		return localDate(upcoming, rules.location), sql.NullTime{Time: rules.next(upcoming), Valid: true}
	}
	return previousBoundaryDate(now, rules.startDay, rules.minutes, rules.location), rules.autoClose(now)
}

// autoClose returns when a week open at now closes itself: the next rollover strictly after now,
// so it is never already due, or NULL in manual mode.
func (rules rules) autoClose(now time.Time) sql.NullTime {
	if rules.manual {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: rules.next(now), Valid: true}
}

// next returns the first rollover strictly after after.
func (rules rules) next(after time.Time) time.Time {
	return nextBoundary(after, rules.startDay, rules.minutes, rules.location)
}

// takesOver reports whether a week opened at now takes over the upcoming period rather than the
// one already running: the household's previous week was closed by hand before its own rollover,
// and that rollover is still ahead. A household that shops on Saturday for a week starting Sunday
// closes the old week and plans the new one on Saturday; without this, the new week would close
// itself on Sunday. previous is the most recently closed week, or the zero Week if there is none.
//
// The upcoming period begins at the next rollover under the household's current settings.
// When nothing has changed that rollover is previous.auto_close_at itself; taking it from the
// current settings instead keeps a settings change made in between, and a week that was itself
// started early and then closed again before its period began, from skipping a whole week.
func takesOver(now time.Time, previous store.Week) bool {
	return previous.ClosedAt.Valid && previous.AutoCloseAt.Valid &&
		previous.ClosedAt.Time.Before(previous.AutoCloseAt.Time) && previous.AutoCloseAt.Time.After(now)
}

// rescheduled returns the open week's automatic close time when a settings change from before to
// after is saved at now: the next rollover after now under the new settings, or none in manual
// mode. A week started early that has not yet reached the rollover it skips keeps skipping one:
// it closes at the second rollover after now. Such a week is recognised by a rollover under the
// old settings falling between now and its close time; an ordinary week closes at the old
// settings' next rollover, which a settings change or opening the week set.
func rescheduled(now time.Time, closeAt sql.NullTime, before, after rules) sql.NullTime {
	if after.manual {
		return sql.NullTime{}
	}
	next := after.next(now)
	if closeAt.Valid && !before.manual && before.next(now).Before(closeAt.Time) {
		next = after.next(next)
	}
	return sql.NullTime{Time: next, Valid: true}
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

// lastDay returns the last local date of a week that ends at end: the date of the instant one
// second before it. A week that rolls over at Sunday 12:00 AM ends on Saturday; one that rolls
// over at Friday 6:00 PM ends on Friday. Closing a week stores lastDay(closed_at) in the zone the
// household has at that moment (weeks.ends_on); CloseExpiredWeek computes the same in SQL.
func lastDay(end time.Time, location *time.Location) string {
	return localDate(end.Add(-time.Second), location)
}

// endsOn returns a week's last local date: the date stored when it closed, or, while it is open,
// the last day before it closes itself. An open manual week has no end.
func endsOn(stored sql.NullString, autoCloseAt sql.NullTime, location *time.Location) sql.NullString {
	if stored.Valid {
		return stored
	}
	if !autoCloseAt.Valid {
		return sql.NullString{}
	}
	return sql.NullString{String: lastDay(autoCloseAt.Time, location), Valid: true}
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
// day, time, or mode, the open week's automatic close time moves in the same transaction (see
// rescheduled). Saving never closes a week; a week whose close time had already passed closes
// first, as on any read.
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
		open, err := queries.GetOpenWeek(ctx, service.householdID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		previous, err := rulesFrom(before.Timezone, before.WeekStartDay, before.RolloverMinutes, before.RolloverMode)
		if err != nil {
			return err
		}
		after, err := rulesFrom(saved.Timezone, saved.WeekStartDay, saved.RolloverMinutes, saved.RolloverMode)
		if err != nil {
			return err
		}
		return queries.SetOpenWeekAutoClose(ctx, store.SetOpenWeekAutoCloseParams{
			HouseholdID: service.householdID, AutoCloseAt: rescheduled(now, open.AutoCloseAt, previous, after),
		})
	})
	if err != nil {
		return settings.Settings{}, err
	}
	return saved, nil
}

// Close ends the household's open week now, by hand, in either rollover mode. The week moves to
// History with its meals and checklist as they are, and its last local date is fixed in the zone
// the household has now. Closing never generates the next week (D050). The week's automatic close
// time is kept, so a week opened before that rollover takes over the upcoming period (takesOver).
// With no open week, after an expired one has closed, it returns ErrNoCurrentWeek.
func (service *Household) Close(ctx context.Context, now time.Time) error {
	var closed int64
	err := service.transact(ctx, func(queries *store.Queries) error {
		// The household's zone is read, and held, before the week row is touched, in the same
		// order as Generate and SaveSettings.
		weekRules, err := service.weekRules(ctx, queries)
		if err != nil {
			return err
		}
		if err := service.closeExpired(ctx, queries, now); err != nil {
			return err
		}
		closed, err = queries.CloseOpenWeek(ctx, store.CloseOpenWeekParams{
			HouseholdID: service.householdID, ClosedAt: now, EndsOn: lastDay(now, weekRules.location),
		})
		return err
	})
	if err != nil {
		return err
	}
	if closed == 0 {
		// An expired week closed at its own rollover above, and that is kept.
		return ErrNoCurrentWeek
	}
	return nil
}
