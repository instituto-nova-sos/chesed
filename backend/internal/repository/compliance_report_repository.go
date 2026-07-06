package repository

import (
	"context"
	"fmt"

	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// ComplianceReportRepository assembles the campus-scoped LGPD compliance
// metrics. Reads run through the per-request RLS transaction (via base.q); every
// query also carries an explicit campus_id filter for clarity and because
// audit-derived tables (person) are additionally scoped here.
type ComplianceReportRepository struct {
	base
}

// NewComplianceReportRepository creates a ComplianceReportRepository.
func NewComplianceReportRepository(pool Querier) *ComplianceReportRepository {
	return &ComplianceReportRepository{base: base{pool: pool}}
}

// BuildComplianceReport aggregates consent posture, anonymization activity, and
// data-subject counts for the caller's campus over the reporting period.
func (r *ComplianceReportRepository) BuildComplianceReport(
	ctx context.Context,
	filter domain.ComplianceReportFilter,
) (*domain.ComplianceReport, error) {
	report := &domain.ComplianceReport{
		Period:         domain.ReportPeriod{Start: filter.Start, End: filter.End},
		ConsentsByType: map[string]int{},
	}

	if err := r.fetchConsentPosture(ctx, filter, report); err != nil {
		return nil, fmt.Errorf("complianceReportRepository.BuildComplianceReport: consents: %w", err)
	}
	if err := r.fetchSubjectCounts(ctx, filter, report); err != nil {
		return nil, fmt.Errorf("complianceReportRepository.BuildComplianceReport: subjects: %w", err)
	}
	if err := r.fetchDocumentCount(ctx, filter, report); err != nil {
		return nil, fmt.Errorf("complianceReportRepository.BuildComplianceReport: documents: %w", err)
	}

	return report, nil
}

// fetchConsentPosture populates the per-type consent breakdown and the
// active/revoked totals in a single grouped query, scoped to the campus and to
// consents granted within the period.
func (r *ComplianceReportRepository) fetchConsentPosture(
	ctx context.Context,
	f domain.ComplianceReportFilter,
	report *domain.ComplianceReport,
) error {
	const q = `
		SELECT consent_type,
		       COUNT(*)::int AS count,
		       COUNT(*) FILTER (WHERE is_active)::int AS active,
		       COUNT(*) FILTER (WHERE NOT is_active)::int AS revoked
		FROM consent
		WHERE campus_id = $1
		  AND granted_at >= $2
		  AND granted_at < ($3::date + INTERVAL '1 day')
		GROUP BY consent_type`
	rows, err := r.q(ctx).Query(ctx, q, f.CampusID, f.Start, f.End)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var consentType string
		var count, active, revoked int
		if err := rows.Scan(&consentType, &count, &active, &revoked); err != nil {
			return err
		}
		report.ConsentsByType[consentType] = count
		report.ActiveConsents += active
		report.RevokedConsents += revoked
	}
	return rows.Err()
}

// fetchSubjectCounts populates the anonymized-subject and total data-subject
// counts for the campus in one query.
func (r *ComplianceReportRepository) fetchSubjectCounts(
	ctx context.Context,
	f domain.ComplianceReportFilter,
	report *domain.ComplianceReport,
) error {
	const q = `
		SELECT COUNT(*) FILTER (WHERE anonymized_at IS NOT NULL)::int AS anonymized,
		       COUNT(*)::int AS total
		FROM person
		WHERE campus_id = $1`
	return r.q(ctx).QueryRow(ctx, q, f.CampusID).
		Scan(&report.AnonymizedSubjects, &report.DataSubjects)
}

// fetchDocumentCount populates the count of active stored documents for the campus.
func (r *ComplianceReportRepository) fetchDocumentCount(
	ctx context.Context,
	f domain.ComplianceReportFilter,
	report *domain.ComplianceReport,
) error {
	const q = `SELECT COUNT(*)::int FROM document WHERE campus_id = $1 AND is_active`
	return r.q(ctx).QueryRow(ctx, q, f.CampusID).Scan(&report.DocumentsStored)
}
