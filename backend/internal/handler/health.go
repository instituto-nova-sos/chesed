package handler

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthHandler handles health check requests.
type HealthHandler struct {
	pool *pgxpool.Pool
}

// NewHealthHandler creates a new HealthHandler.
// Pool may be nil for basic health checks without DB.
func NewHealthHandler(pool *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{pool: pool}
}

// ServeHTTP responds with health status including DB connectivity.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dbStatus := "ok"
	if h.pool != nil {
		if err := h.pool.Ping(r.Context()); err != nil {
			dbStatus = "error"
			slog.ErrorContext(r.Context(), "healthHandler.ServeHTTP: db ping failed",
				"error", err.Error(),
			)
		}
	}

	status := http.StatusOK
	if dbStatus != "ok" {
		status = http.StatusServiceUnavailable
	}

	writeJSON(w, status, map[string]any{
		"status":   dbStatus,
		"database": dbStatus,
	})
}
