package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// presignedDownloadTTL is the lifetime of document download URLs (docs/11).
const presignedDownloadTTL = 15 * time.Minute

// extByContentType maps the sniffed (never client-declared) content type to
// the stored object-key extension — the whitelist of docs/19 §6.
var extByContentType = map[string]string{
	"application/pdf": ".pdf",
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
}

// DocumentRepository defines the interface for document metadata persistence.
type DocumentRepository interface {
	Create(ctx context.Context, d domain.Document) (*domain.Document, error)
	FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Document, error)
	ListByPerson(ctx context.Context, personID, campusID uuid.UUID) ([]domain.Document, error)
	ListByAttendance(ctx context.Context, attendanceID, campusID uuid.UUID) ([]domain.Document, error)
}

// DocumentPersonRepository is the campus-scoped person lookup documents need.
type DocumentPersonRepository interface {
	FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Person, error)
}

// DocumentAttendanceRepository is the campus-scoped attendance lookup used to
// resolve attendance-linked uploads.
type DocumentAttendanceRepository interface {
	FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Attendance, error)
}

// UploadDocumentInput carries a validated upload. ContentType is the
// magic-byte-sniffed type established at the HTTP boundary.
type UploadDocumentInput struct {
	DocumentType string  `validate:"required,oneof=ID PROOF_OF_RESIDENCE MEDICAL_RECORD EXAM CONSENT PHOTO OTHER"`
	Description  *string `validate:"omitempty"`
	FileName     string  `validate:"required,max=255"`
	ContentType  string  `validate:"required"`
	Size         int64   `validate:"required,gt=0"`
	Content      io.Reader
}

// DocumentService handles document business logic.
type DocumentService struct {
	repo           DocumentRepository
	personRepo     DocumentPersonRepository
	attendanceRepo DocumentAttendanceRepository
	storage        ObjectStorage
	auditSvc       *AuditService
}

// NewDocumentService creates a new DocumentService.
func NewDocumentService(repo DocumentRepository, personRepo DocumentPersonRepository, attendanceRepo DocumentAttendanceRepository, storage ObjectStorage, auditSvc *AuditService) *DocumentService {
	return &DocumentService{
		repo:           repo,
		personRepo:     personRepo,
		attendanceRepo: attendanceRepo,
		storage:        storage,
		auditSvc:       auditSvc,
	}
}

// UploadForPerson stores a document owned by a person in the caller's campus.
func (s *DocumentService) UploadForPerson(ctx context.Context, personID uuid.UUID, in UploadDocumentInput) (*domain.Document, error) {
	campusID, err := s.validateUpload(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("documentService.UploadForPerson: %w", err)
	}
	if _, err := s.personRepo.FindByID(ctx, personID, campusID); err != nil {
		return nil, fmt.Errorf("documentService.UploadForPerson: %w", err)
	}

	created, err := s.storeAndPersist(ctx, personID, nil, campusID, in)
	if err != nil {
		return nil, fmt.Errorf("documentService.UploadForPerson: %w", err)
	}
	return created, nil
}

// UploadForAttendance stores a document linked to an attendance (and its
// person) in the caller's campus.
func (s *DocumentService) UploadForAttendance(ctx context.Context, attendanceID uuid.UUID, in UploadDocumentInput) (*domain.Document, error) {
	campusID, err := s.validateUpload(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("documentService.UploadForAttendance: %w", err)
	}
	attendance, err := s.attendanceRepo.FindByID(ctx, attendanceID, campusID)
	if err != nil {
		return nil, fmt.Errorf("documentService.UploadForAttendance: %w", err)
	}

	created, err := s.storeAndPersist(ctx, attendance.PersonID, &attendanceID, campusID, in)
	if err != nil {
		return nil, fmt.Errorf("documentService.UploadForAttendance: %w", err)
	}
	return created, nil
}

// validateUpload runs struct validation, campus presence, and the content
// type whitelist (defense in depth behind the handler's sniff).
func (s *DocumentService) validateUpload(ctx context.Context, in UploadDocumentInput) (uuid.UUID, error) {
	if err := validate.Struct(in); err != nil {
		// Wrapped as ErrValidation so the handler maps it to 400 with a
		// generic message (no field echo, T3/T13).
		return uuid.Nil, fmt.Errorf("invalid upload input: %w", domain.ErrValidation)
	}
	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return uuid.Nil, domain.ErrForbidden
	}
	if _, ok := extByContentType[in.ContentType]; !ok {
		return uuid.Nil, fmt.Errorf("unsupported file content: %w", domain.ErrValidation)
	}
	return campusID, nil
}

// storeAndPersist writes the bytes to object storage under a UUID-based key,
// then persists the metadata row and audits the mutation. Storage comes
// first: a metadata row must never reference a missing object. If the insert
// fails after the Put, the orphan key is logged for out-of-band cleanup.
func (s *DocumentService) storeAndPersist(ctx context.Context, personID uuid.UUID, attendanceID *uuid.UUID, campusID uuid.UUID, in UploadDocumentInput) (*domain.Document, error) {
	key := fmt.Sprintf("documents/%s/%s/%s%s", campusID, personID, uuid.New(), extByContentType[in.ContentType])

	if err := s.storage.Put(ctx, key, in.ContentType, in.Size, in.Content); err != nil {
		return nil, fmt.Errorf("store object: %w", err)
	}

	uploadedBy := uuid.Nil
	if id := parseUserID(auth.ClaimsFromContext(ctx).Subject); id != nil {
		uploadedBy = *id
	}

	doc := domain.Document{
		ID:           uuid.New(),
		PersonID:     personID,
		AttendanceID: attendanceID,
		CampusID:     campusID,
		DocumentType: in.DocumentType,
		FileName:     in.FileName,
		FilePath:     key,
		FileSize:     &in.Size,
		MimeType:     &in.ContentType,
		Description:  in.Description,
		UploadedBy:   uploadedBy,
	}

	created, err := s.repo.Create(ctx, doc)
	if err != nil {
		slog.ErrorContext(ctx, "documentService: metadata insert failed after storage put; orphan object",
			"key", key, "error", err.Error(),
		)
		return nil, err
	}

	s.audit(ctx, AuditParams{
		ActionType:  "CREATE",
		EntityType:  "document",
		EntityID:    &created.ID,
		Module:      "document",
		Description: "document uploaded",
		NewValues: map[string]any{
			"person_id":     created.PersonID,
			"attendance_id": created.AttendanceID,
			"document_type": created.DocumentType,
			"file_size":     created.FileSize,
		},
		Success: true,
	})

	return created, nil
}

// ListByPerson returns a person's documents after resolving the person in the
// caller's campus.
func (s *DocumentService) ListByPerson(ctx context.Context, personID uuid.UUID) ([]domain.Document, error) {
	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return nil, fmt.Errorf("documentService.ListByPerson: %w", domain.ErrForbidden)
	}
	if _, err := s.personRepo.FindByID(ctx, personID, campusID); err != nil {
		return nil, fmt.Errorf("documentService.ListByPerson: %w", err)
	}
	docs, err := s.repo.ListByPerson(ctx, personID, campusID)
	if err != nil {
		return nil, fmt.Errorf("documentService.ListByPerson: %w", err)
	}
	return docs, nil
}

// ListByAttendance returns an attendance's documents after resolving the
// attendance in the caller's campus.
func (s *DocumentService) ListByAttendance(ctx context.Context, attendanceID uuid.UUID) ([]domain.Document, error) {
	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return nil, fmt.Errorf("documentService.ListByAttendance: %w", domain.ErrForbidden)
	}
	if _, err := s.attendanceRepo.FindByID(ctx, attendanceID, campusID); err != nil {
		return nil, fmt.Errorf("documentService.ListByAttendance: %w", err)
	}
	docs, err := s.repo.ListByAttendance(ctx, attendanceID, campusID)
	if err != nil {
		return nil, fmt.Errorf("documentService.ListByAttendance: %w", err)
	}
	return docs, nil
}

// GetDownloadURL returns a time-limited presigned URL for a document visible
// in the caller's campus. The API never streams stored bytes (docs/19).
func (s *DocumentService) GetDownloadURL(ctx context.Context, id uuid.UUID) (*domain.DocumentDownload, error) {
	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return nil, fmt.Errorf("documentService.GetDownloadURL: %w", domain.ErrForbidden)
	}
	doc, err := s.repo.FindByID(ctx, id, campusID)
	if err != nil {
		return nil, fmt.Errorf("documentService.GetDownloadURL: %w", err)
	}
	url, err := s.storage.PresignGet(ctx, doc.FilePath, presignedDownloadTTL)
	if err != nil {
		return nil, fmt.Errorf("documentService.GetDownloadURL: presign: %w", err)
	}
	return &domain.DocumentDownload{
		URL:       url,
		ExpiresAt: time.Now().UTC().Add(presignedDownloadTTL),
	}, nil
}

// audit writes an audit entry, logging (never failing the request) on error.
func (s *DocumentService) audit(ctx context.Context, params AuditParams) {
	if auditErr := s.auditSvc.LogAction(ctx, params); auditErr != nil {
		slog.ErrorContext(ctx, "documentService: audit failed",
			"error", auditErr.Error(), "entity_type", params.EntityType, "action", params.ActionType,
		)
	}
}
