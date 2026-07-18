package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

// AutoProvision creates middleware that ensures an app_user record exists
// and resolves the campus_id from the database.
// Runs after OIDCAuth. Calls UserService.EnsureUser with claims from context.
// Campus is resolved from app_user.campus_id (not from JWT).
func AutoProvision(userSvc *service.UserService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := auth.ClaimsFromContext(r.Context())
			if claims.Subject == "" {
				writeError(w, http.StatusUnauthorized, "unauthorized", "missing authentication claims")
				return
			}

			ip := extractIP(r)
			userAgent := r.UserAgent()

			appUser, err := userSvc.EnsureUser(r.Context(), claims, ip, userAgent)
			if err != nil {
				slog.ErrorContext(r.Context(), "middleware.AutoProvision: failed",
					"error", err.Error(),
					"subject", claims.Subject,
				)
				writeError(w, http.StatusInternalServerError, "internal_error", "user provisioning failed")
				return
			}

			// Resolve campus from app_user record in DB
			if appUser.CampusID == nil || *appUser.CampusID == uuid.Nil {
				slog.WarnContext(r.Context(), "middleware.AutoProvision: no campus assigned",
					"subject", claims.Subject,
				)
				writeError(w, http.StatusForbidden, "forbidden", "missing campus assignment")
				return
			}

			// Enrich auth context with resolved campus and local app_user id.
			// AppUserID (not Subject) is what FK columns such as
			// volunteer_agreement.accepted_by_user reference.
			claims.CampusID = *appUser.CampusID
			claims.AppUserID = appUser.ID

			// Resolve person_id from the DB (app_user), same as campus above.
			// Self-registration links app_user.person_id but cannot update the
			// Keycloak token, so the JWT person_id claim is unreliable for a
			// user who registered after their token was issued. The DB is the
			// source of truth for the person link.
			if appUser.PersonID != nil {
				claims.PersonID = *appUser.PersonID
			}
			ctx := auth.NewContext(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractIP(r *http.Request) string {
	// Use RemoteAddr as the base — it's set by the Go HTTP server and not spoofable.
	// Only trust X-Forwarded-For/X-Real-IP if behind a trusted reverse proxy.
	// In dev (no proxy), these headers should not be present.
	ip := r.RemoteAddr

	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// X-Forwarded-For can contain multiple IPs: "client, proxy1, proxy2".
		// Take the first (leftmost) which is the original client IP.
		if idx := strings.Index(forwarded, ","); idx != -1 {
			ip = strings.TrimSpace(forwarded[:idx])
		} else {
			ip = strings.TrimSpace(forwarded)
		}
	} else if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		ip = strings.TrimSpace(realIP)
	}

	// Strip port from host:port (e.g. "192.168.1.1:8080" → "192.168.1.1").
	// PostgreSQL inet type does not accept port numbers.
	host, _, err := net.SplitHostPort(ip)
	if err == nil {
		ip = host
	}

	// Validate the result is a valid IP address.
	// If not (e.g. spoofed header with garbage), fall back to empty string
	// rather than storing invalid data in the audit log.
	if net.ParseIP(ip) == nil {
		return ""
	}
	return ip
}
