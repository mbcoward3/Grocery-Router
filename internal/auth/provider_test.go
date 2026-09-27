package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

type fakeOIDC struct {
	mu            sync.Mutex
	issuer        string
	clientID      string
	key           *rsa.PrivateKey
	signingKey    *rsa.PrivateKey
	kid           string
	tokenOverride string
	claims        func(string) jwt.Claims
}

func newFakeOIDC(t *testing.T) (*fakeOIDC, *httptest.Server) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeOIDC{clientID: "test-client", key: key, kid: "key-1"}
	server := httptest.NewServer(http.HandlerFunc(fake.serveHTTP))
	fake.issuer = server.URL
	fake.claims = func(nonce string) jwt.Claims {
		now := time.Now()
		return jwt.Claims{Issuer: fake.issuer, Subject: "google-subject", Audience: jwt.Audience{fake.clientID}, IssuedAt: jwt.NewNumericDate(now.Add(-time.Minute)), Expiry: jwt.NewNumericDate(now.Add(time.Hour))}
	}
	t.Cleanup(server.Close)
	return fake, server
}

func (fake *fakeOIDC) serveHTTP(response http.ResponseWriter, request *http.Request) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	switch request.URL.Path {
	case "/.well-known/openid-configuration":
		_ = json.NewEncoder(response).Encode(map[string]any{
			"issuer": fake.issuer, "authorization_endpoint": fake.issuer + "/authorize",
			"token_endpoint": fake.issuer + "/token", "jwks_uri": fake.issuer + "/jwks",
			"response_types_supported": []string{"code"}, "subject_types_supported": []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	case "/jwks":
		_ = json.NewEncoder(response).Encode(map[string]any{"keys": []jose.JSONWebKey{{Key: &fake.key.PublicKey, KeyID: fake.kid, Algorithm: string(jose.RS256), Use: "sig"}}})
	case "/token":
		if request.FormValue("code_verifier") != "verifier" {
			http.Error(response, "missing PKCE verifier", http.StatusBadRequest)
			return
		}
		nonce := request.FormValue("nonce_for_test")
		if nonce == "" {
			nonce = "expected-nonce"
		}
		claims := fake.claims(nonce)
		signingKey := fake.key
		if fake.signingKey != nil {
			signingKey = fake.signingKey
		}
		signer, _ := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: jose.JSONWebKey{Key: signingKey, KeyID: fake.kid}}, nil)
		raw, _ := jwt.Signed(signer).Claims(claims).Claims(map[string]any{
			"nonce": nonce, "email": "owner@example.com", "email_verified": true,
			"name": "Owner", "picture": "https://example.com/avatar.png",
		}).Serialize()
		if fake.tokenOverride != "" {
			raw = fake.tokenOverride
		}
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(map[string]any{"access_token": "not-stored", "token_type": "Bearer", "expires_in": 60, "id_token": raw})
	default:
		http.NotFound(response, request)
	}
}

func mustURL(value string) *url.URL {
	parsed, err := url.Parse(value)
	if err != nil {
		panic(err)
	}
	return parsed
}

func fakeProviderConfig(fake *fakeOIDC) Config {
	origin := mustURL("https://groceries.example.com")
	return Config{Origin: origin, Issuer: fake.issuer, ClientID: fake.clientID, ClientSecret: "secret"}
}

func TestGoogleProviderValidatesOIDCClaimsAndNonce(t *testing.T) {
	fake, _ := newFakeOIDC(t)
	provider, err := NewGoogleProvider(context.Background(), fakeProviderConfig(fake))
	if err != nil {
		t.Fatal(err)
	}
	identity, err := provider.Authenticate(context.Background(), "code", "verifier", "expected-nonce")
	if err != nil {
		t.Fatal(err)
	}
	if identity.Subject != "google-subject" || identity.VerifiedEmail != "owner@example.com" {
		t.Fatalf("unexpected identity: %+v", identity)
	}
	if _, err := provider.Authenticate(context.Background(), "code", "verifier", "wrong-nonce"); err == nil {
		t.Fatal("wrong nonce accepted")
	}
}

func TestGoogleProviderRejectsInvalidTokens(t *testing.T) {
	tests := []struct {
		name  string
		alter func(*fakeOIDC)
	}{
		{"wrong issuer", func(fake *fakeOIDC) {
			previous := fake.claims
			fake.claims = func(n string) jwt.Claims { c := previous(n); c.Issuer = "https://wrong.example"; return c }
		}},
		{"wrong audience", func(fake *fakeOIDC) {
			previous := fake.claims
			fake.claims = func(n string) jwt.Claims { c := previous(n); c.Audience = jwt.Audience{"wrong-client"}; return c }
		}},
		{"malformed", func(fake *fakeOIDC) { fake.tokenOverride = "not-a-jwt" }},
		{"wrong signature", func(fake *fakeOIDC) { fake.signingKey, _ = rsa.GenerateKey(rand.Reader, 2048) }},
		{"expired", func(fake *fakeOIDC) {
			previous := fake.claims
			fake.claims = func(n string) jwt.Claims {
				c := previous(n)
				c.Expiry = jwt.NewNumericDate(time.Now().Add(-time.Hour))
				return c
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fake, _ := newFakeOIDC(t)
			test.alter(fake)
			provider, err := NewGoogleProvider(context.Background(), fakeProviderConfig(fake))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := provider.Authenticate(context.Background(), "code", "verifier", "expected-nonce"); err == nil {
				t.Fatal("invalid token accepted")
			}
		})
	}
}

func TestGoogleProviderRefreshesRotatedJWKS(t *testing.T) {
	fake, _ := newFakeOIDC(t)
	provider, err := NewGoogleProvider(context.Background(), fakeProviderConfig(fake))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Authenticate(context.Background(), "code", "verifier", "expected-nonce"); err != nil {
		t.Fatal(err)
	}
	rotated, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	fake.mu.Lock()
	fake.key = rotated
	fake.kid = "key-2"
	fake.mu.Unlock()
	if _, err := provider.Authenticate(context.Background(), "code", "verifier", "expected-nonce"); err != nil {
		t.Fatalf("rotated key rejected: %v", err)
	}
}
