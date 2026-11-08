// Package ratelimit spaces requests per host (CATALOG_INGEST_SPEC.md §7.3):
// one request in flight per host, and each request starts at least the
// host's interval after the previous one finished.
package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/mbcoward3/grocery-router/internal/catalog/clock"
)

// MinInterval is the floor for every host's spacing.
const MinInterval = 5 * time.Second

// Interval returns the spacing for a host: the largest of MinInterval, the
// site's registry interval, and the host's robots Crawl-delay.
func Interval(siteInterval, crawlDelay time.Duration) time.Duration {
	return max(MinInterval, siteInterval, crawlDelay)
}

// Limiter grants exclusive, spaced access to a host. The database-backed
// implementation shares state across commands; Memory serves tests.
type Limiter interface {
	// Acquire blocks until host has no request in flight and interval has
	// elapsed since the previous request on host finished. The caller must
	// call release exactly once when its request is complete.
	Acquire(ctx context.Context, host string, interval time.Duration) (release func(), err error)
}

// Memory is an in-process Limiter.
type Memory struct {
	clock clock.Clock
	mu    sync.Mutex
	hosts map[string]*hostState
}

type hostState struct {
	inFlight    bool
	nextAllowed time.Time
	wake        chan struct{}
}

// NewMemory returns a Memory limiter using clk.
func NewMemory(clk clock.Clock) *Memory {
	return &Memory{clock: clk, hosts: map[string]*hostState{}}
}

// Acquire implements Limiter.
func (memory *Memory) Acquire(ctx context.Context, host string, interval time.Duration) (func(), error) {
	interval = max(interval, MinInterval)
	for {
		memory.mu.Lock()
		state := memory.hosts[host]
		if state == nil {
			state = &hostState{wake: make(chan struct{})}
			memory.hosts[host] = state
		}
		if state.inFlight {
			wake := state.wake
			memory.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-wake:
			}
			continue
		}
		wait := state.nextAllowed.Sub(memory.clock.Now())
		if wait <= 0 {
			state.inFlight = true
			memory.mu.Unlock()
			var once sync.Once
			return func() { once.Do(func() { memory.release(host, interval) }) }, nil
		}
		memory.mu.Unlock()
		if err := memory.clock.Sleep(ctx, wait); err != nil {
			return nil, err
		}
	}
}

func (memory *Memory) release(host string, interval time.Duration) {
	memory.mu.Lock()
	defer memory.mu.Unlock()
	state := memory.hosts[host]
	state.inFlight = false
	state.nextAllowed = memory.clock.Now().Add(interval)
	close(state.wake)
	state.wake = make(chan struct{})
}
