package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

// TriageHandler handles HTTP requests for triage management.
type TriageHandler struct {
	svc *service.TriageService
}

// NewTriageHandler creates a new TriageHandler.
func NewTriageHandler(svc *service.TriageService) *TriageHandler {
	return &TriageHandler{svc: svc}
}

// Create handles POST /triages.
func (h *TriageHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input service.CreateTriageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if err := validateStruct(input); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	triage, err := h.svc.CreateTriage(r.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "missing campus context")
			return
		}
		slog.ErrorContext(r.Context(), "triageHandler.Create: failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create triage")
		return
	}

	writeJSON(w, http.StatusCreated, triage)
}

// Get handles GET /triages/{id}.
func (h *TriageHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid triage ID format")
		return
	}

	triage, err := h.svc.GetTriage(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "triage not found")
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "missing campus context")
			return
		}
		slog.ErrorContext(r.Context(), "triageHandler.Get: failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to fetch triage")
		return
	}

	writeJSON(w, http.StatusOK, triage)
}

// List handles GET /triages.
func (h *TriageHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, code, msg := parseTriageFilter(r)
	if code != "" {
		writeError(w, http.StatusBadRequest, code, msg)
		return
	}

	result, err := h.svc.ListTriages(r.Context(), filter)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "missing campus context")
			return
		}
		slog.ErrorContext(r.Context(), "triageHandler.List: failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list triages")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// parseTriageFilter builds a TriageFilter from the query string. On a malformed
// parameter it returns a non-empty (code, message) pair.
func parseTriageFilter(r *http.Request) (domain.TriageFilter, string, string) {
	q := r.URL.Query()
	filter := domain.TriageFilter{
		Page:    parseIntParam(q.Get("page"), 1),
		PerPage: parseIntParam(q.Get("per_page"), 20),
	}

	if v := q.Get("person_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return filter, "invalid_person_id", "invalid person_id format"
		}
		filter.PersonID = &id
	}
	if v := q.Get("from"); v != "" {
		t, err := parseQueryDate(v)
		if err != nil {
			return filter, "invalid_from", "invalid from date"
		}
		filter.From = t
	}
	if v := q.Get("to"); v != "" {
		t, err := parseQueryDate(v)
		if err != nil {
			return filter, "invalid_to", "invalid to date"
		}
		filter.To = t
	}
	return filter, "", ""
}

// Update handles PATCH /triages/{id}.
func (h *TriageHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid triage ID format")
		return
	}

	var input service.UpdateTriageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if err := validateStruct(input); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	triage, err := h.svc.UpdateTriage(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "triage not found")
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "missing campus context")
			return
		}
		slog.ErrorContext(r.Context(), "triageHandler.Update: failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update triage")
		return
	}

	writeJSON(w, http.StatusOK, triage)
}

func parseQueryDate(s string) (*time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t, nil
		}
	}
	return nil, errors.New("unsupported date format")
}
