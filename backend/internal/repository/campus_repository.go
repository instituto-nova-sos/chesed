package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// CampusRepository handles campus persistence.
type CampusRepository struct {
	pool *pgxpool.Pool
}

// NewCampusRepository creates a new CampusRepository.
func NewCampusRepository(pool *pgxpool.Pool) *CampusRepository {
	return &CampusRepository{pool: pool}
}

// ListActive returns all active campuses ordered by name.
func (r *CampusRepository) ListActive(ctx context.Context) ([]domain.Campus, error) {
	query := `
		SELECT id, name, region, city, state, country, is_active, created_at, updated_at
		FROM campus
		WHERE is_active = TRUE
		ORDER BY name`

	return r.queryMany(ctx, query)
}

// List returns all campuses ordered by name (including inactive).
func (r *CampusRepository) List(ctx context.Context) ([]domain.Campus, error) {
	query := `
		SELECT id, name, region, city, state, country, is_active, created_at, updated_at
		FROM campus
		ORDER BY name`

	return r.queryMany(ctx, query)
}

// FindByID returns a campus by ID.
func (r *CampusRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Campus, error) {
	query := `
		SELECT id, name, region, city, state, country, is_active, created_at, updated_at
		FROM campus
		WHERE id = $1`

	var c domain.Campus
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Name, &c.Region, &c.City, &c.State,
		&c.Country, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("campusRepository.FindByID: %w", err)
	}

	return &c, nil
}

// Create inserts a new campus.
func (r *CampusRepository) Create(ctx context.Context, campus domain.Campus) (*domain.Campus, error) {
	query := `
		INSERT INTO campus (id, name, region, city, state, country, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		campus.ID, campus.Name, campus.Region, campus.City,
		campus.State, campus.Country, campus.IsActive,
	).Scan(&campus.CreatedAt, &campus.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("campusRepository.Create: %w", err)
	}

	return &campus, nil
}

// Update updates an existing campus.
func (r *CampusRepository) Update(ctx context.Context, campus domain.Campus) (*domain.Campus, error) {
	query := `
		UPDATE campus
		SET name = $2, region = $3, city = $4, state = $5, country = $6,
		    is_active = $7, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		campus.ID, campus.Name, campus.Region, campus.City,
		campus.State, campus.Country, campus.IsActive,
	).Scan(&campus.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("campusRepository.Update: %w", err)
	}

	return &campus, nil
}

func (r *CampusRepository) queryMany(ctx context.Context, query string, args ...any) ([]domain.Campus, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("campusRepository.queryMany: %w", err)
	}
	defer rows.Close()

	var campuses []domain.Campus
	for rows.Next() {
		var c domain.Campus
		if err := rows.Scan(
			&c.ID, &c.Name, &c.Region, &c.City, &c.State,
			&c.Country, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("campusRepository.queryMany: scan: %w", err)
		}
		campuses = append(campuses, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("campusRepository.queryMany: rows: %w", err)
	}

	return campuses, nil
}
