package week_test

import (
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/mbcoward3/grocery-router/internal/week"
)

func TestSundayInUsesHouseholdZone(t *testing.T) {
	load := func(name string) *time.Location {
		t.Helper()
		location, err := time.LoadLocation(name)
		if err != nil {
			t.Fatal(err)
		}
		return location
	}
	chicago := load("America/Chicago")
	cases := []struct {
		name     string
		now      time.Time
		location *time.Location
		want     string
	}{
		{"UTC Sunday 01:00", time.Date(2026, time.August, 16, 1, 0, 0, 0, time.UTC), time.UTC, "2026-08-16"},
		{"Chicago Saturday evening at the same instant", time.Date(2026, time.August, 16, 1, 0, 0, 0, time.UTC), chicago, "2026-08-09"},
		{"Chicago last second of Saturday", time.Date(2026, time.August, 15, 23, 59, 59, 0, chicago), chicago, "2026-08-09"},
		{"Chicago first instant of Sunday", time.Date(2026, time.August, 16, 0, 0, 0, 0, chicago), chicago, "2026-08-16"},
		{"Chicago mid-week", time.Date(2026, time.August, 19, 12, 0, 0, 0, time.UTC), chicago, "2026-08-16"},
		{"Tokyo is already Sunday", time.Date(2026, time.August, 15, 15, 0, 0, 0, time.UTC), load("Asia/Tokyo"), "2026-08-16"},
		{"Chicago week spanning spring forward", time.Date(2026, time.March, 14, 23, 30, 0, 0, chicago), chicago, "2026-03-08"},
		{"Chicago Sunday after spring forward", time.Date(2026, time.March, 8, 3, 30, 0, 0, chicago), chicago, "2026-03-08"},
		{"Chicago week spanning fall back", time.Date(2026, time.November, 7, 23, 30, 0, 0, chicago), chicago, "2026-11-01"},
		{"Chicago repeated hour on fall-back Sunday", time.Date(2026, time.November, 1, 6, 30, 0, 0, time.UTC), chicago, "2026-11-01"},
		// Santiago skips 00:00-01:00 on 2026-09-06; the Sunday must not slip to Saturday.
		{"Santiago week starting at a midnight gap", time.Date(2026, time.September, 12, 0, 30, 0, 0, load("America/Santiago")), load("America/Santiago"), "2026-09-06"},
		{"Havana week starting at a midnight gap", time.Date(2026, time.March, 14, 0, 30, 0, 0, load("America/Havana")), load("America/Havana"), "2026-03-08"},
		{"year boundary", time.Date(2027, time.January, 2, 12, 0, 0, 0, time.UTC), time.UTC, "2026-12-27"},
	}
	for _, testCase := range cases {
		if got := week.SundayIn(testCase.now, testCase.location); got != testCase.want {
			t.Errorf("%s: SundayIn(%s, %s) = %s, want %s", testCase.name, testCase.now, testCase.location, got, testCase.want)
		}
	}
}
