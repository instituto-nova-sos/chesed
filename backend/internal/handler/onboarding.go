package handler

import (
	"log/slog"
	"net/http"

	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

// OnboardingHandler handles onboarding status requests.
type OnboardingHandler struct {
	svc *service.OnboardingService
}

// NewOnboardingHandler creates a new OnboardingHandler.
func NewOnboardingHandler(svc *service.OnboardingService) *OnboardingHandler {
	return &OnboardingHandler{svc: svc}
}

// GetStatus handles GET /auth/me.
// Returns the onboarding status for the authenticated user.
func (h *OnboardingHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())

	status, err := h.svc.GetStatus(r.Context(), claims)
	if err != nil {
		slog.ErrorContext(r.Context(), "onboardingHandler.GetStatus: failed",
			"error", err.Error(),
		)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get onboarding status")
		return
	}

	writeJSON(w, http.StatusOK, status)
}
