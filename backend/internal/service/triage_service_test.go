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

// MockTriageRepository implements TriageRepository for testing.
type MockTriageRepository struct {
	mock.Mock
}

func (m *MockTriageRepository) Create(ctx context.Context, triage domain.Triage) (*domain.Triage, error) {
	args := m.Called(ctx, triage)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Triage), args.Error(1)
}

func (m *MockTriageRepository) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Triage, error) {
	args := m.Called(ctx, id, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Triage), args.Error(1)
}

func (m *MockTriageRepository) List(ctx context.Context, filter domain.TriageFilter) (*domain.TriageListResult, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TriageListResult), args.Error(1)
}

func (m *MockTriageRepository) Update(ctx context.Context, triage domain.Triage) (*domain.Triage, error) {
	args := m.Called(ctx, triage)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Triage), args.Error(1)
}

func newTestTriageService() (*TriageService, *MockTriageRepository, *MockAuditRepository) {
	repo := new(MockTriageRepository)
	auditRepo := new(MockAuditRepository)
	auditSvc := NewAuditService(auditRepo)
	return NewTriageService(repo, auditSvc), repo, auditRepo
}

func triageTestContext() (context.Context, auth.AuthClaims) {
	claims := auth.AuthClaims{
		Subject:  uuid.New().String(),
		Email:    "triager@chesed.test",
		Roles:    []string{"PROFESSIONAL"},
		CampusID: uuid.New(),
	}
	return auth.NewContext(context.Background(), claims), claims
}

func TestTriageService_CreateTriage(t *testing.T) {
	t.Run("success with minimal input", func(t *testing.T) {
		svc, repo, auditRepo := newTestTriageService()
		ctx, claims := triageTestContext()

		repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Triage")).
			Return(&domain.Triage{ID: uuid.New(), MainComplaint: "headache"}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		input := CreateTriageInput{
			PersonID:      uuid.New().String(),
			MainComplaint: "headache",
		}

		result, err := svc.CreateTriage(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, "headache", result.MainComplaint)
		repo.AssertCalled(t, "Create", mock.Anything, mock.MatchedBy(func(t domain.Triage) bool {
			return t.CampusID == claims.CampusID && t.IsActive && !t.TriageDate.IsZero()
		}))
	})

	t.Run("success with requested service types", func(t *testing.T) {
		svc, repo, auditRepo := newTestTriageService()
		ctx, _ := triageTestContext()
		st1, st2 := uuid.New(), uuid.New()

		repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Triage")).
			Return(&domain.Triage{ID: uuid.New()}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		input := CreateTriageInput{
			PersonID:              uuid.New().String(),
			MainComplaint:         "needs food + clothing",
			RequestedServiceTypes: []string{st1.String(), st2.String()},
		}

		_, err := svc.CreateTriage(ctx, input)
		require.NoError(t, err)
		repo.AssertCalled(t, "Create", mock.Anything, mock.MatchedBy(func(tr domain.Triage) bool {
			return len(tr.RequestedTypes) == 2
		}))
	})

	t.Run("forbidden without campus context", func(t *testing.T) {
		svc, _, _ := newTestTriageService()
		ctx := auth.NewContext(context.Background(), auth.AuthClaims{Subject: uuid.New().String()})

		_, err := svc.CreateTriage(ctx, CreateTriageInput{
			PersonID:      uuid.New().String(),
			MainComplaint: "x",
		})
		require.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("validation error on missing complaint", func(t *testing.T) {
		svc, _, _ := newTestTriageService()
		ctx, _ := triageTestContext()

		_, err := svc.CreateTriage(ctx, CreateTriageInput{
			PersonID: uuid.New().String(),
		})
		require.Error(t, err)
	})

	t.Run("validation error on invalid uuid in services", func(t *testing.T) {
		svc, _, _ := newTestTriageService()
		ctx, _ := triageTestContext()

		_, err := svc.CreateTriage(ctx, CreateTriageInput{
			PersonID:              uuid.New().String(),
			MainComplaint:         "x",
			RequestedServiceTypes: []string{"not-a-uuid"},
		})
		require.Error(t, err)
	})
}

func TestTriageService_GetTriage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, repo, _ := newTestTriageService()
		ctx, claims := triageTestContext()
		id := uuid.New()

		repo.On("FindByID", mock.Anything, id, claims.CampusID).
			Return(&domain.Triage{ID: id, MainComplaint: "x"}, nil)

		result, err := svc.GetTriage(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, id, result.ID)
	})

	t.Run("not found", func(t *testing.T) {
		svc, repo, _ := newTestTriageService()
		ctx, _ := triageTestContext()
		id := uuid.New()

		repo.On("FindByID", mock.Anything, id, mock.AnythingOfType("uuid.UUID")).
			Return(nil, domain.ErrNotFound)

		_, err := svc.GetTriage(ctx, id)
		require.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("forbidden without campus", func(t *testing.T) {
		svc, _, _ := newTestTriageService()
		ctx := auth.NewContext(context.Background(), auth.AuthClaims{Subject: uuid.New().String()})
		_, err := svc.GetTriage(ctx, uuid.New())
		require.ErrorIs(t, err, domain.ErrForbidden)
	})
}

func TestTriageService_ListTriages(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, repo, _ := newTestTriageService()
		ctx, claims := triageTestContext()

		repo.On("List", mock.Anything, mock.MatchedBy(func(f domain.TriageFilter) bool {
			return f.CampusID == claims.CampusID
		})).Return(&domain.TriageListResult{
			Data:       []domain.TriageListItem{{ID: uuid.New(), MainComplaint: "x"}},
			Pagination: domain.Pagination{Page: 1, PerPage: 20, Total: 1, TotalPages: 1},
		}, nil)

		result, err := svc.ListTriages(ctx, domain.TriageFilter{Page: 1, PerPage: 20})
		require.NoError(t, err)
		assert.Equal(t, 1, result.Pagination.Total)
	})

	t.Run("forbidden", func(t *testing.T) {
		svc, _, _ := newTestTriageService()
		ctx := auth.NewContext(context.Background(), auth.AuthClaims{Subject: uuid.New().String()})
		_, err := svc.ListTriages(ctx, domain.TriageFilter{})
		require.ErrorIs(t, err, domain.ErrForbidden)
	})
}

func TestTriageService_UpdateTriage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, repo, auditRepo := newTestTriageService()
		ctx, claims := triageTestContext()
		id := uuid.New()

		repo.On("FindByID", mock.Anything, id, claims.CampusID).
			Return(&domain.Triage{ID: id, CampusID: claims.CampusID, MainComplaint: "old", TriageDate: time.Now()}, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("domain.Triage")).
			Return(&domain.Triage{ID: id, MainComplaint: "new"}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		result, err := svc.UpdateTriage(ctx, id, UpdateTriageInput{MainComplaint: "new"})
		require.NoError(t, err)
		assert.Equal(t, "new", result.MainComplaint)
	})

	t.Run("not found", func(t *testing.T) {
		svc, repo, _ := newTestTriageService()
		ctx, claims := triageTestContext()
		id := uuid.New()

		repo.On("FindByID", mock.Anything, id, claims.CampusID).Return(nil, domain.ErrNotFound)

		_, err := svc.UpdateTriage(ctx, id, UpdateTriageInput{MainComplaint: "new"})
		require.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("forbidden", func(t *testing.T) {
		svc, _, _ := newTestTriageService()
		ctx := auth.NewContext(context.Background(), auth.AuthClaims{Subject: uuid.New().String()})
		_, err := svc.UpdateTriage(ctx, uuid.New(), UpdateTriageInput{MainComplaint: "x"})
		require.ErrorIs(t, err, domain.ErrForbidden)
	})
}

func TestParseHelpers(t *testing.T) {
	t.Run("parseOptionalTime returns nil for empty", func(t *testing.T) {
		got, err := parseOptionalTime(nil)
		require.NoError(t, err)
		assert.Nil(t, got)

		empty := ""
		got, err = parseOptionalTime(&empty)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("parseOptionalTime parses RFC3339", func(t *testing.T) {
		s := "2026-05-27T15:00:00Z"
		got, err := parseOptionalTime(&s)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, 2026, got.Year())
	})

	t.Run("parseOptionalTime rejects bad format", func(t *testing.T) {
		s := "not a date"
		_, err := parseOptionalTime(&s)
		require.Error(t, err)
	})

	t.Run("parseOptionalUUID handles nil/empty", func(t *testing.T) {
		got, err := parseOptionalUUID(nil)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("parseUUIDList rejects invalid", func(t *testing.T) {
		_, err := parseUUIDList([]string{"bad"})
		require.Error(t, err)
	})

	t.Run("parseUUIDList returns nil for empty", func(t *testing.T) {
		got, err := parseUUIDList(nil)
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}
