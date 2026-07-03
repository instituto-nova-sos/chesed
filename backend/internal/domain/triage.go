package domain

import (
	"time"

	"github.com/google/uuid"
)

// Triage represents an initial intake assessment for an assisted person.
type Triage struct {
	ID             uuid.UUID   `json:"id"`
	PersonID       uuid.UUID   `json:"person_id"`
	CampaignID     *uuid.UUID  `json:"campaign_id,omitempty"`
	CampusID       uuid.UUID   `json:"campus_id"`
	MainComplaint  string      `json:"main_complaint"`
	AssignedTeam   *uuid.UUID  `json:"assigned_team,omitempty"`
	TriageDate     time.Time   `json:"triage_date"`
	Location       *string     `json:"location,omitempty"`
	TriagedBy      uuid.UUID   `json:"triaged_by"`
	Notes          *string     `json:"notes,omitempty"`
	IsActive       bool        `json:"is_active"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	RequestedTypes []uuid.UUID `json:"requested_service_types,omitempty"`
}

// TriageListItem is a summary representation for list responses.
type TriageListItem struct {
	ID             uuid.UUID `json:"id"`
	PersonID       uuid.UUID `json:"person_id"`
	PersonName     string    `json:"person_name"`
	MainComplaint  string    `json:"main_complaint"`
	TriageDate     time.Time `json:"triage_date"`
	RequestedCount int       `json:"requested_service_count"`
}

// TriageListResult contains paginated triage list data.
type TriageListResult struct {
	Data       []TriageListItem `json:"data"`
	Pagination Pagination       `json:"pagination"`
}

// TriageFilter defines search/filter criteria for listing triages.
type TriageFilter struct {
	CampusID uuid.UUID
	PersonID *uuid.UUID
	From     *time.Time
	To       *time.Time
	Page     int
	PerPage  int
}
