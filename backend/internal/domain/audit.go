package domain

import (
	"time"

	"github.com/google/uuid"
)

// AuditLog represents an immutable audit trail entry.
type AuditLog struct {
	ID          uuid.UUID  `json:"id"`
	UserID      *uuid.UUID `json:"user_id,omitempty"`
	ActionType  string     `json:"action_type"`
	EntityType  string     `json:"entity_type"`
	EntityID    *uuid.UUID `json:"entity_id,omitempty"`
	Module      *string    `json:"module,omitempty"`
	Description *string    `json:"description,omitempty"`
	OldValues   []byte     `json:"old_values,omitempty"`
	NewValues   []byte     `json:"new_values,omitempty"`
	IPAddress   *string    `json:"ip_address,omitempty"`
	UserAgent   *string    `json:"user_agent,omitempty"`
	CampusID    *uuid.UUID `json:"campus_id,omitempty"`
	Success     bool       `json:"success"`
	Timestamp   time.Time  `json:"timestamp"`
}
