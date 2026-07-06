package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/jackc/pgx/v5"
)

// TriageRepository handles triage persistence.
type TriageRepository struct {
	base
}

// NewTriageRepository creates a new TriageRepository.
func NewTriageRepository(pool Querier) *TriageRepository {
	return &TriageRepository{base: base{pool: pool}}
}

// Create inserts a triage and its requested services in a single transaction.
func (r *TriageRepository) Create(ctx context.Context, triage domain.Triage) (*domain.Triage, error) {
	tx, err := r.q(ctx).Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("triageRepository.Create: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const triageQuery = `
		INSERT INTO triage (id, person_id, campaign_id, campus_id, main_complaint, assigned_team,
		                    triage_date, location, triaged_by, notes, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at`

	if err := tx.QueryRow(ctx, triageQuery,
		triage.ID, triage.PersonID, triage.CampaignID, triage.CampusID, triage.MainComplaint,
		triage.AssignedTeam, triage.TriageDate, triage.Location, triage.TriagedBy,
		triage.Notes, triage.IsActive,
	).Scan(&triage.CreatedAt, &triage.UpdatedAt); err != nil {
		return nil, fmt.Errorf("triageRepository.Create: insert triage: %w", err)
	}

	if err := insertRequestedServices(ctx, tx, triage.ID, triage.RequestedTypes); err != nil {
		return nil, fmt.Errorf("triageRepository.Create: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("triageRepository.Create: commit: %w", err)
	}

	return &triage, nil
}

func insertRequestedServices(ctx context.Context, tx pgx.Tx, triageID uuid.UUID, serviceTypeIDs []uuid.UUID) error {
	if len(serviceTypeIDs) == 0 {
		return nil
	}
	const q = `INSERT INTO triage_requested_service (triage_id, service_type_id) VALUES ($1, $2)`
	for _, stID := range serviceTypeIDs {
		if _, err := tx.Exec(ctx, q, triageID, stID); err != nil {
			return fmt.Errorf("insert requested service: %w", err)
		}
	}
	return nil
}

// CreateWithSync inserts a triage carrying the client-supplied sync_id along
// with the requested service junction rows in a single transaction.
// Idempotency at the DB layer is enforced by uq_triage_sync_id.
func (r *TriageRepository) CreateWithSync(ctx context.Context, triage domain.Triage, syncID uuid.UUID) (*domain.Triage, error) {
	tx, err := r.q(ctx).Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("triageRepository.CreateWithSync: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO triage (id, person_id, campaign_id, campus_id, main_complaint, assigned_team,
		                    triage_date, location, triaged_by, notes, is_active, sync_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at, updated_at`

	if err := tx.QueryRow(ctx, q,
		triage.ID, triage.PersonID, triage.CampaignID, triage.CampusID, triage.MainComplaint,
		triage.AssignedTeam, triage.TriageDate, triage.Location, triage.TriagedBy,
		triage.Notes, triage.IsActive, syncID,
	).Scan(&triage.CreatedAt, &triage.UpdatedAt); err != nil {
		return nil, fmt.Errorf("triageRepository.CreateWithSync: insert: %w", err)
	}

	if err := insertRequestedServices(ctx, tx, triage.ID, triage.RequestedTypes); err != nil {
		return nil, fmt.Errorf("triageRepository.CreateWithSync: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("triageRepository.CreateWithSync: commit: %w", err)
	}
	return &triage, nil
}

// FindBySyncID returns a triage by its client-supplied sync_id, scoped to
// campus. Requested services are not loaded here — callers only need the row
// existence for idempotency lookups.
func (r *TriageRepository) FindBySyncID(ctx context.Context, syncID, campusID uuid.UUID) (*domain.Triage, error) {
	const q = `
		SELECT id, person_id, campaign_id, campus_id, main_complaint, assigned_team,
		       triage_date, location, triaged_by, notes, is_active,
		       created_at, updated_at
		FROM triage
		WHERE sync_id = $1 AND campus_id = $2`

	var t domain.Triage
	if err := r.q(ctx).QueryRow(ctx, q, syncID, campusID).Scan(
		&t.ID, &t.PersonID, &t.CampaignID, &t.CampusID, &t.MainComplaint, &t.AssignedTeam,
		&t.TriageDate, &t.Location, &t.TriagedBy, &t.Notes, &t.IsActive,
		&t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("triageRepository.FindBySyncID: %w", err)
	}
	return &t, nil
}

// ListUpdatedSince returns triages modified after the cursor, ordered ASC by
// updated_at, bounded by limit.
func (r *TriageRepository) ListUpdatedSince(ctx context.Context, campusID uuid.UUID, since time.Time, limit int) ([]domain.Triage, error) {
	const q = `
		SELECT id, person_id, campaign_id, campus_id, main_complaint, assigned_team,
		       triage_date, location, triaged_by, notes, is_active,
		       created_at, updated_at
		FROM triage
		WHERE campus_id = $1 AND updated_at > $2
		ORDER BY updated_at ASC
		LIMIT $3`

	rows, err := r.q(ctx).Query(ctx, q, campusID, since, limit)
	if err != nil {
		return nil, fmt.Errorf("triageRepository.ListUpdatedSince: %w", err)
	}
	defer rows.Close()

	out := []domain.Triage{}
	for rows.Next() {
		var t domain.Triage
		if err := rows.Scan(
			&t.ID, &t.PersonID, &t.CampaignID, &t.CampusID, &t.MainComplaint, &t.AssignedTeam,
			&t.TriageDate, &t.Location, &t.TriagedBy, &t.Notes, &t.IsActive,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("triageRepository.ListUpdatedSince: scan: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("triageRepository.ListUpdatedSince: rows: %w", err)
	}
	return out, nil
}

// FindByID returns a triage by ID, scoped to campus, with its requested services.
func (r *TriageRepository) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Triage, error) {
	const q = `
		SELECT id, person_id, campaign_id, campus_id, main_complaint, assigned_team,
		       triage_date, location, triaged_by, notes, is_active,
		       created_at, updated_at
		FROM triage
		WHERE id = $1 AND campus_id = $2 AND is_active = TRUE`

	var t domain.Triage
	if err := r.q(ctx).QueryRow(ctx, q, id, campusID).Scan(
		&t.ID, &t.PersonID, &t.CampaignID, &t.CampusID, &t.MainComplaint, &t.AssignedTeam,
		&t.TriageDate, &t.Location, &t.TriagedBy, &t.Notes, &t.IsActive,
		&t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("triageRepository.FindByID: %w", err)
	}

	requested, err := r.findRequestedServices(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("triageRepository.FindByID: %w", err)
	}
	t.RequestedTypes = requested

	return &t, nil
}

func (r *TriageRepository) findRequestedServices(ctx context.Context, triageID uuid.UUID) ([]uuid.UUID, error) {
	const q = `SELECT service_type_id FROM triage_requested_service WHERE triage_id = $1 ORDER BY service_type_id`
	rows, err := r.q(ctx).Query(ctx, q, triageID)
	if err != nil {
		return nil, fmt.Errorf("find requested services: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan requested service: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// List returns a paginated list of triages scoped to campus, with optional filters.
func buildTriageWhere(filter domain.TriageFilter) (string, []any) {
	args := []any{filter.CampusID}
	where := "t.campus_id = $1 AND t.is_active = TRUE"
	addClause := func(value any, clause string) {
		args = append(args, value)
		where += fmt.Sprintf(clause, len(args))
	}
	if filter.PersonID != nil {
		addClause(*filter.PersonID, " AND t.person_id = $%d")
	}
	if filter.From != nil {
		addClause(*filter.From, " AND t.triage_date >= $%d")
	}
	if filter.To != nil {
		addClause(*filter.To, " AND t.triage_date <= $%d")
	}
	return where, args
}

func (r *TriageRepository) List(ctx context.Context, filter domain.TriageFilter) (*domain.TriageListResult, error) {
	where, args := buildTriageWhere(filter)

	countQuery := "SELECT COUNT(*) FROM triage t WHERE " + where
	var total int
	if err := r.q(ctx).QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("triageRepository.List: count: %w", err)
	}

	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PerPage

	listQuery := fmt.Sprintf(`
		SELECT t.id, t.person_id, p.full_name, t.main_complaint, t.triage_date,
		       (SELECT COUNT(*) FROM triage_requested_service trs WHERE trs.triage_id = t.id)
		FROM triage t
		JOIN person p ON p.id = t.person_id
		WHERE %s
		ORDER BY t.triage_date DESC
		LIMIT $%d OFFSET $%d`, where, len(args)+1, len(args)+2)
	args = append(args, filter.PerPage, offset)

	rows, err := r.q(ctx).Query(ctx, listQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("triageRepository.List: query: %w", err)
	}
	defer rows.Close()

	items := []domain.TriageListItem{}
	for rows.Next() {
		var it domain.TriageListItem
		if err := rows.Scan(&it.ID, &it.PersonID, &it.PersonName, &it.MainComplaint, &it.TriageDate, &it.RequestedCount); err != nil {
			return nil, fmt.Errorf("triageRepository.List: scan: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("triageRepository.List: rows: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.PerPage)))
	return &domain.TriageListResult{
		Data: items,
		Pagination: domain.Pagination{
			Page:       filter.Page,
			PerPage:    filter.PerPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// Update applies an update to mutable fields of a triage scoped to its campus.
func (r *TriageRepository) Update(ctx context.Context, triage domain.Triage) (*domain.Triage, error) {
	tx, err := r.q(ctx).Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("triageRepository.Update: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		UPDATE triage
		SET main_complaint = $1,
		    assigned_team = $2,
		    location = $3,
		    notes = $4,
		    updated_at = NOW()
		WHERE id = $5 AND campus_id = $6 AND is_active = TRUE
		RETURNING updated_at`

	if err := tx.QueryRow(ctx, q,
		triage.MainComplaint, triage.AssignedTeam, triage.Location, triage.Notes,
		triage.ID, triage.CampusID,
	).Scan(&triage.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("triageRepository.Update: %w", err)
	}

	if triage.RequestedTypes != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM triage_requested_service WHERE triage_id = $1`, triage.ID); err != nil {
			return nil, fmt.Errorf("triageRepository.Update: clear services: %w", err)
		}
		if err := insertRequestedServices(ctx, tx, triage.ID, triage.RequestedTypes); err != nil {
			return nil, fmt.Errorf("triageRepository.Update: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("triageRepository.Update: commit: %w", err)
	}

	return &triage, nil
}
