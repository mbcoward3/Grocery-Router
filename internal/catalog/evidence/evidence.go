// Package evidence turns an accepted fetch into immutable evidence with a
// gzip-compressed body, so extraction can replay without network access.
package evidence

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/mbcoward3/grocery-router/internal/catalog/safefetch"
)

// BodyEncodingGzip is the only stored body encoding.
const BodyEncodingGzip = "gzip"

// Evidence is one immutable fetch. A re-fetch creates new evidence.
type Evidence struct {
	// ID is assigned by the Store on insert.
	ID     int64
	SiteID string
	// FetchAttemptID and SourcePageID link evidence to the attempt that
	// produced it and the queued page it answers; set by the caller.
	FetchAttemptID int64
	SourcePageID   int64
	// CanonicalLink is the page's first rel=canonical href exactly as found,
	// valid or not; empty when absent.
	CanonicalLink string
	RequestedURL  string
	FinalURL      string
	Redirects     []safefetch.Redirect
	Status        int
	MediaType     string
	ETag          string
	LastModified  string
	FetchedAt     time.Time
	// DecodedBytes and BodySHA256 describe the decoded (uncompressed) body.
	DecodedBytes   int64
	BodySHA256     string
	BodyEncoding   string
	CompressedBody []byte
}

// New builds evidence from a successful response, compressing its body.
func New(siteID string, response safefetch.Response) (Evidence, error) {
	if response.Status < 200 || response.Status > 299 {
		return Evidence{}, fmt.Errorf("evidence requires a 2xx response, got %d", response.Status)
	}
	var compressed bytes.Buffer
	writer, err := gzip.NewWriterLevel(&compressed, gzip.BestCompression)
	if err != nil {
		return Evidence{}, fmt.Errorf("create gzip writer: %w", err)
	}
	if _, err := writer.Write(response.Body); err != nil {
		return Evidence{}, fmt.Errorf("compress body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return Evidence{}, fmt.Errorf("finish compressed body: %w", err)
	}
	digest := sha256.Sum256(response.Body)
	return Evidence{
		SiteID:         siteID,
		RequestedURL:   response.RequestedURL,
		FinalURL:       response.FinalURL,
		Redirects:      append([]safefetch.Redirect(nil), response.Redirects...),
		Status:         response.Status,
		MediaType:      response.MediaType,
		ETag:           response.Header.Get("ETag"),
		LastModified:   response.Header.Get("Last-Modified"),
		FetchedAt:      response.FinishedAt,
		DecodedBytes:   int64(len(response.Body)),
		BodySHA256:     hex.EncodeToString(digest[:]),
		BodyEncoding:   BodyEncodingGzip,
		CompressedBody: compressed.Bytes(),
	}, nil
}

// Body decompresses the stored body, refusing more than maxBytes and
// verifying the recorded size and digest.
func (evidence Evidence) Body(maxBytes int64) ([]byte, error) {
	if evidence.BodyEncoding != BodyEncodingGzip {
		return nil, fmt.Errorf("unsupported body encoding %q", evidence.BodyEncoding)
	}
	reader, err := gzip.NewReader(bytes.NewReader(evidence.CompressedBody))
	if err != nil {
		return nil, fmt.Errorf("open stored body: %w", err)
	}
	defer reader.Close()
	body, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("decompress stored body: %w", err)
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("stored body exceeds %d bytes", maxBytes)
	}
	digest := sha256.Sum256(body)
	if int64(len(body)) != evidence.DecodedBytes || hex.EncodeToString(digest[:]) != evidence.BodySHA256 {
		return nil, errors.New("stored body does not match its recorded size and digest")
	}
	return body, nil
}

// Store persists evidence (ingest.evidence). Insert assigns the ID; stored
// evidence is never updated.
type Store interface {
	Insert(ctx context.Context, evidence Evidence) (Evidence, error)
	Get(ctx context.Context, id int64) (Evidence, error)
}

// ErrNotFound means no evidence has the requested ID.
var ErrNotFound = errors.New("evidence not found")

// MemoryStore is an in-memory Store.
type MemoryStore struct {
	mu    sync.Mutex
	next  int64
	items map[int64]Evidence
}

// NewMemoryStore returns an empty MemoryStore.
func NewMemoryStore() *MemoryStore { return &MemoryStore{items: map[int64]Evidence{}} }

// Insert stores a copy of evidence under a new ID.
func (store *MemoryStore) Insert(_ context.Context, evidence Evidence) (Evidence, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.next++
	evidence.ID = store.next
	evidence.CompressedBody = append([]byte(nil), evidence.CompressedBody...)
	store.items[evidence.ID] = evidence
	return evidence, nil
}

// Get returns a copy of the evidence with id.
func (store *MemoryStore) Get(_ context.Context, id int64) (Evidence, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	evidence, ok := store.items[id]
	if !ok {
		return Evidence{}, fmt.Errorf("%w: %d", ErrNotFound, id)
	}
	evidence.CompressedBody = append([]byte(nil), evidence.CompressedBody...)
	return evidence, nil
}
