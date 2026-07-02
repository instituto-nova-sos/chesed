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

type mockCampaignRepo struct{ mock.Mock }

func (m *mockCampaignRepo) Create(ctx context.Context, c domain.Campaign) (*domain.Campaign, error) {
	args := m.Called(ctx, c)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Campaign), args.Error(1)
}

func (m *mockCampaignRepo) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Campaign, error) {
	args := m.Called(ctx, id, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Campaign), args.Error(1)
}

func (m *mockCampaignRepo) List(ctx context.Context, filter domain.CampaignFilter) (*domain.CampaignListResult, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CampaignListResult), args.Error(1)
}

func (m *mockCampaignRepo) Update(ctx context.Context, c domain.Campaign) (*domain.Campaign, error) {
	args := m.Called(ctx, c)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Campaign), args.Error(1)
}

func (m *mockCampaignRepo) AddTeamMember(ctx context.Context, campaignID, personID uuid.UUID, role string, assignedBy *uuid.UUID) (*domain.CampaignTeamMember, error) {
	args := m.Called(ctx, campaignID, personID, role, assignedBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CampaignTeamMember), args.Error(1)
}

func (m *mockCampaignRepo) RemoveTeamMember(ctx context.Context, campaignID, personID uuid.UUID) error {
	args := m.Called(ctx, campaignID, personID)
	return args.Error(0)
}

func (m *mockCampaignRepo) ListTeam(ctx context.Context, campaignID uuid.UUID) ([]domain.CampaignTeamMember, error) {
	args := m.Called(ctx, campaignID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.CampaignTeamMember), args.Error(1)
}

type mockCampaignPersonRepo struct{ mock.Mock }

func (m *mockCampaignPersonRepo) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Person, error) {
	args := m.Called(ctx, id, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Person), args.Error(1)
}

func newCampaignTestHandler() (*CampaignHandler, *mockCampaignRepo, *mockCampaignPersonRepo, *mockAuditRepo) {
	repo := new(mockCampaignRepo)
	personRepo := new(mockCampaignPersonRepo)
	aRepo := new(mockAuditRepo)
	auditSvc := service.NewAuditService(aRepo)
	svc := service.NewCampaignService(repo, personRepo, auditSvc)
	return NewCampaignHandler(svc), repo, personRepo, aRepo
}

func campaignRouter(h *CampaignHandler) chi.Router {
	r := chi.NewRouter()
	r.Route("/campaigns", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/{id}", h.Get)
		r.Put("/{id}", h.Update)
		r.Post("/{id}/team", h.AddTeamMember)
		r.Delete("/{id}/team/{personId}", h.RemoveTeamMember)
	})
	return r
}

func campaignAuthedRequest(method, target string, body []byte) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, target, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	claims := auth.AuthClaims{
		Subject:  uuid.New().String(),
		Email:    "coordinator@chesed.test",
		Roles:    []string{"COORDINATOR"},
		CampusID: uuid.New(),
	}
	return req.WithContext(auth.NewContext(req.Context(), claims))
}

func TestCampaignHandler_Create(t *testing.T) {
	t.Run("201 on success", func(t *testing.T) {
		h, repo, _, aRepo := newCampaignTestHandler()
		repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Campaign")).
			Return(&domain.Campaign{ID: uuid.New(), Name: "March", Status: "PLANNED"}, nil)
		aRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		body, _ := json.Marshal(map[string]any{
			"name": "March", "campaign_type": "SOCIAL_ACTION", "start_date": "2026-07-10",
		})
		rec := httptest.NewRecorder()
		campaignRouter(h).ServeHTTP(rec, campaignAuthedRequest(http.MethodPost, "/campaigns", body))

		require.Equal(t, http.StatusCreated, rec.Code)
		var resp domain.Campaign
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, "PLANNED", resp.Status)
	})

	t.Run("400 on invalid body", func(t *testing.T) {
		h, _, _, _ := newCampaignTestHandler()
		rec := httptest.NewRecorder()
		campaignRouter(h).ServeHTTP(rec, campaignAuthedRequest(http.MethodPost, "/campaigns", []byte("{not json")))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("400 on missing required fields", func(t *testing.T) {
		h, _, _, _ := newCampaignTestHandler()
		body, _ := json.Marshal(map[string]any{"name": "March"})
		rec := httptest.NewRecorder()
		campaignRouter(h).ServeHTTP(rec, campaignAuthedRequest(http.MethodPost, "/campaigns", body))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "validation_error")
	})

	t.Run("400 validation_error on cross-campus coordinator reference", func(t *testing.T) {
		h, _, personRepo, _ := newCampaignTestHandler()
		personRepo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, domain.ErrNotFound)

		body, _ := json.Marshal(map[string]any{
			"name": "March", "campaign_type": "SOCIAL_ACTION", "start_date": "2026-07-10",
			"coordinator_id": uuid.New().String(),
		})
		rec := httptest.NewRecorder()
		campaignRouter(h).ServeHTTP(rec, campaignAuthedRequest(http.MethodPost, "/campaigns", body))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "validation_error")
	})

	t.Run("403 without campus context", func(t *testing.T) {
		h, _, _, _ := newCampaignTestHandler()
		body, _ := json.Marshal(map[string]any{
			"name": "March", "campaign_type": "SOCIAL_ACTION", "start_date": "2026-07-10",
		})
		req := httptest.NewRequest(http.MethodPost, "/campaigns", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		campaignRouter(h).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestCampaignHandler_Get(t *testing.T) {
	t.Run("200 with team on success", func(t *testing.T) {
		h, repo, _, _ := newCampaignTestHandler()
		id := uuid.New()
		repo.On("FindByID", mock.Anything, id, mock.Anything).
			Return(&domain.Campaign{ID: id, Name: "March"}, nil)
		repo.On("ListTeam", mock.Anything, id).
			Return([]domain.CampaignTeamMember{{PersonName: "Maria", RoleInCampaign: "VOLUNTEER"}}, nil)

		rec := httptest.NewRecorder()
		campaignRouter(h).ServeHTTP(rec, campaignAuthedRequest(http.MethodGet, "/campaigns/"+id.String(), nil))

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"team"`)
		assert.Contains(t, rec.Body.String(), "Maria")
	})

	t.Run("404 when not found", func(t *testing.T) {
		h, repo, _, _ := newCampaignTestHandler()
		repo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, domain.ErrNotFound)

		rec := httptest.NewRecorder()
		campaignRouter(h).ServeHTTP(rec, campaignAuthedRequest(http.MethodGet, "/campaigns/"+uuid.New().String(), nil))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("400 on malformed id", func(t *testing.T) {
		h, _, _, _ := newCampaignTestHandler()
		rec := httptest.NewRecorder()
		campaignRouter(h).ServeHTTP(rec, campaignAuthedRequest(http.MethodGet, "/campaigns/not-a-uuid", nil))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestCampaignHandler_List(t *testing.T) {
	t.Run("200 with data and pagination", func(t *testing.T) {
		h, repo, _, _ := newCampaignTestHandler()
		repo.On("List", mock.Anything, mock.AnythingOfType("domain.CampaignFilter")).
			Return(&domain.CampaignListResult{
				Data:       []domain.CampaignListItem{{ID: uuid.New(), Name: "March", Status: "ACTIVE"}},
				Pagination: domain.Pagination{Page: 1, PerPage: 20, Total: 1, TotalPages: 1},
			}, nil)

		rec := httptest.NewRecorder()
		campaignRouter(h).ServeHTTP(rec, campaignAuthedRequest(http.MethodGet, "/campaigns?status=ACTIVE", nil))

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"pagination"`)
	})

	t.Run("400 on invalid status filter", func(t *testing.T) {
		h, _, _, _ := newCampaignTestHandler()
		rec := httptest.NewRecorder()
		campaignRouter(h).ServeHTTP(rec, campaignAuthedRequest(http.MethodGet, "/campaigns?status=ARCHIVED", nil))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid_status")
	})
}

func TestCampaignHandler_Update(t *testing.T) {
	validBody := func() []byte {
		b, _ := json.Marshal(map[string]any{
			"name": "Updated", "campaign_type": "HEALTH", "start_date": "2026-07-10", "status": "ACTIVE",
		})
		return b
	}

	t.Run("200 on success", func(t *testing.T) {
		h, repo, _, aRepo := newCampaignTestHandler()
		id := uuid.New()
		repo.On("FindByID", mock.Anything, id, mock.Anything).
			Return(&domain.Campaign{ID: id, Name: "Old", Status: "PLANNED"}, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("domain.Campaign")).
			Return(&domain.Campaign{ID: id, Name: "Updated", Status: "ACTIVE"}, nil)
		aRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		rec := httptest.NewRecorder()
		campaignRouter(h).ServeHTTP(rec, campaignAuthedRequest(http.MethodPut, "/campaigns/"+id.String(), validBody()))
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "ACTIVE")
	})

	t.Run("404 when not found", func(t *testing.T) {
		h, repo, _, _ := newCampaignTestHandler()
		repo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, domain.ErrNotFound)

		rec := httptest.NewRecorder()
		campaignRouter(h).ServeHTTP(rec, campaignAuthedRequest(http.MethodPut, "/campaigns/"+uuid.New().String(), validBody()))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("400 on invalid status enum", func(t *testing.T) {
		h, _, _, _ := newCampaignTestHandler()
		body, _ := json.Marshal(map[string]any{
			"name": "Updated", "campaign_type": "HEALTH", "start_date": "2026-07-10", "status": "ARCHIVED",
		})
		rec := httptest.NewRecorder()
		campaignRouter(h).ServeHTTP(rec, campaignAuthedRequest(http.MethodPut, "/campaigns/"+uuid.New().String(), body))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestCampaignHandler_AddTeamMember(t *testing.T) {
	t.Run("201 on success", func(t *testing.T) {
		h, repo, personRepo, aRepo := newCampaignTestHandler()
		campaignID := uuid.New()
		personID := uuid.New()
		repo.On("FindByID", mock.Anything, campaignID, mock.Anything).
			Return(&domain.Campaign{ID: campaignID}, nil)
		personRepo.On("FindByID", mock.Anything, personID, mock.Anything).
			Return(&domain.Person{ID: personID, FullName: "Maria Silva"}, nil)
		repo.On("AddTeamMember", mock.Anything, campaignID, personID, "VOLUNTEER", mock.Anything).
			Return(&domain.CampaignTeamMember{PersonID: personID, PersonName: "Maria Silva", RoleInCampaign: "VOLUNTEER"}, nil)
		aRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		body, _ := json.Marshal(map[string]any{"person_id": personID.String(), "role_in_campaign": "VOLUNTEER"})
		rec := httptest.NewRecorder()
		campaignRouter(h).ServeHTTP(rec, campaignAuthedRequest(http.MethodPost, "/campaigns/"+campaignID.String()+"/team", body))

		require.Equal(t, http.StatusCreated, rec.Code)
		assert.Contains(t, rec.Body.String(), "Maria Silva")
	})

	t.Run("409 duplicate on repeated assignment", func(t *testing.T) {
		h, repo, personRepo, _ := newCampaignTestHandler()
		campaignID := uuid.New()
		personID := uuid.New()
		repo.On("FindByID", mock.Anything, campaignID, mock.Anything).
			Return(&domain.Campaign{ID: campaignID}, nil)
		personRepo.On("FindByID", mock.Anything, personID, mock.Anything).
			Return(&domain.Person{ID: personID}, nil)
		repo.On("AddTeamMember", mock.Anything, campaignID, personID, "VOLUNTEER", mock.Anything).
			Return(nil, domain.ErrDuplicate)

		body, _ := json.Marshal(map[string]any{"person_id": personID.String(), "role_in_campaign": "VOLUNTEER"})
		rec := httptest.NewRecorder()
		campaignRouter(h).ServeHTTP(rec, campaignAuthedRequest(http.MethodPost, "/campaigns/"+campaignID.String()+"/team", body))

		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), "duplicate")
	})

	t.Run("400 validation_error on cross-campus person", func(t *testing.T) {
		h, repo, personRepo, _ := newCampaignTestHandler()
		campaignID := uuid.New()
		repo.On("FindByID", mock.Anything, campaignID, mock.Anything).
			Return(&domain.Campaign{ID: campaignID}, nil)
		personRepo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, domain.ErrNotFound)

		body, _ := json.Marshal(map[string]any{"person_id": uuid.New().String(), "role_in_campaign": "VOLUNTEER"})
		rec := httptest.NewRecorder()
		campaignRouter(h).ServeHTTP(rec, campaignAuthedRequest(http.MethodPost, "/campaigns/"+campaignID.String()+"/team", body))

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "validation_error")
	})

	t.Run("404 when campaign missing", func(t *testing.T) {
		h, repo, _, _ := newCampaignTestHandler()
		repo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, domain.ErrNotFound)

		body, _ := json.Marshal(map[string]any{"person_id": uuid.New().String(), "role_in_campaign": "VOLUNTEER"})
		rec := httptest.NewRecorder()
		campaignRouter(h).ServeHTTP(rec, campaignAuthedRequest(http.MethodPost, "/campaigns/"+uuid.New().String()+"/team", body))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestCampaignHandler_RemoveTeamMember(t *testing.T) {
	t.Run("204 on success", func(t *testing.T) {
		h, repo, _, aRepo := newCampaignTestHandler()
		campaignID := uuid.New()
		personID := uuid.New()
		repo.On("FindByID", mock.Anything, campaignID, mock.Anything).
			Return(&domain.Campaign{ID: campaignID}, nil)
		repo.On("RemoveTeamMember", mock.Anything, campaignID, personID).Return(nil)
		aRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		rec := httptest.NewRecorder()
		campaignRouter(h).ServeHTTP(rec, campaignAuthedRequest(http.MethodDelete,
			"/campaigns/"+campaignID.String()+"/team/"+personID.String(), nil))
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("404 when assignment missing", func(t *testing.T) {
		h, repo, _, _ := newCampaignTestHandler()
		campaignID := uuid.New()
		repo.On("FindByID", mock.Anything, campaignID, mock.Anything).
			Return(&domain.Campaign{ID: campaignID}, nil)
		repo.On("RemoveTeamMember", mock.Anything, campaignID, mock.Anything).
			Return(domain.ErrNotFound)

		rec := httptest.NewRecorder()
		campaignRouter(h).ServeHTTP(rec, campaignAuthedRequest(http.MethodDelete,
			"/campaigns/"+campaignID.String()+"/team/"+uuid.New().String(), nil))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
