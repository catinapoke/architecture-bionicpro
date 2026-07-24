package keycloak

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"

	"bionicpro-report/internal/config"
)

type Verifier struct {
	verifier *oidc.IDTokenVerifier
}

func NewVerifier(ctx context.Context, cfg config.Config) (*Verifier, error) {
	ctx = oidc.InsecureIssuerURLContext(ctx, cfg.PublicIssuer())
	provider, err := oidc.NewProvider(ctx, cfg.Issuer())
	if err != nil {
		return nil, err
	}

	return &Verifier{
		verifier: provider.Verifier(&oidc.Config{
			SkipClientIDCheck: true,
		}),
	}, nil
}

func (v *Verifier) Username(ctx context.Context, token string) (string, error) {
	verified, err := v.verifier.Verify(ctx, token)
	if err != nil {
		return "", err
	}

	var claims struct {
		PreferredUsername string `json:"preferred_username"`
		Subject           string `json:"sub"`
	}
	if err := verified.Claims(&claims); err != nil {
		return "", err
	}

	if claims.PreferredUsername != "" {
		return claims.PreferredUsername, nil
	}
	if claims.Subject != "" {
		return claims.Subject, nil
	}
	return "", fmt.Errorf("token has no username claim")
}
