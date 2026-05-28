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
				"DATABASE_URL":            "postgres://user:pass@localhost:5432/db",
				"KEYCLOAK_URL":            "http://localhost:8180",
				"KEYCLOAK_REALM":          "chesed",
				"KEYCLOAK_CLIENT_ID":      "chesed-pwa",
				"OIDC_SKIP_ISSUER_CHECK":  "true",
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
			for _, key := range []string{
				"SERVER_PORT", "DATABASE_URL", "KEYCLOAK_URL",
				"KEYCLOAK_REALM", "KEYCLOAK_CLIENT_ID", "LOG_LEVEL",
				"OIDC_SKIP_ISSUER_CHECK",
			} {
				t.Setenv(key, "")
			}

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
