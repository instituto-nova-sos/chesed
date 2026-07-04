package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockConsentRepo struct{ mock.Mock }

func (m *mockConsentRepo) Create(ctx context.Context, c domain.Consent) (*domain.Consent, error) {
	args := m.Called(ctx, c)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Consent), args.Error(1)
}

func (m *mockConsentRepo) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Consent, error) {
	args := m.Called(ctx, id, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Consent), args.Error(1)
}

func (m *mockConsentRepo) ListByPerson(ctx context.Context, personID, campusID uuid.UUID) ([]domain.Consent, error) {
	args := m.Called(ctx, personID, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Consent), args.Error(1)
}

func (m *mockConsentRepo) Revoke(ctx context.Context, id, campusID uuid.UUID, reason string) (*domain.Consent, error) {
	args := m.Called(ctx, id, campusID, reason)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Consent), args.Error(1)
}

type mockConsentPersonRepo struct{ mock.Mock }

func (m *mockConsentPersonRepo) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Person, error) {
	args := m.Called(ctx, id, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Person), args.Error(1)
}

func newConsentTestHandler() (*ConsentHandler, *mockConsentRepo, *mockConsentPersonRepo, *mockAuditRepo) {
	repo := new(mockConsentRepo)
	personRepo := new(mockConsentPersonRepo)
	aRepo := new(mockAuditRepo)
	auditSvc := service.NewAuditService(aRepo)
	svc := service.NewConsentService(repo, personRepo, auditSvc)
	return NewConsentHandler(svc), repo, personRepo, aRepo
}

func consentRouter(h *ConsentHandler) chi.Router {
	r := chi.NewRouter()
	r.Post("/consents", h.Create)
	r.Patch("/consents/{id}/revoke", h.Revoke)
	r.Get("/persons/{id}/consents", h.ListByPerson)
	return r
}

func consentAuthedRequest(method, target string, body []byte) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, target, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	claims := auth.AuthClaims{
		Subject:  uuid.New().String(),
		Email:    "volunteer@chesed.test",
		Roles:    []string{"VOLUNTEER"},
		CampusID: uuid.New(),
	}
	return req.WithContext(auth.NewContext(req.Context(), claims))
}

func validConsentBody(personID uuid.UUID) []byte {
	b, _ := json.Marshal(map[string]any{
		"person_id":    personID.String(),
		"consent_type": "DATA_PROCESSING",
		"purpose":      "Registration and attendance follow-up",
	})
	return b
}

func TestConsentHandler_Create(t *testing.T) {
	t.Run("201 on success with documented shape", func(t *testing.T) {
		h, repo, personRepo, aRepo := newConsentTestHandler()
		personID := uuid.New()
		personRepo.On("FindByID", mock.Anything, personID, mock.Anything).
			Return(&domain.Person{ID: personID}, nil)
		repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Consent")).
			Return(&domain.Consent{
				ID: uuid.New(), PersonID: personID, ConsentType: "DATA_PROCESSING",
				ConsentVersion: "1.0", Purpose: "Registration and attendance follow-up",
				GrantedAt: time.Now(), IsActive: true,
			}, nil)
		aRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		rec := httptest.NewRecorder()
		consentRouter(h).ServeHTTP(rec, consentAuthedRequest(http.MethodPost, "/consents", validConsentBody(personID)))

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var resp map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, "DATA_PROCESSING", resp["consent_type"])
		assert.Equal(t, "1.0", resp["consent_version"])
		assert.Equal(t, true, resp["is_active"])
		assert.Contains(t, resp, "granted_by_person_id")
		assert.Contains(t, resp, "revoked_at")
		assert.Contains(t, resp, "revoked_reason")
	})

	t.Run("400 on invalid JSON body", func(t *testing.T) {
		h, _, _, _ := newConsentTestHandler()
		rec := httptest.NewRecorder()
		consentRouter(h).ServeHTTP(rec, consentAuthedRequest(http.MethodPost, "/consents", []byte("{not json")))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("400 validation_error on missing purpose", func(t *testing.T) {
		h, _, _, _ := newConsentTestHandler()
		body, _ := json.Marshal(map[string]any{
			"person_id": uuid.New().String(), "consent_type": "DATA_PROCESSING",
		})
		rec := httptest.NewRecorder()
		consentRouter(h).ServeHTTP(rec, consentAuthedRequest(http.MethodPost, "/consents", body))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "validation_error")
	})

	t.Run("400 validation_error on unknown consent_type", func(t *testing.T) {
		h, _, _, _ := newConsentTestHandler()
		body, _ := json.Marshal(map[string]any{
			"person_id": uuid.New().String(), "consent_type": "MARKETING", "purpose": "x",
		})
		rec := httptest.NewRecorder()
		consentRouter(h).ServeHTTP(rec, consentAuthedRequest(http.MethodPost, "/consents", body))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "validation_error")
	})

	t.Run("400 validation_error on cross-campus person reference", func(t *testing.T) {
		h, _, personRepo, _ := newConsentTestHandler()
		personRepo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, domain.ErrNotFound)

		rec := httptest.NewRecorder()
		consentRouter(h).ServeHTTP(rec, consentAuthedRequest(http.MethodPost, "/consents", validConsentBody(uuid.New())))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "validation_error")
	})

	t.Run("409 duplicate on second active consent of same type", func(t *testing.T) {
		h, repo, personRepo, _ := newConsentTestHandler()
		personID := uuid.New()
		personRepo.On("FindByID", mock.Anything, personID, mock.Anything).
			Return(&domain.Person{ID: personID}, nil)
		repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Consent")).
			Return(nil, domain.ErrDuplicate)

		rec := httptest.NewRecorder()
		consentRouter(h).ServeHTTP(rec, consentAuthedRequest(http.MethodPost, "/consents", validConsentBody(personID)))
		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), "duplicate")
	})

	t.Run("403 without campus context", func(t *testing.T) {
		h, _, _, _ := newConsentTestHandler()
		req := httptest.NewRequest(http.MethodPost, "/consents", bytes.NewReader(validConsentBody(uuid.New())))
		rec := httptest.NewRecorder()
		consentRouter(h).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestConsentHandler_ListByPerson(t *testing.T) {
	t.Run("200 with data envelope ordered as repository returns", func(t *testing.T) {
		h, repo, personRepo, _ := newConsentTestHandler()
		personID := uuid.New()
		personRepo.On("FindByID", mock.Anything, personID, mock.Anything).
			Return(&domain.Person{ID: personID}, nil)
		repo.On("ListByPerson", mock.Anything, personID, mock.Anything).
			Return([]domain.Consent{
				{ID: uuid.New(), ConsentType: "IMAGE_USAGE", IsActive: false},
				{ID: uuid.New(), ConsentType: "DATA_PROCESSING", IsActive: true},
			}, nil)

		rec := httptest.NewRecorder()
		consentRouter(h).ServeHTTP(rec, consentAuthedRequest(http.MethodGet, "/persons/"+personID.String()+"/consents", nil))

		require.Equal(t, http.StatusOK, rec.Code)
		var resp struct {
			Data []domain.Consent `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Len(t, resp.Data, 2)
		assert.Equal(t, "IMAGE_USAGE", resp.Data[0].ConsentType)
	})

	t.Run("404 when person is not visible in campus", func(t *testing.T) {
		h, _, personRepo, _ := newConsentTestHandler()
		personRepo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, domain.ErrNotFound)

		rec := httptest.NewRecorder()
		consentRouter(h).ServeHTTP(rec, consentAuthedRequest(http.MethodGet, "/persons/"+uuid.New().String()+"/consents", nil))
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "not_found")
	})

	t.Run("400 on malformed person id", func(t *testing.T) {
		h, _, _, _ := newConsentTestHandler()
		rec := httptest.NewRecorder()
		consentRouter(h).ServeHTTP(rec, consentAuthedRequest(http.MethodGet, "/persons/not-a-uuid/consents", nil))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestConsentHandler_Revoke(t *testing.T) {
	revokeBody := func() []byte {
		b, _ := json.Marshal(map[string]any{"revoked_reason": "Data subject request"})
		return b
	}

	t.Run("200 on success with revocation fields set", func(t *testing.T) {
		h, repo, _, aRepo := newConsentTestHandler()
		id := uuid.New()
		now := time.Now()
		reason := "Data subject request"
		repo.On("FindByID", mock.Anything, id, mock.Anything).
			Return(&domain.Consent{ID: id, IsActive: true}, nil)
		repo.On("Revoke", mock.Anything, id, mock.Anything, reason).
			Return(&domain.Consent{ID: id, IsActive: false, RevokedAt: &now, RevokedReason: &reason}, nil)
		aRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		rec := httptest.NewRecorder()
		consentRouter(h).ServeHTTP(rec, consentAuthedRequest(http.MethodPatch, "/consents/"+id.String()+"/revoke", revokeBody()))

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var resp domain.Consent
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.False(t, resp.IsActive)
		require.NotNil(t, resp.RevokedAt)
	})

	t.Run("400 validation_error on missing reason", func(t *testing.T) {
		h, _, _, _ := newConsentTestHandler()
		body, _ := json.Marshal(map[string]any{})
		rec := httptest.NewRecorder()
		consentRouter(h).ServeHTTP(rec, consentAuthedRequest(http.MethodPatch, "/consents/"+uuid.New().String()+"/revoke", body))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "validation_error")
	})

	t.Run("400 validation_error on already revoked consent", func(t *testing.T) {
		h, repo, _, _ := newConsentTestHandler()
		id := uuid.New()
		repo.On("FindByID", mock.Anything, id, mock.Anything).
			Return(&domain.Consent{ID: id, IsActive: false}, nil)

		rec := httptest.NewRecorder()
		consentRouter(h).ServeHTTP(rec, consentAuthedRequest(http.MethodPatch, "/consents/"+id.String()+"/revoke", revokeBody()))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "validation_error")
	})

	t.Run("404 when consent is outside campus or missing", func(t *testing.T) {
		h, repo, _, _ := newConsentTestHandler()
		repo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, domain.ErrNotFound)

		rec := httptest.NewRecorder()
		consentRouter(h).ServeHTTP(rec, consentAuthedRequest(http.MethodPatch, "/consents/"+uuid.New().String()+"/revoke", revokeBody()))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("400 on malformed consent id", func(t *testing.T) {
		h, _, _, _ := newConsentTestHandler()
		rec := httptest.NewRecorder()
		consentRouter(h).ServeHTTP(rec, consentAuthedRequest(http.MethodPatch, "/consents/not-a-uuid/revoke", revokeBody()))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
