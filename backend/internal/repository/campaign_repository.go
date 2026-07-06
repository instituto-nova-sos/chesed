package repository

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// CampaignRepository handles campaign persistence.
type CampaignRepository struct {
	base
}

// NewCampaignRepository creates a new CampaignRepository.
func NewCampaignRepository(pool Querier) *CampaignRepository {
	return &CampaignRepository{base: base{pool: pool}}
}

// Create inserts a campaign.
func (r *CampaignRepository) Create(ctx context.Context, c domain.Campaign) (*domain.Campaign, error) {
	const q = `
		INSERT INTO campaign (id, name, description, campaign_type, start_date, end_date,
		                      location_name, location_address, campus_id, coordinator_id,
		                      status, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at, updated_at`

	if err := r.q(ctx).QueryRow(ctx, q,
		c.ID, c.Name, c.Description, c.CampaignType, c.StartDate, c.EndDate,
		c.LocationName, c.LocationAddress, c.CampusID, c.CoordinatorID,
		c.Status, c.CreatedBy,
	).Scan(&c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, fmt.Errorf("campaignRepository.Create: %w", err)
	}
	return &c, nil
}

// FindByID returns a campaign by ID, scoped to campus.
func (r *CampaignRepository) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Campaign, error) {
	const q = `
		SELECT id, name, description, campaign_type, start_date, end_date,
		       location_name, location_address, campus_id, coordinator_id,
		       status, created_at, updated_at
		FROM campaign
		WHERE id = $1 AND campus_id = $2`

	var c domain.Campaign
	if err := r.q(ctx).QueryRow(ctx, q, id, campusID).Scan(
		&c.ID, &c.Name, &c.Description, &c.CampaignType, &c.StartDate, &c.EndDate,
		&c.LocationName, &c.LocationAddress, &c.CampusID, &c.CoordinatorID,
		&c.Status, &c.CreatedAt, &c.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("campaignRepository.FindByID: %w", err)
	}
	return &c, nil
}

// buildCampaignWhere assembles the WHERE clause for campaign listing.
func buildCampaignWhere(filter domain.CampaignFilter) (string, []any) {
	args := []any{filter.CampusID}
	where := "campus_id = $1"
	if filter.Status != nil {
		args = append(args, *filter.Status)
		where += fmt.Sprintf(" AND status = $%d", len(args))
	}
	if filter.Type != nil {
		args = append(args, *filter.Type)
		where += fmt.Sprintf(" AND campaign_type = $%d", len(args))
	}
	return where, args
}

// List returns a paginated list of campaigns scoped to campus.
func (r *CampaignRepository) List(ctx context.Context, filter domain.CampaignFilter) (*domain.CampaignListResult, error) {
	where, args := buildCampaignWhere(filter)

	countQuery := "SELECT COUNT(*) FROM campaign WHERE " + where
	var total int
	if err := r.q(ctx).QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("campaignRepository.List: count: %w", err)
	}

	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PerPage

	listQuery := fmt.Sprintf(`
		SELECT id, name, campaign_type, status, start_date, end_date, location_name
		FROM campaign
		WHERE %s
		ORDER BY start_date DESC, created_at DESC
		LIMIT $%d OFFSET $%d`, where, len(args)+1, len(args)+2)
	args = append(args, filter.PerPage, offset)

	rows, err := r.q(ctx).Query(ctx, listQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("campaignRepository.List: query: %w", err)
	}
	defer rows.Close()

	items := []domain.CampaignListItem{}
	for rows.Next() {
		var it domain.CampaignListItem
		if err := rows.Scan(&it.ID, &it.Name, &it.CampaignType, &it.Status, &it.StartDate, &it.EndDate, &it.LocationName); err != nil {
			return nil, fmt.Errorf("campaignRepository.List: scan: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("campaignRepository.List: rows: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.PerPage)))
	return &domain.CampaignListResult{
		Data: items,
		Pagination: domain.Pagination{
			Page:       filter.Page,
			PerPage:    filter.PerPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// Update applies mutable fields of a campaign scoped to its campus.
func (r *CampaignRepository) Update(ctx context.Context, c domain.Campaign) (*domain.Campaign, error) {
	const q = `
		UPDATE campaign
		SET name = $1, description = $2, campaign_type = $3, start_date = $4,
		    end_date = $5, location_name = $6, location_address = $7,
		    coordinator_id = $8, status = $9, updated_at = NOW()
		WHERE id = $10 AND campus_id = $11
		RETURNING updated_at`

	if err := r.q(ctx).QueryRow(ctx, q,
		c.Name, c.Description, c.CampaignType, c.StartDate, c.EndDate,
		c.LocationName, c.LocationAddress, c.CoordinatorID, c.Status,
		c.ID, c.CampusID,
	).Scan(&c.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("campaignRepository.Update: %w", err)
	}
	return &c, nil
}

// AddTeamMember inserts a team assignment, mapping the uq_campaign_person
// unique violation to domain.ErrDuplicate.
func (r *CampaignRepository) AddTeamMember(ctx context.Context, campaignID, personID uuid.UUID, role string, assignedBy *uuid.UUID) (*domain.CampaignTeamMember, error) {
	const q = `
		WITH inserted AS (
			INSERT INTO campaign_team (campaign_id, person_id, role_in_campaign, assigned_by)
			VALUES ($1, $2, $3, $4)
			RETURNING person_id, role_in_campaign, assigned_at
		)
		SELECT i.person_id, p.full_name, i.role_in_campaign, i.assigned_at
		FROM inserted i
		JOIN person p ON p.id = i.person_id`

	var m domain.CampaignTeamMember
	if err := r.q(ctx).QueryRow(ctx, q, campaignID, personID, role, assignedBy).Scan(
		&m.PersonID, &m.PersonName, &m.RoleInCampaign, &m.AssignedAt,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.ErrDuplicate
		}
		return nil, fmt.Errorf("campaignRepository.AddTeamMember: %w", err)
	}
	return &m, nil
}

// RemoveTeamMember deletes a team assignment; a missing row is ErrNotFound.
func (r *CampaignRepository) RemoveTeamMember(ctx context.Context, campaignID, personID uuid.UUID) error {
	tag, err := r.q(ctx).Exec(ctx,
		`DELETE FROM campaign_team WHERE campaign_id = $1 AND person_id = $2`,
		campaignID, personID,
	)
	if err != nil {
		return fmt.Errorf("campaignRepository.RemoveTeamMember: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ListTeam returns the team members of a campaign with their person names.
func (r *CampaignRepository) ListTeam(ctx context.Context, campaignID uuid.UUID) ([]domain.CampaignTeamMember, error) {
	const q = `
		SELECT ct.person_id, p.full_name, ct.role_in_campaign, ct.assigned_at
		FROM campaign_team ct
		JOIN person p ON p.id = ct.person_id
		WHERE ct.campaign_id = $1
		ORDER BY ct.assigned_at ASC`

	rows, err := r.q(ctx).Query(ctx, q, campaignID)
	if err != nil {
		return nil, fmt.Errorf("campaignRepository.ListTeam: %w", err)
	}
	defer rows.Close()

	members := []domain.CampaignTeamMember{}
	for rows.Next() {
		var m domain.CampaignTeamMember
		if err := rows.Scan(&m.PersonID, &m.PersonName, &m.RoleInCampaign, &m.AssignedAt); err != nil {
			return nil, fmt.Errorf("campaignRepository.ListTeam: scan: %w", err)
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("campaignRepository.ListTeam: rows: %w", err)
	}
	return members, nil
}
