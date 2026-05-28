package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// AuditRepository handles audit log persistence.
type AuditRepository struct {
	pool *pgxpool.Pool
}

// NewAuditRepository creates a new AuditRepository.
func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

// Create inserts an audit log entry. This is the only write operation on audit_log.
func (r *AuditRepository) Create(ctx context.Context, entry domain.AuditLog) error {
	query := `
		INSERT INTO audit_log (
			id, user_id, action_type, entity_type, entity_id,
			module, description, old_values, new_values,
			ip_address, user_agent, campus_id, success, timestamp
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13, $14
		)`

	_, err := r.pool.Exec(ctx, query,
		entry.ID, entry.UserID, entry.ActionType, entry.EntityType, entry.EntityID,
		entry.Module, entry.Description, entry.OldValues, entry.NewValues,
		entry.IPAddress, entry.UserAgent, entry.CampusID, entry.Success, entry.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("auditRepository.Create: %w", err)
	}

	return nil
}
