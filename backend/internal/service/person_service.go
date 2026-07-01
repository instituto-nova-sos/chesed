package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/utils"
)

// PersonRepository defines the interface for person persistence.
type PersonRepository interface {
	Create(ctx context.Context, person domain.Person, address *domain.Address) (*domain.Person, error)
	FindByID(ctx context.Context, id uuid.UUID, campusID uuid.UUID) (*domain.Person, error)
	FindByEmail(ctx context.Context, email string, campusID uuid.UUID) (*domain.Person, error)
	FindByEmailGlobal(ctx context.Context, email string) ([]domain.Person, error)
	FindByIDWithDetails(ctx context.Context, id uuid.UUID, campusID uuid.UUID) (*domain.Person, []domain.Address, []domain.PersonRole, error)
	Update(ctx context.Context, person domain.Person) (*domain.Person, error)
	List(ctx context.Context, filter domain.PersonFilter) (*domain.PersonListResult, error)
	CheckDuplicate(ctx context.Context, documentType, documentNumber string, campusID uuid.UUID) (*domain.DuplicateCheckResult, error)
	UpdateAddress(ctx context.Context, personID uuid.UUID, address domain.Address) (*domain.Address, error)
}

// PersonRoleRepository defines the interface for person role persistence.
type PersonRoleRepository interface {
	Create(ctx context.Context, role domain.PersonRole) (*domain.PersonRole, error)
	FindByPersonID(ctx context.Context, personID uuid.UUID) ([]domain.PersonRole, error)
	ToggleActive(ctx context.Context, roleID uuid.UUID, isActive bool, userID *uuid.UUID) (*domain.PersonRole, error)
}

var validate = validator.New()

// CreatePersonInput holds validated input for person creation.
type CreatePersonInput struct {
	FullName       string        `json:"full_name" validate:"required,max=200"`
	BirthDate      *string       `json:"birth_date" validate:"omitempty"`
	DocumentType   string        `json:"document_type" validate:"required,oneof=CPF RG SSN EU_ID PASSPORT OTHER"`
	DocumentNumber *string       `json:"document_number" validate:"omitempty,max=30"`
	Nationality    *string       `json:"nationality" validate:"omitempty,len=3"`
	Gender         *string       `json:"gender" validate:"omitempty,oneof=M F OTHER PREFER_NOT_TO_SAY"`
	Email          *string       `json:"email" validate:"omitempty,email,max=255"`
	Phone          *string       `json:"phone" validate:"omitempty,max=30"`
	ReferralSource *string       `json:"referral_source" validate:"omitempty,max=200"`
	Address        *AddressInput `json:"address" validate:"omitempty"`
	SyncID         *string       `json:"sync_id" validate:"omitempty,uuid"`
}

// UpdatePersonInput holds validated input for person update.
type UpdatePersonInput struct {
	FullName       string        `json:"full_name" validate:"required,max=200"`
	BirthDate      *string       `json:"birth_date" validate:"omitempty"`
	DocumentType   string        `json:"document_type" validate:"required,oneof=CPF RG SSN EU_ID PASSPORT OTHER"`
	DocumentNumber *string       `json:"document_number" validate:"omitempty,max=30"`
	Nationality    *string       `json:"nationality" validate:"omitempty,len=3"`
	Gender         *string       `json:"gender" validate:"omitempty,oneof=M F OTHER PREFER_NOT_TO_SAY"`
	Email          *string       `json:"email" validate:"omitempty,email,max=255"`
	Phone          *string       `json:"phone" validate:"omitempty,max=30"`
	ReferralSource *string       `json:"referral_source" validate:"omitempty,max=200"`
	Address        *AddressInput `json:"address" validate:"omitempty"`
}

// AddressInput holds address data for person creation/update.
type AddressInput struct {
	Street       *string `json:"street" validate:"omitempty,max=300"`
	Number       *string `json:"number" validate:"omitempty,max=20"`
	Complement   *string `json:"complement" validate:"omitempty,max=100"`
	Neighborhood *string `json:"neighborhood" validate:"omitempty,max=100"`
	City         *string `json:"city" validate:"omitempty,max=100"`
	State        *string `json:"state" validate:"omitempty,max=100"`
	ZipCode      *string `json:"zip_code" validate:"omitempty,max=20"`
	Country      *string `json:"country" validate:"omitempty,max=3"`
}

// AddRoleInput holds validated input for adding a role to a person.
type AddRoleInput struct {
	RoleType              string  `json:"role_type" validate:"required,oneof=VOLUNTEER ASSISTED PROFESSIONAL COORDINATOR ADMIN"`
	ProfessionalSpecialty *string `json:"professional_specialty" validate:"omitempty,max=100"`
	Notes                 *string `json:"notes" validate:"omitempty"`
}

// PersonService handles person business logic.
type PersonService struct {
	personRepo     PersonRepository
	personRoleRepo PersonRoleRepository
	agreementRepo  VolunteerAgreementRepository
	auditSvc       *AuditService
}

// NewPersonService creates a new PersonService.
func NewPersonService(
	personRepo PersonRepository,
	personRoleRepo PersonRoleRepository,
	agreementRepo VolunteerAgreementRepository,
	auditSvc *AuditService,
) *PersonService {
	return &PersonService{
		personRepo:     personRepo,
		personRoleRepo: personRoleRepo,
		agreementRepo:  agreementRepo,
		auditSvc:       auditSvc,
	}
}

// CreatePerson creates a new person with optional address.
func (s *PersonService) CreatePerson(ctx context.Context, input CreatePersonInput) (*domain.Person, error) {
	if err := validate.Struct(input); err != nil {
		return nil, fmt.Errorf("personService.CreatePerson: %w", err)
	}

	claims := auth.ClaimsFromContext(ctx)

	person, err := buildPersonFromInput(input, claims)
	if err != nil {
		return nil, err
	}
	address := buildPersonAddress(input.Address, person.ID)

	created, err := s.personRepo.Create(ctx, person, address)
	if err != nil {
		return nil, fmt.Errorf("personService.CreatePerson: %w", err)
	}

	if auditErr := s.auditSvc.LogAction(ctx, AuditParams{
		ActionType:  "CREATE",
		EntityType:  "person",
		EntityID:    &created.ID,
		Module:      "person-management",
		Description: "person created",
		NewValues:   map[string]string{"full_name": created.FullName, "document_type": created.DocumentType},
		Success:     true,
	}); auditErr != nil {
		slog.ErrorContext(ctx, "personService.CreatePerson: audit failed",
			"error", auditErr.Error(), "person_id", created.ID,
		)
	}

	return created, nil
}

// buildPersonFromInput parses and maps a create input into a domain.Person,
// honoring an optional client-supplied SyncID and validating the CPF.
func buildPersonFromInput(input CreatePersonInput, claims auth.AuthClaims) (domain.Person, error) {
	personID := uuid.New()
	if input.SyncID != nil {
		if parsed, err := uuid.Parse(*input.SyncID); err == nil {
			personID = parsed
		}
	}

	if input.DocumentType == "CPF" && input.DocumentNumber != nil && *input.DocumentNumber != "" {
		if !utils.ValidateCPF(*input.DocumentNumber) {
			return domain.Person{}, fmt.Errorf("personService.CreatePerson: %w", domain.ErrInvalidCPF)
		}
	}

	birthDate, err := parseOptionalDate(input.BirthDate)
	if err != nil {
		return domain.Person{}, fmt.Errorf("personService.CreatePerson: invalid birth_date: %w", err)
	}

	nationality := "BRA"
	if input.Nationality != nil && *input.Nationality != "" {
		nationality = *input.Nationality
	}

	return domain.Person{
		ID:             personID,
		FullName:       input.FullName,
		BirthDate:      birthDate,
		DocumentType:   input.DocumentType,
		DocumentNumber: input.DocumentNumber,
		Nationality:    nationality,
		Gender:         input.Gender,
		Email:          input.Email,
		Phone:          input.Phone,
		ReferralSource: input.ReferralSource,
		CampusID:       claims.CampusID,
		IsActive:       true,
		CreatedBy:      parseUserID(claims.Subject),
	}, nil
}

// applyPersonUpdate validates the document and birth date, then returns a copy
// of old with the editable fields overwritten from input.
func applyPersonUpdate(old *domain.Person, input UpdatePersonInput) (domain.Person, error) {
	if input.DocumentType == "CPF" && input.DocumentNumber != nil && *input.DocumentNumber != "" {
		if !utils.ValidateCPF(*input.DocumentNumber) {
			return domain.Person{}, fmt.Errorf("personService.UpdatePerson: %w", domain.ErrInvalidCPF)
		}
	}

	birthDate, err := parseOptionalDate(input.BirthDate)
	if err != nil {
		return domain.Person{}, fmt.Errorf("personService.UpdatePerson: invalid birth_date: %w", err)
	}

	nationality := old.Nationality
	if input.Nationality != nil && *input.Nationality != "" {
		nationality = *input.Nationality
	}

	updated := *old
	updated.FullName = input.FullName
	updated.BirthDate = birthDate
	updated.DocumentType = input.DocumentType
	updated.DocumentNumber = input.DocumentNumber
	updated.Nationality = nationality
	updated.Gender = input.Gender
	updated.Email = input.Email
	updated.Phone = input.Phone
	updated.ReferralSource = input.ReferralSource
	return updated, nil
}

// buildPersonAddress maps an optional address input into a primary domain.Address.
func buildPersonAddress(in *AddressInput, personID uuid.UUID) *domain.Address {
	if in == nil {
		return nil
	}
	country := "BRA"
	if in.Country != nil {
		country = *in.Country
	}
	return &domain.Address{
		ID:           uuid.New(),
		PersonID:     personID,
		Street:       in.Street,
		Number:       in.Number,
		Complement:   in.Complement,
		Neighborhood: in.Neighborhood,
		City:         in.City,
		State:        in.State,
		ZipCode:      in.ZipCode,
		Country:      country,
		IsPrimary:    true,
	}
}

// GetPerson returns a person with details by ID.
func (s *PersonService) GetPerson(ctx context.Context, id uuid.UUID) (*domain.PersonDetail, error) {
	campusID := auth.CampusIDFromContext(ctx)

	person, addresses, roles, err := s.personRepo.FindByIDWithDetails(ctx, id, campusID)
	if err != nil {
		return nil, fmt.Errorf("personService.GetPerson: %w", err)
	}

	return &domain.PersonDetail{
		Person:    *person,
		Addresses: addresses,
		Roles:     roles,
	}, nil
}

// UpdatePerson updates an existing person.
func (s *PersonService) UpdatePerson(ctx context.Context, id uuid.UUID, input UpdatePersonInput) (*domain.Person, error) {
	if err := validate.Struct(input); err != nil {
		return nil, fmt.Errorf("personService.UpdatePerson: %w", err)
	}

	campusID := auth.CampusIDFromContext(ctx)

	old, err := s.personRepo.FindByID(ctx, id, campusID)
	if err != nil {
		return nil, fmt.Errorf("personService.UpdatePerson: find: %w", err)
	}

	updated, err := applyPersonUpdate(old, input)
	if err != nil {
		return nil, err
	}

	result, err := s.personRepo.Update(ctx, updated)
	if err != nil {
		return nil, fmt.Errorf("personService.UpdatePerson: update: %w", err)
	}

	if addr := buildPersonAddress(input.Address, id); addr != nil {
		if _, addrErr := s.personRepo.UpdateAddress(ctx, id, *addr); addrErr != nil {
			slog.ErrorContext(ctx, "personService.UpdatePerson: address update failed",
				"error", addrErr.Error(), "person_id", id,
			)
		}
	}

	oldValues := map[string]string{"full_name": old.FullName}
	newValues := map[string]string{"full_name": result.FullName}

	if auditErr := s.auditSvc.LogAction(ctx, AuditParams{
		ActionType:  "UPDATE",
		EntityType:  "person",
		EntityID:    &result.ID,
		Module:      "person-management",
		Description: "person updated",
		OldValues:   oldValues,
		NewValues:   newValues,
		Success:     true,
	}); auditErr != nil {
		slog.ErrorContext(ctx, "personService.UpdatePerson: audit failed",
			"error", auditErr.Error(), "person_id", id,
		)
	}

	return result, nil
}

// ListPersons returns a paginated list of persons.
func (s *PersonService) ListPersons(ctx context.Context, query string, page, perPage int, agreementStatus string) (*domain.PersonListResult, error) {
	campusID := auth.CampusIDFromContext(ctx)

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	filter := domain.PersonFilter{
		Query:           query,
		CampusID:        campusID,
		Page:            page,
		PerPage:         perPage,
		AgreementStatus: agreementStatus,
	}

	result, err := s.personRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("personService.ListPersons: %w", err)
	}

	return result, nil
}

// CheckDuplicate checks for duplicate persons by document.
func (s *PersonService) CheckDuplicate(ctx context.Context, documentType, documentNumber string) (*domain.DuplicateCheckResult, error) {
	campusID := auth.CampusIDFromContext(ctx)

	result, err := s.personRepo.CheckDuplicate(ctx, documentType, documentNumber, campusID)
	if err != nil {
		return nil, fmt.Errorf("personService.CheckDuplicate: %w", err)
	}

	return result, nil
}

// AddRole adds a role to a person, enforcing the role hierarchy.
// When adding PROFESSIONAL, COORDINATOR, or ADMIN, VOLUNTEER is auto-assigned if missing.
func (s *PersonService) AddRole(ctx context.Context, personID uuid.UUID, input AddRoleInput) (*domain.PersonRole, error) {
	if err := validate.Struct(input); err != nil {
		return nil, fmt.Errorf("personService.AddRole: %w", err)
	}

	campusID := auth.CampusIDFromContext(ctx)
	claims := auth.ClaimsFromContext(ctx)
	userID := parseUserID(claims.Subject)

	if _, err := s.personRepo.FindByID(ctx, personID, campusID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("personService.AddRole: find person: %w", err)
	}

	// Enforce role hierarchy: auto-assign VOLUNTEER for roles that require it
	if domain.RequiresVolunteerBase(input.RoleType) {
		if err := s.ensureVolunteerRole(ctx, personID, campusID, userID); err != nil {
			return nil, fmt.Errorf("personService.AddRole: ensure volunteer: %w", err)
		}
	}

	role := domain.PersonRole{
		ID:                    uuid.New(),
		PersonID:              personID,
		RoleType:              input.RoleType,
		ProfessionalSpecialty: input.ProfessionalSpecialty,
		IsActive:              true,
		ActivatedBy:           userID,
		Notes:                 input.Notes,
	}

	created, err := s.personRoleRepo.Create(ctx, role)
	if err != nil {
		return nil, fmt.Errorf("personService.AddRole: %w", err)
	}

	// Create pending agreement if adding VOLUNTEER role directly
	if input.RoleType == domain.RoleVolunteer {
		s.createPendingAgreement(ctx, personID, created.ID, campusID)
	}

	s.logRoleCreated(ctx, created.ID, personID, input.RoleType,
		fmt.Sprintf("role %s added to person", input.RoleType))

	return created, nil
}

// logRoleCreated writes an audit entry for a created person_role, logging on failure.
func (s *PersonService) logRoleCreated(ctx context.Context, roleID, personID uuid.UUID, roleType, description string) {
	if auditErr := s.auditSvc.LogAction(ctx, AuditParams{
		ActionType:  "CREATE",
		EntityType:  "person_role",
		EntityID:    &roleID,
		Module:      "person-management",
		Description: description,
		NewValues:   map[string]string{"role_type": roleType, "person_id": personID.String()},
		Success:     true,
	}); auditErr != nil {
		slog.ErrorContext(ctx, "personService: person_role audit failed",
			"error", auditErr.Error(), "person_id", personID,
		)
	}
}

// ensureVolunteerRole checks if the person already has a VOLUNTEER role.
// If not, it auto-creates one with a PENDING agreement.
func (s *PersonService) ensureVolunteerRole(ctx context.Context, personID uuid.UUID, campusID uuid.UUID, userID *uuid.UUID) error {
	roles, err := s.personRoleRepo.FindByPersonID(ctx, personID)
	if err != nil {
		return fmt.Errorf("find roles: %w", err)
	}

	for _, r := range roles {
		if r.RoleType == domain.RoleVolunteer {
			return nil // Already has VOLUNTEER role
		}
	}

	// Auto-create VOLUNTEER role
	autoNote := "auto-assigned: base role required by role hierarchy"
	volunteerRole := domain.PersonRole{
		ID:          uuid.New(),
		PersonID:    personID,
		RoleType:    domain.RoleVolunteer,
		IsActive:    true,
		ActivatedBy: userID,
		Notes:       &autoNote,
	}

	created, err := s.personRoleRepo.Create(ctx, volunteerRole)
	if err != nil {
		if errors.Is(err, domain.ErrDuplicate) {
			return nil // Race condition: already exists
		}
		return fmt.Errorf("create volunteer role: %w", err)
	}

	slog.InfoContext(ctx, "personService: auto-assigned VOLUNTEER role",
		"person_id", personID, "role_id", created.ID,
	)

	// Create pending agreement for the auto-assigned VOLUNTEER role
	s.createPendingAgreement(ctx, personID, created.ID, campusID)
	s.logRoleCreated(ctx, created.ID, personID, domain.RoleVolunteer,
		"VOLUNTEER role auto-assigned (role hierarchy)")

	return nil
}

// createPendingAgreement creates a PENDING volunteer agreement, logging errors without failing.
func (s *PersonService) createPendingAgreement(ctx context.Context, personID uuid.UUID, roleID uuid.UUID, campusID uuid.UUID) {
	agreement := domain.VolunteerAgreement{
		ID:               uuid.New(),
		PersonID:         personID,
		PersonRoleID:     roleID,
		CampusID:         campusID,
		Status:           domain.AgreementPending,
		AgreementVersion: "v1",
	}

	if _, err := s.agreementRepo.Create(ctx, agreement); err != nil {
		if !errors.Is(err, domain.ErrAgreementExists) {
			slog.ErrorContext(ctx, "personService: create pending agreement failed",
				"error", err.Error(), "person_id", personID,
			)
		}
	}
}

// ToggleRole activates or deactivates a person role.
func (s *PersonService) ToggleRole(ctx context.Context, personID uuid.UUID, roleID uuid.UUID, isActive bool) (*domain.PersonRole, error) {
	campusID := auth.CampusIDFromContext(ctx)
	claims := auth.ClaimsFromContext(ctx)
	userID := parseUserID(claims.Subject)

	if _, err := s.personRepo.FindByID(ctx, personID, campusID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("personService.ToggleRole: find person: %w", err)
	}

	result, err := s.personRoleRepo.ToggleActive(ctx, roleID, isActive, userID)
	if err != nil {
		return nil, fmt.Errorf("personService.ToggleRole: %w", err)
	}

	action := "activated"
	if !isActive {
		action = "deactivated"
	}

	if auditErr := s.auditSvc.LogAction(ctx, AuditParams{
		ActionType:  "UPDATE",
		EntityType:  "person_role",
		EntityID:    &result.ID,
		Module:      "person-management",
		Description: fmt.Sprintf("role %s %s", result.RoleType, action),
		NewValues:   map[string]string{"is_active": fmt.Sprintf("%t", isActive)},
		Success:     true,
	}); auditErr != nil {
		slog.ErrorContext(ctx, "personService.ToggleRole: audit failed",
			"error", auditErr.Error(), "role_id", roleID,
		)
	}

	return result, nil
}

// GetHistory returns the service history for a person.
// Returns empty slice in Sprint 2; populated when triage/attendance exist.
func (s *PersonService) GetHistory(ctx context.Context, personID uuid.UUID) ([]domain.HistoryEntry, error) {
	campusID := auth.CampusIDFromContext(ctx)

	if _, err := s.personRepo.FindByID(ctx, personID, campusID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("personService.GetHistory: %w", err)
	}

	return []domain.HistoryEntry{}, nil
}

func parseUserID(subject string) *uuid.UUID {
	if id, err := uuid.Parse(subject); err == nil {
		return &id
	}
	return nil
}

func parseOptionalDate(dateStr *string) (*time.Time, error) {
	if dateStr == nil || *dateStr == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *dateStr)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
