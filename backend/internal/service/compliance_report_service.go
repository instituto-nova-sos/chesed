package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// ComplianceReportRepository defines persistence for LGPD compliance metrics.
type ComplianceReportRepository interface {
	BuildComplianceReport(ctx context.Context, filter domain.ComplianceReportFilter) (*domain.ComplianceReport, error)
}

// ComplianceReportService produces the campus-scoped LGPD compliance report and
// its CSV export. The audit service is used only to record the EXPORT action.
type ComplianceReportService struct {
	repo     ComplianceReportRepository
	auditSvc *AuditService
}

// NewComplianceReportService creates a ComplianceReportService.
func NewComplianceReportService(repo ComplianceReportRepository, auditSvc *AuditService) *ComplianceReportService {
	return &ComplianceReportService{repo: repo, auditSvc: auditSvc}
}

// GetReport returns the compliance report for the period scoped to the caller's
// campus. A missing campus in the token surfaces as ErrForbidden.
func (s *ComplianceReportService) GetReport(ctx context.Context, start, end time.Time) (*domain.ComplianceReport, error) {
	filter, err := s.buildFilter(ctx, start, end)
	if err != nil {
		return nil, fmt.Errorf("complianceReportService.GetReport: %w", err)
	}
	report, err := s.repo.BuildComplianceReport(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("complianceReportService.GetReport: %w", err)
	}
	return report, nil
}

// ExportCSV builds the report and invokes emit once per metric row, then records
// an EXPORT audit entry. emit lets the handler stream rows directly to the response.
func (s *ComplianceReportService) ExportCSV(
	ctx context.Context,
	start, end time.Time,
	emit func(domain.ComplianceCSVRow) error,
) error {
	report, err := s.GetReport(ctx, start, end)
	if err != nil {
		return fmt.Errorf("complianceReportService.ExportCSV: %w", err)
	}

	for _, row := range complianceCSVRows(report) {
		if err := emit(row); err != nil {
			return fmt.Errorf("complianceReportService.ExportCSV: emit: %w", err)
		}
	}

	// The EXPORT audit is required, not best-effort: an export of personal data
	// that leaves no audit record defeats the compliance report's own purpose, so
	// a failed audit fails the export (docs/adr/0001-audit-logging-durability.md).
	if err := s.auditSvc.LogRequired(ctx, AuditParams{
		ActionType:  "EXPORT",
		EntityType:  "compliance_report",
		Module:      "report",
		Description: "compliance report exported",
		Success:     true,
	}); err != nil {
		return fmt.Errorf("complianceReportService.ExportCSV: %w", err)
	}
	return nil
}

func (s *ComplianceReportService) buildFilter(ctx context.Context, start, end time.Time) (domain.ComplianceReportFilter, error) {
	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return domain.ComplianceReportFilter{}, domain.ErrForbidden
	}
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return domain.ComplianceReportFilter{}, ErrInvalidReportRange
	}
	return domain.ComplianceReportFilter{CampusID: campusID, Start: start, End: end}, nil
}

// complianceCSVRows flattens the report into a deterministic ordered slice of
// metric/value rows. Per-type consent counts are emitted last, sorted by type.
func complianceCSVRows(report *domain.ComplianceReport) []domain.ComplianceCSVRow {
	rows := []domain.ComplianceCSVRow{
		{Metric: "active_consents", Value: report.ActiveConsents},
		{Metric: "revoked_consents", Value: report.RevokedConsents},
		{Metric: "anonymized_subjects", Value: report.AnonymizedSubjects},
		{Metric: "data_subjects", Value: report.DataSubjects},
		{Metric: "documents_stored", Value: report.DocumentsStored},
	}

	types := make([]string, 0, len(report.ConsentsByType))
	for consentType := range report.ConsentsByType {
		types = append(types, consentType)
	}
	sort.Strings(types)
	for _, consentType := range types {
		rows = append(rows, domain.ComplianceCSVRow{
			Metric: "consents_" + consentType,
			Value:  report.ConsentsByType[consentType],
		})
	}
	return rows
}
