package week

import (
	"testing"
	"time"
	_ "time/tzdata"
)

func TestNextBoundary(t *testing.T) {
	load := func(name string) *time.Location {
		t.Helper()
		location, err := time.LoadLocation(name)
		if err != nil {
			t.Fatal(err)
		}
		return location
	}
	chicago := load("America/Chicago")
	santiago := load("America/Santiago")
	cases := []struct {
		name      string
		after     time.Time
		startDay  time.Weekday
		atMinutes int
		location  *time.Location
		want      time.Time
	}{
		{"mid-week goes to the coming Sunday", time.Date(2026, time.August, 19, 12, 0, 0, 0, chicago), time.Sunday, 0, chicago,
			time.Date(2026, time.August, 23, 0, 0, 0, 0, chicago)},
		{"exactly on the boundary goes to the next week", time.Date(2026, time.August, 16, 0, 0, 0, 0, chicago), time.Sunday, 0, chicago,
			time.Date(2026, time.August, 23, 0, 0, 0, 0, chicago)},
		{"one nanosecond before the boundary", time.Date(2026, time.August, 15, 23, 59, 59, 999999999, chicago), time.Sunday, 0, chicago,
			time.Date(2026, time.August, 16, 0, 0, 0, 0, chicago)},
		{"after is read in the household zone", time.Date(2026, time.August, 16, 1, 0, 0, 0, time.UTC), time.Sunday, 0, chicago,
			time.Date(2026, time.August, 16, 0, 0, 0, 0, chicago)},
		{"spring-forward week", time.Date(2026, time.March, 8, 1, 0, 0, 0, chicago), time.Sunday, 0, chicago,
			time.Date(2026, time.March, 15, 0, 0, 0, 0, chicago)},
		{"fall-back week", time.Date(2026, time.November, 1, 0, 30, 0, 0, chicago), time.Sunday, 0, chicago,
			time.Date(2026, time.November, 8, 0, 0, 0, 0, chicago)},
		{"other day and time later the same day", time.Date(2026, time.August, 21, 17, 59, 0, 0, chicago), time.Friday, 18 * 60, chicago,
			time.Date(2026, time.August, 21, 18, 0, 0, 0, chicago)},
		{"other day and time already passed", time.Date(2026, time.August, 21, 18, 0, 0, 0, chicago), time.Friday, 18 * 60, chicago,
			time.Date(2026, time.August, 28, 18, 0, 0, 0, chicago)},
		{"year boundary", time.Date(2026, time.December, 31, 12, 0, 0, 0, time.UTC), time.Sunday, 0, time.UTC,
			time.Date(2027, time.January, 3, 0, 0, 0, 0, time.UTC)},
		// Santiago's clocks skip 00:00-01:00 on 2026-09-06: the week ends when Sunday begins,
		// 01:00 local, not an hour early on Saturday.
		{"midnight skipped by daylight saving", time.Date(2026, time.September, 2, 12, 0, 0, 0, santiago), time.Sunday, 0, santiago,
			time.Date(2026, time.September, 6, 4, 0, 0, 0, time.UTC)},
	}
	for _, testCase := range cases {
		got := nextBoundary(testCase.after, testCase.startDay, testCase.atMinutes, testCase.location)
		if !got.Equal(testCase.want) {
			t.Errorf("%s: nextBoundary(%s) = %s, want %s", testCase.name, testCase.after, got, testCase.want)
		}
	}

	// Across a daylight-saving change the boundary keeps local midnight while its UTC offset
	// differs from the input's.
	for _, after := range []time.Time{
		time.Date(2026, time.March, 8, 1, 0, 0, 0, chicago),
		time.Date(2026, time.November, 1, 0, 30, 0, 0, chicago),
	} {
		got := nextBoundary(after, time.Sunday, 0, chicago).In(chicago)
		if hour, minute, second := got.Clock(); hour != 0 || minute != 0 || second != 0 || got.Weekday() != time.Sunday {
			t.Errorf("nextBoundary(%s) = %s, want Sunday 00:00 local", after, got)
		}
		_, afterOffset := after.Zone()
		_, gotOffset := got.Zone()
		if afterOffset == gotOffset {
			t.Errorf("nextBoundary(%s) = %s kept offset %d across a daylight-saving change", after, got, gotOffset)
		}
	}
}
