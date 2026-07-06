package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

// DonationHandler handles HTTP requests for donation tracking.
type DonationHandler struct {
	svc *service.DonationService
}

// NewDonationHandler creates a new DonationHandler.
func NewDonationHandler(svc *service.DonationService) *DonationHandler {
	return &DonationHandler{svc: svc}
}

// Create handles POST /donations.
func (h *DonationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input service.CreateDonationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if err := validateStruct(input); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	donation, err := h.svc.CreateDonation(r.Context(), input)
	if err != nil {
		h.writeDonationError(w, r, err, "create donation")
		return
	}
	writeJSON(w, http.StatusCreated, donation)
}

// Get handles GET /donations/{id}.
func (h *DonationHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid donation ID format")
		return
	}

	detail, err := h.svc.GetDonation(r.Context(), id)
	if err != nil {
		h.writeDonationError(w, r, err, "fetch donation")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// List handles GET /donations.
func (h *DonationHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, code, msg := parseDonationFilter(r)
	if code != "" {
		writeError(w, http.StatusBadRequest, code, msg)
		return
	}

	result, err := h.svc.ListDonations(r.Context(), filter)
	if err != nil {
		h.writeDonationError(w, r, err, "list donations")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// Update handles PUT /donations/{id}.
func (h *DonationHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid donation ID format")
		return
	}

	var input service.UpdateDonationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if err := validateStruct(input); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	donation, err := h.svc.UpdateDonation(r.Context(), id, input)
	if err != nil {
		h.writeDonationError(w, r, err, "update donation")
		return
	}
	writeJSON(w, http.StatusOK, donation)
}

// Receipt handles GET /donations/{id}/receipt. It issues (once) and returns a
// presigned download URL for the donation's receipt PDF.
func (h *DonationHandler) Receipt(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid donation ID format")
		return
	}

	download, err := h.svc.GenerateReceipt(r.Context(), id)
	if err != nil {
		h.writeDonationError(w, r, err, "generate receipt")
		return
	}
	writeJSON(w, http.StatusOK, download)
}

// writeDonationError maps domain sentinel errors to the documented HTTP
// contract. Validation messages stay generic so cross-campus references never
// disclose foreign existence.
func (h *DonationHandler) writeDonationError(w http.ResponseWriter, r *http.Request, err error, action string) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "donation not found")
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "missing campus context")
	case errors.Is(err, domain.ErrDuplicate):
		writeError(w, http.StatusConflict, "duplicate", "receipt number already in use")
	case errors.Is(err, domain.ErrValidation):
		writeError(w, http.StatusBadRequest, "validation_error", "invalid reference or field values")
	default:
		slog.ErrorContext(r.Context(), "donationHandler: failed", "action", action, "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to "+action)
	}
}

// parseDonationFilter builds a DonationFilter from the query string. On a
// malformed parameter it returns a non-empty (code, message) pair.
func parseDonationFilter(r *http.Request) (domain.DonationFilter, string, string) {
	q := r.URL.Query()
	filter := domain.DonationFilter{
		Page:    parseIntParam(q.Get("page"), 1),
		PerPage: parseIntParam(q.Get("per_page"), 20),
	}

	if v := q.Get("donation_type"); v != "" {
		if !slices.Contains(domain.DonationTypes, v) {
			return filter, "invalid_donation_type", "invalid donation_type filter"
		}
		filter.DonationType = &v
	}
	if v := q.Get("campaign_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return filter, "invalid_campaign_id", "invalid campaign_id filter"
		}
		filter.CampaignID = &id
	}
	return filter, "", ""
}
