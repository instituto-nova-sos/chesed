package domain

import (
	"time"

	"github.com/google/uuid"
)

// Person represents a person registered in the system.
type Person struct {
	ID             uuid.UUID  `json:"id"`
	FullName       string     `json:"full_name"`
	BirthDate      *time.Time `json:"birth_date,omitempty"`
	DocumentType   string     `json:"document_type"`
	DocumentNumber *string    `json:"document_number,omitempty"`
	Gender         *string    `json:"gender,omitempty"`
	Email          *string    `json:"email,omitempty"`
	Phone          *string    `json:"phone,omitempty"`
	PhotoURL       *string    `json:"photo_url,omitempty"`
	ReferralSource *string    `json:"referral_source,omitempty"`
	Nationality    string     `json:"nationality"`
	CampusID       uuid.UUID  `json:"campus_id"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty"`
}

// PersonDetail includes person data with addresses and roles.
type PersonDetail struct {
	Person
	Addresses []Address    `json:"addresses"`
	Roles     []PersonRole `json:"roles"`
}

// PersonListItem is a summary representation for list responses.
type PersonListItem struct {
	ID             uuid.UUID `json:"id"`
	FullName       string    `json:"full_name"`
	DocumentNumber *string   `json:"document_number,omitempty"`
	Phone          *string   `json:"phone,omitempty"`
	Roles          []string  `json:"roles"`
	IsActive       bool      `json:"is_active"`
}

// PersonListResult contains paginated person list data.
type PersonListResult struct {
	Data       []PersonListItem `json:"data"`
	Pagination Pagination       `json:"pagination"`
}

// Pagination holds pagination metadata.
type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// PersonFilter defines search/filter criteria for listing persons.
type PersonFilter struct {
	Query           string
	CampusID        uuid.UUID
	Page            int
	PerPage         int
	AgreementStatus string // "with_agreement", "without_agreement", "rejected"
}

// DuplicateMatch represents a potential duplicate person.
type DuplicateMatch struct {
	ID             uuid.UUID `json:"id"`
	FullName       string    `json:"full_name"`
	DocumentNumber string    `json:"document_number"`
	Campus         string    `json:"campus"`
	MatchType      string    `json:"match_type"`
}

// DuplicateCheckResult is the response for duplicate detection.
type DuplicateCheckResult struct {
	HasDuplicates bool             `json:"has_duplicates"`
	Matches       []DuplicateMatch `json:"matches"`
}
