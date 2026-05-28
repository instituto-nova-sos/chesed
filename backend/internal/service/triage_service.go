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

// TriageRepository defines the interface for triage persistence.
type TriageRepository interface {
	Create(ctx context.Context, triage domain.Triage) (*domain.Triage, error)
	FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Triage, error)
	List(ctx context.Context, filter domain.TriageFilter) (*domain.TriageListResult, error)
	Update(ctx context.Context, triage domain.Triage) (*domain.Triage, error)
}

// CreateTriageInput holds validated input for triage creation.
type CreateTriageInput struct {
	PersonID              string   `json:"person_id" validate:"required,uuid"`
	MainComplaint         string   `json:"main_complaint" validate:"required,max=2000"`
	AssignedTeam          *string  `json:"assigned_team" validate:"omitempty,uuid"`
	TriageDate            *string  `json:"triage_date" validate:"omitempty"`
	Location              *string  `json:"location" validate:"omitempty,max=300"`
	Notes                 *string  `json:"notes" validate:"omitempty"`
	RequestedServiceTypes []string `json:"requested_service_types" validate:"omitempty,dive,uuid"`
}

// UpdateTriageInput holds validated input for triage update.
type UpdateTriageInput struct {
	MainComplaint         string   `json:"main_complaint" validate:"required,max=2000"`
	AssignedTeam          *string  `json:"assigned_team" validate:"omitempty,uuid"`
	Location              *string  `json:"location" validate:"omitempty,max=300"`
	Notes                 *string  `json:"notes" validate:"omitempty"`
	RequestedServiceTypes []string `json:"requested_service_types" validate:"omitempty,dive,uuid"`
}

// TriageService handles triage business logic.
type TriageService struct {
	repo     TriageRepository
	auditSvc *AuditService
}

// NewTriageService creates a new TriageService.
func NewTriageService(repo TriageRepository, auditSvc *AuditService) *TriageService {
	return &TriageService{repo: repo, auditSvc: auditSvc}
}

// CreateTriage creates a new triage.
func (s *TriageService) CreateTriage(ctx context.Context, input CreateTriageInput) (*domain.Triage, error) {
	if err := validate.Struct(input); err != nil {
		return nil, fmt.Errorf("triageService.CreateTriage: %w", err)
	}

	claims := auth.ClaimsFromContext(ctx)
	if claims.CampusID == uuid.Nil {
		return nil, fmt.Errorf("triageService.CreateTriage: %w", domain.ErrForbidden)
	}

	personID, err := uuid.Parse(input.PersonID)
	if err != nil {
		return nil, fmt.Errorf("triageService.CreateTriage: invalid person_id: %w", err)
	}

	triagedBy := parseUserID(claims.Subject)
	if triagedBy == nil {
		return nil, fmt.Errorf("triageService.CreateTriage: missing user identity")
	}

	triageDate, err := parseOptionalTime(input.TriageDate)
	if err != nil {
		return nil, fmt.Errorf("triageService.CreateTriage: invalid triage_date: %w", err)
	}
	if triageDate == nil {
		now := time.Now().UTC()
		triageDate = &now
	}

	requested, err := parseUUIDList(input.RequestedServiceTypes)
	if err != nil {
		return nil, fmt.Errorf("triageService.CreateTriage: %w", err)
	}

	assignedTeam, err := parseOptionalUUID(input.AssignedTeam)
	if err != nil {
		return nil, fmt.Errorf("triageService.CreateTriage: invalid assigned_team: %w", err)
	}

	triage := domain.Triage{
		ID:             uuid.New(),
		PersonID:       personID,
		CampusID:       claims.CampusID,
		MainComplaint:  input.MainComplaint,
		AssignedTeam:   assignedTeam,
		TriageDate:     *triageDate,
		Location:       input.Location,
		TriagedBy:      *triagedBy,
		Notes:          input.Notes,
		IsActive:       true,
		RequestedTypes: requested,
	}

	created, err := s.repo.Create(ctx, triage)
	if err != nil {
		return nil, fmt.Errorf("triageService.CreateTriage: %w", err)
	}

	if auditErr := s.auditSvc.LogAction(ctx, AuditParams{
		ActionType:  "CREATE",
		EntityType:  "triage",
		EntityID:    &created.ID,
		Module:      "triage",
		Description: "triage created",
		NewValues:   map[string]any{"person_id": created.PersonID, "main_complaint": created.MainComplaint},
		Success:     true,
	}); auditErr != nil {
		slog.ErrorContext(ctx, "triageService.CreateTriage: audit failed",
			"error", auditErr.Error(), "triage_id", created.ID,
		)
	}

	return created, nil
}

// GetTriage returns a triage by ID, scoped to campus.
func (s *TriageService) GetTriage(ctx context.Context, id uuid.UUID) (*domain.Triage, error) {
	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return nil, fmt.Errorf("triageService.GetTriage: %w", domain.ErrForbidden)
	}
	t, err := s.repo.FindByID(ctx, id, campusID)
	if err != nil {
		return nil, fmt.Errorf("triageService.GetTriage: %w", err)
	}
	return t, nil
}

// ListTriages returns a paginated list of triages for the campus.
func (s *TriageService) ListTriages(ctx context.Context, filter domain.TriageFilter) (*domain.TriageListResult, error) {
	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return nil, fmt.Errorf("triageService.ListTriages: %w", domain.ErrForbidden)
	}
	filter.CampusID = campusID
	result, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("triageService.ListTriages: %w", err)
	}
	return result, nil
}

// UpdateTriage updates mutable fields of a triage.
func (s *TriageService) UpdateTriage(ctx context.Context, id uuid.UUID, input UpdateTriageInput) (*domain.Triage, error) {
	if err := validate.Struct(input); err != nil {
		return nil, fmt.Errorf("triageService.UpdateTriage: %w", err)
	}

	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return nil, fmt.Errorf("triageService.UpdateTriage: %w", domain.ErrForbidden)
	}

	existing, err := s.repo.FindByID(ctx, id, campusID)
	if err != nil {
		return nil, fmt.Errorf("triageService.UpdateTriage: %w", err)
	}

	requested, err := parseUUIDList(input.RequestedServiceTypes)
	if err != nil {
		return nil, fmt.Errorf("triageService.UpdateTriage: %w", err)
	}

	assignedTeam, err := parseOptionalUUID(input.AssignedTeam)
	if err != nil {
		return nil, fmt.Errorf("triageService.UpdateTriage: invalid assigned_team: %w", err)
	}

	existing.MainComplaint = input.MainComplaint
	existing.AssignedTeam = assignedTeam
	existing.Location = input.Location
	existing.Notes = input.Notes
	if input.RequestedServiceTypes != nil {
		existing.RequestedTypes = requested
	}

	updated, err := s.repo.Update(ctx, *existing)
	if err != nil {
		return nil, fmt.Errorf("triageService.UpdateTriage: %w", err)
	}

	if auditErr := s.auditSvc.LogAction(ctx, AuditParams{
		ActionType:  "UPDATE",
		EntityType:  "triage",
		EntityID:    &updated.ID,
		Module:      "triage",
		Description: "triage updated",
		NewValues:   map[string]any{"main_complaint": updated.MainComplaint},
		Success:     true,
	}); auditErr != nil {
		slog.ErrorContext(ctx, "triageService.UpdateTriage: audit failed",
			"error", auditErr.Error(), "triage_id", updated.ID,
		)
	}

	return updated, nil
}

func parseOptionalTime(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, *s); err == nil {
			return &t, nil
		}
	}
	return nil, errors.New("unsupported time format")
}

func parseOptionalUUID(s *string) (*uuid.UUID, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(*s)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseUUIDList(items []string) ([]uuid.UUID, error) {
	if len(items) == 0 {
		return nil, nil
	}
	ids := make([]uuid.UUID, 0, len(items))
	for _, s := range items {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("invalid uuid in list: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
