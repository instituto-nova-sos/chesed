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

	existing, err := s.checkNotRegistered(ctx, claims.Subject)
	if err != nil {
		return nil, err
	}

	birthDate, campusID, err := s.validateRegistration(input)
	if err != nil {
		return nil, err
	}

	personID := uuid.New()
	person := buildRegistrationPerson(input, personID, claims.Email, birthDate, campusID)
	address := buildRegistrationAddress(input, personID)

	created, err := s.personRepo.Create(ctx, person, address)
	if err != nil {
		return nil, fmt.Errorf("selfRegisterService.Register: create person: %w", err)
	}

	s.createRegistrationRole(ctx, personID, campusID, input.RoleType)
	s.upsertAppUser(ctx, claims, existing, personID, campusID)
	s.logSelfRegistration(ctx, created, input.RoleType)

	return created, nil
}

// createRegistrationRole creates the person's role and, for a VOLUNTEER role, a
// PENDING agreement. Failures are logged, not fatal — the person already exists.
func (s *SelfRegisterService) createRegistrationRole(ctx context.Context, personID, campusID uuid.UUID, roleType string) {
	createdRole, roleErr := s.personRoleRepo.Create(ctx, domain.PersonRole{
		ID:       uuid.New(),
		PersonID: personID,
		RoleType: roleType,
		IsActive: true,
	})
	if roleErr != nil {
		slog.ErrorContext(ctx, "selfRegisterService.Register: create role failed",
			"error", roleErr.Error(), "person_id", personID,
		)
		return
	}

	if roleType != domain.RoleVolunteer {
		return
	}
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

// logSelfRegistration writes the audit entry for a completed self-registration.
func (s *SelfRegisterService) logSelfRegistration(ctx context.Context, created *domain.Person, roleType string) {
	if auditErr := s.auditSvc.LogAction(ctx, AuditParams{
		ActionType:  "CREATE",
		EntityType:  "person",
		EntityID:    &created.ID,
		Module:      "self-registration",
		Description: fmt.Sprintf("self-registered as %s", roleType),
		NewValues:   map[string]string{"full_name": created.FullName, "role_type": roleType},
		Success:     true,
	}); auditErr != nil {
		slog.ErrorContext(ctx, "selfRegisterService.Register: audit failed",
			"error", auditErr.Error(), "person_id", created.ID,
		)
	}
}

// checkNotRegistered ensures the Keycloak subject has no linked person yet.
// It returns the existing app_user (created on first login) when present.
func (s *SelfRegisterService) checkNotRegistered(ctx context.Context, subject string) (*domain.AppUser, error) {
	existing, err := s.userRepo.FindByKeycloakSubject(ctx, subject)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("selfRegisterService.Register: check existing: %w", err)
	}
	if existing != nil && existing.PersonID != nil {
		return nil, fmt.Errorf("selfRegisterService.Register: %w", domain.ErrDuplicate)
	}
	return existing, nil
}

// validateRegistration validates the document, birth date, and campus, returning
// the parsed birth date and campus ID.
func (s *SelfRegisterService) validateRegistration(input SelfRegisterInput) (*time.Time, uuid.UUID, error) {
	if err := validateDocument(input.DocumentType, input.DocumentNumber); err != nil {
		return nil, uuid.Nil, fmt.Errorf("selfRegisterService.Register: %w", err)
	}

	birthDate, err := parseOptionalDate(input.BirthDate)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("selfRegisterService.Register: invalid birth_date: %w", err)
	}

	campusID, err := uuid.Parse(input.CampusID)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("selfRegisterService.Register: invalid campus_id: %w", err)
	}
	return birthDate, campusID, nil
}

// buildRegistrationPerson maps a self-registration input into a domain.Person.
func buildRegistrationPerson(input SelfRegisterInput, personID uuid.UUID, email string, birthDate *time.Time, campusID uuid.UUID) domain.Person {
	nationality := "BRA"
	if input.Nationality != nil && *input.Nationality != "" {
		nationality = *input.Nationality
	}
	emailPtr := &email
	return domain.Person{
		ID:             personID,
		FullName:       input.FullName,
		BirthDate:      birthDate,
		DocumentType:   input.DocumentType,
		DocumentNumber: input.DocumentNumber,
		Nationality:    nationality,
		Gender:         input.Gender,
		Email:          emailPtr,
		Phone:          input.Phone,
		ReferralSource: input.ReferralSource,
		CampusID:       campusID,
		IsActive:       true,
	}
}

// buildRegistrationAddress maps the optional address input into a domain.Address.
func buildRegistrationAddress(input SelfRegisterInput, personID uuid.UUID) *domain.Address {
	if input.Address == nil {
		return nil
	}
	country := "BRA"
	if input.Address.Country != nil {
		country = *input.Address.Country
	}
	return &domain.Address{
		ID:           uuid.New(),
		PersonID:     personID,
		Street:       input.Address.Street,
		Number:       input.Address.Number,
		Complement:   input.Address.Complement,
		Neighborhood: input.Address.Neighborhood,
		City:         input.Address.City,
		State:        input.Address.State,
		ZipCode:      input.Address.ZipCode,
		Country:      country,
		IsPrimary:    true,
	}
}

// upsertAppUser links the person/campus to an existing app_user (created by
// AutoProvision on first login) or creates a new VOLUNTEER app_user. Failures
// are logged, not fatal — self-registration of the person already succeeded.
func (s *SelfRegisterService) upsertAppUser(ctx context.Context, claims auth.AuthClaims, existing *domain.AppUser, personID, campusID uuid.UUID) {
	if existing != nil {
		if err := s.userRepo.LinkPersonAndCampus(ctx, existing.ID, personID, campusID); err != nil {
			slog.ErrorContext(ctx, "selfRegisterService.upsertAppUser: link person and campus failed",
				"error", err.Error(), "user_id", existing.ID, "person_id", personID,
			)
			return
		}
		slog.InfoContext(ctx, "selfRegisterService.upsertAppUser: linked person and campus to existing app_user",
			"user_id", existing.ID, "person_id", personID, "campus_id", campusID,
		)
		return
	}

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
		slog.ErrorContext(ctx, "selfRegisterService.upsertAppUser: create app_user failed",
			"error", err.Error(), "person_id", personID,
		)
	}
}
