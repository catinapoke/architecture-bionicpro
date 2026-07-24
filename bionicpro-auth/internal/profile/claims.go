package profile

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

// FromIDToken builds a Profile from an OIDC ID token JWT payload (no signature check).
// Used after Keycloak brokering (e.g. Yandex ID) to persist profile locally.
func FromIDToken(idToken string) (Profile, bool) {
	if idToken == "" {
		return Profile{}, false
	}

	parts := strings.Split(idToken, ".")
	if len(parts) < 2 {
		return Profile{}, false
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Profile{}, false
	}

	var claims struct {
		Subject           string `json:"sub"`
		PreferredUsername string `json:"preferred_username"`
		Email             string `json:"email"`
		Name              string `json:"name"`
		Issuer            string `json:"iss"`
		IdentityProvider  string `json:"identity_provider"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Profile{}, false
	}
	if claims.Subject == "" {
		return Profile{}, false
	}

	provider := claims.IdentityProvider
	if provider == "" {
		provider = "keycloak"
	}

	return Profile{
		Subject:   claims.Subject,
		Username:  claims.PreferredUsername,
		Email:     claims.Email,
		Name:      claims.Name,
		Issuer:    claims.Issuer,
		Provider:  provider,
		RawClaims: string(payload),
	}, true
}
