package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockReportRepository implements ReportRepository for testing.
type MockReportRepository struct {
	mock.Mock
}

func (m *MockReportRepository) BuildCampaignMetrics(ctx context.Context, campaignID, campusID uuid.UUID) (*domain.CampaignMetrics, error) {
	args := m.Called(ctx, campaignID, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CampaignMetrics), args.Error(1)
}

func (m *MockReportRepository) BuildDashboard(ctx context.Context, campusID uuid.UUID) (*domain.DashboardMetrics, error) {
	args := m.Called(ctx, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.DashboardMetrics), args.Error(1)
}

func (m *MockReportRepository) BuildAttendanceReport(
	ctx context.Context,
	filter domain.AttendanceReportFilter,
) (*domain.AttendanceReport, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AttendanceReport), args.Error(1)
}

func (m *MockReportRepository) StreamAttendancesForCSV(
	ctx context.Context,
	filter domain.AttendanceReportFilter,
	emit func(domain.AttendanceCSVRow) error,
) error {
	args := m.Called(ctx, filter, emit)
	return args.Error(0)
}

func newTestReportService() (*ReportService, *MockReportRepository) {
	repo := new(MockReportRepository)
	return NewReportService(repo), repo
}

func reportTestContext() (context.Context, auth.AuthClaims) {
	claims := auth.AuthClaims{
		Subject:  uuid.New().String(),
		Email:    "coord@chesed.test",
		Roles:    []string{"COORDINATOR"},
		CampusID: uuid.New(),
	}
	return auth.NewContext(context.Background(), claims), claims
}

func mustParseDay(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", s)
	require.NoError(t, err)
	return parsed
}

func TestReportService_GetAttendanceReport(t *testing.T) {
	t.Run("success applies campus scope and forwards range", func(t *testing.T) {
		svc, repo := newTestReportService()
		ctx, claims := reportTestContext()
		start := mustParseDay(t, "2026-01-01")
		end := mustParseDay(t, "2026-03-31")

		want := &domain.AttendanceReport{
			Period:           domain.ReportPeriod{Start: start, End: end},
			TotalAttendances: 234,
			UniquePersons:    187,
			ByStatus:         map[string]int{"COMPLETED": 198, "SCHEDULED": 10, "CANCELLED": 26},
			ByServiceType:    []domain.ServiceTypeCount{{ServiceType: "LEGAL", Count: 45}},
			ByMonth:          []domain.MonthCount{{Month: "2026-01", Count: 72}},
		}
		repo.On("BuildAttendanceReport", mock.Anything, mock.MatchedBy(func(f domain.AttendanceReportFilter) bool {
			return f.CampusID == claims.CampusID && f.Start.Equal(start) && f.End.Equal(end)
		})).Return(want, nil)

		got, err := svc.GetAttendanceReport(ctx, start, end, AttendanceReportOptions{})
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("threads optional filters into the repository", func(t *testing.T) {
		svc, repo := newTestReportService()
		ctx, claims := reportTestContext()
		start := mustParseDay(t, "2026-01-01")
		end := mustParseDay(t, "2026-03-31")

		serviceTypeID := uuid.New()
		campaignID := uuid.New()
		professionalID := uuid.New()
		opts := AttendanceReportOptions{
			ServiceTypeID:  &serviceTypeID,
			CampaignID:     &campaignID,
			ProfessionalID: &professionalID,
		}

		repo.On("BuildAttendanceReport", mock.Anything, mock.MatchedBy(func(f domain.AttendanceReportFilter) bool {
			return f.CampusID == claims.CampusID &&
				f.ServiceTypeID != nil && *f.ServiceTypeID == serviceTypeID &&
				f.CampaignID != nil && *f.CampaignID == campaignID &&
				f.ProfessionalID != nil && *f.ProfessionalID == professionalID
		})).Return(&domain.AttendanceReport{}, nil)

		_, err := svc.GetAttendanceReport(ctx, start, end, opts)
		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("forbidden when no campus in context", func(t *testing.T) {
		svc, _ := newTestReportService()
		ctx := auth.NewContext(context.Background(), auth.AuthClaims{Subject: uuid.New().String()})
		_, err := svc.GetAttendanceReport(ctx, mustParseDay(t, "2026-01-01"), mustParseDay(t, "2026-01-31"), AttendanceReportOptions{})
		require.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("rejects inverted range", func(t *testing.T) {
		svc, _ := newTestReportService()
		ctx, _ := reportTestContext()
		_, err := svc.GetAttendanceReport(ctx, mustParseDay(t, "2026-03-31"), mustParseDay(t, "2026-01-01"), AttendanceReportOptions{})
		require.ErrorIs(t, err, ErrInvalidReportRange)
	})

	t.Run("rejects zero range", func(t *testing.T) {
		svc, _ := newTestReportService()
		ctx, _ := reportTestContext()
		_, err := svc.GetAttendanceReport(ctx, time.Time{}, time.Time{}, AttendanceReportOptions{})
		require.ErrorIs(t, err, ErrInvalidReportRange)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		svc, repo := newTestReportService()
		ctx, _ := reportTestContext()
		boom := errors.New("db down")
		repo.On("BuildAttendanceReport", mock.Anything, mock.Anything).Return(nil, boom)

		_, err := svc.GetAttendanceReport(ctx, mustParseDay(t, "2026-01-01"), mustParseDay(t, "2026-01-31"), AttendanceReportOptions{})
		require.ErrorIs(t, err, boom)
	})
}

func TestReportService_GetDashboard(t *testing.T) {
	t.Run("success resolves campus and returns repository snapshot", func(t *testing.T) {
		svc, repo := newTestReportService()
		ctx, claims := reportTestContext()

		want := &domain.DashboardMetrics{
			TotalPersons:         512,
			AttendancesThisMonth: 84,
			UpcomingScheduled:    17,
			ActiveCampaigns:      3,
			AttendancesByStatus:  map[string]int{"COMPLETED": 640, "SCHEDULED": 40},
			RecentMonths:         []domain.MonthCount{{Month: "2026-07", Count: 84}},
		}
		repo.On("BuildDashboard", mock.Anything, claims.CampusID).Return(want, nil)

		got, err := svc.GetDashboard(ctx)
		require.NoError(t, err)
		assert.Equal(t, want, got)
		repo.AssertExpectations(t)
	})

	t.Run("forbidden when no campus in context", func(t *testing.T) {
		svc, _ := newTestReportService()
		ctx := auth.NewContext(context.Background(), auth.AuthClaims{Subject: uuid.New().String()})
		_, err := svc.GetDashboard(ctx)
		require.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		svc, repo := newTestReportService()
		ctx, _ := reportTestContext()
		boom := errors.New("db down")
		repo.On("BuildDashboard", mock.Anything, mock.Anything).Return(nil, boom)

		_, err := svc.GetDashboard(ctx)
		require.ErrorIs(t, err, boom)
	})
}

func TestReportService_StreamAttendanceCSV(t *testing.T) {
	t.Run("emits rows through callback", func(t *testing.T) {
		svc, repo := newTestReportService()
		ctx, claims := reportTestContext()
		start := mustParseDay(t, "2026-01-01")
		end := mustParseDay(t, "2026-01-31")

		row1 := domain.AttendanceCSVRow{
			AttendanceID: uuid.New(), PersonName: "Maria",
			ServiceType: "LEGAL", Status: domain.AttendanceStatusCompleted,
		}
		row2 := domain.AttendanceCSVRow{
			AttendanceID: uuid.New(), PersonName: "Joao",
			ServiceType: "MEDICAL", Status: domain.AttendanceStatusScheduled,
		}

		repo.On("StreamAttendancesForCSV",
			mock.Anything,
			mock.MatchedBy(func(f domain.AttendanceReportFilter) bool {
				return f.CampusID == claims.CampusID && f.Start.Equal(start) && f.End.Equal(end)
			}),
			mock.AnythingOfType("func(domain.AttendanceCSVRow) error"),
		).Run(func(args mock.Arguments) {
			emit := args.Get(2).(func(domain.AttendanceCSVRow) error)
			_ = emit(row1)
			_ = emit(row2)
		}).Return(nil)

		var collected []domain.AttendanceCSVRow
		err := svc.StreamAttendanceCSV(ctx, start, end, func(r domain.AttendanceCSVRow) error {
			collected = append(collected, r)
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, []domain.AttendanceCSVRow{row1, row2}, collected)
	})

	t.Run("forbidden when no campus", func(t *testing.T) {
		svc, _ := newTestReportService()
		ctx := auth.NewContext(context.Background(), auth.AuthClaims{Subject: uuid.New().String()})
		err := svc.StreamAttendanceCSV(ctx, mustParseDay(t, "2026-01-01"), mustParseDay(t, "2026-01-31"),
			func(domain.AttendanceCSVRow) error { return nil })
		require.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("rejects inverted range", func(t *testing.T) {
		svc, _ := newTestReportService()
		ctx, _ := reportTestContext()
		err := svc.StreamAttendanceCSV(ctx, mustParseDay(t, "2026-12-31"), mustParseDay(t, "2026-01-01"),
			func(domain.AttendanceCSVRow) error { return nil })
		require.ErrorIs(t, err, ErrInvalidReportRange)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		svc, repo := newTestReportService()
		ctx, _ := reportTestContext()
		boom := errors.New("scan failed")
		repo.On("StreamAttendancesForCSV", mock.Anything, mock.Anything, mock.Anything).Return(boom)
		err := svc.StreamAttendanceCSV(ctx, mustParseDay(t, "2026-01-01"), mustParseDay(t, "2026-01-31"),
			func(domain.AttendanceCSVRow) error { return nil })
		require.ErrorIs(t, err, boom)
	})
}
