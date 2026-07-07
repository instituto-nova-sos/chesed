package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// campaignStatusActive is the only campaign lifecycle state exposed publicly.
const campaignStatusActive = "ACTIVE"

// PublicCampaignRepository is the campus-scoped campaign list the public API
// exposes. It runs on the request's (public campus) transaction so the query
// is anchored to filter.CampusID and RLS is the backstop.
type PublicCampaignRepository interface {
	List(ctx context.Context, filter domain.CampaignFilter) (*domain.CampaignListResult, error)
}

// PublicPersonRepository creates the person record backing a public volunteer
// sign-up.
type PublicPersonRepository interface {
	Create(ctx context.Context, person domain.Person, address *domain.Address) (*domain.Person, error)
}

// PublicPersonRoleRepository assigns the VOLUNTEER role to a signed-up person.
type PublicPersonRoleRepository interface {
	Create(ctx context.Context, role domain.PersonRole) (*domain.PersonRole, error)
}

// PublicAgreementRepository creates the PENDING volunteer agreement a public
// sign-up must complete later (in-person or digitally).
type PublicAgreementRepository interface {
	Create(ctx context.Context, agreement domain.VolunteerAgreement) (*domain.VolunteerAgreement, error)
}

// PublicVolunteerInput holds validated input for an unauthenticated volunteer
// sign-up submitted from the public (WordPress) site.
type PublicVolunteerInput struct {
	FullName       string  `json:"full_name" validate:"required,max=200"`
	Email          *string `json:"email" validate:"omitempty,email,max=255"`
	Phone          *string `json:"phone" validate:"omitempty,max=30"`
	BirthDate      *string `json:"birth_date" validate:"omitempty"`
	ReferralSource *string `json:"referral_source" validate:"omitempty,max=200"`
	CampusID       string  `json:"campus_id" validate:"required,uuid"`
}

// PublicService serves the unauthenticated WordPress-facing API: listing active
// campaigns and accepting volunteer sign-ups. It has NO Keycloak/app_user
// coupling — a public sign-up produces only a person, a VOLUNTEER role, and a
// PENDING agreement for staff to follow up on.
type PublicService struct {
	campaignRepo  PublicCampaignRepository
	personRepo    PublicPersonRepository
	roleRepo      PublicPersonRoleRepository
	agreementRepo PublicAgreementRepository
	auditSvc      *AuditService
}

// NewPublicService creates a new PublicService.
func NewPublicService(
	campaignRepo PublicCampaignRepository,
	personRepo PublicPersonRepository,
	roleRepo PublicPersonRoleRepository,
	agreementRepo PublicAgreementRepository,
	auditSvc *AuditService,
) *PublicService {
	return &PublicService{
		campaignRepo:  campaignRepo,
		personRepo:    personRepo,
		roleRepo:      roleRepo,
		agreementRepo: agreementRepo,
		auditSvc:      auditSvc,
	}
}

// ListActiveCampaigns returns the paginated list of ACTIVE campaigns for the
// campus. The campus is supplied by the caller (resolved and validated by the
// PublicCampusValidator middleware), never derived from user input here.
func (s *PublicService) ListActiveCampaigns(ctx context.Context, campusID uuid.UUID, page, perPage int) (*domain.CampaignListResult, error) {
	active := campaignStatusActive
	filter := domain.CampaignFilter{
		CampusID: campusID,
		Status:   &active,
		Page:     page,
		PerPage:  perPage,
	}
	result, err := s.campaignRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("publicService.ListActiveCampaigns: %w", err)
	}
	return result, nil
}

// RegisterVolunteer registers a volunteer from the public site: it creates the
// person, assigns the VOLUNTEER role, and opens a PENDING agreement. The audit
// entry is campus-scoped with no actor (public, unauthenticated) and records
// the client IP/user-agent.
func (s *PublicService) RegisterVolunteer(ctx context.Context, input PublicVolunteerInput, ip, userAgent string) (*domain.Person, error) {
	campusID, birthDate, err := validatePublicVolunteer(input)
	if err != nil {
		return nil, err
	}

	person := buildPublicVolunteerPerson(input, campusID, birthDate)
	created, err := s.personRepo.Create(ctx, person, nil)
	if err != nil {
		return nil, fmt.Errorf("publicService.RegisterVolunteer: create person: %w", err)
	}

	s.createVolunteerRoleAndAgreement(ctx, created.ID, campusID)
	s.auditPublicSignup(ctx, created, campusID, ip, userAgent)

	return created, nil
}

// validatePublicVolunteer runs struct validation and parses the campus id and
// optional birth date. Every failure maps to ErrValidation so the handler can
// respond with a generic 400.
func validatePublicVolunteer(input PublicVolunteerInput) (uuid.UUID, *time.Time, error) {
	if err := validate.Struct(input); err != nil {
		return uuid.Nil, nil, fmt.Errorf("publicService.RegisterVolunteer: %w: %w", domain.ErrValidation, err)
	}
	campusID, err := uuid.Parse(input.CampusID)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("publicService.RegisterVolunteer: invalid campus_id: %w", domain.ErrValidation)
	}
	birthDate, err := parseOptionalDate(input.BirthDate)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("publicService.RegisterVolunteer: invalid birth_date: %w", domain.ErrValidation)
	}
	return campusID, birthDate, nil
}

// buildPublicVolunteerPerson maps sign-up input into a domain.Person. Public
// sign-ups have no identity document; DocumentType defaults to OTHER.
func buildPublicVolunteerPerson(input PublicVolunteerInput, campusID uuid.UUID, birthDate *time.Time) domain.Person {
	return domain.Person{
		ID:             uuid.New(),
		FullName:       input.FullName,
		BirthDate:      birthDate,
		DocumentType:   "OTHER",
		Nationality:    "BRA",
		Email:          input.Email,
		Phone:          input.Phone,
		ReferralSource: input.ReferralSource,
		CampusID:       campusID,
		IsActive:       true,
	}
}

// createVolunteerRoleAndAgreement assigns the VOLUNTEER role and opens a PENDING
// agreement. Failures are logged, not fatal — the person already exists and
// staff can reconcile the role/agreement during follow-up.
func (s *PublicService) createVolunteerRoleAndAgreement(ctx context.Context, personID, campusID uuid.UUID) {
	role, err := s.roleRepo.Create(ctx, domain.PersonRole{
		ID:       uuid.New(),
		PersonID: personID,
		RoleType: domain.RoleVolunteer,
		IsActive: true,
	})
	if err != nil {
		slog.ErrorContext(ctx, "publicService.RegisterVolunteer: create role failed",
			"error", err.Error(), "person_id", personID)
		return
	}

	agreement := domain.VolunteerAgreement{
		ID:               uuid.New(),
		PersonID:         personID,
		PersonRoleID:     role.ID,
		CampusID:         campusID,
		Status:           domain.AgreementPending,
		AgreementVersion: "v1",
	}
	if _, err := s.agreementRepo.Create(ctx, agreement); err != nil {
		slog.ErrorContext(ctx, "publicService.RegisterVolunteer: create pending agreement failed",
			"error", err.Error(), "person_id", personID)
	}
}

// auditPublicSignup writes the audit entry for a public sign-up. There are no
// auth claims on a public request, so the campus is synthesized into the
// context (empty Subject => nil actor) before logging.
func (s *PublicService) auditPublicSignup(ctx context.Context, person *domain.Person, campusID uuid.UUID, ip, userAgent string) {
	auditCtx := auth.NewContext(ctx, auth.AuthClaims{CampusID: campusID})
	if err := s.auditSvc.LogAction(auditCtx, AuditParams{
		ActionType:  "CREATE",
		EntityType:  "person",
		EntityID:    &person.ID,
		Module:      "public-api",
		Description: "public volunteer sign-up",
		IPAddress:   ip,
		UserAgent:   userAgent,
		NewValues:   map[string]string{"full_name": person.FullName, "role_type": domain.RoleVolunteer},
		Success:     true,
	}); err != nil {
		slog.ErrorContext(ctx, "publicService.RegisterVolunteer: audit failed",
			"error", err.Error(), "person_id", person.ID)
	}
}
