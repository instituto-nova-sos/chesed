package domain

import (
	"encoding/json"
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

// AuditLogFilter holds the query criteria for the audit log viewer. CampusID is
// mandatory (applied at the SQL layer because audit_log is intentionally not
// under RLS); the remaining fields are optional and combine with AND.
type AuditLogFilter struct {
	CampusID   uuid.UUID
	UserID     *uuid.UUID
	EntityType *string
	ActionType *string
	Start      *time.Time
	End        *time.Time
	Page       int
	PerPage    int
}

// AuditLogListItem is a single row of the audit viewer, with the acting user's
// email resolved and the JSONB value columns passed through verbatim.
type AuditLogListItem struct {
	ID          uuid.UUID       `json:"id"`
	UserEmail   *string         `json:"user_email,omitempty"`
	ActionType  string          `json:"action_type"`
	EntityType  string          `json:"entity_type"`
	EntityID    *uuid.UUID      `json:"entity_id,omitempty"`
	Description *string         `json:"description,omitempty"`
	OldValues   json.RawMessage `json:"old_values,omitempty"`
	NewValues   json.RawMessage `json:"new_values,omitempty"`
	IPAddress   *string         `json:"ip_address,omitempty"`
	Timestamp   time.Time       `json:"timestamp"`
}

// AuditLogListResult is the paginated audit viewer response.
type AuditLogListResult struct {
	Data       []AuditLogListItem `json:"data"`
	Pagination Pagination         `json:"pagination"`
}
