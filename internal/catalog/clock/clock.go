// Package clock abstracts time for catalog acquisition so politeness waits
// can be tested without sleeping.
package clock

import (
	"context"
	"sync"
	"time"
)

// Clock reports the current time and waits.
type Clock interface {
	Now() time.Time
	// Sleep waits for d or until ctx ends, returning ctx's error in that case.
	Sleep(ctx context.Context, d time.Duration) error
}

// Real is the wall clock.
type Real struct{}

// Now returns the current wall-clock time.
func (Real) Now() time.Time { return time.Now() }

// Sleep waits for d or until ctx ends.
func (Real) Sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Fake is a manual clock whose Sleep advances time instantly and records the
// requested durations.
type Fake struct {
	mu     sync.Mutex
	now    time.Time
	sleeps []time.Duration
}

// NewFake returns a fake clock starting at start.
func NewFake(start time.Time) *Fake { return &Fake{now: start} }

// Now returns the fake time.
func (fake *Fake) Now() time.Time {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return fake.now
}

// Sleep advances the fake time by d without blocking.
func (fake *Fake) Sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if d > 0 {
		fake.now = fake.now.Add(d)
	}
	fake.sleeps = append(fake.sleeps, d)
	return nil
}

// Advance moves the fake time forward by d.
func (fake *Fake) Advance(d time.Duration) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.now = fake.now.Add(d)
}

// Sleeps returns a copy of every duration passed to Sleep.
func (fake *Fake) Sleeps() []time.Duration {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return append([]time.Duration(nil), fake.sleeps...)
}
