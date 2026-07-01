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

func (m *mockUserRepo) LinkPersonID(ctx context.Context, userID uuid.UUID, personID uuid.UUID) error {
	args := m.Called(ctx, userID, personID)
	return args.Error(0)
}

func (m *mockUserRepo) LinkPersonAndCampus(ctx context.Context, userID uuid.UUID, personID uuid.UUID, campusID uuid.UUID) error {
	args := m.Called(ctx, userID, personID, campusID)
	return args.Error(0)
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

// provisionOKHandler returns 200 when the campus was enriched into the context,
// 418 otherwise — so tests can distinguish enrichment from a bare pass-through.
func provisionOKHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if auth.ClaimsFromContext(r.Context()).CampusID != uuid.Nil {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusTeapot)
		}
	}
}

// newProvisionTest wires the AutoProvision middleware over fresh mocks.
func newProvisionTest() (*mockUserRepo, *mockAuditRepo, func(http.Handler) http.Handler) {
	userRepo := new(mockUserRepo)
	auditRepo := new(mockAuditRepo)
	userSvc := service.NewUserService(userRepo, service.NewAuditService(auditRepo))
	return userRepo, auditRepo, AutoProvision(userSvc)
}

// runProvision executes the middleware for the given claims (nil = no claims).
func runProvision(mw func(http.Handler) http.Handler, claims *auth.AuthClaims) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	if claims != nil {
		req = req.WithContext(auth.NewContext(req.Context(), *claims))
	}
	rec := httptest.NewRecorder()
	mw(provisionOKHandler()).ServeHTTP(rec, req)
	return rec
}

func volunteerClaims(email string) auth.AuthClaims {
	return auth.AuthClaims{Subject: uuid.New().String(), Email: email, Roles: []string{"VOLUNTEER"}}
}

func TestAutoProvision_ExistingUserWithCampus(t *testing.T) {
	userRepo, auditRepo, mw := newProvisionTest()
	campusID := uuid.New()
	claims := volunteerClaims("test@chesed.test")
	existing := &domain.AppUser{ID: uuid.New(), Email: claims.Email, CampusID: &campusID}
	userRepo.On("FindByKeycloakSubject", mock.Anything, claims.Subject).Return(existing, nil)
	userRepo.On("UpdateLastLogin", mock.Anything, existing.ID).Return(nil)
	auditRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	rec := runProvision(mw, &claims)
	assert.Equal(t, http.StatusOK, rec.Code)
	userRepo.AssertExpectations(t)
}

func TestAutoProvision_NewUserProvisioned(t *testing.T) {
	userRepo, auditRepo, mw := newProvisionTest()
	campusID := uuid.New()
	claims := auth.AuthClaims{Subject: uuid.New().String(), Email: "new@chesed.test", Roles: []string{"SECRETARY"}}
	userRepo.On("FindByKeycloakSubject", mock.Anything, claims.Subject).Return(nil, domain.ErrNotFound)
	userRepo.On("Create", mock.Anything, mock.Anything).Return(&domain.AppUser{ID: uuid.New(), Email: claims.Email, CampusID: &campusID}, nil)
	auditRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	rec := runProvision(mw, &claims)
	assert.Equal(t, http.StatusOK, rec.Code)
	userRepo.AssertExpectations(t)
}

func TestAutoProvision_NilCampusForbidden(t *testing.T) {
	userRepo, auditRepo, mw := newProvisionTest()
	claims := volunteerClaims("test@chesed.test")
	existing := &domain.AppUser{ID: uuid.New(), Email: claims.Email, CampusID: nil}
	userRepo.On("FindByKeycloakSubject", mock.Anything, claims.Subject).Return(existing, nil)
	userRepo.On("UpdateLastLogin", mock.Anything, existing.ID).Return(nil)
	auditRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	rec := runProvision(mw, &claims)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAutoProvision_MissingClaimsUnauthorized(t *testing.T) {
	_, _, mw := newProvisionTest()
	rec := runProvision(mw, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAutoProvision_ServiceErrorInternal(t *testing.T) {
	userRepo, _, mw := newProvisionTest()
	claims := volunteerClaims("test@chesed.test")
	userRepo.On("FindByKeycloakSubject", mock.Anything, claims.Subject).Return(nil, assert.AnError)

	rec := runProvision(mw, &claims)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
