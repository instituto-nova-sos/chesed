package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockRetentionRepo struct{ mock.Mock }

func (m *mockRetentionRepo) ListExpiredPersonIDs(ctx context.Context, campusID uuid.UUID, olderThan time.Time) ([]uuid.UUID, error) {
	args := m.Called(ctx, campusID, olderThan)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

type mockAnonymizer struct{ mock.Mock }

func (m *mockAnonymizer) Anonymize(ctx context.Context, personID, campusID uuid.UUID) error {
	return m.Called(ctx, personID, campusID).Error(0)
}

type mockAdminAuditRepo struct{ mock.Mock }

func (m *mockAdminAuditRepo) Create(ctx context.Context, entry domain.AuditLog) error {
	return m.Called(ctx, entry).Error(0)
}

func newAdminHandler() (*AdminHandler, *mockRetentionRepo, *mockAnonymizer) {
	repo := new(mockRetentionRepo)
	anon := new(mockAnonymizer)
	auditRepo := new(mockAdminAuditRepo)
	auditRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	svc := service.NewRetentionService(repo, anon, service.NewAuditService(auditRepo))
	return NewAdminHandler(svc), repo, anon
}

func adminReq(campusID uuid.UUID) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/retention/run", nil)
	claims := auth.AuthClaims{Subject: uuid.New().String(), Roles: []string{"ADMIN"}, CampusID: campusID}
	return req.WithContext(auth.NewContext(req.Context(), claims))
}

func TestAdminHandler_RunRetention(t *testing.T) {
	t.Run("returns the sweep summary", func(t *testing.T) {
		h, repo, anon := newAdminHandler()
		campusID := uuid.New()
		id := uuid.New()
		repo.On("ListExpiredPersonIDs", mock.Anything, campusID, mock.Anything).Return([]uuid.UUID{id}, nil)
		anon.On("Anonymize", mock.Anything, id, campusID).Return(nil)

		rec := httptest.NewRecorder()
		h.RunRetention(rec, adminReq(campusID))

		require.Equal(t, http.StatusOK, rec.Code)
		var body service.RetentionSummary
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, 1, body.Scanned)
		assert.Equal(t, 1, body.Anonymized)
	})

	t.Run("missing campus is 403", func(t *testing.T) {
		h, _, _ := newAdminHandler()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/retention/run", nil)
		req = req.WithContext(auth.NewContext(req.Context(), auth.AuthClaims{Roles: []string{"ADMIN"}}))

		rec := httptest.NewRecorder()
		h.RunRetention(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}
