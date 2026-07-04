package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockConsentRepository implements ConsentRepository for testing.
type MockConsentRepository struct {
	mock.Mock
}

func (m *MockConsentRepository) Create(ctx context.Context, c domain.Consent) (*domain.Consent, error) {
	args := m.Called(ctx, c)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Consent), args.Error(1)
}

func (m *MockConsentRepository) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Consent, error) {
	args := m.Called(ctx, id, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Consent), args.Error(1)
}

func (m *MockConsentRepository) ListByPerson(ctx context.Context, personID, campusID uuid.UUID) ([]domain.Consent, error) {
	args := m.Called(ctx, personID, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Consent), args.Error(1)
}

func (m *MockConsentRepository) Revoke(ctx context.Context, id, campusID uuid.UUID, reason string) (*domain.Consent, error) {
	args := m.Called(ctx, id, campusID, reason)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Consent), args.Error(1)
}

// MockConsentPersonRepository implements ConsentPersonRepository for testing.
type MockConsentPersonRepository struct {
	mock.Mock
}

func (m *MockConsentPersonRepository) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Person, error) {
	args := m.Called(ctx, id, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Person), args.Error(1)
}

func newTestConsentService() (*ConsentService, *MockConsentRepository, *MockConsentPersonRepository, *MockAuditRepository) {
	repo := new(MockConsentRepository)
	personRepo := new(MockConsentPersonRepository)
	auditRepo := new(MockAuditRepository)
	auditSvc := NewAuditService(auditRepo)
	return NewConsentService(repo, personRepo, auditSvc), repo, personRepo, auditRepo
}

func consentTestContext() (context.Context, auth.AuthClaims) {
	claims := auth.AuthClaims{
		Subject:  uuid.New().String(),
		Email:    "volunteer@chesed.test",
		Roles:    []string{"VOLUNTEER"},
		CampusID: uuid.New(),
	}
	return auth.NewContext(context.Background(), claims), claims
}

func validCreateConsentInput(personID uuid.UUID) CreateConsentInput {
	return CreateConsentInput{
		PersonID:    personID.String(),
		ConsentType: "DATA_PROCESSING",
		Purpose:     "Registration and attendance follow-up",
	}
}

func TestConsentService_CreateConsent(t *testing.T) {
	t.Run("success with defaults stamps campus, active flag and version 1.0", func(t *testing.T) {
		svc, repo, personRepo, auditRepo := newTestConsentService()
		ctx, claims := consentTestContext()

		personID := uuid.New()
		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).
			Return(&domain.Person{ID: personID}, nil)
		repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Consent")).
			Return(&domain.Consent{ID: uuid.New(), PersonID: personID, ConsentType: "DATA_PROCESSING", ConsentVersion: "1.0", IsActive: true}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		created, err := svc.CreateConsent(ctx, validCreateConsentInput(personID))
		require.NoError(t, err)
		assert.True(t, created.IsActive)

		stored := repo.Calls[0].Arguments.Get(1).(domain.Consent)
		assert.Equal(t, claims.CampusID, stored.CampusID)
		assert.Equal(t, "1.0", stored.ConsentVersion)
		assert.True(t, stored.IsActive)
		assert.NotEqual(t, uuid.Nil, stored.ID)
		auditRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("domain.AuditLog"))
	})

	t.Run("explicit consent_version is preserved", func(t *testing.T) {
		svc, repo, personRepo, auditRepo := newTestConsentService()
		ctx, claims := consentTestContext()

		personID := uuid.New()
		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).
			Return(&domain.Person{ID: personID}, nil)
		repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Consent")).
			Return(&domain.Consent{ID: uuid.New(), ConsentVersion: "2.1", IsActive: true}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		input := validCreateConsentInput(personID)
		input.ConsentVersion = "2.1"
		_, err := svc.CreateConsent(ctx, input)
		require.NoError(t, err)

		stored := repo.Calls[0].Arguments.Get(1).(domain.Consent)
		assert.Equal(t, "2.1", stored.ConsentVersion)
	})

	t.Run("missing campus context returns forbidden", func(t *testing.T) {
		svc, _, _, _ := newTestConsentService()

		_, err := svc.CreateConsent(context.Background(), validCreateConsentInput(uuid.New()))
		require.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("unknown consent type fails validation without repo call", func(t *testing.T) {
		svc, repo, _, _ := newTestConsentService()
		ctx, _ := consentTestContext()

		input := validCreateConsentInput(uuid.New())
		input.ConsentType = "MARKETING"
		_, err := svc.CreateConsent(ctx, input)
		require.Error(t, err)
		repo.AssertNotCalled(t, "Create")
	})

	t.Run("missing purpose fails validation without repo call", func(t *testing.T) {
		svc, repo, _, _ := newTestConsentService()
		ctx, _ := consentTestContext()

		input := validCreateConsentInput(uuid.New())
		input.Purpose = ""
		_, err := svc.CreateConsent(ctx, input)
		require.Error(t, err)
		repo.AssertNotCalled(t, "Create")
	})
}

func TestConsentService_CreateConsentReferences(t *testing.T) {
	t.Run("person outside campus is a generic validation error", func(t *testing.T) {
		svc, repo, personRepo, _ := newTestConsentService()
		ctx, _ := consentTestContext()

		personRepo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, domain.ErrNotFound)

		_, err := svc.CreateConsent(ctx, validCreateConsentInput(uuid.New()))
		require.ErrorIs(t, err, domain.ErrValidation)
		repo.AssertNotCalled(t, "Create")
	})

	t.Run("granted_by outside campus is a generic validation error", func(t *testing.T) {
		svc, repo, personRepo, _ := newTestConsentService()
		ctx, claims := consentTestContext()

		personID := uuid.New()
		guardianID := uuid.New()
		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).
			Return(&domain.Person{ID: personID}, nil)
		personRepo.On("FindByID", mock.Anything, guardianID, claims.CampusID).
			Return(nil, domain.ErrNotFound)

		guardian := guardianID.String()
		input := validCreateConsentInput(personID)
		input.GrantedByPersonID = &guardian
		_, err := svc.CreateConsent(ctx, input)
		require.ErrorIs(t, err, domain.ErrValidation)
		repo.AssertNotCalled(t, "Create")
	})

	t.Run("granted_by in campus is accepted and stored", func(t *testing.T) {
		svc, repo, personRepo, auditRepo := newTestConsentService()
		ctx, claims := consentTestContext()

		personID := uuid.New()
		guardianID := uuid.New()
		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).
			Return(&domain.Person{ID: personID}, nil)
		personRepo.On("FindByID", mock.Anything, guardianID, claims.CampusID).
			Return(&domain.Person{ID: guardianID}, nil)
		repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Consent")).
			Return(&domain.Consent{ID: uuid.New(), IsActive: true}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		guardian := guardianID.String()
		input := validCreateConsentInput(personID)
		input.ConsentType = "MINOR_GUARDIAN"
		input.GrantedByPersonID = &guardian
		_, err := svc.CreateConsent(ctx, input)
		require.NoError(t, err)

		stored := repo.Calls[0].Arguments.Get(1).(domain.Consent)
		require.NotNil(t, stored.GrantedByPerson)
		assert.Equal(t, guardianID, *stored.GrantedByPerson)
	})

	t.Run("duplicate active consent propagates", func(t *testing.T) {
		svc, repo, personRepo, _ := newTestConsentService()
		ctx, claims := consentTestContext()

		personID := uuid.New()
		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).
			Return(&domain.Person{ID: personID}, nil)
		repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Consent")).
			Return(nil, domain.ErrDuplicate)

		_, err := svc.CreateConsent(ctx, validCreateConsentInput(personID))
		require.ErrorIs(t, err, domain.ErrDuplicate)
	})
}

func TestConsentService_ListPersonConsents(t *testing.T) {
	t.Run("returns registry rows for a person in campus", func(t *testing.T) {
		svc, repo, personRepo, _ := newTestConsentService()
		ctx, claims := consentTestContext()

		personID := uuid.New()
		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).
			Return(&domain.Person{ID: personID}, nil)
		repo.On("ListByPerson", mock.Anything, personID, claims.CampusID).
			Return([]domain.Consent{
				{ID: uuid.New(), ConsentType: "IMAGE_USAGE", IsActive: false},
				{ID: uuid.New(), ConsentType: "DATA_PROCESSING", IsActive: true},
			}, nil)

		consents, err := svc.ListPersonConsents(ctx, personID)
		require.NoError(t, err)
		require.Len(t, consents, 2)
	})

	t.Run("person not visible in campus returns not found", func(t *testing.T) {
		svc, repo, personRepo, _ := newTestConsentService()
		ctx, _ := consentTestContext()

		personRepo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, domain.ErrNotFound)

		_, err := svc.ListPersonConsents(ctx, uuid.New())
		require.ErrorIs(t, err, domain.ErrNotFound)
		repo.AssertNotCalled(t, "ListByPerson")
	})

	t.Run("missing campus returns forbidden", func(t *testing.T) {
		svc, _, _, _ := newTestConsentService()

		_, err := svc.ListPersonConsents(context.Background(), uuid.New())
		require.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("nil repository result becomes an empty slice", func(t *testing.T) {
		svc, repo, personRepo, _ := newTestConsentService()
		ctx, claims := consentTestContext()

		personID := uuid.New()
		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).
			Return(&domain.Person{ID: personID}, nil)
		repo.On("ListByPerson", mock.Anything, personID, claims.CampusID).
			Return([]domain.Consent(nil), nil)

		consents, err := svc.ListPersonConsents(ctx, personID)
		require.NoError(t, err)
		assert.NotNil(t, consents)
		assert.Empty(t, consents)
	})
}

func TestConsentService_RevokeConsent(t *testing.T) {
	t.Run("success revokes and audits old and new values", func(t *testing.T) {
		svc, repo, _, auditRepo := newTestConsentService()
		ctx, claims := consentTestContext()

		id := uuid.New()
		now := time.Now()
		reason := "Data subject request"
		repo.On("FindByID", mock.Anything, id, claims.CampusID).
			Return(&domain.Consent{ID: id, IsActive: true}, nil)
		repo.On("Revoke", mock.Anything, id, claims.CampusID, reason).
			Return(&domain.Consent{ID: id, IsActive: false, RevokedAt: &now, RevokedReason: &reason}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		revoked, err := svc.RevokeConsent(ctx, id, RevokeConsentInput{RevokedReason: reason})
		require.NoError(t, err)
		assert.False(t, revoked.IsActive)
		require.NotNil(t, revoked.RevokedAt)
		auditRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("domain.AuditLog"))
	})

	t.Run("not found propagates for cross-campus consent", func(t *testing.T) {
		svc, repo, _, _ := newTestConsentService()
		ctx, _ := consentTestContext()

		repo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, domain.ErrNotFound)

		_, err := svc.RevokeConsent(ctx, uuid.New(), RevokeConsentInput{RevokedReason: "reason"})
		require.ErrorIs(t, err, domain.ErrNotFound)
		repo.AssertNotCalled(t, "Revoke")
	})

	t.Run("already revoked consent is a validation error", func(t *testing.T) {
		svc, repo, _, _ := newTestConsentService()
		ctx, claims := consentTestContext()

		id := uuid.New()
		repo.On("FindByID", mock.Anything, id, claims.CampusID).
			Return(&domain.Consent{ID: id, IsActive: false}, nil)

		_, err := svc.RevokeConsent(ctx, id, RevokeConsentInput{RevokedReason: "reason"})
		require.ErrorIs(t, err, domain.ErrValidation)
		repo.AssertNotCalled(t, "Revoke")
	})

	t.Run("missing reason fails validation without repo call", func(t *testing.T) {
		svc, repo, _, _ := newTestConsentService()
		ctx, _ := consentTestContext()

		_, err := svc.RevokeConsent(ctx, uuid.New(), RevokeConsentInput{})
		require.Error(t, err)
		repo.AssertNotCalled(t, "FindByID")
	})

	t.Run("missing campus returns forbidden", func(t *testing.T) {
		svc, _, _, _ := newTestConsentService()

		_, err := svc.RevokeConsent(context.Background(), uuid.New(), RevokeConsentInput{RevokedReason: "reason"})
		require.ErrorIs(t, err, domain.ErrForbidden)
	})
}
