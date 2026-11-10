package week

import (
	"database/sql"
	"testing"
	"time"
	_ "time/tzdata"
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
		startsOn, autoClose := rules{location: testCase.location, startDay: time.Sunday}.opening(testCase.now)
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
	startsOn, autoClose := friday.opening(monday)
	if startsOn != "2026-08-21" || !autoClose.Valid || !autoClose.Time.Equal(time.Date(2026, time.August, 28, 18, 0, 0, 0, chicago)) {
		t.Fatalf("automatic opening = %s, %v", startsOn, autoClose)
	}

	// A manual week starts on today's local date and has no close time.
	manual := friday
	manual.manual = true
	saturdayEvening := time.Date(2026, time.August, 16, 1, 0, 0, 0, time.UTC)
	startsOn, autoClose = manual.opening(saturdayEvening)
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
