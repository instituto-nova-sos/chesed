package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// CampusRepository defines the interface for campus persistence.
type CampusRepository interface {
	ListActive(ctx context.Context) ([]domain.Campus, error)
	List(ctx context.Context) ([]domain.Campus, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Campus, error)
	Create(ctx context.Context, campus domain.Campus) (*domain.Campus, error)
	Update(ctx context.Context, campus domain.Campus) (*domain.Campus, error)
}

// CampusService handles campus operations.
type CampusService struct {
	repo     CampusRepository
	auditSvc *AuditService
}

// NewCampusService creates a new CampusService.
func NewCampusService(repo CampusRepository, auditSvc *AuditService) *CampusService {
	return &CampusService{repo: repo, auditSvc: auditSvc}
}

// CreateCampusInput holds validated input for campus creation.
type CreateCampusInput struct {
	Name     string  `json:"name" validate:"required,max=150"`
	Region   string  `json:"region" validate:"required,oneof=BRAZIL USA EUROPE"`
	City     *string `json:"city" validate:"omitempty,max=100"`
	State    *string `json:"state" validate:"omitempty,max=100"`
	Country  string  `json:"country" validate:"required,len=3"`
	Timezone string  `json:"timezone" validate:"omitempty,max=64"`
}

// UpdateCampusInput holds validated input for campus update.
type UpdateCampusInput struct {
	Name     string  `json:"name" validate:"required,max=150"`
	Region   string  `json:"region" validate:"required,oneof=BRAZIL USA EUROPE"`
	City     *string `json:"city" validate:"omitempty,max=100"`
	State    *string `json:"state" validate:"omitempty,max=100"`
	Country  string  `json:"country" validate:"required,len=3"`
	Timezone string  `json:"timezone" validate:"omitempty,max=64"`
	IsActive bool    `json:"is_active"`
}

var campusValidate = validator.New()

// ListActive returns all active campuses.
func (s *CampusService) ListActive(ctx context.Context) ([]domain.Campus, error) {
	campuses, err := s.repo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("campusService.ListActive: %w", err)
	}
	return campuses, nil
}

// List returns all campuses including inactive ones.
func (s *CampusService) List(ctx context.Context) ([]domain.Campus, error) {
	campuses, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("campusService.List: %w", err)
	}
	return campuses, nil
}

// GetByID returns a campus by ID.
func (s *CampusService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Campus, error) {
	campus, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("campusService.GetByID: %w", err)
	}
	return campus, nil
}

// Create creates a new campus.
func (s *CampusService) Create(ctx context.Context, input CreateCampusInput) (*domain.Campus, error) {
	if err := campusValidate.Struct(input); err != nil {
		return nil, fmt.Errorf("campusService.Create: validation: %w", err)
	}

	campus := domain.Campus{
		ID:       uuid.New(),
		Name:     input.Name,
		Region:   input.Region,
		City:     input.City,
		State:    input.State,
		Country:  input.Country,
		Timezone: input.Timezone,
		IsActive: true,
	}

	created, err := s.repo.Create(ctx, campus)
	if err != nil {
		return nil, fmt.Errorf("campusService.Create: %w", err)
	}

	campusID := created.ID
	if auditErr := s.auditSvc.LogAction(ctx, AuditParams{
		ActionType:  "CREATE",
		EntityType:  "campus",
		EntityID:    &campusID,
		Module:      "campus",
		Description: "campus created",
		NewValues:   map[string]string{"name": created.Name, "region": created.Region},
		Success:     true,
	}); auditErr != nil {
		slog.ErrorContext(ctx, "campusService.Create: audit failed",
			"error", auditErr.Error(), "campus_id", campusID,
		)
	}

	return created, nil
}

// Update updates an existing campus.
func (s *CampusService) Update(ctx context.Context, id uuid.UUID, input UpdateCampusInput) (*domain.Campus, error) {
	if err := campusValidate.Struct(input); err != nil {
		return nil, fmt.Errorf("campusService.Update: validation: %w", err)
	}

	campus := domain.Campus{
		ID:       id,
		Name:     input.Name,
		Region:   input.Region,
		City:     input.City,
		State:    input.State,
		Country:  input.Country,
		Timezone: input.Timezone,
		IsActive: input.IsActive,
	}

	updated, err := s.repo.Update(ctx, campus)
	if err != nil {
		return nil, fmt.Errorf("campusService.Update: %w", err)
	}

	if auditErr := s.auditSvc.LogAction(ctx, AuditParams{
		ActionType:  "UPDATE",
		EntityType:  "campus",
		EntityID:    &id,
		Module:      "campus",
		Description: "campus updated",
		NewValues:   map[string]string{"name": updated.Name, "region": updated.Region},
		Success:     true,
	}); auditErr != nil {
		slog.ErrorContext(ctx, "campusService.Update: audit failed",
			"error", auditErr.Error(), "campus_id", id,
		)
	}

	return updated, nil
}
