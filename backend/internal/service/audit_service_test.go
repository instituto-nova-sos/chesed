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

// MockAuditRepository is a testify mock for AuditRepository.
type MockAuditRepository struct {
	mock.Mock
}

func (m *MockAuditRepository) Create(ctx context.Context, entry domain.AuditLog) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func TestAuditService_LogAction(t *testing.T) {
	campusID := uuid.New()
	userSubject := uuid.New().String()
	entityID := uuid.New()

	baseClaims := auth.AuthClaims{
		Subject:  userSubject,
		Email:    "user@example.com",
		Roles:    []string{"ADMIN"},
		CampusID: campusID,
	}

	baseParams := AuditParams{
		ActionType:  "CREATE",
		EntityType:  "app_user",
		EntityID:    &entityID,
		Module:      "auth",
		Description: "test action",
		IPAddress:   "192.168.1.1",
		UserAgent:   "test-agent/1.0",
		Success:     true,
	}

	t.Run("creates entry with all fields", func(t *testing.T) {
		repo := new(MockAuditRepository)
		svc := NewAuditService(repo)
		ctx := auth.NewContext(context.Background(), baseClaims)

		repo.On("Create", ctx, mock.MatchedBy(func(entry domain.AuditLog) bool {
			parsedUserID, _ := uuid.Parse(userSubject)
			return entry.ActionType == "CREATE" &&
				entry.EntityType == "app_user" &&
				entry.EntityID != nil && *entry.EntityID == entityID &&
				entry.UserID != nil && *entry.UserID == parsedUserID &&
				entry.CampusID != nil && *entry.CampusID == campusID &&
				entry.Module != nil && *entry.Module == "auth" &&
				entry.Description != nil && *entry.Description == "test action" &&
				entry.IPAddress != nil && *entry.IPAddress == "192.168.1.1" &&
				entry.UserAgent != nil && *entry.UserAgent == "test-agent/1.0" &&
				entry.Success &&
				entry.ID != uuid.Nil
		})).Return(nil)

		err := svc.LogAction(ctx, baseParams)
		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("marshals old and new values", func(t *testing.T) {
		repo := new(MockAuditRepository)
		svc := NewAuditService(repo)
		ctx := auth.NewContext(context.Background(), baseClaims)

		params := AuditParams{
			ActionType: "UPDATE",
			EntityType: "person",
			OldValues:  map[string]string{"name": "old"},
			NewValues:  map[string]string{"name": "new"},
			Success:    true,
		}

		repo.On("Create", ctx, mock.MatchedBy(func(entry domain.AuditLog) bool {
			return entry.OldValues != nil && entry.NewValues != nil
		})).Return(nil)

		err := svc.LogAction(ctx, params)
		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("handles nil old and new values", func(t *testing.T) {
		repo := new(MockAuditRepository)
		svc := NewAuditService(repo)
		ctx := auth.NewContext(context.Background(), baseClaims)

		params := AuditParams{
			ActionType: "DELETE",
			EntityType: "person",
			Success:    true,
		}

		repo.On("Create", ctx, mock.MatchedBy(func(entry domain.AuditLog) bool {
			return entry.OldValues == nil && entry.NewValues == nil
		})).Return(nil)

		err := svc.LogAction(ctx, params)
		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("handles non-UUID subject", func(t *testing.T) {
		repo := new(MockAuditRepository)
		svc := NewAuditService(repo)
		claims := auth.AuthClaims{Subject: "not-a-uuid", CampusID: campusID}
		ctx := auth.NewContext(context.Background(), claims)

		repo.On("Create", ctx, mock.MatchedBy(func(entry domain.AuditLog) bool {
			return entry.UserID == nil
		})).Return(nil)

		err := svc.LogAction(ctx, AuditParams{ActionType: "CREATE", EntityType: "test", Success: true})
		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("handles empty subject", func(t *testing.T) {
		repo := new(MockAuditRepository)
		svc := NewAuditService(repo)
		claims := auth.AuthClaims{CampusID: campusID}
		ctx := auth.NewContext(context.Background(), claims)

		repo.On("Create", ctx, mock.MatchedBy(func(entry domain.AuditLog) bool {
			return entry.UserID == nil
		})).Return(nil)

		err := svc.LogAction(ctx, AuditParams{ActionType: "CREATE", EntityType: "test", Success: true})
		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("nil campus_id in context", func(t *testing.T) {
		repo := new(MockAuditRepository)
		svc := NewAuditService(repo)
		claims := auth.AuthClaims{Subject: userSubject}
		ctx := auth.NewContext(context.Background(), claims)

		repo.On("Create", ctx, mock.MatchedBy(func(entry domain.AuditLog) bool {
			return entry.CampusID == nil
		})).Return(nil)

		err := svc.LogAction(ctx, AuditParams{ActionType: "CREATE", EntityType: "test", Success: true})
		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("repo error propagated", func(t *testing.T) {
		repo := new(MockAuditRepository)
		svc := NewAuditService(repo)
		ctx := auth.NewContext(context.Background(), baseClaims)

		repo.On("Create", ctx, mock.Anything).Return(errors.New("db connection failed"))

		err := svc.LogAction(ctx, baseParams)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db connection failed")
		repo.AssertExpectations(t)
	})
}
