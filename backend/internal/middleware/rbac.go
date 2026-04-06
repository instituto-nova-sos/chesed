package middleware

import (
	"log/slog"
	"net/http"

	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// RequireRole returns middleware that checks if the authenticated user
// has at least one of the specified roles. Must be applied AFTER OIDCAuth.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := auth.ClaimsFromContext(r.Context())

			if !domain.HasRole(claims.Roles, roles...) {
				slog.WarnContext(r.Context(), "middleware.RequireRole: insufficient permissions",
					"subject", claims.Subject,
					"path", r.URL.Path,
					"method", r.Method,
					"required_roles", roles,
					"user_roles", claims.Roles,
				)
				writeError(w, http.StatusForbidden, "forbidden", "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
