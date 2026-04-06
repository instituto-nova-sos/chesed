package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		wantStatus int
		wantBody   map[string]string
	}{
		{
			name:       "GET returns 200 with ok status when no pool",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			wantBody:   map[string]string{"status": "ok", "database": "ok"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHealthHandler(nil)
			req := httptest.NewRequest(tt.method, "/api/v1/health", nil)
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			var body map[string]string
			err := json.NewDecoder(rec.Body).Decode(&body)
			require.NoError(t, err)

			for key, want := range tt.wantBody {
				assert.Equal(t, want, body[key], "body[%q]", key)
			}
		})
	}
}
