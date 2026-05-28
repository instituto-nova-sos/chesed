package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// PersonRoleRepository handles person_role persistence.
type PersonRoleRepository struct {
	pool *pgxpool.Pool
}

// NewPersonRoleRepository creates a new PersonRoleRepository.
func NewPersonRoleRepository(pool *pgxpool.Pool) *PersonRoleRepository {
	return &PersonRoleRepository{pool: pool}
}

// Create inserts a new person role.
func (r *PersonRoleRepository) Create(ctx context.Context, role domain.PersonRole) (*domain.PersonRole, error) {
	query := `
		INSERT INTO person_role (id, person_id, role_type, professional_specialty,
		                         is_active, activated_by, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING activated_at, created_at`

	err := r.pool.QueryRow(ctx, query,
		role.ID, role.PersonID, role.RoleType, role.ProfessionalSpecialty,
		role.IsActive, role.ActivatedBy, role.Notes,
	).Scan(&role.ActivatedAt, &role.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.ErrDuplicate
		}
		return nil, fmt.Errorf("personRoleRepository.Create: %w", err)
	}

	return &role, nil
}

// FindByPersonID returns all roles for a person.
func (r *PersonRoleRepository) FindByPersonID(ctx context.Context, personID uuid.UUID) ([]domain.PersonRole, error) {
	query := `
		SELECT id, person_id, role_type, professional_specialty, is_active,
		       activated_at, deactivated_at, activated_by, deactivated_by, notes, created_at
		FROM person_role
		WHERE person_id = $1
		ORDER BY activated_at`

	rows, err := r.pool.Query(ctx, query, personID)
	if err != nil {
		return nil, fmt.Errorf("personRoleRepository.FindByPersonID: %w", err)
	}
	defer rows.Close()

	var roles []domain.PersonRole
	for rows.Next() {
		var pr domain.PersonRole
		if err := rows.Scan(
			&pr.ID, &pr.PersonID, &pr.RoleType, &pr.ProfessionalSpecialty,
			&pr.IsActive, &pr.ActivatedAt, &pr.DeactivatedAt,
			&pr.ActivatedBy, &pr.DeactivatedBy, &pr.Notes, &pr.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("personRoleRepository.FindByPersonID: scan: %w", err)
		}
		roles = append(roles, pr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("personRoleRepository.FindByPersonID: rows: %w", err)
	}

	if roles == nil {
		roles = []domain.PersonRole{}
	}
	return roles, nil
}

// ToggleActive activates or deactivates a person role.
func (r *PersonRoleRepository) ToggleActive(ctx context.Context, roleID uuid.UUID, isActive bool, userID *uuid.UUID) (*domain.PersonRole, error) {
	var query string
	if isActive {
		query = `
			UPDATE person_role
			SET is_active = TRUE, deactivated_at = NULL, deactivated_by = NULL
			WHERE id = $1
			RETURNING id, person_id, role_type, professional_specialty, is_active,
			          activated_at, deactivated_at, activated_by, deactivated_by, notes, created_at`
	} else {
		query = `
			UPDATE person_role
			SET is_active = FALSE, deactivated_at = NOW(), deactivated_by = $2
			WHERE id = $1
			RETURNING id, person_id, role_type, professional_specialty, is_active,
			          activated_at, deactivated_at, activated_by, deactivated_by, notes, created_at`
	}

	var pr domain.PersonRole
	var err error

	if isActive {
		err = r.pool.QueryRow(ctx, query, roleID).Scan(
			&pr.ID, &pr.PersonID, &pr.RoleType, &pr.ProfessionalSpecialty,
			&pr.IsActive, &pr.ActivatedAt, &pr.DeactivatedAt,
			&pr.ActivatedBy, &pr.DeactivatedBy, &pr.Notes, &pr.CreatedAt,
		)
	} else {
		err = r.pool.QueryRow(ctx, query, roleID, userID).Scan(
			&pr.ID, &pr.PersonID, &pr.RoleType, &pr.ProfessionalSpecialty,
			&pr.IsActive, &pr.ActivatedAt, &pr.DeactivatedAt,
			&pr.ActivatedBy, &pr.DeactivatedBy, &pr.Notes, &pr.CreatedAt,
		)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("personRoleRepository.ToggleActive: %w", err)
	}

	return &pr, nil
}
