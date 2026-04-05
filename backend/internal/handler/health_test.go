package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		wantStatus     int
		wantBody       string
		wantHeaderKey  string
		wantHeaderVal  string
	}{
		{
			name:          "GET returns 200 with ok status",
			method:        http.MethodGet,
			wantStatus:    http.StatusOK,
			wantBody:      `{"status":"ok"}`,
			wantHeaderKey: "Content-Type",
			wantHeaderVal: "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHealthHandler()
			req := httptest.NewRequest(tt.method, "/api/v1/health", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if rec.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.wantBody)
			}
			if got := rec.Header().Get(tt.wantHeaderKey); got != tt.wantHeaderVal {
				t.Errorf("header %q = %q, want %q", tt.wantHeaderKey, got, tt.wantHeaderVal)
			}
		})
	}
}
