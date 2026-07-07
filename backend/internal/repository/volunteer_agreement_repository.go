package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// VolunteerAgreementRepository handles volunteer_agreement persistence.
type VolunteerAgreementRepository struct {
	base
}

// NewVolunteerAgreementRepository creates a new VolunteerAgreementRepository.
func NewVolunteerAgreementRepository(pool Querier) *VolunteerAgreementRepository {
	return &VolunteerAgreementRepository{base: base{pool: pool}}
}

// Create inserts a new volunteer agreement record.
func (r *VolunteerAgreementRepository) Create(ctx context.Context, agreement domain.VolunteerAgreement) (*domain.VolunteerAgreement, error) {
	query := `
		INSERT INTO volunteer_agreement (
			id, person_id, person_role_id, campus_id, status,
			signature_method, agreement_version, notes
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at`

	err := r.q(ctx).QueryRow(ctx, query,
		agreement.ID, agreement.PersonID, agreement.PersonRoleID,
		agreement.CampusID, agreement.Status,
		agreement.SignatureMethod, agreement.AgreementVersion, agreement.Notes,
	).Scan(&agreement.CreatedAt, &agreement.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.ErrAgreementExists
		}
		return nil, fmt.Errorf("volunteerAgreementRepository.Create: %w", err)
	}

	return &agreement, nil
}

// FindByPersonID returns all agreements for a person.
func (r *VolunteerAgreementRepository) FindByPersonID(ctx context.Context, personID uuid.UUID) ([]domain.VolunteerAgreement, error) {
	query := `
		SELECT id, person_id, person_role_id, campus_id, status,
		       signature_method, accepted_at, accepted_by_user,
		       ip_address, user_agent, document_path, uploaded_at, uploaded_by,
		       rejected_at, rejection_reason, agreement_version, notes,
		       created_at, updated_at
		FROM volunteer_agreement
		WHERE person_id = $1
		ORDER BY created_at DESC`

	rows, err := r.q(ctx).Query(ctx, query, personID)
	if err != nil {
		return nil, fmt.Errorf("volunteerAgreementRepository.FindByPersonID: %w", err)
	}
	defer rows.Close()

	var agreements []domain.VolunteerAgreement
	for rows.Next() {
		a, err := scanAgreement(rows)
		if err != nil {
			return nil, fmt.Errorf("volunteerAgreementRepository.FindByPersonID: scan: %w", err)
		}
		agreements = append(agreements, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("volunteerAgreementRepository.FindByPersonID: rows: %w", err)
	}

	if agreements == nil {
		agreements = []domain.VolunteerAgreement{}
	}
	return agreements, nil
}

// FindByPersonRoleID returns the agreement for a specific person role.
func (r *VolunteerAgreementRepository) FindByPersonRoleID(ctx context.Context, personRoleID uuid.UUID) (*domain.VolunteerAgreement, error) {
	query := `
		SELECT id, person_id, person_role_id, campus_id, status,
		       signature_method, accepted_at, accepted_by_user,
		       ip_address, user_agent, document_path, uploaded_at, uploaded_by,
		       rejected_at, rejection_reason, agreement_version, notes,
		       created_at, updated_at
		FROM volunteer_agreement
		WHERE person_role_id = $1`

	row := r.q(ctx).QueryRow(ctx, query, personRoleID)
	a, err := scanAgreementRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("volunteerAgreementRepository.FindByPersonRoleID: %w", err)
	}

	return &a, nil
}

// FindPendingByPersonID returns the first PENDING agreement for a person.
func (r *VolunteerAgreementRepository) FindPendingByPersonID(ctx context.Context, personID uuid.UUID) (*domain.VolunteerAgreement, error) {
	query := `
		SELECT id, person_id, person_role_id, campus_id, status,
		       signature_method, accepted_at, accepted_by_user,
		       ip_address, user_agent, document_path, uploaded_at, uploaded_by,
		       rejected_at, rejection_reason, agreement_version, notes,
		       created_at, updated_at
		FROM volunteer_agreement
		WHERE person_id = $1 AND status = 'PENDING'
		LIMIT 1`

	row := r.q(ctx).QueryRow(ctx, query, personID)
	a, err := scanAgreementRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("volunteerAgreementRepository.FindPendingByPersonID: %w", err)
	}

	return &a, nil
}

// HasAcceptedAgreement checks if a person has at least one ACCEPTED agreement.
func (r *VolunteerAgreementRepository) HasAcceptedAgreement(ctx context.Context, personID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM volunteer_agreement
			WHERE person_id = $1 AND status = 'ACCEPTED'
		)`

	var exists bool
	err := r.q(ctx).QueryRow(ctx, query, personID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("volunteerAgreementRepository.HasAcceptedAgreement: %w", err)
	}

	return exists, nil
}

// AcceptDigital sets an agreement to ACCEPTED with digital signature metadata.
func (r *VolunteerAgreementRepository) AcceptDigital(ctx context.Context, id uuid.UUID, userID uuid.UUID, ip string, userAgent string) (*domain.VolunteerAgreement, error) {
	query := `
		UPDATE volunteer_agreement
		SET status = 'ACCEPTED',
		    signature_method = 'DIGITAL',
		    accepted_at = NOW(),
		    accepted_by_user = $2,
		    ip_address = $3::inet,
		    user_agent = $4,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, person_id, person_role_id, campus_id, status,
		          signature_method, accepted_at, accepted_by_user,
		          ip_address, user_agent, document_path, uploaded_at, uploaded_by,
		          rejected_at, rejection_reason, agreement_version, notes,
		          created_at, updated_at`

	row := r.q(ctx).QueryRow(ctx, query, id, userID, ip, userAgent)
	a, err := scanAgreementRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("volunteerAgreementRepository.AcceptDigital: %w", err)
	}

	return &a, nil
}

// Reject sets an agreement to REJECTED with optional reason.
func (r *VolunteerAgreementRepository) Reject(ctx context.Context, id uuid.UUID, reason *string) (*domain.VolunteerAgreement, error) {
	query := `
		UPDATE volunteer_agreement
		SET status = 'REJECTED',
		    rejected_at = NOW(),
		    rejection_reason = $2,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, person_id, person_role_id, campus_id, status,
		          signature_method, accepted_at, accepted_by_user,
		          ip_address, user_agent, document_path, uploaded_at, uploaded_by,
		          rejected_at, rejection_reason, agreement_version, notes,
		          created_at, updated_at`

	row := r.q(ctx).QueryRow(ctx, query, id, reason)
	a, err := scanAgreementRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("volunteerAgreementRepository.Reject: %w", err)
	}

	return &a, nil
}

// AcceptManualUpload sets an agreement to ACCEPTED with manual upload metadata.
func (r *VolunteerAgreementRepository) AcceptManualUpload(ctx context.Context, id uuid.UUID, documentPath string, uploadedBy uuid.UUID) (*domain.VolunteerAgreement, error) {
	now := time.Now()
	query := `
		UPDATE volunteer_agreement
		SET status = 'ACCEPTED',
		    signature_method = 'MANUAL_UPLOAD',
		    document_path = $2,
		    uploaded_at = $3,
		    uploaded_by = $4,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, person_id, person_role_id, campus_id, status,
		          signature_method, accepted_at, accepted_by_user,
		          ip_address, user_agent, document_path, uploaded_at, uploaded_by,
		          rejected_at, rejection_reason, agreement_version, notes,
		          created_at, updated_at`

	row := r.q(ctx).QueryRow(ctx, query, id, documentPath, now, uploadedBy)
	a, err := scanAgreementRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("volunteerAgreementRepository.AcceptManualUpload: %w", err)
	}

	return &a, nil
}

// scanAgreement scans a volunteer_agreement row from pgx.Rows.
func scanAgreement(rows pgx.Rows) (domain.VolunteerAgreement, error) {
	var a domain.VolunteerAgreement
	err := rows.Scan(
		&a.ID, &a.PersonID, &a.PersonRoleID, &a.CampusID, &a.Status,
		&a.SignatureMethod, &a.AcceptedAt, &a.AcceptedByUser,
		&a.IPAddress, &a.UserAgent, &a.DocumentPath, &a.UploadedAt, &a.UploadedBy,
		&a.RejectedAt, &a.RejectionReason, &a.AgreementVersion, &a.Notes,
		&a.CreatedAt, &a.UpdatedAt,
	)
	return a, err
}

// scanAgreementRow scans a volunteer_agreement row from pgx.Row.
func scanAgreementRow(row pgx.Row) (domain.VolunteerAgreement, error) {
	var a domain.VolunteerAgreement
	err := row.Scan(
		&a.ID, &a.PersonID, &a.PersonRoleID, &a.CampusID, &a.Status,
		&a.SignatureMethod, &a.AcceptedAt, &a.AcceptedByUser,
		&a.IPAddress, &a.UserAgent, &a.DocumentPath, &a.UploadedAt, &a.UploadedBy,
		&a.RejectedAt, &a.RejectionReason, &a.AgreementVersion, &a.Notes,
		&a.CreatedAt, &a.UpdatedAt,
	)
	return a, err
}
