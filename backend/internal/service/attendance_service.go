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

// AttendanceRepository defines the interface for attendance persistence.
type AttendanceRepository interface {
	Create(ctx context.Context, a domain.Attendance) (*domain.Attendance, error)
	FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Attendance, error)
	FindByIDWithTransitions(ctx context.Context, id, campusID uuid.UUID) (*domain.AttendanceDetail, error)
	List(ctx context.Context, filter domain.AttendanceFilter) (*domain.AttendanceListResult, error)
	Transition(ctx context.Context, t domain.AttendanceTransition) (*domain.Attendance, error)
	UpdateNotes(ctx context.Context, id, campusID uuid.UUID, observations, recommendations *string) (*domain.Attendance, error)
}

// CreateAttendanceInput holds validated input for attendance creation.
type CreateAttendanceInput struct {
	PersonID        string  `json:"person_id" validate:"required,uuid"`
	TriageID        *string `json:"triage_id" validate:"omitempty,uuid"`
	ServiceTypeID   string  `json:"service_type_id" validate:"required,uuid"`
	ProfessionalID  string  `json:"professional_id" validate:"required,uuid"`
	AttendanceDate  *string `json:"attendance_date" validate:"omitempty"`
	Observations    *string `json:"observations" validate:"omitempty"`
	Recommendations *string `json:"recommendations" validate:"omitempty"`
}

// UpdateAttendanceNotesInput holds validated input for updating attendance notes.
type UpdateAttendanceNotesInput struct {
	Observations    *string `json:"observations" validate:"omitempty"`
	Recommendations *string `json:"recommendations" validate:"omitempty"`
}

// TransitionAttendanceInput holds validated input for state transitions.
type TransitionAttendanceInput struct {
	To     string  `json:"to" validate:"required,oneof=IN_PROGRESS COMPLETED CANCELLED"`
	Reason *string `json:"reason" validate:"omitempty,max=500"`
}

// AttendanceService handles attendance business logic.
type AttendanceService struct {
	repo     AttendanceRepository
	auditSvc *AuditService
}

// NewAttendanceService creates a new AttendanceService.
func NewAttendanceService(repo AttendanceRepository, auditSvc *AuditService) *AttendanceService {
	return &AttendanceService{repo: repo, auditSvc: auditSvc}
}

// CreateAttendance creates a new attendance in SCHEDULED status.
func (s *AttendanceService) CreateAttendance(ctx context.Context, input CreateAttendanceInput) (*domain.Attendance, error) {
	if err := validate.Struct(input); err != nil {
		return nil, fmt.Errorf("attendanceService.CreateAttendance: %w", err)
	}

	claims := auth.ClaimsFromContext(ctx)
	if claims.CampusID == uuid.Nil {
		return nil, fmt.Errorf("attendanceService.CreateAttendance: %w", domain.ErrForbidden)
	}

	att, err := buildAttendanceFromInput(input, claims)
	if err != nil {
		return nil, err
	}

	created, err := s.repo.Create(ctx, att)
	if err != nil {
		return nil, fmt.Errorf("attendanceService.CreateAttendance: %w", err)
	}

	if auditErr := s.auditSvc.LogAction(ctx, AuditParams{
		ActionType:  "CREATE",
		EntityType:  "attendance",
		EntityID:    &created.ID,
		Module:      "attendance",
		Description: "attendance scheduled",
		NewValues: map[string]any{
			"person_id":       created.PersonID,
			"service_type_id": created.ServiceTypeID,
			"status":          created.Status,
		},
		Success: true,
	}); auditErr != nil {
		slog.ErrorContext(ctx, "attendanceService.CreateAttendance: audit failed",
			"error", auditErr.Error(), "attendance_id", created.ID,
		)
	}

	return created, nil
}

// buildAttendanceFromInput parses the identifiers and date from a create input
// and assembles a scheduled domain.Attendance.
func buildAttendanceFromInput(input CreateAttendanceInput, claims auth.AuthClaims) (domain.Attendance, error) {
	personID, err := uuid.Parse(input.PersonID)
	if err != nil {
		return domain.Attendance{}, fmt.Errorf("attendanceService.CreateAttendance: invalid person_id: %w", err)
	}
	serviceTypeID, err := uuid.Parse(input.ServiceTypeID)
	if err != nil {
		return domain.Attendance{}, fmt.Errorf("attendanceService.CreateAttendance: invalid service_type_id: %w", err)
	}
	professionalID, err := uuid.Parse(input.ProfessionalID)
	if err != nil {
		return domain.Attendance{}, fmt.Errorf("attendanceService.CreateAttendance: invalid professional_id: %w", err)
	}
	triageID, err := parseOptionalUUID(input.TriageID)
	if err != nil {
		return domain.Attendance{}, fmt.Errorf("attendanceService.CreateAttendance: invalid triage_id: %w", err)
	}
	attendanceDate, err := parseOptionalTime(input.AttendanceDate)
	if err != nil {
		return domain.Attendance{}, fmt.Errorf("attendanceService.CreateAttendance: invalid attendance_date: %w", err)
	}
	if attendanceDate == nil {
		now := time.Now().UTC()
		attendanceDate = &now
	}

	return domain.Attendance{
		ID:              uuid.New(),
		PersonID:        personID,
		TriageID:        triageID,
		CampusID:        claims.CampusID,
		ServiceTypeID:   serviceTypeID,
		ProfessionalID:  professionalID,
		Status:          domain.AttendanceStatusScheduled,
		AttendanceDate:  *attendanceDate,
		Observations:    input.Observations,
		Recommendations: input.Recommendations,
		CreatedBy:       parseUserID(claims.Subject),
	}, nil
}

// GetAttendance returns an attendance with its transition history.
func (s *AttendanceService) GetAttendance(ctx context.Context, id uuid.UUID) (*domain.AttendanceDetail, error) {
	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return nil, fmt.Errorf("attendanceService.GetAttendance: %w", domain.ErrForbidden)
	}
	detail, err := s.repo.FindByIDWithTransitions(ctx, id, campusID)
	if err != nil {
		return nil, fmt.Errorf("attendanceService.GetAttendance: %w", err)
	}
	return detail, nil
}

// ListAttendances returns a paginated list of attendances.
func (s *AttendanceService) ListAttendances(ctx context.Context, filter domain.AttendanceFilter) (*domain.AttendanceListResult, error) {
	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return nil, fmt.Errorf("attendanceService.ListAttendances: %w", domain.ErrForbidden)
	}
	filter.CampusID = campusID
	result, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("attendanceService.ListAttendances: %w", err)
	}
	return result, nil
}

// TransitionAttendance moves an attendance to a new state if the transition is allowed.
func (s *AttendanceService) TransitionAttendance(ctx context.Context, id uuid.UUID, input TransitionAttendanceInput) (*domain.Attendance, error) {
	if err := validate.Struct(input); err != nil {
		return nil, fmt.Errorf("attendanceService.TransitionAttendance: %w", err)
	}

	claims := auth.ClaimsFromContext(ctx)
	if claims.CampusID == uuid.Nil {
		return nil, fmt.Errorf("attendanceService.TransitionAttendance: %w", domain.ErrForbidden)
	}
	actorID := parseUserID(claims.Subject)
	if actorID == nil {
		return nil, fmt.Errorf("attendanceService.TransitionAttendance: missing user identity")
	}

	existing, err := s.repo.FindByID(ctx, id, claims.CampusID)
	if err != nil {
		return nil, fmt.Errorf("attendanceService.TransitionAttendance: %w", err)
	}

	if !domain.CanTransition(existing.Status, input.To) {
		return nil, fmt.Errorf("attendanceService.TransitionAttendance: %s→%s: %w",
			existing.Status, input.To, domain.ErrInvalidTransition)
	}

	transition := domain.AttendanceTransition{
		ID:             uuid.New(),
		AttendanceID:   id,
		FromStatus:     existing.Status,
		ToStatus:       input.To,
		Reason:         input.Reason,
		TransitionedBy: *actorID,
	}

	updated, err := s.repo.Transition(ctx, transition)
	if err != nil {
		return nil, fmt.Errorf("attendanceService.TransitionAttendance: %w", err)
	}

	if auditErr := s.auditSvc.LogAction(ctx, AuditParams{
		ActionType:  "TRANSITION",
		EntityType:  "attendance",
		EntityID:    &updated.ID,
		Module:      "attendance",
		Description: fmt.Sprintf("attendance %s→%s", existing.Status, input.To),
		OldValues:   map[string]string{"status": existing.Status},
		NewValues:   map[string]string{"status": updated.Status},
		Success:     true,
	}); auditErr != nil {
		slog.ErrorContext(ctx, "attendanceService.TransitionAttendance: audit failed",
			"error", auditErr.Error(), "attendance_id", updated.ID,
		)
	}

	return updated, nil
}

// UpdateNotes updates observations and recommendations for an attendance.
func (s *AttendanceService) UpdateNotes(ctx context.Context, id uuid.UUID, input UpdateAttendanceNotesInput) (*domain.Attendance, error) {
	if err := validate.Struct(input); err != nil {
		return nil, fmt.Errorf("attendanceService.UpdateNotes: %w", err)
	}

	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return nil, fmt.Errorf("attendanceService.UpdateNotes: %w", domain.ErrForbidden)
	}

	updated, err := s.repo.UpdateNotes(ctx, id, campusID, input.Observations, input.Recommendations)
	if err != nil {
		return nil, fmt.Errorf("attendanceService.UpdateNotes: %w", err)
	}

	if auditErr := s.auditSvc.LogAction(ctx, AuditParams{
		ActionType:  "UPDATE",
		EntityType:  "attendance",
		EntityID:    &updated.ID,
		Module:      "attendance",
		Description: "attendance notes updated",
		Success:     true,
	}); auditErr != nil {
		slog.ErrorContext(ctx, "attendanceService.UpdateNotes: audit failed",
			"error", auditErr.Error(), "attendance_id", updated.ID,
		)
	}

	return updated, nil
}
