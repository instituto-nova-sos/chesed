package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockAgreementRepo implements VolunteerAgreementRepository for onboarding tests.
type mockAgreementRepo struct {
	mock.Mock
}

func (m *mockAgreementRepo) Create(ctx context.Context, a domain.VolunteerAgreement) (*domain.VolunteerAgreement, error) {
	args := m.Called(ctx, a)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VolunteerAgreement), args.Error(1)
}

func (m *mockAgreementRepo) FindByPersonID(ctx context.Context, personID uuid.UUID) ([]domain.VolunteerAgreement, error) {
	args := m.Called(ctx, personID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.VolunteerAgreement), args.Error(1)
}

func (m *mockAgreementRepo) FindByPersonRoleID(ctx context.Context, personRoleID uuid.UUID) (*domain.VolunteerAgreement, error) {
	args := m.Called(ctx, personRoleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VolunteerAgreement), args.Error(1)
}

func (m *mockAgreementRepo) FindPendingByPersonID(ctx context.Context, personID uuid.UUID) (*domain.VolunteerAgreement, error) {
	args := m.Called(ctx, personID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VolunteerAgreement), args.Error(1)
}

func (m *mockAgreementRepo) HasAcceptedAgreement(ctx context.Context, personID uuid.UUID) (bool, error) {
	args := m.Called(ctx, personID)
	return args.Bool(0), args.Error(1)
}

func (m *mockAgreementRepo) AcceptDigital(ctx context.Context, id, userID uuid.UUID, ip, ua string) (*domain.VolunteerAgreement, error) {
	args := m.Called(ctx, id, userID, ip, ua)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VolunteerAgreement), args.Error(1)
}

func (m *mockAgreementRepo) Reject(ctx context.Context, id uuid.UUID, reason *string) (*domain.VolunteerAgreement, error) {
	args := m.Called(ctx, id, reason)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VolunteerAgreement), args.Error(1)
}

func (m *mockAgreementRepo) AcceptManualUpload(ctx context.Context, id uuid.UUID, docPath string, uploadedBy uuid.UUID) (*domain.VolunteerAgreement, error) {
	args := m.Called(ctx, id, docPath, uploadedBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VolunteerAgreement), args.Error(1)
}

func newOnboardingTestDeps() (*MockUserRepository, *MockPersonRepository, *MockPersonRoleRepository, *mockAgreementRepo, *OnboardingService) {
	userRepo := new(MockUserRepository)
	personRepo := new(MockPersonRepository)
	roleRepo := new(MockPersonRoleRepository)
	agreementRepo := new(mockAgreementRepo)
	auditRepo := new(MockAuditRepository)
	auditSvc := NewAuditService(auditRepo)

	auditRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()

	svc := NewOnboardingService(userRepo, personRepo, roleRepo, agreementRepo, auditSvc)
	return userRepo, personRepo, roleRepo, agreementRepo, svc
}

func TestOnboardingService_GetStatus(t *testing.T) {
	campusID := uuid.New()
	personID := uuid.New()
	userID := uuid.New()

	t.Run("existing user with campus and person — accepted agreement", func(t *testing.T) {
		userRepo, _, roleRepo, agreementRepo, svc := newOnboardingTestDeps()

		appUser := &domain.AppUser{ID: userID, PersonID: &personID, CampusID: &campusID}
		userRepo.On("FindByKeycloakSubject", mock.Anything, "sub-1").Return(appUser, nil)
		roleRepo.On("FindByPersonID", mock.Anything, personID).Return([]domain.PersonRole{
			{RoleType: domain.RoleVolunteer, IsActive: true},
		}, nil)
		agreementRepo.On("HasAcceptedAgreement", mock.Anything, personID).Return(true, nil)

		claims := auth.AuthClaims{Subject: "sub-1", Email: "test@test.com", Roles: []string{"VOLUNTEER"}}
		status, err := svc.GetStatus(context.Background(), claims)

		require.NoError(t, err)
		assert.Equal(t, &personID, status.PersonID)
		assert.Equal(t, &campusID, status.CampusID)
		assert.False(t, status.NeedsProfileCompletion)
		assert.False(t, status.NeedsCampusAssignment)
		assert.False(t, status.NeedsAgreement)
	})

	t.Run("existing user with campus and person — pending agreement", func(t *testing.T) {
		userRepo, _, roleRepo, agreementRepo, svc := newOnboardingTestDeps()

		appUser := &domain.AppUser{ID: userID, PersonID: &personID, CampusID: &campusID}
		userRepo.On("FindByKeycloakSubject", mock.Anything, "sub-2").Return(appUser, nil)
		roleRepo.On("FindByPersonID", mock.Anything, personID).Return([]domain.PersonRole{
			{RoleType: domain.RoleVolunteer, IsActive: true},
		}, nil)
		agreementRepo.On("HasAcceptedAgreement", mock.Anything, personID).Return(false, nil)

		claims := auth.AuthClaims{Subject: "sub-2", Email: "test@test.com", Roles: []string{"VOLUNTEER"}}
		status, err := svc.GetStatus(context.Background(), claims)

		require.NoError(t, err)
		assert.False(t, status.NeedsProfileCompletion)
		assert.True(t, status.NeedsAgreement)
	})

	t.Run("existing user with campus but no person — needs profile completion", func(t *testing.T) {
		userRepo, personRepo, _, _, svc := newOnboardingTestDeps()

		appUser := &domain.AppUser{ID: userID, PersonID: nil, CampusID: &campusID}
		userRepo.On("FindByKeycloakSubject", mock.Anything, "sub-3").Return(appUser, nil)
		personRepo.On("FindByEmail", mock.Anything, "new@test.com", campusID).Return(nil, domain.ErrNotFound)

		claims := auth.AuthClaims{Subject: "sub-3", Email: "new@test.com", Roles: []string{"VOLUNTEER"}}
		status, err := svc.GetStatus(context.Background(), claims)

		require.NoError(t, err)
		assert.Nil(t, status.PersonID)
		assert.True(t, status.NeedsProfileCompletion)
		assert.False(t, status.NeedsCampusAssignment)
	})

	t.Run("existing user with nil campus — auto-links person and campus", func(t *testing.T) {
		userRepo, personRepo, roleRepo, agreementRepo, svc := newOnboardingTestDeps()

		appUser := &domain.AppUser{ID: userID, PersonID: nil, CampusID: nil}
		existingPerson := domain.Person{ID: personID, CampusID: campusID}

		userRepo.On("FindByKeycloakSubject", mock.Anything, "sub-4").Return(appUser, nil)
		personRepo.On("FindByEmailGlobal", mock.Anything, "existing@test.com").Return([]domain.Person{existingPerson}, nil)
		userRepo.On("LinkPersonAndCampus", mock.Anything, userID, personID, campusID).Return(nil)
		roleRepo.On("FindByPersonID", mock.Anything, personID).Return([]domain.PersonRole{
			{RoleType: domain.RoleVolunteer, IsActive: true},
		}, nil)
		agreementRepo.On("HasAcceptedAgreement", mock.Anything, personID).Return(true, nil)

		claims := auth.AuthClaims{Subject: "sub-4", Email: "existing@test.com", Roles: []string{"VOLUNTEER"}}
		status, err := svc.GetStatus(context.Background(), claims)

		require.NoError(t, err)
		assert.Equal(t, &personID, status.PersonID)
		assert.Equal(t, &campusID, status.CampusID)
		assert.False(t, status.NeedsProfileCompletion)
		assert.False(t, status.NeedsCampusAssignment)
		userRepo.AssertCalled(t, "LinkPersonAndCampus", mock.Anything, userID, personID, campusID)
	})

	t.Run("no app_user and no person — needs campus and profile", func(t *testing.T) {
		userRepo, personRepo, _, _, svc := newOnboardingTestDeps()

		userRepo.On("FindByKeycloakSubject", mock.Anything, "sub-5").Return(nil, domain.ErrNotFound)
		personRepo.On("FindByEmailGlobal", mock.Anything, "brand-new@test.com").Return([]domain.Person{}, nil)

		claims := auth.AuthClaims{Subject: "sub-5", Email: "brand-new@test.com", Roles: []string{"VOLUNTEER"}}
		status, err := svc.GetStatus(context.Background(), claims)

		require.NoError(t, err)
		assert.Nil(t, status.PersonID)
		assert.Nil(t, status.CampusID)
		assert.True(t, status.NeedsProfileCompletion)
		assert.True(t, status.NeedsCampusAssignment)
	})

	t.Run("no app_user but person found by email — derives campus", func(t *testing.T) {
		userRepo, personRepo, roleRepo, _, svc := newOnboardingTestDeps()

		person := domain.Person{ID: personID, CampusID: campusID}

		userRepo.On("FindByKeycloakSubject", mock.Anything, "sub-6").Return(nil, domain.ErrNotFound)
		personRepo.On("FindByEmailGlobal", mock.Anything, "precreated@test.com").Return([]domain.Person{person}, nil)
		roleRepo.On("FindByPersonID", mock.Anything, personID).Return([]domain.PersonRole{
			{RoleType: domain.RoleProfessional, IsActive: true},
		}, nil)

		claims := auth.AuthClaims{Subject: "sub-6", Email: "precreated@test.com", Roles: []string{"PROFESSIONAL"}}
		status, err := svc.GetStatus(context.Background(), claims)

		require.NoError(t, err)
		assert.Equal(t, &personID, status.PersonID)
		assert.Equal(t, &campusID, status.CampusID)
		assert.False(t, status.NeedsProfileCompletion)
		assert.False(t, status.NeedsCampusAssignment)
		assert.False(t, status.NeedsAgreement)
	})

	t.Run("non-volunteer skips agreement check", func(t *testing.T) {
		userRepo, _, roleRepo, _, svc := newOnboardingTestDeps()

		appUser := &domain.AppUser{ID: userID, PersonID: &personID, CampusID: &campusID}
		userRepo.On("FindByKeycloakSubject", mock.Anything, "sub-7").Return(appUser, nil)
		roleRepo.On("FindByPersonID", mock.Anything, personID).Return([]domain.PersonRole{
			{RoleType: domain.RoleProfessional, IsActive: true},
		}, nil)

		claims := auth.AuthClaims{Subject: "sub-7", Email: "pro@test.com", Roles: []string{"PROFESSIONAL"}}
		status, err := svc.GetStatus(context.Background(), claims)

		require.NoError(t, err)
		assert.False(t, status.NeedsAgreement)
	})
}
