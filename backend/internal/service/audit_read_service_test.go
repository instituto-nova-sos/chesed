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

// MockAuditListRepository implements AuditListRepository for testing.
type MockAuditListRepository struct {
	mock.Mock
}

func (m *MockAuditListRepository) List(ctx context.Context, filter domain.AuditLogFilter) (*domain.AuditLogListResult, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AuditLogListResult), args.Error(1)
}

func newTestAuditReadService() (*AuditReadService, *MockAuditListRepository) {
	repo := new(MockAuditListRepository)
	return NewAuditReadService(repo), repo
}

func auditReadContext(campusID uuid.UUID) context.Context {
	return auth.NewContext(context.Background(), auth.AuthClaims{
		Subject:  uuid.New().String(),
		Roles:    []string{"ADMIN"},
		CampusID: campusID,
	})
}

func TestAuditReadService_List(t *testing.T) {
	t.Run("injects caller campus and delegates to repo", func(t *testing.T) {
		svc, repo := newTestAuditReadService()
		campusID := uuid.New()
		ctx := auditReadContext(campusID)

		repo.On("List", ctx, mock.MatchedBy(func(f domain.AuditLogFilter) bool {
			return f.CampusID == campusID && f.Page == 1 && f.PerPage == 50
		})).Return(&domain.AuditLogListResult{Data: []domain.AuditLogListItem{}, Pagination: domain.Pagination{Page: 1, PerPage: 50}}, nil)

		result, err := svc.List(ctx, domain.AuditLogFilter{Page: 1, PerPage: 50})
		require.NoError(t, err)
		assert.NotNil(t, result)
		repo.AssertExpectations(t)
	})

	t.Run("missing campus returns ErrForbidden", func(t *testing.T) {
		svc, repo := newTestAuditReadService()
		ctx := context.Background() // no claims

		_, err := svc.List(ctx, domain.AuditLogFilter{Page: 1, PerPage: 50})
		require.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrForbidden))
		repo.AssertNotCalled(t, "List", mock.Anything, mock.Anything)
	})

	t.Run("per_page above 100 is clamped to 100", func(t *testing.T) {
		svc, repo := newTestAuditReadService()
		campusID := uuid.New()
		ctx := auditReadContext(campusID)

		repo.On("List", ctx, mock.MatchedBy(func(f domain.AuditLogFilter) bool {
			return f.PerPage == 100
		})).Return(&domain.AuditLogListResult{Data: []domain.AuditLogListItem{}}, nil)

		_, err := svc.List(ctx, domain.AuditLogFilter{Page: 1, PerPage: 5000})
		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("defaults page and per_page when unset", func(t *testing.T) {
		svc, repo := newTestAuditReadService()
		campusID := uuid.New()
		ctx := auditReadContext(campusID)

		repo.On("List", ctx, mock.MatchedBy(func(f domain.AuditLogFilter) bool {
			return f.Page == 1 && f.PerPage == 50
		})).Return(&domain.AuditLogListResult{Data: []domain.AuditLogListItem{}}, nil)

		_, err := svc.List(ctx, domain.AuditLogFilter{})
		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}
