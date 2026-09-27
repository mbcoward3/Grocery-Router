package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type contextKey struct{}

func SessionFromContext(ctx context.Context) (Session, bool) {
	session, ok := ctx.Value(contextKey{}).(Session)
	return session, ok
}

type HTTPHandler struct {
	config   Config
	service  *Service
	provider Provider
	limiter  *rateLimiter
}

func NewHTTPHandler(config Config, service *Service, provider Provider) *HTTPHandler {
	return &HTTPHandler{config: config, service: service, provider: provider, limiter: newRateLimiter(20, time.Minute)}
}

func (handler *HTTPHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v2/auth/google/start", handler.googleStart)
	mux.HandleFunc("GET /api/v2/auth/google/callback", handler.googleCallback)
	mux.HandleFunc("POST /api/v2/auth/logout", handler.logout)
	mux.Handle("GET /api/v2/session", handler.RequireSession(http.HandlerFunc(handler.session)))
}

func (handler *HTTPHandler) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		cookie, err := request.Cookie(handler.config.CookieName())
		if err != nil {
			writeAuthError(response, http.StatusUnauthorized, "authentication_required", "Sign in is required.")
			return
		}
		session, err := handler.service.Authenticate(request.Context(), cookie.Value)
		if err != nil {
			if errors.Is(err, ErrUnauthenticated) {
				http.SetCookie(response, ExpiredSessionCookie(handler.config))
			}
			writeAuthError(response, http.StatusUnauthorized, "authentication_required", "Sign in is required.")
			return
		}
		if request.Method != http.MethodGet && request.Method != http.MethodHead && request.Method != http.MethodOptions && !handler.trustedOrigin(request) {
			writeAuthError(response, http.StatusForbidden, "untrusted_origin", "The request origin was not trusted.")
			return
		}
		next.ServeHTTP(response, request.WithContext(context.WithValue(request.Context(), contextKey{}, session)))
	})
}

// RequireHousehold authorizes the explicit path tenant without disclosing other households.
func (handler *HTTPHandler) RequireHousehold(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		session, ok := SessionFromContext(request.Context())
		if !ok {
			writeAuthError(response, http.StatusUnauthorized, "authentication_required", "Sign in is required.")
			return
		}
		householdID := request.PathValue("householdID")
		for _, membership := range session.Memberships {
			if membership.HouseholdID == householdID {
				next.ServeHTTP(response, request)
				return
			}
		}
		response.WriteHeader(http.StatusNotFound)
	})
}

func (handler *HTTPHandler) googleStart(response http.ResponseWriter, request *http.Request) {
	if !handler.limiter.Allow(clientAddress(request)) {
		writeAuthError(response, http.StatusTooManyRequests, "rate_limited", "Try again later.")
		return
	}
	transaction, protected, err := handler.service.NewLoginTransaction(request.URL.Query().Get("returnTo"))
	if err != nil {
		writeAuthError(response, http.StatusInternalServerError, "auth_unavailable", "Sign in is unavailable.")
		return
	}
	http.SetCookie(response, &http.Cookie{
		Name: handler.transactionCookieName(), Value: protected, Path: "/api/v2/auth/google/callback",
		Secure: handler.config.SecureCookies, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 600,
	})
	response.Header().Set("Cache-Control", "no-store")
	http.Redirect(response, request, handler.provider.AuthorizationURL(transaction.State, transaction.Nonce, PKCEChallenge(transaction.Verifier)), http.StatusFound)
}

func (handler *HTTPHandler) googleCallback(response http.ResponseWriter, request *http.Request) {
	if !handler.limiter.Allow(clientAddress(request)) {
		handler.authFailure(response, http.StatusTooManyRequests)
		return
	}
	cookie, err := request.Cookie(handler.transactionCookieName())
	if err != nil || request.URL.Query().Get("error") != "" {
		handler.authFailure(response, http.StatusUnauthorized)
		return
	}
	transaction, err := handler.service.ParseLoginTransaction(cookie.Value, request.URL.Query().Get("state"))
	if err != nil {
		handler.authFailure(response, http.StatusUnauthorized)
		return
	}
	identity, err := handler.provider.Authenticate(request.Context(), request.URL.Query().Get("code"), transaction.Verifier, transaction.Nonce)
	if err != nil {
		handler.authFailure(response, http.StatusUnauthorized)
		return
	}
	user, err := handler.service.ResolveIdentity(request.Context(), identity)
	if err != nil {
		handler.authFailure(response, http.StatusForbidden)
		return
	}
	token, err := handler.service.CreateSession(request.Context(), user.ID, request.UserAgent())
	if err != nil {
		handler.authFailure(response, http.StatusInternalServerError)
		return
	}
	http.SetCookie(response, SessionCookie(handler.config, token, time.Now().Add(handler.config.AbsoluteLifetime)))
	handler.clearTransaction(response)
	response.Header().Set("Cache-Control", "no-store")
	http.Redirect(response, request, transaction.ReturnTo, http.StatusSeeOther)
}

func (handler *HTTPHandler) logout(response http.ResponseWriter, request *http.Request) {
	if !handler.trustedOrigin(request) {
		writeAuthError(response, http.StatusForbidden, "untrusted_origin", "The request origin was not trusted.")
		return
	}
	if cookie, err := request.Cookie(handler.config.CookieName()); err == nil {
		_ = handler.service.Revoke(request.Context(), cookie.Value)
	}
	http.SetCookie(response, ExpiredSessionCookie(handler.config))
	response.Header().Set("Cache-Control", "no-store")
	response.WriteHeader(http.StatusNoContent)
}

func (handler *HTTPHandler) session(response http.ResponseWriter, request *http.Request) {
	session, _ := SessionFromContext(request.Context())
	writeAuthJSON(response, http.StatusOK, map[string]any{
		"user":       map[string]any{"id": session.User.ID, "email": session.User.Email, "displayName": session.User.DisplayName, "avatarUrl": session.User.AvatarURL},
		"households": session.Memberships,
	})
}

func (handler *HTTPHandler) trustedOrigin(request *http.Request) bool {
	origin := strings.TrimSuffix(request.Header.Get("Origin"), "/")
	return origin != "" && origin == handler.config.Origin.String()
}

func (handler *HTTPHandler) authFailure(response http.ResponseWriter, status int) {
	handler.clearTransaction(response)
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.WriteHeader(status)
	_, _ = response.Write([]byte("<!doctype html><title>Sign in failed</title><h1>Sign in failed</h1><p>This account cannot access Grocery Router.</p>"))
}

func (handler *HTTPHandler) transactionCookieName() string {
	if handler.config.SecureCookies {
		return "__Host-grocery_auth"
	}
	return "grocery_auth_dev"
}

func (handler *HTTPHandler) clearTransaction(response http.ResponseWriter) {
	http.SetCookie(response, &http.Cookie{Name: handler.transactionCookieName(), Value: "", Path: "/api/v2/auth/google/callback", Secure: handler.config.SecureCookies, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0)})
}

func writeAuthError(response http.ResponseWriter, status int, code, message string) {
	writeAuthJSON(response, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func writeAuthJSON(response http.ResponseWriter, status int, body any) {
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(body)
}

func clientAddress(request *http.Request) string {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil {
		return host
	}
	return request.RemoteAddr
}

type rateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	entries map[string]rateEntry
}
type rateEntry struct {
	start time.Time
	count int
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, entries: make(map[string]rateEntry)}
}
func (limiter *rateLimiter) Allow(key string) bool {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	now := time.Now()
	entry := limiter.entries[key]
	if entry.start.IsZero() || now.Sub(entry.start) >= limiter.window {
		entry = rateEntry{start: now}
	}
	entry.count++
	limiter.entries[key] = entry
	return entry.count <= limiter.limit
}
