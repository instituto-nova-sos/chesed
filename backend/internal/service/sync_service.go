package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// SyncPersonRepository is the subset of person persistence used by the sync engine.
type SyncPersonRepository interface {
	FindBySyncID(ctx context.Context, syncID, campusID uuid.UUID) (*domain.Person, error)
	CreateWithSync(ctx context.Context, person domain.Person, address *domain.Address, syncID uuid.UUID) (*domain.Person, error)
	ListUpdatedSince(ctx context.Context, campusID uuid.UUID, since time.Time, limit int) ([]domain.Person, error)
}

// SyncTriageRepository is the subset of triage persistence used by the sync engine.
type SyncTriageRepository interface {
	FindBySyncID(ctx context.Context, syncID, campusID uuid.UUID) (*domain.Triage, error)
	CreateWithSync(ctx context.Context, triage domain.Triage, syncID uuid.UUID) (*domain.Triage, error)
	ListUpdatedSince(ctx context.Context, campusID uuid.UUID, since time.Time, limit int) ([]domain.Triage, error)
}

// SyncAttendanceRepository is the subset of attendance persistence used by the sync engine.
type SyncAttendanceRepository interface {
	FindBySyncID(ctx context.Context, syncID, campusID uuid.UUID) (*domain.Attendance, error)
	CreateWithSync(ctx context.Context, a domain.Attendance, syncID uuid.UUID) (*domain.Attendance, error)
	ListUpdatedSince(ctx context.Context, campusID uuid.UUID, since time.Time, limit int) ([]domain.Attendance, error)
}

// SyncService orchestrates batch push and delta pull for offline records.
//
// Push semantics:
//   - sync_id is the idempotency key; re-pushing the same sync_id returns the
//     existing server record without duplicate creation.
//   - Per-record errors do not abort the batch — callers see a results array.
//   - Batch-level errors (oversize, missing campus) abort the entire request.
//
// Pull semantics:
//   - Delta queries each requested entity since the given cursor, sorts by
//     updated_at, and pages by limit. has_more is true when any entity returned
//     exactly `limit` records, indicating more may exist past the cursor.
type SyncService struct {
	personRepo     SyncPersonRepository
	triageRepo     SyncTriageRepository
	attendanceRepo SyncAttendanceRepository
	auditSvc       *AuditService
	validate       *validator.Validate
}

// NewSyncService constructs a SyncService.
func NewSyncService(p SyncPersonRepository, t SyncTriageRepository, a SyncAttendanceRepository, audit *AuditService) *SyncService {
	return &SyncService{
		personRepo:     p,
		triageRepo:     t,
		attendanceRepo: a,
		auditSvc:       audit,
		validate:       validator.New(),
	}
}

// --- Push -------------------------------------------------------------------

// Push validates and dispatches a batch of offline records.
func (s *SyncService) Push(ctx context.Context, req domain.SyncPushRequest) (*domain.SyncPushResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if auth.CampusIDFromContext(ctx) == uuid.Nil {
		return nil, domain.ErrForbidden
	}
	return s.PushSkippingValidation(ctx, req.Records)
}

// PushSkippingValidation dispatches records assumed already validated.
// Exposed for tests that need to exercise per-record error paths bypassing
// request-level Validate() — production callers should use Push().
func (s *SyncService) PushSkippingValidation(ctx context.Context, records []domain.SyncPushRecord) (*domain.SyncPushResponse, error) {
	if auth.CampusIDFromContext(ctx) == uuid.Nil {
		return nil, domain.ErrForbidden
	}

	results := make([]domain.SyncPushResult, 0, len(records))
	for _, rec := range records {
		results = append(results, s.handleRecord(ctx, rec))
	}
	return &domain.SyncPushResponse{
		Results:         results,
		ServerTimestamp: time.Now().UTC(),
	}, nil
}

func (s *SyncService) handleRecord(ctx context.Context, rec domain.SyncPushRecord) domain.SyncPushResult {
	switch rec.EntityType {
	case domain.SyncEntityPerson:
		return s.handlePerson(ctx, rec)
	case domain.SyncEntityTriage:
		return s.handleTriage(ctx, rec)
	case domain.SyncEntityAttendance:
		return s.handleAttendance(ctx, rec)
	default:
		return errorResult(rec.SyncID, fmt.Sprintf("unknown entity_type %q", rec.EntityType))
	}
}

// --- Person push ------------------------------------------------------------

type syncPersonInput struct {
	FullName       string  `json:"full_name" validate:"required,max=200"`
	BirthDate      *string `json:"birth_date" validate:"omitempty"`
	DocumentType   string  `json:"document_type" validate:"required,oneof=CPF RG SSN EU_ID PASSPORT OTHER"`
	DocumentNumber *string `json:"document_number" validate:"omitempty,max=30"`
	Nationality    *string `json:"nationality" validate:"omitempty,len=3"`
	Gender         *string `json:"gender" validate:"omitempty,oneof=M F OTHER PREFER_NOT_TO_SAY"`
	Email          *string `json:"email" validate:"omitempty,email,max=255"`
	Phone          *string `json:"phone" validate:"omitempty,max=30"`
	ReferralSource *string `json:"referral_source" validate:"omitempty,max=200"`
}

func (s *SyncService) handlePerson(ctx context.Context, rec domain.SyncPushRecord) domain.SyncPushResult {
	campusID := auth.CampusIDFromContext(ctx)

	existing, err := s.personRepo.FindBySyncID(ctx, rec.SyncID, campusID)
	if err == nil {
		// Idempotent re-push: already applied; no audit, no insert.
		return createdResult(rec.SyncID, existing.ID)
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return errorResult(rec.SyncID, err.Error())
	}

	var in syncPersonInput
	if err := decodeRecord(rec.Data, &in); err != nil {
		return errorResult(rec.SyncID, err.Error())
	}
	if err := s.validate.Struct(in); err != nil {
		return errorResult(rec.SyncID, err.Error())
	}

	birthDate, err := parseOptionalDate(in.BirthDate)
	if err != nil {
		return errorResult(rec.SyncID, fmt.Sprintf("invalid birth_date: %v", err))
	}

	nationality := "BRA"
	if in.Nationality != nil && *in.Nationality != "" {
		nationality = *in.Nationality
	}

	person := domain.Person{
		ID:             uuid.New(),
		FullName:       in.FullName,
		BirthDate:      birthDate,
		DocumentType:   in.DocumentType,
		DocumentNumber: in.DocumentNumber,
		Nationality:    nationality,
		Gender:         in.Gender,
		Email:          in.Email,
		Phone:          in.Phone,
		ReferralSource: in.ReferralSource,
		CampusID:       campusID,
		IsActive:       true,
		CreatedBy:      parseUserID(auth.ClaimsFromContext(ctx).Subject),
	}

	var address *domain.Address
	created, err := s.personRepo.CreateWithSync(ctx, person, address, rec.SyncID)
	if err != nil {
		if errors.Is(err, domain.ErrDuplicate) || errors.Is(err, domain.ErrDuplicateEmail) || errors.Is(err, domain.ErrDuplicatePhone) {
			return conflictResult(rec.SyncID, nil, err.Error())
		}
		return errorResult(rec.SyncID, err.Error())
	}

	s.logCreate(ctx, "person", created.ID)
	return createdResult(rec.SyncID, created.ID)
}

// --- Triage push ------------------------------------------------------------

type syncTriageInput struct {
	PersonID       string   `json:"person_id" validate:"required,uuid"`
	MainComplaint  string   `json:"main_complaint" validate:"required"`
	TriageDate     *string  `json:"triage_date" validate:"omitempty"`
	Location       *string  `json:"location" validate:"omitempty,max=300"`
	Notes          *string  `json:"notes" validate:"omitempty"`
	RequestedTypes []string `json:"requested_service_type_ids" validate:"omitempty,dive,uuid"`
}

func (s *SyncService) handleTriage(ctx context.Context, rec domain.SyncPushRecord) domain.SyncPushResult {
	campusID := auth.CampusIDFromContext(ctx)
	claims := auth.ClaimsFromContext(ctx)

	existing, err := s.triageRepo.FindBySyncID(ctx, rec.SyncID, campusID)
	if err == nil {
		return createdResult(rec.SyncID, existing.ID)
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return errorResult(rec.SyncID, err.Error())
	}

	var in syncTriageInput
	if err := decodeRecord(rec.Data, &in); err != nil {
		return errorResult(rec.SyncID, err.Error())
	}
	if err := s.validate.Struct(in); err != nil {
		return errorResult(rec.SyncID, err.Error())
	}

	personID, err := uuid.Parse(in.PersonID)
	if err != nil {
		return errorResult(rec.SyncID, fmt.Sprintf("invalid person_id: %v", err))
	}

	triagedBy := uuid.Nil
	if id := parseUserID(claims.Subject); id != nil {
		triagedBy = *id
	}

	triageDate := time.Now().UTC()
	if in.TriageDate != nil && *in.TriageDate != "" {
		t, err := time.Parse(time.RFC3339, *in.TriageDate)
		if err != nil {
			return errorResult(rec.SyncID, fmt.Sprintf("invalid triage_date: %v", err))
		}
		triageDate = t
	}

	requested := make([]uuid.UUID, 0, len(in.RequestedTypes))
	for _, raw := range in.RequestedTypes {
		id, err := uuid.Parse(raw)
		if err != nil {
			return errorResult(rec.SyncID, fmt.Sprintf("invalid requested_service_type_id %q: %v", raw, err))
		}
		requested = append(requested, id)
	}

	triage := domain.Triage{
		ID:             uuid.New(),
		PersonID:       personID,
		CampusID:       campusID,
		MainComplaint:  in.MainComplaint,
		TriageDate:     triageDate,
		Location:       in.Location,
		TriagedBy:      triagedBy,
		Notes:          in.Notes,
		IsActive:       true,
		RequestedTypes: requested,
	}

	created, err := s.triageRepo.CreateWithSync(ctx, triage, rec.SyncID)
	if err != nil {
		if errors.Is(err, domain.ErrDuplicate) {
			return conflictResult(rec.SyncID, nil, err.Error())
		}
		return errorResult(rec.SyncID, err.Error())
	}

	s.logCreate(ctx, "triage", created.ID)
	return createdResult(rec.SyncID, created.ID)
}

// --- Attendance push --------------------------------------------------------

type syncAttendanceInput struct {
	PersonID        string  `json:"person_id" validate:"required,uuid"`
	TriageID        *string `json:"triage_id" validate:"omitempty,uuid"`
	ServiceTypeID   string  `json:"service_type_id" validate:"required,uuid"`
	ProfessionalID  string  `json:"professional_id" validate:"required,uuid"`
	Status          string  `json:"status" validate:"required,oneof=SCHEDULED IN_PROGRESS COMPLETED CANCELLED"`
	AttendanceDate  *string `json:"attendance_date" validate:"omitempty"`
	Observations    *string `json:"observations" validate:"omitempty"`
	Recommendations *string `json:"recommendations" validate:"omitempty"`
}

func (s *SyncService) handleAttendance(ctx context.Context, rec domain.SyncPushRecord) domain.SyncPushResult {
	campusID := auth.CampusIDFromContext(ctx)
	claims := auth.ClaimsFromContext(ctx)

	existing, err := s.attendanceRepo.FindBySyncID(ctx, rec.SyncID, campusID)
	if err == nil {
		return createdResult(rec.SyncID, existing.ID)
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return errorResult(rec.SyncID, err.Error())
	}

	var in syncAttendanceInput
	if err := decodeRecord(rec.Data, &in); err != nil {
		return errorResult(rec.SyncID, err.Error())
	}
	if err := s.validate.Struct(in); err != nil {
		return errorResult(rec.SyncID, err.Error())
	}

	personID, err := uuid.Parse(in.PersonID)
	if err != nil {
		return errorResult(rec.SyncID, fmt.Sprintf("invalid person_id: %v", err))
	}
	serviceTypeID, err := uuid.Parse(in.ServiceTypeID)
	if err != nil {
		return errorResult(rec.SyncID, fmt.Sprintf("invalid service_type_id: %v", err))
	}
	professionalID, err := uuid.Parse(in.ProfessionalID)
	if err != nil {
		return errorResult(rec.SyncID, fmt.Sprintf("invalid professional_id: %v", err))
	}

	var triageID *uuid.UUID
	if in.TriageID != nil && *in.TriageID != "" {
		id, err := uuid.Parse(*in.TriageID)
		if err != nil {
			return errorResult(rec.SyncID, fmt.Sprintf("invalid triage_id: %v", err))
		}
		triageID = &id
	}

	attDate := time.Now().UTC()
	if in.AttendanceDate != nil && *in.AttendanceDate != "" {
		t, err := time.Parse(time.RFC3339, *in.AttendanceDate)
		if err != nil {
			return errorResult(rec.SyncID, fmt.Sprintf("invalid attendance_date: %v", err))
		}
		attDate = t
	}

	a := domain.Attendance{
		ID:              uuid.New(),
		PersonID:        personID,
		TriageID:        triageID,
		CampusID:        campusID,
		ServiceTypeID:   serviceTypeID,
		ProfessionalID:  professionalID,
		Status:          in.Status,
		AttendanceDate:  attDate,
		Observations:    in.Observations,
		Recommendations: in.Recommendations,
		CreatedBy:       parseUserID(claims.Subject),
	}

	created, err := s.attendanceRepo.CreateWithSync(ctx, a, rec.SyncID)
	if err != nil {
		if errors.Is(err, domain.ErrDuplicate) {
			return conflictResult(rec.SyncID, nil, err.Error())
		}
		return errorResult(rec.SyncID, err.Error())
	}

	s.logCreate(ctx, "attendance", created.ID)
	return createdResult(rec.SyncID, created.ID)
}

// --- Pull -------------------------------------------------------------------

// Pull returns records updated since the cursor across the requested entity
// types, bounded by limit. The same limit is applied per entity to keep the
// response size predictable.
func (s *SyncService) Pull(ctx context.Context, since time.Time, entityTypes []string, limit int) (*domain.SyncPullResponse, error) {
	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return nil, domain.ErrForbidden
	}
	if limit <= 0 {
		limit = 100
	}

	out := []domain.SyncPullRecord{}
	hasMore := false
	for _, et := range entityTypes {
		more, err := s.pullEntity(ctx, et, campusID, since, limit, &out)
		if err != nil {
			return nil, err
		}
		if more {
			hasMore = true
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt.Before(out[j].UpdatedAt)
	})

	resp := &domain.SyncPullResponse{
		Records:         out,
		ServerTimestamp: time.Now().UTC(),
		HasMore:         hasMore,
	}
	if hasMore && len(out) > 0 {
		last := out[len(out)-1].UpdatedAt
		resp.NextSince = &last
	}
	return resp, nil
}

func (s *SyncService) pullEntity(ctx context.Context, entity string, campusID uuid.UUID, since time.Time, limit int, out *[]domain.SyncPullRecord) (bool, error) {
	switch entity {
	case domain.SyncEntityPerson:
		rows, err := s.personRepo.ListUpdatedSince(ctx, campusID, since, limit)
		if err != nil {
			return false, fmt.Errorf("pull person: %w", err)
		}
		for _, p := range rows {
			*out = append(*out, toPullRecord(domain.SyncEntityPerson, p.ID, p.UpdatedAt, p))
		}
		return len(rows) >= limit, nil
	case domain.SyncEntityTriage:
		rows, err := s.triageRepo.ListUpdatedSince(ctx, campusID, since, limit)
		if err != nil {
			return false, fmt.Errorf("pull triage: %w", err)
		}
		for _, t := range rows {
			*out = append(*out, toPullRecord(domain.SyncEntityTriage, t.ID, t.UpdatedAt, t))
		}
		return len(rows) >= limit, nil
	case domain.SyncEntityAttendance:
		rows, err := s.attendanceRepo.ListUpdatedSince(ctx, campusID, since, limit)
		if err != nil {
			return false, fmt.Errorf("pull attendance: %w", err)
		}
		for _, a := range rows {
			*out = append(*out, toPullRecord(domain.SyncEntityAttendance, a.ID, a.UpdatedAt, a))
		}
		return len(rows) >= limit, nil
	default:
		return false, fmt.Errorf("%w: %s", domain.ErrInvalidEntityType, entity)
	}
}

// --- Helpers ----------------------------------------------------------------

func (s *SyncService) logCreate(ctx context.Context, entityType string, id uuid.UUID) {
	if err := s.auditSvc.LogAction(ctx, AuditParams{
		ActionType:  "CREATE",
		EntityType:  entityType,
		EntityID:    &id,
		Module:      "sync",
		Description: "record created via sync push",
		Success:     true,
	}); err != nil {
		slog.ErrorContext(ctx, "syncService: audit failed",
			"error", err.Error(), "entity_type", entityType, "entity_id", id,
		)
	}
}

func decodeRecord(data map[string]any, target any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal record data: %w", err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("decode record data: %w", err)
	}
	return nil
}

func toPullRecord(entity string, id uuid.UUID, updated time.Time, payload any) domain.SyncPullRecord {
	raw, _ := json.Marshal(payload)
	m := map[string]any{}
	_ = json.Unmarshal(raw, &m)
	return domain.SyncPullRecord{
		EntityType: entity,
		ID:         id,
		Data:       m,
		UpdatedAt:  updated,
	}
}

func createdResult(syncID, serverID uuid.UUID) domain.SyncPushResult {
	id := serverID
	return domain.SyncPushResult{SyncID: syncID, Status: domain.SyncStatusCreated, ServerID: &id}
}

func conflictResult(syncID uuid.UUID, serverID *uuid.UUID, msg string) domain.SyncPushResult {
	return domain.SyncPushResult{SyncID: syncID, Status: domain.SyncStatusConflict, ServerID: serverID, Message: msg}
}

func errorResult(syncID uuid.UUID, msg string) domain.SyncPushResult {
	return domain.SyncPushResult{SyncID: syncID, Status: domain.SyncStatusError, Message: msg}
}
