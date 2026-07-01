package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

const MaxSyncBatchSize = 50

const (
	SyncEntityPerson     = "person"
	SyncEntityTriage     = "triage"
	SyncEntityAttendance = "attendance"
)

const (
	SyncStatusCreated  = "created"
	SyncStatusConflict = "conflict"
	SyncStatusError    = "error"
)

var (
	ErrBatchTooLarge     = errors.New("sync batch exceeds maximum size")
	ErrInvalidEntityType = errors.New("invalid sync entity type")
	ErrMissingSyncID     = errors.New("sync record missing sync_id")
	ErrMissingData       = errors.New("sync record missing data")
)

// SyncPushRequest is the payload sent by clients to upload offline records.
type SyncPushRequest struct {
	DeviceID *uuid.UUID       `json:"device_id,omitempty"`
	Records  []SyncPushRecord `json:"records"`
}

// SyncPushRecord is a single offline-created record awaiting server persistence.
type SyncPushRecord struct {
	EntityType string         `json:"entity_type"`
	SyncID     uuid.UUID      `json:"sync_id"`
	Data       map[string]any `json:"data"`
	CreatedAt  *time.Time     `json:"created_at,omitempty"`
}

// SyncPushResult reports the outcome of a single push record.
type SyncPushResult struct {
	SyncID   uuid.UUID  `json:"sync_id"`
	Status   string     `json:"status"`
	ServerID *uuid.UUID `json:"server_id,omitempty"`
	Message  string     `json:"message,omitempty"`
}

// SyncPushResponse is returned to the client after a batch push.
type SyncPushResponse struct {
	Results         []SyncPushResult `json:"results"`
	ServerTimestamp time.Time        `json:"server_timestamp"`
}

// SyncPullRecord is one server record returned to a client during a delta pull.
type SyncPullRecord struct {
	EntityType string         `json:"entity_type"`
	ID         uuid.UUID      `json:"id"`
	Data       map[string]any `json:"data"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// SyncPullResponse is the server reply to a delta pull request.
type SyncPullResponse struct {
	Records         []SyncPullRecord `json:"records"`
	ServerTimestamp time.Time        `json:"server_timestamp"`
	HasMore         bool             `json:"has_more"`
	NextSince       *time.Time       `json:"next_since,omitempty"`
}

// Validate enforces the documented sync push contract.
func (r SyncPushRequest) Validate() error {
	if len(r.Records) > MaxSyncBatchSize {
		return ErrBatchTooLarge
	}
	for _, rec := range r.Records {
		if !isValidSyncEntity(rec.EntityType) {
			return ErrInvalidEntityType
		}
		if rec.SyncID == uuid.Nil {
			return ErrMissingSyncID
		}
		if len(rec.Data) == 0 {
			return ErrMissingData
		}
	}
	return nil
}

// ParseSyncEntityTypes parses a comma-separated entity_types query param.
// It rejects empty input, unknown types, and duplicates.
func ParseSyncEntityTypes(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, ErrInvalidEntityType
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if !isValidSyncEntity(trimmed) {
			return nil, ErrInvalidEntityType
		}
		if _, dup := seen[trimmed]; dup {
			return nil, ErrInvalidEntityType
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out, nil
}

func isValidSyncEntity(s string) bool {
	switch s {
	case SyncEntityPerson, SyncEntityTriage, SyncEntityAttendance:
		return true
	default:
		return false
	}
}
