package recipepilot

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestReadManifestValidatesStrictContract(t *testing.T) {
	data := []byte("version: 1\nsources:\n  - id: example-recipe\n    url: https://example.com/recipe\n")
	manifest, digest, err := ReadManifest(data)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if len(manifest.Sources) != 1 || len(digest) != 64 {
		t.Fatalf("manifest = %#v, digest = %q", manifest, digest)
	}

	cases := map[string]string{
		"unknown field": "version: 1\nextra: true\nsources:\n  - id: recipe\n    url: https://example.com/\n",
		"unsafe scheme": "version: 1\nsources:\n  - id: recipe\n    url: http://example.com/\n",
		"credentials":   "version: 1\nsources:\n  - id: recipe\n    url: https://user@example.com/\n",
		"fragment":      "version: 1\nsources:\n  - id: recipe\n    url: https://example.com/#recipe\n",
		"duplicate":     "version: 1\nsources:\n  - id: recipe\n    url: https://example.com/a\n  - id: recipe\n    url: https://example.com/b\n",
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, _, err := ReadManifest([]byte(input)); err == nil {
				t.Fatal("ReadManifest unexpectedly succeeded")
			}
		})
	}
}

func TestPublicIPPolicyRejectsNonPublicNetworks(t *testing.T) {
	for _, raw := range []string{"127.0.0.1", "10.0.0.1", "169.254.1.1", "192.0.2.1", "192.88.99.1", "100.64.0.1", "::1", "fc00::1", "64:ff9b::1", "2001:db8::1"} {
		if isPublicIP(net.ParseIP(raw)) {
			t.Errorf("isPublicIP(%s) = true", raw)
		}
	}
	if !isPublicIP(net.ParseIP("8.8.8.8")) {
		t.Error("isPublicIP(8.8.8.8) = false")
	}
}

func TestSafeFetcherRejectsLoopbackBeforeConnecting(t *testing.T) {
	_, err := NewSafeFetcher().Fetch(t.Context(), "https://127.0.0.1/recipe")
	if err == nil || !strings.Contains(err.Error(), "non-public") {
		t.Fatalf("Fetch error = %v, want non-public rejection", err)
	}
}

func TestRedirectPolicyRevalidatesURLAndBoundsChain(t *testing.T) {
	previousURL, _ := url.Parse("https://example.com/start")
	previous := &http.Request{URL: previousURL}
	unsafeURL, _ := url.Parse("http://example.com/next")
	redirects := make([]Redirect, 0)
	if err := validateRedirect(&http.Request{URL: unsafeURL}, []*http.Request{previous}, &redirects); err == nil || !strings.Contains(err.Error(), "unsafe redirect") {
		t.Fatalf("unsafe redirect error = %v", err)
	}

	safeURL, _ := url.Parse("https://example.com/next")
	tooMany := make([]*http.Request, maxRedirects+1)
	for index := range tooMany {
		tooMany[index] = previous
	}
	if err := validateRedirect(&http.Request{URL: safeURL}, tooMany, &redirects); err == nil || !strings.Contains(err.Error(), "more than") {
		t.Fatalf("redirect limit error = %v", err)
	}
}
