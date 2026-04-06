package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// ServiceTypeRepository handles service type persistence.
type ServiceTypeRepository struct {
	pool *pgxpool.Pool
}

// NewServiceTypeRepository creates a new ServiceTypeRepository.
func NewServiceTypeRepository(pool *pgxpool.Pool) *ServiceTypeRepository {
	return &ServiceTypeRepository{pool: pool}
}

// ListActive returns all active service types ordered by name.
func (r *ServiceTypeRepository) ListActive(ctx context.Context) ([]domain.ServiceType, error) {
	query := `
		SELECT id, name, category, description, is_active, created_at, updated_at
		FROM service_type
		WHERE is_active = TRUE
		ORDER BY name`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("serviceTypeRepository.ListActive: %w", err)
	}
	defer rows.Close()

	var types []domain.ServiceType
	for rows.Next() {
		var st domain.ServiceType
		if err := rows.Scan(
			&st.ID, &st.Name, &st.Category, &st.Description,
			&st.IsActive, &st.CreatedAt, &st.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("serviceTypeRepository.ListActive: scan: %w", err)
		}
		types = append(types, st)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("serviceTypeRepository.ListActive: rows: %w", err)
	}

	return types, nil
}
