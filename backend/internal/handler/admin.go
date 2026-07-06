package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

// AdminHandler exposes campus-scoped administrative operations for compliance.
type AdminHandler struct {
	retentionSvc *service.RetentionService
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(retentionSvc *service.RetentionService) *AdminHandler {
	return &AdminHandler{retentionSvc: retentionSvc}
}

// RunRetention handles POST /admin/retention/run — a synchronous retention
// sweep over the caller's campus (ADMIN only, enforced at the route).
func (h *AdminHandler) RunRetention(w http.ResponseWriter, r *http.Request) {
	summary, err := h.retentionSvc.Run(r.Context())
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "missing campus context")
			return
		}
		slog.ErrorContext(r.Context(), "adminHandler: retention run failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "retention sweep failed")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}
