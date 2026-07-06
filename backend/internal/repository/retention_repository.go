package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RetentionRepository finds operational records whose retention window has
// lapsed. It embeds base so it runs inside the per-request RLS transaction.
type RetentionRepository struct {
	base
}

// NewRetentionRepository creates a new RetentionRepository.
func NewRetentionRepository(pool Querier) *RetentionRepository {
	return &RetentionRepository{base: base{pool: pool}}
}

// ListExpiredPersonIDs returns the ids of persons in the campus whose last
// update predates olderThan and that have not already been anonymized. The
// anonymized_at guard makes repeated retention runs idempotent.
func (r *RetentionRepository) ListExpiredPersonIDs(ctx context.Context, campusID uuid.UUID, olderThan time.Time) ([]uuid.UUID, error) {
	const q = `
		SELECT id FROM person
		WHERE campus_id = $1
		  AND anonymized_at IS NULL
		  AND updated_at < $2
		ORDER BY updated_at`

	rows, err := r.q(ctx).Query(ctx, q, campusID, olderThan)
	if err != nil {
		return nil, fmt.Errorf("retentionRepository.ListExpiredPersonIDs: %w", err)
	}
	defer rows.Close()

	ids := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("retentionRepository.ListExpiredPersonIDs: scan: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("retentionRepository.ListExpiredPersonIDs: rows: %w", err)
	}
	return ids, nil
}
