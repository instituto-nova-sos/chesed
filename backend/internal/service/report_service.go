package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// ErrInvalidReportRange is returned when the supplied period is invalid.
var ErrInvalidReportRange = errors.New("invalid report range")

// ReportRepository defines persistence operations for reporting queries.
type ReportRepository interface {
	BuildAttendanceReport(ctx context.Context, filter domain.AttendanceReportFilter) (*domain.AttendanceReport, error)
	StreamAttendancesForCSV(
		ctx context.Context,
		filter domain.AttendanceReportFilter,
		emit func(domain.AttendanceCSVRow) error,
	) error
	BuildCampaignMetrics(ctx context.Context, campaignID, campusID uuid.UUID) (*domain.CampaignMetrics, error)
	BuildDashboard(ctx context.Context, campusID uuid.UUID) (*domain.DashboardMetrics, error)
}

// AttendanceReportOptions carries the optional filters for the attendance
// summary endpoint. A nil field means the filter is not applied.
type AttendanceReportOptions struct {
	ServiceTypeID  *uuid.UUID
	CampaignID     *uuid.UUID
	ProfessionalID *uuid.UUID
}

// ReportService produces attendance summary reports and CSV exports.
type ReportService struct {
	repo ReportRepository
}

// NewReportService creates a ReportService.
func NewReportService(repo ReportRepository) *ReportService {
	return &ReportService{repo: repo}
}

// GetAttendanceReport returns the aggregated attendance report for the period
// constrained to the caller's campus. start and end are inclusive day boundaries.
// opts narrows the campus-scoped result set by service type, campaign, and/or
// acting professional.
func (s *ReportService) GetAttendanceReport(
	ctx context.Context,
	start, end time.Time,
	opts AttendanceReportOptions,
) (*domain.AttendanceReport, error) {
	filter, err := buildReportFilter(ctx, start, end)
	if err != nil {
		return nil, fmt.Errorf("reportService.GetAttendanceReport: %w", err)
	}
	filter.ServiceTypeID = opts.ServiceTypeID
	filter.CampaignID = opts.CampaignID
	filter.ProfessionalID = opts.ProfessionalID

	report, err := s.repo.BuildAttendanceReport(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("reportService.GetAttendanceReport: %w", err)
	}
	return report, nil
}

// GetDashboard returns the campus-scoped operational snapshot for the caller.
// A missing campus in the token surfaces as ErrForbidden.
func (s *ReportService) GetDashboard(ctx context.Context) (*domain.DashboardMetrics, error) {
	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return nil, fmt.Errorf("reportService.GetDashboard: %w", domain.ErrForbidden)
	}

	metrics, err := s.repo.BuildDashboard(ctx, campusID)
	if err != nil {
		return nil, fmt.Errorf("reportService.GetDashboard: %w", err)
	}
	return metrics, nil
}

// StreamAttendanceCSV invokes emit once per attendance row in the period,
// allowing the handler to stream rows directly to the HTTP response.
func (s *ReportService) StreamAttendanceCSV(
	ctx context.Context,
	start, end time.Time,
	emit func(domain.AttendanceCSVRow) error,
) error {
	filter, err := buildReportFilter(ctx, start, end)
	if err != nil {
		return fmt.Errorf("reportService.StreamAttendanceCSV: %w", err)
	}

	if err := s.repo.StreamAttendancesForCSV(ctx, filter, emit); err != nil {
		return fmt.Errorf("reportService.StreamAttendanceCSV: %w", err)
	}
	return nil
}

func buildReportFilter(ctx context.Context, start, end time.Time) (domain.AttendanceReportFilter, error) {
	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return domain.AttendanceReportFilter{}, domain.ErrForbidden
	}
	if start.IsZero() || end.IsZero() {
		return domain.AttendanceReportFilter{}, ErrInvalidReportRange
	}
	if end.Before(start) {
		return domain.AttendanceReportFilter{}, ErrInvalidReportRange
	}
	return domain.AttendanceReportFilter{
		CampusID: campusID,
		Start:    start,
		End:      end,
	}, nil
}

// GetCampaignMetrics returns per-campaign counters scoped to the caller's
// campus. A campaign outside the campus surfaces as ErrNotFound.
func (s *ReportService) GetCampaignMetrics(ctx context.Context, campaignID uuid.UUID) (*domain.CampaignMetrics, error) {
	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return nil, fmt.Errorf("reportService.GetCampaignMetrics: %w", domain.ErrForbidden)
	}

	metrics, err := s.repo.BuildCampaignMetrics(ctx, campaignID, campusID)
	if err != nil {
		return nil, fmt.Errorf("reportService.GetCampaignMetrics: %w", err)
	}
	return metrics, nil
}
