package week_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/week"
)

const testHousehold = "c0a7a2d8-669b-4e47-91c1-4d9a32f339d5"

func testNowUTC() time.Time {
	return time.Date(2026, time.August, 19, 12, 0, 0, 0, time.UTC)
}

func closedAt(t *testing.T, db *sql.DB, weekID int64) sql.NullTime {
	t.Helper()
	var closed sql.NullTime
	if err := db.QueryRow(`SELECT closed_at FROM weeks WHERE id = $1`, weekID).Scan(&closed); err != nil {
		t.Fatal(err)
	}
	return closed
}

// Closing a plan moves it to History as it was and creates nothing new.
func TestCloseMovesThePlanToHistory(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	generated, err := service.Generate(ctx, testNowUTC(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if generated.Week.StartsOn != "2026-08-19" || generated.Week.EndsOn.Valid {
		t.Fatalf("new plan = %#v, want it to start today with no end", generated.Week)
	}
	checklist, err := service.Checklist(ctx)
	if err != nil || len(checklist.Lines) < 2 {
		t.Fatalf("checklist = %d lines, %v", len(checklist.Lines), err)
	}
	if err := service.SetLineCompleted(ctx, checklist.Lines[0].ID, true); err != nil {
		t.Fatal(err)
	}

	closeTime := testNowUTC().Add(time.Hour)
	if err := service.Close(ctx, closeTime); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Current(ctx); !errors.Is(err, week.ErrNoCurrentWeek) {
		t.Fatalf("Current after Close error = %v, want ErrNoCurrentWeek", err)
	}
	if closed := closedAt(t, db, generated.Week.ID); !closed.Time.Equal(closeTime) {
		t.Fatalf("closed_at = %v, want %s", closed, closeTime)
	}
	past, err := service.PastWeeks(ctx)
	if err != nil || len(past) != 1 || past[0].ID != generated.Week.ID || past[0].EndsOn.String != "2026-08-19" {
		t.Fatalf("history = %#v, %v", past, err)
	}
	detail, err := service.PastWeek(ctx, generated.Week.ID)
	if err != nil || len(detail.Recipes) != 2 || len(detail.Checklist.Lines) != len(checklist.Lines) {
		t.Fatalf("closed plan detail = %#v, %v", detail, err)
	}
	for _, line := range detail.Checklist.Lines {
		if want := line.ID == checklist.Lines[0].ID; (line.IsCompleted == 1) != want {
			t.Fatalf("line %d completed = %d, want %t", line.ID, line.IsCompleted, want)
		}
	}
	var weeks int
	if err := db.QueryRow(`SELECT count(*) FROM weeks`).Scan(&weeks); err != nil || weeks != 1 {
		t.Fatalf("plans after Close = %d, %v; want 1", weeks, err)
	}

	// Nothing is left to close.
	if err := service.Close(ctx, closeTime.Add(time.Minute)); !errors.Is(err, week.ErrNoCurrentWeek) {
		t.Fatalf("second Close error = %v, want ErrNoCurrentWeek", err)
	}
}

func TestCloseWithoutAPlan(t *testing.T) {
	service, _ := testService(t)
	if err := service.Close(context.Background(), testNowUTC()); !errors.Is(err, week.ErrNoCurrentWeek) {
		t.Fatalf("Close error = %v, want ErrNoCurrentWeek", err)
	}
}

// A plan has no length: it stays current however long the household takes to shop again, and it
// is dated in the household's own zone.
func TestPlanStaysOpenUntilClosed(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	if _, err := db.Exec(`UPDATE households SET timezone = 'America/Chicago' WHERE id = $1`, testHousehold); err != nil {
		t.Fatal(err)
	}
	chicago, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatal(err)
	}
	// 11:30 PM in Chicago is already the next day in UTC.
	opened := time.Date(2026, time.August, 19, 23, 30, 0, 0, chicago)
	tenDaysLater := opened.Add(10 * 24 * time.Hour)

	generated, err := service.Generate(ctx, opened, 2)
	if err != nil {
		t.Fatal(err)
	}
	if generated.Week.StartsOn != "2026-08-19" || !generated.Week.OpenedAt.Equal(opened) {
		t.Fatalf("generated plan = %#v", generated.Week)
	}
	checklist, err := service.Checklist(ctx)
	if err != nil || len(checklist.Lines) == 0 {
		t.Fatalf("checklist = %#v, %v", checklist, err)
	}
	generatedLine := checklist.Lines[0].ID
	manual, err := service.AddManualLine(ctx, "Birthday candles")
	if err != nil {
		t.Fatal(err)
	}
	var baking int64
	if err := db.QueryRow(`SELECT id FROM store_sections WHERE household_id = $1 AND key = 'baking'`, testHousehold).Scan(&baking); err != nil {
		t.Fatal(err)
	}

	if current, err := service.Current(ctx); err != nil || current.Week.ID != generated.Week.ID || current.Week.EndsOn.Valid {
		t.Fatalf("current plan after ten days = %#v, %v", current.Week, err)
	}
	if err := service.Close(ctx, tenDaysLater); err != nil {
		t.Fatal(err)
	}
	past, err := service.PastWeeks(ctx)
	if err != nil || len(past) != 1 || past[0].StartsOn != "2026-08-19" || past[0].EndsOn.String != "2026-08-29" {
		t.Fatalf("history = %#v, %v", past, err)
	}

	// The closed plan's lines can no longer be changed.
	assertLineNotFound := func(name string, err error) {
		t.Helper()
		var validation week.ValidationError
		if !errors.As(err, &validation) || validation.Message != "shopping line not found or operation is not supported" {
			t.Fatalf("%s on a closed plan: %v", name, err)
		}
	}
	assertLineNotFound("SetLineCompleted", service.SetLineCompleted(ctx, generatedLine, true))
	assertLineNotFound("SetLineRemoved", service.SetLineRemoved(ctx, generatedLine, true))
	assertLineNotFound("SetLineOverride", service.SetLineOverride(ctx, generatedLine, "two"))
	assertLineNotFound("SetLineSection", service.SetLineSection(ctx, manual.ID, baking))

	// The next plan may start on the day the last one ended.
	next, err := service.Generate(ctx, tenDaysLater, 2)
	if err != nil || next.Week.ID == generated.Week.ID || next.Week.StartsOn != "2026-08-29" {
		t.Fatalf("next plan = %#v, %v", next.Week, err)
	}
}

// Reopening a past plan makes it current again with its meals and checklist as they were, and
// moves the plan that was current to History.
func TestReopenSwapsThePlans(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	first, err := service.Generate(ctx, testNowUTC(), 2)
	if err != nil {
		t.Fatal(err)
	}
	checklist, err := service.Checklist(ctx)
	if err != nil || len(checklist.Lines) == 0 {
		t.Fatalf("checklist = %#v, %v", checklist, err)
	}
	if err := service.SetLineCompleted(ctx, checklist.Lines[0].ID, true); err != nil {
		t.Fatal(err)
	}
	if err := service.Close(ctx, testNowUTC().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	later := testNowUTC().Add(3 * 24 * time.Hour)
	second, err := service.Generate(ctx, later, 3)
	if err != nil {
		t.Fatal(err)
	}

	reopenedAt := later.Add(2 * 24 * time.Hour)
	reopened, err := service.Reopen(ctx, reopenedAt, first.Week.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Week.ID != first.Week.ID || reopened.Week.StartsOn != "2026-08-19" || reopened.Week.EndsOn.Valid || len(reopened.Recipes) != 2 {
		t.Fatalf("reopened plan = %#v", reopened)
	}
	current, err := service.Current(ctx)
	if err != nil || current.Week.ID != first.Week.ID {
		t.Fatalf("current after Reopen = %#v, %v", current.Week, err)
	}
	after, err := service.Checklist(ctx)
	if err != nil || len(after.Lines) != len(checklist.Lines) {
		t.Fatalf("reopened checklist = %#v, %v", after, err)
	}
	for _, line := range after.Lines {
		if want := line.ID == checklist.Lines[0].ID; (line.IsCompleted == 1) != want {
			t.Fatalf("line %d completed = %d, want %t", line.ID, line.IsCompleted, want)
		}
	}
	// The reopened plan takes changes again.
	if err := service.SetLineCompleted(ctx, checklist.Lines[0].ID, false); err != nil {
		t.Fatal(err)
	}
	past, err := service.PastWeeks(ctx)
	if err != nil || len(past) != 1 || past[0].ID != second.Week.ID || past[0].EndsOn.String != "2026-08-24" {
		t.Fatalf("history after Reopen = %#v, %v", past, err)
	}
	if closed := closedAt(t, db, second.Week.ID); !closed.Time.Equal(reopenedAt) {
		t.Fatalf("replaced plan closed_at = %v, want %s", closed, reopenedAt)
	}

	// Only a plan in History can be reopened.
	for _, id := range []int64{first.Week.ID, 999999} {
		if _, err := service.Reopen(ctx, reopenedAt, id); !errors.Is(err, week.ErrPastWeek) {
			t.Fatalf("Reopen(%d) error = %v, want ErrPastWeek", id, err)
		}
	}
	// With no current plan, reopening just makes the past one current.
	if err := service.Close(ctx, reopenedAt); err != nil {
		t.Fatal(err)
	}
	if again, err := service.Reopen(ctx, reopenedAt, second.Week.ID); err != nil || again.Week.ID != second.Week.ID {
		t.Fatalf("Reopen without a current plan = %#v, %v", again.Week, err)
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
		_, err := service.Add(ctx, generated.Recipes[0].RecipeID)
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

// With avoid_recent_plans set, Generate picks recipes from the last plans only when too few
// others remain, and random swap prefers recipes outside them (D052).
func TestGenerateAvoidsRecentPlans(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	recipeIDs := func(view week.View) map[int64]bool {
		ids := map[int64]bool{}
		for _, occurrence := range view.Recipes {
			ids[occurrence.RecipeID] = true
		}
		return ids
	}
	first, err := service.Generate(ctx, testNowUTC(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Close(ctx, testNowUTC()); err != nil {
		t.Fatal(err)
	}
	// Off by default: the deterministic picker chooses the same recipes again.
	again, err := service.Generate(ctx, testNowUTC(), 5)
	if err != nil {
		t.Fatal(err)
	}
	for id := range recipeIDs(first) {
		if !recipeIDs(again)[id] {
			t.Fatalf("with the setting off, the second plan = %v, want the first plan's recipes %v", recipeIDs(again), recipeIDs(first))
		}
	}

	if _, err := db.Exec(`UPDATE households SET avoid_recent_plans = 1 WHERE id = $1`, testHousehold); err != nil {
		t.Fatal(err)
	}
	// The open plan is not a past plan, so regenerating it avoids only the closed one.
	avoided, err := service.Generate(ctx, testNowUTC(), 5)
	if err != nil {
		t.Fatal(err)
	}
	for id := range recipeIDs(avoided) {
		if recipeIDs(first)[id] {
			t.Fatalf("plan repeats recipe %d from the last plan", id)
		}
	}
	// Random swap does not bring a recent recipe back while others remain.
	swapped, err := service.RandomSwap(ctx, avoided.Recipes[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if recipeIDs(first)[swapped.Recipes[0].RecipeID] {
		t.Fatalf("random swap chose recipe %d from the last plan", swapped.Recipes[0].RecipeID)
	}

	// Asking for more than the other recipes allows fills the rest from the recent ones.
	var verified int
	if err := db.QueryRow(`SELECT count(*) FROM recipes WHERE household_id = $1 AND status = 'verified'`, testHousehold).Scan(&verified); err != nil {
		t.Fatal(err)
	}
	full, err := service.Generate(ctx, testNowUTC(), verified-2)
	if err != nil {
		t.Fatal(err)
	}
	repeats := 0
	for id := range recipeIDs(full) {
		if recipeIDs(first)[id] {
			repeats++
		}
	}
	if len(full.Recipes) != verified-2 || repeats != 3 {
		t.Fatalf("full plan = %d recipes with %d repeats, want %d with 3", len(full.Recipes), repeats, verified-2)
	}
}
