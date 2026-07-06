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
// activity predates olderThan and that have not already been anonymized. Last
// activity is the most recent of the person's own row and any related triage,
// attendance, or donation — so a subject still being assisted (recent triage or
// attendance) is never anonymized even if their profile row is old. The
// anonymized_at guard makes repeated retention runs idempotent.
func (r *RetentionRepository) ListExpiredPersonIDs(ctx context.Context, campusID uuid.UUID, olderThan time.Time) ([]uuid.UUID, error) {
	const q = `
		SELECT p.id
		FROM person p
		WHERE p.campus_id = $1
		  AND p.anonymized_at IS NULL
		  AND GREATEST(
		        p.updated_at,
		        COALESCE((SELECT max(t.updated_at) FROM triage t WHERE t.person_id = p.id), p.updated_at),
		        COALESCE((SELECT max(a.updated_at) FROM attendance a WHERE a.person_id = p.id), p.updated_at),
		        COALESCE((SELECT max(d.updated_at) FROM donation d WHERE d.donor_person_id = p.id), p.updated_at)
		      ) < $2
		ORDER BY p.updated_at`

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
