package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mbcoward3/grocery-router/internal/tenant"
)

var (
	ErrUnauthenticated = errors.New("authentication required")
	ErrForbidden       = errors.New("household access denied")
	ErrNotAllowed      = errors.New("identity is not allowed")
	ErrLinkRequired    = errors.New("an existing account must be linked while authenticated")
)

type User struct {
	ID, Email, DisplayName string
	AvatarURL              *string
}

type Membership struct {
	HouseholdID   string `json:"householdID"`
	HouseholdName string `json:"householdName"`
	Role          string `json:"role"`
}

type Session struct {
	ID          string
	User        User
	Memberships []Membership
}

type Service struct {
	db     *sql.DB
	config Config
	now    func() time.Time
}

func NewService(db *sql.DB, config Config) *Service {
	return &Service{db: db, config: config, now: time.Now}
}

func (service *Service) ResolveIdentity(ctx context.Context, identity ExternalIdentity) (User, error) {
	if !service.config.ValidateAllowed(identity.VerifiedEmail) {
		return User{}, ErrNotAllowed
	}
	tx, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, fmt.Errorf("begin identity transaction: %w", err)
	}
	defer tx.Rollback()

	var user User
	var avatar sql.NullString
	err = tx.QueryRowContext(ctx, `
		SELECT u.id::text, u.primary_email, u.display_name, u.avatar_url
		FROM user_identities i JOIN app_users u ON u.id = i.user_id
		WHERE i.issuer = $1 AND i.subject = $2
		FOR UPDATE OF i, u`, identity.Issuer, identity.Subject,
	).Scan(&user.ID, &user.Email, &user.DisplayName, &avatar)
	switch {
	case err == nil:
		if identity.AvatarURL == "" {
			avatar = sql.NullString{}
		} else {
			avatar = sql.NullString{String: identity.AvatarURL, Valid: true}
		}
		_, err = tx.ExecContext(ctx, `
			UPDATE app_users SET primary_email = $1, display_name = $2, avatar_url = $3,
				updated_at = clock_timestamp(), last_seen_at = clock_timestamp()
			WHERE id = $4 AND disabled_at IS NULL`, identity.VerifiedEmail, identity.DisplayName, avatar, user.ID)
		if err != nil {
			return User{}, fmt.Errorf("update application user: %w", err)
		}
		var enabled bool
		if err = tx.QueryRowContext(ctx, `SELECT disabled_at IS NULL FROM app_users WHERE id = $1`, user.ID).Scan(&enabled); err != nil || !enabled {
			return User{}, ErrNotAllowed
		}
		result, execErr := tx.ExecContext(ctx, `
			UPDATE user_identities SET provider_email = $1, provider_email_verified = TRUE,
				last_seen_at = clock_timestamp() WHERE issuer = $2 AND subject = $3`,
			identity.VerifiedEmail, identity.Issuer, identity.Subject)
		if execErr != nil {
			return User{}, fmt.Errorf("update external identity: %w", execErr)
		}
		_ = result
	case errors.Is(err, sql.ErrNoRows):
		var collision bool
		if scanErr := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM app_users WHERE primary_email = $1)`, identity.VerifiedEmail).Scan(&collision); scanErr != nil {
			return User{}, fmt.Errorf("check account collision: %w", scanErr)
		}
		if collision {
			return User{}, ErrLinkRequired
		}
		if identity.AvatarURL != "" {
			avatar = sql.NullString{String: identity.AvatarURL, Valid: true}
		}
		err = tx.QueryRowContext(ctx, `
			INSERT INTO app_users (primary_email, display_name, avatar_url) VALUES ($1, $2, $3)
			RETURNING id::text, primary_email, display_name, avatar_url`,
			identity.VerifiedEmail, identity.DisplayName, avatar,
		).Scan(&user.ID, &user.Email, &user.DisplayName, &avatar)
		if err != nil {
			return User{}, fmt.Errorf("create application user: %w", err)
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO user_identities (user_id, provider, issuer, subject, provider_email, provider_email_verified)
			VALUES ($1, 'google', $2, $3, $4, TRUE)`, user.ID, identity.Issuer, identity.Subject, identity.VerifiedEmail)
		if err != nil {
			return User{}, fmt.Errorf("create external identity: %w", err)
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO household_memberships (household_id, user_id, role) VALUES ($1, $2, 'owner')
			ON CONFLICT (household_id, user_id) DO NOTHING`, tenant.CowardHouseholdID, user.ID)
		if err != nil {
			return User{}, fmt.Errorf("claim household membership: %w", err)
		}
	default:
		return User{}, fmt.Errorf("resolve external identity: %w", err)
	}
	if avatar.Valid {
		user.AvatarURL = &avatar.String
	}
	if err := tx.Commit(); err != nil {
		return User{}, fmt.Errorf("commit identity transaction: %w", err)
	}
	return user, nil
}

func (service *Service) CreateSession(ctx context.Context, userID, metadata string) (string, error) {
	token, err := randomToken(32)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(token))
	now := service.now().UTC()
	_, err = service.db.ExecContext(ctx, `
		INSERT INTO auth_sessions (user_id, token_digest, idle_expires_at, absolute_expires_at, client_metadata)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''))`, userID, digest[:], now.Add(service.config.IdleLifetime), now.Add(service.config.AbsoluteLifetime), truncate(metadata, 512))
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return token, nil
}

func (service *Service) Authenticate(ctx context.Context, token string) (Session, error) {
	if token == "" {
		return Session{}, ErrUnauthenticated
	}
	digest := sha256.Sum256([]byte(token))
	now := service.now().UTC()
	var session Session
	var avatar sql.NullString
	var idle, absolute time.Time
	err := service.db.QueryRowContext(ctx, `
		SELECT s.id::text, u.id::text, u.primary_email, u.display_name, u.avatar_url,
			s.idle_expires_at, s.absolute_expires_at
		FROM auth_sessions s JOIN app_users u ON u.id = s.user_id
		WHERE s.token_digest = $1 AND s.revoked_at IS NULL AND u.disabled_at IS NULL`, digest[:],
	).Scan(&session.ID, &session.User.ID, &session.User.Email, &session.User.DisplayName, &avatar, &idle, &absolute)
	if errors.Is(err, sql.ErrNoRows) || err == nil && (now.After(idle) || now.After(absolute)) {
		return Session{}, ErrUnauthenticated
	}
	if err != nil {
		return Session{}, fmt.Errorf("load session: %w", err)
	}
	if !service.config.ValidateAllowed(session.User.Email) {
		return Session{}, ErrUnauthenticated
	}
	if avatar.Valid {
		session.User.AvatarURL = &avatar.String
	}
	rows, err := service.db.QueryContext(ctx, `
		SELECT h.id::text, h.name, m.role FROM household_memberships m
		JOIN households h ON h.id = m.household_id WHERE m.user_id = $1 ORDER BY h.name`, session.User.ID)
	if err != nil {
		return Session{}, fmt.Errorf("load memberships: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var membership Membership
		if err := rows.Scan(&membership.HouseholdID, &membership.HouseholdName, &membership.Role); err != nil {
			return Session{}, err
		}
		session.Memberships = append(session.Memberships, membership)
	}
	if err := rows.Err(); err != nil {
		return Session{}, err
	}
	newIdle := now.Add(service.config.IdleLifetime)
	if newIdle.After(absolute) {
		newIdle = absolute
	}
	_, err = service.db.ExecContext(ctx, `UPDATE auth_sessions SET last_seen_at = $1, idle_expires_at = $2 WHERE id = $3`, now, newIdle, session.ID)
	if err != nil {
		return Session{}, fmt.Errorf("renew session: %w", err)
	}
	return session, nil
}

func (service *Service) Revoke(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	digest := sha256.Sum256([]byte(token))
	_, err := service.db.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at = clock_timestamp() WHERE token_digest = $1 AND revoked_at IS NULL`, digest[:])
	return err
}

type loginTransaction struct {
	State, Nonce, Verifier, ReturnTo string
	ExpiresAt                        int64
}

func (service *Service) NewLoginTransaction(returnTo string) (loginTransaction, string, error) {
	if !validReturnPath(returnTo) {
		returnTo = "/"
	}
	state, err := randomToken(32)
	if err != nil {
		return loginTransaction{}, "", err
	}
	nonce, err := randomToken(32)
	if err != nil {
		return loginTransaction{}, "", err
	}
	verifier, err := randomToken(48)
	if err != nil {
		return loginTransaction{}, "", err
	}
	transaction := loginTransaction{State: state, Nonce: nonce, Verifier: verifier, ReturnTo: returnTo, ExpiresAt: service.now().Add(10 * time.Minute).Unix()}
	payload, err := service.signTransaction(transaction)
	return transaction, payload, err
}

func (service *Service) ParseLoginTransaction(value, state string) (loginTransaction, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return loginTransaction{}, ErrUnauthenticated
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return loginTransaction{}, ErrUnauthenticated
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return loginTransaction{}, ErrUnauthenticated
	}
	mac := hmac.New(sha256.New, service.config.SessionKey)
	_, _ = mac.Write(body)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return loginTransaction{}, ErrUnauthenticated
	}
	var transaction loginTransaction
	if json.Unmarshal(body, &transaction) != nil || transaction.ExpiresAt < service.now().Unix() || transaction.State != state || !validReturnPath(transaction.ReturnTo) {
		return loginTransaction{}, ErrUnauthenticated
	}
	return transaction, nil
}

func (service *Service) signTransaction(transaction loginTransaction) (string, error) {
	body, err := json.Marshal(transaction)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, service.config.SessionKey)
	_, _ = mac.Write(body)
	return base64.RawURLEncoding.EncodeToString(body) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func SessionCookie(config Config, token string, expires time.Time) *http.Cookie {
	return &http.Cookie{Name: config.CookieName(), Value: token, Path: "/", Secure: config.SecureCookies, HttpOnly: true, SameSite: http.SameSiteLaxMode, Expires: expires, MaxAge: int(time.Until(expires).Seconds())}
}

func ExpiredSessionCookie(config Config) *http.Cookie {
	cookie := SessionCookie(config, "", time.Unix(1, 0))
	cookie.MaxAge = -1
	return cookie
}

func validReturnPath(value string) bool {
	if value == "" {
		return true
	}
	parsed, err := url.Parse(value)
	return err == nil && strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") && parsed.IsAbs() == false && parsed.Host == ""
}

func randomToken(bytes int) (string, error) {
	value := make([]byte, bytes)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func truncate(value string, max int) string {
	if len(value) > max {
		return value[:max]
	}
	return value
}
