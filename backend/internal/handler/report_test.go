package handler

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockReportRepo struct {
	mock.Mock
}

func (m *mockReportRepo) BuildAttendanceReport(
	ctx context.Context,
	filter domain.AttendanceReportFilter,
) (*domain.AttendanceReport, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AttendanceReport), args.Error(1)
}

func (m *mockReportRepo) StreamAttendancesForCSV(
	ctx context.Context,
	filter domain.AttendanceReportFilter,
	emit func(domain.AttendanceCSVRow) error,
) error {
	args := m.Called(ctx, filter, emit)
	return args.Error(0)
}

func authedRequest(method, target string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	claims := auth.AuthClaims{
		Subject:  uuid.New().String(),
		Email:    "coord@chesed.test",
		Roles:    []string{"COORDINATOR"},
		CampusID: uuid.New(),
	}
	return req.WithContext(auth.NewContext(req.Context(), claims))
}

func TestReportHandler_AttendanceSummary(t *testing.T) {
	t.Run("returns 200 with summary payload", func(t *testing.T) {
		repo := new(mockReportRepo)
		svc := service.NewReportService(repo)
		h := NewReportHandler(svc)

		want := &domain.AttendanceReport{
			Period:           domain.ReportPeriod{},
			TotalAttendances: 12,
			UniquePersons:    10,
			ByStatus:         map[string]int{"COMPLETED": 8, "SCHEDULED": 4},
			ByServiceType:    []domain.ServiceTypeCount{{ServiceType: "LEGAL", Count: 6}},
			ByMonth:          []domain.MonthCount{{Month: "2026-01", Count: 12}},
		}
		repo.On("BuildAttendanceReport", mock.Anything, mock.Anything).Return(want, nil)

		req := authedRequest(http.MethodGet, "/api/v1/reports/attendances?start=2026-01-01&end=2026-01-31")
		rec := httptest.NewRecorder()
		h.AttendanceSummary(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		var got domain.AttendanceReport
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, 12, got.TotalAttendances)
		assert.Equal(t, 10, got.UniquePersons)
		assert.Equal(t, 8, got.ByStatus["COMPLETED"])
	})

	t.Run("returns 400 when start missing", func(t *testing.T) {
		h := NewReportHandler(service.NewReportService(new(mockReportRepo)))
		req := authedRequest(http.MethodGet, "/api/v1/reports/attendances?end=2026-01-31")
		rec := httptest.NewRecorder()
		h.AttendanceSummary(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("returns 400 on malformed start", func(t *testing.T) {
		h := NewReportHandler(service.NewReportService(new(mockReportRepo)))
		req := authedRequest(http.MethodGet, "/api/v1/reports/attendances?start=2026/01/01&end=2026-01-31")
		rec := httptest.NewRecorder()
		h.AttendanceSummary(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "invalid_start", body["error"])
	})

	t.Run("returns 400 when end before start", func(t *testing.T) {
		h := NewReportHandler(service.NewReportService(new(mockReportRepo)))
		req := authedRequest(http.MethodGet, "/api/v1/reports/attendances?start=2026-03-31&end=2026-01-01")
		rec := httptest.NewRecorder()
		h.AttendanceSummary(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("returns 400 when range too large", func(t *testing.T) {
		h := NewReportHandler(service.NewReportService(new(mockReportRepo)))
		req := authedRequest(http.MethodGet, "/api/v1/reports/attendances?start=2024-01-01&end=2026-12-31")
		rec := httptest.NewRecorder()
		h.AttendanceSummary(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "range_too_large", body["error"])
	})

	t.Run("returns 403 without campus context", func(t *testing.T) {
		repo := new(mockReportRepo)
		h := NewReportHandler(service.NewReportService(repo))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/attendances?start=2026-01-01&end=2026-01-31", nil)
		req = req.WithContext(auth.NewContext(req.Context(), auth.AuthClaims{Subject: uuid.New().String()}))
		rec := httptest.NewRecorder()
		h.AttendanceSummary(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("returns 500 when repository fails", func(t *testing.T) {
		repo := new(mockReportRepo)
		repo.On("BuildAttendanceReport", mock.Anything, mock.Anything).
			Return(nil, errors.New("db down"))
		h := NewReportHandler(service.NewReportService(repo))

		req := authedRequest(http.MethodGet, "/api/v1/reports/attendances?start=2026-01-01&end=2026-01-31")
		rec := httptest.NewRecorder()
		h.AttendanceSummary(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestReportHandler_AttendanceExport(t *testing.T) {
	t.Run("streams CSV with header and rows", func(t *testing.T) {
		repo := new(mockReportRepo)
		row := domain.AttendanceCSVRow{
			AttendanceID:     uuid.New(),
			AttendanceDate:   time.Date(2026, 1, 15, 9, 30, 0, 0, time.UTC),
			PersonName:       "Maria Silva",
			PersonDocument:   "12345678900",
			ServiceType:      "Consulta Medica",
			Status:           domain.AttendanceStatusCompleted,
			ProfessionalName: "Dr. Joao",
			CreatedAt:        time.Date(2026, 1, 10, 14, 0, 0, 0, time.UTC),
		}
		repo.On("StreamAttendancesForCSV", mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				emit := args.Get(2).(func(domain.AttendanceCSVRow) error)
				_ = emit(row)
			}).
			Return(nil)

		h := NewReportHandler(service.NewReportService(repo))

		req := authedRequest(http.MethodGet,
			"/api/v1/reports/attendances/export?start=2026-01-01&end=2026-01-31&format=csv")
		rec := httptest.NewRecorder()
		h.AttendanceExport(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/csv")
		assert.Contains(t, rec.Header().Get("Content-Disposition"), "attachment")
		assert.Contains(t, rec.Header().Get("Content-Disposition"), "attendances_2026-01-01_2026-01-31.csv")

		reader := csv.NewReader(strings.NewReader(rec.Body.String()))
		records, err := reader.ReadAll()
		require.NoError(t, err)
		require.Len(t, records, 2, "header + one row")
		assert.Equal(t, csvAttendanceHeader, records[0])
		assert.Equal(t, row.AttendanceID.String(), records[1][0])
		assert.Equal(t, "Maria Silva", records[1][2])
		assert.Equal(t, "12345678900", records[1][3])
		assert.Equal(t, "Consulta Medica", records[1][4])
		assert.Equal(t, domain.AttendanceStatusCompleted, records[1][5])
		assert.Equal(t, "Dr. Joao", records[1][6])
	})

	t.Run("returns 400 on unsupported format", func(t *testing.T) {
		h := NewReportHandler(service.NewReportService(new(mockReportRepo)))
		req := authedRequest(http.MethodGet,
			"/api/v1/reports/attendances/export?start=2026-01-01&end=2026-01-31&format=xlsx")
		rec := httptest.NewRecorder()
		h.AttendanceExport(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "invalid_format", body["error"])
	})

	t.Run("defaults format to csv when omitted", func(t *testing.T) {
		repo := new(mockReportRepo)
		repo.On("StreamAttendancesForCSV", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		h := NewReportHandler(service.NewReportService(repo))

		req := authedRequest(http.MethodGet,
			"/api/v1/reports/attendances/export?start=2026-01-01&end=2026-01-31")
		rec := httptest.NewRecorder()
		h.AttendanceExport(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/csv")
	})

	t.Run("returns 400 on missing range", func(t *testing.T) {
		h := NewReportHandler(service.NewReportService(new(mockReportRepo)))
		req := authedRequest(http.MethodGet, "/api/v1/reports/attendances/export?format=csv")
		rec := httptest.NewRecorder()
		h.AttendanceExport(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func (m *mockReportRepo) BuildCampaignMetrics(ctx context.Context, campaignID, campusID uuid.UUID) (*domain.CampaignMetrics, error) {
	args := m.Called(ctx, campaignID, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CampaignMetrics), args.Error(1)
}

func (m *mockReportRepo) BuildDashboard(ctx context.Context, campusID uuid.UUID) (*domain.DashboardMetrics, error) {
	args := m.Called(ctx, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.DashboardMetrics), args.Error(1)
}

func TestReportHandler_AttendanceSummary_Filters(t *testing.T) {
	t.Run("threads optional filters into the report filter", func(t *testing.T) {
		repo := new(mockReportRepo)
		svc := service.NewReportService(repo)
		h := NewReportHandler(svc)

		serviceTypeID := uuid.New()
		campaignID := uuid.New()
		professionalID := uuid.New()

		repo.On("BuildAttendanceReport", mock.Anything, mock.MatchedBy(func(f domain.AttendanceReportFilter) bool {
			return f.ServiceTypeID != nil && *f.ServiceTypeID == serviceTypeID &&
				f.CampaignID != nil && *f.CampaignID == campaignID &&
				f.ProfessionalID != nil && *f.ProfessionalID == professionalID
		})).Return(&domain.AttendanceReport{}, nil)

		target := "/api/v1/reports/attendances?start=2026-01-01&end=2026-01-31" +
			"&service_type_id=" + serviceTypeID.String() +
			"&campaign_id=" + campaignID.String() +
			"&professional_id=" + professionalID.String()
		req := authedRequest(http.MethodGet, target)
		rec := httptest.NewRecorder()
		h.AttendanceSummary(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		repo.AssertExpectations(t)
	})

	t.Run("returns 400 invalid_filter on malformed filter uuid", func(t *testing.T) {
		h := NewReportHandler(service.NewReportService(new(mockReportRepo)))
		req := authedRequest(http.MethodGet,
			"/api/v1/reports/attendances?start=2026-01-01&end=2026-01-31&professional_id=not-a-uuid")
		rec := httptest.NewRecorder()
		h.AttendanceSummary(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "invalid_filter", body["error"])
	})
}

func dashboardRouter(h *ReportHandler) http.Handler {
	r := chi.NewRouter()
	r.Get("/reports/dashboard", h.Dashboard)
	return r
}

func TestReportHandler_Dashboard(t *testing.T) {
	t.Run("returns 200 with dashboard payload", func(t *testing.T) {
		repo := new(mockReportRepo)
		h := NewReportHandler(service.NewReportService(repo))

		repo.On("BuildDashboard", mock.Anything, mock.Anything).Return(&domain.DashboardMetrics{
			TotalPersons:         512,
			AttendancesThisMonth: 84,
			UpcomingScheduled:    17,
			ActiveCampaigns:      3,
			AttendancesByStatus:  map[string]int{"COMPLETED": 640},
			RecentMonths:         []domain.MonthCount{{Month: "2026-07", Count: 84}},
		}, nil)

		req := authedRequest(http.MethodGet, "/reports/dashboard")
		rec := httptest.NewRecorder()
		dashboardRouter(h).ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var got domain.DashboardMetrics
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, 512, got.TotalPersons)
		assert.Equal(t, 3, got.ActiveCampaigns)
		assert.Equal(t, 640, got.AttendancesByStatus["COMPLETED"])
		require.Len(t, got.RecentMonths, 1)
	})

	t.Run("returns 403 without campus context", func(t *testing.T) {
		repo := new(mockReportRepo)
		h := NewReportHandler(service.NewReportService(repo))

		req := httptest.NewRequest(http.MethodGet, "/reports/dashboard", nil)
		req = req.WithContext(auth.NewContext(req.Context(), auth.AuthClaims{Subject: uuid.New().String()}))
		rec := httptest.NewRecorder()
		dashboardRouter(h).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("returns 500 when repository fails", func(t *testing.T) {
		repo := new(mockReportRepo)
		repo.On("BuildDashboard", mock.Anything, mock.Anything).Return(nil, errors.New("db down"))
		h := NewReportHandler(service.NewReportService(repo))

		req := authedRequest(http.MethodGet, "/reports/dashboard")
		rec := httptest.NewRecorder()
		dashboardRouter(h).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func campaignMetricsRouter(h *ReportHandler) http.Handler {
	r := chi.NewRouter()
	r.Get("/reports/campaigns/{id}", h.CampaignMetrics)
	return r
}

func TestReportHandler_CampaignMetrics(t *testing.T) {
	t.Run("returns 200 with metrics payload", func(t *testing.T) {
		repo := new(mockReportRepo)
		h := NewReportHandler(service.NewReportService(repo))

		campaignID := uuid.New()
		repo.On("BuildCampaignMetrics", mock.Anything, campaignID, mock.Anything).
			Return(&domain.CampaignMetrics{
				CampaignID:          campaignID,
				CampaignName:        "March Social Action",
				Status:              "ACTIVE",
				TriageCount:         42,
				AttendanceTotal:     61,
				AttendancesByStatus: map[string]int{"COMPLETED": 50, "SCHEDULED": 11},
				TeamSize:            12,
			}, nil)

		req := authedRequest(http.MethodGet, "/reports/campaigns/"+campaignID.String())
		rec := httptest.NewRecorder()
		campaignMetricsRouter(h).ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var got domain.CampaignMetrics
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, 42, got.TriageCount)
		assert.Equal(t, 12, got.TeamSize)
	})

	t.Run("404 when campaign not visible in campus", func(t *testing.T) {
		repo := new(mockReportRepo)
		h := NewReportHandler(service.NewReportService(repo))

		repo.On("BuildCampaignMetrics", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, domain.ErrNotFound)

		req := authedRequest(http.MethodGet, "/reports/campaigns/"+uuid.New().String())
		rec := httptest.NewRecorder()
		campaignMetricsRouter(h).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("400 on malformed id", func(t *testing.T) {
		repo := new(mockReportRepo)
		h := NewReportHandler(service.NewReportService(repo))

		req := authedRequest(http.MethodGet, "/reports/campaigns/not-a-uuid")
		rec := httptest.NewRecorder()
		campaignMetricsRouter(h).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("403 without campus context", func(t *testing.T) {
		repo := new(mockReportRepo)
		h := NewReportHandler(service.NewReportService(repo))

		req := httptest.NewRequest(http.MethodGet, "/reports/campaigns/"+uuid.New().String(), nil)
		rec := httptest.NewRecorder()
		campaignMetricsRouter(h).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}
