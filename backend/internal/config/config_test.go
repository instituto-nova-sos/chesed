package config

import (
	"testing"
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
			wantErr: false,
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.ServerPort != "9090" {
					t.Errorf("ServerPort = %q, want %q", cfg.ServerPort, "9090")
				}
				if cfg.DatabaseURL != "postgres://user:pass@localhost:5432/db" {
					t.Errorf("DatabaseURL = %q, want postgres URL", cfg.DatabaseURL)
				}
				if cfg.LogLevel != "debug" {
					t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
				}
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
			wantErr: false,
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.ServerPort != "8080" {
					t.Errorf("ServerPort = %q, want default %q", cfg.ServerPort, "8080")
				}
				if cfg.LogLevel != "info" {
					t.Errorf("LogLevel = %q, want default %q", cfg.LogLevel, "info")
				}
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
			// Clear all relevant env vars
			for _, key := range []string{
				"SERVER_PORT", "DATABASE_URL", "KEYCLOAK_URL",
				"KEYCLOAK_REALM", "KEYCLOAK_CLIENT_ID", "LOG_LEVEL",
			} {
				t.Setenv(key, "")
			}

			// Set test env vars
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			cfg, err := Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.check != nil && err == nil {
				tt.check(t, cfg)
			}
		})
	}
}
