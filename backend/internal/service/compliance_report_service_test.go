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

// mockComplianceRepo implements ComplianceReportRepository for testing.
type mockComplianceRepo struct {
	mock.Mock
}

func (m *mockComplianceRepo) BuildComplianceReport(ctx context.Context, filter domain.ComplianceReportFilter) (*domain.ComplianceReport, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ComplianceReport), args.Error(1)
}

func complianceTestContext() (context.Context, auth.AuthClaims) {
	claims := auth.AuthClaims{
		Subject:  uuid.New().String(),
		Email:    "coord@chesed.test",
		Roles:    []string{"COORDINATOR"},
		CampusID: uuid.New(),
	}
	return auth.NewContext(context.Background(), claims), claims
}

func newComplianceService() (*ComplianceReportService, *mockComplianceRepo, *AuditService, *MockAuditRepository) {
	repo := new(mockComplianceRepo)
	auditRepo := new(MockAuditRepository)
	auditSvc := NewAuditService(auditRepo)
	return NewComplianceReportService(repo, auditSvc), repo, auditSvc, auditRepo
}

func TestComplianceReportService_GetReport(t *testing.T) {
	t.Run("success applies campus scope and forwards range", func(t *testing.T) {
		svc, repo, _, _ := newComplianceService()
		ctx, claims := complianceTestContext()
		start := mustParseDay(t, "2026-01-01")
		end := mustParseDay(t, "2026-03-31")

		want := &domain.ComplianceReport{
			Period:             domain.ReportPeriod{Start: start, End: end},
			ConsentsByType:     map[string]int{"DATA_PROCESSING": 10},
			ActiveConsents:     8,
			RevokedConsents:    2,
			AnonymizedSubjects: 1,
			DataSubjects:       40,
			DocumentsStored:    12,
		}
		repo.On("BuildComplianceReport", mock.Anything, mock.MatchedBy(func(f domain.ComplianceReportFilter) bool {
			return f.CampusID == claims.CampusID && f.Start.Equal(start) && f.End.Equal(end)
		})).Return(want, nil)

		got, err := svc.GetReport(ctx, start, end)
		require.NoError(t, err)
		assert.Equal(t, want, got)
		repo.AssertExpectations(t)
	})

	t.Run("forbidden when no campus in context", func(t *testing.T) {
		svc, _, _, _ := newComplianceService()
		ctx := auth.NewContext(context.Background(), auth.AuthClaims{Subject: uuid.New().String()})
		_, err := svc.GetReport(ctx, mustParseDay(t, "2026-01-01"), mustParseDay(t, "2026-01-31"))
		require.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		svc, repo, _, _ := newComplianceService()
		ctx, _ := complianceTestContext()
		boom := errors.New("db down")
		repo.On("BuildComplianceReport", mock.Anything, mock.Anything).Return(nil, boom)
		_, err := svc.GetReport(ctx, mustParseDay(t, "2026-01-01"), mustParseDay(t, "2026-01-31"))
		require.ErrorIs(t, err, boom)
	})
}

func TestComplianceReportService_ExportCSV(t *testing.T) {
	t.Run("emits metric rows and writes an EXPORT audit entry", func(t *testing.T) {
		svc, repo, _, auditRepo := newComplianceService()
		ctx, claims := complianceTestContext()
		start := mustParseDay(t, "2026-01-01")
		end := mustParseDay(t, "2026-03-31")

		report := &domain.ComplianceReport{
			Period:             domain.ReportPeriod{Start: start, End: end},
			ConsentsByType:     map[string]int{"DATA_PROCESSING": 10, "IMAGE_USAGE": 3},
			ActiveConsents:     11,
			RevokedConsents:    2,
			AnonymizedSubjects: 1,
			DataSubjects:       40,
			DocumentsStored:    12,
		}
		repo.On("BuildComplianceReport", mock.Anything, mock.MatchedBy(func(f domain.ComplianceReportFilter) bool {
			return f.CampusID == claims.CampusID
		})).Return(report, nil)

		auditRepo.On("Create", mock.Anything, mock.MatchedBy(func(entry domain.AuditLog) bool {
			return entry.ActionType == "EXPORT" && entry.EntityType == "compliance_report"
		})).Return(nil)

		var rows []domain.ComplianceCSVRow
		err := svc.ExportCSV(ctx, start, end, func(row domain.ComplianceCSVRow) error {
			rows = append(rows, row)
			return nil
		})
		require.NoError(t, err)
		require.NotEmpty(t, rows)

		// The rows must at least cover the scalar metrics plus per-type consent counts.
		metrics := map[string]int{}
		for _, r := range rows {
			metrics[r.Metric] = r.Value
		}
		assert.Equal(t, 11, metrics["active_consents"])
		assert.Equal(t, 2, metrics["revoked_consents"])
		assert.Equal(t, 1, metrics["anonymized_subjects"])
		assert.Equal(t, 40, metrics["data_subjects"])
		assert.Equal(t, 12, metrics["documents_stored"])
		assert.Equal(t, 10, metrics["consents_DATA_PROCESSING"])
		auditRepo.AssertExpectations(t)
	})

	t.Run("forbidden when no campus in context", func(t *testing.T) {
		svc, _, _, _ := newComplianceService()
		ctx := auth.NewContext(context.Background(), auth.AuthClaims{Subject: uuid.New().String()})
		err := svc.ExportCSV(ctx, mustParseDay(t, "2026-01-01"), mustParseDay(t, "2026-01-31"),
			func(domain.ComplianceCSVRow) error { return nil })
		require.ErrorIs(t, err, domain.ErrForbidden)
	})
}
