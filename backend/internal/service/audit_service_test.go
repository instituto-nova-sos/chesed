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

const (
	auditUserSubject = "11111111-1111-1111-1111-111111111111"
)

func auditBaseClaims(campusID uuid.UUID) auth.AuthClaims {
	return auth.AuthClaims{
		Subject:  auditUserSubject,
		Email:    "user@example.com",
		Roles:    []string{"ADMIN"},
		CampusID: campusID,
	}
}

func TestAuditService_LogAction_AllFields(t *testing.T) {
	campusID := uuid.New()
	entityID := uuid.New()
	repo := new(MockAuditRepository)
	svc := NewAuditService(repo)
	ctx := auth.NewContext(context.Background(), auditBaseClaims(campusID))

	params := AuditParams{
		ActionType:  "CREATE",
		EntityType:  "app_user",
		EntityID:    &entityID,
		Module:      "auth",
		Description: "test action",
		IPAddress:   "192.168.1.1",
		UserAgent:   "test-agent/1.0",
		Success:     true,
	}

	repo.On("Create", ctx, mock.MatchedBy(func(entry domain.AuditLog) bool {
		return auditEntryMatchesAllFields(entry, campusID, entityID)
	})).Return(nil)

	require.NoError(t, svc.LogAction(ctx, params))
	repo.AssertExpectations(t)
}

func ptrEq[T comparable](p *T, want T) bool { return p != nil && *p == want }

// auditEntryMatchesAllFields verifies every field of a fully-populated audit entry.
func auditEntryMatchesAllFields(entry domain.AuditLog, campusID, entityID uuid.UUID) bool {
	parsedUserID, _ := uuid.Parse(auditUserSubject)
	checks := []bool{
		entry.ActionType == "CREATE",
		entry.EntityType == "app_user",
		ptrEq(entry.EntityID, entityID),
		ptrEq(entry.UserID, parsedUserID),
		ptrEq(entry.CampusID, campusID),
		ptrEq(entry.Module, "auth"),
		ptrEq(entry.Description, "test action"),
		ptrEq(entry.IPAddress, "192.168.1.1"),
		ptrEq(entry.UserAgent, "test-agent/1.0"),
		entry.Success,
		entry.ID != uuid.Nil,
	}
	for _, ok := range checks {
		if !ok {
			return false
		}
	}
	return true
}

func TestAuditService_LogAction_Values(t *testing.T) {
	campusID := uuid.New()
	tests := []struct {
		name      string
		params    AuditParams
		wantValue bool // true: OldValues/NewValues populated; false: both nil
	}{
		{
			name:      "marshals old and new values",
			params:    AuditParams{ActionType: "UPDATE", EntityType: "person", OldValues: map[string]string{"name": "old"}, NewValues: map[string]string{"name": "new"}, Success: true},
			wantValue: true,
		},
		{
			name:      "handles nil old and new values",
			params:    AuditParams{ActionType: "DELETE", EntityType: "person", Success: true},
			wantValue: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockAuditRepository)
			svc := NewAuditService(repo)
			ctx := auth.NewContext(context.Background(), auditBaseClaims(campusID))
			repo.On("Create", ctx, mock.MatchedBy(func(entry domain.AuditLog) bool {
				if tt.wantValue {
					return entry.OldValues != nil && entry.NewValues != nil
				}
				return entry.OldValues == nil && entry.NewValues == nil
			})).Return(nil)

			require.NoError(t, svc.LogAction(ctx, tt.params))
			repo.AssertExpectations(t)
		})
	}
}

func TestAuditService_LogAction_Identity(t *testing.T) {
	campusID := uuid.New()
	tests := []struct {
		name        string
		claims      auth.AuthClaims
		wantNilUser bool
		wantNilCamp bool
	}{
		{"non-UUID subject", auth.AuthClaims{Subject: "not-a-uuid", CampusID: campusID}, true, false},
		{"empty subject", auth.AuthClaims{CampusID: campusID}, true, false},
		{"nil campus_id", auth.AuthClaims{Subject: auditUserSubject}, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockAuditRepository)
			svc := NewAuditService(repo)
			ctx := auth.NewContext(context.Background(), tt.claims)
			repo.On("Create", ctx, mock.MatchedBy(func(entry domain.AuditLog) bool {
				userOK := !tt.wantNilUser || entry.UserID == nil
				campOK := !tt.wantNilCamp || entry.CampusID == nil
				return userOK && campOK
			})).Return(nil)

			require.NoError(t, svc.LogAction(ctx, AuditParams{ActionType: "CREATE", EntityType: "test", Success: true}))
			repo.AssertExpectations(t)
		})
	}
}

func TestAuditService_LogAction_RepoError(t *testing.T) {
	campusID := uuid.New()
	repo := new(MockAuditRepository)
	svc := NewAuditService(repo)
	ctx := auth.NewContext(context.Background(), auditBaseClaims(campusID))

	repo.On("Create", ctx, mock.Anything).Return(errors.New("db connection failed"))

	err := svc.LogAction(ctx, AuditParams{ActionType: "CREATE", EntityType: "app_user", Success: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db connection failed")
	repo.AssertExpectations(t)
}
