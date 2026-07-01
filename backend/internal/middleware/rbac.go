package middleware

import (
	"log/slog"
	"net/http"

	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

// RequireRole returns middleware that checks if the authenticated user has at
// least one of the specified roles. Must be applied AFTER OIDCAuth. Every 403
// denial is recorded to audit_log via auditSvc (security Finding 4) so access
// refusals leave a forensic trail, not just an application-log line.
func RequireRole(auditSvc *service.AuditService, roles ...string) func(http.Handler) http.Handler {
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
				auditDenial(r, auditSvc)
				writeError(w, http.StatusForbidden, "forbidden", "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// auditDenial records an RBAC 403 as an ACCESS_DENIED audit entry. The actor
// and campus are auto-filled from the request context by the audit service. A
// write failure is logged but never fails the request — the caller is already
// being rejected.
func auditDenial(r *http.Request, auditSvc *service.AuditService) {
	if auditSvc == nil {
		return
	}
	if err := auditSvc.LogAction(r.Context(), service.AuditParams{
		ActionType:  "ACCESS_DENIED",
		EntityType:  "rbac",
		Module:      "auth",
		Description: r.Method + " " + r.URL.Path,
		IPAddress:   extractIP(r),
		UserAgent:   r.UserAgent(),
		Success:     false,
	}); err != nil {
		slog.ErrorContext(r.Context(), "middleware.RequireRole: audit write failed",
			"error", err.Error(),
			"path", r.URL.Path,
		)
	}
}
