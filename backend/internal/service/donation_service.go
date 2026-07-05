package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// DonationRepository defines the interface for donation persistence.
type DonationRepository interface {
	Create(ctx context.Context, d domain.Donation) (*domain.Donation, error)
	FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.DonationDetail, error)
	List(ctx context.Context, filter domain.DonationFilter) (*domain.DonationListResult, error)
	Update(ctx context.Context, d domain.Donation) (*domain.Donation, error)
}

// DonationPersonRepository is the campus-scoped person lookup donations need to
// validate the optional donor reference without leaking foreign-campus
// existence (threat model T3).
type DonationPersonRepository interface {
	FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Person, error)
}

// CreateDonationInput holds validated input for donation creation.
type CreateDonationInput struct {
	DonorPersonID   *string  `json:"donor_person_id" validate:"omitempty,uuid"`
	CampaignID      *string  `json:"campaign_id" validate:"omitempty,uuid"`
	DonationType    string   `json:"donation_type" validate:"required,oneof=FINANCIAL GOODS SERVICES"`
	Amount          *float64 `json:"amount" validate:"omitempty"`
	Currency        *string  `json:"currency" validate:"omitempty,len=3"`
	ItemDescription *string  `json:"item_description" validate:"omitempty"`
	DonationDate    *string  `json:"donation_date" validate:"omitempty"`
	Notes           *string  `json:"notes" validate:"omitempty"`
}

// UpdateDonationInput holds validated input for donation update.
type UpdateDonationInput struct {
	DonorPersonID   *string  `json:"donor_person_id" validate:"omitempty,uuid"`
	CampaignID      *string  `json:"campaign_id" validate:"omitempty,uuid"`
	DonationType    string   `json:"donation_type" validate:"required,oneof=FINANCIAL GOODS SERVICES"`
	Amount          *float64 `json:"amount" validate:"omitempty"`
	Currency        *string  `json:"currency" validate:"omitempty,len=3"`
	ItemDescription *string  `json:"item_description" validate:"omitempty"`
	DonationDate    *string  `json:"donation_date" validate:"omitempty"`
	Notes           *string  `json:"notes" validate:"omitempty"`
}

// DonationService handles donation business logic.
type DonationService struct {
	repo         DonationRepository
	personRepo   DonationPersonRepository
	campaignRepo CampaignRefRepository
	auditSvc     *AuditService
}

// NewDonationService creates a new DonationService.
func NewDonationService(repo DonationRepository, personRepo DonationPersonRepository, campaignRepo CampaignRefRepository, auditSvc *AuditService) *DonationService {
	return &DonationService{repo: repo, personRepo: personRepo, campaignRepo: campaignRepo, auditSvc: auditSvc}
}

// CreateDonation records a new donation stamped with the caller's campus and
// identity.
func (s *DonationService) CreateDonation(ctx context.Context, input CreateDonationInput) (*domain.Donation, error) {
	if err := validate.Struct(input); err != nil {
		return nil, fmt.Errorf("donationService.CreateDonation: %w", err)
	}

	claims := auth.ClaimsFromContext(ctx)
	if claims.CampusID == uuid.Nil {
		return nil, fmt.Errorf("donationService.CreateDonation: %w", domain.ErrForbidden)
	}

	registeredBy := parseUserID(claims.Subject)
	if registeredBy == nil {
		return nil, fmt.Errorf("donationService.CreateDonation: %w", domain.ErrForbidden)
	}

	donation, err := s.buildDonationFromInput(ctx, input, claims.CampusID)
	if err != nil {
		return nil, fmt.Errorf("donationService.CreateDonation: %w", err)
	}
	donation.ID = uuid.New()
	donation.RegisteredBy = *registeredBy

	created, err := s.repo.Create(ctx, donation)
	if err != nil {
		return nil, fmt.Errorf("donationService.CreateDonation: %w", err)
	}

	s.audit(ctx, AuditParams{
		ActionType:  "CREATE",
		EntityType:  "donation",
		EntityID:    &created.ID,
		Module:      "donation",
		Description: "donation created",
		NewValues:   map[string]any{"donation_type": created.DonationType, "amount": created.Amount, "currency": created.Currency},
		Success:     true,
	})

	return created, nil
}

// buildDonationFromInput validates business rules, resolves the optional donor
// and campaign references against the caller's campus, and assembles a
// domain.Donation with defaults applied.
func (s *DonationService) buildDonationFromInput(ctx context.Context, input CreateDonationInput, campusID uuid.UUID) (domain.Donation, error) {
	if err := validateDonationBusinessRules(input.DonationType, input.Amount, input.ItemDescription); err != nil {
		return domain.Donation{}, err
	}

	donorID, err := s.resolveDonor(ctx, input.DonorPersonID, campusID)
	if err != nil {
		return domain.Donation{}, err
	}

	campaignID, err := resolveCampaignRef(ctx, s.campaignRepo, input.CampaignID, campusID)
	if err != nil {
		return domain.Donation{}, err
	}

	donationDate, err := parseDonationDate(input.DonationDate)
	if err != nil {
		return domain.Donation{}, err
	}

	return domain.Donation{
		DonorPersonID:   donorID,
		CampaignID:      campaignID,
		CampusID:        campusID,
		DonationType:    input.DonationType,
		Amount:          input.Amount,
		Currency:        resolveCurrency(input.Currency),
		ItemDescription: input.ItemDescription,
		DonationDate:    donationDate,
		Notes:           input.Notes,
	}, nil
}

// validateDonationBusinessRules enforces that FINANCIAL donations carry a
// positive amount and that GOODS/SERVICES donations describe the item.
func validateDonationBusinessRules(donationType string, amount *float64, itemDescription *string) error {
	if donationType == "FINANCIAL" {
		if amount == nil || *amount <= 0 {
			return fmt.Errorf("financial donation requires a positive amount: %w", domain.ErrValidation)
		}
		return nil
	}
	if itemDescription == nil || *itemDescription == "" {
		return fmt.Errorf("goods/services donation requires an item description: %w", domain.ErrValidation)
	}
	return nil
}

// resolveDonor validates the optional donor reference against the caller's
// campus. An empty reference yields a nil donor; a foreign-campus reference is
// a generic ErrValidation so existence is never disclosed (threat model T3).
func (s *DonationService) resolveDonor(ctx context.Context, donorID *string, campusID uuid.UUID) (*uuid.UUID, error) {
	id, err := parseOptionalUUID(donorID)
	if err != nil {
		return nil, fmt.Errorf("invalid donor_person_id: %w", domain.ErrValidation)
	}
	if id == nil {
		return nil, nil
	}
	if _, err := s.personRepo.FindByID(ctx, *id, campusID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("donor reference not resolvable: %w", domain.ErrValidation)
		}
		return nil, fmt.Errorf("resolve donor: %w", err)
	}
	return id, nil
}

// parseDonationDate parses the optional YYYY-MM-DD input, defaulting to today
// (server clock) when omitted so the value is deterministic.
func parseDonationDate(dateStr *string) (time.Time, error) {
	parsed, err := parseOptionalTime(dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid donation_date: %w", domain.ErrValidation)
	}
	if parsed == nil {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	return *parsed, nil
}

// resolveCurrency defaults an omitted currency to BRL.
func resolveCurrency(currency *string) string {
	if currency == nil || *currency == "" {
		return "BRL"
	}
	return *currency
}

// GetDonation returns a donation with resolved donor and campaign names, scoped
// to campus.
func (s *DonationService) GetDonation(ctx context.Context, id uuid.UUID) (*domain.DonationDetail, error) {
	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return nil, fmt.Errorf("donationService.GetDonation: %w", domain.ErrForbidden)
	}

	detail, err := s.repo.FindByID(ctx, id, campusID)
	if err != nil {
		return nil, fmt.Errorf("donationService.GetDonation: %w", err)
	}
	return detail, nil
}

// ListDonations returns a paginated list of donations for the campus.
func (s *DonationService) ListDonations(ctx context.Context, filter domain.DonationFilter) (*domain.DonationListResult, error) {
	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return nil, fmt.Errorf("donationService.ListDonations: %w", domain.ErrForbidden)
	}
	filter.CampusID = campusID
	result, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("donationService.ListDonations: %w", err)
	}
	return result, nil
}

// UpdateDonation updates the mutable fields of a donation. campus_id,
// registered_by, and created_at are immutable.
func (s *DonationService) UpdateDonation(ctx context.Context, id uuid.UUID, input UpdateDonationInput) (*domain.Donation, error) {
	if err := validate.Struct(input); err != nil {
		return nil, fmt.Errorf("donationService.UpdateDonation: %w", err)
	}

	claims := auth.ClaimsFromContext(ctx)
	if claims.CampusID == uuid.Nil {
		return nil, fmt.Errorf("donationService.UpdateDonation: %w", domain.ErrForbidden)
	}

	existing, err := s.repo.FindByID(ctx, id, claims.CampusID)
	if err != nil {
		return nil, fmt.Errorf("donationService.UpdateDonation: %w", err)
	}
	oldValues := map[string]any{"donation_type": existing.DonationType, "amount": existing.Amount, "currency": existing.Currency}

	mutated, err := s.applyDonationUpdate(ctx, existing, input, claims.CampusID)
	if err != nil {
		return nil, fmt.Errorf("donationService.UpdateDonation: %w", err)
	}

	updated, err := s.repo.Update(ctx, mutated)
	if err != nil {
		return nil, fmt.Errorf("donationService.UpdateDonation: %w", err)
	}

	s.audit(ctx, AuditParams{
		ActionType:  "UPDATE",
		EntityType:  "donation",
		EntityID:    &updated.ID,
		Module:      "donation",
		Description: "donation updated",
		OldValues:   oldValues,
		NewValues:   map[string]any{"donation_type": updated.DonationType, "amount": updated.Amount, "currency": updated.Currency},
		Success:     true,
	})

	return updated, nil
}

// applyDonationUpdate validates business rules, resolves references, and
// assembles the updated donation carrying forward the immutable fields.
func (s *DonationService) applyDonationUpdate(ctx context.Context, existing *domain.DonationDetail, input UpdateDonationInput, campusID uuid.UUID) (domain.Donation, error) {
	if err := validateDonationBusinessRules(input.DonationType, input.Amount, input.ItemDescription); err != nil {
		return domain.Donation{}, err
	}

	donorID, err := s.resolveDonor(ctx, input.DonorPersonID, campusID)
	if err != nil {
		return domain.Donation{}, err
	}

	campaignID, err := resolveCampaignRef(ctx, s.campaignRepo, input.CampaignID, campusID)
	if err != nil {
		return domain.Donation{}, err
	}

	donationDate, err := parseDonationDate(input.DonationDate)
	if err != nil {
		return domain.Donation{}, err
	}

	return domain.Donation{
		ID:              existing.ID,
		DonorPersonID:   donorID,
		CampaignID:      campaignID,
		CampusID:        existing.CampusID,
		DonationType:    input.DonationType,
		Amount:          input.Amount,
		Currency:        resolveCurrency(input.Currency),
		ItemDescription: input.ItemDescription,
		DonationDate:    donationDate,
		Notes:           input.Notes,
		RegisteredBy:    existing.RegisteredBy,
		CreatedAt:       existing.CreatedAt,
	}, nil
}

// audit writes an audit entry, logging (never failing the request) on error.
func (s *DonationService) audit(ctx context.Context, params AuditParams) {
	if auditErr := s.auditSvc.LogAction(ctx, params); auditErr != nil {
		slog.ErrorContext(ctx, "donationService: audit failed",
			"error", auditErr.Error(), "entity_type", params.EntityType, "action", params.ActionType,
		)
	}
}
