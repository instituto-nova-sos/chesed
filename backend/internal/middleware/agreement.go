package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// AgreementChecker checks if a person has an accepted volunteer agreement.
type AgreementChecker interface {
	HasAcceptedAgreement(ctx context.Context, personID uuid.UUID) (bool, error)
}

// RoleFinder looks up roles for a person.
type RoleFinder interface {
	FindByPersonID(ctx context.Context, personID uuid.UUID) ([]domain.PersonRole, error)
}

// RequireAgreement returns middleware that blocks access for volunteers
// who have not accepted their volunteer agreement.
// Must be applied AFTER OIDCAuth and AutoProvision.
func RequireAgreement(agreementChecker AgreementChecker, roleFinder RoleFinder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := auth.ClaimsFromContext(r.Context())

			// No person linked yet — ProfileCompletionGuard handles this
			if claims.PersonID == uuid.Nil {
				next.ServeHTTP(w, r)
				return
			}

			// Check if person has a VOLUNTEER role
			roles, err := roleFinder.FindByPersonID(r.Context(), claims.PersonID)
			if err != nil {
				slog.ErrorContext(r.Context(), "middleware.RequireAgreement: find roles failed",
					"error", err.Error(), "person_id", claims.PersonID,
				)
				// Fail open: let the request through rather than blocking on infra errors
				next.ServeHTTP(w, r)
				return
			}

			hasVolunteerRole := false
			for _, role := range roles {
				if role.RoleType == domain.RoleVolunteer && role.IsActive {
					hasVolunteerRole = true
					break
				}
			}

			// Not a volunteer — no agreement needed
			if !hasVolunteerRole {
				next.ServeHTTP(w, r)
				return
			}

			// Check if volunteer has accepted agreement
			accepted, err := agreementChecker.HasAcceptedAgreement(r.Context(), claims.PersonID)
			if err != nil {
				slog.ErrorContext(r.Context(), "middleware.RequireAgreement: check agreement failed",
					"error", err.Error(), "person_id", claims.PersonID,
				)
				next.ServeHTTP(w, r)
				return
			}

			if !accepted {
				slog.InfoContext(r.Context(), "middleware.RequireAgreement: agreement not accepted",
					"person_id", claims.PersonID,
					"path", r.URL.Path,
				)
				writeError(w, http.StatusForbidden, "agreement_required", "volunteer agreement must be accepted before accessing the platform")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
