package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// ConsentRepository defines the interface for consent persistence.
type ConsentRepository interface {
	Create(ctx context.Context, c domain.Consent) (*domain.Consent, error)
	FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Consent, error)
	ListByPerson(ctx context.Context, personID, campusID uuid.UUID) ([]domain.Consent, error)
	Revoke(ctx context.Context, id, campusID uuid.UUID, reason string) (*domain.Consent, error)
}

// ConsentPersonRepository is the campus-scoped person lookup consents need to
// validate the person and guardian references without leaking foreign-campus
// existence (threat model T3).
type ConsentPersonRepository interface {
	FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Person, error)
}

// CreateConsentInput holds validated input for consent creation.
type CreateConsentInput struct {
	PersonID          string  `json:"person_id" validate:"required,uuid"`
	ConsentType       string  `json:"consent_type" validate:"required,oneof=DATA_PROCESSING IMAGE_USAGE HEALTH_DATA MINOR_GUARDIAN"`
	Purpose           string  `json:"purpose" validate:"required"`
	ConsentVersion    string  `json:"consent_version" validate:"omitempty,max=20"`
	GrantedByPersonID *string `json:"granted_by_person_id" validate:"omitempty,uuid"`
	SignatureData     *string `json:"signature_data" validate:"omitempty"`
}

// RevokeConsentInput holds validated input for consent revocation.
type RevokeConsentInput struct {
	RevokedReason string `json:"revoked_reason" validate:"required"`
}

// ConsentService handles consent business logic.
type ConsentService struct {
	repo       ConsentRepository
	personRepo ConsentPersonRepository
	auditSvc   *AuditService
}

// NewConsentService creates a new ConsentService.
func NewConsentService(repo ConsentRepository, personRepo ConsentPersonRepository, auditSvc *AuditService) *ConsentService {
	return &ConsentService{repo: repo, personRepo: personRepo, auditSvc: auditSvc}
}

// CreateConsent registers a new active consent for a person in the caller's
// campus.
func (s *ConsentService) CreateConsent(ctx context.Context, input CreateConsentInput) (*domain.Consent, error) {
	if err := validate.Struct(input); err != nil {
		return nil, fmt.Errorf("consentService.CreateConsent: %w", err)
	}

	claims := auth.ClaimsFromContext(ctx)
	if claims.CampusID == uuid.Nil {
		return nil, fmt.Errorf("consentService.CreateConsent: %w", domain.ErrForbidden)
	}

	consent, err := s.buildConsentFromInput(ctx, input, claims.CampusID)
	if err != nil {
		return nil, fmt.Errorf("consentService.CreateConsent: %w", err)
	}

	created, err := s.repo.Create(ctx, consent)
	if err != nil {
		return nil, fmt.Errorf("consentService.CreateConsent: %w", err)
	}

	s.audit(ctx, AuditParams{
		ActionType:  "CREATE",
		EntityType:  "consent",
		EntityID:    &created.ID,
		Module:      "consent",
		Description: "consent created",
		NewValues: map[string]any{
			"person_id":       created.PersonID,
			"consent_type":    created.ConsentType,
			"consent_version": created.ConsentVersion,
			"is_active":       created.IsActive,
		},
		Success: true,
	})

	return created, nil
}

// buildConsentFromInput resolves the person and guardian references against
// the caller's campus and assembles a domain.Consent with defaults applied.
func (s *ConsentService) buildConsentFromInput(ctx context.Context, input CreateConsentInput, campusID uuid.UUID) (domain.Consent, error) {
	personID, err := s.resolvePersonRef(ctx, input.PersonID, campusID, "person")
	if err != nil {
		return domain.Consent{}, err
	}

	var grantedBy *uuid.UUID
	if input.GrantedByPersonID != nil && *input.GrantedByPersonID != "" {
		id, err := s.resolvePersonRef(ctx, *input.GrantedByPersonID, campusID, "granted_by_person")
		if err != nil {
			return domain.Consent{}, err
		}
		grantedBy = &id
	}

	version := input.ConsentVersion
	if version == "" {
		version = "1.0"
	}

	return domain.Consent{
		ID:              uuid.New(),
		PersonID:        personID,
		ConsentType:     input.ConsentType,
		ConsentVersion:  version,
		Purpose:         input.Purpose,
		GrantedByPerson: grantedBy,
		SignatureData:   input.SignatureData,
		IsActive:        true,
		CampusID:        campusID,
	}, nil
}

// resolvePersonRef validates a person reference against the caller's campus.
// Rejections are generic (ErrValidation) so foreign-campus existence is never
// disclosed (threat model T3).
func (s *ConsentService) resolvePersonRef(ctx context.Context, ref string, campusID uuid.UUID, label string) (uuid.UUID, error) {
	id, err := uuid.Parse(ref)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s_id: %w", label, domain.ErrValidation)
	}
	if _, err := s.personRepo.FindByID(ctx, id, campusID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return uuid.Nil, fmt.Errorf("%s reference not resolvable: %w", label, domain.ErrValidation)
		}
		return uuid.Nil, fmt.Errorf("resolve %s: %w", label, err)
	}
	return id, nil
}

// ListPersonConsents returns the full consent registry (active and revoked) of
// a person visible in the caller's campus, newest grant first.
func (s *ConsentService) ListPersonConsents(ctx context.Context, personID uuid.UUID) ([]domain.Consent, error) {
	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return nil, fmt.Errorf("consentService.ListPersonConsents: %w", domain.ErrForbidden)
	}

	if _, err := s.personRepo.FindByID(ctx, personID, campusID); err != nil {
		return nil, fmt.Errorf("consentService.ListPersonConsents: %w", err)
	}

	consents, err := s.repo.ListByPerson(ctx, personID, campusID)
	if err != nil {
		return nil, fmt.Errorf("consentService.ListPersonConsents: %w", err)
	}
	if consents == nil {
		consents = []domain.Consent{}
	}
	return consents, nil
}

// RevokeConsent deactivates an active consent recording the reason, keeping
// the row as the registry trail.
func (s *ConsentService) RevokeConsent(ctx context.Context, id uuid.UUID, input RevokeConsentInput) (*domain.Consent, error) {
	if err := validate.Struct(input); err != nil {
		return nil, fmt.Errorf("consentService.RevokeConsent: %w", err)
	}

	claims := auth.ClaimsFromContext(ctx)
	if claims.CampusID == uuid.Nil {
		return nil, fmt.Errorf("consentService.RevokeConsent: %w", domain.ErrForbidden)
	}

	existing, err := s.repo.FindByID(ctx, id, claims.CampusID)
	if err != nil {
		return nil, fmt.Errorf("consentService.RevokeConsent: %w", err)
	}
	if !existing.IsActive {
		return nil, fmt.Errorf("consentService.RevokeConsent: consent already revoked: %w", domain.ErrValidation)
	}

	revoked, err := s.repo.Revoke(ctx, id, claims.CampusID, input.RevokedReason)
	if err != nil {
		return nil, fmt.Errorf("consentService.RevokeConsent: %w", err)
	}

	s.audit(ctx, AuditParams{
		ActionType:  "UPDATE",
		EntityType:  "consent",
		EntityID:    &revoked.ID,
		Module:      "consent",
		Description: "consent revoked",
		OldValues:   map[string]any{"is_active": true, "revoked_at": existing.RevokedAt, "revoked_reason": existing.RevokedReason},
		NewValues:   map[string]any{"is_active": false, "revoked_at": revoked.RevokedAt, "revoked_reason": revoked.RevokedReason},
		Success:     true,
	})

	return revoked, nil
}

// audit writes an audit entry, logging (never failing the request) on error.
func (s *ConsentService) audit(ctx context.Context, params AuditParams) {
	if auditErr := s.auditSvc.LogAction(ctx, params); auditErr != nil {
		slog.ErrorContext(ctx, "consentService: audit failed",
			"error", auditErr.Error(), "entity_type", params.EntityType, "action", params.ActionType,
		)
	}
}
