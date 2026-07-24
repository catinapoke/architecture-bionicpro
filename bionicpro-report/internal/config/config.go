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
	S3Endpoint        string
	S3AccessKey       string
	S3SecretKey       string
	S3Bucket          string
	S3UseSSL          bool
	CDNBaseURL        string
}

func Load() (Config, error) {
	cfg := Config{
		Port:        getEnv("API_PORT", "8001"),
		KeycloakURL: getEnv("KEYCLOAK_URL", "http://localhost:8080"),
		Realm:       getEnv("KEYCLOAK_REALM", "reports-realm"),
		DBURL:       os.Getenv("DB_URL"),
		S3Endpoint:  getEnv("S3_ENDPOINT", "localhost:9000"),
		S3AccessKey: getEnv("S3_ACCESS_KEY", "admin"),
		S3SecretKey: getEnv("S3_SECRET_KEY", "adminadmin"),
		S3Bucket:    getEnv("S3_BUCKET", "reports"),
		S3UseSSL:    getEnv("S3_USE_SSL", "false") == "true",
		CDNBaseURL:  getEnv("CDN_BASE_URL", "http://localhost:9990"),
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
