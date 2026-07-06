package handler

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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

type noopAuditRepo struct{}

func (noopAuditRepo) Create(context.Context, domain.AuditLog) error { return nil }

func newComplianceHandler(repo *mockComplianceRepo) *ComplianceReportHandler {
	auditSvc := service.NewAuditService(noopAuditRepo{})
	svc := service.NewComplianceReportService(repo, auditSvc)
	return NewComplianceReportHandler(svc)
}

func TestComplianceReportHandler_Report(t *testing.T) {
	t.Run("returns 200 with the compliance payload", func(t *testing.T) {
		repo := new(mockComplianceRepo)
		want := &domain.ComplianceReport{
			ConsentsByType:     map[string]int{"DATA_PROCESSING": 10},
			ActiveConsents:     8,
			RevokedConsents:    2,
			AnonymizedSubjects: 1,
			DataSubjects:       40,
			DocumentsStored:    12,
		}
		repo.On("BuildComplianceReport", mock.Anything, mock.Anything).Return(want, nil)
		h := newComplianceHandler(repo)

		req := authedRequest(http.MethodGet, "/api/v1/reports/compliance?start=2026-01-01&end=2026-03-31")
		rec := httptest.NewRecorder()
		h.Report(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var got domain.ComplianceReport
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, 8, got.ActiveConsents)
		assert.Equal(t, 40, got.DataSubjects)
		assert.Equal(t, 10, got.ConsentsByType["DATA_PROCESSING"])
	})

	t.Run("returns 400 on malformed range", func(t *testing.T) {
		h := newComplianceHandler(new(mockComplianceRepo))
		req := authedRequest(http.MethodGet, "/api/v1/reports/compliance?start=2026-03-31&end=2026-01-01")
		rec := httptest.NewRecorder()
		h.Report(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("returns 403 without campus context", func(t *testing.T) {
		repo := new(mockComplianceRepo)
		h := newComplianceHandler(repo)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/compliance?start=2026-01-01&end=2026-01-31", nil)
		req = req.WithContext(auth.NewContext(req.Context(), auth.AuthClaims{Subject: uuid.New().String()}))
		rec := httptest.NewRecorder()
		h.Report(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestComplianceReportHandler_Export(t *testing.T) {
	t.Run("streams CSV metric rows", func(t *testing.T) {
		repo := new(mockComplianceRepo)
		repo.On("BuildComplianceReport", mock.Anything, mock.Anything).Return(&domain.ComplianceReport{
			ConsentsByType:  map[string]int{"DATA_PROCESSING": 10},
			ActiveConsents:  8,
			RevokedConsents: 2,
			DataSubjects:    40,
			DocumentsStored: 12,
		}, nil)
		h := newComplianceHandler(repo)

		req := authedRequest(http.MethodGet,
			"/api/v1/reports/compliance/export?format=csv&start=2026-01-01&end=2026-03-31")
		rec := httptest.NewRecorder()
		h.Export(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/csv")
		assert.Contains(t, rec.Header().Get("Content-Disposition"), "attachment")

		reader := csv.NewReader(strings.NewReader(rec.Body.String()))
		records, err := reader.ReadAll()
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(records), 2, "header + at least one metric row")
		assert.Equal(t, []string{"metric", "value"}, records[0])
	})

	t.Run("returns 400 on unsupported format", func(t *testing.T) {
		h := newComplianceHandler(new(mockComplianceRepo))
		req := authedRequest(http.MethodGet,
			"/api/v1/reports/compliance/export?format=xlsx&start=2026-01-01&end=2026-03-31")
		rec := httptest.NewRecorder()
		h.Export(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "invalid_format", body["error"])
	})
}
