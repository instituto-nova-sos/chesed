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

// AttendanceReport is the aggregated response for /reports/attendances.
type AttendanceReport struct {
	Period           ReportPeriod       `json:"period"`
	TotalAttendances int                `json:"total_attendances"`
	UniquePersons    int                `json:"unique_persons"`
	ByStatus         map[string]int     `json:"by_status"`
	ByServiceType    []ServiceTypeCount `json:"by_service_type"`
	ByMonth          []MonthCount       `json:"by_month"`
}

// AttendanceReportFilter is the input to attendance report queries.
type AttendanceReportFilter struct {
	CampusID uuid.UUID
	Start    time.Time
	End      time.Time
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
