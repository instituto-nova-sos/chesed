package repository

import (
	"context"
	"fmt"
	"math"

	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// AuditRepository handles audit log persistence.
//
// It deliberately does NOT use the base/Querier request-transaction pattern:
// audit_log is intentionally excluded from RLS (campus_id is nullable and rows
// are written for pre-campus events), so both the write (Create) and the read
// (List) run on the constructor-provided pool, never on the per-request RLS
// transaction. Campus scoping for reads is applied explicitly in SQL.
type AuditRepository struct {
	pool Querier
}

// NewAuditRepository creates a new AuditRepository. The pool is typed as Querier
// so unit tests can inject a pgxmock pool; *pgxpool.Pool satisfies it in prod.
func NewAuditRepository(pool Querier) *AuditRepository {
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

// buildAuditWhere assembles the WHERE clause for the audit viewer. campus_id is
// mandatory (audit_log is not under RLS); the remaining predicates are optional.
// The date range uses a half-open upper bound (< end + 1 day) so the end day is
// inclusive.
func buildAuditWhere(filter domain.AuditLogFilter) (string, []any) {
	args := []any{filter.CampusID}
	where := "a.campus_id = $1"
	if filter.UserID != nil {
		args = append(args, *filter.UserID)
		where += fmt.Sprintf(" AND a.user_id = $%d", len(args))
	}
	if filter.EntityType != nil {
		args = append(args, *filter.EntityType)
		where += fmt.Sprintf(" AND a.entity_type = $%d", len(args))
	}
	if filter.ActionType != nil {
		args = append(args, *filter.ActionType)
		where += fmt.Sprintf(" AND a.action_type = $%d", len(args))
	}
	if filter.Start != nil {
		args = append(args, *filter.Start)
		where += fmt.Sprintf(" AND a.timestamp >= $%d", len(args))
	}
	if filter.End != nil {
		args = append(args, filter.End.AddDate(0, 0, 1))
		where += fmt.Sprintf(" AND a.timestamp < $%d", len(args))
	}
	return where, args
}

// List returns a paginated, newest-first page of audit log entries for the
// caller's campus. The acting user's email is resolved by joining app_user on
// the Keycloak subject stored in audit_log.user_id.
func (r *AuditRepository) List(ctx context.Context, filter domain.AuditLogFilter) (*domain.AuditLogListResult, error) {
	where, args := buildAuditWhere(filter)

	countQuery := "SELECT COUNT(*) FROM audit_log a WHERE " + where
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("auditRepository.List: count: %w", err)
	}

	if filter.PerPage <= 0 {
		filter.PerPage = 50
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PerPage

	listQuery := fmt.Sprintf(`
		SELECT a.id, u.email, a.action_type, a.entity_type, a.entity_id, a.description,
		       a.old_values, a.new_values, host(a.ip_address), a.timestamp
		FROM audit_log a
		LEFT JOIN app_user u ON u.keycloak_subject_id = a.user_id::text
		WHERE %s
		ORDER BY a.timestamp DESC
		LIMIT $%d OFFSET $%d`, where, len(args)+1, len(args)+2)
	args = append(args, filter.PerPage, offset)

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("auditRepository.List: query: %w", err)
	}
	defer rows.Close()

	items := []domain.AuditLogListItem{}
	for rows.Next() {
		var it domain.AuditLogListItem
		if err := rows.Scan(
			&it.ID, &it.UserEmail, &it.ActionType, &it.EntityType, &it.EntityID, &it.Description,
			&it.OldValues, &it.NewValues, &it.IPAddress, &it.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("auditRepository.List: scan: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("auditRepository.List: rows: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.PerPage)))
	return &domain.AuditLogListResult{
		Data: items,
		Pagination: domain.Pagination{
			Page:       filter.Page,
			PerPage:    filter.PerPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}
