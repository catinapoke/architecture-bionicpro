package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                string
	KeycloakURL         string // для фронта
	KeycloakInternalURL string // для контейнера
	Realm               string
	ClientID            string
	ClientSecret        string
	RedirectURL         string
	FrontendURL         string
	CookieName          string
	CookieSecure        bool
	SessionTTL          time.Duration
	CORSOrigin          string
	DatabaseURL         string
}

func Load() (Config, error) {
	cfg := Config{
		Port:                getEnv("API_PORT", "8000"),
		KeycloakURL:         getEnv("KEYCLOAK_URL", "http://localhost:8080"),
		KeycloakInternalURL: getEnv("KEYCLOAK_INTERNAL_URL", ""),
		Realm:               getEnv("KEYCLOAK_REALM", "reports-realm"),
		ClientID:            getEnv("KEYCLOAK_CLIENT_ID", "reports-api"),
		ClientSecret:        getEnv("KEYCLOAK_CLIENT_SECRET", ""),
		RedirectURL:         getEnv("REDIRECT_URL", "http://localhost:8000/auth/callback"),
		FrontendURL:         getEnv("FRONTEND_URL", "http://localhost:3000"),
		CookieName:          getEnv("COOKIE_NAME", "SESSION_ID"),
		CookieSecure:        getEnvBool("COOKIE_SECURE", false),
		SessionTTL:          getEnvDuration("SESSION_TTL", 30*time.Minute),
		CORSOrigin:          getEnv("CORS_ORIGIN", "http://localhost:3000"),
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://auth_user:auth_password@localhost:5434/auth_db?sslmode=disable"),
	}

	if cfg.KeycloakInternalURL == "" {
		cfg.KeycloakInternalURL = cfg.KeycloakURL
	}
	if cfg.ClientSecret == "" {
		return Config{}, fmt.Errorf("KEYCLOAK_CLIENT_SECRET is required")
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func (c Config) Issuer() string {
	return fmt.Sprintf("%s/realms/%s", c.KeycloakInternalURL, c.Realm)
}

func (c Config) PublicIssuer() string {
	return fmt.Sprintf("%s/realms/%s", c.KeycloakURL, c.Realm)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
