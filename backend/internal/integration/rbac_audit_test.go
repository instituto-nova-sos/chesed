//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/instituto-nova-sos/chesed/internal/auth"
)

// TestRequireRole_DeniedWritesAuditLog proves that an RBAC 403 denial is
// persisted to audit_log (security Finding 4). Before the fix, denials were
// only emitted to the application log, leaving no forensic record of who was
// refused access to what. A VOLUNTEER hitting an ADMIN-only route must be
// rejected AND leave an ACCESS_DENIED, success=false audit row.
func TestRequireRole_DeniedWritesAuditLog(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	subject := uuid.New()

	req, err := http.NewRequest(http.MethodGet, "/api/v1/test/admin-only", nil)
	require.NoError(t, err)
	req = h.authedRequest(req, func(c *auth.AuthClaims) {
		c.Subject = subject.String()
		c.Roles = []string{"VOLUNTEER"} // lacks ADMIN
	})

	rec := h.doRequest(req)
	require.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())

	var (
		action  string
		success bool
	)
	err = h.pool.QueryRow(ctx,
		`SELECT action_type, success FROM audit_log
		 WHERE user_id = $1 ORDER BY timestamp DESC LIMIT 1`, subject,
	).Scan(&action, &success)
	require.NoError(t, err, "expected an audit_log row for the denied user")
	assert.Equal(t, "ACCESS_DENIED", action)
	assert.False(t, success, "a denial must be recorded with success=false")
}

// TestRequireRole_AllowedWritesNoDenialAudit confirms the audit write is
// scoped to denials: a permitted request must not emit an ACCESS_DENIED row.
func TestRequireRole_AllowedWritesNoDenialAudit(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	subject := uuid.New()

	req, err := http.NewRequest(http.MethodGet, "/api/v1/test/admin-only", nil)
	require.NoError(t, err)
	req = h.authedRequest(req, func(c *auth.AuthClaims) {
		c.Subject = subject.String()
		c.Roles = []string{"ADMIN"}
	})

	rec := h.doRequest(req)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var count int
	err = h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_log WHERE user_id = $1 AND action_type = 'ACCESS_DENIED'`,
		subject,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "an allowed request must not write a denial audit row")
}
