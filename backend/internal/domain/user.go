package domain

import (
	"time"

	"github.com/google/uuid"
)

// AppUser represents a local projection of a Keycloak identity.
type AppUser struct {
	ID                uuid.UUID  `json:"id"`
	PersonID          *uuid.UUID `json:"person_id,omitempty"`
	Email             string     `json:"email"`
	KeycloakSubjectID string     `json:"-"`
	AccessProfile     string     `json:"access_profile"`
	CampusID          uuid.UUID  `json:"campus_id"`
	IsActive          bool       `json:"is_active"`
	LastLogin         *time.Time `json:"last_login,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}
