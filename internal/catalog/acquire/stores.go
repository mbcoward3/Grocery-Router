package acquire

import (
	"context"
	"sync"
	"time"

	"github.com/mbcoward3/grocery-router/internal/catalog/safefetch"
)

// Kind is the type of request an attempt made.
type Kind string

// Request kinds.
const (
	KindPage    Kind = "page"
	KindSitemap Kind = "sitemap"
	KindRobots  Kind = "robots"
)

// Outcome is the result of one attempt or one logical fetch.
type Outcome string

// Outcomes.
const (
	// OutcomeFetched means a 2xx body was accepted.
	OutcomeFetched Outcome = "fetched"
	// OutcomeRetryable means a timeout or 5xx that may be retried.
	OutcomeRetryable Outcome = "retryable"
	// OutcomeFailed means a final failure: a 4xx other than 403/429, a
	// non-retried transport error, or retries exhausted.
	OutcomeFailed Outcome = "failed"
	// OutcomeRejected means the fetch policy refused the request or body
	// (unsafe destination, off-host redirect, too large, wrong media type).
	OutcomeRejected Outcome = "rejected"
	// OutcomeDisallowed means robots.txt disallows the URL.
	OutcomeDisallowed Outcome = "disallowed"
	// OutcomePaused means the site is (now) paused; no request was made or
	// the response caused the pause.
	OutcomePaused Outcome = "paused"
)

// Attempt is one recorded attempt. Every attempt, including ones refused
// before reaching the network, is reported to the AttemptRecorder.
type Attempt struct {
	SiteID       string
	Kind         Kind
	URL          string
	Number       int
	StartedAt    time.Time
	FinishedAt   time.Time
	Status       int
	FinalURL     string
	Redirects    []safefetch.Redirect
	Outcome      Outcome
	ErrorKind    safefetch.ErrorKind
	Error        string
	DecodedBytes int
}

// AttemptRecorder persists attempts (ingest.fetch_attempts).
type AttemptRecorder interface {
	RecordAttempt(ctx context.Context, attempt Attempt) error
}

// PauseReason says why a site was paused.
type PauseReason string

// Pause reasons.
const (
	PauseForbidden         PauseReason = "http_403"
	PauseTooManyRequests   PauseReason = "http_429"
	PauseBotChallenge      PauseReason = "bot_challenge"
	PauseRobotsUnavailable PauseReason = "robots_unavailable"
)

// Pause is a persistent stop on a site until an operator resumes it.
type Pause struct {
	SiteID   string
	Reason   PauseReason
	URL      string
	Status   int
	Detail   string
	PausedAt time.Time
}

// PauseStore persists site pauses (ingest.site_pauses). Pause keeps an
// existing active pause unchanged; Resume clears it (catalog-site-resume).
type PauseStore interface {
	ActivePause(ctx context.Context, siteID string) (Pause, bool, error)
	Pause(ctx context.Context, pause Pause) error
	Resume(ctx context.Context, siteID string) error
}

// MemoryRecorder is an in-memory AttemptRecorder.
type MemoryRecorder struct {
	mu       sync.Mutex
	attempts []Attempt
}

// RecordAttempt appends attempt.
func (recorder *MemoryRecorder) RecordAttempt(_ context.Context, attempt Attempt) error {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	recorder.attempts = append(recorder.attempts, attempt)
	return nil
}

// Attempts returns a copy of the recorded attempts.
func (recorder *MemoryRecorder) Attempts() []Attempt {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	return append([]Attempt(nil), recorder.attempts...)
}

// MemoryPauses is an in-memory PauseStore.
type MemoryPauses struct {
	mu     sync.Mutex
	active map[string]Pause
}

// NewMemoryPauses returns an empty MemoryPauses.
func NewMemoryPauses() *MemoryPauses { return &MemoryPauses{active: map[string]Pause{}} }

// ActivePause returns the site's active pause.
func (pauses *MemoryPauses) ActivePause(_ context.Context, siteID string) (Pause, bool, error) {
	pauses.mu.Lock()
	defer pauses.mu.Unlock()
	pause, ok := pauses.active[siteID]
	return pause, ok, nil
}

// Pause records pause unless the site is already paused.
func (pauses *MemoryPauses) Pause(_ context.Context, pause Pause) error {
	pauses.mu.Lock()
	defer pauses.mu.Unlock()
	if _, ok := pauses.active[pause.SiteID]; !ok {
		pauses.active[pause.SiteID] = pause
	}
	return nil
}

// Resume clears the site's pause.
func (pauses *MemoryPauses) Resume(_ context.Context, siteID string) error {
	pauses.mu.Lock()
	defer pauses.mu.Unlock()
	delete(pauses.active, siteID)
	return nil
}
