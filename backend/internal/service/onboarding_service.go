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

// OnboardingStatus represents the onboarding state for an authenticated user.
type OnboardingStatus struct {
	PersonID               *uuid.UUID `json:"person_id"`
	CampusID               *uuid.UUID `json:"campus_id"`
	CampusTimezone         string     `json:"campus_timezone,omitempty"`
	NeedsProfileCompletion bool       `json:"needs_profile_completion"`
	NeedsCampusAssignment  bool       `json:"needs_campus_assignment"`
	NeedsAgreement         bool       `json:"needs_agreement"`
	Roles                  []string   `json:"roles"`
}

// OnboardingService resolves the onboarding state of an authenticated user.
// It auto-links pre-created persons by email when possible and resolves campus from DB.
// CampusTimezoneLookup resolves a campus by ID for timezone enrichment. It is a
// narrow, consumer-defined subset of the campus repository (interface
// segregation): onboarding only needs to read one campus.
type CampusTimezoneLookup interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Campus, error)
}

type OnboardingService struct {
	userRepo      UserRepository
	personRepo    PersonRepository
	roleRepo      PersonRoleRepository
	agreementRepo VolunteerAgreementRepository
	campusRepo    CampusTimezoneLookup
	auditSvc      *AuditService
}

// NewOnboardingService creates a new OnboardingService.
func NewOnboardingService(
	userRepo UserRepository,
	personRepo PersonRepository,
	roleRepo PersonRoleRepository,
	agreementRepo VolunteerAgreementRepository,
	campusRepo CampusTimezoneLookup,
	auditSvc *AuditService,
) *OnboardingService {
	return &OnboardingService{
		userRepo:      userRepo,
		personRepo:    personRepo,
		roleRepo:      roleRepo,
		agreementRepo: agreementRepo,
		campusRepo:    campusRepo,
		auditSvc:      auditSvc,
	}
}

// resolveCampusTimezone looks up the IANA timezone for the resolved campus so
// the frontend can render times in the campus's zone. A lookup failure is
// non-fatal: the field stays empty and the client falls back to the browser zone.
func (s *OnboardingService) resolveCampusTimezone(ctx context.Context, status *OnboardingStatus) {
	if status.CampusID == nil {
		return
	}
	campus, err := s.campusRepo.FindByID(ctx, *status.CampusID)
	if err != nil {
		slog.WarnContext(ctx, "onboardingService: campus timezone lookup failed",
			"error", err.Error(), "campus_id", *status.CampusID)
		return
	}
	status.CampusTimezone = campus.Timezone
}

// GetStatus returns the onboarding status for the authenticated user.
// Handles three scenarios:
// 1. App user exists with campus and person → normal flow
// 2. App user exists without person → try auto-link by email
// 3. No app user yet → try to find person by email globally
func (s *OnboardingService) GetStatus(ctx context.Context, claims auth.AuthClaims) (*OnboardingStatus, error) {
	appUser, err := s.userRepo.FindByKeycloakSubject(ctx, claims.Subject)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("onboardingService.GetStatus: find user: %w", err)
	}

	// Case: no app_user exists yet (first login, before AutoProvision)
	var status *OnboardingStatus
	if appUser == nil {
		status, err = s.handleNewUser(ctx, claims)
	} else {
		status, err = s.handleExistingUser(ctx, claims, appUser)
	}
	if err != nil {
		return nil, err
	}

	s.resolveCampusTimezone(ctx, status)
	return status, nil
}

// handleNewUser resolves onboarding status when no app_user exists.
// Tries to find a person by email globally to derive campus.
func (s *OnboardingService) handleNewUser(ctx context.Context, claims auth.AuthClaims) (*OnboardingStatus, error) {
	status := &OnboardingStatus{Roles: claims.Roles}

	if claims.Email == "" {
		status.NeedsProfileCompletion = true
		status.NeedsCampusAssignment = true
		return status, nil
	}

	persons, err := s.personRepo.FindByEmailGlobal(ctx, claims.Email)
	if err != nil {
		return nil, fmt.Errorf("onboardingService.handleNewUser: find person: %w", err)
	}

	switch len(persons) {
	case 0:
		status.NeedsProfileCompletion = true
		status.NeedsCampusAssignment = true
	case 1:
		person := persons[0]
		status.PersonID = &person.ID
		status.CampusID = &person.CampusID
		status.NeedsAgreement = s.checkAgreementNeeded(ctx, person.ID)
	default:
		// Multiple persons with same email in different campuses — user must pick
		status.NeedsCampusAssignment = true
	}

	return status, nil
}

// handleExistingUser resolves onboarding status when app_user exists.
func (s *OnboardingService) handleExistingUser(ctx context.Context, claims auth.AuthClaims, appUser *domain.AppUser) (*OnboardingStatus, error) {
	status := &OnboardingStatus{
		CampusID: appUser.CampusID,
		PersonID: appUser.PersonID,
		Roles:    claims.Roles,
	}

	// If no campus assigned yet, try to resolve it from a matching person.
	if appUser.CampusID == nil {
		return s.resolveWithoutCampus(ctx, claims, appUser, status)
	}

	return s.resolveWithCampus(ctx, claims, appUser, status)
}

// resolveWithoutCampus handles an app_user that has no campus assigned yet,
// attempting to auto-link a globally-matching person by email.
func (s *OnboardingService) resolveWithoutCampus(ctx context.Context, claims auth.AuthClaims, appUser *domain.AppUser, status *OnboardingStatus) (*OnboardingStatus, error) {
	status.NeedsCampusAssignment = true
	status.NeedsProfileCompletion = appUser.PersonID == nil

	if appUser.PersonID != nil || claims.Email == "" {
		return status, nil
	}

	persons, err := s.personRepo.FindByEmailGlobal(ctx, claims.Email)
	if err != nil {
		return nil, fmt.Errorf("onboardingService.resolveWithoutCampus: find person: %w", err)
	}
	if len(persons) != 1 {
		return status, nil
	}

	person := persons[0]
	if linkErr := s.userRepo.LinkPersonAndCampus(ctx, appUser.ID, person.ID, person.CampusID); linkErr != nil {
		return nil, fmt.Errorf("onboardingService.resolveWithoutCampus: link: %w", linkErr)
	}
	status.PersonID = &person.ID
	status.CampusID = &person.CampusID
	status.NeedsCampusAssignment = false
	status.NeedsProfileCompletion = false
	status.NeedsAgreement = s.checkAgreementNeeded(ctx, person.ID)

	slog.InfoContext(ctx, "onboardingService: auto-linked person and campus",
		"app_user_id", appUser.ID, "person_id", person.ID, "campus_id", person.CampusID,
	)
	s.logAutoLinkAudit(ctx, appUser.ID, person.ID)
	return status, nil
}

// resolveWithCampus handles an app_user that already has a campus, auto-linking
// a person by email within that campus when possible.
func (s *OnboardingService) resolveWithCampus(ctx context.Context, claims auth.AuthClaims, appUser *domain.AppUser, status *OnboardingStatus) (*OnboardingStatus, error) {
	if appUser.PersonID == nil && claims.Email != "" {
		if err := s.tryLinkPersonByEmail(ctx, claims.Email, appUser, status); err != nil {
			return nil, err
		}
	}

	if status.PersonID == nil {
		status.NeedsProfileCompletion = true
	} else {
		status.NeedsAgreement = s.checkAgreementNeeded(ctx, *status.PersonID)
	}

	return status, nil
}

// tryLinkPersonByEmail links a campus-scoped person to the app_user if found.
func (s *OnboardingService) tryLinkPersonByEmail(ctx context.Context, email string, appUser *domain.AppUser, status *OnboardingStatus) error {
	person, findErr := s.personRepo.FindByEmail(ctx, email, *appUser.CampusID)
	if findErr != nil && !errors.Is(findErr, domain.ErrNotFound) {
		return fmt.Errorf("onboardingService.tryLinkPersonByEmail: find by email: %w", findErr)
	}
	if person == nil {
		return nil
	}

	if linkErr := s.userRepo.LinkPersonID(ctx, appUser.ID, person.ID); linkErr != nil {
		return fmt.Errorf("onboardingService.tryLinkPersonByEmail: link person: %w", linkErr)
	}
	status.PersonID = &person.ID

	slog.InfoContext(ctx, "onboardingService: auto-linked person by email",
		"app_user_id", appUser.ID, "person_id", person.ID,
	)
	s.logAutoLinkAudit(ctx, appUser.ID, person.ID)
	return nil
}

// checkAgreementNeeded returns true if the person has an active VOLUNTEER role without accepted agreement.
func (s *OnboardingService) checkAgreementNeeded(ctx context.Context, personID uuid.UUID) bool {
	roles, err := s.roleRepo.FindByPersonID(ctx, personID)
	if err != nil {
		slog.ErrorContext(ctx, "onboardingService: check roles failed", "error", err.Error())
		return false
	}

	hasActiveVolunteer := false
	for _, role := range roles {
		if role.RoleType == domain.RoleVolunteer && role.IsActive {
			hasActiveVolunteer = true
			break
		}
	}

	if !hasActiveVolunteer {
		return false
	}

	accepted, err := s.agreementRepo.HasAcceptedAgreement(ctx, personID)
	if err != nil {
		slog.ErrorContext(ctx, "onboardingService: check agreement failed", "error", err.Error())
		return false
	}
	return !accepted
}

func (s *OnboardingService) logAutoLinkAudit(ctx context.Context, appUserID, personID uuid.UUID) {
	auditErr := s.auditSvc.LogAction(ctx, AuditParams{
		ActionType:  "UPDATE",
		EntityType:  "app_user",
		EntityID:    &appUserID,
		Module:      "onboarding",
		Description: "auto-linked person by email",
		NewValues:   map[string]string{"person_id": personID.String()},
		Success:     true,
	})
	if auditErr != nil {
		slog.ErrorContext(ctx, "onboardingService: audit failed", "error", auditErr.Error())
	}
}
