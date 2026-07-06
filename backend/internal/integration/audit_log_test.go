//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/handler"
	"github.com/instituto-nova-sos/chesed/internal/middleware"
	"github.com/instituto-nova-sos/chesed/internal/repository"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

// auditListResponse mirrors domain.AuditLogListResult for decoding.
type auditListResponse struct {
	Data []struct {
		ID          uuid.UUID       `json:"id"`
		UserEmail   *string         `json:"user_email"`
		ActionType  string          `json:"action_type"`
		EntityType  string          `json:"entity_type"`
		Description *string         `json:"description"`
		OldValues   json.RawMessage `json:"old_values"`
		NewValues   json.RawMessage `json:"new_values"`
		Timestamp   time.Time       `json:"timestamp"`
	} `json:"data"`
	Pagination struct {
		Page    int `json:"page"`
		PerPage int `json:"per_page"`
		Total   int `json:"total"`
	} `json:"pagination"`
}

// auditRouter mounts the audit viewer route against the harness pool. audit_log
// is not under RLS, so the read runs directly on the pool (as in production).
// Mirrors registerAuditRoutes in cmd/server/main.go (ADMIN only).
func auditRouter(h *testHarness) chi.Router {
	auditRepo := repository.NewAuditRepository(h.pool)
	auditSvc := service.NewAuditService(auditRepo)
	auditH := handler.NewAuditHandler(service.NewAuditReadService(auditRepo))

	adminOnly := middleware.RequireRole(auditSvc, "ADMIN")
	r := chi.NewRouter()
	r.With(adminOnly).Get("/api/v1/audit/logs", auditH.List)
	return r
}

// seedAudit inserts an audit_log row directly for a given campus/actor.
func seedAudit(t *testing.T, h *testHarness, campusID *uuid.UUID, subject *uuid.UUID, action, entityType string, ts time.Time) {
	t.Helper()
	_, err := h.pool.Exec(context.Background(),
		`INSERT INTO audit_log (id, user_id, action_type, entity_type, campus_id, success, timestamp)
		 VALUES (gen_random_uuid(), $1, $2, $3, $4, true, $5)`,
		subject, action, entityType, campusID, ts)
	require.NoError(t, err)
}

func TestAuditLogViewer(t *testing.T) {
	t.Run("ADMIN sees only own-campus rows with resolved user_email", func(t *testing.T) {
		h := freshHarness(t)
		ctx := context.Background()
		router := auditRouter(h)

		// Actor with a resolvable app_user (email via keycloak_subject_id join).
		subject := uuid.New()
		_, err := h.pool.Exec(ctx,
			`INSERT INTO app_user (id, email, keycloak_subject_id, access_profile, campus_id)
			 VALUES (gen_random_uuid(), $1, $2, 'ADMIN', $3)`,
			"admin@chesed.test", subject.String(), h.campusID)
		require.NoError(t, err)

		now := time.Now()
		seedAudit(t, h, &h.campusID, &subject, "UPDATE", "person", now)
		seedAudit(t, h, &h.campusID, &subject, "CREATE", "consent", now.Add(-time.Hour))

		// Foreign campus row (must NOT appear).
		foreignCampus := seedSecondCampus(t, h)
		foreignSubject := uuid.New()
		seedAudit(t, h, &foreignCampus, &foreignSubject, "UPDATE", "person", now)

		// System row with no campus (must NOT appear).
		seedAudit(t, h, nil, &foreignSubject, "LOGIN", "app_user", now)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/logs", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, h.authedRequest(req, asAdmin))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		var body auditListResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, 2, body.Pagination.Total, "only the two own-campus rows are visible")
		require.Len(t, body.Data, 2)
		// Newest first.
		assert.Equal(t, "UPDATE", body.Data[0].ActionType)
		require.NotNil(t, body.Data[0].UserEmail)
		assert.Equal(t, "admin@chesed.test", *body.Data[0].UserEmail)
	})

	t.Run("filters narrow the result set", func(t *testing.T) {
		h := freshHarness(t)
		router := auditRouter(h)
		subject := uuid.New()
		now := time.Now()
		seedAudit(t, h, &h.campusID, &subject, "UPDATE", "person", now)
		seedAudit(t, h, &h.campusID, &subject, "CREATE", "consent", now)
		seedAudit(t, h, &h.campusID, &subject, "DELETE", "person", now)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/logs?entity_type=person&action_type=UPDATE", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, h.authedRequest(req, asAdmin))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		var body auditListResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, 1, body.Pagination.Total)
		require.Len(t, body.Data, 1)
		assert.Equal(t, "UPDATE", body.Data[0].ActionType)
		assert.Equal(t, "person", body.Data[0].EntityType)
	})

	t.Run("pagination reports the campus total", func(t *testing.T) {
		h := freshHarness(t)
		router := auditRouter(h)
		subject := uuid.New()
		now := time.Now()
		for i := 0; i < 5; i++ {
			seedAudit(t, h, &h.campusID, &subject, "UPDATE", "person", now.Add(-time.Duration(i)*time.Minute))
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/logs?page=1&per_page=2", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, h.authedRequest(req, asAdmin))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		var body auditListResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, 5, body.Pagination.Total)
		assert.Equal(t, 2, body.Pagination.PerPage)
		assert.Len(t, body.Data, 2, "page size respected")
	})

	t.Run("non-admin is forbidden and the denial is audited", func(t *testing.T) {
		h := freshHarness(t)
		ctx := context.Background()
		router := auditRouter(h)
		subject := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/logs", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, h.authedRequest(req, func(c *auth.AuthClaims) {
			c.Subject = subject.String()
			c.Roles = []string{"COORDINATOR"} // lacks ADMIN
		}))
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

		var action string
		err := h.pool.QueryRow(ctx,
			`SELECT action_type FROM audit_log WHERE user_id = $1 ORDER BY timestamp DESC LIMIT 1`, subject,
		).Scan(&action)
		require.NoError(t, err)
		assert.Equal(t, "ACCESS_DENIED", action)
	})
}
