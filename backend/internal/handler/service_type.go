package handler

import (
	"log/slog"
	"net/http"

	"github.com/instituto-nova-sos/chesed/internal/service"
)

// ServiceTypeHandler handles service type HTTP requests.
type ServiceTypeHandler struct {
	svc *service.ServiceTypeService
}

// NewServiceTypeHandler creates a new ServiceTypeHandler.
func NewServiceTypeHandler(svc *service.ServiceTypeService) *ServiceTypeHandler {
	return &ServiceTypeHandler{svc: svc}
}

// List handles GET /api/v1/service-types.
func (h *ServiceTypeHandler) List(w http.ResponseWriter, r *http.Request) {
	types, err := h.svc.ListActive(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "serviceTypeHandler.List: failed",
			"error", err.Error(),
		)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list service types")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": types})
}
