package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		check   func(t *testing.T, cfg Config)
	}{
		{
			name: "valid config with all required fields",
			envVars: map[string]string{
				"SERVER_PORT":        "9090",
				"DATABASE_URL":       "postgres://user:pass@localhost:5432/db",
				"KEYCLOAK_URL":       "http://localhost:8180",
				"KEYCLOAK_REALM":     "chesed",
				"KEYCLOAK_CLIENT_ID": "chesed-pwa",
				"LOG_LEVEL":          "debug",
			},
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				assert.Equal(t, "9090", cfg.ServerPort)
				assert.Equal(t, "postgres://user:pass@localhost:5432/db", cfg.DatabaseURL)
				assert.Equal(t, "debug", cfg.LogLevel)
			},
		},
		{
			name: "defaults applied for optional fields",
			envVars: map[string]string{
				"DATABASE_URL":       "postgres://user:pass@localhost:5432/db",
				"KEYCLOAK_URL":       "http://localhost:8180",
				"KEYCLOAK_REALM":     "chesed",
				"KEYCLOAK_CLIENT_ID": "chesed-pwa",
			},
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				assert.Equal(t, "8080", cfg.ServerPort)
				assert.Equal(t, "info", cfg.LogLevel)
				assert.False(t, cfg.OIDCSkipIssuerCheck)
			},
		},
		{
			name: "OIDC_SKIP_ISSUER_CHECK enabled",
			envVars: map[string]string{
				"DATABASE_URL":           "postgres://user:pass@localhost:5432/db",
				"KEYCLOAK_URL":           "http://localhost:8180",
				"KEYCLOAK_REALM":         "chesed",
				"KEYCLOAK_CLIENT_ID":     "chesed-pwa",
				"OIDC_SKIP_ISSUER_CHECK": "true",
			},
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				assert.True(t, cfg.OIDCSkipIssuerCheck)
			},
		},
		{
			name: "missing DATABASE_URL returns error",
			envVars: map[string]string{
				"KEYCLOAK_URL":       "http://localhost:8180",
				"KEYCLOAK_REALM":     "chesed",
				"KEYCLOAK_CLIENT_ID": "chesed-pwa",
			},
			wantErr: true,
		},
		{
			name: "missing KEYCLOAK_URL returns error",
			envVars: map[string]string{
				"DATABASE_URL":       "postgres://user:pass@localhost:5432/db",
				"KEYCLOAK_REALM":     "chesed",
				"KEYCLOAK_CLIENT_ID": "chesed-pwa",
			},
			wantErr: true,
		},
		{
			name: "missing KEYCLOAK_REALM returns error",
			envVars: map[string]string{
				"DATABASE_URL":       "postgres://user:pass@localhost:5432/db",
				"KEYCLOAK_URL":       "http://localhost:8180",
				"KEYCLOAK_CLIENT_ID": "chesed-pwa",
			},
			wantErr: true,
		},
		{
			name: "missing KEYCLOAK_CLIENT_ID returns error",
			envVars: map[string]string{
				"DATABASE_URL":   "postgres://user:pass@localhost:5432/db",
				"KEYCLOAK_URL":   "http://localhost:8180",
				"KEYCLOAK_REALM": "chesed",
			},
			wantErr: true,
		},
		{
			name:    "all required fields missing returns error",
			envVars: map[string]string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearBaseConfigEnv(t)

			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			cfg, err := Load()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

// clearBaseConfigEnv resets the non-S3 configuration variables and sets a
// valid S3 credential baseline — the credentials are required with no
// source-code default, so non-S3 cases must not fail on them.
func clearBaseConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"SERVER_PORT", "DATABASE_URL", "KEYCLOAK_URL",
		"KEYCLOAK_REALM", "KEYCLOAK_CLIENT_ID", "LOG_LEVEL",
		"OIDC_SKIP_ISSUER_CHECK",
	} {
		t.Setenv(key, "")
	}
	t.Setenv("S3_ACCESS_KEY", "test-access")
	t.Setenv("S3_SECRET_KEY", "test-secret")
}

func TestLoadS3(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		check   func(t *testing.T, cfg Config)
	}{
		{
			// Endpoint and bucket are non-secret and may default; the
			// credentials must come from the environment (docs/19: no
			// secrets in source, not even development ones).
			name: "non-secret defaults applied, credentials from env",
			envVars: map[string]string{
				"S3_ACCESS_KEY": "env-access",
				"S3_SECRET_KEY": "env-secret",
			},
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				assert.Equal(t, "localhost:9000", cfg.S3Endpoint)
				assert.Equal(t, "chesed-docs", cfg.S3Bucket)
				assert.Equal(t, "env-access", cfg.S3AccessKey)
				assert.Equal(t, "env-secret", cfg.S3SecretKey)
				assert.False(t, cfg.S3UseSSL)
			},
		},
		{
			name:    "missing credentials fail fast",
			envVars: map[string]string{},
			wantErr: true,
		},
		{
			name:    "missing secret key fails fast",
			envVars: map[string]string{"S3_ACCESS_KEY": "env-access"},
			wantErr: true,
		},
		{
			name: "overridden from environment",
			envVars: map[string]string{
				"S3_ENDPOINT":   "s3.example.com:443",
				"S3_BUCKET":     "prod-docs",
				"S3_ACCESS_KEY": "prod-access",
				"S3_SECRET_KEY": "prod-secret",
				"S3_USE_SSL":    "true",
			},
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				assert.Equal(t, "s3.example.com:443", cfg.S3Endpoint)
				assert.Equal(t, "prod-docs", cfg.S3Bucket)
				assert.Equal(t, "prod-access", cfg.S3AccessKey)
				assert.Equal(t, "prod-secret", cfg.S3SecretKey)
				assert.True(t, cfg.S3UseSSL)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{
				"S3_ENDPOINT", "S3_BUCKET", "S3_ACCESS_KEY",
				"S3_SECRET_KEY", "S3_USE_SSL",
			} {
				t.Setenv(key, "")
			}
			t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
			t.Setenv("KEYCLOAK_URL", "http://localhost:8180")
			t.Setenv("KEYCLOAK_REALM", "chesed")
			t.Setenv("KEYCLOAK_CLIENT_ID", "chesed-pwa")

			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			cfg, err := Load()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.check(t, cfg)
		})
	}
}
