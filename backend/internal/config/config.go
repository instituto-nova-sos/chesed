package config

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	ServerPort       string `validate:"required"`
	DatabaseURL      string `validate:"required"`
	KeycloakURL      string `validate:"required,url"`
	KeycloakRealm    string `validate:"required"`
	KeycloakClientID string `validate:"required"`
	LogLevel         string
}

// Load reads configuration from environment variables and validates required fields.
func Load() (Config, error) {
	cfg := Config{
		ServerPort:       getEnvOrDefault("SERVER_PORT", "8080"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		KeycloakURL:      os.Getenv("KEYCLOAK_URL"),
		KeycloakRealm:    os.Getenv("KEYCLOAK_REALM"),
		KeycloakClientID: os.Getenv("KEYCLOAK_CLIENT_ID"),
		LogLevel:         getEnvOrDefault("LOG_LEVEL", "info"),
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
