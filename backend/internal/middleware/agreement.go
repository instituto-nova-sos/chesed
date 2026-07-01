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
			if agreementSatisfied(r, claims, agreementChecker, roleFinder) {
				next.ServeHTTP(w, r)
				return
			}
			slog.InfoContext(r.Context(), "middleware.RequireAgreement: agreement not accepted",
				"person_id", claims.PersonID, "path", r.URL.Path,
			)
			writeError(w, http.StatusForbidden, "agreement_required", "volunteer agreement must be accepted before accessing the platform")
		})
	}
}

// agreementSatisfied reports whether the request may proceed: true when the user
// has no linked person, is not an active volunteer, has accepted the agreement,
// or when an infrastructure error occurs (fail-open).
func agreementSatisfied(r *http.Request, claims auth.AuthClaims, agreementChecker AgreementChecker, roleFinder RoleFinder) bool {
	// No person linked yet — ProfileCompletionGuard handles this.
	if claims.PersonID == uuid.Nil {
		return true
	}

	roles, err := roleFinder.FindByPersonID(r.Context(), claims.PersonID)
	if err != nil {
		slog.ErrorContext(r.Context(), "middleware.RequireAgreement: find roles failed",
			"error", err.Error(), "person_id", claims.PersonID,
		)
		return true // fail open on infra errors
	}
	if !hasActiveVolunteerRole(roles) {
		return true
	}

	accepted, err := agreementChecker.HasAcceptedAgreement(r.Context(), claims.PersonID)
	if err != nil {
		slog.ErrorContext(r.Context(), "middleware.RequireAgreement: check agreement failed",
			"error", err.Error(), "person_id", claims.PersonID,
		)
		return true // fail open on infra errors
	}
	return accepted
}

func hasActiveVolunteerRole(roles []domain.PersonRole) bool {
	for _, role := range roles {
		if role.RoleType == domain.RoleVolunteer && role.IsActive {
			return true
		}
	}
	return false
}
