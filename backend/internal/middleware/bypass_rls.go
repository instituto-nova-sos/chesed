package middleware

import (
	"net/http"

	"github.com/instituto-nova-sos/chesed/internal/repository"
)

// BypassRLS installs the admin (owner) Querier into the request context so
// repositories run their queries on the RLS-bypassing connection instead of the
// non-owner app pool. It is mounted on the pre-campus routes (self-registration
// and onboarding's cross-campus email lookup) that legitimately operate outside
// any single campus: those flows must read/write the person table before a
// campus GUC can be established, so they cannot be subject to per-request RLS.
//
// No transaction is opened — the owner role bypasses RLS, so no GUC is needed.
func BypassRLS(admin repository.Querier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := repository.NewQuerierContext(r.Context(), admin)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
