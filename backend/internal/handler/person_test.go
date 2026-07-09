package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock repositories for person handler tests.
type mockPersonRepo struct{ mock.Mock }

func (m *mockPersonRepo) Create(ctx context.Context, p domain.Person, a *domain.Address) (*domain.Person, error) {
	args := m.Called(ctx, p, a)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Person), args.Error(1)
}
func (m *mockPersonRepo) FindByEmail(ctx context.Context, email string, campusID uuid.UUID) (*domain.Person, error) {
	args := m.Called(ctx, email, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Person), args.Error(1)
}
func (m *mockPersonRepo) FindByEmailGlobal(ctx context.Context, email string) ([]domain.Person, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Person), args.Error(1)
}
func (m *mockPersonRepo) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Person, error) {
	args := m.Called(ctx, id, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Person), args.Error(1)
}
func (m *mockPersonRepo) FindByIDWithDetails(ctx context.Context, id, campusID uuid.UUID) (*domain.Person, []domain.Address, []domain.PersonRole, error) {
	args := m.Called(ctx, id, campusID)
	if args.Get(0) == nil {
		return nil, nil, nil, args.Error(3)
	}
	return args.Get(0).(*domain.Person), args.Get(1).([]domain.Address), args.Get(2).([]domain.PersonRole), args.Error(3)
}
func (m *mockPersonRepo) Update(ctx context.Context, p domain.Person) (*domain.Person, error) {
	args := m.Called(ctx, p)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Person), args.Error(1)
}
func (m *mockPersonRepo) List(ctx context.Context, f domain.PersonFilter) (*domain.PersonListResult, error) {
	args := m.Called(ctx, f)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PersonListResult), args.Error(1)
}
func (m *mockPersonRepo) CheckDuplicate(ctx context.Context, dt, dn string, campusID uuid.UUID) (*domain.DuplicateCheckResult, error) {
	args := m.Called(ctx, dt, dn, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.DuplicateCheckResult), args.Error(1)
}
func (m *mockPersonRepo) UpdateAddress(ctx context.Context, personID uuid.UUID, a domain.Address) (*domain.Address, error) {
	args := m.Called(ctx, personID, a)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Address), args.Error(1)
}

type mockPersonRoleRepo struct{ mock.Mock }

func (m *mockPersonRoleRepo) Create(ctx context.Context, r domain.PersonRole) (*domain.PersonRole, error) {
	args := m.Called(ctx, r)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PersonRole), args.Error(1)
}
func (m *mockPersonRoleRepo) FindByPersonID(ctx context.Context, personID uuid.UUID) ([]domain.PersonRole, error) {
	args := m.Called(ctx, personID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.PersonRole), args.Error(1)
}
func (m *mockPersonRoleRepo) ToggleActive(ctx context.Context, roleID uuid.UUID, isActive bool, userID *uuid.UUID) (*domain.PersonRole, error) {
	args := m.Called(ctx, roleID, isActive, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PersonRole), args.Error(1)
}

type mockAuditRepo struct{ mock.Mock }

func (m *mockAuditRepo) Create(ctx context.Context, entry domain.AuditLog) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

type mockAgreementRepo struct{ mock.Mock }

func (m *mockAgreementRepo) Create(ctx context.Context, a domain.VolunteerAgreement) (*domain.VolunteerAgreement, error) {
	args := m.Called(ctx, a)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VolunteerAgreement), args.Error(1)
}
func (m *mockAgreementRepo) FindByPersonID(ctx context.Context, personID uuid.UUID) ([]domain.VolunteerAgreement, error) {
	args := m.Called(ctx, personID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.VolunteerAgreement), args.Error(1)
}
func (m *mockAgreementRepo) FindByPersonRoleID(ctx context.Context, personRoleID uuid.UUID) (*domain.VolunteerAgreement, error) {
	args := m.Called(ctx, personRoleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VolunteerAgreement), args.Error(1)
}
func (m *mockAgreementRepo) FindPendingByPersonID(ctx context.Context, personID uuid.UUID) (*domain.VolunteerAgreement, error) {
	args := m.Called(ctx, personID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VolunteerAgreement), args.Error(1)
}
func (m *mockAgreementRepo) HasAcceptedAgreement(ctx context.Context, personID uuid.UUID) (bool, error) {
	args := m.Called(ctx, personID)
	return args.Bool(0), args.Error(1)
}
func (m *mockAgreementRepo) AcceptDigital(ctx context.Context, id uuid.UUID, userID uuid.UUID, ip string, userAgent string, signatureData string) (*domain.VolunteerAgreement, error) {
	args := m.Called(ctx, id, userID, ip, userAgent, signatureData)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VolunteerAgreement), args.Error(1)
}
func (m *mockAgreementRepo) Reject(ctx context.Context, id uuid.UUID, reason *string) (*domain.VolunteerAgreement, error) {
	args := m.Called(ctx, id, reason)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VolunteerAgreement), args.Error(1)
}
func (m *mockAgreementRepo) AcceptManualUpload(ctx context.Context, id uuid.UUID, documentPath string, uploadedBy uuid.UUID) (*domain.VolunteerAgreement, error) {
	args := m.Called(ctx, id, documentPath, uploadedBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VolunteerAgreement), args.Error(1)
}

func newPersonTestHandler() (*PersonHandler, *mockPersonRepo, *mockPersonRoleRepo, *mockAuditRepo) {
	pRepo := new(mockPersonRepo)
	rRepo := new(mockPersonRoleRepo)
	agRepo := new(mockAgreementRepo)
	aRepo := new(mockAuditRepo)
	auditSvc := service.NewAuditService(aRepo)
	svc := service.NewPersonService(pRepo, rRepo, agRepo, auditSvc)

	// Default: agreement creation succeeds
	agRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.VolunteerAgreement")).
		Return(&domain.VolunteerAgreement{ID: uuid.New()}, nil).Maybe()

	h := NewPersonHandler(svc)
	return h, pRepo, rRepo, aRepo
}

func withAuthContext(r *http.Request) *http.Request {
	claims := auth.AuthClaims{
		Subject:  uuid.New().String(),
		Email:    "test@chesed.test",
		Roles:    []string{"COORDINATOR"},
		CampusID: uuid.New(),
	}
	ctx := auth.NewContext(r.Context(), claims)
	return r.WithContext(ctx)
}

func withChiParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestPersonHandler_Create(t *testing.T) {
	t.Run("201 success", func(t *testing.T) {
		h, pRepo, _, aRepo := newPersonTestHandler()

		body, _ := json.Marshal(service.CreatePersonInput{FullName: "Maria", DocumentType: "CPF"})
		req := withAuthContext(httptest.NewRequest(http.MethodPost, "/api/v1/persons", bytes.NewReader(body)))

		pRepo.On("Create", mock.Anything, mock.Anything, mock.Anything).
			Return(&domain.Person{ID: uuid.New(), FullName: "Maria"}, nil)
		aRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

		rec := httptest.NewRecorder()
		h.Create(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var result domain.Person
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
		assert.Equal(t, "Maria", result.FullName)
	})

	t.Run("400 bad JSON", func(t *testing.T) {
		h, _, _, _ := newPersonTestHandler()

		req := withAuthContext(httptest.NewRequest(http.MethodPost, "/api/v1/persons", bytes.NewReader([]byte("not json"))))

		rec := httptest.NewRecorder()
		h.Create(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("400 validation error", func(t *testing.T) {
		h, _, _, _ := newPersonTestHandler()

		body, _ := json.Marshal(map[string]string{"document_type": "CPF"})
		req := withAuthContext(httptest.NewRequest(http.MethodPost, "/api/v1/persons", bytes.NewReader(body)))

		rec := httptest.NewRecorder()
		h.Create(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestPersonHandler_List(t *testing.T) {
	t.Run("200 with data", func(t *testing.T) {
		h, pRepo, _, _ := newPersonTestHandler()

		expected := &domain.PersonListResult{
			Data:       []domain.PersonListItem{{ID: uuid.New(), FullName: "Maria", Roles: []string{"VOLUNTEER"}}},
			Pagination: domain.Pagination{Page: 1, PerPage: 20, Total: 1, TotalPages: 1},
		}
		pRepo.On("List", mock.Anything, mock.AnythingOfType("domain.PersonFilter")).Return(expected, nil)

		req := withAuthContext(httptest.NewRequest(http.MethodGet, "/api/v1/persons?q=Maria", nil))

		rec := httptest.NewRecorder()
		h.List(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var result domain.PersonListResult
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
		assert.Len(t, result.Data, 1)
	})
}

func TestPersonHandler_Get(t *testing.T) {
	t.Run("200 success", func(t *testing.T) {
		h, pRepo, _, _ := newPersonTestHandler()

		personID := uuid.New()
		person := &domain.Person{ID: personID, FullName: "Maria"}
		pRepo.On("FindByIDWithDetails", mock.Anything, personID, mock.Anything).
			Return(person, []domain.Address{}, []domain.PersonRole{}, nil)

		req := withAuthContext(httptest.NewRequest(http.MethodGet, "/api/v1/persons/"+personID.String(), nil))
		req = withChiParam(req, "id", personID.String())

		rec := httptest.NewRecorder()
		h.Get(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("400 bad UUID", func(t *testing.T) {
		h, _, _, _ := newPersonTestHandler()

		req := withAuthContext(httptest.NewRequest(http.MethodGet, "/api/v1/persons/not-a-uuid", nil))
		req = withChiParam(req, "id", "not-a-uuid")

		rec := httptest.NewRecorder()
		h.Get(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("404 not found", func(t *testing.T) {
		h, pRepo, _, _ := newPersonTestHandler()

		personID := uuid.New()
		pRepo.On("FindByIDWithDetails", mock.Anything, personID, mock.Anything).
			Return(nil, nil, nil, domain.ErrNotFound)

		req := withAuthContext(httptest.NewRequest(http.MethodGet, "/api/v1/persons/"+personID.String(), nil))
		req = withChiParam(req, "id", personID.String())

		rec := httptest.NewRecorder()
		h.Get(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestPersonHandler_CheckDuplicate(t *testing.T) {
	t.Run("200 with matches", func(t *testing.T) {
		h, pRepo, _, _ := newPersonTestHandler()

		expected := &domain.DuplicateCheckResult{
			HasDuplicates: true,
			Matches:       []domain.DuplicateMatch{{ID: uuid.New(), FullName: "Existing", MatchType: "exact_document"}},
		}
		pRepo.On("CheckDuplicate", mock.Anything, "CPF", "123", mock.Anything).Return(expected, nil)

		req := withAuthContext(httptest.NewRequest(http.MethodGet, "/api/v1/persons/check-duplicate?document_type=CPF&document_number=123", nil))

		rec := httptest.NewRecorder()
		h.CheckDuplicate(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var result domain.DuplicateCheckResult
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
		assert.True(t, result.HasDuplicates)
	})

	t.Run("400 missing params", func(t *testing.T) {
		h, _, _, _ := newPersonTestHandler()

		req := withAuthContext(httptest.NewRequest(http.MethodGet, "/api/v1/persons/check-duplicate", nil))

		rec := httptest.NewRecorder()
		h.CheckDuplicate(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestPersonHandler_AddRole(t *testing.T) {
	t.Run("201 success", func(t *testing.T) {
		h, pRepo, rRepo, aRepo := newPersonTestHandler()

		personID := uuid.New()
		body, _ := json.Marshal(service.AddRoleInput{RoleType: "VOLUNTEER"})

		pRepo.On("FindByID", mock.Anything, personID, mock.Anything).
			Return(&domain.Person{ID: personID}, nil)
		rRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.PersonRole")).
			Return(&domain.PersonRole{ID: uuid.New(), RoleType: "VOLUNTEER", IsActive: true}, nil)
		aRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

		req := withAuthContext(httptest.NewRequest(http.MethodPost, "/api/v1/persons/"+personID.String()+"/roles", bytes.NewReader(body)))
		req = withChiParam(req, "id", personID.String())

		rec := httptest.NewRecorder()
		h.AddRole(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("409 duplicate role", func(t *testing.T) {
		h, pRepo, rRepo, _ := newPersonTestHandler()

		personID := uuid.New()
		body, _ := json.Marshal(service.AddRoleInput{RoleType: "VOLUNTEER"})

		pRepo.On("FindByID", mock.Anything, personID, mock.Anything).
			Return(&domain.Person{ID: personID}, nil)
		rRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.PersonRole")).
			Return(nil, domain.ErrDuplicate)

		req := withAuthContext(httptest.NewRequest(http.MethodPost, "/api/v1/persons/"+personID.String()+"/roles", bytes.NewReader(body)))
		req = withChiParam(req, "id", personID.String())

		rec := httptest.NewRecorder()
		h.AddRole(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})
}

func TestPersonHandler_GetHistory(t *testing.T) {
	t.Run("200 empty history", func(t *testing.T) {
		h, pRepo, _, _ := newPersonTestHandler()

		personID := uuid.New()
		pRepo.On("FindByID", mock.Anything, personID, mock.Anything).
			Return(&domain.Person{ID: personID}, nil)

		req := withAuthContext(httptest.NewRequest(http.MethodGet, "/api/v1/persons/"+personID.String()+"/history", nil))
		req = withChiParam(req, "id", personID.String())

		rec := httptest.NewRecorder()
		h.GetHistory(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
