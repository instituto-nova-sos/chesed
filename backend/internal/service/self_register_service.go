package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/utils"
)

// SelfRegisterInput holds validated input for self-registration.
type SelfRegisterInput struct {
	FullName       string        `json:"full_name" validate:"required,max=200"`
	BirthDate      *string       `json:"birth_date" validate:"omitempty"`
	DocumentType   string        `json:"document_type" validate:"required,oneof=CPF RG SSN EU_ID PASSPORT OTHER"`
	DocumentNumber *string       `json:"document_number" validate:"omitempty,max=30"`
	Nationality    *string       `json:"nationality" validate:"omitempty,len=3"`
	Gender         *string       `json:"gender" validate:"omitempty,oneof=M F OTHER PREFER_NOT_TO_SAY"`
	Phone          *string       `json:"phone" validate:"omitempty,max=30"`
	ReferralSource *string       `json:"referral_source" validate:"omitempty,max=200"`
	Address        *AddressInput `json:"address" validate:"omitempty"`
	RoleType       string        `json:"role_type" validate:"required,oneof=VOLUNTEER ASSISTED"`
	CampusID       string        `json:"campus_id" validate:"required,uuid"`
}

// SelfRegisterService handles self-registration for new users.
type SelfRegisterService struct {
	personRepo     PersonRepository
	personRoleRepo PersonRoleRepository
	userRepo       UserRepository
	agreementRepo  VolunteerAgreementRepository
	auditSvc       *AuditService
}

// NewSelfRegisterService creates a new SelfRegisterService.
func NewSelfRegisterService(
	personRepo PersonRepository,
	personRoleRepo PersonRoleRepository,
	userRepo UserRepository,
	agreementRepo VolunteerAgreementRepository,
	auditSvc *AuditService,
) *SelfRegisterService {
	return &SelfRegisterService{
		personRepo:     personRepo,
		personRoleRepo: personRoleRepo,
		userRepo:       userRepo,
		agreementRepo:  agreementRepo,
		auditSvc:       auditSvc,
	}
}

// Register creates a person, assigns a role, and links to the authenticated user.
func (s *SelfRegisterService) Register(ctx context.Context, input SelfRegisterInput) (*domain.Person, error) {
	if err := validate.Struct(input); err != nil {
		return nil, fmt.Errorf("selfRegisterService.Register: %w", err)
	}

	claims := auth.ClaimsFromContext(ctx)

	// Check user is not already registered
	existing, err := s.userRepo.FindByKeycloakSubject(ctx, claims.Subject)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("selfRegisterService.Register: check existing: %w", err)
	}
	if existing != nil && existing.PersonID != nil {
		return nil, fmt.Errorf("selfRegisterService.Register: %w", domain.ErrDuplicate)
	}

	if input.DocumentType == "CPF" && input.DocumentNumber != nil && *input.DocumentNumber != "" {
		if !utils.ValidateCPF(*input.DocumentNumber) {
			return nil, fmt.Errorf("selfRegisterService.Register: %w", domain.ErrInvalidCPF)
		}
	}

	birthDate, err := parseOptionalDate(input.BirthDate)
	if err != nil {
		return nil, fmt.Errorf("selfRegisterService.Register: invalid birth_date: %w", err)
	}

	campusID, err := uuid.Parse(input.CampusID)
	if err != nil {
		return nil, fmt.Errorf("selfRegisterService.Register: invalid campus_id: %w", err)
	}

	nationality := "BRA"
	if input.Nationality != nil && *input.Nationality != "" {
		nationality = *input.Nationality
	}

	personID := uuid.New()
	email := &claims.Email

	person := domain.Person{
		ID:             personID,
		FullName:       input.FullName,
		BirthDate:      birthDate,
		DocumentType:   input.DocumentType,
		DocumentNumber: input.DocumentNumber,
		Nationality:    nationality,
		Gender:         input.Gender,
		Email:          email,
		Phone:          input.Phone,
		ReferralSource: input.ReferralSource,
		CampusID:       campusID,
		IsActive:       true,
	}

	var address *domain.Address
	if input.Address != nil {
		country := "BRA"
		if input.Address.Country != nil {
			country = *input.Address.Country
		}
		address = &domain.Address{
			ID:        uuid.New(),
			PersonID:  personID,
			Street:    input.Address.Street,
			Number:    input.Address.Number,
			Complement: input.Address.Complement,
			Neighborhood: input.Address.Neighborhood,
			City:      input.Address.City,
			State:     input.Address.State,
			ZipCode:   input.Address.ZipCode,
			Country:   country,
			IsPrimary: true,
		}
	}

	created, err := s.personRepo.Create(ctx, person, address)
	if err != nil {
		return nil, fmt.Errorf("selfRegisterService.Register: create person: %w", err)
	}

	// Create person role
	role := domain.PersonRole{
		ID:       uuid.New(),
		PersonID: personID,
		RoleType: input.RoleType,
		IsActive: true,
	}
	createdRole, roleErr := s.personRoleRepo.Create(ctx, role)
	if roleErr != nil {
		slog.ErrorContext(ctx, "selfRegisterService.Register: create role failed",
			"error", roleErr.Error(), "person_id", personID,
		)
	}

	// Create pending volunteer agreement if registering as VOLUNTEER
	if input.RoleType == domain.RoleVolunteer && createdRole != nil {
		agreement := domain.VolunteerAgreement{
			ID:               uuid.New(),
			PersonID:         personID,
			PersonRoleID:     createdRole.ID,
			CampusID:         campusID,
			Status:           domain.AgreementPending,
			AgreementVersion: "v1",
		}
		if _, agErr := s.agreementRepo.Create(ctx, agreement); agErr != nil {
			slog.ErrorContext(ctx, "selfRegisterService.Register: create pending agreement failed",
				"error", agErr.Error(), "person_id", personID,
			)
		}
	}

	// Create or update app_user with person link
	if existing != nil {
		// User exists (from AutoProvision on first login) — link person and set campus
		if err := s.userRepo.LinkPersonAndCampus(ctx, existing.ID, personID, campusID); err != nil {
			slog.ErrorContext(ctx, "selfRegisterService.Register: link person and campus failed",
				"error", err.Error(), "user_id", existing.ID, "person_id", personID,
			)
		} else {
			slog.InfoContext(ctx, "selfRegisterService.Register: linked person and campus to existing app_user",
				"user_id", existing.ID, "person_id", personID, "campus_id", campusID,
			)
		}
	} else {
		// Create new app_user
		newUser := domain.AppUser{
			ID:                uuid.New(),
			PersonID:          &personID,
			Email:             claims.Email,
			KeycloakSubjectID: claims.Subject,
			AccessProfile:     domain.RoleVolunteer,
			CampusID:          &campusID,
			IsActive:          true,
		}
		if _, err := s.userRepo.Create(ctx, newUser); err != nil {
			slog.ErrorContext(ctx, "selfRegisterService.Register: create app_user failed",
				"error", err.Error(), "person_id", personID,
			)
		}
	}

	// Audit log
	if auditErr := s.auditSvc.LogAction(ctx, AuditParams{
		ActionType:  "CREATE",
		EntityType:  "person",
		EntityID:    &created.ID,
		Module:      "self-registration",
		Description: fmt.Sprintf("self-registered as %s", input.RoleType),
		NewValues:   map[string]string{"full_name": created.FullName, "role_type": input.RoleType},
		Success:     true,
	}); auditErr != nil {
		slog.ErrorContext(ctx, "selfRegisterService.Register: audit failed",
			"error", auditErr.Error(), "person_id", created.ID,
		)
	}

	return created, nil
}
