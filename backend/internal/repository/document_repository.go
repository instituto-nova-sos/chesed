package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/jackc/pgx/v5"
)

// DocumentRepository handles document metadata persistence.
type DocumentRepository struct {
	pool Querier
}

// NewDocumentRepository creates a new DocumentRepository.
func NewDocumentRepository(pool Querier) *DocumentRepository {
	return &DocumentRepository{pool: pool}
}

// documentColumns is the canonical column list; scanDocument must stay in
// lockstep with it (every SELECT/RETURNING site shares both).
const documentColumns = `id, person_id, attendance_id, campus_id, document_type, file_name,
	       file_path, file_size, mime_type, description, uploaded_by, uploaded_at,
	       is_active, created_at, updated_at`

// scanDocument reads one row in documentColumns order.
func scanDocument(row pgx.Row) (*domain.Document, error) {
	var d domain.Document
	if err := row.Scan(
		&d.ID, &d.PersonID, &d.AttendanceID, &d.CampusID, &d.DocumentType, &d.FileName,
		&d.FilePath, &d.FileSize, &d.MimeType, &d.Description, &d.UploadedBy, &d.UploadedAt,
		&d.IsActive, &d.CreatedAt, &d.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &d, nil
}

// Create inserts a document metadata row.
func (r *DocumentRepository) Create(ctx context.Context, d domain.Document) (*domain.Document, error) {
	const q = `
		INSERT INTO document (id, person_id, attendance_id, campus_id, document_type,
		                      file_name, file_path, file_size, mime_type, description, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING uploaded_at, is_active, created_at, updated_at`

	if err := r.pool.QueryRow(ctx, q,
		d.ID, d.PersonID, d.AttendanceID, d.CampusID, d.DocumentType,
		d.FileName, d.FilePath, d.FileSize, d.MimeType, d.Description, d.UploadedBy,
	).Scan(&d.UploadedAt, &d.IsActive, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return nil, fmt.Errorf("documentRepository.Create: %w", err)
	}
	return &d, nil
}

// FindByID returns a document by ID, scoped to campus.
func (r *DocumentRepository) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Document, error) {
	q := `SELECT ` + documentColumns + ` FROM document WHERE id = $1 AND campus_id = $2`

	d, err := scanDocument(r.pool.QueryRow(ctx, q, id, campusID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("documentRepository.FindByID: %w", err)
	}
	return d, nil
}

// ListByPerson returns a person's documents, newest upload first, scoped to
// campus.
func (r *DocumentRepository) ListByPerson(ctx context.Context, personID, campusID uuid.UUID) ([]domain.Document, error) {
	q := `SELECT ` + documentColumns + `
		FROM document
		WHERE person_id = $1 AND campus_id = $2
		ORDER BY uploaded_at DESC`

	return r.listDocuments(ctx, q, personID, campusID)
}

// ListByAttendance returns an attendance's documents, newest upload first,
// scoped to campus.
func (r *DocumentRepository) ListByAttendance(ctx context.Context, attendanceID, campusID uuid.UUID) ([]domain.Document, error) {
	q := `SELECT ` + documentColumns + `
		FROM document
		WHERE attendance_id = $1 AND campus_id = $2
		ORDER BY uploaded_at DESC`

	return r.listDocuments(ctx, q, attendanceID, campusID)
}

func (r *DocumentRepository) listDocuments(ctx context.Context, q string, args ...any) ([]domain.Document, error) {
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("documentRepository.list: %w", err)
	}
	defer rows.Close()

	docs := []domain.Document{}
	for rows.Next() {
		d, err := scanDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("documentRepository.list: scan: %w", err)
		}
		docs = append(docs, *d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("documentRepository.list: rows: %w", err)
	}
	return docs, nil
}
