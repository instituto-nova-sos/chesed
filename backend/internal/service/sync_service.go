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
	FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Person, error)
	FindBySyncID(ctx context.Context, syncID, campusID uuid.UUID) (*domain.Person, error)
	CreateWithSync(ctx context.Context, person domain.Person, address *domain.Address, syncID uuid.UUID) (*domain.Person, error)
	ListUpdatedSince(ctx context.Context, campusID uuid.UUID, since time.Time, limit int) ([]domain.Person, error)
}

// SyncTriageRepository is the subset of triage persistence used by the sync engine.
type SyncTriageRepository interface {
	FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Triage, error)
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
	campaignRepo   CampaignRefRepository
	auditSvc       *AuditService
	validate       *validator.Validate
}

// NewSyncService constructs a SyncService.
func NewSyncService(p SyncPersonRepository, t SyncTriageRepository, a SyncAttendanceRepository, c CampaignRefRepository, audit *AuditService) *SyncService {
	return &SyncService{
		personRepo:     p,
		triageRepo:     t,
		attendanceRepo: a,
		campaignRepo:   c,
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

	person, errResult := s.buildSyncPerson(ctx, rec, campusID)
	if errResult != nil {
		return *errResult
	}

	var address *domain.Address
	created, err := s.personRepo.CreateWithSync(ctx, person, address, rec.SyncID)
	if err != nil {
		return mapSyncCreateError(rec.SyncID, err)
	}

	s.logCreate(ctx, "person", created.ID)
	return createdResult(rec.SyncID, created.ID)
}

// buildSyncPerson decodes, validates, and maps a sync person record into a
// domain.Person. A non-nil result indicates an early-return error.
func (s *SyncService) buildSyncPerson(ctx context.Context, rec domain.SyncPushRecord, campusID uuid.UUID) (domain.Person, *domain.SyncPushResult) {
	var in syncPersonInput
	if err := decodeRecord(rec.Data, &in); err != nil {
		r := errorResult(rec.SyncID, err.Error())
		return domain.Person{}, &r
	}
	if err := s.validate.Struct(in); err != nil {
		r := errorResult(rec.SyncID, err.Error())
		return domain.Person{}, &r
	}

	birthDate, err := parseOptionalDate(in.BirthDate)
	if err != nil {
		r := errorResult(rec.SyncID, fmt.Sprintf("invalid birth_date: %v", err))
		return domain.Person{}, &r
	}

	nationality := "BRA"
	if in.Nationality != nil && *in.Nationality != "" {
		nationality = *in.Nationality
	}

	return domain.Person{
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
	}, nil
}

// mapSyncCreateError maps a repository create error to a conflict or error result.
func mapSyncCreateError(syncID uuid.UUID, err error) domain.SyncPushResult {
	if errors.Is(err, domain.ErrDuplicate) || errors.Is(err, domain.ErrDuplicateEmail) || errors.Is(err, domain.ErrDuplicatePhone) {
		return conflictResult(syncID, nil, err.Error())
	}
	return errorResult(syncID, err.Error())
}

// --- Triage push ------------------------------------------------------------

type syncTriageInput struct {
	PersonID       string   `json:"person_id" validate:"required,uuid"`
	MainComplaint  string   `json:"main_complaint" validate:"required"`
	TriageDate     *string  `json:"triage_date" validate:"omitempty"`
	Location       *string  `json:"location" validate:"omitempty,max=300"`
	Notes          *string  `json:"notes" validate:"omitempty"`
	RequestedTypes []string `json:"requested_service_type_ids" validate:"omitempty,dive,uuid"`
	CampaignID     *string  `json:"campaign_id" validate:"omitempty,uuid"`
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

	triage, errResult := s.buildSyncTriage(ctx, rec, campusID, claims)
	if errResult != nil {
		return *errResult
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

// buildSyncTriage decodes, validates, and maps a sync triage record. A non-nil
// result indicates an early-return error.
func (s *SyncService) buildSyncTriage(ctx context.Context, rec domain.SyncPushRecord, campusID uuid.UUID, claims auth.AuthClaims) (domain.Triage, *domain.SyncPushResult) {
	var in syncTriageInput
	if err := decodeRecord(rec.Data, &in); err != nil {
		r := errorResult(rec.SyncID, err.Error())
		return domain.Triage{}, &r
	}
	if err := s.validate.Struct(in); err != nil {
		r := errorResult(rec.SyncID, err.Error())
		return domain.Triage{}, &r
	}

	personID, err := uuid.Parse(in.PersonID)
	if err != nil {
		r := errorResult(rec.SyncID, fmt.Sprintf("invalid person_id: %v", err))
		return domain.Triage{}, &r
	}
	if errResult := s.requirePersonInCampus(ctx, rec.SyncID, personID, campusID); errResult != nil {
		return domain.Triage{}, errResult
	}
	campaignID, errResult := s.resolveCampaignLink(ctx, rec.SyncID, in.CampaignID, campusID)
	if errResult != nil {
		return domain.Triage{}, errResult
	}

	triageDate, err := parseSyncTimestamp(in.TriageDate)
	if err != nil {
		r := errorResult(rec.SyncID, fmt.Sprintf("invalid triage_date: %v", err))
		return domain.Triage{}, &r
	}

	requested, err := parseUUIDList(in.RequestedTypes)
	if err != nil {
		r := errorResult(rec.SyncID, err.Error())
		return domain.Triage{}, &r
	}

	triagedBy := uuid.Nil
	if id := parseUserID(claims.Subject); id != nil {
		triagedBy = *id
	}

	return domain.Triage{
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
		CampaignID:     campaignID,
	}, nil
}

// requirePersonInCampus rejects a pushed record whose referenced person does
// not belong to the caller's campus (security Finding 3, threat T3). A campus
// mismatch surfaces as ErrNotFound from the campus-scoped lookup and yields a
// generic error result — the response never reveals whether the person exists
// in another campus. Any other lookup error also rejects the record.
func (s *SyncService) requirePersonInCampus(ctx context.Context, syncID, personID, campusID uuid.UUID) *domain.SyncPushResult {
	if _, err := s.personRepo.FindByID(ctx, personID, campusID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			r := errorResult(syncID, "person_id not found in campus")
			return &r
		}
		r := errorResult(syncID, err.Error())
		return &r
	}
	return nil
}

// requireTriageInCampus is the triage-reference counterpart of
// requirePersonInCampus, guarding an optional attendance.triage_id link.
func (s *SyncService) requireTriageInCampus(ctx context.Context, syncID, triageID, campusID uuid.UUID) *domain.SyncPushResult {
	if _, err := s.triageRepo.FindByID(ctx, triageID, campusID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			r := errorResult(syncID, "triage_id not found in campus")
			return &r
		}
		r := errorResult(syncID, err.Error())
		return &r
	}
	return nil
}

// resolveCampaignLink validates an optional pushed campaign reference against
// the caller's campus, reusing resolveCampaignRef so rejections stay generic
// (threat model T3) and surface as a per-record error result (S07.6).
func (s *SyncService) resolveCampaignLink(ctx context.Context, syncID uuid.UUID, raw *string, campusID uuid.UUID) (*uuid.UUID, *domain.SyncPushResult) {
	id, err := resolveCampaignRef(ctx, s.campaignRepo, raw, campusID)
	if err != nil {
		r := errorResult(syncID, err.Error())
		return nil, &r
	}
	return id, nil
}

// parseSyncTimestamp parses an optional RFC3339 timestamp, defaulting to now.
func parseSyncTimestamp(raw *string) (time.Time, error) {
	if raw == nil || *raw == "" {
		return time.Now().UTC(), nil
	}
	return time.Parse(time.RFC3339, *raw)
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
	CampaignID      *string `json:"campaign_id" validate:"omitempty,uuid"`
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

	a, errResult := s.buildSyncAttendance(ctx, rec, campusID, claims)
	if errResult != nil {
		return *errResult
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

// buildSyncAttendance decodes, validates, and maps a sync attendance record. A
// non-nil result indicates an early-return error.
// syncAttendanceRefs holds the resolved UUID references of an attendance push
// record after parsing and campus-scoped validation.
type syncAttendanceRefs struct {
	personID       uuid.UUID
	serviceTypeID  uuid.UUID
	professionalID uuid.UUID
	triageID       *uuid.UUID
	campaignID     *uuid.UUID
}

func (s *SyncService) buildSyncAttendance(ctx context.Context, rec domain.SyncPushRecord, campusID uuid.UUID, claims auth.AuthClaims) (domain.Attendance, *domain.SyncPushResult) {
	var in syncAttendanceInput
	if err := decodeRecord(rec.Data, &in); err != nil {
		r := errorResult(rec.SyncID, err.Error())
		return domain.Attendance{}, &r
	}
	if err := s.validate.Struct(in); err != nil {
		r := errorResult(rec.SyncID, err.Error())
		return domain.Attendance{}, &r
	}

	refs, errResult := s.resolveAttendanceRefs(ctx, rec, in, campusID)
	if errResult != nil {
		return domain.Attendance{}, errResult
	}

	attDate, err := parseSyncTimestamp(in.AttendanceDate)
	if err != nil {
		r := errorResult(rec.SyncID, fmt.Sprintf("invalid attendance_date: %v", err))
		return domain.Attendance{}, &r
	}

	return domain.Attendance{
		ID:              uuid.New(),
		PersonID:        refs.personID,
		TriageID:        refs.triageID,
		CampusID:        campusID,
		ServiceTypeID:   refs.serviceTypeID,
		ProfessionalID:  refs.professionalID,
		Status:          in.Status,
		AttendanceDate:  attDate,
		Observations:    in.Observations,
		Recommendations: in.Recommendations,
		CreatedBy:       parseUserID(claims.Subject),
		CampaignID:      refs.campaignID,
	}, nil
}

// resolveAttendanceRefs parses and campus-validates the UUID references of an
// attendance record. The referenced person (and optional triage) must belong
// to the caller's campus (security Finding 3). A non-nil result is an
// early-return rejection.
func (s *SyncService) resolveAttendanceRefs(ctx context.Context, rec domain.SyncPushRecord, in syncAttendanceInput, campusID uuid.UUID) (syncAttendanceRefs, *domain.SyncPushResult) {
	personID, err := uuid.Parse(in.PersonID)
	if err != nil {
		r := errorResult(rec.SyncID, fmt.Sprintf("invalid person_id: %v", err))
		return syncAttendanceRefs{}, &r
	}
	if errResult := s.requirePersonInCampus(ctx, rec.SyncID, personID, campusID); errResult != nil {
		return syncAttendanceRefs{}, errResult
	}
	serviceTypeID, err := uuid.Parse(in.ServiceTypeID)
	if err != nil {
		r := errorResult(rec.SyncID, fmt.Sprintf("invalid service_type_id: %v", err))
		return syncAttendanceRefs{}, &r
	}
	professionalID, err := uuid.Parse(in.ProfessionalID)
	if err != nil {
		r := errorResult(rec.SyncID, fmt.Sprintf("invalid professional_id: %v", err))
		return syncAttendanceRefs{}, &r
	}
	triageID, err := parseOptionalUUID(in.TriageID)
	if err != nil {
		r := errorResult(rec.SyncID, fmt.Sprintf("invalid triage_id: %v", err))
		return syncAttendanceRefs{}, &r
	}
	if triageID != nil {
		if errResult := s.requireTriageInCampus(ctx, rec.SyncID, *triageID, campusID); errResult != nil {
			return syncAttendanceRefs{}, errResult
		}
	}
	campaignID, errResult := s.resolveCampaignLink(ctx, rec.SyncID, in.CampaignID, campusID)
	if errResult != nil {
		return syncAttendanceRefs{}, errResult
	}
	return syncAttendanceRefs{
		personID:       personID,
		serviceTypeID:  serviceTypeID,
		professionalID: professionalID,
		triageID:       triageID,
		campaignID:     campaignID,
	}, nil
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
		return s.pullPersons(ctx, campusID, since, limit, out)
	case domain.SyncEntityTriage:
		return s.pullTriages(ctx, campusID, since, limit, out)
	case domain.SyncEntityAttendance:
		return s.pullAttendances(ctx, campusID, since, limit, out)
	default:
		return false, fmt.Errorf("%w: %s", domain.ErrInvalidEntityType, entity)
	}
}

func (s *SyncService) pullPersons(ctx context.Context, campusID uuid.UUID, since time.Time, limit int, out *[]domain.SyncPullRecord) (bool, error) {
	rows, err := s.personRepo.ListUpdatedSince(ctx, campusID, since, limit)
	if err != nil {
		return false, fmt.Errorf("pull person: %w", err)
	}
	for _, p := range rows {
		*out = append(*out, toPullRecord(domain.SyncEntityPerson, p.ID, p.UpdatedAt, p))
	}
	return len(rows) >= limit, nil
}

func (s *SyncService) pullTriages(ctx context.Context, campusID uuid.UUID, since time.Time, limit int, out *[]domain.SyncPullRecord) (bool, error) {
	rows, err := s.triageRepo.ListUpdatedSince(ctx, campusID, since, limit)
	if err != nil {
		return false, fmt.Errorf("pull triage: %w", err)
	}
	for _, t := range rows {
		*out = append(*out, toPullRecord(domain.SyncEntityTriage, t.ID, t.UpdatedAt, t))
	}
	return len(rows) >= limit, nil
}

func (s *SyncService) pullAttendances(ctx context.Context, campusID uuid.UUID, since time.Time, limit int, out *[]domain.SyncPullRecord) (bool, error) {
	rows, err := s.attendanceRepo.ListUpdatedSince(ctx, campusID, since, limit)
	if err != nil {
		return false, fmt.Errorf("pull attendance: %w", err)
	}
	for _, a := range rows {
		*out = append(*out, toPullRecord(domain.SyncEntityAttendance, a.ID, a.UpdatedAt, a))
	}
	return len(rows) >= limit, nil
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
	m := map[string]any{}
	// payload is always a concrete domain struct, so these round-trips do not
	// fail in practice; we still surface a failure rather than silently dropping
	// it (consistent with the no-ignored-errors rule) and emit an empty Data map.
	if raw, err := json.Marshal(payload); err != nil {
		slog.Error("syncService.toPullRecord: marshal payload", "error", err.Error(), "entity", entity, "id", id)
	} else if err := json.Unmarshal(raw, &m); err != nil {
		slog.Error("syncService.toPullRecord: unmarshal payload", "error", err.Error(), "entity", entity, "id", id)
	}
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
