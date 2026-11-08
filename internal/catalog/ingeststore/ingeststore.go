// Package ingeststore implements the catalog acquisition store interfaces on
// the ingest schema (internal/database/migrations/00004_catalog_ingest.sql)
// through the sqlc queries in internal/store.
package ingeststore

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mbcoward3/grocery-router/internal/catalog/acquire"
	"github.com/mbcoward3/grocery-router/internal/catalog/clock"
	"github.com/mbcoward3/grocery-router/internal/catalog/evidence"
	"github.com/mbcoward3/grocery-router/internal/catalog/extract"
	"github.com/mbcoward3/grocery-router/internal/catalog/ratelimit"
	"github.com/mbcoward3/grocery-router/internal/catalog/safefetch"
	"github.com/mbcoward3/grocery-router/internal/store"
)

// storedURLPattern mirrors the ingest schema's URL CHECK constraints.
var storedURLPattern = regexp.MustCompile(`^https://[a-z0-9.-]+/[^[:space:]]*$`)

// StorableURL reports whether raw satisfies the ingest schema's URL checks.
func StorableURL(raw string) bool { return storedURLPattern.MatchString(raw) }

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func nullInt64(value int64) sql.NullInt64 {
	return sql.NullInt64{Int64: value, Valid: value != 0}
}

// CleanText makes text storable in a PostgreSQL TEXT column (valid UTF-8,
// no NUL bytes).
func CleanText(value string) string {
	if !utf8.ValidString(value) {
		value = strings.ToValidUTF8(value, "�")
	}
	return strings.ReplaceAll(value, "\x00", "�")
}

// ---- Rate limiter -------------------------------------------------------

// Limiter is the PostgreSQL-backed ratelimit.Limiter over
// ingest.host_rate_limits, shared by every process and command.
type Limiter struct {
	Queries *store.Queries
	Clock   clock.Clock
	// Lease bounds how long a crashed holder can block a host.
	Lease time.Duration
	// Poll bounds how long a waiter sleeps before re-checking a held lease.
	Poll time.Duration
}

// NewLimiter returns a Limiter with production lease and poll settings.
func NewLimiter(db *sql.DB) *Limiter {
	return &Limiter{Queries: store.New(db), Clock: clock.Real{}, Lease: 5 * time.Minute, Poll: 250 * time.Millisecond}
}

// Acquire implements ratelimit.Limiter.
func (limiter *Limiter) Acquire(ctx context.Context, host string, interval time.Duration) (func(), error) {
	interval = max(interval, ratelimit.MinInterval)
	for {
		slot, err := limiter.Queries.AcquireIngestHostSlot(ctx, store.AcquireIngestHostSlotParams{
			Host: host, LeaseMs: limiter.Lease.Milliseconds(),
		})
		if err == nil {
			token := slot.LeaseToken.String
			released := false
			return func() {
				if released {
					return
				}
				released = true
				// Release even when the request's context was canceled.
				releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
				defer cancel()
				_, _ = limiter.Queries.ReleaseIngestHostSlot(releaseCtx, store.ReleaseIngestHostSlotParams{
					IntervalMs: interval.Milliseconds(), Host: host, LeaseToken: token,
				})
			}, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("acquire host slot %s: %w", host, err)
		}
		state, err := limiter.Queries.GetIngestHostRateLimit(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("read host slot %s: %w", host, err)
		}
		wait := state.NextAllowedAt.Sub(state.DatabaseNow)
		if state.LeaseExpiresAt.Valid && state.LeaseExpiresAt.Time.After(state.DatabaseNow) {
			// Held by someone else: its release time is unknown, so poll.
			wait = max(wait, limiter.Poll)
		}
		wait = max(wait, 10*time.Millisecond)
		if err := limiter.Clock.Sleep(ctx, wait); err != nil {
			return nil, err
		}
	}
}

// Defer implements ratelimit.Limiter.
func (limiter *Limiter) Defer(ctx context.Context, host string, d time.Duration) error {
	state, err := limiter.Queries.GetIngestHostRateLimit(ctx, host)
	now := limiter.Clock.Now()
	if err == nil {
		now = state.DatabaseNow
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("read host slot %s: %w", host, err)
	}
	if err := limiter.Queries.DeferIngestHost(ctx, store.DeferIngestHostParams{Host: host, NotBefore: now.Add(d)}); err != nil {
		return fmt.Errorf("defer host %s: %w", host, err)
	}
	return nil
}

// ---- Attempts and pauses ------------------------------------------------

// Recorder is the acquire.AttemptRecorder over ingest.fetch_attempts.
type Recorder struct {
	Queries *store.Queries
	RunID   int64
}

// AttemptOutcome maps an attempt to the schema's outcome vocabulary.
func AttemptOutcome(attempt acquire.Attempt) string {
	switch attempt.ErrorKind {
	case safefetch.KindTimeout:
		return "timeout"
	case safefetch.KindPolicy, safefetch.KindMediaType:
		return "policy_rejected"
	case safefetch.KindTooLarge:
		return "body_too_large"
	case "":
	default:
		return "network_error"
	}
	switch {
	case attempt.BotChallenge:
		return "bot_challenge"
	case attempt.Status >= 200 && attempt.Status <= 299:
		return "success"
	default:
		return "http_status"
	}
}

// RecordAttempt implements acquire.AttemptRecorder.
func (recorder *Recorder) RecordAttempt(ctx context.Context, attempt acquire.Attempt) (int64, error) {
	params := store.InsertIngestFetchAttemptParams{
		RunID:        nullInt64(recorder.RunID),
		SiteID:       attempt.SiteID,
		SourcePageID: nullInt64(attempt.SourcePageID),
		Kind:         string(attempt.Kind),
		RequestedUrl: attempt.URL,
		Outcome:      AttemptOutcome(attempt),
		StartedAt:    attempt.StartedAt,
		FinishedAt:   attempt.FinishedAt,
	}
	if params.FinishedAt.Before(params.StartedAt) {
		params.FinishedAt = params.StartedAt
	}
	if attempt.Status >= 100 && attempt.Status <= 599 {
		params.HttpStatus = sql.NullInt32{Int32: int32(attempt.Status), Valid: true}
	}
	message := attempt.Error
	if StorableURL(attempt.FinalURL) {
		params.FinalUrl = nullString(attempt.FinalURL)
	} else if attempt.FinalURL != "" {
		message = strings.TrimSpace(message + " (final URL " + attempt.FinalURL + ")")
	}
	params.Error = nullString(CleanText(message))
	id, err := recorder.Queries.InsertIngestFetchAttempt(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("insert fetch attempt: %w", err)
	}
	return id, nil
}

// Pauses is the acquire.PauseStore over ingest.site_pauses.
type Pauses struct {
	Queries *store.Queries
	RunID   int64
}

// ActivePause implements acquire.PauseStore.
func (pauses *Pauses) ActivePause(ctx context.Context, siteID string) (acquire.Pause, bool, error) {
	row, err := pauses.Queries.GetOpenIngestSitePause(ctx, siteID)
	if errors.Is(err, sql.ErrNoRows) {
		return acquire.Pause{}, false, nil
	}
	if err != nil {
		return acquire.Pause{}, false, fmt.Errorf("read open pause for %s: %w", siteID, err)
	}
	return acquire.Pause{
		SiteID: row.SiteID, Reason: acquire.PauseReason(row.Reason), Detail: row.Detail,
		AttemptID: row.FetchAttemptID.Int64, PausedAt: row.PausedAt,
	}, true, nil
}

// Pause implements acquire.PauseStore; an already-paused site keeps its
// first open pause.
func (pauses *Pauses) Pause(ctx context.Context, pause acquire.Pause) error {
	detail := strings.TrimSpace(CleanText(pause.Detail))
	if detail == "" {
		detail = string(pause.Reason)
	}
	if _, err := pauses.Queries.PauseIngestSite(ctx, store.PauseIngestSiteParams{
		SiteID: pause.SiteID, Reason: string(pause.Reason), Detail: detail,
		FetchAttemptID: nullInt64(pause.AttemptID), RunID: nullInt64(pauses.RunID),
	}); err != nil {
		return fmt.Errorf("pause site %s: %w", pause.SiteID, err)
	}
	return nil
}

// Resume implements acquire.PauseStore.
func (pauses *Pauses) Resume(ctx context.Context, siteID, note string) (bool, error) {
	note = strings.TrimSpace(note)
	if note == "" {
		return false, errors.New("a resume note is required")
	}
	rows, err := pauses.Queries.ResumeIngestSite(ctx, store.ResumeIngestSiteParams{ResumeNote: CleanText(note), SiteID: siteID})
	if err != nil {
		return false, fmt.Errorf("resume site %s: %w", siteID, err)
	}
	return rows > 0, nil
}

// ---- Evidence and extraction ---------------------------------------------

// Evidence is the evidence.Store over ingest.evidence.
type Evidence struct {
	Queries *store.Queries
}

// Insert implements evidence.Store.
func (evidenceStore *Evidence) Insert(ctx context.Context, record evidence.Evidence) (evidence.Evidence, error) {
	params, err := evidenceParams(record)
	if err != nil {
		return evidence.Evidence{}, err
	}
	id, err := evidenceStore.Queries.InsertIngestEvidence(ctx, params)
	if err != nil {
		return evidence.Evidence{}, fmt.Errorf("insert evidence: %w", err)
	}
	record.ID = id
	return record, nil
}

// Get implements evidence.Store.
func (evidenceStore *Evidence) Get(ctx context.Context, id int64) (evidence.Evidence, error) {
	row, err := evidenceStore.Queries.GetIngestEvidence(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return evidence.Evidence{}, fmt.Errorf("%w: %d", evidence.ErrNotFound, id)
	}
	if err != nil {
		return evidence.Evidence{}, fmt.Errorf("read evidence %d: %w", id, err)
	}
	return FromRow(row)
}

func evidenceParams(record evidence.Evidence) (store.InsertIngestEvidenceParams, error) {
	digest, err := hex.DecodeString(record.BodySHA256)
	if err != nil || len(digest) != 32 {
		return store.InsertIngestEvidenceParams{}, fmt.Errorf("evidence body digest %q is not SHA-256 hex", record.BodySHA256)
	}
	redirects, err := json.Marshal(nonNilRedirects(record.Redirects))
	if err != nil {
		return store.InsertIngestEvidenceParams{}, fmt.Errorf("encode redirects: %w", err)
	}
	canonical := CleanText(record.CanonicalLink)
	if len(canonical) > 4096 {
		canonical = canonical[:4096]
		for !utf8.ValidString(canonical) {
			canonical = canonical[:len(canonical)-1]
		}
	}
	return store.InsertIngestEvidenceParams{
		FetchAttemptID: record.FetchAttemptID, SiteID: record.SiteID, SourcePageID: record.SourcePageID,
		RequestedUrl: record.RequestedURL, FinalUrl: record.FinalURL, RedirectChain: redirects,
		HttpStatus: int32(record.Status), FetchedAt: record.FetchedAt, MediaType: record.MediaType,
		BodySha256: digest, BodySize: int32(record.DecodedBytes), BodyEncoding: record.BodyEncoding,
		BodyCompressed: record.CompressedBody, CanonicalLink: nullString(canonical),
	}, nil
}

func nonNilRedirects(redirects []safefetch.Redirect) []safefetch.Redirect {
	if redirects == nil {
		return []safefetch.Redirect{}
	}
	return redirects
}

// FromRow converts a stored evidence row back into evidence.Evidence.
func FromRow(row store.IngestEvidence) (evidence.Evidence, error) {
	var redirects []safefetch.Redirect
	if err := json.Unmarshal(row.RedirectChain, &redirects); err != nil {
		return evidence.Evidence{}, fmt.Errorf("decode redirects of evidence %d: %w", row.ID, err)
	}
	return evidence.Evidence{
		ID: row.ID, SiteID: row.SiteID, FetchAttemptID: row.FetchAttemptID, SourcePageID: row.SourcePageID,
		CanonicalLink: row.CanonicalLink.String, RequestedURL: row.RequestedUrl, FinalURL: row.FinalUrl,
		Redirects: redirects, Status: int(row.HttpStatus), MediaType: row.MediaType, FetchedAt: row.FetchedAt,
		DecodedBytes: int64(row.BodySize), BodySHA256: hex.EncodeToString(row.BodySha256),
		BodyEncoding: row.BodyEncoding, CompressedBody: row.BodyCompressed,
	}, nil
}

// Extractions is the extract.Store over ingest.evidence_jsonld_blocks and
// ingest.source_nodes. Rows are immutable, so an extraction can be saved
// only once per evidence; replay compares instead (see stage.Replay).
type Extractions struct {
	Queries *store.Queries
	SiteID  string
}

// SaveExtraction implements extract.Store.
func (extractions *Extractions) SaveExtraction(ctx context.Context, evidenceID int64, result extract.Result) error {
	blockIDs := make(map[int]int64, len(result.Blocks))
	for _, block := range result.Blocks {
		params := store.InsertIngestJSONLDBlockParams{
			EvidenceID: evidenceID, Position: int32(block.Index), RawText: CleanText(block.Raw),
			ParseStatus: "parsed",
		}
		if block.ParseError != "" {
			params.ParseStatus, params.ParseError = "invalid_json", nullString(CleanText(block.ParseError))
		}
		id, err := extractions.Queries.InsertIngestJSONLDBlock(ctx, params)
		if err != nil {
			return fmt.Errorf("insert JSON-LD block %d: %w", block.Index, err)
		}
		blockIDs[block.Index] = id
	}
	for _, node := range result.Nodes {
		digest, err := hex.DecodeString(node.ContentDigest)
		if err != nil {
			return fmt.Errorf("node digest %q: %w", node.ContentDigest, err)
		}
		if _, err := extractions.Queries.InsertIngestSourceNode(ctx, store.InsertIngestSourceNodeParams{
			SiteID: extractions.SiteID, EvidenceID: evidenceID, JsonldBlockID: blockIDs[node.BlockIndex],
			Location: node.Location, NodeID: nullString(strings.TrimSpace(node.ID)),
			NormalizationVersion: extract.NormalizationVersion, ContentDigest: digest, RawNode: node.Raw,
		}); err != nil {
			return fmt.Errorf("insert source node %d%s: %w", node.BlockIndex, node.Location, err)
		}
	}
	return nil
}

// SaveFetchedPage stores evidence, its extraction, and marks the page fetched
// in one transaction, so an interrupted run never leaves a half-stored page.
func SaveFetchedPage(ctx context.Context, db *sql.DB, record evidence.Evidence, result extract.Result) (int64, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin fetched page transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	queries := store.New(tx)
	stored, err := (&Evidence{Queries: queries}).Insert(ctx, record)
	if err != nil {
		return 0, err
	}
	if err := (&Extractions{Queries: queries, SiteID: record.SiteID}).SaveExtraction(ctx, stored.ID, result); err != nil {
		return 0, err
	}
	if err := queries.MarkIngestSourcePageFetched(ctx, store.MarkIngestSourcePageFetchedParams{
		EvidenceID: sql.NullInt64{Int64: stored.ID, Valid: true}, ID: record.SourcePageID,
	}); err != nil {
		return 0, fmt.Errorf("mark page %d fetched: %w", record.SourcePageID, err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit fetched page: %w", err)
	}
	return stored.ID, nil
}
