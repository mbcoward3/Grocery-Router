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

func TestEndsOn(t *testing.T) {
	chicago := loadZone(t, "America/Chicago")
	at := func(instant time.Time) sql.NullTime { return sql.NullTime{Time: instant, Valid: true} }
	sundayMidnight := time.Date(2026, time.August, 23, 0, 0, 0, 0, chicago)
	fridayEvening := time.Date(2026, time.August, 21, 18, 0, 0, 0, chicago)
	cases := []struct {
		name        string
		closedAt    sql.NullTime
		autoCloseAt sql.NullTime
		location    *time.Location
		want        sql.NullString
	}{
		{"Sunday 00:00 rollover ends Saturday", sql.NullTime{}, at(sundayMidnight), chicago, sql.NullString{String: "2026-08-22", Valid: true}},
		{"Friday 18:00 rollover ends Friday", sql.NullTime{}, at(fridayEvening), chicago, sql.NullString{String: "2026-08-21", Valid: true}},
		{"closed by time at Friday 18:00 ends Friday", at(fridayEvening), at(fridayEvening), chicago, sql.NullString{String: "2026-08-21", Valid: true}},
		{"closed time wins over the automatic one", at(fridayEvening), at(sundayMidnight), chicago, sql.NullString{String: "2026-08-21", Valid: true}},
		{"the date is household-local", sql.NullTime{}, at(sundayMidnight), time.UTC, sql.NullString{String: "2026-08-23", Valid: true}},
		{"open manual week has no end", sql.NullTime{}, sql.NullTime{}, chicago, sql.NullString{}},
	}
	for _, testCase := range cases {
		if got := endsOn(testCase.closedAt, testCase.autoCloseAt, testCase.location); got != testCase.want {
			t.Errorf("%s: endsOn = %v, want %v", testCase.name, got, testCase.want)
		}
	}
}
