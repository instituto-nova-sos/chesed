package domain

import (
	"time"

	"github.com/google/uuid"
)

// Document is a file attachment linked to a person and optionally to an
// attendance, scoped to a campus. FilePath holds the object-storage key and
// must never be serialized — clients obtain content exclusively through
// time-limited presigned URLs (docs/19).
type Document struct {
	ID           uuid.UUID  `json:"id"`
	PersonID     uuid.UUID  `json:"person_id"`
	AttendanceID *uuid.UUID `json:"attendance_id"`
	CampusID     uuid.UUID  `json:"campus_id"`
	DocumentType string     `json:"document_type"`
	FileName     string     `json:"file_name"`
	FilePath     string     `json:"-"`
	FileSize     *int64     `json:"file_size"`
	MimeType     *string    `json:"mime_type"`
	Description  *string    `json:"description"`
	UploadedBy   uuid.UUID  `json:"-"`
	UploadedAt   time.Time  `json:"uploaded_at"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// DocumentDownload is the presigned-URL payload of GET /documents/:id/download.
type DocumentDownload struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// DocumentTypes enumerates the valid document categories.
var DocumentTypes = []string{"ID", "PROOF_OF_RESIDENCE", "MEDICAL_RECORD", "EXAM", "CONSENT", "PHOTO", "OTHER"}
