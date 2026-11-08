// Package fetchtest provides an offline HTTPS network for catalog fetch tests:
// a TLS fixture server whose certificate covers chosen host names, a static
// resolver, and a dialer that routes every validated connection to the server.
package fetchtest

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/catalog/safefetch"
)

// PublicIP is the address hosts resolve to by default. It is publicly routable
// but never contacted: the Dialer redirects every connection to the fixture.
const PublicIP = "93.184.215.14"

// Network is an offline HTTPS network for tests.
type Network struct {
	Server   *httptest.Server
	Resolver *Resolver
	Dialer   *Dialer
	roots    *x509.CertPool
}

// New starts a TLS server for handler whose certificate is valid for hosts.
// Every host initially resolves to PublicIP.
func New(t testing.TB, handler http.Handler, hosts ...string) *Network {
	t.Helper()
	certificate, roots := selfSigned(t, hosts)
	server := httptest.NewUnstartedServer(handler)
	server.TLS = &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12}
	server.StartTLS()
	t.Cleanup(server.Close)
	resolver := &Resolver{answers: map[string][]string{}}
	for _, host := range hosts {
		resolver.Set(host, PublicIP)
	}
	return &Network{
		Server:   server,
		Resolver: resolver,
		Dialer:   &Dialer{target: server.Listener.Addr().String()},
		roots:    roots,
	}
}

// Client returns a safefetch client wired to this network.
func (network *Network) Client(t testing.TB, options ...safefetch.Option) *safefetch.Client {
	t.Helper()
	base := []safefetch.Option{
		safefetch.WithResolver(network.Resolver),
		safefetch.WithDialer(network.Dialer),
		safefetch.WithTLSConfig(&tls.Config{RootCAs: network.roots, MinVersion: tls.VersionTLS12}),
	}
	client, err := safefetch.New("support@example.com", append(base, options...)...)
	if err != nil {
		t.Fatalf("safefetch.New: %v", err)
	}
	return client
}

// Resolver answers lookups from a fixed table.
type Resolver struct {
	mu      sync.Mutex
	answers map[string][]string
	lookups []string
}

// Set replaces the answers for host.
func (resolver *Resolver) Set(host string, ips ...string) {
	resolver.mu.Lock()
	defer resolver.mu.Unlock()
	resolver.answers[host] = ips
}

// LookupIPAddr returns the configured answers or a not-found error.
func (resolver *Resolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	resolver.mu.Lock()
	defer resolver.mu.Unlock()
	resolver.lookups = append(resolver.lookups, host)
	ips, ok := resolver.answers[host]
	if !ok {
		return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
	}
	answers := make([]net.IPAddr, 0, len(ips))
	for _, ip := range ips {
		answers = append(answers, net.IPAddr{IP: net.ParseIP(ip)})
	}
	return answers, nil
}

// Lookups returns every host looked up, in order.
func (resolver *Resolver) Lookups() []string {
	resolver.mu.Lock()
	defer resolver.mu.Unlock()
	return append([]string(nil), resolver.lookups...)
}

// Dialer records validated dial addresses and connects to the fixture server.
type Dialer struct {
	mu     sync.Mutex
	target string
	dialed []string
}

// DialContext connects to the fixture server regardless of address.
func (dialer *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	dialer.mu.Lock()
	dialer.dialed = append(dialer.dialed, address)
	dialer.mu.Unlock()
	var base net.Dialer
	return base.DialContext(ctx, network, dialer.target)
}

// Dialed returns every address passed to DialContext.
func (dialer *Dialer) Dialed() []string {
	dialer.mu.Lock()
	defer dialer.mu.Unlock()
	return append([]string(nil), dialer.dialed...)
}

func selfSigned(t testing.TB, hosts []string) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "fetchtest"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              hosts,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(parsed)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: parsed}, roots
}

// Mux routes requests by Host header and path, for multi-host fixtures.
type Mux struct {
	mu     sync.Mutex
	routes map[string]http.HandlerFunc
	hits   map[string]int
}

// NewMux returns an empty host-aware mux.
func NewMux() *Mux {
	return &Mux{routes: map[string]http.HandlerFunc{}, hits: map[string]int{}}
}

// Handle registers handler for host and path (e.g. "www.a.test", "/robots.txt").
func (mux *Mux) Handle(host, path string, handler http.HandlerFunc) {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	mux.routes[host+path] = handler
}

// Hits returns how many requests reached host and path.
func (mux *Mux) Hits(host, path string) int {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	return mux.hits[host+path]
}

// ServeHTTP dispatches by host and path; unknown routes return 404.
func (mux *Mux) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	host, _, err := net.SplitHostPort(request.Host)
	if err != nil {
		host = request.Host
	}
	key := host + request.URL.Path
	mux.mu.Lock()
	mux.hits[key]++
	handler := mux.routes[key]
	mux.mu.Unlock()
	if handler == nil {
		http.Error(writer, fmt.Sprintf("no fixture for %s", key), http.StatusNotFound)
		return
	}
	handler(writer, request)
}

// Text returns a handler that writes body with the given status and content type.
func Text(status int, contentType, body string) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		if contentType != "" {
			writer.Header().Set("Content-Type", contentType)
		}
		writer.WriteHeader(status)
		_, _ = writer.Write([]byte(body))
	}
}

// Redirect returns a handler that redirects to location with status.
func Redirect(status int, location string) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Location", location)
		writer.WriteHeader(status)
	}
}
