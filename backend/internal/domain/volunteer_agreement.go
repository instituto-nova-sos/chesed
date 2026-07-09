package domain

import (
	"net"
	"time"

	"github.com/google/uuid"
)

// Agreement status constants.
const (
	AgreementPending  = "PENDING"
	AgreementAccepted = "ACCEPTED"
	AgreementRejected = "REJECTED"
)

// Signature method constants.
const (
	SignatureDigital      = "DIGITAL"
	SignatureManualUpload = "MANUAL_UPLOAD"
)

// VolunteerAgreement tracks a volunteer's agreement acceptance or rejection.
type VolunteerAgreement struct {
	ID               uuid.UUID  `json:"id"`
	PersonID         uuid.UUID  `json:"person_id"`
	PersonRoleID     uuid.UUID  `json:"person_role_id"`
	CampusID         uuid.UUID  `json:"campus_id"`
	Status           string     `json:"status"`
	SignatureMethod  *string    `json:"signature_method,omitempty"`
	AcceptedAt       *time.Time `json:"accepted_at,omitempty"`
	AcceptedByUser   *uuid.UUID `json:"accepted_by_user,omitempty"`
	IPAddress        net.IP     `json:"ip_address,omitempty"`
	UserAgent        *string    `json:"user_agent,omitempty"`
	SignatureData    *string    `json:"signature_data,omitempty"`
	DocumentPath     *string    `json:"document_path,omitempty"`
	UploadedAt       *time.Time `json:"uploaded_at,omitempty"`
	UploadedBy       *uuid.UUID `json:"uploaded_by,omitempty"`
	RejectedAt       *time.Time `json:"rejected_at,omitempty"`
	RejectionReason  *string    `json:"rejection_reason,omitempty"`
	AgreementVersion string     `json:"agreement_version"`
	Notes            *string    `json:"notes,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
