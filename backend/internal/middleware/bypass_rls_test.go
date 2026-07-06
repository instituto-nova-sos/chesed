package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/instituto-nova-sos/chesed/internal/repository"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBypassRLS_InstallsAdminQuerier proves the middleware installs the admin
// (owner, RLS-bypassing) Querier into the request context, so pre-campus routes
// (self-register, onboarding global lookup) run without a campus GUC and are not
// fail-closed by RLS.
func TestBypassRLS_InstallsAdminQuerier(t *testing.T) {
	admin, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer admin.Close()

	mw := BypassRLS(admin)

	var installed repository.Querier
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		installed = repository.QuerierFrom(r.Context(), nil)
	})).ServeHTTP(rec, req)

	assert.Same(t, repository.Querier(admin), installed,
		"the admin Querier must be installed so RLS-subject reads bypass RLS on pre-campus routes")
}
