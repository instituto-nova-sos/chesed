package domain

import (
	"time"

	"github.com/google/uuid"
)

// ComplianceReportFilter is the input to compliance report queries. The period
// bounds consent activity; CampusID scopes every metric to the caller's campus.
type ComplianceReportFilter struct {
	CampusID uuid.UUID
	Start    time.Time
	End      time.Time
}

// ComplianceReport is the campus-scoped LGPD compliance snapshot returned by
// GET /reports/compliance. All counts are restricted to the caller's campus and
// the reporting period.
type ComplianceReport struct {
	Period             ReportPeriod   `json:"period"`
	ConsentsByType     map[string]int `json:"consents_by_type"`
	ActiveConsents     int            `json:"active_consents"`
	RevokedConsents    int            `json:"revoked_consents"`
	AnonymizedSubjects int            `json:"anonymized_subjects"`
	DataSubjects       int            `json:"data_subjects"`
	DocumentsStored    int            `json:"documents_stored"`
}

// ComplianceCSVRow is a single metric/value pair for the compliance CSV export.
type ComplianceCSVRow struct {
	Metric string
	Value  int
}
