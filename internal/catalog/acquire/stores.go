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

// Attempt is one network attempt. Requests refused before reaching the
// network (site paused, robots disallow) are not attempts and are not
// recorded; policy refusals of the initial URL are recorded as rejected.
type Attempt struct {
	SiteID string
	Kind   Kind
	// SourcePageID identifies the queued page for page attempts; 0 otherwise.
	SourcePageID int64
	URL          string
	Number       int
	StartedAt    time.Time
	FinishedAt   time.Time
	Status       int
	FinalURL     string
	Redirects    []safefetch.Redirect
	Outcome      Outcome
	ErrorKind    safefetch.ErrorKind
	BotChallenge bool
	Error        string
	DecodedBytes int
}

// AttemptRecorder persists attempts (ingest.fetch_attempts).
// RecordAttempt returns the stored attempt's ID.
type AttemptRecorder interface {
	RecordAttempt(ctx context.Context, attempt Attempt) (int64, error)
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
	SiteID string
	Reason PauseReason
	URL    string
	Status int
	// Detail is a non-empty human-readable explanation.
	Detail string
	// AttemptID is the attempt that caused the pause, when known.
	AttemptID int64
	PausedAt  time.Time
}

// PauseStore persists site pauses (ingest.site_pauses). Pause keeps an
// existing active pause unchanged; Resume closes the open pause with an
// operator note (catalog-site-resume) and reports whether one was open.
type PauseStore interface {
	ActivePause(ctx context.Context, siteID string) (Pause, bool, error)
	Pause(ctx context.Context, pause Pause) error
	Resume(ctx context.Context, siteID, note string) (bool, error)
}

// MemoryRecorder is an in-memory AttemptRecorder.
type MemoryRecorder struct {
	mu       sync.Mutex
	attempts []Attempt
}

// RecordAttempt appends attempt and returns its 1-based position.
func (recorder *MemoryRecorder) RecordAttempt(_ context.Context, attempt Attempt) (int64, error) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	recorder.attempts = append(recorder.attempts, attempt)
	return int64(len(recorder.attempts)), nil
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
func (pauses *MemoryPauses) Resume(_ context.Context, siteID, _ string) (bool, error) {
	pauses.mu.Lock()
	defer pauses.mu.Unlock()
	_, ok := pauses.active[siteID]
	delete(pauses.active, siteID)
	return ok, nil
}
