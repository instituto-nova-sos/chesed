package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

// CampaignHandler handles HTTP requests for campaign management.
type CampaignHandler struct {
	svc *service.CampaignService
}

// NewCampaignHandler creates a new CampaignHandler.
func NewCampaignHandler(svc *service.CampaignService) *CampaignHandler {
	return &CampaignHandler{svc: svc}
}

// Create handles POST /campaigns.
func (h *CampaignHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input service.CreateCampaignInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if err := validateStruct(input); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	campaign, err := h.svc.CreateCampaign(r.Context(), input)
	if err != nil {
		h.writeCampaignError(w, r, err, "create campaign")
		return
	}
	writeJSON(w, http.StatusCreated, campaign)
}

// Get handles GET /campaigns/{id}.
func (h *CampaignHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid campaign ID format")
		return
	}

	detail, err := h.svc.GetCampaign(r.Context(), id)
	if err != nil {
		h.writeCampaignError(w, r, err, "fetch campaign")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// List handles GET /campaigns.
func (h *CampaignHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, code, msg := parseCampaignFilter(r)
	if code != "" {
		writeError(w, http.StatusBadRequest, code, msg)
		return
	}

	result, err := h.svc.ListCampaigns(r.Context(), filter)
	if err != nil {
		h.writeCampaignError(w, r, err, "list campaigns")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// Update handles PUT /campaigns/{id}.
func (h *CampaignHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid campaign ID format")
		return
	}

	var input service.UpdateCampaignInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if err := validateStruct(input); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	campaign, err := h.svc.UpdateCampaign(r.Context(), id, input)
	if err != nil {
		h.writeCampaignError(w, r, err, "update campaign")
		return
	}
	writeJSON(w, http.StatusOK, campaign)
}

// AddTeamMember handles POST /campaigns/{id}/team.
func (h *CampaignHandler) AddTeamMember(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid campaign ID format")
		return
	}

	var input service.AddTeamMemberInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if err := validateStruct(input); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	member, err := h.svc.AddTeamMember(r.Context(), id, input)
	if err != nil {
		h.writeCampaignError(w, r, err, "add team member")
		return
	}
	writeJSON(w, http.StatusCreated, member)
}

// RemoveTeamMember handles DELETE /campaigns/{id}/team/{personId}.
func (h *CampaignHandler) RemoveTeamMember(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid campaign ID format")
		return
	}
	personID, err := uuid.Parse(chi.URLParam(r, "personId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid person ID format")
		return
	}

	if err := h.svc.RemoveTeamMember(r.Context(), id, personID); err != nil {
		h.writeCampaignError(w, r, err, "remove team member")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeCampaignError maps domain sentinel errors to the documented HTTP
// contract. Validation messages stay generic so cross-campus references
// never disclose foreign existence.
func (h *CampaignHandler) writeCampaignError(w http.ResponseWriter, r *http.Request, err error, action string) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "campaign not found")
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "missing campus context")
	case errors.Is(err, domain.ErrDuplicate):
		writeError(w, http.StatusConflict, "duplicate", "person already assigned to this campaign")
	case errors.Is(err, domain.ErrValidation):
		writeError(w, http.StatusBadRequest, "validation_error", "invalid reference or field values")
	default:
		slog.ErrorContext(r.Context(), "campaignHandler: failed", "action", action, "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to "+action)
	}
}

// parseCampaignFilter builds a CampaignFilter from the query string. On a
// malformed parameter it returns a non-empty (code, message) pair.
func parseCampaignFilter(r *http.Request) (domain.CampaignFilter, string, string) {
	q := r.URL.Query()
	filter := domain.CampaignFilter{
		Page:    parseIntParam(q.Get("page"), 1),
		PerPage: parseIntParam(q.Get("per_page"), 20),
	}

	if v := q.Get("status"); v != "" {
		if !slices.Contains(domain.CampaignStatuses, v) {
			return filter, "invalid_status", "invalid status filter"
		}
		filter.Status = &v
	}
	if v := q.Get("campaign_type"); v != "" {
		if !slices.Contains(domain.CampaignTypes, v) {
			return filter, "invalid_campaign_type", "invalid campaign_type filter"
		}
		filter.Type = &v
	}
	return filter, "", ""
}
