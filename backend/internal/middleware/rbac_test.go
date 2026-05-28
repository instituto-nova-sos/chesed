package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/stretchr/testify/assert"
)

func TestRequireRole(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name           string
		claims         *auth.AuthClaims
		requiredRoles  []string
		wantStatusCode int
	}{
		{
			name:           "user has required role",
			claims:         &auth.AuthClaims{Subject: "user-1", Roles: []string{"ADMIN"}},
			requiredRoles:  []string{"ADMIN"},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "user has one of multiple required",
			claims:         &auth.AuthClaims{Subject: "user-1", Roles: []string{"SECRETARY"}},
			requiredRoles:  []string{"ADMIN", "SECRETARY"},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "user lacks role",
			claims:         &auth.AuthClaims{Subject: "user-1", Roles: []string{"VOLUNTEER"}},
			requiredRoles:  []string{"ADMIN"},
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "no claims in context",
			claims:         nil,
			requiredRoles:  []string{"ADMIN"},
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "user has empty roles",
			claims:         &auth.AuthClaims{Subject: "user-1", Roles: []string{}},
			requiredRoles:  []string{"ADMIN"},
			wantStatusCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := RequireRole(tt.requiredRoles...)
			handler := mw(okHandler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.claims != nil {
				req = req.WithContext(auth.NewContext(req.Context(), *tt.claims))
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)
		})
	}
}
