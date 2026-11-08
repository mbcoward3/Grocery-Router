// Package safefetch performs bounded, public-network-only HTTPS GET requests
// for catalog acquisition (CATALOG_INGEST_SPEC.md §7.3).
//
// Redirects are followed one hop at a time so that every hop is revalidated,
// restricted to the site's hosts, and passed through a caller-supplied Gate
// (robots and per-host rate limiting live in the gate, not here).
package safefetch

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/mbcoward3/grocery-router/internal/catalog/clock"
)

// MaxRedirects is the most redirect hops a single fetch may follow.
const MaxRedirects = 5

// ProductToken is the user-agent product token robots groups are matched on.
const ProductToken = "GroceryRouter"

// maxErrorBodyBytes bounds how much of a non-2xx body is kept for
// bot-challenge detection.
const maxErrorBodyBytes = 64 << 10

// Profile bounds one kind of request.
type Profile struct {
	Name         string
	MaxBodyBytes int64
	Timeout      time.Duration
	Accept       string
	// MediaTypes lists acceptable 2xx media types; empty accepts any.
	MediaTypes []string
}

// Fetch profiles. The page and sitemap limits come from the spec; robots.txt
// uses the 500 KiB ceiling from RFC 9309 §2.5.
var (
	PageProfile = Profile{
		Name: "page", MaxBodyBytes: 5 << 20, Timeout: 20 * time.Second,
		Accept:     "text/html, application/xhtml+xml;q=0.9",
		MediaTypes: []string{"text/html", "application/xhtml+xml"},
	}
	SitemapProfile = Profile{
		Name: "sitemap", MaxBodyBytes: 50 << 20, Timeout: 60 * time.Second,
		Accept: "application/xml, text/xml;q=0.9, application/gzip;q=0.8",
		MediaTypes: []string{
			"application/xml", "text/xml", "application/gzip", "application/x-gzip",
			"application/octet-stream", "text/plain",
		},
	}
	RobotsProfile = Profile{
		Name: "robots", MaxBodyBytes: 500 << 10, Timeout: 20 * time.Second,
		Accept: "text/plain",
	}
)

// Resolver resolves host names immediately before dialing.
type Resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

// Dialer opens connections to already-validated IP addresses.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Gate is called before every hop, including the first. It may block (rate
// limiting) or refuse (robots). The returned release is called once the hop's
// response has been fully read.
type Gate func(ctx context.Context, hop *url.URL) (release func(), err error)

// Request describes one fetch.
type Request struct {
	URL          string
	Profile      Profile
	AllowedHosts []string
	Gate         Gate
}

// Redirect records one followed redirect hop.
type Redirect struct {
	Status int    `json:"status"`
	From   string `json:"from"`
	To     string `json:"to"`
}

// Response is the provenance and decoded body of the final hop. It is
// returned alongside most errors so failures can be recorded.
type Response struct {
	RequestedURL string
	FinalURL     string
	Redirects    []Redirect
	Status       int
	Header       http.Header
	MediaType    string
	// Body is the decoded body. For non-2xx responses it holds at most the
	// first 64 KiB, kept for bot-challenge detection.
	Body       []byte
	StartedAt  time.Time
	FinishedAt time.Time
}

// ErrorKind classifies fetch failures for the failure policy.
type ErrorKind string

// Error kinds.
const (
	// KindPolicy is a safety violation: unsafe URL, off-host or excess
	// redirects, or a non-public destination. Never retried.
	KindPolicy ErrorKind = "policy"
	// KindTooLarge means the decoded body exceeded the profile limit.
	KindTooLarge ErrorKind = "too_large"
	// KindMediaType means a 2xx response had an unacceptable media type.
	KindMediaType ErrorKind = "media_type"
	// KindTimeout means the per-hop deadline expired.
	KindTimeout ErrorKind = "timeout"
	// KindNetwork is any other transport failure.
	KindNetwork ErrorKind = "network"
	// KindGate means the gate refused or failed the hop.
	KindGate ErrorKind = "gate"
	// KindCanceled means the caller's context ended.
	KindCanceled ErrorKind = "canceled"
)

// Error is a classified fetch failure.
type Error struct {
	Kind ErrorKind
	Err  error
}

func (err *Error) Error() string { return fmt.Sprintf("%s: %v", err.Kind, err.Err) }

// Unwrap returns the underlying error.
func (err *Error) Unwrap() error { return err.Err }

// KindOf returns the ErrorKind of err, or "" when err is not an *Error.
func KindOf(err error) ErrorKind {
	var fetchErr *Error
	if errors.As(err, &fetchErr) {
		return fetchErr.Kind
	}
	return ""
}

// Client performs safe fetches. The zero value is not usable; use New.
type Client struct {
	userAgent string
	resolver  Resolver
	dialer    Dialer
	tlsConfig *tls.Config
	clock     clock.Clock
	transport *http.Transport
}

// Option customizes a Client.
type Option func(*Client)

// WithResolver replaces the DNS resolver (tests inject fixed answers).
func WithResolver(resolver Resolver) Option { return func(c *Client) { c.resolver = resolver } }

// WithDialer replaces the dialer used after address validation.
func WithDialer(dialer Dialer) Option { return func(c *Client) { c.dialer = dialer } }

// WithTLSConfig replaces the TLS client configuration (tests trust a fixture CA).
func WithTLSConfig(config *tls.Config) Option { return func(c *Client) { c.tlsConfig = config } }

// WithClock replaces the clock used for timestamps.
func WithClock(clk clock.Clock) Option { return func(c *Client) { c.clock = clk } }

// UserAgent returns the user agent naming Grocery Router and a contact address.
func UserAgent(contact string) string {
	return ProductToken + "/1.0 (catalog acquisition; +" + contact + ")"
}

// New returns a Client whose user agent names contact (the public support
// address or URL).
func New(contact string, options ...Option) (*Client, error) {
	contact = strings.TrimSpace(contact)
	if contact == "" || strings.ContainsAny(contact, "\r\n\t()") {
		return nil, fmt.Errorf("contact address %q is empty or contains forbidden characters", contact)
	}
	client := &Client{
		userAgent: UserAgent(contact),
		resolver:  net.DefaultResolver,
		dialer:    &net.Dialer{Timeout: 10 * time.Second},
		clock:     clock.Real{},
	}
	for _, option := range options {
		option(client)
	}
	client.transport = &http.Transport{
		Proxy:                 nil,
		DialContext:           client.dialPublic,
		TLSClientConfig:       client.tlsConfig,
		ForceAttemptHTTP2:     true,
		DisableKeepAlives:     true,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
	}
	return client, nil
}

// UserAgentString returns the configured user agent.
func (client *Client) UserAgentString() string { return client.userAgent }

// dialPublic resolves the host immediately before dialing and refuses any
// answer set containing a non-public address.
func (client *Client) dialPublic(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, &Error{Kind: KindPolicy, Err: fmt.Errorf("split dial address: %w", err)}
	}
	if port != "443" {
		return nil, &Error{Kind: KindPolicy, Err: fmt.Errorf("dial port %q is not allowed", port)}
	}
	answers, err := client.resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve %q: %w", host, err)
	}
	if len(answers) == 0 {
		return nil, fmt.Errorf("resolve %q: no addresses", host)
	}
	addrs := make([]netip.Addr, 0, len(answers))
	for _, answer := range answers {
		addr, ok := netip.AddrFromSlice(answer.IP)
		if !ok || !IsPublicAddr(addr) {
			return nil, &Error{Kind: KindPolicy, Err: fmt.Errorf("host %q resolves to non-public address %s", host, answer.IP)}
		}
		addrs = append(addrs, addr.Unmap())
	}
	var lastErr error
	for _, addr := range addrs {
		connection, err := client.dialer.DialContext(ctx, network, net.JoinHostPort(addr.String(), port))
		if err == nil {
			return connection, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("dial %q: %w", host, lastErr)
}

// Fetch performs request, following at most MaxRedirects validated redirects.
func (client *Client) Fetch(ctx context.Context, request Request) (Response, error) {
	response := Response{RequestedURL: request.URL, StartedAt: client.clock.Now().UTC()}
	finish := func(err error) (Response, error) {
		response.FinishedAt = client.clock.Now().UTC()
		return response, err
	}
	current, err := client.checkHop(request.URL, request.AllowedHosts)
	if err != nil {
		return finish(err)
	}
	for hops := 0; ; hops++ {
		response.FinalURL = current.String()
		next, status, err := client.hop(ctx, request, current, &response)
		if err != nil || next == nil {
			return finish(err)
		}
		response.Redirects = append(response.Redirects, Redirect{Status: status, From: current.String(), To: next.String()})
		response.FinalURL = next.String()
		if hops+1 > MaxRedirects {
			return finish(&Error{Kind: KindPolicy, Err: fmt.Errorf("more than %d redirects", MaxRedirects)})
		}
		if current, err = client.checkHop(next.String(), request.AllowedHosts); err != nil {
			return finish(err)
		}
	}
}

func (client *Client) checkHop(raw string, hosts []string) (*url.URL, error) {
	parsed, err := ValidateURL(raw)
	if err != nil {
		return nil, &Error{Kind: KindPolicy, Err: err}
	}
	if !HostAllowed(parsed, hosts) {
		return nil, &Error{Kind: KindPolicy, Err: fmt.Errorf("host %q is not one of the site's hosts", parsed.Hostname())}
	}
	// Canonical form: lowercase host without the default port, and "/" for
	// an empty path.
	parsed.Host = NormalizeHost(parsed.Hostname())
	if parsed.Path == "" && parsed.RawPath == "" {
		parsed.Path = "/"
	}
	return parsed, nil
}

// hop performs one GET. It returns the next URL for a redirect, or nil when
// response holds the final result.
func (client *Client) hop(ctx context.Context, request Request, target *url.URL, response *Response) (*url.URL, int, error) {
	if request.Gate != nil {
		release, err := request.Gate(ctx, target)
		if err != nil {
			if ctx.Err() != nil {
				return nil, 0, &Error{Kind: KindCanceled, Err: err}
			}
			return nil, 0, &Error{Kind: KindGate, Err: err}
		}
		if release != nil {
			defer release()
		}
	}
	hopCtx, cancel := context.WithTimeout(ctx, request.Profile.Timeout)
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(hopCtx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, 0, &Error{Kind: KindPolicy, Err: fmt.Errorf("build request: %w", err)}
	}
	httpRequest.Header.Set("User-Agent", client.userAgent)
	if request.Profile.Accept != "" {
		httpRequest.Header.Set("Accept", request.Profile.Accept)
	}
	httpClient := &http.Client{
		Transport:     client.transport,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		return nil, 0, client.classify(ctx, hopCtx, err)
	}
	defer httpResponse.Body.Close()
	response.Status = httpResponse.StatusCode
	response.Header = httpResponse.Header.Clone()
	response.Body = nil
	response.MediaType = ""
	if mediaType, _, err := mime.ParseMediaType(httpResponse.Header.Get("Content-Type")); err == nil {
		response.MediaType = strings.ToLower(mediaType)
	}

	if isRedirect(httpResponse.StatusCode) {
		location := httpResponse.Header.Get("Location")
		if location == "" {
			return nil, 0, &Error{Kind: KindPolicy, Err: fmt.Errorf("redirect %d without Location", httpResponse.StatusCode)}
		}
		next, err := target.Parse(location)
		if err != nil {
			return nil, 0, &Error{Kind: KindPolicy, Err: fmt.Errorf("parse redirect Location %q: %w", location, err)}
		}
		next.Fragment = ""
		next.RawFragment = ""
		return next, httpResponse.StatusCode, nil
	}

	limit := request.Profile.MaxBodyBytes
	success := httpResponse.StatusCode >= 200 && httpResponse.StatusCode <= 299
	if !success {
		limit = min(limit, maxErrorBodyBytes)
	}
	body, err := io.ReadAll(io.LimitReader(httpResponse.Body, limit+1))
	if err != nil {
		return nil, 0, client.classify(ctx, hopCtx, fmt.Errorf("read body: %w", err))
	}
	if int64(len(body)) > limit {
		if !success {
			response.Body = body[:limit]
			return nil, 0, nil
		}
		return nil, 0, &Error{Kind: KindTooLarge, Err: fmt.Errorf("decoded body exceeds %d bytes", limit)}
	}
	response.Body = body
	if success && len(request.Profile.MediaTypes) > 0 && !contains(request.Profile.MediaTypes, response.MediaType) {
		return nil, 0, &Error{Kind: KindMediaType, Err: fmt.Errorf("media type %q is not acceptable for %s", response.MediaType, request.Profile.Name)}
	}
	return nil, 0, nil
}

func (client *Client) classify(parent, hopCtx context.Context, err error) error {
	var fetchErr *Error
	if errors.As(err, &fetchErr) {
		return fetchErr
	}
	if parent.Err() != nil {
		return &Error{Kind: KindCanceled, Err: err}
	}
	var netErr net.Error
	if errors.Is(hopCtx.Err(), context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
		return &Error{Kind: KindTimeout, Err: err}
	}
	return &Error{Kind: KindNetwork, Err: err}
}

func isRedirect(status int) bool {
	switch status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	}
	return false
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
