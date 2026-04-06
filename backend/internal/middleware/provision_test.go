package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockUserRepo implements service.UserRepository for testing.
type mockUserRepo struct {
	mock.Mock
}

func (m *mockUserRepo) FindByKeycloakSubject(ctx context.Context, subjectID string) (*domain.AppUser, error) {
	args := m.Called(ctx, subjectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AppUser), args.Error(1)
}

func (m *mockUserRepo) Create(ctx context.Context, user domain.AppUser) (*domain.AppUser, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AppUser), args.Error(1)
}

func (m *mockUserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// mockAuditRepo implements service.AuditRepository for testing.
type mockAuditRepo struct {
	mock.Mock
}

func (m *mockAuditRepo) Create(ctx context.Context, entry domain.AuditLog) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		name        string
		remoteAddr  string
		forwardedIP string
		realIP      string
		want        string
	}{
		{
			name:       "RemoteAddr with port",
			remoteAddr: "192.168.1.1:8080",
			want:       "192.168.1.1",
		},
		{
			name:        "X-Forwarded-For takes precedence",
			remoteAddr:  "192.168.1.1:8080",
			forwardedIP: "10.0.0.1",
			want:        "10.0.0.1",
		},
		{
			name:       "X-Real-IP used when no XFF",
			remoteAddr: "192.168.1.1:8080",
			realIP:     "10.0.0.2",
			want:       "10.0.0.2",
		},
		{
			name:       "no port in RemoteAddr",
			remoteAddr: "192.168.1.1",
			want:       "192.168.1.1",
		},
		{
			name:       "IPv6 with port",
			remoteAddr: "[::1]:8080",
			want:       "::1",
		},
		{
			name:        "XFF with multiple IPs takes first",
			remoteAddr:  "192.168.1.1:8080",
			forwardedIP: "10.0.0.1, 10.0.0.2, 10.0.0.3",
			want:        "10.0.0.1",
		},
		{
			name:        "invalid XFF returns empty",
			remoteAddr:  "192.168.1.1:8080",
			forwardedIP: "not-an-ip",
			want:        "",
		},
		{
			name:       "garbage RemoteAddr returns empty",
			remoteAddr: "garbage",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.forwardedIP != "" {
				req.Header.Set("X-Forwarded-For", tt.forwardedIP)
			}
			if tt.realIP != "" {
				req.Header.Set("X-Real-IP", tt.realIP)
			}

			got := extractIP(req)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAutoProvision(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("existing user passes through", func(t *testing.T) {
		userRepo := new(mockUserRepo)
		auditRepo := new(mockAuditRepo)
		auditSvc := service.NewAuditService(auditRepo)
		userSvc := service.NewUserService(userRepo, auditSvc)

		claims := auth.AuthClaims{
			Subject:  uuid.New().String(),
			Email:    "test@chesed.test",
			Roles:    []string{"VOLUNTEER"},
			CampusID: uuid.New(),
		}

		existing := &domain.AppUser{ID: uuid.New(), Email: claims.Email}
		userRepo.On("FindByKeycloakSubject", mock.Anything, claims.Subject).Return(existing, nil)
		userRepo.On("UpdateLastLogin", mock.Anything, existing.ID).Return(nil)
		auditRepo.On("Create", mock.Anything, mock.Anything).Return(nil) // LOGIN audit

		mw := AutoProvision(userSvc)
		handler := mw(okHandler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = req.WithContext(auth.NewContext(req.Context(), claims))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		userRepo.AssertExpectations(t)
	})

	t.Run("new user provisioned", func(t *testing.T) {
		userRepo := new(mockUserRepo)
		auditRepo := new(mockAuditRepo)
		auditSvc := service.NewAuditService(auditRepo)
		userSvc := service.NewUserService(userRepo, auditSvc)

		claims := auth.AuthClaims{
			Subject:  uuid.New().String(),
			Email:    "new@chesed.test",
			Roles:    []string{"SECRETARY"},
			CampusID: uuid.New(),
		}

		userRepo.On("FindByKeycloakSubject", mock.Anything, claims.Subject).Return(nil, domain.ErrNotFound)
		userRepo.On("Create", mock.Anything, mock.Anything).Return(&domain.AppUser{
			ID:    uuid.New(),
			Email: claims.Email,
		}, nil)
		auditRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

		mw := AutoProvision(userSvc)
		handler := mw(okHandler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = req.WithContext(auth.NewContext(req.Context(), claims))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		userRepo.AssertExpectations(t)
	})

	t.Run("missing campus_id returns 403", func(t *testing.T) {
		userRepo := new(mockUserRepo)
		auditRepo := new(mockAuditRepo)
		auditSvc := service.NewAuditService(auditRepo)
		userSvc := service.NewUserService(userRepo, auditSvc)

		claims := auth.AuthClaims{
			Subject: uuid.New().String(),
			Email:   "test@chesed.test",
			Roles:   []string{"VOLUNTEER"},
			// CampusID intentionally left as uuid.Nil
		}

		mw := AutoProvision(userSvc)
		handler := mw(okHandler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = req.WithContext(auth.NewContext(req.Context(), claims))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("missing claims returns 401", func(t *testing.T) {
		userRepo := new(mockUserRepo)
		auditRepo := new(mockAuditRepo)
		auditSvc := service.NewAuditService(auditRepo)
		userSvc := service.NewUserService(userRepo, auditSvc)

		mw := AutoProvision(userSvc)
		handler := mw(okHandler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		userRepo := new(mockUserRepo)
		auditRepo := new(mockAuditRepo)
		auditSvc := service.NewAuditService(auditRepo)
		userSvc := service.NewUserService(userRepo, auditSvc)

		claims := auth.AuthClaims{
			Subject:  uuid.New().String(),
			Email:    "test@chesed.test",
			Roles:    []string{"VOLUNTEER"},
			CampusID: uuid.New(),
		}

		userRepo.On("FindByKeycloakSubject", mock.Anything, claims.Subject).Return(nil, assert.AnError)

		mw := AutoProvision(userSvc)
		handler := mw(okHandler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = req.WithContext(auth.NewContext(req.Context(), claims))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}
