package domain

import (
	"time"

	"github.com/google/uuid"
)

// PersonRole represents a role assignment for a person.
type PersonRole struct {
	ID                    uuid.UUID  `json:"id"`
	PersonID              uuid.UUID  `json:"person_id"`
	RoleType              string     `json:"role_type"`
	ProfessionalSpecialty *string    `json:"professional_specialty,omitempty"`
	IsActive              bool       `json:"is_active"`
	ActivatedAt           time.Time  `json:"activated_at"`
	DeactivatedAt         *time.Time `json:"deactivated_at,omitempty"`
	ActivatedBy           *uuid.UUID `json:"activated_by,omitempty"`
	DeactivatedBy         *uuid.UUID `json:"deactivated_by,omitempty"`
	Notes                 *string    `json:"notes,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
}
