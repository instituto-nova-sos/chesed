package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCORS(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name            string
		allowedOrigins  []string
		method          string
		originHeader    string
		wantStatusCode  int
		wantCORSHeaders bool
		wantOrigin      string
	}{
		{
			name:            "preflight OPTIONS with allowed origin",
			allowedOrigins:  []string{"http://localhost:5173"},
			method:          http.MethodOptions,
			originHeader:    "http://localhost:5173",
			wantStatusCode:  http.StatusNoContent,
			wantCORSHeaders: true,
			wantOrigin:      "http://localhost:5173",
		},
		{
			name:            "regular GET with allowed origin",
			allowedOrigins:  []string{"http://localhost:5173"},
			method:          http.MethodGet,
			originHeader:    "http://localhost:5173",
			wantStatusCode:  http.StatusOK,
			wantCORSHeaders: true,
			wantOrigin:      "http://localhost:5173",
		},
		{
			name:            "disallowed origin",
			allowedOrigins:  []string{"http://localhost:5173"},
			method:          http.MethodGet,
			originHeader:    "http://evil.com",
			wantStatusCode:  http.StatusOK,
			wantCORSHeaders: false,
		},
		{
			name:            "no origin header",
			allowedOrigins:  []string{"http://localhost:5173"},
			method:          http.MethodGet,
			originHeader:    "",
			wantStatusCode:  http.StatusOK,
			wantCORSHeaders: false,
		},
		{
			name:            "multiple allowed origins matches second",
			allowedOrigins:  []string{"http://localhost:5173", "http://app.example.com"},
			method:          http.MethodGet,
			originHeader:    "http://app.example.com",
			wantStatusCode:  http.StatusOK,
			wantCORSHeaders: true,
			wantOrigin:      "http://app.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := CORS(tt.allowedOrigins...)
			handler := mw(okHandler)

			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.originHeader != "" {
				req.Header.Set("Origin", tt.originHeader)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)

			if tt.wantCORSHeaders {
				assert.Equal(t, tt.wantOrigin, rec.Header().Get("Access-Control-Allow-Origin"))
				assert.NotEmpty(t, rec.Header().Get("Access-Control-Allow-Methods"))
				assert.NotEmpty(t, rec.Header().Get("Access-Control-Allow-Headers"))
			} else {
				assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}
