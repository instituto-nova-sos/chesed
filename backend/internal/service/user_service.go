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

// UserRepository defines the interface for app_user persistence.
type UserRepository interface {
	FindByKeycloakSubject(ctx context.Context, subjectID string) (*domain.AppUser, error)
	Create(ctx context.Context, user domain.AppUser) (*domain.AppUser, error)
	LinkPersonID(ctx context.Context, userID uuid.UUID, personID uuid.UUID) error
	LinkPersonAndCampus(ctx context.Context, userID uuid.UUID, personID uuid.UUID, campusID uuid.UUID) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

// UserService handles user operations including auto-provisioning.
type UserService struct {
	repo     UserRepository
	auditSvc *AuditService
}

// NewUserService creates a new UserService.
func NewUserService(repo UserRepository, auditSvc *AuditService) *UserService {
	return &UserService{repo: repo, auditSvc: auditSvc}
}

// EnsureUser implements auto-provisioning.
// On first login, creates an app_user from token claims.
// On subsequent logins, updates last_login.
func (s *UserService) EnsureUser(ctx context.Context, claims auth.AuthClaims, ip, userAgent string) (*domain.AppUser, error) {
	existing, err := s.repo.FindByKeycloakSubject(ctx, claims.Subject)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("userService.EnsureUser: find: %w", err)
	}

	if existing != nil {
		if err := s.repo.UpdateLastLogin(ctx, existing.ID); err != nil {
			return nil, fmt.Errorf("userService.EnsureUser: update login: %w", err)
		}
		if auditErr := s.auditSvc.LogAction(ctx, AuditParams{
			ActionType:  "LOGIN",
			EntityType:  "app_user",
			EntityID:    &existing.ID,
			Module:      "auth",
			Description: "user login",
			IPAddress:   ip,
			UserAgent:   userAgent,
			Success:     true,
		}); auditErr != nil {
			slog.ErrorContext(ctx, "userService.EnsureUser: login audit failed",
				"error", auditErr.Error(),
				"subject", claims.Subject,
			)
		}
		return existing, nil
	}

	return s.provisionUser(ctx, claims, ip, userAgent)
}

func (s *UserService) provisionUser(ctx context.Context, claims auth.AuthClaims, ip, userAgent string) (*domain.AppUser, error) {
	profile := resolveAccessProfile(claims.Roles)

	var campusPtr *uuid.UUID
	if claims.CampusID != uuid.Nil {
		campusPtr = &claims.CampusID
	}

	newUser := domain.AppUser{
		ID:                uuid.New(),
		Email:             claims.Email,
		KeycloakSubjectID: claims.Subject,
		AccessProfile:     profile,
		CampusID:          campusPtr,
		IsActive:          true,
	}

	created, err := s.repo.Create(ctx, newUser)
	if err != nil {
		return nil, fmt.Errorf("userService.provisionUser: create: %w", err)
	}

	auditErr := s.auditSvc.LogAction(ctx, AuditParams{
		ActionType:  "CREATE",
		EntityType:  "app_user",
		EntityID:    &created.ID,
		Module:      "auth",
		Description: "auto-provisioned user on first login",
		NewValues:   map[string]string{"email": claims.Email, "access_profile": profile},
		IPAddress:   ip,
		UserAgent:   userAgent,
		Success:     true,
	})
	if auditErr != nil {
		return created, fmt.Errorf("userService.provisionUser: audit: %w", auditErr)
	}

	return created, nil
}

// ResolveCampusFromDB looks up the campus_id from an existing app_user record.
// Returns uuid.Nil if user not found or campus not set.
func (s *UserService) ResolveCampusFromDB(ctx context.Context, subject string) (uuid.UUID, error) {
	user, err := s.repo.FindByKeycloakSubject(ctx, subject)
	if err != nil {
		return uuid.Nil, err
	}
	if user.CampusID == nil {
		return uuid.Nil, nil
	}
	return *user.CampusID, nil
}

func resolveAccessProfile(roles []string) string {
	highest := domain.RoleVolunteer
	highestLevel := 0

	for _, role := range roles {
		if level, ok := domain.RoleHierarchy[role]; ok && level > highestLevel {
			highest = role
			highestLevel = level
		}
	}

	return highest
}
