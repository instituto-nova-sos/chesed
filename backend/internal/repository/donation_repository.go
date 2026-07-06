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

// DonationRepository handles donation persistence.
type DonationRepository struct {
	base
}

// NewDonationRepository creates a new DonationRepository.
func NewDonationRepository(pool Querier) *DonationRepository {
	return &DonationRepository{base: base{pool: pool}}
}

// Create inserts a donation, mapping the receipt_number unique violation to
// domain.ErrDuplicate.
func (r *DonationRepository) Create(ctx context.Context, d domain.Donation) (*domain.Donation, error) {
	const q = `
		INSERT INTO donation (id, donor_person_id, campaign_id, campus_id, donation_type,
		                      amount, currency, item_description, donation_date, notes,
		                      registered_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at`

	if err := r.q(ctx).QueryRow(ctx, q,
		d.ID, d.DonorPersonID, d.CampaignID, d.CampusID, d.DonationType,
		d.Amount, d.Currency, d.ItemDescription, d.DonationDate, d.Notes,
		d.RegisteredBy,
	).Scan(&d.CreatedAt, &d.UpdatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.ErrDuplicate
		}
		return nil, fmt.Errorf("donationRepository.Create: %w", err)
	}
	return &d, nil
}

// FindByID returns a donation with resolved donor and campaign names, scoped to
// campus.
func (r *DonationRepository) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.DonationDetail, error) {
	const q = `
		SELECT d.id, d.donor_person_id, d.campaign_id, d.campus_id, d.donation_type,
		       d.amount, d.currency, d.item_description, d.donation_date,
		       d.receipt_number, d.receipt_issued_at, d.notes, d.registered_by,
		       d.created_at, d.updated_at,
		       p.full_name, c.name
		FROM donation d
		LEFT JOIN person p ON p.id = d.donor_person_id
		LEFT JOIN campaign c ON c.id = d.campaign_id
		WHERE d.id = $1 AND d.campus_id = $2`

	var detail domain.DonationDetail
	if err := r.q(ctx).QueryRow(ctx, q, id, campusID).Scan(
		&detail.ID, &detail.DonorPersonID, &detail.CampaignID, &detail.CampusID, &detail.DonationType,
		&detail.Amount, &detail.Currency, &detail.ItemDescription, &detail.DonationDate,
		&detail.ReceiptNumber, &detail.ReceiptIssuedAt, &detail.Notes, &detail.RegisteredBy,
		&detail.CreatedAt, &detail.UpdatedAt,
		&detail.DonorName, &detail.CampaignName,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("donationRepository.FindByID: %w", err)
	}
	return &detail, nil
}

// buildDonationWhere assembles the WHERE clause for donation listing.
func buildDonationWhere(filter domain.DonationFilter) (string, []any) {
	args := []any{filter.CampusID}
	where := "d.campus_id = $1"
	if filter.DonationType != nil {
		args = append(args, *filter.DonationType)
		where += fmt.Sprintf(" AND d.donation_type = $%d", len(args))
	}
	if filter.CampaignID != nil {
		args = append(args, *filter.CampaignID)
		where += fmt.Sprintf(" AND d.campaign_id = $%d", len(args))
	}
	return where, args
}

// List returns a paginated list of donations scoped to campus.
func (r *DonationRepository) List(ctx context.Context, filter domain.DonationFilter) (*domain.DonationListResult, error) {
	where, args := buildDonationWhere(filter)

	countQuery := "SELECT COUNT(*) FROM donation d WHERE " + where
	var total int
	if err := r.q(ctx).QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("donationRepository.List: count: %w", err)
	}

	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PerPage

	listQuery := fmt.Sprintf(`
		SELECT d.id, d.donation_type, d.amount, d.currency, d.donation_date,
		       p.full_name, c.name
		FROM donation d
		LEFT JOIN person p ON p.id = d.donor_person_id
		LEFT JOIN campaign c ON c.id = d.campaign_id
		WHERE %s
		ORDER BY d.donation_date DESC, d.created_at DESC
		LIMIT $%d OFFSET $%d`, where, len(args)+1, len(args)+2)
	args = append(args, filter.PerPage, offset)

	rows, err := r.q(ctx).Query(ctx, listQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("donationRepository.List: query: %w", err)
	}
	defer rows.Close()

	items := []domain.DonationListItem{}
	for rows.Next() {
		var it domain.DonationListItem
		if err := rows.Scan(&it.ID, &it.DonationType, &it.Amount, &it.Currency, &it.DonationDate, &it.DonorName, &it.CampaignName); err != nil {
			return nil, fmt.Errorf("donationRepository.List: scan: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("donationRepository.List: rows: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.PerPage)))
	return &domain.DonationListResult{
		Data: items,
		Pagination: domain.Pagination{
			Page:       filter.Page,
			PerPage:    filter.PerPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// Update applies mutable fields of a donation scoped to its campus. campus_id,
// registered_by, and created_at are immutable and never touched here.
func (r *DonationRepository) Update(ctx context.Context, d domain.Donation) (*domain.Donation, error) {
	const q = `
		UPDATE donation
		SET donor_person_id = $1, campaign_id = $2, donation_type = $3, amount = $4,
		    currency = $5, item_description = $6, donation_date = $7, notes = $8,
		    updated_at = NOW()
		WHERE id = $9 AND campus_id = $10
		RETURNING updated_at`

	if err := r.q(ctx).QueryRow(ctx, q,
		d.DonorPersonID, d.CampaignID, d.DonationType, d.Amount,
		d.Currency, d.ItemDescription, d.DonationDate, d.Notes,
		d.ID, d.CampusID,
	).Scan(&d.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.ErrDuplicate
		}
		return nil, fmt.Errorf("donationRepository.Update: %w", err)
	}
	return &d, nil
}
