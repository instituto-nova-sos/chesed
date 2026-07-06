package domain

import (
	"time"

	"github.com/google/uuid"
)

// ReportPeriod represents the inclusive date range applied to a report query.
type ReportPeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ServiceTypeCount is one row of the by_service_type aggregation.
type ServiceTypeCount struct {
	ServiceType string `json:"service_type"`
	Count       int    `json:"count"`
}

// MonthCount is one row of the by_month aggregation. Month uses YYYY-MM format.
type MonthCount struct {
	Month string `json:"month"`
	Count int    `json:"count"`
}

// ProfessionalCount is one row of the by_professional aggregation.
type ProfessionalCount struct {
	ProfessionalID   uuid.UUID `json:"professional_id"`
	ProfessionalName string    `json:"professional_name"`
	Count            int       `json:"count"`
}

// AttendanceReport is the aggregated response for /reports/attendances.
type AttendanceReport struct {
	Period           ReportPeriod        `json:"period"`
	TotalAttendances int                 `json:"total_attendances"`
	UniquePersons    int                 `json:"unique_persons"`
	ByStatus         map[string]int      `json:"by_status"`
	ByServiceType    []ServiceTypeCount  `json:"by_service_type"`
	ByMonth          []MonthCount        `json:"by_month"`
	ByProfessional   []ProfessionalCount `json:"by_professional"`
}

// AttendanceReportFilter is the input to attendance report queries. The optional
// pointer fields narrow the campus-scoped result set; nil means no filter.
type AttendanceReportFilter struct {
	CampusID       uuid.UUID
	Start          time.Time
	End            time.Time
	ServiceTypeID  *uuid.UUID
	CampaignID     *uuid.UUID
	ProfessionalID *uuid.UUID
}

// DashboardMetrics is the campus-scoped operational snapshot for
// GET /reports/dashboard. All windows are computed server-side.
type DashboardMetrics struct {
	TotalPersons         int            `json:"total_persons"`
	AttendancesThisMonth int            `json:"attendances_this_month"`
	UpcomingScheduled    int            `json:"upcoming_scheduled"`
	ActiveCampaigns      int            `json:"active_campaigns"`
	AttendancesByStatus  map[string]int `json:"attendances_by_status"`
	RecentMonths         []MonthCount   `json:"recent_months"`
}

// AttendanceCSVRow is a single denormalized attendance row for CSV export.
type AttendanceCSVRow struct {
	AttendanceID     uuid.UUID
	AttendanceDate   time.Time
	PersonName       string
	PersonDocument   string
	ServiceType      string
	Status           string
	ProfessionalName string
	CreatedAt        time.Time
}
