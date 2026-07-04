package domain

import (
	"time"

	"github.com/google/uuid"
)

// Consent is an LGPD consent record scoped to a campus. Revoked rows are
// preserved as the registry trail (docs/13); re-granting creates a new row.
type Consent struct {
	ID              uuid.UUID  `json:"id"`
	PersonID        uuid.UUID  `json:"person_id"`
	ConsentType     string     `json:"consent_type"`
	ConsentVersion  string     `json:"consent_version"`
	Purpose         string     `json:"purpose"`
	GrantedAt       time.Time  `json:"granted_at"`
	GrantedByPerson *uuid.UUID `json:"granted_by_person_id"`
	SignatureData   *string    `json:"signature_data"`
	IsActive        bool       `json:"is_active"`
	RevokedAt       *time.Time `json:"revoked_at"`
	RevokedReason   *string    `json:"revoked_reason"`
	CampusID        uuid.UUID  `json:"campus_id"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ConsentTypes enumerates the valid LGPD consent categories.
var ConsentTypes = []string{"DATA_PROCESSING", "IMAGE_USAGE", "HEALTH_DATA", "MINOR_GUARDIAN"}
