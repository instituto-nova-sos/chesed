package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// SyncService is the subset of service.SyncService used by the HTTP handler.
type SyncService interface {
	Push(ctx context.Context, req domain.SyncPushRequest) (*domain.SyncPushResponse, error)
	Pull(ctx context.Context, since time.Time, entityTypes []string, limit int) (*domain.SyncPullResponse, error)
}

// defaultPullLimit caps per-call pull size for delta responses.
const defaultPullLimit = 100

// SyncHandler exposes the offline sync endpoints over HTTP.
type SyncHandler struct {
	svc SyncService
}

// NewSyncHandler constructs a SyncHandler.
func NewSyncHandler(svc SyncService) *SyncHandler {
	return &SyncHandler{svc: svc}
}

// Push handles POST /api/v1/sync/push.
func (h *SyncHandler) Push(w http.ResponseWriter, r *http.Request) {
	var req domain.SyncPushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "malformed JSON body")
		return
	}

	resp, err := h.svc.Push(r.Context(), req)
	if err != nil {
		writeSyncError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Pull handles GET /api/v1/sync/pull.
func (h *SyncHandler) Pull(w http.ResponseWriter, r *http.Request) {
	rawSince := strings.TrimSpace(r.URL.Query().Get("since"))
	if rawSince == "" {
		writeError(w, http.StatusBadRequest, "missing_since", "the 'since' query parameter is required")
		return
	}
	since, err := time.Parse(time.RFC3339, rawSince)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_since", "'since' must be an RFC3339 timestamp")
		return
	}

	rawTypes := strings.TrimSpace(r.URL.Query().Get("entity_types"))
	if rawTypes == "" {
		rawTypes = strings.Join([]string{
			domain.SyncEntityPerson,
			domain.SyncEntityTriage,
			domain.SyncEntityAttendance,
		}, ",")
	}
	entityTypes, err := domain.ParseSyncEntityTypes(rawTypes)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_types", err.Error())
		return
	}

	resp, err := h.svc.Pull(r.Context(), since, entityTypes, defaultPullLimit)
	if err != nil {
		writeSyncError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeSyncError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrBatchTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "batch_too_large", err.Error())
	case errors.Is(err, domain.ErrInvalidEntityType):
		writeError(w, http.StatusBadRequest, "invalid_entity_types", err.Error())
	case errors.Is(err, domain.ErrMissingSyncID), errors.Is(err, domain.ErrMissingData):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "campus context required")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
	}
}
