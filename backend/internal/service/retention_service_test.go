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

type mockRetentionRepo struct{ mock.Mock }

func (m *mockRetentionRepo) ListExpiredPersonIDs(ctx context.Context, campusID uuid.UUID, olderThan time.Time) ([]uuid.UUID, error) {
	args := m.Called(ctx, campusID, olderThan)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

// MockPersonAnonymizer (declared in consent_service_test.go, WS-A) satisfies the
// PersonAnonymizer interface; retention reuses it. Declare a local stub here to
// avoid coupling test ordering across files.
type retentionAnonymizerStub struct{ mock.Mock }

func (m *retentionAnonymizerStub) Anonymize(ctx context.Context, personID, campusID uuid.UUID) error {
	args := m.Called(ctx, personID, campusID)
	return args.Error(0)
}

func retentionCtx() (context.Context, auth.AuthClaims) {
	claims := auth.AuthClaims{
		Subject:  uuid.New().String(),
		Roles:    []string{"ADMIN"},
		CampusID: uuid.New(),
	}
	return auth.NewContext(context.Background(), claims), claims
}

func TestRetentionService_Run(t *testing.T) {
	t.Run("anonymizes each expired person and audits, returning the summary", func(t *testing.T) {
		repo := new(mockRetentionRepo)
		anon := new(retentionAnonymizerStub)
		auditRepo := new(MockAuditRepository)
		svc := NewRetentionService(repo, anon, NewAuditService(auditRepo))
		ctx, claims := retentionCtx()

		id1, id2 := uuid.New(), uuid.New()
		repo.On("ListExpiredPersonIDs", mock.Anything, claims.CampusID, mock.AnythingOfType("time.Time")).
			Return([]uuid.UUID{id1, id2}, nil)
		anon.On("Anonymize", mock.Anything, id1, claims.CampusID).Return(nil)
		anon.On("Anonymize", mock.Anything, id2, claims.CampusID).Return(nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		summary, err := svc.Run(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, summary.Scanned)
		assert.Equal(t, 2, summary.Anonymized)
		anon.AssertNumberOfCalls(t, "Anonymize", 2)
	})

	t.Run("empty sweep returns zero summary", func(t *testing.T) {
		repo := new(mockRetentionRepo)
		anon := new(retentionAnonymizerStub)
		auditRepo := new(MockAuditRepository)
		svc := NewRetentionService(repo, anon, NewAuditService(auditRepo))
		ctx, claims := retentionCtx()

		repo.On("ListExpiredPersonIDs", mock.Anything, claims.CampusID, mock.Anything).
			Return([]uuid.UUID{}, nil)

		summary, err := svc.Run(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, summary.Scanned)
		assert.Equal(t, 0, summary.Anonymized)
		anon.AssertNotCalled(t, "Anonymize", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("missing campus is forbidden", func(t *testing.T) {
		repo := new(mockRetentionRepo)
		anon := new(retentionAnonymizerStub)
		auditRepo := new(MockAuditRepository)
		svc := NewRetentionService(repo, anon, NewAuditService(auditRepo))

		_, err := svc.Run(context.Background())
		require.ErrorIs(t, err, domain.ErrForbidden)
		repo.AssertNotCalled(t, "ListExpiredPersonIDs")
	})
}
