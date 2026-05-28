package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name      string
		authValue string
		wantToken string
		wantErr   bool
	}{
		{
			name:      "valid Bearer token",
			authValue: "Bearer abc123",
			wantToken: "abc123",
		},
		{
			name:    "missing header",
			wantErr: true,
		},
		{
			name:      "no Bearer prefix",
			authValue: "Basic abc123",
			wantErr:   true,
		},
		{
			name:      "bearer lowercase accepted",
			authValue: "bearer abc123",
			wantToken: "abc123",
		},
		{
			name:      "Bearer with long JWT",
			authValue: "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.payload.signature",
			wantToken: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.payload.signature",
		},
		{
			name:      "only keyword no token",
			authValue: "Bearer",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authValue != "" {
				req.Header.Set("Authorization", tt.authValue)
			}

			got, err := extractBearerToken(req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantToken, got)
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, http.StatusUnauthorized, "unauthorized", "invalid token")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "invalid token", body["message"])
}
