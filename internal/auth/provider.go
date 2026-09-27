package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type ExternalIdentity struct {
	Issuer        string
	Subject       string
	VerifiedEmail string
	DisplayName   string
	AvatarURL     string
}

type Provider interface {
	AuthorizationURL(state, nonce, codeChallenge string) string
	Authenticate(ctx context.Context, code, verifier, expectedNonce string) (ExternalIdentity, error)
}

type GoogleProvider struct {
	issuer string
	oauth  oauth2.Config
	verify *oidc.IDTokenVerifier
}

func NewGoogleProvider(ctx context.Context, config Config) (*GoogleProvider, error) {
	provider, err := oidc.NewProvider(ctx, config.Issuer)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC provider: %w", err)
	}
	return &GoogleProvider{
		issuer: config.Issuer,
		oauth: oauth2.Config{
			ClientID: config.ClientID, ClientSecret: config.ClientSecret,
			Endpoint: provider.Endpoint(), RedirectURL: config.CallbackURL(),
			Scopes: []string{oidc.ScopeOpenID, "email", "profile"},
		},
		verify: provider.Verifier(&oidc.Config{ClientID: config.ClientID}),
	}, nil
}

func (provider *GoogleProvider) AuthorizationURL(state, nonce, codeChallenge string) string {
	return provider.oauth.AuthCodeURL(state,
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

func (provider *GoogleProvider) Authenticate(ctx context.Context, code, verifier, expectedNonce string) (ExternalIdentity, error) {
	token, err := provider.oauth.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
	if err != nil {
		return ExternalIdentity{}, errors.New("OIDC authorization code exchange failed")
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return ExternalIdentity{}, errors.New("OIDC response did not contain an ID token")
	}
	idToken, err := provider.verify.Verify(ctx, rawIDToken)
	if err != nil {
		return ExternalIdentity{}, errors.New("OIDC ID token validation failed")
	}
	if idToken.Nonce == "" || idToken.Nonce != expectedNonce {
		return ExternalIdentity{}, errors.New("OIDC nonce did not match")
	}
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return ExternalIdentity{}, errors.New("OIDC identity claims were invalid")
	}
	if !claims.EmailVerified || normalizeEmail(claims.Email) == "" {
		return ExternalIdentity{}, errors.New("OIDC provider did not verify the email")
	}
	name := claims.Name
	if name == "" {
		name = claims.Email
	}
	return ExternalIdentity{
		Issuer: provider.issuer, Subject: idToken.Subject, VerifiedEmail: normalizeEmail(claims.Email),
		DisplayName: name, AvatarURL: claims.Picture,
	}, nil
}

func PKCEChallenge(verifier string) string {
	digest := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}
