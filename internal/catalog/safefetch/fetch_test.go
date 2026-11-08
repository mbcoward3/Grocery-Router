package safefetch_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/catalog/fetchtest"
	"github.com/mbcoward3/grocery-router/internal/catalog/safefetch"
)

const (
	siteHost  = "www.a.test"
	otherHost = "cdn.a.test"
	foreign   = "evil.test"
)

var siteHosts = []string{siteHost, otherHost}

func newNetwork(t *testing.T) (*fetchtest.Network, *fetchtest.Mux) {
	t.Helper()
	mux := fetchtest.NewMux()
	return fetchtest.New(t, mux, siteHost, otherHost, foreign), mux
}

func page(raw string) safefetch.Request {
	return safefetch.Request{URL: raw, Profile: safefetch.PageProfile, AllowedHosts: siteHosts}
}

func TestFetchPageRecordsProvenanceAndHeaders(t *testing.T) {
	network, mux := newNetwork(t)
	var gotUA, gotCookie, gotAuth string
	mux.Handle(siteHost, "/start", fetchtest.Redirect(http.StatusMovedPermanently, "https://cdn.a.test/recipe#frag"))
	mux.Handle(otherHost, "/recipe", func(w http.ResponseWriter, r *http.Request) {
		gotUA, gotCookie, gotAuth = r.UserAgent(), r.Header.Get("Cookie"), r.Header.Get("Authorization")
		http.SetCookie(w, &http.Cookie{Name: "track", Value: "1"})
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html>ok</html>"))
	})
	client := network.Client(t)
	response, err := client.Fetch(context.Background(), page("https://www.a.test/start"))
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if response.FinalURL != "https://cdn.a.test/recipe" || len(response.Redirects) != 1 || response.Redirects[0].Status != 301 {
		t.Fatalf("provenance = %q %#v", response.FinalURL, response.Redirects)
	}
	if string(response.Body) != "<html>ok</html>" || response.MediaType != "text/html" || response.Status != 200 {
		t.Fatalf("response = %d %q %q", response.Status, response.MediaType, response.Body)
	}
	if !strings.HasPrefix(gotUA, "GroceryRouter/") || !strings.Contains(gotUA, "support@example.com") {
		t.Errorf("user agent = %q", gotUA)
	}
	if gotCookie != "" || gotAuth != "" {
		t.Errorf("cookie/auth leaked: %q %q", gotCookie, gotAuth)
	}
	for _, dialed := range network.Dialer.Dialed() {
		if dialed != fetchtest.PublicIP+":443" {
			t.Errorf("dialed %q, want validated IP", dialed)
		}
	}
}

func TestFetchRejectsUnsafeURLsBeforeNetwork(t *testing.T) {
	network, _ := newNetwork(t)
	client := network.Client(t)
	for _, raw := range []string{
		"http://www.a.test/", "https://www.a.test:8443/", "https://user:pw@www.a.test/",
		"https://www.a.test/#x", "https://93.184.215.14/", "/relative", "https://evil.test/",
		"https://sub.www.a.test/",
	} {
		_, err := client.Fetch(context.Background(), page(raw))
		if safefetch.KindOf(err) != safefetch.KindPolicy {
			t.Errorf("%s: err = %v, want policy", raw, err)
		}
	}
	if len(network.Dialer.Dialed()) != 0 {
		t.Fatalf("unsafe URLs reached the network: %v", network.Dialer.Dialed())
	}
}

func TestFetchRejectsNonPublicResolution(t *testing.T) {
	cases := map[string][]string{
		"loopback":       {"127.0.0.1"},
		"private":        {"10.1.2.3"},
		"cgnat":          {"100.64.0.1"},
		"link-local":     {"169.254.169.254"},
		"documentation":  {"192.0.2.10"},
		"ipv6 loopback":  {"::1"},
		"ipv6 ula":       {"fd00::1"},
		"mapped private": {"::ffff:10.0.0.1"},
		"one bad answer": {fetchtest.PublicIP, "10.0.0.1"},
	}
	for name, ips := range cases {
		t.Run(name, func(t *testing.T) {
			network, mux := newNetwork(t)
			mux.Handle(siteHost, "/", fetchtest.Text(200, "text/html", "x"))
			network.Resolver.Set(siteHost, ips...)
			_, err := network.Client(t).Fetch(context.Background(), page("https://www.a.test/"))
			if safefetch.KindOf(err) != safefetch.KindPolicy {
				t.Fatalf("err = %v, want policy", err)
			}
			if len(network.Dialer.Dialed()) != 0 {
				t.Fatalf("dialed %v", network.Dialer.Dialed())
			}
		})
	}
}

func TestFetchRevalidatesRedirects(t *testing.T) {
	t.Run("redirect to private IP host", func(t *testing.T) {
		network, mux := newNetwork(t)
		network.Resolver.Set(otherHost, "127.0.0.1")
		mux.Handle(siteHost, "/", fetchtest.Redirect(302, "https://cdn.a.test/x"))
		mux.Handle(otherHost, "/x", fetchtest.Text(200, "text/html", "secret"))
		response, err := network.Client(t).Fetch(context.Background(), page("https://www.a.test/"))
		if safefetch.KindOf(err) != safefetch.KindPolicy || mux.Hits(otherHost, "/x") != 0 {
			t.Fatalf("err = %v hits = %d", err, mux.Hits(otherHost, "/x"))
		}
		if len(response.Redirects) != 1 {
			t.Errorf("redirects = %#v", response.Redirects)
		}
	})
	t.Run("redirect to IP literal", func(t *testing.T) {
		network, mux := newNetwork(t)
		mux.Handle(siteHost, "/", fetchtest.Redirect(302, "https://10.0.0.1/x"))
		_, err := network.Client(t).Fetch(context.Background(), page("https://www.a.test/"))
		if safefetch.KindOf(err) != safefetch.KindPolicy {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("redirect off host", func(t *testing.T) {
		network, mux := newNetwork(t)
		mux.Handle(siteHost, "/", fetchtest.Redirect(302, "https://evil.test/x"))
		mux.Handle(foreign, "/x", fetchtest.Text(200, "text/html", "x"))
		response, err := network.Client(t).Fetch(context.Background(), page("https://www.a.test/"))
		if safefetch.KindOf(err) != safefetch.KindPolicy || mux.Hits(foreign, "/x") != 0 {
			t.Fatalf("err = %v", err)
		}
		if response.FinalURL != "https://evil.test/x" {
			t.Errorf("final URL = %q", response.FinalURL)
		}
	})
	t.Run("redirect to http", func(t *testing.T) {
		network, mux := newNetwork(t)
		mux.Handle(siteHost, "/", fetchtest.Redirect(302, "http://www.a.test/x"))
		_, err := network.Client(t).Fetch(context.Background(), page("https://www.a.test/"))
		if safefetch.KindOf(err) != safefetch.KindPolicy {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("five redirects allowed, sixth refused", func(t *testing.T) {
		network, mux := newNetwork(t)
		for i := 0; i < 6; i++ {
			mux.Handle(siteHost, "/r"+string(rune('0'+i)), fetchtest.Redirect(302, "/r"+string(rune('1'+i))))
		}
		mux.Handle(siteHost, "/r5", fetchtest.Text(200, "text/html", "done"))
		response, err := network.Client(t).Fetch(context.Background(), page("https://www.a.test/r0"))
		if err != nil || string(response.Body) != "done" || len(response.Redirects) != 5 {
			t.Fatalf("five: err=%v redirects=%d", err, len(response.Redirects))
		}
		mux.Handle(siteHost, "/r5", fetchtest.Redirect(302, "/r6"))
		mux.Handle(siteHost, "/r6", fetchtest.Text(200, "text/html", "done"))
		_, err = network.Client(t).Fetch(context.Background(), page("https://www.a.test/r0"))
		if safefetch.KindOf(err) != safefetch.KindPolicy || mux.Hits(siteHost, "/r6") != 0 {
			t.Fatalf("six: err=%v", err)
		}
	})
}

func TestFetchGateRunsBeforeEveryHop(t *testing.T) {
	network, mux := newNetwork(t)
	mux.Handle(siteHost, "/", fetchtest.Redirect(302, "https://cdn.a.test/x"))
	mux.Handle(otherHost, "/x", fetchtest.Text(200, "text/html", "x"))
	var gated, released []string
	request := page("https://www.a.test/")
	request.Gate = func(_ context.Context, hop *url.URL) (func(), error) {
		gated = append(gated, hop.Host)
		return func() { released = append(released, hop.Host) }, nil
	}
	if _, err := network.Client(t).Fetch(context.Background(), request); err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if strings.Join(gated, ",") != "www.a.test,cdn.a.test" || strings.Join(released, ",") != "www.a.test,cdn.a.test" {
		t.Fatalf("gated %v released %v", gated, released)
	}
	refusal := errors.New("robots disallow")
	request.Gate = func(context.Context, *url.URL) (func(), error) { return nil, refusal }
	_, err := network.Client(t).Fetch(context.Background(), request)
	if safefetch.KindOf(err) != safefetch.KindGate || !errors.Is(err, refusal) {
		t.Fatalf("err = %v", err)
	}
}

func TestFetchBoundsDecodedBody(t *testing.T) {
	network, mux := newNetwork(t)
	limit := int(safefetch.PageProfile.MaxBodyBytes)
	mux.Handle(siteHost, "/exact", fetchtest.Text(200, "text/html", strings.Repeat("a", limit)))
	mux.Handle(siteHost, "/over", fetchtest.Text(200, "text/html", strings.Repeat("a", limit+1)))
	// A small gzip body that decodes past the limit must still be refused.
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, _ = writer.Write(bytes.Repeat([]byte("a"), limit+10))
	_ = writer.Close()
	mux.Handle(siteHost, "/bomb", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = w.Write(compressed.Bytes())
	})
	client := network.Client(t)
	if response, err := client.Fetch(context.Background(), page("https://www.a.test/exact")); err != nil || len(response.Body) != limit {
		t.Fatalf("exact: %v", err)
	}
	for _, path := range []string{"/over", "/bomb"} {
		if _, err := client.Fetch(context.Background(), page("https://www.a.test"+path)); safefetch.KindOf(err) != safefetch.KindTooLarge {
			t.Errorf("%s: err = %v", path, err)
		}
	}
}

func TestFetchRejectsWrongMediaTypeAndKeepsErrorBodies(t *testing.T) {
	network, mux := newNetwork(t)
	mux.Handle(siteHost, "/json", fetchtest.Text(200, "application/json", "{}"))
	mux.Handle(siteHost, "/503", fetchtest.Text(503, "text/html", "Just a moment..."))
	client := network.Client(t)
	if _, err := client.Fetch(context.Background(), page("https://www.a.test/json")); safefetch.KindOf(err) != safefetch.KindMediaType {
		t.Errorf("json: err = %v", err)
	}
	response, err := client.Fetch(context.Background(), page("https://www.a.test/503"))
	if err != nil || response.Status != 503 || string(response.Body) != "Just a moment..." {
		t.Errorf("503: %v %d %q", err, response.Status, response.Body)
	}
}

func TestFetchTimesOut(t *testing.T) {
	network, mux := newNetwork(t)
	done := make(chan struct{})
	t.Cleanup(func() { close(done) })
	mux.Handle(siteHost, "/slow", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		w.(http.Flusher).Flush()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	})
	request := page("https://www.a.test/slow")
	request.Profile.Timeout = 100 * time.Millisecond
	_, err := network.Client(t).Fetch(context.Background(), request)
	if safefetch.KindOf(err) != safefetch.KindTimeout {
		t.Fatalf("err = %v, want timeout", err)
	}
}

func TestIsPublicAddr(t *testing.T) {
	for raw, want := range map[string]bool{
		"93.184.215.14": true, "2606:4700::1": true, "10.0.0.1": false, "172.16.0.1": false,
		"192.168.1.1": false, "100.64.1.1": false, "198.18.0.1": false, "198.51.100.1": false,
		"203.0.113.1": false, "224.0.0.1": false, "255.255.255.255": false, "0.0.0.0": false,
		"2001:db8::1": false, "fe80::1": false, "ff02::1": false, "::": false,
	} {
		if got := safefetch.IsPublicAddr(netip.MustParseAddr(raw)); got != want {
			t.Errorf("IsPublicAddr(%s) = %v, want %v", raw, got, want)
		}
	}
}

func TestNewRequiresContact(t *testing.T) {
	if _, err := safefetch.New(""); err == nil {
		t.Fatal("empty contact accepted")
	}
	if _, err := safefetch.New("a\r\nX-Injected: 1"); err == nil {
		t.Fatal("header injection accepted")
	}
}
