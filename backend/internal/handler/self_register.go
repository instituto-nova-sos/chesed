package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

// SelfRegisterHandler handles self-registration for new users.
type SelfRegisterHandler struct {
	svc *service.SelfRegisterService
}

// NewSelfRegisterHandler creates a new SelfRegisterHandler.
func NewSelfRegisterHandler(svc *service.SelfRegisterService) *SelfRegisterHandler {
	return &SelfRegisterHandler{svc: svc}
}

// Register handles POST /self-register.
func (h *SelfRegisterHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input service.SelfRegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if err := validateStruct(input); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	person, err := h.svc.Register(r.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrDuplicate) {
			writeError(w, http.StatusConflict, "already_registered", "user is already registered")
			return
		}
		if errors.Is(err, domain.ErrInvalidCPF) {
			writeError(w, http.StatusBadRequest, "invalid_cpf", "CPF inválido")
			return
		}
		slog.ErrorContext(r.Context(), "selfRegisterHandler.Register: failed",
			"error", err.Error(),
		)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to register")
		return
	}

	writeJSON(w, http.StatusCreated, person)
}
