package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		wantAudited    bool
	}{
		{
			name:           "user has required role",
			claims:         &auth.AuthClaims{Subject: "user-1", Roles: []string{"ADMIN"}},
			requiredRoles:  []string{"ADMIN"},
			wantStatusCode: http.StatusOK,
			wantAudited:    false,
		},
		{
			name:           "user has one of multiple required",
			claims:         &auth.AuthClaims{Subject: "user-1", Roles: []string{"SECRETARY"}},
			requiredRoles:  []string{"ADMIN", "SECRETARY"},
			wantStatusCode: http.StatusOK,
			wantAudited:    false,
		},
		{
			name:           "user lacks role",
			claims:         &auth.AuthClaims{Subject: "user-1", Roles: []string{"VOLUNTEER"}},
			requiredRoles:  []string{"ADMIN"},
			wantStatusCode: http.StatusForbidden,
			wantAudited:    true,
		},
		{
			name:           "no claims in context",
			claims:         nil,
			requiredRoles:  []string{"ADMIN"},
			wantStatusCode: http.StatusForbidden,
			wantAudited:    true,
		},
		{
			name:           "user has empty roles",
			claims:         &auth.AuthClaims{Subject: "user-1", Roles: []string{}},
			requiredRoles:  []string{"ADMIN"},
			wantStatusCode: http.StatusForbidden,
			wantAudited:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auditRepo := new(mockAuditRepo)
			if tt.wantAudited {
				auditRepo.On("Create", mock.Anything, mock.MatchedBy(func(e domain.AuditLog) bool {
					return e.ActionType == "ACCESS_DENIED" && !e.Success
				})).Return(nil).Once()
			}
			auditSvc := service.NewAuditService(auditRepo)

			mw := RequireRole(auditSvc, tt.requiredRoles...)
			handler := mw(okHandler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.claims != nil {
				req = req.WithContext(auth.NewContext(req.Context(), *tt.claims))
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)
			if tt.wantAudited {
				auditRepo.AssertExpectations(t)
			} else {
				auditRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			}
		})
	}
}
