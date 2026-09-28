package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeProvider struct{ authorizationURL string }

func (provider fakeProvider) AuthorizationURL(_, _, _ string) string {
	return provider.authorizationURL
}
func (provider fakeProvider) Authenticate(context.Context, string, string, string) (ExternalIdentity, error) {
	return ExternalIdentity{}, nil
}

func TestGoogleStartCreatesProtectedTransaction(t *testing.T) {
	config, err := ParseConfig(validRawConfig())
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHTTPHandler(config, NewService(nil, config), fakeProvider{authorizationURL: "https://accounts.example.com/auth"})
	mux := http.NewServeMux()
	handler.Register(mux)
	request := httptest.NewRequest(http.MethodGet, "/api/v2/auth/google/start?returnTo=%2Frecipes", nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusFound || response.Header().Get("Location") != "https://accounts.example.com/auth" {
		t.Fatalf("unexpected response: %d %s", response.Code, response.Header().Get("Location"))
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "__Host-grocery_auth" || cookies[0].Path != "/" || !cookies[0].HttpOnly || !cookies[0].Secure {
		t.Fatalf("unexpected transaction cookie: %+v", cookies)
	}
}

func TestLogoutRequiresTrustedOrigin(t *testing.T) {
	config, err := ParseConfig(validRawConfig())
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHTTPHandler(config, NewService(nil, config), fakeProvider{})
	request := httptest.NewRequest(http.MethodPost, "/api/v2/auth/logout", strings.NewReader(""))
	response := httptest.NewRecorder()
	handler.logout(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d", response.Code)
	}
}
