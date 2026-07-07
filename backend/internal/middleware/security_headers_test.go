package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const hstsHeader = "Strict-Transport-Security"

func assertBaseSecurityHeaders(t *testing.T, h http.Header) {
	t.Helper()
	assert.Equal(t, "nosniff", h.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", h.Get("X-Frame-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", h.Get("Referrer-Policy"))
	assert.Equal(t, "camera=(), microphone=(), geolocation=()", h.Get("Permissions-Policy"))
	assert.Equal(t, "default-src 'none'; frame-ancestors 'none'", h.Get("Content-Security-Policy"))
	assert.Equal(t, "no-store", h.Get("Cache-Control"))
}

func TestSecurityHeaders(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := SecurityHeaders(okHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assertBaseSecurityHeaders(t, rec.Header())
	assert.Empty(t, rec.Header().Get(hstsHeader), "default SecurityHeaders must not emit HSTS")
}

func TestSecurityHeadersWith(t *testing.T) {
	tests := []struct {
		name        string
		hstsEnabled bool
		wantHSTS    string
	}{
		{name: "hsts disabled", hstsEnabled: false, wantHSTS: ""},
		{
			name:        "hsts enabled",
			hstsEnabled: true,
			wantHSTS:    "max-age=63072000; includeSubDomains; preload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := SecurityHeadersWith(tt.hstsEnabled)(okHandler)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assertBaseSecurityHeaders(t, rec.Header())
			assert.Equal(t, tt.wantHSTS, rec.Header().Get(hstsHeader))
		})
	}
}
