package config

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	ServerPort          string `validate:"required"`
	DatabaseURL         string `validate:"required"`
	KeycloakURL         string `validate:"required,url"`
	KeycloakRealm       string `validate:"required"`
	KeycloakClientID    string `validate:"required"`
	LogLevel            string
	OIDCSkipIssuerCheck bool
	S3Endpoint          string `validate:"required"`
	S3Bucket            string `validate:"required"`
	S3AccessKey         string `validate:"required"`
	S3SecretKey         string `validate:"required"`
	S3UseSSL            bool
}

// Load reads configuration from environment variables and validates required fields.
func Load() (Config, error) {
	cfg := Config{
		ServerPort:          getEnvOrDefault("SERVER_PORT", "8080"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		KeycloakURL:         os.Getenv("KEYCLOAK_URL"),
		KeycloakRealm:       os.Getenv("KEYCLOAK_REALM"),
		KeycloakClientID:    os.Getenv("KEYCLOAK_CLIENT_ID"),
		LogLevel:            getEnvOrDefault("LOG_LEVEL", "info"),
		OIDCSkipIssuerCheck: os.Getenv("OIDC_SKIP_ISSUER_CHECK") == "true",
		// Endpoint/bucket defaults match the docker-compose minio service.
		// The credentials have NO source-code default (docs/19 secret
		// management rule 1): required-field validation fails the boot when
		// they are absent — the compose stacks inject them via environment.
		S3Endpoint:  getEnvOrDefault("S3_ENDPOINT", "localhost:9000"),
		S3Bucket:    getEnvOrDefault("S3_BUCKET", "chesed-docs"),
		S3AccessKey: os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey: os.Getenv("S3_SECRET_KEY"),
		S3UseSSL:    os.Getenv("S3_USE_SSL") == "true",
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return Config{}, fmt.Errorf("config.Load: validation failed: %w", err)
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
