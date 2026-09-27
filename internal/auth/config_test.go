package auth

import (
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
	"time"
)

func validRawConfig() RawConfig {
	return RawConfig{
		Origin: "https://groceries.example.com", Issuer: "https://accounts.example.com",
		ClientID: "client", ClientSecret: "secret",
		SessionSecret: base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("k", 32))),
		AllowedEmails: "Owner@Example.com, second@example.com", HouseholdName: "Coward",
	}
}

func TestParseConfigFailsClosed(t *testing.T) {
	tests := []struct {
		name   string
		change func(*RawConfig)
	}{
		{"missing origin", func(raw *RawConfig) { raw.Origin = "" }},
		{"insecure remote origin", func(raw *RawConfig) { raw.Origin = "http://groceries.example.com" }},
		{"origin path", func(raw *RawConfig) { raw.Origin += "/app" }},
		{"missing issuer", func(raw *RawConfig) { raw.Issuer = "" }},
		{"missing client", func(raw *RawConfig) { raw.ClientID = "" }},
		{"short session secret", func(raw *RawConfig) { raw.SessionSecret = base64.RawURLEncoding.EncodeToString([]byte("short")) }},
		{"empty allowlist", func(raw *RawConfig) { raw.AllowedEmails = " , " }},
		{"wrong household", func(raw *RawConfig) { raw.HouseholdName = "Other" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw := validRawConfig()
			test.change(&raw)
			if _, err := ParseConfig(raw); err == nil {
				t.Fatal("configuration unexpectedly accepted")
			}
		})
	}
}

func TestConfigAndProtectedLoginTransaction(t *testing.T) {
	config, err := ParseConfig(validRawConfig())
	if err != nil {
		t.Fatal(err)
	}
	if !config.ValidateAllowed(" OWNER@example.com ") {
		t.Fatal("normalized allowlisted email rejected")
	}
	service := NewService(nil, config)
	fixed := time.Unix(1_800_000_000, 0)
	service.now = func() time.Time { return fixed }
	transaction, protected, err := service.NewLoginTransaction("/recipes?q=pasta")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := service.ParseLoginTransaction(protected, transaction.State)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.ReturnTo != "/recipes?q=pasta" || parsed.Nonce == "" || parsed.Verifier == "" {
		t.Fatalf("unexpected transaction: %+v", parsed)
	}
	if _, err := service.ParseLoginTransaction(protected, "wrong"); err == nil {
		t.Fatal("wrong state accepted")
	}
	if _, _, err := service.NewLoginTransaction("https://evil.example/"); err != nil {
		t.Fatal(err)
	}
}

func TestSessionCookies(t *testing.T) {
	config, err := ParseConfig(validRawConfig())
	if err != nil {
		t.Fatal(err)
	}
	cookie := SessionCookie(config, "token", time.Now().Add(time.Hour))
	if cookie.Name != "__Host-grocery_session" || !cookie.Secure || !cookie.HttpOnly || cookie.Path != "/" || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("insecure session cookie: %+v", cookie)
	}
}
