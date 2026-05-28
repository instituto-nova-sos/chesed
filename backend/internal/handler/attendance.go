package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

// AttendanceHandler handles HTTP requests for attendance management.
type AttendanceHandler struct {
	svc *service.AttendanceService
}

// NewAttendanceHandler creates a new AttendanceHandler.
func NewAttendanceHandler(svc *service.AttendanceService) *AttendanceHandler {
	return &AttendanceHandler{svc: svc}
}

// Create handles POST /attendances.
func (h *AttendanceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input service.CreateAttendanceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if err := validateStruct(input); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	att, err := h.svc.CreateAttendance(r.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "missing campus context")
			return
		}
		slog.ErrorContext(r.Context(), "attendanceHandler.Create: failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create attendance")
		return
	}

	writeJSON(w, http.StatusCreated, att)
}

// Get handles GET /attendances/{id}.
func (h *AttendanceHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid attendance ID format")
		return
	}

	detail, err := h.svc.GetAttendance(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "attendance not found")
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "missing campus context")
			return
		}
		slog.ErrorContext(r.Context(), "attendanceHandler.Get: failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to fetch attendance")
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

// List handles GET /attendances.
func (h *AttendanceHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := domain.AttendanceFilter{
		Page:    parseIntParam(r.URL.Query().Get("page"), 1),
		PerPage: parseIntParam(r.URL.Query().Get("per_page"), 20),
	}

	if personIDStr := r.URL.Query().Get("person_id"); personIDStr != "" {
		id, err := uuid.Parse(personIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_person_id", "invalid person_id format")
			return
		}
		filter.PersonID = &id
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = &status
	}
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		t, err := parseQueryDate(fromStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_from", "invalid from date")
			return
		}
		filter.From = t
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		t, err := parseQueryDate(toStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_to", "invalid to date")
			return
		}
		filter.To = t
	}

	result, err := h.svc.ListAttendances(r.Context(), filter)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "missing campus context")
			return
		}
		slog.ErrorContext(r.Context(), "attendanceHandler.List: failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list attendances")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Transition handles POST /attendances/{id}/transitions.
func (h *AttendanceHandler) Transition(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid attendance ID format")
		return
	}

	var input service.TransitionAttendanceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if err := validateStruct(input); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	att, err := h.svc.TransitionAttendance(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "attendance not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidTransition) {
			writeError(w, http.StatusConflict, "invalid_transition", "state transition not allowed")
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "missing campus context")
			return
		}
		slog.ErrorContext(r.Context(), "attendanceHandler.Transition: failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to transition attendance")
		return
	}

	writeJSON(w, http.StatusOK, att)
}

// UpdateNotes handles PATCH /attendances/{id}/notes.
func (h *AttendanceHandler) UpdateNotes(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid attendance ID format")
		return
	}

	var input service.UpdateAttendanceNotesInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if err := validateStruct(input); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	att, err := h.svc.UpdateNotes(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "attendance not found")
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "missing campus context")
			return
		}
		slog.ErrorContext(r.Context(), "attendanceHandler.UpdateNotes: failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update notes")
		return
	}

	writeJSON(w, http.StatusOK, att)
}
