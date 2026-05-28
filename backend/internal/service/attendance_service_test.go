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

// MockAttendanceRepository implements AttendanceRepository for testing.
type MockAttendanceRepository struct {
	mock.Mock
}

func (m *MockAttendanceRepository) Create(ctx context.Context, a domain.Attendance) (*domain.Attendance, error) {
	args := m.Called(ctx, a)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Attendance), args.Error(1)
}

func (m *MockAttendanceRepository) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Attendance, error) {
	args := m.Called(ctx, id, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Attendance), args.Error(1)
}

func (m *MockAttendanceRepository) FindByIDWithTransitions(ctx context.Context, id, campusID uuid.UUID) (*domain.AttendanceDetail, error) {
	args := m.Called(ctx, id, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AttendanceDetail), args.Error(1)
}

func (m *MockAttendanceRepository) List(ctx context.Context, filter domain.AttendanceFilter) (*domain.AttendanceListResult, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AttendanceListResult), args.Error(1)
}

func (m *MockAttendanceRepository) Transition(ctx context.Context, t domain.AttendanceTransition) (*domain.Attendance, error) {
	args := m.Called(ctx, t)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Attendance), args.Error(1)
}

func (m *MockAttendanceRepository) UpdateNotes(ctx context.Context, id, campusID uuid.UUID, observations, recommendations *string) (*domain.Attendance, error) {
	args := m.Called(ctx, id, campusID, observations, recommendations)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Attendance), args.Error(1)
}

func newTestAttendanceService() (*AttendanceService, *MockAttendanceRepository, *MockAuditRepository) {
	repo := new(MockAttendanceRepository)
	auditRepo := new(MockAuditRepository)
	auditSvc := NewAuditService(auditRepo)
	return NewAttendanceService(repo, auditSvc), repo, auditRepo
}

func attendanceTestContext() (context.Context, auth.AuthClaims) {
	claims := auth.AuthClaims{
		Subject:  uuid.New().String(),
		Email:    "pro@chesed.test",
		Roles:    []string{"PROFESSIONAL"},
		CampusID: uuid.New(),
	}
	return auth.NewContext(context.Background(), claims), claims
}

func validCreateInput() CreateAttendanceInput {
	return CreateAttendanceInput{
		PersonID:       uuid.New().String(),
		ServiceTypeID:  uuid.New().String(),
		ProfessionalID: uuid.New().String(),
	}
}

func TestAttendanceService_CreateAttendance(t *testing.T) {
	t.Run("success creates SCHEDULED attendance", func(t *testing.T) {
		svc, repo, auditRepo := newTestAttendanceService()
		ctx, claims := attendanceTestContext()

		repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Attendance")).
			Return(&domain.Attendance{
				ID:     uuid.New(),
				Status: domain.AttendanceStatusScheduled,
			}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		result, err := svc.CreateAttendance(ctx, validCreateInput())
		require.NoError(t, err)
		assert.Equal(t, domain.AttendanceStatusScheduled, result.Status)

		repo.AssertCalled(t, "Create", mock.Anything, mock.MatchedBy(func(a domain.Attendance) bool {
			return a.CampusID == claims.CampusID && a.Status == domain.AttendanceStatusScheduled
		}))
	})

	t.Run("forbidden without campus", func(t *testing.T) {
		svc, _, _ := newTestAttendanceService()
		ctx := auth.NewContext(context.Background(), auth.AuthClaims{Subject: uuid.New().String()})
		_, err := svc.CreateAttendance(ctx, validCreateInput())
		require.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("validation rejects missing person_id", func(t *testing.T) {
		svc, _, _ := newTestAttendanceService()
		ctx, _ := attendanceTestContext()
		_, err := svc.CreateAttendance(ctx, CreateAttendanceInput{})
		require.Error(t, err)
	})
}

func TestAttendanceService_TransitionAttendance(t *testing.T) {
	t.Run("SCHEDULED to IN_PROGRESS succeeds", func(t *testing.T) {
		svc, repo, auditRepo := newTestAttendanceService()
		ctx, claims := attendanceTestContext()
		id := uuid.New()

		repo.On("FindByID", mock.Anything, id, claims.CampusID).
			Return(&domain.Attendance{ID: id, Status: domain.AttendanceStatusScheduled}, nil)
		repo.On("Transition", mock.Anything, mock.AnythingOfType("domain.AttendanceTransition")).
			Return(&domain.Attendance{ID: id, Status: domain.AttendanceStatusInProgress}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		result, err := svc.TransitionAttendance(ctx, id, TransitionAttendanceInput{
			To: domain.AttendanceStatusInProgress,
		})
		require.NoError(t, err)
		assert.Equal(t, domain.AttendanceStatusInProgress, result.Status)
	})

	t.Run("IN_PROGRESS to COMPLETED succeeds", func(t *testing.T) {
		svc, repo, auditRepo := newTestAttendanceService()
		ctx, claims := attendanceTestContext()
		id := uuid.New()

		repo.On("FindByID", mock.Anything, id, claims.CampusID).
			Return(&domain.Attendance{ID: id, Status: domain.AttendanceStatusInProgress}, nil)
		repo.On("Transition", mock.Anything, mock.AnythingOfType("domain.AttendanceTransition")).
			Return(&domain.Attendance{ID: id, Status: domain.AttendanceStatusCompleted}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		_, err := svc.TransitionAttendance(ctx, id, TransitionAttendanceInput{
			To: domain.AttendanceStatusCompleted,
		})
		require.NoError(t, err)
	})

	t.Run("rejects COMPLETED to IN_PROGRESS", func(t *testing.T) {
		svc, repo, _ := newTestAttendanceService()
		ctx, claims := attendanceTestContext()
		id := uuid.New()

		repo.On("FindByID", mock.Anything, id, claims.CampusID).
			Return(&domain.Attendance{ID: id, Status: domain.AttendanceStatusCompleted}, nil)

		_, err := svc.TransitionAttendance(ctx, id, TransitionAttendanceInput{
			To: domain.AttendanceStatusInProgress,
		})
		require.ErrorIs(t, err, domain.ErrInvalidTransition)
	})

	t.Run("rejects SCHEDULED directly to COMPLETED", func(t *testing.T) {
		svc, repo, _ := newTestAttendanceService()
		ctx, claims := attendanceTestContext()
		id := uuid.New()

		repo.On("FindByID", mock.Anything, id, claims.CampusID).
			Return(&domain.Attendance{ID: id, Status: domain.AttendanceStatusScheduled}, nil)

		_, err := svc.TransitionAttendance(ctx, id, TransitionAttendanceInput{
			To: domain.AttendanceStatusCompleted,
		})
		require.ErrorIs(t, err, domain.ErrInvalidTransition)
	})

	t.Run("not found", func(t *testing.T) {
		svc, repo, _ := newTestAttendanceService()
		ctx, claims := attendanceTestContext()
		id := uuid.New()
		repo.On("FindByID", mock.Anything, id, claims.CampusID).Return(nil, domain.ErrNotFound)

		_, err := svc.TransitionAttendance(ctx, id, TransitionAttendanceInput{
			To: domain.AttendanceStatusInProgress,
		})
		require.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("validation rejects invalid status", func(t *testing.T) {
		svc, _, _ := newTestAttendanceService()
		ctx, _ := attendanceTestContext()

		_, err := svc.TransitionAttendance(ctx, uuid.New(), TransitionAttendanceInput{To: "FOLLOW_UP"})
		require.Error(t, err)
	})
}

func TestAttendanceService_GetAttendance(t *testing.T) {
	t.Run("success with transitions", func(t *testing.T) {
		svc, repo, _ := newTestAttendanceService()
		ctx, claims := attendanceTestContext()
		id := uuid.New()

		repo.On("FindByIDWithTransitions", mock.Anything, id, claims.CampusID).
			Return(&domain.AttendanceDetail{
				Attendance:  domain.Attendance{ID: id},
				Transitions: []domain.AttendanceTransition{{ID: uuid.New()}},
			}, nil)

		result, err := svc.GetAttendance(ctx, id)
		require.NoError(t, err)
		assert.Len(t, result.Transitions, 1)
	})

	t.Run("not found", func(t *testing.T) {
		svc, repo, _ := newTestAttendanceService()
		ctx, claims := attendanceTestContext()
		id := uuid.New()
		repo.On("FindByIDWithTransitions", mock.Anything, id, claims.CampusID).Return(nil, domain.ErrNotFound)
		_, err := svc.GetAttendance(ctx, id)
		require.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("forbidden", func(t *testing.T) {
		svc, _, _ := newTestAttendanceService()
		ctx := auth.NewContext(context.Background(), auth.AuthClaims{Subject: uuid.New().String()})
		_, err := svc.GetAttendance(ctx, uuid.New())
		require.ErrorIs(t, err, domain.ErrForbidden)
	})
}

func TestAttendanceService_ListAttendances(t *testing.T) {
	t.Run("success applies campus scope", func(t *testing.T) {
		svc, repo, _ := newTestAttendanceService()
		ctx, claims := attendanceTestContext()

		repo.On("List", mock.Anything, mock.MatchedBy(func(f domain.AttendanceFilter) bool {
			return f.CampusID == claims.CampusID
		})).Return(&domain.AttendanceListResult{
			Pagination: domain.Pagination{Page: 1, PerPage: 20, Total: 0},
		}, nil)

		_, err := svc.ListAttendances(ctx, domain.AttendanceFilter{Page: 1, PerPage: 20})
		require.NoError(t, err)
	})

	t.Run("forbidden", func(t *testing.T) {
		svc, _, _ := newTestAttendanceService()
		ctx := auth.NewContext(context.Background(), auth.AuthClaims{Subject: uuid.New().String()})
		_, err := svc.ListAttendances(ctx, domain.AttendanceFilter{})
		require.ErrorIs(t, err, domain.ErrForbidden)
	})
}

func TestAttendanceService_UpdateNotes(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, repo, auditRepo := newTestAttendanceService()
		ctx, claims := attendanceTestContext()
		id := uuid.New()
		obs := "patient stable"

		repo.On("UpdateNotes", mock.Anything, id, claims.CampusID, &obs, (*string)(nil)).
			Return(&domain.Attendance{ID: id, Observations: &obs, UpdatedAt: time.Now()}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		result, err := svc.UpdateNotes(ctx, id, UpdateAttendanceNotesInput{Observations: &obs})
		require.NoError(t, err)
		require.NotNil(t, result.Observations)
		assert.Equal(t, "patient stable", *result.Observations)
	})

	t.Run("forbidden", func(t *testing.T) {
		svc, _, _ := newTestAttendanceService()
		ctx := auth.NewContext(context.Background(), auth.AuthClaims{Subject: uuid.New().String()})
		_, err := svc.UpdateNotes(ctx, uuid.New(), UpdateAttendanceNotesInput{})
		require.ErrorIs(t, err, domain.ErrForbidden)
	})
}
