package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

// AuditHandler exposes the read-only audit log viewer for compliance teams.
type AuditHandler struct {
	svc *service.AuditReadService
}

// NewAuditHandler creates an AuditHandler.
func NewAuditHandler(svc *service.AuditReadService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

// List handles GET /audit/logs. ADMIN only (enforced by route middleware).
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, ok := parseAuditFilter(w, r)
	if !ok {
		return
	}

	result, err := h.svc.List(r.Context(), filter)
	if err != nil {
		h.writeAuditError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// parseAuditFilter reads the audit viewer query string, writing a 400 and
// returning ok=false on any malformed value.
func parseAuditFilter(w http.ResponseWriter, r *http.Request) (domain.AuditLogFilter, bool) {
	q := r.URL.Query()
	filter := domain.AuditLogFilter{
		Page:    parseIntParam(q.Get("page"), 1),
		PerPage: parseIntParam(q.Get("per_page"), 50),
	}

	if v := q.Get("user_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_filter", "user_id must be a valid UUID")
			return domain.AuditLogFilter{}, false
		}
		filter.UserID = &id
	}
	if v := q.Get("entity_type"); v != "" {
		filter.EntityType = &v
	}
	if v := q.Get("action_type"); v != "" {
		filter.ActionType = &v
	}
	if v := q.Get("start"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_start", "start must be YYYY-MM-DD")
			return domain.AuditLogFilter{}, false
		}
		filter.Start = &t
	}
	if v := q.Get("end"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_end", "end must be YYYY-MM-DD")
			return domain.AuditLogFilter{}, false
		}
		filter.End = &t
	}
	return filter, true
}

func (h *AuditHandler) writeAuditError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "missing campus context")
	default:
		slog.ErrorContext(r.Context(), "auditHandler.List: failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to query audit logs")
	}
}
