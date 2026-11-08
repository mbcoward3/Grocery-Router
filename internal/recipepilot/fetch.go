package recipepilot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"strconv"
	"time"
)

const (
	maxResponseBytes = 5 << 20
	maxRedirects     = 5
	requestTimeout   = 20 * time.Second
	pilotUserAgent   = "GroceryRouter-RecipeSourcePilot/1.0"
)

// Fetcher acquires one manifested source and returns provenance even on response errors.
type Fetcher interface {
	Fetch(context.Context, string) (FetchResult, error)
}

// SafeFetcher performs bounded public-network-only HTTPS requests.
type SafeFetcher struct {
	Resolver *net.Resolver
	Dialer   *net.Dialer
	Now      func() time.Time
}

// NewSafeFetcher returns a fetcher configured with production network defaults.
func NewSafeFetcher() *SafeFetcher {
	return &SafeFetcher{
		Resolver: net.DefaultResolver,
		Dialer:   &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second},
		Now:      time.Now,
	}
}

// Fetch validates, resolves, fetches, and bounds one public HTML page.
func (fetcher *SafeFetcher) Fetch(ctx context.Context, rawURL string) (FetchResult, error) {
	result := FetchResult{Metadata: FetchMetadata{RequestedURL: rawURL, FetchedAt: fetcher.now().UTC()}}
	if _, err := ValidateFetchURL(rawURL); err != nil {
		return result, err
	}
	resolver := fetcher.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	dialer := fetcher.Dialer
	if dialer == nil {
		dialer = &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	}
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(dialContext context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, fmt.Errorf("split dial address: %w", err)
			}
			if port != "443" {
				return nil, fmt.Errorf("dial port %q is not allowed", port)
			}
			addresses, err := resolver.LookupIPAddr(dialContext, host)
			if err != nil {
				return nil, fmt.Errorf("resolve %q: %w", host, err)
			}
			if len(addresses) == 0 {
				return nil, fmt.Errorf("resolve %q: no addresses", host)
			}
			for _, address := range addresses {
				if !isPublicIP(address.IP) {
					return nil, fmt.Errorf("resolve %q: non-public address %s is not allowed", host, address.IP)
				}
			}
			var lastErr error
			for _, resolved := range addresses {
				connection, err := dialer.DialContext(dialContext, network, net.JoinHostPort(resolved.IP.String(), port))
				if err == nil {
					return connection, nil
				}
				lastErr = err
			}
			return nil, fmt.Errorf("dial public addresses for %q: %w", host, lastErr)
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          2,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}
	defer transport.CloseIdleConnections()

	redirects := make([]Redirect, 0)
	client := &http.Client{
		Transport: transport,
		Timeout:   requestTimeout,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			return validateRedirect(request, via, &redirects)
		},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return result, fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("User-Agent", pilotUserAgent)
	request.Header.Set("Accept", "text/html, application/xhtml+xml;q=0.9")
	response, err := client.Do(request)
	if err != nil {
		result.Metadata.Redirects = redirects
		return result, fmt.Errorf("fetch: %w", err)
	}
	defer response.Body.Close()

	metadata := result.Metadata
	metadata.FinalURL = response.Request.URL.String()
	metadata.Redirects = redirects
	metadata.Status = response.StatusCode
	metadata.ETag = response.Header.Get("ETag")
	metadata.LastModified = response.Header.Get("Last-Modified")
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return FetchResult{Metadata: metadata}, fmt.Errorf("HTTP status %d", response.StatusCode)
	}
	contentType := response.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return FetchResult{Metadata: metadata}, fmt.Errorf("parse Content-Type %q: %w", contentType, err)
	}
	metadata.MediaType = mediaType
	if mediaType != "text/html" && mediaType != "application/xhtml+xml" {
		return FetchResult{Metadata: metadata}, fmt.Errorf("Content-Type %q is not HTML", mediaType)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return FetchResult{Metadata: metadata}, fmt.Errorf("read response body: %w", err)
	}
	metadata.DecodedBytes = int64(len(body))
	if len(body) > maxResponseBytes {
		return FetchResult{Metadata: metadata}, fmt.Errorf("decoded response body exceeds %s bytes", strconv.Itoa(maxResponseBytes))
	}
	digest := sha256.Sum256(body)
	metadata.BodySHA256 = hex.EncodeToString(digest[:])
	return FetchResult{Metadata: metadata, Body: body}, nil
}

func validateRedirect(request *http.Request, via []*http.Request, redirects *[]Redirect) error {
	if len(via) > maxRedirects {
		return fmt.Errorf("more than %d redirects", maxRedirects)
	}
	if _, err := ValidateFetchURL(request.URL.String()); err != nil {
		return fmt.Errorf("unsafe redirect: %w", err)
	}
	if len(via) == 0 {
		return fmt.Errorf("redirect history is empty")
	}
	previous := via[len(via)-1]
	status := 0
	if request.Response != nil {
		status = request.Response.StatusCode
	}
	*redirects = append(*redirects, Redirect{Status: status, From: previous.URL.String(), To: request.URL.String()})
	return nil
}

func (fetcher *SafeFetcher) now() time.Time {
	if fetcher.Now != nil {
		return fetcher.Now()
	}
	return time.Now()
}
