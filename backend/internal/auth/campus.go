package auth

import (
	"fmt"

	"github.com/google/uuid"
)

// ResolveCampusID returns the effective campus_id for a query.
// ADMIN users may override with a different campus_id for cross-campus queries.
// Non-ADMIN users always get their own campus_id from token claims.
func ResolveCampusID(claims AuthClaims, overrideCampusID *uuid.UUID) (uuid.UUID, error) {
	if claims.CampusID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("auth.ResolveCampusID: missing campus_id in claims")
	}

	if overrideCampusID != nil && *overrideCampusID != claims.CampusID {
		if !hasAdminRole(claims.Roles) {
			return uuid.Nil, fmt.Errorf("auth.ResolveCampusID: cross-campus access requires ADMIN role")
		}
		return *overrideCampusID, nil
	}

	return claims.CampusID, nil
}

func hasAdminRole(roles []string) bool {
	for _, r := range roles {
		if r == "ADMIN" {
			return true
		}
	}
	return false
}
