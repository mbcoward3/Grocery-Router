package week

import (
	"database/sql"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/mbcoward3/grocery-router/internal/store"
)

func loadZone(t *testing.T, name string) *time.Location {
	t.Helper()
	location, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return location
}

func TestPreviousBoundaryDate(t *testing.T) {
	chicago := loadZone(t, "America/Chicago")
	cases := []struct {
		name      string
		now       time.Time
		startDay  time.Weekday
		atMinutes int
		want      string
	}{
		{"Monday after a Friday 18:00 rollover", time.Date(2026, time.August, 24, 12, 0, 0, 0, chicago), time.Friday, 18 * 60, "2026-08-21"},
		{"Friday 17:59 is still the week before", time.Date(2026, time.August, 21, 17, 59, 0, 0, chicago), time.Friday, 18 * 60, "2026-08-14"},
		{"Friday 18:00 begins the week", time.Date(2026, time.August, 21, 18, 0, 0, 0, chicago), time.Friday, 18 * 60, "2026-08-21"},
		{"Friday 18:00 is read in the household zone", time.Date(2026, time.August, 21, 23, 30, 0, 0, time.UTC), time.Friday, 18 * 60, "2026-08-21"},
		{"Saturday 08:00 rollover on a Wednesday", time.Date(2026, time.August, 19, 12, 0, 0, 0, chicago), time.Saturday, 8 * 60, "2026-08-15"},
		{"year boundary", time.Date(2027, time.January, 1, 12, 0, 0, 0, chicago), time.Thursday, 23 * 60, "2026-12-31"},
	}
	for _, testCase := range cases {
		if got := previousBoundaryDate(testCase.now, testCase.startDay, testCase.atMinutes, chicago); got != testCase.want {
			t.Errorf("%s: previousBoundaryDate(%s) = %s, want %s", testCase.name, testCase.now, got, testCase.want)
		}
	}
}

// The default rollover, Sunday 12:00 AM, must start weeks exactly where SundayIn did before
// rollover settings existed, including at daylight-saving changes.
func TestDefaultRolloverMatchesSundayIn(t *testing.T) {
	chicago := loadZone(t, "America/Chicago")
	santiago := loadZone(t, "America/Santiago")
	havana := loadZone(t, "America/Havana")
	cases := []struct {
		now      time.Time
		location *time.Location
	}{
		{time.Date(2026, time.August, 16, 1, 0, 0, 0, time.UTC), time.UTC},
		{time.Date(2026, time.August, 16, 1, 0, 0, 0, time.UTC), chicago},
		{time.Date(2026, time.August, 15, 23, 59, 59, 0, chicago), chicago},
		{time.Date(2026, time.August, 16, 0, 0, 0, 0, chicago), chicago},
		{time.Date(2026, time.March, 14, 23, 30, 0, 0, chicago), chicago},
		{time.Date(2026, time.March, 8, 3, 30, 0, 0, chicago), chicago},
		{time.Date(2026, time.November, 1, 6, 30, 0, 0, time.UTC), chicago},
		{time.Date(2026, time.September, 6, 3, 59, 0, 0, time.UTC), santiago},
		{time.Date(2026, time.September, 6, 4, 30, 0, 0, time.UTC), santiago},
		{time.Date(2026, time.September, 12, 0, 30, 0, 0, santiago), santiago},
		{time.Date(2026, time.March, 14, 0, 30, 0, 0, havana), havana},
		{time.Date(2027, time.January, 2, 12, 0, 0, 0, time.UTC), time.UTC},
	}
	for _, testCase := range cases {
		startsOn, autoClose := rules{location: testCase.location, startDay: time.Sunday}.opening(testCase.now, false)
		if want := SundayIn(testCase.now, testCase.location); startsOn != want {
			t.Errorf("opening(%s in %s) starts %s, want %s", testCase.now, testCase.location, startsOn, want)
		}
		if want := nextBoundary(testCase.now, time.Sunday, 0, testCase.location); !autoClose.Valid || !autoClose.Time.Equal(want) {
			t.Errorf("opening(%s in %s) closes %v, want %s", testCase.now, testCase.location, autoClose, want)
		}
	}
}

func TestOpening(t *testing.T) {
	chicago := loadZone(t, "America/Chicago")
	friday := rules{location: chicago, startDay: time.Friday, minutes: 18 * 60}
	monday := time.Date(2026, time.August, 24, 12, 0, 0, 0, chicago)
	startsOn, autoClose := friday.opening(monday, false)
	if startsOn != "2026-08-21" || !autoClose.Valid || !autoClose.Time.Equal(time.Date(2026, time.August, 28, 18, 0, 0, 0, chicago)) {
		t.Fatalf("automatic opening = %s, %v", startsOn, autoClose)
	}

	// A manual week starts on today's local date and has no close time, even right after a week
	// was closed early: only automatic weeks take over the upcoming period.
	manual := friday
	manual.manual = true
	saturdayEvening := time.Date(2026, time.August, 16, 1, 0, 0, 0, time.UTC)
	startsOn, autoClose = manual.opening(saturdayEvening, true)
	if startsOn != "2026-08-15" || autoClose.Valid {
		t.Fatalf("manual opening = %s, %v", startsOn, autoClose)
	}
	if got := manual.autoClose(monday); got.Valid {
		t.Fatalf("manual autoClose = %v", got)
	}
}

func TestLastDay(t *testing.T) {
	chicago := loadZone(t, "America/Chicago")
	cases := []struct {
		name     string
		end      time.Time
		location *time.Location
		want     string
	}{
		{"Sunday 00:00 rollover ends Saturday", time.Date(2026, time.August, 23, 0, 0, 0, 0, chicago), chicago, "2026-08-22"},
		{"Friday 18:00 rollover ends Friday", time.Date(2026, time.August, 21, 18, 0, 0, 0, chicago), chicago, "2026-08-21"},
		{"closed by hand mid-afternoon ends that day", time.Date(2026, time.August, 22, 15, 4, 0, 0, chicago), chicago, "2026-08-22"},
		{"the date is household-local", time.Date(2026, time.August, 23, 0, 0, 0, 0, chicago), time.UTC, "2026-08-23"},
	}
	for _, testCase := range cases {
		if got := lastDay(testCase.end, testCase.location); got != testCase.want {
			t.Errorf("%s: lastDay = %s, want %s", testCase.name, got, testCase.want)
		}
	}
}

func TestEndsOn(t *testing.T) {
	chicago := loadZone(t, "America/Chicago")
	at := func(instant time.Time) sql.NullTime { return sql.NullTime{Time: instant, Valid: true} }
	date := func(value string) sql.NullString { return sql.NullString{String: value, Valid: true} }
	sundayMidnight := time.Date(2026, time.August, 23, 0, 0, 0, 0, chicago)
	cases := []struct {
		name        string
		stored      sql.NullString
		autoCloseAt sql.NullTime
		want        sql.NullString
	}{
		{"an open automatic week ends the day before its rollover", sql.NullString{}, at(sundayMidnight), date("2026-08-22")},
		// A closed week keeps the date fixed when it closed, whatever the zone is now.
		{"a closed week keeps its stored date", date("2026-08-21"), at(sundayMidnight), date("2026-08-21")},
		{"an open manual week has no end", sql.NullString{}, sql.NullTime{}, sql.NullString{}},
	}
	for _, testCase := range cases {
		if got := endsOn(testCase.stored, testCase.autoCloseAt, chicago); got != testCase.want {
			t.Errorf("%s: endsOn = %v, want %v", testCase.name, got, testCase.want)
		}
	}
}

func TestTakesOver(t *testing.T) {
	chicago := loadZone(t, "America/Chicago")
	at := func(instant time.Time) sql.NullTime { return sql.NullTime{Time: instant, Valid: true} }
	sunday := time.Date(2026, time.August, 23, 0, 0, 0, 0, chicago)
	saturday := time.Date(2026, time.August, 22, 10, 0, 0, 0, chicago)
	closedEarly := store.Week{AutoCloseAt: at(sunday), ClosedAt: at(saturday)}
	cases := []struct {
		name     string
		now      time.Time
		previous store.Week
		want     bool
	}{
		{"no earlier week", saturday, store.Week{}, false},
		{"closed by hand before a rollover still ahead", saturday.Add(5 * time.Minute), closedEarly, true},
		{"the skipped rollover has arrived", sunday, closedEarly, false},
		{"the skipped rollover has passed", sunday.AddDate(0, 0, 1), closedEarly, false},
		{"closed by its own rollover", sunday.Add(time.Minute), store.Week{AutoCloseAt: at(sunday), ClosedAt: at(sunday)}, false},
		{"a manual week has no rollover to skip", saturday.Add(5 * time.Minute), store.Week{ClosedAt: at(saturday)}, false},
	}
	for _, testCase := range cases {
		if got := takesOver(testCase.now, testCase.previous); got != testCase.want {
			t.Errorf("%s: takesOver = %t, want %t", testCase.name, got, testCase.want)
		}
	}
}

func TestOpeningTakesOverTheUpcomingPeriod(t *testing.T) {
	chicago := loadZone(t, "America/Chicago")
	santiago := loadZone(t, "America/Santiago")
	cases := []struct {
		name      string
		rules     rules
		now       time.Time
		startsOn  string
		autoClose time.Time
	}{
		{"Saturday morning for a Sunday 12:00 AM rollover", rules{location: chicago, startDay: time.Sunday},
			time.Date(2026, time.August, 22, 10, 5, 0, 0, chicago), "2026-08-23", time.Date(2026, time.August, 30, 0, 0, 0, 0, chicago)},
		{"earlier the same day as a Friday 6:00 PM rollover", rules{location: chicago, startDay: time.Friday, minutes: 18 * 60},
			time.Date(2026, time.August, 21, 10, 0, 0, 0, chicago), "2026-08-21", time.Date(2026, time.August, 28, 18, 0, 0, 0, chicago)},
		// Santiago skips 00:00-01:00 on 2026-09-06: the period still starts that Sunday.
		{"rollover skipped by daylight saving", rules{location: santiago, startDay: time.Sunday},
			time.Date(2026, time.September, 5, 12, 0, 0, 0, santiago), "2026-09-06", time.Date(2026, time.September, 13, 0, 0, 0, 0, santiago)},
	}
	for _, testCase := range cases {
		startsOn, autoClose := testCase.rules.opening(testCase.now, true)
		if startsOn != testCase.startsOn || !autoClose.Valid || !autoClose.Time.Equal(testCase.autoClose) {
			t.Errorf("%s: opening = %s, %v; want %s, %s", testCase.name, startsOn, autoClose, testCase.startsOn, testCase.autoClose)
		}
	}
}

func TestRescheduled(t *testing.T) {
	chicago := loadZone(t, "America/Chicago")
	newYork := loadZone(t, "America/New_York")
	at := func(instant time.Time) sql.NullTime { return sql.NullTime{Time: instant, Valid: true} }
	local := func(month time.Month, day, hour int) time.Time {
		return time.Date(2026, month, day, hour, 0, 0, 0, chicago)
	}
	sunday := rules{location: chicago, startDay: time.Sunday}
	manual := sunday
	manual.manual = true
	// A week opened early on Saturday 2026-08-22 skips the Sunday 08-23 rollover and closes 08-30.
	early := at(local(time.August, 30, 0))
	saturdayLate := local(time.August, 22, 11)
	cases := []struct {
		name    string
		now     time.Time
		closeAt sql.NullTime
		before  rules
		after   rules
		want    sql.NullTime
	}{
		{"an ordinary week moves to the next rollover", local(time.August, 19, 12), at(local(time.August, 23, 0)),
			sunday, rules{location: chicago, startDay: time.Saturday, minutes: 8 * 60}, at(local(time.August, 22, 8))},
		{"manual mode has no close time", saturdayLate, early, sunday, manual, sql.NullTime{}},
		{"a manual week switched to automatic closes at the next rollover", saturdayLate, sql.NullTime{}, manual, sunday,
			at(local(time.August, 23, 0))},
		{"a week started early still skips a later rollover the same night", saturdayLate, early,
			sunday, rules{location: chicago, startDay: time.Sunday, minutes: 6 * 60}, at(local(time.August, 30, 6))},
		{"a week started early still skips a rollover moved earlier the same day", saturdayLate, early,
			sunday, rules{location: chicago, startDay: time.Saturday, minutes: 18 * 60}, at(local(time.August, 29, 18))},
		{"a week started early still skips a rollover moved a day later", saturdayLate, early,
			sunday, rules{location: chicago, startDay: time.Monday}, at(local(time.August, 31, 0))},
		{"a week started early still skips its rollover in a new zone", saturdayLate, early,
			sunday, rules{location: newYork, startDay: time.Sunday}, at(time.Date(2026, time.August, 30, 0, 0, 0, 0, newYork))},
		// Once the rollover it skipped has passed, the week is running its own period.
		{"a week started early is ordinary once its period has begun", local(time.August, 23, 1), early,
			sunday, rules{location: chicago, startDay: time.Sunday, minutes: 6 * 60}, at(local(time.August, 23, 6))},
	}
	for _, testCase := range cases {
		got := rescheduled(testCase.now, testCase.closeAt, testCase.before, testCase.after)
		if got.Valid != testCase.want.Valid || !got.Time.Equal(testCase.want.Time) {
			t.Errorf("%s: rescheduled = %v, want %v", testCase.name, got, testCase.want)
		}
	}
}
