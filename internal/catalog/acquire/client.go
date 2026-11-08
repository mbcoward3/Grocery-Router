// Package acquire applies the politeness and failure policy of
// CATALOG_INGEST_SPEC.md §7.3–§7.4 around safefetch: robots per actual host,
// per-host rate limiting, bounded retries, persistent site pauses, and a
// recorded attempt for every network try.
package acquire

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/mbcoward3/grocery-router/internal/catalog/clock"
	"github.com/mbcoward3/grocery-router/internal/catalog/ratelimit"
	"github.com/mbcoward3/grocery-router/internal/catalog/robots"
	"github.com/mbcoward3/grocery-router/internal/catalog/safefetch"
)

// MaxAttempts is one try plus at most two retries.
const MaxAttempts = 3

// DefaultBackoff is the wait before each retry.
var DefaultBackoff = []time.Duration{10 * time.Second, 30 * time.Second}

// Site is the subset of a registry entry acquisition needs.
type Site struct {
	ID    string
	Hosts []string
	// MinInterval is the registry's optional stricter per-host interval.
	MinInterval time.Duration
}

// Fetcher performs one safe fetch (implemented by *safefetch.Client).
type Fetcher interface {
	Fetch(ctx context.Context, request safefetch.Request) (safefetch.Response, error)
}

// Client is the polite acquisition client.
type Client struct {
	Fetcher  Fetcher
	Limiter  ratelimit.Limiter
	Robots   robots.Cache
	Pauses   PauseStore
	Recorder AttemptRecorder
	Clock    clock.Clock
	// Backoff[i] is the wait before retry i+1; DefaultBackoff when nil.
	Backoff []time.Duration
}

// Result is the outcome of one logical fetch.
type Result struct {
	Outcome  Outcome
	Response safefetch.Response
	Attempts int
	// AttemptID is the recorder's ID for the last recorded attempt, or 0
	// when no network attempt was made.
	AttemptID int64
	// Pause is set when the site is paused, whether by this fetch or earlier.
	Pause *Pause
	// Err describes a non-fetched outcome.
	Err error
}

// Errors carried in Result.Err.
var (
	ErrSitePaused   = errors.New("site is paused")
	ErrDisallowed   = errors.New("disallowed by robots.txt")
	errRobotsPaused = errors.New("robots.txt unavailable; site paused")
)

// GetPage fetches the HTML page sourcePageID at rawURL for site.
func (client *Client) GetPage(ctx context.Context, site Site, sourcePageID int64, rawURL string) (Result, error) {
	return client.get(ctx, site, KindPage, sourcePageID, safefetch.PageProfile, rawURL)
}

// GetSitemap fetches a sitemap or sitemap index for site.
func (client *Client) GetSitemap(ctx context.Context, site Site, rawURL string) (Result, error) {
	return client.get(ctx, site, KindSitemap, 0, safefetch.SitemapProfile, rawURL)
}

// get runs the failure policy. The returned error is reserved for
// infrastructure failures (stores, cancellation); fetch outcomes are in Result.
func (client *Client) get(ctx context.Context, site Site, kind Kind, sourcePageID int64, profile safefetch.Profile, rawURL string) (Result, error) {
	if pause, paused, err := client.Pauses.ActivePause(ctx, site.ID); err != nil {
		return Result{}, fmt.Errorf("read pause for %s: %w", site.ID, err)
	} else if paused {
		return Result{Outcome: OutcomePaused, Pause: &pause, Err: ErrSitePaused}, nil
	}
	backoff := client.Backoff
	if backoff == nil {
		backoff = DefaultBackoff
	}
	var result Result
	for attempt := 1; attempt <= MaxAttempts; attempt++ {
		request := safefetch.Request{URL: rawURL, Profile: profile, AllowedHosts: site.Hosts}
		var gateErr error
		var robotsPause *Pause
		request.Gate = func(ctx context.Context, hop *url.URL) (func(), error) {
			release, pause, err := client.pageGate(ctx, site, hop)
			robotsPause, gateErr = pause, err
			return release, err
		}
		response, fetchErr := client.Fetcher.Fetch(ctx, request)
		result = Result{Response: response, Attempts: attempt}
		switch {
		case gateErr != nil && !errors.Is(gateErr, ErrDisallowed) && !errors.Is(gateErr, errRobotsPaused):
			return result, gateErr // store or cancellation failure inside the gate
		case safefetch.KindOf(fetchErr) == safefetch.KindCanceled:
			return result, fmt.Errorf("fetch %s: %w", rawURL, fetchErr)
		case errors.Is(gateErr, errRobotsPaused):
			// No request reached the page; the robots attempt was recorded.
			result.Outcome, result.Err, result.Pause = OutcomePaused, gateErr, robotsPause
			return result, nil
		case errors.Is(gateErr, ErrDisallowed):
			result.Outcome, result.Err = OutcomeDisallowed, gateErr
			return result, nil
		}
		var pause *Pause
		result.Outcome, pause, result.Err = classify(site, response, fetchErr, client.Clock.Now().UTC())
		attemptID, err := client.record(ctx, site, kind, sourcePageID, rawURL, attempt, result, fetchErr)
		if err != nil {
			return result, err
		}
		result.AttemptID = attemptID
		if pause != nil {
			pause.AttemptID = attemptID
			if err := client.Pauses.Pause(ctx, *pause); err != nil {
				return result, fmt.Errorf("pause %s: %w", site.ID, err)
			}
			result.Pause = pause
		}
		if result.Outcome != OutcomeRetryable {
			return result, nil
		}
		if attempt < MaxAttempts {
			if err := client.Clock.Sleep(ctx, backoff[min(attempt-1, len(backoff)-1)]); err != nil {
				return result, fmt.Errorf("retry backoff: %w", err)
			}
		}
	}
	result.Outcome = OutcomeFailed
	result.Err = fmt.Errorf("retries exhausted: %w", result.Err)
	return result, nil
}

// classify maps a completed fetch to an outcome and an optional pause.
// The returned error describes why the outcome is not OutcomeFetched.
func classify(site Site, response safefetch.Response, fetchErr error, now time.Time) (Outcome, *Pause, error) {
	newPause := func(reason PauseReason, detail string) *Pause {
		return &Pause{
			SiteID: site.ID, Reason: reason, URL: response.FinalURL, Status: response.Status,
			Detail: detail, PausedAt: now,
		}
	}
	if fetchErr != nil {
		switch safefetch.KindOf(fetchErr) {
		case safefetch.KindTimeout:
			return OutcomeRetryable, nil, fetchErr
		case safefetch.KindPolicy, safefetch.KindTooLarge, safefetch.KindMediaType:
			return OutcomeRejected, nil, fetchErr
		default:
			return OutcomeFailed, nil, fetchErr
		}
	}
	status := response.Status
	if IsBotChallenge(response.Header, response.Body) {
		detail := fmt.Sprintf("HTTP %d bot challenge at %s", status, response.FinalURL)
		return OutcomePaused, newPause(PauseBotChallenge, detail), errors.New(detail)
	}
	switch {
	case status >= 200 && status <= 299:
		return OutcomeFetched, nil, nil
	case status == 403, status == 429:
		reason := PauseForbidden
		if status == 429 {
			reason = PauseTooManyRequests
		}
		detail := fmt.Sprintf("HTTP %d at %s", status, response.FinalURL)
		return OutcomePaused, newPause(reason, detail), errors.New(detail)
	case status >= 500 && status <= 599:
		return OutcomeRetryable, nil, fmt.Errorf("HTTP %d", status)
	default:
		return OutcomeFailed, nil, fmt.Errorf("HTTP %d", status)
	}
}

// pageGate enforces robots for hop's host, then rate limits the hop.
func (client *Client) pageGate(ctx context.Context, site Site, hop *url.URL) (func(), *Pause, error) {
	host := safefetch.NormalizeHost(hop.Hostname())
	policy, pause, err := client.robotsPolicy(ctx, site, host)
	if err != nil {
		return nil, pause, err
	}
	if !policy.Allowed(hop) {
		return nil, nil, fmt.Errorf("%w: %s", ErrDisallowed, hop)
	}
	release, err := client.Limiter.Acquire(ctx, host, ratelimit.Interval(site.MinInterval, policy.CrawlDelay))
	if err != nil {
		return nil, nil, fmt.Errorf("rate limit %s: %w", host, err)
	}
	return release, nil, nil
}

// robotsPolicy returns a fresh policy for host, fetching robots.txt when the
// cache is stale. An unavailable robots.txt pauses the site and is not cached.
func (client *Client) robotsPolicy(ctx context.Context, site Site, host string) (robots.Policy, *Pause, error) {
	cached, ok, err := client.Robots.Get(ctx, host)
	if err != nil {
		return robots.Policy{}, nil, fmt.Errorf("read robots cache for %s: %w", host, err)
	}
	if ok && cached.Fresh(client.Clock.Now()) {
		return cached, nil, nil
	}
	robotsURL := "https://" + host + "/robots.txt"
	request := safefetch.Request{
		URL: robotsURL, Profile: safefetch.RobotsProfile, AllowedHosts: site.Hosts,
		Gate: func(ctx context.Context, hop *url.URL) (func(), error) {
			return client.Limiter.Acquire(ctx, safefetch.NormalizeHost(hop.Hostname()), ratelimit.Interval(site.MinInterval, 0))
		},
	}
	response, fetchErr := client.Fetcher.Fetch(ctx, request)
	if kind := safefetch.KindOf(fetchErr); kind == safefetch.KindCanceled || kind == safefetch.KindGate {
		return robots.Policy{}, nil, fmt.Errorf("fetch %s: %w", robotsURL, fetchErr)
	}
	now := client.Clock.Now().UTC()
	policy := robots.Policy{Host: host, Status: response.Status, FetchedAt: now}
	outcome := OutcomeFetched
	switch status := response.Status; {
	case fetchErr != nil, status == 429, status >= 500:
		policy.Mode = robots.ModeDisallowAll
		outcome = OutcomePaused
	case status >= 200 && status <= 299:
		policy.Mode = robots.ModeRules
		policy.Rules, policy.CrawlDelay = robots.Parse(response.Body, safefetch.ProductToken)
	case status >= 400:
		policy.Mode = robots.ModeAllowAll
	default:
		policy.Mode = robots.ModeDisallowAll
		outcome = OutcomePaused
	}
	attemptErr := fetchErr
	if outcome == OutcomePaused && attemptErr == nil {
		attemptErr = fmt.Errorf("robots.txt HTTP %d", response.Status)
	}
	result := Result{Outcome: outcome, Response: response, Attempts: 1, Err: attemptErr}
	attemptID, err := client.record(ctx, site, KindRobots, 0, robotsURL, 1, result, fetchErr)
	if err != nil {
		return robots.Policy{}, nil, err
	}
	if outcome == OutcomePaused {
		pause := &Pause{
			SiteID: site.ID, Reason: PauseRobotsUnavailable, URL: robotsURL, Status: response.Status,
			Detail: robotsURL + ": " + attemptErr.Error(), AttemptID: attemptID, PausedAt: now,
		}
		if err := client.Pauses.Pause(ctx, *pause); err != nil {
			return robots.Policy{}, nil, fmt.Errorf("pause %s: %w", site.ID, err)
		}
		return policy, pause, fmt.Errorf("%w: %s", errRobotsPaused, pause.Detail)
	}
	// The robots request was spaced before its Crawl-delay was known.
	if delay := ratelimit.Interval(site.MinInterval, policy.CrawlDelay); delay > ratelimit.Interval(site.MinInterval, 0) {
		if err := client.Limiter.Defer(ctx, host, delay); err != nil {
			return robots.Policy{}, nil, fmt.Errorf("apply crawl-delay for %s: %w", host, err)
		}
	}
	if err := client.Robots.Put(ctx, policy); err != nil {
		return robots.Policy{}, nil, fmt.Errorf("store robots policy for %s: %w", host, err)
	}
	return policy, nil, nil
}

func (client *Client) record(ctx context.Context, site Site, kind Kind, sourcePageID int64, rawURL string, number int, result Result, fetchErr error) (int64, error) {
	attempt := Attempt{
		SiteID: site.ID, Kind: kind, SourcePageID: sourcePageID, URL: rawURL, Number: number,
		StartedAt: result.Response.StartedAt, FinishedAt: result.Response.FinishedAt,
		Status: result.Response.Status, FinalURL: result.Response.FinalURL,
		Redirects: result.Response.Redirects, Outcome: result.Outcome,
		ErrorKind: safefetch.KindOf(fetchErr), DecodedBytes: len(result.Response.Body),
	}
	if fetchErr == nil && IsBotChallenge(result.Response.Header, result.Response.Body) {
		attempt.BotChallenge = true
	}
	if result.Err != nil {
		attempt.Error = strings.TrimSpace(result.Err.Error())
	}
	if attempt.StartedAt.IsZero() {
		now := client.Clock.Now().UTC()
		attempt.StartedAt, attempt.FinishedAt = now, now
	}
	id, err := client.Recorder.RecordAttempt(ctx, attempt)
	if err != nil {
		return 0, fmt.Errorf("record attempt for %s: %w", rawURL, err)
	}
	return id, nil
}
