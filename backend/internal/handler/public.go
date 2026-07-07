package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/middleware"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

// PublicHandler serves the unauthenticated WordPress-facing API. The target
// campus is resolved and validated by PublicCampusValidator middleware and read
// back from context here; it is never trusted from raw request parsing.
type PublicHandler struct {
	svc *service.PublicService
}

// NewPublicHandler creates a new PublicHandler.
func NewPublicHandler(svc *service.PublicService) *PublicHandler {
	return &PublicHandler{svc: svc}
}

// publicPersonResponse is the lean, PII-minimal body returned on sign-up.
type publicPersonResponse struct {
	ID        uuid.UUID `json:"id"`
	FullName  string    `json:"full_name"`
	CampusID  uuid.UUID `json:"campus_id"`
	CreatedAt time.Time `json:"created_at"`
}

// ListCampaigns handles GET /api/v1/public/campaigns. It lists ACTIVE campaigns
// for the validated campus, paginated by page/per_page.
func (h *PublicHandler) ListCampaigns(w http.ResponseWriter, r *http.Request) {
	campusID := middleware.PublicCampusFromContext(r.Context())
	if campusID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "campus_id is required")
		return
	}

	q := r.URL.Query()
	page := parseIntParam(q.Get("page"), 1)
	perPage := parseIntParam(q.Get("per_page"), 20)

	result, err := h.svc.ListActiveCampaigns(r.Context(), campusID, page, perPage)
	if err != nil {
		h.writePublicError(w, r, err, "list public campaigns")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// VolunteerSignup handles POST /api/v1/public/volunteer-signup. The body was
// already buffered and restored by PublicCampusValidator, so it can be decoded
// here.
func (h *PublicHandler) VolunteerSignup(w http.ResponseWriter, r *http.Request) {
	var input service.PublicVolunteerInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	person, err := h.svc.RegisterVolunteer(r.Context(), input, extractIP(r), r.UserAgent())
	if err != nil {
		h.writePublicError(w, r, err, "register volunteer")
		return
	}

	writeJSON(w, http.StatusCreated, publicPersonResponse{
		ID:        person.ID,
		FullName:  person.FullName,
		CampusID:  person.CampusID,
		CreatedAt: person.CreatedAt,
	})
}

// writePublicError maps sentinel errors to the public HTTP contract. Messages
// stay generic so foreign-campus existence is never disclosed.
func (h *PublicHandler) writePublicError(w http.ResponseWriter, r *http.Request, err error, action string) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		writeError(w, http.StatusBadRequest, "validation_error", "invalid field values")
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
	default:
		slog.ErrorContext(r.Context(), "publicHandler: failed", "action", action, "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to "+action)
	}
}
