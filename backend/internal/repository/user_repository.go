package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository handles app_user persistence.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// FindByKeycloakSubject looks up an app_user by Keycloak subject ID.
func (r *UserRepository) FindByKeycloakSubject(ctx context.Context, subjectID string) (*domain.AppUser, error) {
	query := `
		SELECT id, person_id, email, keycloak_subject_id, access_profile,
		       campus_id, is_active, last_login, created_at, updated_at
		FROM app_user
		WHERE keycloak_subject_id = $1`

	var user domain.AppUser
	err := r.pool.QueryRow(ctx, query, subjectID).Scan(
		&user.ID, &user.PersonID, &user.Email, &user.KeycloakSubjectID,
		&user.AccessProfile, &user.CampusID, &user.IsActive,
		&user.LastLogin, &user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("userRepository.FindByKeycloakSubject: %w", err)
	}

	return &user, nil
}

// Create inserts a new app_user record.
func (r *UserRepository) Create(ctx context.Context, user domain.AppUser) (*domain.AppUser, error) {
	query := `
		INSERT INTO app_user (id, person_id, email, keycloak_subject_id, access_profile, campus_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		user.ID, user.PersonID, user.Email, user.KeycloakSubjectID,
		user.AccessProfile, user.CampusID, user.IsActive,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("userRepository.Create: %w", err)
	}

	return &user, nil
}

// LinkPersonID sets the person_id for an app_user that has no person linked yet.
// Idempotent: does nothing if person_id is already set.
func (r *UserRepository) LinkPersonID(ctx context.Context, userID uuid.UUID, personID uuid.UUID) error {
	query := `UPDATE app_user SET person_id = $2, updated_at = NOW() WHERE id = $1 AND person_id IS NULL`

	_, err := r.pool.Exec(ctx, query, userID, personID)
	if err != nil {
		return fmt.Errorf("userRepository.LinkPersonID: %w", err)
	}

	return nil
}

// LinkPersonAndCampus sets person_id and campus_id for an app_user.
// Used during self-registration when an app_user was auto-provisioned without these.
func (r *UserRepository) LinkPersonAndCampus(ctx context.Context, userID uuid.UUID, personID uuid.UUID, campusID uuid.UUID) error {
	query := `UPDATE app_user SET person_id = $2, campus_id = $3, updated_at = NOW() WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, userID, personID, campusID)
	if err != nil {
		return fmt.Errorf("userRepository.LinkPersonAndCampus: %w", err)
	}

	return nil
}

// UpdateLastLogin updates the last_login timestamp for an app_user.
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE app_user SET last_login = NOW(), updated_at = NOW() WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("userRepository.UpdateLastLogin: %w", err)
	}

	return nil
}
