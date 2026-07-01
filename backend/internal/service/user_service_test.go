package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockUserRepository is a testify mock for UserRepository.
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) FindByKeycloakSubject(ctx context.Context, subjectID string) (*domain.AppUser, error) {
	args := m.Called(ctx, subjectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AppUser), args.Error(1)
}

func (m *MockUserRepository) Create(ctx context.Context, user domain.AppUser) (*domain.AppUser, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AppUser), args.Error(1)
}

func (m *MockUserRepository) LinkPersonID(ctx context.Context, userID uuid.UUID, personID uuid.UUID) error {
	args := m.Called(ctx, userID, personID)
	return args.Error(0)
}

func (m *MockUserRepository) LinkPersonAndCampus(ctx context.Context, userID uuid.UUID, personID uuid.UUID, campusID uuid.UUID) error {
	args := m.Called(ctx, userID, personID, campusID)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func newTestClaims() auth.AuthClaims {
	return auth.AuthClaims{
		Subject:  uuid.New().String(),
		Email:    "test@chesed.test",
		Roles:    []string{"VOLUNTEER"},
		CampusID: uuid.New(),
	}
}

// newUserServiceTest wires UserService over fresh mocks with a base context.
func newUserServiceTest() (*MockUserRepository, *MockAuditRepository, *UserService, auth.AuthClaims, context.Context) {
	userRepo := new(MockUserRepository)
	auditRepo := new(MockAuditRepository)
	svc := NewUserService(userRepo, NewAuditService(auditRepo))
	claims := newTestClaims()
	return userRepo, auditRepo, svc, claims, auth.NewContext(context.Background(), claims)
}

func TestUserService_EnsureUser_ExistingLogsLogin(t *testing.T) {
	userRepo, auditRepo, svc, claims, ctx := newUserServiceTest()
	existing := &domain.AppUser{
		ID: uuid.New(), Email: claims.Email, KeycloakSubjectID: claims.Subject,
		AccessProfile: "VOLUNTEER", CampusID: &claims.CampusID, IsActive: true,
	}
	userRepo.On("FindByKeycloakSubject", ctx, claims.Subject).Return(existing, nil)
	userRepo.On("UpdateLastLogin", ctx, existing.ID).Return(nil)
	auditRepo.On("Create", ctx, mock.MatchedBy(func(entry domain.AuditLog) bool {
		return entry.ActionType == "LOGIN" && entry.EntityType == "app_user"
	})).Return(nil)

	got, err := svc.EnsureUser(ctx, claims, "1.2.3.4", "test-agent")
	require.NoError(t, err)
	assert.Equal(t, existing.ID, got.ID)
	assert.Equal(t, existing.Email, got.Email)
	userRepo.AssertExpectations(t)
	auditRepo.AssertExpectations(t)
	userRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestUserService_EnsureUser_NewProvisioned(t *testing.T) {
	userRepo, auditRepo, svc, claims, _ := newUserServiceTest()
	claims.Roles = []string{"COORDINATOR"}
	ctx := auth.NewContext(context.Background(), claims)

	userRepo.On("FindByKeycloakSubject", ctx, claims.Subject).Return(nil, domain.ErrNotFound)
	userRepo.On("Create", ctx, mock.MatchedBy(func(user domain.AppUser) bool {
		return user.Email == claims.Email &&
			user.KeycloakSubjectID == claims.Subject &&
			user.AccessProfile == "COORDINATOR" &&
			user.CampusID != nil && *user.CampusID == claims.CampusID &&
			user.IsActive && user.ID != uuid.Nil
	})).Return(&domain.AppUser{
		ID: uuid.New(), Email: claims.Email, KeycloakSubjectID: claims.Subject,
		AccessProfile: "COORDINATOR", CampusID: &claims.CampusID, IsActive: true,
	}, nil)
	auditRepo.On("Create", ctx, mock.Anything).Return(nil)

	got, err := svc.EnsureUser(ctx, claims, "1.2.3.4", "test-agent")
	require.NoError(t, err)
	assert.Equal(t, claims.Email, got.Email)
	assert.Equal(t, "COORDINATOR", got.AccessProfile)
	userRepo.AssertExpectations(t)
	auditRepo.AssertExpectations(t)
}

func TestUserService_EnsureUser_AuditErrorStillReturnsUser(t *testing.T) {
	userRepo, auditRepo, svc, claims, ctx := newUserServiceTest()
	createdUser := &domain.AppUser{ID: uuid.New(), Email: claims.Email}
	userRepo.On("FindByKeycloakSubject", ctx, claims.Subject).Return(nil, domain.ErrNotFound)
	userRepo.On("Create", ctx, mock.Anything).Return(createdUser, nil)
	auditRepo.On("Create", ctx, mock.Anything).Return(errors.New("audit db down"))

	got, err := svc.EnsureUser(ctx, claims, "1.2.3.4", "test-agent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audit")
	assert.NotNil(t, got, "user should still be returned even if audit fails")
	assert.Equal(t, createdUser.ID, got.ID)
}

func TestUserService_EnsureUser_FindErrorPropagated(t *testing.T) {
	userRepo, _, svc, claims, ctx := newUserServiceTest()
	userRepo.On("FindByKeycloakSubject", ctx, claims.Subject).Return(nil, errors.New("db timeout"))

	got, err := svc.EnsureUser(ctx, claims, "1.2.3.4", "test-agent")
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "db timeout")
}

func TestUserService_EnsureUser_UpdateLastLoginErrorPropagated(t *testing.T) {
	userRepo, _, svc, claims, ctx := newUserServiceTest()
	existing := &domain.AppUser{ID: uuid.New()}
	userRepo.On("FindByKeycloakSubject", ctx, claims.Subject).Return(existing, nil)
	userRepo.On("UpdateLastLogin", ctx, existing.ID).Return(errors.New("update failed"))

	got, err := svc.EnsureUser(ctx, claims, "1.2.3.4", "test-agent")
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "update failed")
}

func TestUserService_EnsureUser_CreateErrorPropagated(t *testing.T) {
	userRepo, _, svc, claims, ctx := newUserServiceTest()
	userRepo.On("FindByKeycloakSubject", ctx, claims.Subject).Return(nil, domain.ErrNotFound)
	userRepo.On("Create", ctx, mock.Anything).Return(nil, errors.New("unique constraint"))

	got, err := svc.EnsureUser(ctx, claims, "1.2.3.4", "test-agent")
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "unique constraint")
}

func TestResolveAccessProfile(t *testing.T) {
	tests := []struct {
		name  string
		roles []string
		want  string
	}{
		{
			name:  "single ADMIN role",
			roles: []string{"ADMIN"},
			want:  "ADMIN",
		},
		{
			name:  "single VOLUNTEER role",
			roles: []string{"VOLUNTEER"},
			want:  "VOLUNTEER",
		},
		{
			name:  "multiple roles picks highest",
			roles: []string{"VOLUNTEER", "COORDINATOR"},
			want:  "COORDINATOR",
		},
		{
			name:  "all roles picks ADMIN",
			roles: []string{"VOLUNTEER", "SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN"},
			want:  "ADMIN",
		},
		{
			name:  "unknown role defaults to VOLUNTEER",
			roles: []string{"UNKNOWN_ROLE"},
			want:  "VOLUNTEER",
		},
		{
			name:  "empty roles defaults to VOLUNTEER",
			roles: []string{},
			want:  "VOLUNTEER",
		},
		{
			name:  "nil roles defaults to VOLUNTEER",
			roles: nil,
			want:  "VOLUNTEER",
		},
		{
			name:  "mixed known and unknown picks highest known",
			roles: []string{"offline_access", "SECRETARY", "uma_authorization"},
			want:  "SECRETARY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveAccessProfile(tt.roles)
			assert.Equal(t, tt.want, got)
		})
	}
}
