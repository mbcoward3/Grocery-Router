// Package auth implements provider-independent users, Google OIDC, and opaque sessions.
package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	Origin           *url.URL
	Issuer           string
	ClientID         string
	ClientSecret     string
	SessionKey       []byte
	AllowedEmails    map[string]struct{}
	IdleLifetime     time.Duration
	AbsoluteLifetime time.Duration
	SecureCookies    bool
}

type RawConfig struct {
	Origin, Issuer, ClientID, ClientSecret, SessionSecret, AllowedEmails, HouseholdName string
}

func ParseConfig(raw RawConfig) (Config, error) {
	origin, err := url.Parse(strings.TrimSpace(raw.Origin))
	if err != nil || origin.Scheme == "" || origin.Host == "" || origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" {
		return Config{}, errors.New("GROCERY_ROUTER_AUTH_ORIGIN must be an origin without a path, query, or fragment")
	}
	secure := origin.Scheme == "https"
	if !secure && !(origin.Scheme == "http" && (origin.Hostname() == "localhost" || origin.Hostname() == "127.0.0.1")) {
		return Config{}, errors.New("GROCERY_ROUTER_AUTH_ORIGIN must use HTTPS except on localhost")
	}
	issuer := strings.TrimSuffix(strings.TrimSpace(raw.Issuer), "/")
	issuerURL, err := url.Parse(issuer)
	if err != nil || issuerURL.Scheme != "https" || issuerURL.Host == "" {
		return Config{}, errors.New("GROCERY_ROUTER_AUTH_OIDC_ISSUER must be an HTTPS URL")
	}
	if strings.TrimSpace(raw.ClientID) == "" || strings.TrimSpace(raw.ClientSecret) == "" {
		return Config{}, errors.New("Google OIDC client ID and secret are required")
	}
	key, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(raw.SessionSecret))
	if err != nil || len(key) < 32 {
		return Config{}, errors.New("GROCERY_ROUTER_AUTH_SESSION_SECRET must be base64url-encoded and decode to at least 32 bytes")
	}
	allowed, err := parseAllowedEmails(raw.AllowedEmails)
	if err != nil || len(allowed) == 0 {
		return Config{}, errors.New("GROCERY_ROUTER_AUTH_BOOTSTRAP_USERS must contain at least one valid email")
	}
	if strings.TrimSpace(raw.HouseholdName) != "Coward" {
		return Config{}, errors.New("GROCERY_ROUTER_AUTH_BOOTSTRAP_HOUSEHOLD must be Coward")
	}
	return Config{
		Origin: origin, Issuer: issuer, ClientID: strings.TrimSpace(raw.ClientID),
		ClientSecret: strings.TrimSpace(raw.ClientSecret), SessionKey: key, AllowedEmails: allowed,
		IdleLifetime: 30 * 24 * time.Hour, AbsoluteLifetime: 90 * 24 * time.Hour, SecureCookies: secure,
	}, nil
}

func (config Config) CallbackURL() string {
	return config.Origin.String() + "/api/v2/auth/google/callback"
}

func (config Config) CookieName() string {
	if config.SecureCookies {
		return "__Host-grocery_session"
	}
	return "grocery_session_dev"
}

func (config Config) ValidateAllowed(email string) bool {
	_, ok := config.AllowedEmails[normalizeEmail(email)]
	return ok
}

func normalizeEmail(value string) string { return strings.ToLower(strings.TrimSpace(value)) }

func parseAllowedEmails(value string) (map[string]struct{}, error) {
	allowed := make(map[string]struct{})
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "[") {
		var users []struct {
			Email string `json:"email"`
			Role  string `json:"role"`
		}
		if err := json.Unmarshal([]byte(value), &users); err != nil {
			return nil, err
		}
		for _, user := range users {
			if email := normalizeEmail(user.Email); email != "" && user.Role == "owner" {
				allowed[email] = struct{}{}
			}
		}
		return allowed, nil
	}
	for _, item := range strings.Split(value, ",") {
		if email := normalizeEmail(item); email != "" {
			allowed[email] = struct{}{}
		}
	}
	return allowed, nil
}

func (config Config) String() string {
	return fmt.Sprintf("origin=%s issuer=%s", config.Origin, config.Issuer)
}
