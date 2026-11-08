// Package ratelimit spaces requests per host (CATALOG_INGEST_SPEC.md §7.3):
// one request in flight per host, and a next-allowed time that each finished
// request pushes at least its interval into the future.
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
	// Acquire blocks until host has no request in flight and its
	// next-allowed time has passed. Calling release ends the request and
	// sets the next-allowed time to at least interval (never below
	// MinInterval) from then. Extra release calls are ignored.
	Acquire(ctx context.Context, host string, interval time.Duration) (release func(), err error)
	// Defer moves host's next-allowed time to at least d from now; it never
	// moves it earlier.
	Defer(ctx context.Context, host string, d time.Duration) error
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

func (memory *Memory) state(host string) *hostState {
	state := memory.hosts[host]
	if state == nil {
		state = &hostState{wake: make(chan struct{})}
		memory.hosts[host] = state
	}
	return state
}

// Acquire implements Limiter.
func (memory *Memory) Acquire(ctx context.Context, host string, interval time.Duration) (func(), error) {
	interval = max(interval, MinInterval)
	for {
		memory.mu.Lock()
		state := memory.state(host)
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

// Defer implements Limiter.
func (memory *Memory) Defer(_ context.Context, host string, d time.Duration) error {
	memory.mu.Lock()
	defer memory.mu.Unlock()
	state := memory.state(host)
	if next := memory.clock.Now().Add(d); next.After(state.nextAllowed) {
		state.nextAllowed = next
	}
	return nil
}

func (memory *Memory) release(host string, interval time.Duration) {
	memory.mu.Lock()
	defer memory.mu.Unlock()
	state := memory.hosts[host]
	state.inFlight = false
	if next := memory.clock.Now().Add(interval); next.After(state.nextAllowed) {
		state.nextAllowed = next
	}
	close(state.wake)
	state.wake = make(chan struct{})
}
