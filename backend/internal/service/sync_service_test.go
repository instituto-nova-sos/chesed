package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// --- Mocks --------------------------------------------------------------------
// S12.2: sync conflict results carry a lean, non-PII server_data projection.

type MockSyncPersonRepo struct{ mock.Mock }

func (m *MockSyncPersonRepo) FindByEmail(ctx context.Context, email string, campusID uuid.UUID) (*domain.Person, error) {
	args := m.Called(ctx, email, campusID)
	if p, ok := args.Get(0).(*domain.Person); ok {
		return p, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSyncPersonRepo) FindBySyncID(ctx context.Context, syncID, campusID uuid.UUID) (*domain.Person, error) {
	args := m.Called(ctx, syncID, campusID)
	if p, ok := args.Get(0).(*domain.Person); ok {
		return p, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSyncPersonRepo) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Person, error) {
	args := m.Called(ctx, id, campusID)
	if p, ok := args.Get(0).(*domain.Person); ok {
		return p, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSyncPersonRepo) CreateWithSync(ctx context.Context, person domain.Person, address *domain.Address, syncID uuid.UUID) (*domain.Person, error) {
	args := m.Called(ctx, person, address, syncID)
	if p, ok := args.Get(0).(*domain.Person); ok {
		return p, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSyncPersonRepo) ListUpdatedSince(ctx context.Context, campusID uuid.UUID, since time.Time, limit int) ([]domain.Person, error) {
	args := m.Called(ctx, campusID, since, limit)
	if v, ok := args.Get(0).([]domain.Person); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

type MockSyncTriageRepo struct{ mock.Mock }

func (m *MockSyncTriageRepo) FindBySyncID(ctx context.Context, syncID, campusID uuid.UUID) (*domain.Triage, error) {
	args := m.Called(ctx, syncID, campusID)
	if v, ok := args.Get(0).(*domain.Triage); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSyncTriageRepo) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Triage, error) {
	args := m.Called(ctx, id, campusID)
	if v, ok := args.Get(0).(*domain.Triage); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSyncTriageRepo) CreateWithSync(ctx context.Context, triage domain.Triage, syncID uuid.UUID) (*domain.Triage, error) {
	args := m.Called(ctx, triage, syncID)
	if v, ok := args.Get(0).(*domain.Triage); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSyncTriageRepo) ListUpdatedSince(ctx context.Context, campusID uuid.UUID, since time.Time, limit int) ([]domain.Triage, error) {
	args := m.Called(ctx, campusID, since, limit)
	if v, ok := args.Get(0).([]domain.Triage); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

type MockSyncAttendanceRepo struct{ mock.Mock }

func (m *MockSyncAttendanceRepo) FindBySyncID(ctx context.Context, syncID, campusID uuid.UUID) (*domain.Attendance, error) {
	args := m.Called(ctx, syncID, campusID)
	if v, ok := args.Get(0).(*domain.Attendance); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSyncAttendanceRepo) CreateWithSync(ctx context.Context, a domain.Attendance, syncID uuid.UUID) (*domain.Attendance, error) {
	args := m.Called(ctx, a, syncID)
	if v, ok := args.Get(0).(*domain.Attendance); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSyncAttendanceRepo) ListUpdatedSince(ctx context.Context, campusID uuid.UUID, since time.Time, limit int) ([]domain.Attendance, error) {
	args := m.Called(ctx, campusID, since, limit)
	if v, ok := args.Get(0).([]domain.Attendance); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

// --- Test helpers -------------------------------------------------------------

func ctxWithCampus(t *testing.T, campusID uuid.UUID) context.Context {
	t.Helper()
	return auth.NewContext(context.Background(), auth.AuthClaims{
		Subject:  uuid.New().String(),
		Email:    "tester@example.com",
		Roles:    []string{"VOLUNTEER"},
		CampusID: campusID,
	})
}

func newSyncService(t *testing.T) (*SyncService, *MockSyncPersonRepo, *MockSyncTriageRepo, *MockSyncAttendanceRepo, *MockAuditRepository) {
	t.Helper()
	svc, pRepo, tRepo, aRepo, _, auditRepo := newSyncServiceWithCampaign(t)
	return svc, pRepo, tRepo, aRepo, auditRepo
}

func newSyncServiceWithCampaign(t *testing.T) (*SyncService, *MockSyncPersonRepo, *MockSyncTriageRepo, *MockSyncAttendanceRepo, *MockCampaignRepository, *MockAuditRepository) {
	t.Helper()
	pRepo := new(MockSyncPersonRepo)
	tRepo := new(MockSyncTriageRepo)
	aRepo := new(MockSyncAttendanceRepo)
	cRepo := new(MockCampaignRepository)
	auditRepo := new(MockAuditRepository)
	auditSvc := NewAuditService(auditRepo)
	svc := NewSyncService(pRepo, tRepo, aRepo, cRepo, auditSvc)
	return svc, pRepo, tRepo, aRepo, cRepo, auditRepo
}

// --- PushBatch ----------------------------------------------------------------

func TestSyncService_Push_EmptyBatch(t *testing.T) {
	svc, _, _, _, _ := newSyncService(t)
	ctx := ctxWithCampus(t, uuid.New())

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{}})
	require.NoError(t, err)
	assert.Empty(t, resp.Results)
	assert.False(t, resp.ServerTimestamp.IsZero())
}

func TestSyncService_Push_MissingCampusForbidden(t *testing.T) {
	svc, _, _, _, _ := newSyncService(t)

	_, err := svc.Push(context.Background(), domain.SyncPushRequest{Records: []domain.SyncPushRecord{
		validPersonRecord(),
	}})
	assert.True(t, errors.Is(err, domain.ErrForbidden), "expected ErrForbidden, got %v", err)
}

func TestSyncService_Push_OversizeBatchRejected(t *testing.T) {
	svc, _, _, _, _ := newSyncService(t)
	ctx := ctxWithCampus(t, uuid.New())

	recs := make([]domain.SyncPushRecord, domain.MaxSyncBatchSize+1)
	for i := range recs {
		recs[i] = validPersonRecord()
	}
	_, err := svc.Push(ctx, domain.SyncPushRequest{Records: recs})
	assert.True(t, errors.Is(err, domain.ErrBatchTooLarge))
}

func TestSyncService_Push_PersonNewCreated(t *testing.T) {
	svc, pRepo, _, _, auditRepo := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)

	rec := validPersonRecord()
	pRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("CreateWithSync", ctx,
		mock.MatchedBy(func(p domain.Person) bool { return p.FullName == "Maria" && p.CampusID == campusID }),
		mock.AnythingOfType("*domain.Address"),
		rec.SyncID,
	).Return(&domain.Person{ID: uuid.New(), FullName: "Maria", CampusID: campusID}, nil).Once()
	auditRepo.On("Create", ctx, mock.AnythingOfType("domain.AuditLog")).Return(nil).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, domain.SyncStatusCreated, resp.Results[0].Status)
	assert.Equal(t, rec.SyncID, resp.Results[0].SyncID)
	require.NotNil(t, resp.Results[0].ServerID)

	pRepo.AssertExpectations(t)
	auditRepo.AssertExpectations(t)
}

func TestSyncService_Push_PersonIdempotent(t *testing.T) {
	svc, pRepo, _, _, auditRepo := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)

	existingID := uuid.New()
	rec := validPersonRecord()
	pRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).
		Return(&domain.Person{ID: existingID, CampusID: campusID}, nil).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, domain.SyncStatusCreated, resp.Results[0].Status, "idempotent re-push reports created")
	require.NotNil(t, resp.Results[0].ServerID)
	assert.Equal(t, existingID, *resp.Results[0].ServerID)

	pRepo.AssertNotCalled(t, "CreateWithSync", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	auditRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestSyncService_Push_PersonDuplicateDocumentConflict(t *testing.T) {
	svc, pRepo, _, _, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)

	rec := validPersonRecord()
	pRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("CreateWithSync", ctx,
		mock.AnythingOfType("domain.Person"),
		mock.AnythingOfType("*domain.Address"),
		rec.SyncID,
	).Return(nil, domain.ErrDuplicate).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, domain.SyncStatusConflict, resp.Results[0].Status)
}

// --- S12.2 conflict server_data projection ------------------------------------

func personEmailRecord(email string) domain.SyncPushRecord {
	return domain.SyncPushRecord{
		EntityType: domain.SyncEntityPerson,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"full_name":     "Maria",
			"document_type": "CPF",
			"nationality":   "BRA",
			"email":         email,
		},
	}
}

func TestSyncService_Push_PersonConflictReturnsLeanServerData(t *testing.T) {
	svc, pRepo, _, _, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)

	email := "maria@example.com"
	phone := "+5511999998888"
	serverUpdated := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	serverID := uuid.New()
	rec := personEmailRecord(email)

	pRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("CreateWithSync", ctx,
		mock.AnythingOfType("domain.Person"),
		mock.AnythingOfType("*domain.Address"),
		rec.SyncID,
	).Return(nil, domain.ErrDuplicateEmail).Once()
	pRepo.On("FindByEmail", ctx, email, campusID).Return(&domain.Person{
		ID:             serverID,
		FullName:       "Maria Silva",
		Email:          &email,
		Phone:          &phone,
		DocumentNumber: strPtr("12345678900"), // MUST NOT leak into server_data
		CampusID:       campusID,
		UpdatedAt:      serverUpdated,
	}, nil).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	res := resp.Results[0]
	assert.Equal(t, domain.SyncStatusConflict, res.Status)
	require.NotNil(t, res.ServerID)
	assert.Equal(t, serverID, *res.ServerID)
	require.NotNil(t, res.ServerData)
	assert.Equal(t, "Maria Silva", res.ServerData["full_name"])
	assert.Equal(t, email, res.ServerData["email"])
	assert.Equal(t, phone, res.ServerData["phone"])
	assert.NotContains(t, res.ServerData, "document_number", "document_number is PII and must not leak")
	assert.NotContains(t, res.ServerData, "birth_date")
	assert.NotContains(t, res.ServerData, "gender")
	require.NotNil(t, res.ServerUpdatedAt)
	assert.True(t, serverUpdated.Equal(*res.ServerUpdatedAt))
	pRepo.AssertExpectations(t)
}

func TestSyncService_Push_PersonConflictWithoutServerRowOmitsServerData(t *testing.T) {
	svc, pRepo, _, _, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)

	// Validation-only / document-or-phone conflict where the server row cannot
	// be resolved by email (no email in payload) — back-compat: no server_data.
	rec := validPersonRecord()
	pRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("CreateWithSync", ctx,
		mock.AnythingOfType("domain.Person"),
		mock.AnythingOfType("*domain.Address"),
		rec.SyncID,
	).Return(nil, domain.ErrDuplicate).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	res := resp.Results[0]
	assert.Equal(t, domain.SyncStatusConflict, res.Status)
	assert.Nil(t, res.ServerData, "no resolvable server row -> server_data omitted (back-compat)")
	assert.Nil(t, res.ServerUpdatedAt)
	pRepo.AssertNotCalled(t, "FindByEmail", mock.Anything, mock.Anything, mock.Anything)
}

func TestSyncService_Push_PersonConflictLookupErrorFallsBack(t *testing.T) {
	svc, pRepo, _, _, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)

	email := "maria@example.com"
	rec := personEmailRecord(email)
	pRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("CreateWithSync", ctx,
		mock.AnythingOfType("domain.Person"),
		mock.AnythingOfType("*domain.Address"),
		rec.SyncID,
	).Return(nil, domain.ErrDuplicateEmail).Once()
	pRepo.On("FindByEmail", ctx, email, campusID).
		Return(nil, errors.New("connection lost")).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	res := resp.Results[0]
	assert.Equal(t, domain.SyncStatusConflict, res.Status, "lookup failure must not break the push")
	assert.Nil(t, res.ServerData)
}

func TestSyncService_Push_TriageConflictReturnsServerData(t *testing.T) {
	svc, pRepo, tRepo, _, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	personID := uuid.New()
	serverID := uuid.New()
	serverUpdated := time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC)
	serverTriageDate := time.Date(2026, 6, 2, 8, 30, 0, 0, time.UTC)

	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityTriage,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":      personID.String(),
			"main_complaint": "x",
		},
	}
	tRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("FindByID", ctx, mock.AnythingOfType("uuid.UUID"), campusID).
		Return(&domain.Person{CampusID: campusID}, nil).Once()
	tRepo.On("CreateWithSync", ctx, mock.AnythingOfType("domain.Triage"), rec.SyncID).
		Return(nil, domain.ErrDuplicate).Once()
	tRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(&domain.Triage{
		ID:            serverID,
		PersonID:      personID,
		MainComplaint: "Dor de cabeça",
		TriageDate:    serverTriageDate,
		Notes:         strPtr("sensitive clinical note"), // MUST NOT leak
		CampusID:      campusID,
		UpdatedAt:     serverUpdated,
	}, nil).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	res := resp.Results[0]
	assert.Equal(t, domain.SyncStatusConflict, res.Status)
	require.NotNil(t, res.ServerID)
	assert.Equal(t, serverID, *res.ServerID)
	require.NotNil(t, res.ServerData)
	assert.Equal(t, "Dor de cabeça", res.ServerData["main_complaint"])
	assert.NotContains(t, res.ServerData, "notes", "clinical notes are PII and must not leak")
	require.NotNil(t, res.ServerUpdatedAt)
	assert.True(t, serverUpdated.Equal(*res.ServerUpdatedAt))
}

func TestSyncService_Push_AttendanceConflictReturnsServerData(t *testing.T) {
	svc, pRepo, _, aRepo, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	serverID := uuid.New()
	serverUpdated := time.Date(2026, 6, 3, 11, 0, 0, 0, time.UTC)

	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityAttendance,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":       uuid.New().String(),
			"service_type_id": uuid.New().String(),
			"professional_id": uuid.New().String(),
			"status":          domain.AttendanceStatusScheduled,
		},
	}
	aRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("FindByID", ctx, mock.AnythingOfType("uuid.UUID"), campusID).
		Return(&domain.Person{CampusID: campusID}, nil).Once()
	aRepo.On("CreateWithSync", ctx, mock.AnythingOfType("domain.Attendance"), rec.SyncID).
		Return(nil, domain.ErrDuplicate).Once()
	aRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(&domain.Attendance{
		ID:        serverID,
		Status:    domain.AttendanceStatusInProgress,
		CampusID:  campusID,
		UpdatedAt: serverUpdated,
	}, nil).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	res := resp.Results[0]
	assert.Equal(t, domain.SyncStatusConflict, res.Status)
	require.NotNil(t, res.ServerID)
	assert.Equal(t, serverID, *res.ServerID)
	require.NotNil(t, res.ServerData)
	assert.Equal(t, domain.AttendanceStatusInProgress, res.ServerData["status"])
	require.NotNil(t, res.ServerUpdatedAt)
	assert.True(t, serverUpdated.Equal(*res.ServerUpdatedAt))
}

func TestSyncService_Push_UnknownEntityTypeReportsErrorPerRecord(t *testing.T) {
	svc, _, _, _, _ := newSyncService(t)
	ctx := ctxWithCampus(t, uuid.New())

	// Request-level Validate() rejects unknown entity types up front. The
	// per-record error path is reachable only via PushSkippingValidation,
	// which the service exposes for callers that pre-validated. We exercise
	// it directly here.
	bad := domain.SyncPushRecord{EntityType: "campaign", SyncID: uuid.New(), Data: map[string]any{"x": 1}}

	resp, err := svc.PushSkippingValidation(ctx, []domain.SyncPushRecord{bad})
	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, domain.SyncStatusError, resp.Results[0].Status)
}

func TestSyncService_Push_TriageNewCreated(t *testing.T) {
	svc, pRepo, tRepo, _, auditRepo := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	personID := uuid.New()

	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityTriage,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":      personID.String(),
			"main_complaint": "Dor de cabeça",
		},
	}

	tRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("FindByID", ctx, personID, campusID).
		Return(&domain.Person{ID: personID, CampusID: campusID}, nil).Once()
	tRepo.On("CreateWithSync", ctx,
		mock.MatchedBy(func(tr domain.Triage) bool {
			return tr.PersonID == personID && tr.CampusID == campusID && tr.MainComplaint == "Dor de cabeça"
		}),
		rec.SyncID,
	).Return(&domain.Triage{ID: uuid.New(), PersonID: personID, CampusID: campusID}, nil).Once()
	auditRepo.On("Create", ctx, mock.AnythingOfType("domain.AuditLog")).Return(nil).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, domain.SyncStatusCreated, resp.Results[0].Status)
	tRepo.AssertExpectations(t)
}

func TestSyncService_Push_TriageIdempotent(t *testing.T) {
	svc, _, tRepo, _, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	existingID := uuid.New()
	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityTriage,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":      uuid.New().String(),
			"main_complaint": "x",
		},
	}

	tRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).
		Return(&domain.Triage{ID: existingID, CampusID: campusID}, nil).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	require.Equal(t, domain.SyncStatusCreated, resp.Results[0].Status)
	require.NotNil(t, resp.Results[0].ServerID)
	assert.Equal(t, existingID, *resp.Results[0].ServerID)
	tRepo.AssertNotCalled(t, "CreateWithSync", mock.Anything, mock.Anything, mock.Anything)
}

func TestSyncService_Push_TriageInvalidPersonIDReportsError(t *testing.T) {
	svc, _, tRepo, _, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)

	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityTriage,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":      "not-a-uuid",
			"main_complaint": "x",
		},
	}
	tRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	assert.Equal(t, domain.SyncStatusError, resp.Results[0].Status)
}

func TestSyncService_Push_AttendanceNewCreated(t *testing.T) {
	svc, pRepo, _, aRepo, auditRepo := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	personID := uuid.New()
	serviceTypeID := uuid.New()
	professionalID := uuid.New()

	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityAttendance,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":       personID.String(),
			"service_type_id": serviceTypeID.String(),
			"professional_id": professionalID.String(),
			"status":          domain.AttendanceStatusScheduled,
		},
	}

	aRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("FindByID", ctx, personID, campusID).
		Return(&domain.Person{ID: personID, CampusID: campusID}, nil).Once()
	aRepo.On("CreateWithSync", ctx,
		mock.MatchedBy(func(a domain.Attendance) bool {
			return a.PersonID == personID && a.ServiceTypeID == serviceTypeID && a.Status == domain.AttendanceStatusScheduled
		}),
		rec.SyncID,
	).Return(&domain.Attendance{ID: uuid.New(), PersonID: personID, CampusID: campusID}, nil).Once()
	auditRepo.On("Create", ctx, mock.AnythingOfType("domain.AuditLog")).Return(nil).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	assert.Equal(t, domain.SyncStatusCreated, resp.Results[0].Status)
	aRepo.AssertExpectations(t)
}

func TestSyncService_Push_AttendanceInvalidStatus(t *testing.T) {
	svc, _, _, aRepo, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)

	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityAttendance,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":       uuid.New().String(),
			"service_type_id": uuid.New().String(),
			"professional_id": uuid.New().String(),
			"status":          "FOLLOW_UP", // Phase 2; rejected in Phase 1
		},
	}
	aRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	assert.Equal(t, domain.SyncStatusError, resp.Results[0].Status)
}

func TestSyncService_Push_TriageFindBySyncIDInternalErrorReportsError(t *testing.T) {
	svc, _, tRepo, _, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityTriage,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":      uuid.New().String(),
			"main_complaint": "x",
		},
	}
	tRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).
		Return(nil, errors.New("connection lost")).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, domain.SyncStatusError, resp.Results[0].Status)
	assert.Contains(t, resp.Results[0].Message, "connection lost")
	tRepo.AssertNotCalled(t, "CreateWithSync", mock.Anything, mock.Anything, mock.Anything)
}

func TestSyncService_Push_TriageWithValidTriageDateAndRequestedServices(t *testing.T) {
	svc, pRepo, tRepo, _, auditRepo := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	personID := uuid.New()
	serviceTypeID := uuid.New()
	triageDate := "2026-05-15T14:30:00Z"
	parsedDate, _ := time.Parse(time.RFC3339, triageDate)

	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityTriage,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":                  personID.String(),
			"main_complaint":             "Dor",
			"triage_date":                triageDate,
			"requested_service_type_ids": []string{serviceTypeID.String()},
		},
	}
	tRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("FindByID", ctx, personID, campusID).
		Return(&domain.Person{ID: personID, CampusID: campusID}, nil).Once()
	tRepo.On("CreateWithSync", ctx,
		mock.MatchedBy(func(tr domain.Triage) bool {
			return tr.TriageDate.Equal(parsedDate) &&
				len(tr.RequestedTypes) == 1 &&
				tr.RequestedTypes[0] == serviceTypeID
		}),
		rec.SyncID,
	).Return(&domain.Triage{ID: uuid.New(), CampusID: campusID, TriageDate: parsedDate}, nil).Once()
	auditRepo.On("Create", ctx, mock.AnythingOfType("domain.AuditLog")).Return(nil).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	assert.Equal(t, domain.SyncStatusCreated, resp.Results[0].Status)
	tRepo.AssertExpectations(t)
}

func TestSyncService_Push_TriageInvalidTriageDateReportsError(t *testing.T) {
	svc, pRepo, tRepo, _, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityTriage,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":      uuid.New().String(),
			"main_complaint": "x",
			"triage_date":    "not-a-date",
		},
	}
	tRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("FindByID", ctx, mock.AnythingOfType("uuid.UUID"), campusID).
		Return(&domain.Person{CampusID: campusID}, nil).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	assert.Equal(t, domain.SyncStatusError, resp.Results[0].Status)
	assert.Contains(t, resp.Results[0].Message, "triage_date")
}

func TestSyncService_Push_TriageCreateDuplicateReturnsConflict(t *testing.T) {
	svc, pRepo, tRepo, _, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityTriage,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":      uuid.New().String(),
			"main_complaint": "x",
		},
	}
	tRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("FindByID", ctx, mock.AnythingOfType("uuid.UUID"), campusID).
		Return(&domain.Person{CampusID: campusID}, nil).Once()
	tRepo.On("CreateWithSync", ctx, mock.AnythingOfType("domain.Triage"), rec.SyncID).
		Return(nil, domain.ErrDuplicate).Once()
	// S12.2: on conflict the service re-resolves the server row by sync_id; when
	// it is unresolvable the result stays the plain conflict (back-compat).
	tRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	assert.Equal(t, domain.SyncStatusConflict, resp.Results[0].Status)
	assert.Nil(t, resp.Results[0].ServerData)
}

func TestSyncService_Push_TriageCreateGenericErrorReturnsError(t *testing.T) {
	svc, pRepo, tRepo, _, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityTriage,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":      uuid.New().String(),
			"main_complaint": "x",
		},
	}
	tRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("FindByID", ctx, mock.AnythingOfType("uuid.UUID"), campusID).
		Return(&domain.Person{CampusID: campusID}, nil).Once()
	tRepo.On("CreateWithSync", ctx, mock.AnythingOfType("domain.Triage"), rec.SyncID).
		Return(nil, errors.New("FK violation")).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	assert.Equal(t, domain.SyncStatusError, resp.Results[0].Status)
	assert.Contains(t, resp.Results[0].Message, "FK violation")
}

func TestSyncService_Push_AttendanceFindBySyncIDInternalErrorReportsError(t *testing.T) {
	svc, _, _, aRepo, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityAttendance,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":       uuid.New().String(),
			"service_type_id": uuid.New().String(),
			"professional_id": uuid.New().String(),
			"status":          domain.AttendanceStatusScheduled,
		},
	}
	aRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).
		Return(nil, errors.New("DB unreachable")).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	assert.Equal(t, domain.SyncStatusError, resp.Results[0].Status)
	aRepo.AssertNotCalled(t, "CreateWithSync", mock.Anything, mock.Anything, mock.Anything)
}

func TestSyncService_Push_AttendanceIdempotent(t *testing.T) {
	svc, _, _, aRepo, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	existingID := uuid.New()
	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityAttendance,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":       uuid.New().String(),
			"service_type_id": uuid.New().String(),
			"professional_id": uuid.New().String(),
			"status":          domain.AttendanceStatusScheduled,
		},
	}
	aRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).
		Return(&domain.Attendance{ID: existingID, CampusID: campusID}, nil).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	assert.Equal(t, domain.SyncStatusCreated, resp.Results[0].Status)
	require.NotNil(t, resp.Results[0].ServerID)
	assert.Equal(t, existingID, *resp.Results[0].ServerID)
	aRepo.AssertNotCalled(t, "CreateWithSync", mock.Anything, mock.Anything, mock.Anything)
}

func TestSyncService_Push_AttendanceWithTriageIDAndAttendanceDate(t *testing.T) {
	svc, pRepo, tRepo, aRepo, auditRepo := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	personID := uuid.New()
	serviceTypeID := uuid.New()
	professionalID := uuid.New()
	triageID := uuid.New()
	attDate := "2026-05-20T09:00:00Z"
	parsedDate, _ := time.Parse(time.RFC3339, attDate)

	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityAttendance,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":       personID.String(),
			"triage_id":       triageID.String(),
			"service_type_id": serviceTypeID.String(),
			"professional_id": professionalID.String(),
			"status":          domain.AttendanceStatusScheduled,
			"attendance_date": attDate,
		},
	}
	aRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("FindByID", ctx, personID, campusID).
		Return(&domain.Person{ID: personID, CampusID: campusID}, nil).Once()
	tRepo.On("FindByID", ctx, triageID, campusID).
		Return(&domain.Triage{ID: triageID, CampusID: campusID}, nil).Once()
	aRepo.On("CreateWithSync", ctx,
		mock.MatchedBy(func(a domain.Attendance) bool {
			return a.TriageID != nil && *a.TriageID == triageID &&
				a.AttendanceDate.Equal(parsedDate)
		}),
		rec.SyncID,
	).Return(&domain.Attendance{ID: uuid.New(), CampusID: campusID}, nil).Once()
	auditRepo.On("Create", ctx, mock.AnythingOfType("domain.AuditLog")).Return(nil).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	assert.Equal(t, domain.SyncStatusCreated, resp.Results[0].Status)
	aRepo.AssertExpectations(t)
}

func TestSyncService_Push_AttendanceInvalidAttendanceDateReportsError(t *testing.T) {
	svc, pRepo, _, aRepo, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityAttendance,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":       uuid.New().String(),
			"service_type_id": uuid.New().String(),
			"professional_id": uuid.New().String(),
			"status":          domain.AttendanceStatusScheduled,
			"attendance_date": "not-a-date",
		},
	}
	aRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("FindByID", ctx, mock.AnythingOfType("uuid.UUID"), campusID).
		Return(&domain.Person{CampusID: campusID}, nil).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	assert.Equal(t, domain.SyncStatusError, resp.Results[0].Status)
	assert.Contains(t, resp.Results[0].Message, "attendance_date")
}

func TestSyncService_Push_AttendanceCreateDuplicateReturnsConflict(t *testing.T) {
	svc, pRepo, _, aRepo, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityAttendance,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":       uuid.New().String(),
			"service_type_id": uuid.New().String(),
			"professional_id": uuid.New().String(),
			"status":          domain.AttendanceStatusScheduled,
		},
	}
	aRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("FindByID", ctx, mock.AnythingOfType("uuid.UUID"), campusID).
		Return(&domain.Person{CampusID: campusID}, nil).Once()
	aRepo.On("CreateWithSync", ctx, mock.AnythingOfType("domain.Attendance"), rec.SyncID).
		Return(nil, domain.ErrDuplicate).Once()
	// S12.2: on conflict the service re-resolves the server row by sync_id; when
	// it is unresolvable the result stays the plain conflict (back-compat).
	aRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	assert.Equal(t, domain.SyncStatusConflict, resp.Results[0].Status)
	assert.Nil(t, resp.Results[0].ServerData)
}

func TestSyncService_Push_AttendanceCreateGenericErrorReturnsError(t *testing.T) {
	svc, pRepo, _, aRepo, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityAttendance,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":       uuid.New().String(),
			"service_type_id": uuid.New().String(),
			"professional_id": uuid.New().String(),
			"status":          domain.AttendanceStatusScheduled,
		},
	}
	aRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("FindByID", ctx, mock.AnythingOfType("uuid.UUID"), campusID).
		Return(&domain.Person{CampusID: campusID}, nil).Once()
	aRepo.On("CreateWithSync", ctx, mock.AnythingOfType("domain.Attendance"), rec.SyncID).
		Return(nil, errors.New("FK violation: person_id")).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	assert.Equal(t, domain.SyncStatusError, resp.Results[0].Status)
	assert.Contains(t, resp.Results[0].Message, "FK violation")
}

// --- PullSince ----------------------------------------------------------------

func TestSyncService_Pull_MissingCampusForbidden(t *testing.T) {
	svc, _, _, _, _ := newSyncService(t)
	since := time.Now().Add(-time.Hour)

	_, err := svc.Pull(context.Background(), since, []string{domain.SyncEntityPerson}, 100)
	assert.True(t, errors.Is(err, domain.ErrForbidden))
}

func TestSyncService_Pull_NoRecords(t *testing.T) {
	svc, pRepo, _, _, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	since := time.Now().Add(-time.Hour)

	pRepo.On("ListUpdatedSince", ctx, campusID, since, 100).Return([]domain.Person{}, nil).Once()

	resp, err := svc.Pull(ctx, since, []string{domain.SyncEntityPerson}, 100)
	require.NoError(t, err)
	assert.Empty(t, resp.Records)
	assert.False(t, resp.HasMore)
	assert.False(t, resp.ServerTimestamp.IsZero())
}

func TestSyncService_Pull_RespectsEntityTypeFilter(t *testing.T) {
	svc, pRepo, tRepo, aRepo, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	since := time.Now().Add(-time.Hour)

	tRepo.On("ListUpdatedSince", ctx, campusID, since, 100).Return([]domain.Triage{}, nil).Once()

	_, err := svc.Pull(ctx, since, []string{domain.SyncEntityTriage}, 100)
	require.NoError(t, err)

	pRepo.AssertNotCalled(t, "ListUpdatedSince", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	aRepo.AssertNotCalled(t, "ListUpdatedSince", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestSyncService_Pull_AllEntityTypesQueriedAndSortedByUpdatedAt(t *testing.T) {
	svc, pRepo, tRepo, aRepo, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	since := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	tEarly := time.Date(2026, 5, 2, 8, 0, 0, 0, time.UTC)
	tMid := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	tLate := time.Date(2026, 5, 2, 16, 0, 0, 0, time.UTC)

	pRepo.On("ListUpdatedSince", ctx, campusID, since, 100).
		Return([]domain.Person{{ID: uuid.New(), CampusID: campusID, UpdatedAt: tLate}}, nil).Once()
	tRepo.On("ListUpdatedSince", ctx, campusID, since, 100).
		Return([]domain.Triage{{ID: uuid.New(), CampusID: campusID, UpdatedAt: tEarly}}, nil).Once()
	aRepo.On("ListUpdatedSince", ctx, campusID, since, 100).
		Return([]domain.Attendance{{ID: uuid.New(), CampusID: campusID, UpdatedAt: tMid}}, nil).Once()

	resp, err := svc.Pull(ctx, since,
		[]string{domain.SyncEntityPerson, domain.SyncEntityTriage, domain.SyncEntityAttendance}, 100)
	require.NoError(t, err)
	require.Len(t, resp.Records, 3)
	// records sorted ASC by updated_at: triage(8h) < attendance(12h) < person(16h)
	assert.Equal(t, domain.SyncEntityTriage, resp.Records[0].EntityType)
	assert.Equal(t, domain.SyncEntityAttendance, resp.Records[1].EntityType)
	assert.Equal(t, domain.SyncEntityPerson, resp.Records[2].EntityType)
	assert.False(t, resp.HasMore)
}

func TestSyncService_Pull_HasMoreWhenLimitReached(t *testing.T) {
	svc, pRepo, _, _, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	since := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	lastUpdated := time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)

	persons := make([]domain.Person, 100)
	for i := range persons {
		persons[i] = domain.Person{ID: uuid.New(), CampusID: campusID, UpdatedAt: lastUpdated}
	}
	pRepo.On("ListUpdatedSince", ctx, campusID, since, 100).Return(persons, nil).Once()

	resp, err := svc.Pull(ctx, since, []string{domain.SyncEntityPerson}, 100)
	require.NoError(t, err)
	assert.True(t, resp.HasMore)
	require.NotNil(t, resp.NextSince)
	assert.Equal(t, lastUpdated, *resp.NextSince)
}

// --- Cross-campus reference rejection (S04.9, security Finding 3) --------------

func TestSyncService_Push_TriageCrossCampusPersonRejected(t *testing.T) {
	svc, pRepo, tRepo, _, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	personID := uuid.New() // belongs to a different campus

	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityTriage,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":      personID.String(),
			"main_complaint": "Dor de cabeça",
		},
	}

	tRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	// The referenced person is not visible in the caller's campus.
	pRepo.On("FindByID", ctx, personID, campusID).Return(nil, domain.ErrNotFound).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, domain.SyncStatusError, resp.Results[0].Status)
	pRepo.AssertExpectations(t)
	// The record must never be persisted when the reference is cross-campus.
	tRepo.AssertNotCalled(t, "CreateWithSync", mock.Anything, mock.Anything, mock.Anything)
}

func TestSyncService_Push_AttendanceCrossCampusPersonRejected(t *testing.T) {
	svc, pRepo, _, aRepo, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	personID := uuid.New() // belongs to a different campus

	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityAttendance,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":       personID.String(),
			"service_type_id": uuid.New().String(),
			"professional_id": uuid.New().String(),
			"status":          domain.AttendanceStatusScheduled,
		},
	}

	aRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("FindByID", ctx, personID, campusID).Return(nil, domain.ErrNotFound).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, domain.SyncStatusError, resp.Results[0].Status)
	pRepo.AssertExpectations(t)
	aRepo.AssertNotCalled(t, "CreateWithSync", mock.Anything, mock.Anything, mock.Anything)
}

func TestSyncService_Push_AttendanceCrossCampusTriageRejected(t *testing.T) {
	svc, pRepo, tRepo, aRepo, _ := newSyncService(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	personID := uuid.New()
	triageID := uuid.New() // person is in-campus, but the triage reference is not

	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityAttendance,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":       personID.String(),
			"triage_id":       triageID.String(),
			"service_type_id": uuid.New().String(),
			"professional_id": uuid.New().String(),
			"status":          domain.AttendanceStatusScheduled,
		},
	}

	aRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("FindByID", ctx, personID, campusID).
		Return(&domain.Person{ID: personID, CampusID: campusID}, nil).Once()
	tRepo.On("FindByID", ctx, triageID, campusID).Return(nil, domain.ErrNotFound).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, domain.SyncStatusError, resp.Results[0].Status)
	pRepo.AssertExpectations(t)
	tRepo.AssertExpectations(t)
	aRepo.AssertNotCalled(t, "CreateWithSync", mock.Anything, mock.Anything, mock.Anything)
}

// --- Campaign link (S07.6) ------------------------------------------------------

func TestSyncService_Push_TriageWithCampaignPersisted(t *testing.T) {
	svc, pRepo, tRepo, _, cRepo, auditRepo := newSyncServiceWithCampaign(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	personID := uuid.New()
	campaignID := uuid.New()

	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityTriage,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":      personID.String(),
			"main_complaint": "Dor",
			"campaign_id":    campaignID.String(),
		},
	}
	tRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("FindByID", ctx, personID, campusID).
		Return(&domain.Person{ID: personID, CampusID: campusID}, nil).Once()
	cRepo.On("FindByID", ctx, campaignID, campusID).
		Return(&domain.Campaign{ID: campaignID, CampusID: campusID}, nil).Once()
	tRepo.On("CreateWithSync", ctx,
		mock.MatchedBy(func(tr domain.Triage) bool {
			return tr.CampaignID != nil && *tr.CampaignID == campaignID
		}),
		rec.SyncID,
	).Return(&domain.Triage{ID: uuid.New(), CampusID: campusID, CampaignID: &campaignID}, nil).Once()
	auditRepo.On("Create", ctx, mock.AnythingOfType("domain.AuditLog")).Return(nil).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	assert.Equal(t, domain.SyncStatusCreated, resp.Results[0].Status)
	tRepo.AssertExpectations(t)
	cRepo.AssertExpectations(t)
}

func TestSyncService_Push_TriageCampaignCrossCampusRejected(t *testing.T) {
	svc, pRepo, tRepo, _, cRepo, _ := newSyncServiceWithCampaign(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	personID := uuid.New()
	campaignID := uuid.New()

	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityTriage,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":      personID.String(),
			"main_complaint": "Dor",
			"campaign_id":    campaignID.String(),
		},
	}
	tRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("FindByID", ctx, personID, campusID).
		Return(&domain.Person{ID: personID, CampusID: campusID}, nil).Once()
	cRepo.On("FindByID", ctx, campaignID, campusID).Return(nil, domain.ErrNotFound).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, domain.SyncStatusError, resp.Results[0].Status)
	assert.Contains(t, resp.Results[0].Message, "campaign")
	tRepo.AssertNotCalled(t, "CreateWithSync", mock.Anything, mock.Anything, mock.Anything)
}

func TestSyncService_Push_TriageInvalidCampaignIDReportsError(t *testing.T) {
	svc, pRepo, tRepo, _, _, _ := newSyncServiceWithCampaign(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	personID := uuid.New()

	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityTriage,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":      personID.String(),
			"main_complaint": "Dor",
			"campaign_id":    "not-a-uuid",
		},
	}
	tRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("FindByID", ctx, personID, campusID).
		Return(&domain.Person{ID: personID, CampusID: campusID}, nil).Maybe()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	assert.Equal(t, domain.SyncStatusError, resp.Results[0].Status)
	tRepo.AssertNotCalled(t, "CreateWithSync", mock.Anything, mock.Anything, mock.Anything)
}

func TestSyncService_Push_AttendanceWithCampaignPersisted(t *testing.T) {
	svc, pRepo, _, aRepo, cRepo, auditRepo := newSyncServiceWithCampaign(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	personID := uuid.New()
	campaignID := uuid.New()

	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityAttendance,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":       personID.String(),
			"service_type_id": uuid.New().String(),
			"professional_id": uuid.New().String(),
			"status":          domain.AttendanceStatusScheduled,
			"campaign_id":     campaignID.String(),
		},
	}
	aRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("FindByID", ctx, personID, campusID).
		Return(&domain.Person{ID: personID, CampusID: campusID}, nil).Once()
	cRepo.On("FindByID", ctx, campaignID, campusID).
		Return(&domain.Campaign{ID: campaignID, CampusID: campusID}, nil).Once()
	aRepo.On("CreateWithSync", ctx,
		mock.MatchedBy(func(a domain.Attendance) bool {
			return a.CampaignID != nil && *a.CampaignID == campaignID
		}),
		rec.SyncID,
	).Return(&domain.Attendance{ID: uuid.New(), PersonID: personID, CampusID: campusID, CampaignID: &campaignID}, nil).Once()
	auditRepo.On("Create", ctx, mock.AnythingOfType("domain.AuditLog")).Return(nil).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	assert.Equal(t, domain.SyncStatusCreated, resp.Results[0].Status)
	aRepo.AssertExpectations(t)
	cRepo.AssertExpectations(t)
}

func TestSyncService_Push_AttendanceCampaignCrossCampusRejected(t *testing.T) {
	svc, pRepo, _, aRepo, cRepo, _ := newSyncServiceWithCampaign(t)
	campusID := uuid.New()
	ctx := ctxWithCampus(t, campusID)
	personID := uuid.New()
	campaignID := uuid.New()

	rec := domain.SyncPushRecord{
		EntityType: domain.SyncEntityAttendance,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"person_id":       personID.String(),
			"service_type_id": uuid.New().String(),
			"professional_id": uuid.New().String(),
			"status":          domain.AttendanceStatusScheduled,
			"campaign_id":     campaignID.String(),
		},
	}
	aRepo.On("FindBySyncID", ctx, rec.SyncID, campusID).Return(nil, domain.ErrNotFound).Once()
	pRepo.On("FindByID", ctx, personID, campusID).
		Return(&domain.Person{ID: personID, CampusID: campusID}, nil).Once()
	cRepo.On("FindByID", ctx, campaignID, campusID).Return(nil, domain.ErrNotFound).Once()

	resp, err := svc.Push(ctx, domain.SyncPushRequest{Records: []domain.SyncPushRecord{rec}})
	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, domain.SyncStatusError, resp.Results[0].Status)
	assert.Contains(t, resp.Results[0].Message, "campaign")
	aRepo.AssertNotCalled(t, "CreateWithSync", mock.Anything, mock.Anything, mock.Anything)
}

// --- Helpers ------------------------------------------------------------------

func validPersonRecord() domain.SyncPushRecord {
	return domain.SyncPushRecord{
		EntityType: domain.SyncEntityPerson,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"full_name":     "Maria",
			"document_type": "CPF",
			"nationality":   "BRA",
		},
	}
}
