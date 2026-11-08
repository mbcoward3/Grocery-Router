// Package acquire applies the politeness and failure policy of
// CATALOG_INGEST_SPEC.md §7.3–§7.4 around safefetch: robots per actual host,
// per-host rate limiting, bounded retries, persistent site pauses, and a
// recorded attempt for every try.
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

// GetPage fetches an HTML page for site.
func (client *Client) GetPage(ctx context.Context, site Site, rawURL string) (Result, error) {
	return client.get(ctx, site, KindPage, safefetch.PageProfile, rawURL)
}

// GetSitemap fetches a sitemap or sitemap index for site.
func (client *Client) GetSitemap(ctx context.Context, site Site, rawURL string) (Result, error) {
	return client.get(ctx, site, KindSitemap, safefetch.SitemapProfile, rawURL)
}

// get runs the failure policy. The returned error is reserved for
// infrastructure failures (stores, cancellation); fetch outcomes are in Result.
func (client *Client) get(ctx context.Context, site Site, kind Kind, profile safefetch.Profile, rawURL string) (Result, error) {
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
		var pause *Pause
		switch {
		case gateErr != nil && !errors.Is(gateErr, ErrDisallowed) && !errors.Is(gateErr, errRobotsPaused):
			return result, gateErr // store or cancellation failure inside the gate
		case safefetch.KindOf(fetchErr) == safefetch.KindCanceled:
			return result, fmt.Errorf("fetch %s: %w", rawURL, fetchErr)
		case errors.Is(gateErr, errRobotsPaused):
			result.Outcome, result.Err, result.Pause = OutcomePaused, gateErr, robotsPause
		case errors.Is(gateErr, ErrDisallowed):
			result.Outcome, result.Err = OutcomeDisallowed, gateErr
		default:
			result.Outcome, result.Err, pause = classify(site, response, fetchErr, client.Clock.Now().UTC())
		}
		if pause != nil {
			if err := client.Pauses.Pause(ctx, *pause); err != nil {
				return result, fmt.Errorf("pause %s: %w", site.ID, err)
			}
			result.Pause = pause
		}
		if err := client.record(ctx, site, kind, rawURL, attempt, result, fetchErr); err != nil {
			return result, err
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
func classify(site Site, response safefetch.Response, fetchErr error, now time.Time) (Outcome, error, *Pause) {
	newPause := func(reason PauseReason) *Pause {
		return &Pause{SiteID: site.ID, Reason: reason, URL: response.FinalURL, Status: response.Status, PausedAt: now}
	}
	if fetchErr != nil {
		switch safefetch.KindOf(fetchErr) {
		case safefetch.KindTimeout:
			return OutcomeRetryable, fetchErr, nil
		case safefetch.KindPolicy, safefetch.KindTooLarge, safefetch.KindMediaType:
			return OutcomeRejected, fetchErr, nil
		default:
			return OutcomeFailed, fetchErr, nil
		}
	}
	status := response.Status
	if IsBotChallenge(response.Header, response.Body) {
		return OutcomePaused, fmt.Errorf("HTTP %d bot challenge", status), newPause(PauseBotChallenge)
	}
	switch {
	case status >= 200 && status <= 299:
		return OutcomeFetched, nil, nil
	case status == 403:
		return OutcomePaused, fmt.Errorf("HTTP 403"), newPause(PauseForbidden)
	case status == 429:
		return OutcomePaused, fmt.Errorf("HTTP 429"), newPause(PauseTooManyRequests)
	case status >= 500 && status <= 599:
		return OutcomeRetryable, fmt.Errorf("HTTP %d", status), nil
	default:
		return OutcomeFailed, fmt.Errorf("HTTP %d", status), nil
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
	if safefetch.KindOf(fetchErr) == safefetch.KindCanceled {
		return robots.Policy{}, nil, fmt.Errorf("fetch %s: %w", robotsURL, fetchErr)
	}
	if safefetch.KindOf(fetchErr) == safefetch.KindGate {
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
	if outcome == OutcomePaused {
		detail := "robots.txt unavailable"
		if attemptErr != nil {
			detail = attemptErr.Error()
		}
		pause := &Pause{
			SiteID: site.ID, Reason: PauseRobotsUnavailable, URL: robotsURL,
			Status: response.Status, Detail: detail, PausedAt: now,
		}
		if err := client.Pauses.Pause(ctx, *pause); err != nil {
			return robots.Policy{}, nil, fmt.Errorf("pause %s: %w", site.ID, err)
		}
		result.Pause = pause
		if err := client.record(ctx, site, KindRobots, robotsURL, 1, result, fetchErr); err != nil {
			return robots.Policy{}, nil, err
		}
		return policy, pause, fmt.Errorf("%w: %s", errRobotsPaused, detail)
	}
	if err := client.record(ctx, site, KindRobots, robotsURL, 1, result, fetchErr); err != nil {
		return robots.Policy{}, nil, err
	}
	if err := client.Robots.Put(ctx, policy); err != nil {
		return robots.Policy{}, nil, fmt.Errorf("store robots policy for %s: %w", host, err)
	}
	return policy, nil, nil
}

func (client *Client) record(ctx context.Context, site Site, kind Kind, rawURL string, number int, result Result, fetchErr error) error {
	attempt := Attempt{
		SiteID: site.ID, Kind: kind, URL: rawURL, Number: number,
		StartedAt: result.Response.StartedAt, FinishedAt: result.Response.FinishedAt,
		Status: result.Response.Status, FinalURL: result.Response.FinalURL,
		Redirects: result.Response.Redirects, Outcome: result.Outcome,
		ErrorKind: safefetch.KindOf(fetchErr), DecodedBytes: len(result.Response.Body),
	}
	if result.Err != nil {
		attempt.Error = strings.TrimSpace(result.Err.Error())
	}
	if attempt.StartedAt.IsZero() {
		now := client.Clock.Now().UTC()
		attempt.StartedAt, attempt.FinishedAt = now, now
	}
	if err := client.Recorder.RecordAttempt(ctx, attempt); err != nil {
		return fmt.Errorf("record attempt for %s: %w", rawURL, err)
	}
	return nil
}
