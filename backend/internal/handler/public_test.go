package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/middleware"
	"github.com/instituto-nova-sos/chesed/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockPublicCampaignRepo struct{ mock.Mock }

func (m *mockPublicCampaignRepo) List(ctx context.Context, filter domain.CampaignFilter) (*domain.CampaignListResult, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CampaignListResult), args.Error(1)
}

type mockPublicPersonRepo struct{ mock.Mock }

func (m *mockPublicPersonRepo) Create(ctx context.Context, person domain.Person, address *domain.Address) (*domain.Person, error) {
	args := m.Called(ctx, person, address)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Person), args.Error(1)
}

type mockPublicRoleRepo struct{ mock.Mock }

func (m *mockPublicRoleRepo) Create(ctx context.Context, role domain.PersonRole) (*domain.PersonRole, error) {
	args := m.Called(ctx, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PersonRole), args.Error(1)
}

type mockPublicAgreementRepo struct{ mock.Mock }

func (m *mockPublicAgreementRepo) Create(ctx context.Context, agreement domain.VolunteerAgreement) (*domain.VolunteerAgreement, error) {
	args := m.Called(ctx, agreement)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VolunteerAgreement), args.Error(1)
}

func newPublicTestHandler() (
	*PublicHandler,
	*mockPublicCampaignRepo,
	*mockPublicPersonRepo,
	*mockPublicRoleRepo,
	*mockPublicAgreementRepo,
	*mockAuditRepo,
) {
	campaignRepo := new(mockPublicCampaignRepo)
	personRepo := new(mockPublicPersonRepo)
	roleRepo := new(mockPublicRoleRepo)
	agreementRepo := new(mockPublicAgreementRepo)
	auditRepo := new(mockAuditRepo)
	svc := service.NewPublicService(campaignRepo, personRepo, roleRepo, agreementRepo, service.NewAuditService(auditRepo))
	return NewPublicHandler(svc), campaignRepo, personRepo, roleRepo, agreementRepo, auditRepo
}

// publicRequest builds a request with the campus already in context, as the
// PublicCampusValidator middleware would have set it in production.
func publicRequest(method, target string, body []byte, campusID uuid.UUID) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, target, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	return req.WithContext(middleware.WithPublicCampus(req.Context(), campusID))
}

func TestPublicHandler_ListCampaigns(t *testing.T) {
	h, campaignRepo, _, _, _, _ := newPublicTestHandler()
	campusID := uuid.New()
	expected := &domain.CampaignListResult{
		Data: []domain.CampaignListItem{{ID: uuid.New(), Name: "Food Drive", Status: "ACTIVE"}},
	}
	campaignRepo.On("List", mock.Anything, mock.MatchedBy(func(f domain.CampaignFilter) bool {
		return f.CampusID == campusID && f.Status != nil && *f.Status == "ACTIVE" && f.Page == 3 && f.PerPage == 5
	})).Return(expected, nil)

	req := publicRequest(http.MethodGet, "/api/v1/public/campaigns?campus_id="+campusID.String()+"&page=3&per_page=5", nil, campusID)
	rec := httptest.NewRecorder()
	h.ListCampaigns(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body domain.CampaignListResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Len(t, body.Data, 1)
	assert.Equal(t, "Food Drive", body.Data[0].Name)
	campaignRepo.AssertExpectations(t)
}

func TestPublicHandler_VolunteerSignup_Created(t *testing.T) {
	h, _, personRepo, roleRepo, agreementRepo, auditRepo := newPublicTestHandler()
	campusID := uuid.New()
	created := &domain.Person{ID: uuid.New(), FullName: "Grace Hopper", CampusID: campusID, IsActive: true}

	personRepo.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(created, nil)
	roleRepo.On("Create", mock.Anything, mock.Anything).
		Return(&domain.PersonRole{ID: uuid.New(), RoleType: domain.RoleVolunteer, IsActive: true}, nil)
	agreementRepo.On("Create", mock.Anything, mock.Anything).
		Return(&domain.VolunteerAgreement{ID: uuid.New(), Status: domain.AgreementPending}, nil)
	auditRepo.On("Create", mock.Anything, mock.MatchedBy(func(e domain.AuditLog) bool {
		return e.IPAddress != nil && *e.IPAddress == "198.51.100.9"
	})).Return(nil)

	payload := []byte(`{"full_name":"Grace Hopper","campus_id":"` + campusID.String() + `"}`)
	req := publicRequest(http.MethodPost, "/api/v1/public/volunteers", payload, campusID)
	req.Header.Set("X-Forwarded-For", "198.51.100.9, 10.0.0.1")
	req.Header.Set("User-Agent", "wp-form")
	rec := httptest.NewRecorder()
	h.VolunteerSignup(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, created.ID.String(), body["id"])
	assert.Equal(t, "Grace Hopper", body["full_name"])
	assert.Equal(t, campusID.String(), body["campus_id"])
	auditRepo.AssertExpectations(t)
}

func TestPublicHandler_VolunteerSignup_Errors(t *testing.T) {
	campusID := uuid.New()

	t.Run("400 on malformed json", func(t *testing.T) {
		h, _, _, _, _, _ := newPublicTestHandler()
		req := publicRequest(http.MethodPost, "/api/v1/public/volunteers", []byte(`{bad`), campusID)
		rec := httptest.NewRecorder()
		h.VolunteerSignup(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("400 on validation failure", func(t *testing.T) {
		h, _, _, _, _, _ := newPublicTestHandler()
		payload := []byte(`{"full_name":"","campus_id":"` + campusID.String() + `"}`)
		req := publicRequest(http.MethodPost, "/api/v1/public/volunteers", payload, campusID)
		rec := httptest.NewRecorder()
		h.VolunteerSignup(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
