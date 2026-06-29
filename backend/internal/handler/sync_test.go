package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

type mockSyncService struct{ mock.Mock }

func (m *mockSyncService) Push(ctx context.Context, req domain.SyncPushRequest) (*domain.SyncPushResponse, error) {
	args := m.Called(ctx, req)
	if v, ok := args.Get(0).(*domain.SyncPushResponse); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockSyncService) Pull(ctx context.Context, since time.Time, entityTypes []string, limit int) (*domain.SyncPullResponse, error) {
	args := m.Called(ctx, since, entityTypes, limit)
	if v, ok := args.Get(0).(*domain.SyncPullResponse); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func syncAuthedRequest(method, target string, body []byte) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, target, bytes.NewReader(body))
	} else {
		r = httptest.NewRequest(method, target, nil)
	}
	claims := auth.AuthClaims{
		Subject:  uuid.New().String(),
		Email:    "u@example.com",
		Roles:    []string{"VOLUNTEER"},
		CampusID: uuid.New(),
	}
	return r.WithContext(auth.NewContext(r.Context(), claims))
}

// --- POST /sync/push --------------------------------------------------------

func TestSyncHandler_Push_HappyPath(t *testing.T) {
	svc := new(mockSyncService)
	h := NewSyncHandler(svc)

	now := time.Now().UTC()
	syncID := uuid.New()
	serverID := uuid.New()
	svc.On("Push", mock.Anything, mock.AnythingOfType("domain.SyncPushRequest")).
		Return(&domain.SyncPushResponse{
			Results: []domain.SyncPushResult{
				{SyncID: syncID, Status: domain.SyncStatusCreated, ServerID: &serverID},
			},
			ServerTimestamp: now,
		}, nil)

	body, _ := json.Marshal(domain.SyncPushRequest{Records: []domain.SyncPushRecord{
		{EntityType: domain.SyncEntityPerson, SyncID: syncID, Data: map[string]any{"full_name": "x", "document_type": "CPF"}},
	}})
	req := syncAuthedRequest(http.MethodPost, "/api/v1/sync/push", body)
	rec := httptest.NewRecorder()

	h.Push(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp domain.SyncPushResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Len(t, resp.Results, 1)
	assert.Equal(t, domain.SyncStatusCreated, resp.Results[0].Status)
}

func TestSyncHandler_Push_MalformedJSON(t *testing.T) {
	svc := new(mockSyncService)
	h := NewSyncHandler(svc)

	req := syncAuthedRequest(http.MethodPost, "/api/v1/sync/push", []byte("{not json"))
	rec := httptest.NewRecorder()
	h.Push(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), `"error"`)
}

func TestSyncHandler_Push_BatchTooLarge(t *testing.T) {
	svc := new(mockSyncService)
	h := NewSyncHandler(svc)

	svc.On("Push", mock.Anything, mock.Anything).Return(nil, domain.ErrBatchTooLarge)

	body, _ := json.Marshal(domain.SyncPushRequest{Records: []domain.SyncPushRecord{
		{EntityType: domain.SyncEntityPerson, SyncID: uuid.New(), Data: map[string]any{"x": 1}},
	}})
	req := syncAuthedRequest(http.MethodPost, "/api/v1/sync/push", body)
	rec := httptest.NewRecorder()
	h.Push(rec, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}

func TestSyncHandler_Push_Forbidden(t *testing.T) {
	svc := new(mockSyncService)
	h := NewSyncHandler(svc)

	svc.On("Push", mock.Anything, mock.Anything).Return(nil, domain.ErrForbidden)

	body, _ := json.Marshal(domain.SyncPushRequest{Records: []domain.SyncPushRecord{
		{EntityType: domain.SyncEntityPerson, SyncID: uuid.New(), Data: map[string]any{"x": 1}},
	}})
	req := syncAuthedRequest(http.MethodPost, "/api/v1/sync/push", body)
	rec := httptest.NewRecorder()
	h.Push(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// --- GET /sync/pull ---------------------------------------------------------

func TestSyncHandler_Pull_HappyPath(t *testing.T) {
	svc := new(mockSyncService)
	h := NewSyncHandler(svc)

	since, _ := time.Parse(time.RFC3339, "2026-05-01T00:00:00Z")
	now := time.Now().UTC()
	svc.On("Pull", mock.Anything, since, []string{domain.SyncEntityPerson, domain.SyncEntityTriage}, 100).
		Return(&domain.SyncPullResponse{
			Records:         []domain.SyncPullRecord{},
			ServerTimestamp: now,
			HasMore:         false,
		}, nil)

	req := syncAuthedRequest(http.MethodGet,
		"/api/v1/sync/pull?since=2026-05-01T00%3A00%3A00Z&entity_types=person,triage", nil)
	rec := httptest.NewRecorder()
	h.Pull(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp domain.SyncPullResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.False(t, resp.HasMore)
}

func TestSyncHandler_Pull_MissingSince(t *testing.T) {
	svc := new(mockSyncService)
	h := NewSyncHandler(svc)

	req := syncAuthedRequest(http.MethodGet, "/api/v1/sync/pull?entity_types=person", nil)
	rec := httptest.NewRecorder()
	h.Pull(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, strings.ToLower(rec.Body.String()), "since")
}

func TestSyncHandler_Pull_InvalidSinceFormat(t *testing.T) {
	svc := new(mockSyncService)
	h := NewSyncHandler(svc)

	req := syncAuthedRequest(http.MethodGet, "/api/v1/sync/pull?since=not-a-date&entity_types=person", nil)
	rec := httptest.NewRecorder()
	h.Pull(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSyncHandler_Pull_InvalidEntityType(t *testing.T) {
	svc := new(mockSyncService)
	h := NewSyncHandler(svc)

	req := syncAuthedRequest(http.MethodGet,
		"/api/v1/sync/pull?since=2026-05-01T00%3A00%3A00Z&entity_types=campaign", nil)
	rec := httptest.NewRecorder()
	h.Pull(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSyncHandler_Pull_Forbidden(t *testing.T) {
	svc := new(mockSyncService)
	h := NewSyncHandler(svc)

	svc.On("Pull", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, domain.ErrForbidden)

	req := syncAuthedRequest(http.MethodGet,
		"/api/v1/sync/pull?since=2026-05-01T00%3A00%3A00Z&entity_types=person", nil)
	rec := httptest.NewRecorder()
	h.Pull(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}
