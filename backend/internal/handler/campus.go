package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

// CampusHandler handles campus HTTP requests.
type CampusHandler struct {
	svc *service.CampusService
}

// NewCampusHandler creates a new CampusHandler.
func NewCampusHandler(svc *service.CampusService) *CampusHandler {
	return &CampusHandler{svc: svc}
}

// ListActive handles GET /campuses — returns active campuses only.
func (h *CampusHandler) ListActive(w http.ResponseWriter, r *http.Request) {
	campuses, err := h.svc.ListActive(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "campusHandler.ListActive: failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list campuses")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": campuses})
}

// ListAll handles GET /campuses/all — returns all campuses including inactive (ADMIN only).
func (h *CampusHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	campuses, err := h.svc.List(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "campusHandler.ListAll: failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list campuses")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": campuses})
}

// Get handles GET /campuses/{id}.
func (h *CampusHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid campus ID")
		return
	}

	campus, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "campus not found")
			return
		}
		slog.ErrorContext(r.Context(), "campusHandler.Get: failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get campus")
		return
	}

	writeJSON(w, http.StatusOK, campus)
}

// Create handles POST /campuses.
func (h *CampusHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input service.CreateCampusInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	campus, err := h.svc.Create(r.Context(), input)
	if err != nil {
		slog.ErrorContext(r.Context(), "campusHandler.Create: failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create campus")
		return
	}

	writeJSON(w, http.StatusCreated, campus)
}

// Update handles PUT /campuses/{id}.
func (h *CampusHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid campus ID")
		return
	}

	var input service.UpdateCampusInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	campus, err := h.svc.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "campus not found")
			return
		}
		slog.ErrorContext(r.Context(), "campusHandler.Update: failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update campus")
		return
	}

	writeJSON(w, http.StatusOK, campus)
}
