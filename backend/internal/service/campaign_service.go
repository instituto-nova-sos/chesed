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

// CampaignRepository defines the interface for campaign persistence.
type CampaignRepository interface {
	Create(ctx context.Context, c domain.Campaign) (*domain.Campaign, error)
	FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Campaign, error)
	List(ctx context.Context, filter domain.CampaignFilter) (*domain.CampaignListResult, error)
	Update(ctx context.Context, c domain.Campaign) (*domain.Campaign, error)
	AddTeamMember(ctx context.Context, campaignID, personID uuid.UUID, role string, assignedBy *uuid.UUID) (*domain.CampaignTeamMember, error)
	RemoveTeamMember(ctx context.Context, campaignID, personID uuid.UUID) error
	ListTeam(ctx context.Context, campaignID uuid.UUID) ([]domain.CampaignTeamMember, error)
}

// CampaignPersonRepository is the campus-scoped person lookup campaigns need
// to validate coordinator and team references without leaking foreign-campus
// existence (threat model T3).
type CampaignPersonRepository interface {
	FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Person, error)
}

// CreateCampaignInput holds validated input for campaign creation.
type CreateCampaignInput struct {
	Name            string  `json:"name" validate:"required,max=200"`
	Description     *string `json:"description" validate:"omitempty"`
	CampaignType    string  `json:"campaign_type" validate:"required,oneof=SOCIAL_ACTION EDUCATIONAL HEALTH COMMUNITY OTHER"`
	StartDate       string  `json:"start_date" validate:"required"`
	EndDate         *string `json:"end_date" validate:"omitempty"`
	LocationName    *string `json:"location_name" validate:"omitempty,max=200"`
	LocationAddress *string `json:"location_address" validate:"omitempty"`
	CoordinatorID   *string `json:"coordinator_id" validate:"omitempty,uuid"`
}

// UpdateCampaignInput holds validated input for campaign update.
type UpdateCampaignInput struct {
	Name            string  `json:"name" validate:"required,max=200"`
	Description     *string `json:"description" validate:"omitempty"`
	CampaignType    string  `json:"campaign_type" validate:"required,oneof=SOCIAL_ACTION EDUCATIONAL HEALTH COMMUNITY OTHER"`
	StartDate       string  `json:"start_date" validate:"required"`
	EndDate         *string `json:"end_date" validate:"omitempty"`
	LocationName    *string `json:"location_name" validate:"omitempty,max=200"`
	LocationAddress *string `json:"location_address" validate:"omitempty"`
	CoordinatorID   *string `json:"coordinator_id" validate:"omitempty,uuid"`
	Status          string  `json:"status" validate:"required,oneof=PLANNED ACTIVE COMPLETED CANCELLED"`
}

// AddTeamMemberInput holds validated input for a team assignment.
type AddTeamMemberInput struct {
	PersonID       string `json:"person_id" validate:"required,uuid"`
	RoleInCampaign string `json:"role_in_campaign" validate:"required,oneof=COORDINATOR PROFESSIONAL VOLUNTEER SUPPORT"`
}

// CampaignService handles campaign business logic.
type CampaignService struct {
	repo       CampaignRepository
	personRepo CampaignPersonRepository
	auditSvc   *AuditService
}

// NewCampaignService creates a new CampaignService.
func NewCampaignService(repo CampaignRepository, personRepo CampaignPersonRepository, auditSvc *AuditService) *CampaignService {
	return &CampaignService{repo: repo, personRepo: personRepo, auditSvc: auditSvc}
}

// CreateCampaign creates a new campaign in PLANNED status.
func (s *CampaignService) CreateCampaign(ctx context.Context, input CreateCampaignInput) (*domain.Campaign, error) {
	if err := validate.Struct(input); err != nil {
		return nil, fmt.Errorf("campaignService.CreateCampaign: %w", err)
	}

	claims := auth.ClaimsFromContext(ctx)
	if claims.CampusID == uuid.Nil {
		return nil, fmt.Errorf("campaignService.CreateCampaign: %w", domain.ErrForbidden)
	}

	campaign, err := s.buildCampaignFromInput(ctx, input, claims)
	if err != nil {
		return nil, err
	}

	created, err := s.repo.Create(ctx, campaign)
	if err != nil {
		return nil, fmt.Errorf("campaignService.CreateCampaign: %w", err)
	}

	s.audit(ctx, AuditParams{
		ActionType:  "CREATE",
		EntityType:  "campaign",
		EntityID:    &created.ID,
		Module:      "campaign",
		Description: "campaign created",
		NewValues:   map[string]any{"name": created.Name, "campaign_type": created.CampaignType, "status": created.Status},
		Success:     true,
	})

	return created, nil
}

// buildCampaignFromInput parses dates, validates the date range and the
// coordinator reference, and assembles a domain.Campaign.
func (s *CampaignService) buildCampaignFromInput(ctx context.Context, input CreateCampaignInput, claims auth.AuthClaims) (domain.Campaign, error) {
	start, end, err := parseCampaignDates(input.StartDate, input.EndDate)
	if err != nil {
		return domain.Campaign{}, fmt.Errorf("campaignService.CreateCampaign: %w", err)
	}

	coordinatorID, err := s.resolveCoordinator(ctx, input.CoordinatorID, claims.CampusID)
	if err != nil {
		return domain.Campaign{}, fmt.Errorf("campaignService.CreateCampaign: %w", err)
	}

	return domain.Campaign{
		ID:              uuid.New(),
		Name:            input.Name,
		Description:     input.Description,
		CampaignType:    input.CampaignType,
		StartDate:       *start,
		EndDate:         end,
		LocationName:    input.LocationName,
		LocationAddress: input.LocationAddress,
		CampusID:        claims.CampusID,
		CoordinatorID:   coordinatorID,
		Status:          "PLANNED",
		CreatedBy:       parseUserID(claims.Subject),
	}, nil
}

// parseCampaignDates validates the start date and the optional end date,
// rejecting an end date earlier than the start date.
func parseCampaignDates(startStr string, endStr *string) (*time.Time, *time.Time, error) {
	start, err := parseOptionalTime(&startStr)
	if err != nil || start == nil {
		return nil, nil, fmt.Errorf("invalid start_date: %w", domain.ErrValidation)
	}
	end, err := parseOptionalTime(endStr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid end_date: %w", domain.ErrValidation)
	}
	if end != nil && end.Before(*start) {
		return nil, nil, fmt.Errorf("end_date before start_date: %w", domain.ErrValidation)
	}
	return start, end, nil
}

// resolveCoordinator validates that a referenced coordinator exists in the
// caller's campus. The error is generic so foreign-campus existence is never
// disclosed.
func (s *CampaignService) resolveCoordinator(ctx context.Context, coordinatorID *string, campusID uuid.UUID) (*uuid.UUID, error) {
	id, err := parseOptionalUUID(coordinatorID)
	if err != nil {
		return nil, fmt.Errorf("invalid coordinator_id: %w", domain.ErrValidation)
	}
	if id == nil {
		return nil, nil
	}
	if _, err := s.personRepo.FindByID(ctx, *id, campusID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("coordinator reference not resolvable: %w", domain.ErrValidation)
		}
		return nil, fmt.Errorf("resolve coordinator: %w", err)
	}
	return id, nil
}

// GetCampaign returns a campaign with its team, scoped to campus.
func (s *CampaignService) GetCampaign(ctx context.Context, id uuid.UUID) (*domain.CampaignDetail, error) {
	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return nil, fmt.Errorf("campaignService.GetCampaign: %w", domain.ErrForbidden)
	}

	campaign, err := s.repo.FindByID(ctx, id, campusID)
	if err != nil {
		return nil, fmt.Errorf("campaignService.GetCampaign: %w", err)
	}

	team, err := s.repo.ListTeam(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("campaignService.GetCampaign: team: %w", err)
	}
	if team == nil {
		team = []domain.CampaignTeamMember{}
	}

	return &domain.CampaignDetail{Campaign: *campaign, Team: team}, nil
}

// ListCampaigns returns a paginated list of campaigns for the campus.
func (s *CampaignService) ListCampaigns(ctx context.Context, filter domain.CampaignFilter) (*domain.CampaignListResult, error) {
	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return nil, fmt.Errorf("campaignService.ListCampaigns: %w", domain.ErrForbidden)
	}
	filter.CampusID = campusID
	result, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("campaignService.ListCampaigns: %w", err)
	}
	return result, nil
}

// UpdateCampaign updates mutable fields and the lifecycle status of a campaign.
func (s *CampaignService) UpdateCampaign(ctx context.Context, id uuid.UUID, input UpdateCampaignInput) (*domain.Campaign, error) {
	if err := validate.Struct(input); err != nil {
		return nil, fmt.Errorf("campaignService.UpdateCampaign: %w", err)
	}

	claims := auth.ClaimsFromContext(ctx)
	if claims.CampusID == uuid.Nil {
		return nil, fmt.Errorf("campaignService.UpdateCampaign: %w", domain.ErrForbidden)
	}

	existing, err := s.repo.FindByID(ctx, id, claims.CampusID)
	if err != nil {
		return nil, fmt.Errorf("campaignService.UpdateCampaign: %w", err)
	}
	oldValues := map[string]any{"name": existing.Name, "status": existing.Status}

	if err := s.applyCampaignUpdate(ctx, existing, input, claims.CampusID); err != nil {
		return nil, err
	}

	updated, err := s.repo.Update(ctx, *existing)
	if err != nil {
		return nil, fmt.Errorf("campaignService.UpdateCampaign: %w", err)
	}

	s.audit(ctx, AuditParams{
		ActionType:  "UPDATE",
		EntityType:  "campaign",
		EntityID:    &updated.ID,
		Module:      "campaign",
		Description: "campaign updated",
		OldValues:   oldValues,
		NewValues:   map[string]any{"name": updated.Name, "status": updated.Status},
		Success:     true,
	})

	return updated, nil
}

// applyCampaignUpdate parses and applies the update input onto the existing
// campaign in place.
func (s *CampaignService) applyCampaignUpdate(ctx context.Context, existing *domain.Campaign, input UpdateCampaignInput, campusID uuid.UUID) error {
	start, end, err := parseCampaignDates(input.StartDate, input.EndDate)
	if err != nil {
		return fmt.Errorf("campaignService.UpdateCampaign: %w", err)
	}

	coordinatorID, err := s.resolveCoordinator(ctx, input.CoordinatorID, campusID)
	if err != nil {
		return fmt.Errorf("campaignService.UpdateCampaign: %w", err)
	}

	existing.Name = input.Name
	existing.Description = input.Description
	existing.CampaignType = input.CampaignType
	existing.StartDate = *start
	existing.EndDate = end
	existing.LocationName = input.LocationName
	existing.LocationAddress = input.LocationAddress
	existing.CoordinatorID = coordinatorID
	existing.Status = input.Status
	return nil
}

// AddTeamMember assigns a person to a campaign team.
func (s *CampaignService) AddTeamMember(ctx context.Context, campaignID uuid.UUID, input AddTeamMemberInput) (*domain.CampaignTeamMember, error) {
	if err := validate.Struct(input); err != nil {
		return nil, fmt.Errorf("campaignService.AddTeamMember: %w", err)
	}

	claims := auth.ClaimsFromContext(ctx)
	if claims.CampusID == uuid.Nil {
		return nil, fmt.Errorf("campaignService.AddTeamMember: %w", domain.ErrForbidden)
	}

	if _, err := s.repo.FindByID(ctx, campaignID, claims.CampusID); err != nil {
		return nil, fmt.Errorf("campaignService.AddTeamMember: %w", err)
	}

	personID, err := uuid.Parse(input.PersonID)
	if err != nil {
		return nil, fmt.Errorf("campaignService.AddTeamMember: invalid person_id: %w", domain.ErrValidation)
	}
	if _, err := s.personRepo.FindByID(ctx, personID, claims.CampusID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("campaignService.AddTeamMember: person reference not resolvable: %w", domain.ErrValidation)
		}
		return nil, fmt.Errorf("campaignService.AddTeamMember: resolve person: %w", err)
	}

	member, err := s.repo.AddTeamMember(ctx, campaignID, personID, input.RoleInCampaign, parseUserID(claims.Subject))
	if err != nil {
		return nil, fmt.Errorf("campaignService.AddTeamMember: %w", err)
	}

	s.audit(ctx, AuditParams{
		ActionType:  "CREATE",
		EntityType:  "campaign_team",
		EntityID:    &campaignID,
		Module:      "campaign",
		Description: "team member assigned",
		NewValues:   map[string]any{"person_id": personID, "role_in_campaign": input.RoleInCampaign},
		Success:     true,
	})

	return member, nil
}

// RemoveTeamMember removes a person from a campaign team.
func (s *CampaignService) RemoveTeamMember(ctx context.Context, campaignID, personID uuid.UUID) error {
	claims := auth.ClaimsFromContext(ctx)
	if claims.CampusID == uuid.Nil {
		return fmt.Errorf("campaignService.RemoveTeamMember: %w", domain.ErrForbidden)
	}

	if _, err := s.repo.FindByID(ctx, campaignID, claims.CampusID); err != nil {
		return fmt.Errorf("campaignService.RemoveTeamMember: %w", err)
	}

	if err := s.repo.RemoveTeamMember(ctx, campaignID, personID); err != nil {
		return fmt.Errorf("campaignService.RemoveTeamMember: %w", err)
	}

	s.audit(ctx, AuditParams{
		ActionType:  "DELETE",
		EntityType:  "campaign_team",
		EntityID:    &campaignID,
		Module:      "campaign",
		Description: "team member removed",
		OldValues:   map[string]any{"person_id": personID},
		Success:     true,
	})

	return nil
}

// audit writes an audit entry, logging (never failing the request) on error.
func (s *CampaignService) audit(ctx context.Context, params AuditParams) {
	if auditErr := s.auditSvc.LogAction(ctx, params); auditErr != nil {
		slog.ErrorContext(ctx, "campaignService: audit failed",
			"error", auditErr.Error(), "entity_type", params.EntityType, "action", params.ActionType,
		)
	}
}
