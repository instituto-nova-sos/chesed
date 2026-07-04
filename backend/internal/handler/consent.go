package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

// ConsentHandler handles HTTP requests for LGPD consent management.
type ConsentHandler struct {
	svc *service.ConsentService
}

// NewConsentHandler creates a new ConsentHandler.
func NewConsentHandler(svc *service.ConsentService) *ConsentHandler {
	return &ConsentHandler{svc: svc}
}

// consentListResponse is the documented list envelope (docs/11).
type consentListResponse struct {
	Data []domain.Consent `json:"data"`
}

// Create handles POST /consents.
func (h *ConsentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input service.CreateConsentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if err := validateStruct(input); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	consent, err := h.svc.CreateConsent(r.Context(), input)
	if err != nil {
		h.writeConsentError(w, r, err, "create consent")
		return
	}
	writeJSON(w, http.StatusCreated, consent)
}

// ListByPerson handles GET /persons/{id}/consents.
func (h *ConsentHandler) ListByPerson(w http.ResponseWriter, r *http.Request) {
	personID, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid person ID format")
		return
	}

	consents, err := h.svc.ListPersonConsents(r.Context(), personID)
	if err != nil {
		h.writeConsentError(w, r, err, "list consents")
		return
	}
	writeJSON(w, http.StatusOK, consentListResponse{Data: consents})
}

// Revoke handles PATCH /consents/{id}/revoke.
func (h *ConsentHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid consent ID format")
		return
	}

	var input service.RevokeConsentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if err := validateStruct(input); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	consent, err := h.svc.RevokeConsent(r.Context(), id, input)
	if err != nil {
		h.writeConsentError(w, r, err, "revoke consent")
		return
	}
	writeJSON(w, http.StatusOK, consent)
}

// writeConsentError maps domain sentinel errors to the documented HTTP
// contract. Validation messages stay generic so cross-campus references
// never disclose foreign existence.
func (h *ConsentHandler) writeConsentError(w http.ResponseWriter, r *http.Request, err error, action string) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "missing campus context")
	case errors.Is(err, domain.ErrDuplicate):
		writeError(w, http.StatusConflict, "duplicate", "an active consent of this type already exists for the person")
	case errors.Is(err, domain.ErrValidation):
		writeError(w, http.StatusBadRequest, "validation_error", "invalid reference or field values")
	default:
		slog.ErrorContext(r.Context(), "consentHandler: failed", "action", action, "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to "+action)
	}
}
