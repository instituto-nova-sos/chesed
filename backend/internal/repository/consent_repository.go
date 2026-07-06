package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ConsentRepository handles consent persistence.
type ConsentRepository struct {
	base
}

// NewConsentRepository creates a new ConsentRepository.
func NewConsentRepository(pool Querier) *ConsentRepository {
	return &ConsentRepository{base: base{pool: pool}}
}

// consentColumns is the canonical column list; scanConsent must stay in
// lockstep with it (every SELECT/RETURNING site shares both).
const consentColumns = `id, person_id, consent_type, consent_version, purpose, granted_at,
	       granted_by_person, signature_data, is_active, revoked_at, revoked_reason,
	       campus_id, created_at, updated_at`

// scanConsent reads one row in consentColumns order.
func scanConsent(row pgx.Row) (*domain.Consent, error) {
	var c domain.Consent
	if err := row.Scan(
		&c.ID, &c.PersonID, &c.ConsentType, &c.ConsentVersion, &c.Purpose, &c.GrantedAt,
		&c.GrantedByPerson, &c.SignatureData, &c.IsActive, &c.RevokedAt, &c.RevokedReason,
		&c.CampusID, &c.CreatedAt, &c.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &c, nil
}

// Create inserts a consent, mapping the uq_consent_active_person_type unique
// violation to domain.ErrDuplicate.
func (r *ConsentRepository) Create(ctx context.Context, c domain.Consent) (*domain.Consent, error) {
	const q = `
		INSERT INTO consent (id, person_id, consent_type, consent_version, purpose,
		                     granted_by_person, signature_data, campus_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING granted_at, is_active, created_at, updated_at`

	if err := r.q(ctx).QueryRow(ctx, q,
		c.ID, c.PersonID, c.ConsentType, c.ConsentVersion, c.Purpose,
		c.GrantedByPerson, c.SignatureData, c.CampusID,
	).Scan(&c.GrantedAt, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.ErrDuplicate
		}
		return nil, fmt.Errorf("consentRepository.Create: %w", err)
	}
	return &c, nil
}

// FindByID returns a consent by ID, scoped to campus.
func (r *ConsentRepository) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Consent, error) {
	q := `SELECT ` + consentColumns + ` FROM consent WHERE id = $1 AND campus_id = $2`

	c, err := scanConsent(r.q(ctx).QueryRow(ctx, q, id, campusID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("consentRepository.FindByID: %w", err)
	}
	return c, nil
}

// ListByPerson returns the full consent registry of a person (active and
// revoked), newest grant first, scoped to campus.
func (r *ConsentRepository) ListByPerson(ctx context.Context, personID, campusID uuid.UUID) ([]domain.Consent, error) {
	q := `SELECT ` + consentColumns + `
		FROM consent
		WHERE person_id = $1 AND campus_id = $2
		ORDER BY granted_at DESC`

	rows, err := r.q(ctx).Query(ctx, q, personID, campusID)
	if err != nil {
		return nil, fmt.Errorf("consentRepository.ListByPerson: %w", err)
	}
	defer rows.Close()

	consents := []domain.Consent{}
	for rows.Next() {
		c, err := scanConsent(rows)
		if err != nil {
			return nil, fmt.Errorf("consentRepository.ListByPerson: scan: %w", err)
		}
		consents = append(consents, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("consentRepository.ListByPerson: rows: %w", err)
	}
	return consents, nil
}

// Revoke deactivates an active consent, stamping revoked_at and the reason.
// The is_active predicate keeps the update race-safe: a concurrently revoked
// row matches nothing and surfaces as domain.ErrNotFound.
func (r *ConsentRepository) Revoke(ctx context.Context, id, campusID uuid.UUID, reason string) (*domain.Consent, error) {
	q := `UPDATE consent
		SET is_active = FALSE, revoked_at = NOW(), revoked_reason = $3, updated_at = NOW()
		WHERE id = $1 AND campus_id = $2 AND is_active = TRUE
		RETURNING ` + consentColumns

	c, err := scanConsent(r.q(ctx).QueryRow(ctx, q, id, campusID, reason))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("consentRepository.Revoke: %w", err)
	}
	return c, nil
}
