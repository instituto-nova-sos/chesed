package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockAuditListRepo struct {
	mock.Mock
}

func (m *mockAuditListRepo) List(ctx context.Context, filter domain.AuditLogFilter) (*domain.AuditLogListResult, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AuditLogListResult), args.Error(1)
}

func newAuditHandler(repo *mockAuditListRepo) *AuditHandler {
	return NewAuditHandler(service.NewAuditReadService(repo))
}

func TestAuditHandler_List(t *testing.T) {
	t.Run("returns 200 with paginated logs", func(t *testing.T) {
		repo := new(mockAuditListRepo)
		email := "maria@example.com"
		repo.On("List", mock.Anything, mock.Anything).Return(&domain.AuditLogListResult{
			Data: []domain.AuditLogListItem{
				{ID: uuid.New(), UserEmail: &email, ActionType: "UPDATE", EntityType: "person"},
			},
			Pagination: domain.Pagination{Page: 1, PerPage: 50, Total: 1},
		}, nil)
		h := newAuditHandler(repo)

		req := authedRequest(http.MethodGet, "/api/v1/audit/logs")
		rec := httptest.NewRecorder()
		h.List(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var got domain.AuditLogListResult
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		require.Len(t, got.Data, 1)
		assert.Equal(t, "UPDATE", got.Data[0].ActionType)
		assert.Equal(t, 1, got.Pagination.Total)
	})

	t.Run("passes filters through to the service", func(t *testing.T) {
		repo := new(mockAuditListRepo)
		userID := uuid.New()
		repo.On("List", mock.Anything, mock.MatchedBy(func(f domain.AuditLogFilter) bool {
			return f.UserID != nil && *f.UserID == userID &&
				f.EntityType != nil && *f.EntityType == "person" &&
				f.ActionType != nil && *f.ActionType == "UPDATE" &&
				f.Start != nil && f.End != nil
		})).Return(&domain.AuditLogListResult{Data: []domain.AuditLogListItem{}}, nil)
		h := newAuditHandler(repo)

		req := authedRequest(http.MethodGet, "/api/v1/audit/logs?user_id="+userID.String()+"&entity_type=person&action_type=UPDATE&start=2026-01-01&end=2026-03-31&page=2&per_page=10")
		rec := httptest.NewRecorder()
		h.List(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		repo.AssertExpectations(t)
	})

	t.Run("returns 400 on malformed user_id", func(t *testing.T) {
		h := newAuditHandler(new(mockAuditListRepo))
		req := authedRequest(http.MethodGet, "/api/v1/audit/logs?user_id=not-a-uuid")
		rec := httptest.NewRecorder()
		h.List(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("returns 400 on malformed date", func(t *testing.T) {
		h := newAuditHandler(new(mockAuditListRepo))
		req := authedRequest(http.MethodGet, "/api/v1/audit/logs?start=2026/01/01")
		rec := httptest.NewRecorder()
		h.List(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
