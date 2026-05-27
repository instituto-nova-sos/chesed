package service

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

//go:embed agreement_text_v1.md
var agreementTextV1 string

// VolunteerAgreementRepository defines the interface for volunteer agreement persistence.
type VolunteerAgreementRepository interface {
	Create(ctx context.Context, agreement domain.VolunteerAgreement) (*domain.VolunteerAgreement, error)
	FindByPersonID(ctx context.Context, personID uuid.UUID) ([]domain.VolunteerAgreement, error)
	FindByPersonRoleID(ctx context.Context, personRoleID uuid.UUID) (*domain.VolunteerAgreement, error)
	FindPendingByPersonID(ctx context.Context, personID uuid.UUID) (*domain.VolunteerAgreement, error)
	HasAcceptedAgreement(ctx context.Context, personID uuid.UUID) (bool, error)
	AcceptDigital(ctx context.Context, id uuid.UUID, userID uuid.UUID, ip string, userAgent string) (*domain.VolunteerAgreement, error)
	Reject(ctx context.Context, id uuid.UUID, reason *string) (*domain.VolunteerAgreement, error)
	AcceptManualUpload(ctx context.Context, id uuid.UUID, documentPath string, uploadedBy uuid.UUID) (*domain.VolunteerAgreement, error)
}

// VolunteerAgreementService handles volunteer agreement business logic.
type VolunteerAgreementService struct {
	agreementRepo  VolunteerAgreementRepository
	personRoleRepo PersonRoleRepository
	auditSvc       *AuditService
}

// NewVolunteerAgreementService creates a new VolunteerAgreementService.
func NewVolunteerAgreementService(
	agreementRepo VolunteerAgreementRepository,
	personRoleRepo PersonRoleRepository,
	auditSvc *AuditService,
) *VolunteerAgreementService {
	return &VolunteerAgreementService{
		agreementRepo:  agreementRepo,
		personRoleRepo: personRoleRepo,
		auditSvc:       auditSvc,
	}
}

// GetAgreementText returns the agreement text for the given version.
func (s *VolunteerAgreementService) GetAgreementText(version string) (string, string) {
	// Currently only v1 is supported
	return agreementTextV1, "v1"
}

// AcceptDigital digitally accepts the volunteer agreement for the authenticated user.
func (s *VolunteerAgreementService) AcceptDigital(ctx context.Context, personID uuid.UUID, ip string, userAgent string) (*domain.VolunteerAgreement, error) {
	claims := auth.ClaimsFromContext(ctx)

	// Check if already accepted
	accepted, err := s.agreementRepo.HasAcceptedAgreement(ctx, personID)
	if err != nil {
		return nil, fmt.Errorf("volunteerAgreementService.AcceptDigital: check: %w", err)
	}
	if accepted {
		return nil, fmt.Errorf("volunteerAgreementService.AcceptDigital: %w", domain.ErrAgreementExists)
	}

	// Find the pending agreement
	pending, err := s.agreementRepo.FindPendingByPersonID(ctx, personID)
	if err != nil {
		return nil, fmt.Errorf("volunteerAgreementService.AcceptDigital: find pending: %w", err)
	}

	userID := parseUserID(claims.Subject)
	var uid uuid.UUID
	if userID != nil {
		uid = *userID
	}

	result, err := s.agreementRepo.AcceptDigital(ctx, pending.ID, uid, ip, userAgent)
	if err != nil {
		return nil, fmt.Errorf("volunteerAgreementService.AcceptDigital: accept: %w", err)
	}

	if auditErr := s.auditSvc.LogAction(ctx, AuditParams{
		ActionType:  "UPDATE",
		EntityType:  "volunteer_agreement",
		EntityID:    &result.ID,
		Module:      "volunteer-agreement",
		Description: "volunteer agreement digitally accepted",
		IPAddress:   ip,
		UserAgent:   userAgent,
		NewValues:   map[string]string{"status": domain.AgreementAccepted, "signature_method": domain.SignatureDigital},
		Success:     true,
	}); auditErr != nil {
		slog.ErrorContext(ctx, "volunteerAgreementService.AcceptDigital: audit failed", "error", auditErr.Error())
	}

	return result, nil
}

// Reject rejects the volunteer agreement for the authenticated user.
func (s *VolunteerAgreementService) Reject(ctx context.Context, personID uuid.UUID, reason *string) (*domain.VolunteerAgreement, error) {
	pending, err := s.agreementRepo.FindPendingByPersonID(ctx, personID)
	if err != nil {
		return nil, fmt.Errorf("volunteerAgreementService.Reject: find pending: %w", err)
	}

	result, err := s.agreementRepo.Reject(ctx, pending.ID, reason)
	if err != nil {
		return nil, fmt.Errorf("volunteerAgreementService.Reject: reject: %w", err)
	}

	if auditErr := s.auditSvc.LogAction(ctx, AuditParams{
		ActionType:  "UPDATE",
		EntityType:  "volunteer_agreement",
		EntityID:    &result.ID,
		Module:      "volunteer-agreement",
		Description: "volunteer agreement rejected",
		NewValues:   map[string]string{"status": domain.AgreementRejected},
		Success:     true,
	}); auditErr != nil {
		slog.ErrorContext(ctx, "volunteerAgreementService.Reject: audit failed", "error", auditErr.Error())
	}

	return result, nil
}

// UploadManual uploads a manually signed agreement for a person.
func (s *VolunteerAgreementService) UploadManual(ctx context.Context, personID uuid.UUID, documentPath string) (*domain.VolunteerAgreement, error) {
	claims := auth.ClaimsFromContext(ctx)
	uploaderID := parseUserID(claims.Subject)
	var uid uuid.UUID
	if uploaderID != nil {
		uid = *uploaderID
	}

	// Find existing pending or create one if missing
	pending, err := s.agreementRepo.FindPendingByPersonID(ctx, personID)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("volunteerAgreementService.UploadManual: find pending: %w", err)
		}
		// Create a pending agreement for this person's VOLUNTEER role
		pending, err = s.createPendingForPerson(ctx, personID)
		if err != nil {
			return nil, fmt.Errorf("volunteerAgreementService.UploadManual: create pending: %w", err)
		}
	}

	result, err := s.agreementRepo.AcceptManualUpload(ctx, pending.ID, documentPath, uid)
	if err != nil {
		return nil, fmt.Errorf("volunteerAgreementService.UploadManual: accept: %w", err)
	}

	if auditErr := s.auditSvc.LogAction(ctx, AuditParams{
		ActionType:  "UPDATE",
		EntityType:  "volunteer_agreement",
		EntityID:    &result.ID,
		Module:      "volunteer-agreement",
		Description: "volunteer agreement manually uploaded",
		NewValues:   map[string]string{"status": domain.AgreementAccepted, "signature_method": domain.SignatureManualUpload},
		Success:     true,
	}); auditErr != nil {
		slog.ErrorContext(ctx, "volunteerAgreementService.UploadManual: audit failed", "error", auditErr.Error())
	}

	return result, nil
}

// GetByPersonID returns all agreements for a person.
func (s *VolunteerAgreementService) GetByPersonID(ctx context.Context, personID uuid.UUID) ([]domain.VolunteerAgreement, error) {
	agreements, err := s.agreementRepo.FindByPersonID(ctx, personID)
	if err != nil {
		return nil, fmt.Errorf("volunteerAgreementService.GetByPersonID: %w", err)
	}
	return agreements, nil
}

// CheckStatus returns whether a person has an accepted agreement.
func (s *VolunteerAgreementService) CheckStatus(ctx context.Context, personID uuid.UUID) (bool, error) {
	accepted, err := s.agreementRepo.HasAcceptedAgreement(ctx, personID)
	if err != nil {
		return false, fmt.Errorf("volunteerAgreementService.CheckStatus: %w", err)
	}
	return accepted, nil
}

// CreatePendingAgreement creates a PENDING agreement for a volunteer role.
func (s *VolunteerAgreementService) CreatePendingAgreement(ctx context.Context, personID uuid.UUID, personRoleID uuid.UUID, campusID uuid.UUID) (*domain.VolunteerAgreement, error) {
	agreement := domain.VolunteerAgreement{
		ID:               uuid.New(),
		PersonID:         personID,
		PersonRoleID:     personRoleID,
		CampusID:         campusID,
		Status:           domain.AgreementPending,
		AgreementVersion: "v1",
	}

	created, err := s.agreementRepo.Create(ctx, agreement)
	if err != nil {
		if errors.Is(err, domain.ErrAgreementExists) {
			return nil, nil // Already exists, idempotent
		}
		return nil, fmt.Errorf("volunteerAgreementService.CreatePendingAgreement: %w", err)
	}

	return created, nil
}

// createPendingForPerson finds the VOLUNTEER role and creates a pending agreement.
func (s *VolunteerAgreementService) createPendingForPerson(ctx context.Context, personID uuid.UUID) (*domain.VolunteerAgreement, error) {
	roles, err := s.personRoleRepo.FindByPersonID(ctx, personID)
	if err != nil {
		return nil, fmt.Errorf("find roles: %w", err)
	}

	for _, r := range roles {
		if r.RoleType == domain.RoleVolunteer && r.IsActive {
			campusID := auth.CampusIDFromContext(ctx)
			return s.CreatePendingAgreement(ctx, personID, r.ID, campusID)
		}
	}

	return nil, fmt.Errorf("no active VOLUNTEER role found for person %s", personID)
}
