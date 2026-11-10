package week_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/settings"
	"github.com/mbcoward3/grocery-router/internal/week"
)

// Closing a week by hand moves it to History as it was, keeps its own rollover time, and creates
// nothing new.
func TestCloseMovesTheWeekToHistory(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	generated, err := service.Generate(ctx, testNowUTC(), 2)
	if err != nil {
		t.Fatal(err)
	}
	checklist, err := service.Checklist(ctx, testNowUTC())
	if err != nil || len(checklist.Lines) < 2 {
		t.Fatalf("checklist = %d lines, %v", len(checklist.Lines), err)
	}
	if err := service.SetLineCompleted(ctx, testNowUTC(), checklist.Lines[0].ID, true); err != nil {
		t.Fatal(err)
	}

	closedAt := testNowUTC().Add(time.Hour)
	if err := service.Close(ctx, closedAt); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Current(ctx, closedAt); !errors.Is(err, week.ErrNoCurrentWeek) {
		t.Fatalf("Current after Close error = %v, want ErrNoCurrentWeek", err)
	}
	autoClose, closed := weekCloseTimes(t, db, generated.Week.ID)
	if !closed.Valid || !closed.Time.Equal(closedAt) || !autoClose.Time.Equal(time.Date(2026, time.August, 23, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("closed week: auto_close_at = %v, closed_at = %v", autoClose, closed)
	}
	past, err := service.PastWeeks(ctx, closedAt)
	if err != nil || len(past) != 1 || past[0].ID != generated.Week.ID || past[0].EndsOn.String != "2026-08-19" {
		t.Fatalf("history = %#v, %v", past, err)
	}
	detail, err := service.PastWeek(ctx, closedAt, generated.Week.ID)
	if err != nil || len(detail.Recipes) != 2 || len(detail.Checklist.Lines) != len(checklist.Lines) {
		t.Fatalf("closed week detail = %#v, %v", detail, err)
	}
	for _, line := range detail.Checklist.Lines {
		if want := line.ID == checklist.Lines[0].ID; (line.IsCompleted == 1) != want {
			t.Fatalf("line %d completed = %d, want %t", line.ID, line.IsCompleted, want)
		}
	}
	var weeks int
	if err := db.QueryRow(`SELECT count(*) FROM weeks`).Scan(&weeks); err != nil || weeks != 1 {
		t.Fatalf("weeks after Close = %d, %v; want 1", weeks, err)
	}

	// Nothing is left to close.
	if err := service.Close(ctx, closedAt.Add(time.Minute)); !errors.Is(err, week.ErrNoCurrentWeek) {
		t.Fatalf("second Close error = %v, want ErrNoCurrentWeek", err)
	}
}

func TestCloseWithoutAWeek(t *testing.T) {
	service, _ := testService(t)
	if err := service.Close(context.Background(), testNowUTC()); !errors.Is(err, week.ErrNoCurrentWeek) {
		t.Fatalf("Close error = %v, want ErrNoCurrentWeek", err)
	}
}

// A week whose rollover has already passed closes at that rollover, not when Close is called.
func TestCloseAfterTheRolloverFindsNoWeek(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	generated, err := service.Generate(ctx, testNowUTC(), 2)
	if err != nil {
		t.Fatal(err)
	}
	boundary := time.Date(2026, time.August, 23, 0, 0, 0, 0, time.UTC)
	if err := service.Close(ctx, boundary.Add(time.Hour)); !errors.Is(err, week.ErrNoCurrentWeek) {
		t.Fatalf("Close error = %v, want ErrNoCurrentWeek", err)
	}
	if _, closed := weekCloseTimes(t, db, generated.Week.ID); !closed.Time.Equal(boundary) {
		t.Fatalf("closed_at = %v, want %s", closed, boundary)
	}
}

// A household that shops on Saturday for the week starting Sunday closes the old week and plans
// the new one on Saturday. The new week takes over the coming week instead of closing on Sunday.
func TestStartingNextWeekEarlyTakesOverTheUpcomingPeriod(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	if _, err := service.Generate(ctx, testNowUTC(), 2); err != nil {
		t.Fatal(err)
	}
	saturday := time.Date(2026, time.August, 22, 10, 0, 0, 0, time.UTC)
	if err := service.Close(ctx, saturday); err != nil {
		t.Fatal(err)
	}
	next, err := service.Generate(ctx, saturday.Add(5*time.Minute), 2)
	if err != nil {
		t.Fatal(err)
	}
	followingSunday := time.Date(2026, time.August, 30, 0, 0, 0, 0, time.UTC)
	if next.Week.StartsOn != "2026-08-23" || next.EndsOn.String != "2026-08-29" || !next.Week.AutoCloseAt.Time.Equal(followingSunday) {
		t.Fatalf("week started early = %#v, ends %v", next.Week, next.EndsOn)
	}
	for _, now := range []time.Time{
		time.Date(2026, time.August, 23, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.August, 29, 10, 0, 0, 0, time.UTC),
	} {
		current, err := service.Current(ctx, now)
		if err != nil || current.Week.ID != next.Week.ID {
			t.Fatalf("Current at %s = %#v, %v; want the week started early", now, current.Week, err)
		}
	}
	if _, err := service.Current(ctx, followingSunday); !errors.Is(err, week.ErrNoCurrentWeek) {
		t.Fatalf("Current at %s error = %v, want ErrNoCurrentWeek", followingSunday, err)
	}
	past, err := service.PastWeeks(ctx, followingSunday)
	if err != nil || len(past) != 2 || past[0].ID != next.Week.ID || past[0].EndsOn.String != "2026-08-29" {
		t.Fatalf("history = %#v, %v", past, err)
	}
}

// Once the rollover a closed week would have reached has passed, the next week opens as usual.
func TestClosingThenWaitingPastTheRolloverOpensAnOrdinaryWeek(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	if _, err := service.Generate(ctx, testNowUTC(), 2); err != nil {
		t.Fatal(err)
	}
	if err := service.Close(ctx, time.Date(2026, time.August, 22, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	monday := time.Date(2026, time.August, 24, 9, 0, 0, 0, time.UTC)
	next, err := service.Generate(ctx, monday, 2)
	if err != nil {
		t.Fatal(err)
	}
	if next.Week.StartsOn != "2026-08-23" || !next.Week.AutoCloseAt.Time.Equal(time.Date(2026, time.August, 30, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("week after waiting = %#v", next.Week)
	}
}

// A week started early and closed again before its own period began hands that same period to
// the next week; it does not skip a whole week.
func TestClosingAWeekStartedEarlyBeforeItBegins(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	if _, err := service.Generate(ctx, testNowUTC(), 2); err != nil {
		t.Fatal(err)
	}
	saturday := time.Date(2026, time.August, 22, 10, 0, 0, 0, time.UTC)
	if err := service.Close(ctx, saturday); err != nil {
		t.Fatal(err)
	}
	early, err := service.Generate(ctx, saturday.Add(5*time.Minute), 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Close(ctx, saturday.Add(10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	again, err := service.Generate(ctx, saturday.Add(15*time.Minute), 2)
	if err != nil {
		t.Fatal(err)
	}
	if again.Week.ID == early.Week.ID || again.Week.StartsOn != "2026-08-23" ||
		!again.Week.AutoCloseAt.Time.Equal(time.Date(2026, time.August, 30, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("week planned again = %#v", again.Week)
	}
	// The week closed before its period began ends on its start date, not the day before.
	past, err := service.PastWeeks(ctx, saturday.Add(20*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	for _, closed := range past {
		if closed.EndsOn.String < closed.StartsOn {
			t.Fatalf("week %d ends %s before it starts %s", closed.ID, closed.EndsOn.String, closed.StartsOn)
		}
		if closed.ID == early.Week.ID && closed.EndsOn.String != "2026-08-23" {
			t.Fatalf("week started early and closed again ends %s, want 2026-08-23", closed.EndsOn.String)
		}
	}
}

// In manual mode the week closes only by hand; the next week starts on the day it is planned.
func TestCloseInManualMode(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	manual := settings.Manual
	saveSettings(t, service, testNowUTC(), settings.Change{RolloverMode: &manual})
	generated, err := service.Generate(ctx, testNowUTC(), 2)
	if err != nil {
		t.Fatal(err)
	}
	closedAt := testNowUTC().AddDate(0, 0, 10)
	if err := service.Close(ctx, closedAt); err != nil {
		t.Fatal(err)
	}
	past, err := service.PastWeeks(ctx, closedAt)
	if err != nil || len(past) != 1 || past[0].ID != generated.Week.ID || past[0].StartsOn != "2026-08-19" ||
		past[0].EndsOn.String != "2026-08-29" || past[0].AutoCloseAt.Valid {
		t.Fatalf("history = %#v, %v", past, err)
	}
	next, err := service.Generate(ctx, closedAt.Add(time.Minute), 2)
	if err != nil || next.Week.StartsOn != "2026-08-29" || next.Week.AutoCloseAt.Valid {
		t.Fatalf("next manual week = %#v, %v", next.Week, err)
	}
}

// A settings change while a week started early is waiting for its period keeps it from closing
// at the rollover it was opened to skip.
func TestSaveSettingsKeepsAWeekStartedEarly(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	if _, err := service.Generate(ctx, testNowUTC(), 2); err != nil {
		t.Fatal(err)
	}
	saturday := time.Date(2026, time.August, 22, 10, 0, 0, 0, time.UTC)
	if err := service.Close(ctx, saturday); err != nil {
		t.Fatal(err)
	}
	early, err := service.Generate(ctx, saturday.Add(5*time.Minute), 2)
	if err != nil {
		t.Fatal(err)
	}

	// Sunday 06:00 instead of 12:00 AM: the week still runs through the coming week.
	morning := 6 * 60
	saveSettings(t, service, saturday.Add(time.Hour), settings.Change{RolloverMinutes: &morning})
	want := time.Date(2026, time.August, 30, 6, 0, 0, 0, time.UTC)
	if autoClose, closed := weekCloseTimes(t, db, early.Week.ID); !autoClose.Time.Equal(want) || closed.Valid {
		t.Fatalf("after Sunday 06:00: auto_close_at = %v, closed_at = %v; want %s", autoClose, closed, want)
	}
	if current, err := service.Current(ctx, time.Date(2026, time.August, 23, 7, 0, 0, 0, time.UTC)); err != nil || current.Week.ID != early.Week.ID {
		t.Fatalf("Sunday 07:00 current = %#v, %v", current.Week, err)
	}
}

// A change that waits for a concurrent close does not write into the week that closed.
func TestChangeWaitingForACloseFindsNoWeek(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	generated, err := service.Generate(ctx, testNowUTC(), 2)
	if err != nil {
		t.Fatal(err)
	}
	closing, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer closing.Rollback()
	if _, err := closing.ExecContext(ctx, `UPDATE weeks SET closed_at = $1, ends_on = '2026-08-19' WHERE id = $2`,
		testNowUTC().Add(time.Minute), generated.Week.ID); err != nil {
		t.Fatal(err)
	}
	added := make(chan error, 1)
	go func() {
		_, err := service.Add(ctx, testNowUTC().Add(2*time.Minute), generated.Recipes[0].RecipeID)
		added <- err
	}()
	// Add must wait for the close's row lock; without LockOpenWeek it would not, and would finish.
	deadline := time.Now().Add(5 * time.Second)
	for {
		var waiting int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_stat_activity
			WHERE datname = current_database() AND wait_event_type = 'Lock'`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting > 0 {
			break
		}
		select {
		case err := <-added:
			t.Fatalf("Add finished without waiting for the close: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("Add never waited for the close's lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := closing.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-added:
		if !errors.Is(err, week.ErrNoCurrentWeek) {
			t.Fatalf("Add during a close error = %v, want ErrNoCurrentWeek", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Add did not finish after the close committed")
	}
	var recipes int
	if err := db.QueryRow(`SELECT count(*) FROM week_recipes WHERE week_id = $1`, generated.Week.ID).Scan(&recipes); err != nil || recipes != 2 {
		t.Fatalf("closed week recipes = %d, %v; want 2", recipes, err)
	}
}
