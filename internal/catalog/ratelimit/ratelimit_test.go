package ratelimit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/catalog/clock"
	"github.com/mbcoward3/grocery-router/internal/catalog/ratelimit"
)

var start = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

func TestInterval(t *testing.T) {
	if got := ratelimit.Interval(0, 0); got != 5*time.Second {
		t.Errorf("floor = %v", got)
	}
	if got := ratelimit.Interval(8*time.Second, 3*time.Second); got != 8*time.Second {
		t.Errorf("site = %v", got)
	}
	if got := ratelimit.Interval(8*time.Second, 12*time.Second); got != 12*time.Second {
		t.Errorf("crawl-delay = %v", got)
	}
}

func TestMemorySpacesRequestsPerHost(t *testing.T) {
	clk := clock.NewFake(start)
	limiter := ratelimit.NewMemory(clk)
	ctx := context.Background()

	release, err := limiter.Acquire(ctx, "www.a.test", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	clk.Advance(2 * time.Second)
	release()
	release() // idempotent

	// A different host is independent.
	other, err := limiter.Acquire(ctx, "cdn.a.test", 5*time.Second)
	if err != nil || len(clk.Sleeps()) != 0 {
		t.Fatalf("other host waited: %v %v", err, clk.Sleeps())
	}
	other()

	before := clk.Now()
	second, err := limiter.Acquire(ctx, "www.a.test", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if waited := clk.Now().Sub(before); waited != 5*time.Second {
		t.Fatalf("waited %v, want 5s after previous finish", waited)
	}
	second()

	// Intervals below the floor are raised to MinInterval; longer ones apply.
	before = clk.Now()
	third, _ := limiter.Acquire(ctx, "www.a.test", time.Second)
	if waited := clk.Now().Sub(before); waited != 5*time.Second {
		t.Fatalf("floor not applied: %v", waited)
	}
	third()
}

func TestMemoryAllowsOneInFlight(t *testing.T) {
	clk := clock.NewFake(start)
	limiter := ratelimit.NewMemory(clk)
	release, err := limiter.Acquire(context.Background(), "www.a.test", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	acquired := make(chan struct{})
	go func() {
		next, err := limiter.Acquire(context.Background(), "www.a.test", 5*time.Second)
		if err == nil {
			next()
		}
		close(acquired)
	}()
	select {
	case <-acquired:
		t.Fatal("second request acquired while first in flight")
	case <-time.After(50 * time.Millisecond):
	}
	release()
	select {
	case <-acquired:
	case <-time.After(2 * time.Second):
		t.Fatal("second request never acquired")
	}

	held, _ := limiter.Acquire(context.Background(), "www.b.test", 5*time.Second)
	defer held()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := limiter.Acquire(ctx, "www.b.test", 5*time.Second); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled acquire err = %v", err)
	}
}

func TestMemoryDeferNeverMovesEarlier(t *testing.T) {
	clk := clock.NewFake(start)
	limiter := ratelimit.NewMemory(clk)
	ctx := context.Background()
	release, _ := limiter.Acquire(ctx, "www.a.test", 5*time.Second)
	release()
	if err := limiter.Defer(ctx, "www.a.test", 12*time.Second); err != nil {
		t.Fatal(err)
	}
	_ = limiter.Defer(ctx, "www.a.test", time.Second) // shorter: ignored
	before := clk.Now()
	next, _ := limiter.Acquire(ctx, "www.a.test", 5*time.Second)
	next()
	if waited := clk.Now().Sub(before); waited != 12*time.Second {
		t.Fatalf("waited %v, want deferred 12s", waited)
	}
}
