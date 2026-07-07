package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type publicCampusCtxKey struct{}

// WithPublicCampus stashes the validated campus for a public (unauthenticated)
// request so PublicCampusTx can scope the request transaction to it. Public
// routes carry no auth claims, so the campus travels here instead.
func WithPublicCampus(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, publicCampusCtxKey{}, id)
}

// PublicCampusFromContext returns the validated public campus stashed by
// PublicCampusValidator, or uuid.Nil when absent. Public handlers use it to
// scope service calls without re-parsing the request.
func PublicCampusFromContext(ctx context.Context) uuid.UUID {
	id, ok := ctx.Value(publicCampusCtxKey{}).(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}

// publicCampusSource reads the campus stashed by PublicCampusValidator. It is
// the campusSource for the public transaction middleware.
func publicCampusSource(r *http.Request) (uuid.UUID, bool) {
	id := PublicCampusFromContext(r.Context())
	if id == uuid.Nil {
		return uuid.Nil, false
	}
	return id, true
}

// PublicCampusTx opens a per-request campus transaction for public routes,
// scoped to the campus PublicCampusValidator put in context. It runs on the
// supplied (non-owner) pool so RLS applies, and must be chained AFTER
// PublicCampusValidator.
func PublicCampusTx(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return campusTxWithSource(poolBeginner{pool: pool}, publicCampusSource)
}

// PublicCampusLookup is the minimal, campus-table lookup the validator needs.
// The campus table is RLS-exempt, so this runs on the pool directly (before any
// campus transaction is open), letting the validator resolve a campus before
// PublicCampusTx scopes the request to it.
type PublicCampusLookup interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Campus, error)
}

// PublicCampusValidator resolves and validates the target campus for a public
// request, then stores it in context for PublicCampusTx. For GET it reads the
// campus_id query parameter; for other methods it peeks the JSON body's
// campus_id field, restoring the body so the downstream handler can decode it
// again. Malformed input yields 400 invalid_request; an unknown or inactive
// campus yields 404 not_found (never disclosing which).
func PublicCampusValidator(repo PublicCampusLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawID, ok := extractCampusID(w, r)
			if !ok {
				return
			}
			campusID, ok := resolveCampus(w, r, repo, rawID)
			if !ok {
				return
			}
			next.ServeHTTP(w, r.WithContext(WithPublicCampus(r.Context(), campusID)))
		})
	}
}

// extractCampusID reads the raw campus_id string from the query (GET) or the
// JSON body (other methods). On failure it writes a 400 and returns false.
func extractCampusID(w http.ResponseWriter, r *http.Request) (string, bool) {
	if r.Method == http.MethodGet {
		raw := r.URL.Query().Get("campus_id")
		if raw == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "campus_id is required")
			return "", false
		}
		return raw, true
	}
	return campusIDFromBody(w, r)
}

// campusIDFromBody reads the request body once to extract campus_id, then
// restores the body (via io.NopCloser over the buffered bytes) so the handler
// can decode the full payload again.
func campusIDFromBody(w http.ResponseWriter, r *http.Request) (string, bool) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "could not read request body")
		return "", false
	}
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	var peek struct {
		CampusID string `json:"campus_id"`
	}
	if err := json.Unmarshal(bodyBytes, &peek); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return "", false
	}
	if peek.CampusID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "campus_id is required")
		return "", false
	}
	return peek.CampusID, true
}

// resolveCampus parses the raw campus id and confirms the campus exists and is
// active. On a bad id it writes 400; on unknown/inactive it writes 404.
func resolveCampus(w http.ResponseWriter, r *http.Request, repo PublicCampusLookup, rawID string) (uuid.UUID, bool) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid campus_id format")
		return uuid.Nil, false
	}
	campus, err := repo.FindByID(r.Context(), id)
	if err != nil || campus == nil || !campus.IsActive {
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusInternalServerError, "internal_error", "could not validate campus")
			return uuid.Nil, false
		}
		writeError(w, http.StatusNotFound, "not_found", "campus not found")
		return uuid.Nil, false
	}
	return id, true
}
