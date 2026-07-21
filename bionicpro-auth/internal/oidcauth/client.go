package oidcauth

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"bionicpro-auth/internal/config"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type Client struct {
	oauth2Config oauth2.Config
	logoutURL    string
}

func New(ctx context.Context, cfg config.Config) (*Client, error) {
	issuer := cfg.Issuer()
	// Keycloak may advertise localhost issuer while we discover via Docker DNS.
	if cfg.KeycloakInternalURL != cfg.KeycloakURL {
		ctx = oidc.InsecureIssuerURLContext(ctx, cfg.PublicIssuer())
	}

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc provider: %w", err)
	}

	endpoint := provider.Endpoint()
	// Browser must hit the public Keycloak URL when internal != public (Docker).
	if cfg.KeycloakInternalURL != cfg.KeycloakURL {
		endpoint.AuthURL = strings.Replace(endpoint.AuthURL, cfg.KeycloakInternalURL, cfg.KeycloakURL, 1)
		endpoint.TokenURL = strings.Replace(endpoint.TokenURL, cfg.KeycloakURL, cfg.KeycloakInternalURL, 1)
	}

	return &Client{
		oauth2Config: oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint:     endpoint,
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		logoutURL: cfg.PublicIssuer() + "/protocol/openid-connect/logout",
	}, nil
}

// LogoutURL builds an RP-initiated logout URL for the browser.
func (c *Client) LogoutURL(idToken, postLogoutRedirectURI string) string {
	u, err := url.Parse(c.logoutURL)
	if err != nil {
		return c.logoutURL
	}
	q := u.Query()
	q.Set("client_id", c.oauth2Config.ClientID)
	q.Set("post_logout_redirect_uri", postLogoutRedirectURI)
	if idToken != "" {
		q.Set("id_token_hint", idToken)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func IDToken(tok *oauth2.Token) string {
	if tok == nil {
		return ""
	}
	raw, _ := tok.Extra("id_token").(string)
	return raw
}

func (c *Client) AuthCodeURL(state, codeChallenge string) string {
	return c.oauth2Config.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

func (c *Client) Exchange(ctx context.Context, code, codeVerifier string) (*oauth2.Token, error) {
	return c.oauth2Config.Exchange(
		ctx,
		code,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
	)
}

func (c *Client) Refresh(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	src := c.oauth2Config.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	return src.Token()
}

func AccessExpiry(tok *oauth2.Token) time.Time {
	if !tok.Expiry.IsZero() {
		return tok.Expiry
	}
	return time.Now().Add(2 * time.Minute)
}
