package domain

import (
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
)

// AttendanceStatus enumerates valid attendance statuses for Phase 1.
const (
	AttendanceStatusScheduled  = "SCHEDULED"
	AttendanceStatusInProgress = "IN_PROGRESS"
	AttendanceStatusCompleted  = "COMPLETED"
	AttendanceStatusCancelled  = "CANCELLED"
)

// ErrInvalidTransition is returned when an attendance state change is not allowed.
var ErrInvalidTransition = errors.New("invalid attendance transition")

// ValidAttendanceTransitions defines the allowed Phase 1 state transitions.
// FOLLOW_UP is reserved for Phase 2 and is not enabled here.
var ValidAttendanceTransitions = map[string][]string{
	AttendanceStatusScheduled:  {AttendanceStatusInProgress, AttendanceStatusCancelled},
	AttendanceStatusInProgress: {AttendanceStatusCompleted, AttendanceStatusCancelled},
	AttendanceStatusCompleted:  {},
	AttendanceStatusCancelled:  {},
}

// CanTransition returns true if from→to is a valid Phase 1 attendance transition.
func CanTransition(from, to string) bool {
	allowed, ok := ValidAttendanceTransitions[from]
	if !ok {
		return false
	}
	return slices.Contains(allowed, to)
}

// Attendance represents a service delivery to an assisted person.
type Attendance struct {
	ID              uuid.UUID  `json:"id"`
	PersonID        uuid.UUID  `json:"person_id"`
	TriageID        *uuid.UUID `json:"triage_id,omitempty"`
	CampusID        uuid.UUID  `json:"campus_id"`
	ServiceTypeID   uuid.UUID  `json:"service_type_id"`
	ProfessionalID  uuid.UUID  `json:"professional_id"`
	Status          string     `json:"status"`
	AttendanceDate  time.Time  `json:"attendance_date"`
	Observations    *string    `json:"observations,omitempty"`
	Recommendations *string    `json:"recommendations,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	CreatedBy       *uuid.UUID `json:"created_by,omitempty"`
}

// AttendanceTransition records a state change for an attendance.
type AttendanceTransition struct {
	ID             uuid.UUID `json:"id"`
	AttendanceID   uuid.UUID `json:"attendance_id"`
	FromStatus     string    `json:"from_status"`
	ToStatus       string    `json:"to_status"`
	Reason         *string   `json:"reason,omitempty"`
	TransitionedBy uuid.UUID `json:"transitioned_by"`
	TransitionedAt time.Time `json:"transitioned_at"`
}

// AttendanceDetail bundles an attendance with its transition history.
type AttendanceDetail struct {
	Attendance
	Transitions []AttendanceTransition `json:"transitions"`
}

// AttendanceListItem is a summary representation for list responses.
type AttendanceListItem struct {
	ID             uuid.UUID `json:"id"`
	PersonID       uuid.UUID `json:"person_id"`
	PersonName     string    `json:"person_name"`
	ServiceType    string    `json:"service_type"`
	Status         string    `json:"status"`
	AttendanceDate time.Time `json:"attendance_date"`
}

// AttendanceListResult contains paginated attendance list data.
type AttendanceListResult struct {
	Data       []AttendanceListItem `json:"data"`
	Pagination Pagination           `json:"pagination"`
}

// AttendanceFilter defines search/filter criteria for listing attendances.
type AttendanceFilter struct {
	CampusID uuid.UUID
	PersonID *uuid.UUID
	Status   *string
	From     *time.Time
	To       *time.Time
	Page     int
	PerPage  int
}
