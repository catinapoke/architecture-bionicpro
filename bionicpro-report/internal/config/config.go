package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port              string
	KeycloakURL       string
	KeycloakPublicURL string
	Realm             string
	DBURL             string
}

func Load() (Config, error) {
	cfg := Config{
		Port:        getEnv("API_PORT", "8001"),
		KeycloakURL: getEnv("KEYCLOAK_URL", "http://localhost:8080"),
		Realm:       getEnv("KEYCLOAK_REALM", "reports-realm"),
		DBURL:       os.Getenv("DB_URL"),
	}
	cfg.KeycloakPublicURL = getEnv("KEYCLOAK_PUBLIC_URL", cfg.KeycloakURL)

	if cfg.DBURL == "" {
		return Config{}, fmt.Errorf("DB_URL is required")
	}

	return cfg, nil
}

func (c Config) Issuer() string {
	return fmt.Sprintf("%s/realms/%s", strings.TrimRight(c.KeycloakURL, "/"), c.Realm)
}

func (c Config) PublicIssuer() string {
	return fmt.Sprintf("%s/realms/%s", strings.TrimRight(c.KeycloakPublicURL, "/"), c.Realm)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
