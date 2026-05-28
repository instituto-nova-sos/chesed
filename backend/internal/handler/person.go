package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

// PersonHandler handles HTTP requests for person management.
type PersonHandler struct {
	svc *service.PersonService
}

// NewPersonHandler creates a new PersonHandler.
func NewPersonHandler(svc *service.PersonService) *PersonHandler {
	return &PersonHandler{svc: svc}
}

// Create handles POST /persons.
func (h *PersonHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input service.CreatePersonInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if err := validateStruct(input); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	person, err := h.svc.CreatePerson(r.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrDuplicateEmail) {
			writeError(w, http.StatusConflict, "duplicate_email", "a person with this email already exists")
			return
		}
		if errors.Is(err, domain.ErrDuplicatePhone) {
			writeError(w, http.StatusConflict, "duplicate_phone", "a person with this phone already exists")
			return
		}
		if errors.Is(err, domain.ErrDuplicate) {
			writeError(w, http.StatusConflict, "duplicate", "person with this document already exists")
			return
		}
		if errors.Is(err, domain.ErrInvalidCPF) {
			writeError(w, http.StatusBadRequest, "invalid_cpf", "CPF inválido")
			return
		}
		slog.ErrorContext(r.Context(), "personHandler.Create: failed",
			"error", err.Error(),
		)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create person")
		return
	}

	writeJSON(w, http.StatusCreated, person)
}

// List handles GET /persons.
func (h *PersonHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	page := parseIntParam(r.URL.Query().Get("page"), 1)
	perPage := parseIntParam(r.URL.Query().Get("per_page"), 20)
	agreementStatus := r.URL.Query().Get("agreement_status")

	result, err := h.svc.ListPersons(r.Context(), query, page, perPage, agreementStatus)
	if err != nil {
		slog.ErrorContext(r.Context(), "personHandler.List: failed",
			"error", err.Error(),
		)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list persons")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Get handles GET /persons/{id}.
func (h *PersonHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid person ID format")
		return
	}

	detail, err := h.svc.GetPerson(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "person not found")
			return
		}
		slog.ErrorContext(r.Context(), "personHandler.Get: failed",
			"error", err.Error(),
		)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get person")
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

// Update handles PUT /persons/{id}.
func (h *PersonHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid person ID format")
		return
	}

	var input service.UpdatePersonInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if err := validateStruct(input); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	person, err := h.svc.UpdatePerson(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "person not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidCPF) {
			writeError(w, http.StatusBadRequest, "invalid_cpf", "CPF inválido")
			return
		}
		slog.ErrorContext(r.Context(), "personHandler.Update: failed",
			"error", err.Error(),
		)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update person")
		return
	}

	writeJSON(w, http.StatusOK, person)
}

// GetHistory handles GET /persons/{id}/history.
func (h *PersonHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid person ID format")
		return
	}

	history, err := h.svc.GetHistory(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "person not found")
			return
		}
		slog.ErrorContext(r.Context(), "personHandler.GetHistory: failed",
			"error", err.Error(),
		)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get history")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"person": map[string]string{"id": id.String()},
		"history": history,
	})
}

// CheckDuplicate handles GET /persons/check-duplicate.
func (h *PersonHandler) CheckDuplicate(w http.ResponseWriter, r *http.Request) {
	documentType := r.URL.Query().Get("document_type")
	documentNumber := r.URL.Query().Get("document_number")

	if documentType == "" || documentNumber == "" {
		writeError(w, http.StatusBadRequest, "missing_params", "document_type and document_number are required")
		return
	}

	result, err := h.svc.CheckDuplicate(r.Context(), documentType, documentNumber)
	if err != nil {
		slog.ErrorContext(r.Context(), "personHandler.CheckDuplicate: failed",
			"error", err.Error(),
		)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to check duplicates")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// AddRole handles POST /persons/{id}/roles.
func (h *PersonHandler) AddRole(w http.ResponseWriter, r *http.Request) {
	personID, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid person ID format")
		return
	}

	var input service.AddRoleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if err := validateStruct(input); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	role, err := h.svc.AddRole(r.Context(), personID, input)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "person not found")
			return
		}
		if errors.Is(err, domain.ErrDuplicate) {
			writeError(w, http.StatusConflict, "duplicate", "person already has this role")
			return
		}
		slog.ErrorContext(r.Context(), "personHandler.AddRole: failed",
			"error", err.Error(),
		)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to add role")
		return
	}

	writeJSON(w, http.StatusCreated, role)
}

// ToggleRole handles PATCH /persons/{id}/roles/{roleId}.
func (h *PersonHandler) ToggleRole(w http.ResponseWriter, r *http.Request) {
	personID, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid person ID format")
		return
	}

	roleID, err := uuid.Parse(chi.URLParam(r, "roleId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid role ID format")
		return
	}

	var body struct {
		IsActive bool `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	role, err := h.svc.ToggleRole(r.Context(), personID, roleID, body.IsActive)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "person or role not found")
			return
		}
		slog.ErrorContext(r.Context(), "personHandler.ToggleRole: failed",
			"error", err.Error(),
		)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to toggle role")
		return
	}

	writeJSON(w, http.StatusOK, role)
}

func parseUUIDParam(r *http.Request, name string) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, name))
}

func parseIntParam(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(s)
	if err != nil || val < 1 {
		return defaultVal
	}
	return val
}
