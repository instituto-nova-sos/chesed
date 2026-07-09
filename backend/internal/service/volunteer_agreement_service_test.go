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

func newAgreementTestContext() (context.Context, auth.AuthClaims) {
	claims := auth.AuthClaims{
		Subject:  uuid.New().String(),
		Email:    "volunteer@chesed.test",
		Roles:    []string{"VOLUNTEER"},
		CampusID: uuid.New(),
		PersonID: uuid.New(),
	}
	ctx := auth.NewContext(context.Background(), claims)
	return ctx, claims
}

func newTestAgreementService() (*VolunteerAgreementService, *MockVolunteerAgreementRepository, *MockPersonRoleRepository, *MockAuditRepository) {
	agreementRepo := new(MockVolunteerAgreementRepository)
	roleRepo := new(MockPersonRoleRepository)
	auditRepo := new(MockAuditRepository)
	auditSvc := NewAuditService(auditRepo)
	svc := NewVolunteerAgreementService(agreementRepo, roleRepo, auditSvc)
	return svc, agreementRepo, roleRepo, auditRepo
}

func TestVolunteerAgreementService_AcceptDigital(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, agreementRepo, _, auditRepo := newTestAgreementService()
		ctx, claims := newAgreementTestContext()

		pendingID := uuid.New()
		pending := &domain.VolunteerAgreement{
			ID:       pendingID,
			PersonID: claims.PersonID,
			Status:   domain.AgreementPending,
		}
		accepted := &domain.VolunteerAgreement{
			ID:              pendingID,
			PersonID:        claims.PersonID,
			Status:          domain.AgreementAccepted,
			SignatureMethod: strPtr(domain.SignatureDigital),
		}

		agreementRepo.On("HasAcceptedAgreement", mock.Anything, claims.PersonID).Return(false, nil)
		agreementRepo.On("FindPendingByPersonID", mock.Anything, claims.PersonID).Return(pending, nil)
		agreementRepo.On("AcceptDigital", mock.Anything, pendingID, mock.AnythingOfType("uuid.UUID"), "127.0.0.1", "TestBrowser", "data:image/png;base64,SIG").Return(accepted, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		result, err := svc.AcceptDigital(ctx, claims.PersonID, "127.0.0.1", "TestBrowser", "data:image/png;base64,SIG")

		require.NoError(t, err)
		assert.Equal(t, domain.AgreementAccepted, result.Status)
	})

	t.Run("empty signature returns validation error", func(t *testing.T) {
		svc, _, _, _ := newTestAgreementService()
		ctx, claims := newAgreementTestContext()

		_, err := svc.AcceptDigital(ctx, claims.PersonID, "127.0.0.1", "TestBrowser", "")

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("already accepted returns error", func(t *testing.T) {
		svc, agreementRepo, _, _ := newTestAgreementService()
		ctx, claims := newAgreementTestContext()

		agreementRepo.On("HasAcceptedAgreement", mock.Anything, claims.PersonID).Return(true, nil)

		_, err := svc.AcceptDigital(ctx, claims.PersonID, "127.0.0.1", "TestBrowser", "data:image/png;base64,SIG")

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAgreementExists)
	})

	t.Run("no pending agreement returns error", func(t *testing.T) {
		svc, agreementRepo, _, _ := newTestAgreementService()
		ctx, claims := newAgreementTestContext()

		agreementRepo.On("HasAcceptedAgreement", mock.Anything, claims.PersonID).Return(false, nil)
		agreementRepo.On("FindPendingByPersonID", mock.Anything, claims.PersonID).Return(nil, domain.ErrNotFound)

		_, err := svc.AcceptDigital(ctx, claims.PersonID, "127.0.0.1", "TestBrowser", "data:image/png;base64,SIG")

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestVolunteerAgreementService_Reject(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, agreementRepo, _, auditRepo := newTestAgreementService()
		ctx, claims := newAgreementTestContext()

		pendingID := uuid.New()
		pending := &domain.VolunteerAgreement{
			ID:       pendingID,
			PersonID: claims.PersonID,
			Status:   domain.AgreementPending,
		}
		reason := "Not interested"
		rejected := &domain.VolunteerAgreement{
			ID:              pendingID,
			PersonID:        claims.PersonID,
			Status:          domain.AgreementRejected,
			RejectionReason: &reason,
		}

		agreementRepo.On("FindPendingByPersonID", mock.Anything, claims.PersonID).Return(pending, nil)
		agreementRepo.On("Reject", mock.Anything, pendingID, &reason).Return(rejected, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		result, err := svc.Reject(ctx, claims.PersonID, &reason)

		require.NoError(t, err)
		assert.Equal(t, domain.AgreementRejected, result.Status)
		assert.Equal(t, "Not interested", *result.RejectionReason)
	})
}

func TestVolunteerAgreementService_CheckStatus(t *testing.T) {
	t.Run("has accepted", func(t *testing.T) {
		svc, agreementRepo, _, _ := newTestAgreementService()
		ctx, claims := newAgreementTestContext()

		agreementRepo.On("HasAcceptedAgreement", mock.Anything, claims.PersonID).Return(true, nil)

		accepted, err := svc.CheckStatus(ctx, claims.PersonID)

		require.NoError(t, err)
		assert.True(t, accepted)
	})

	t.Run("not accepted", func(t *testing.T) {
		svc, agreementRepo, _, _ := newTestAgreementService()
		ctx, claims := newAgreementTestContext()

		agreementRepo.On("HasAcceptedAgreement", mock.Anything, claims.PersonID).Return(false, nil)

		accepted, err := svc.CheckStatus(ctx, claims.PersonID)

		require.NoError(t, err)
		assert.False(t, accepted)
	})
}

func TestVolunteerAgreementService_GetAgreementText(t *testing.T) {
	svc, _, _, _ := newTestAgreementService()

	text, version := svc.GetAgreementText("v1")

	assert.Equal(t, "v1", version)
	assert.NotEmpty(t, text)
	assert.Contains(t, text, "Voluntário")
}

func TestVolunteerAgreementService_CreatePendingAgreement(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, agreementRepo, _, _ := newTestAgreementService()
		ctx, _ := newAgreementTestContext()

		personID := uuid.New()
		roleID := uuid.New()
		campusID := uuid.New()

		agreementRepo.On("Create", mock.Anything, mock.MatchedBy(func(a domain.VolunteerAgreement) bool {
			return a.PersonID == personID && a.PersonRoleID == roleID && a.Status == domain.AgreementPending
		})).Return(&domain.VolunteerAgreement{
			ID:       uuid.New(),
			PersonID: personID,
			Status:   domain.AgreementPending,
		}, nil)

		result, err := svc.CreatePendingAgreement(ctx, personID, roleID, campusID)

		require.NoError(t, err)
		assert.Equal(t, domain.AgreementPending, result.Status)
	})

	t.Run("idempotent if exists", func(t *testing.T) {
		svc, agreementRepo, _, _ := newTestAgreementService()
		ctx, _ := newAgreementTestContext()

		agreementRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.VolunteerAgreement")).
			Return(nil, domain.ErrAgreementExists)

		result, err := svc.CreatePendingAgreement(ctx, uuid.New(), uuid.New(), uuid.New())

		require.NoError(t, err)
		assert.Nil(t, result)
	})
}
