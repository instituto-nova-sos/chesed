package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicRateLimit(t *testing.T) {
	const limit = 3
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := PublicRateLimit(limit)(okHandler)

	send := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/public/campaigns", nil)
		req.RemoteAddr = "203.0.113.7:54321"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	for i := 0; i < limit; i++ {
		rec := send()
		require.Equal(t, http.StatusOK, rec.Code, "request %d within limit must pass", i+1)
	}

	rec := send()
	assert.Equal(t, http.StatusTooManyRequests, rec.Code, "the (limit+1)th request must be throttled")

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "rate_limited", body["error"])
	assert.NotEmpty(t, body["message"])
}

func TestPublicRateLimit_PerIPIsolation(t *testing.T) {
	const limit = 2
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := PublicRateLimit(limit)(okHandler)

	send := func(ip string) int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip + ":1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	// Exhaust the first IP's budget.
	for i := 0; i < limit; i++ {
		require.Equal(t, http.StatusOK, send("198.51.100.1"))
	}
	assert.Equal(t, http.StatusTooManyRequests, send("198.51.100.1"))

	// A different IP still has a full budget.
	assert.Equal(t, http.StatusOK, send("198.51.100.2"))
}
