package auth

import (
	"context"

	"github.com/google/uuid"
)

type contextKey int

const claimsKey contextKey = iota

// AuthClaims represents validated claims from a Keycloak JWT.
//
//nolint:revive // domain-meaningful name used across ~25 files; cross-package rename deferred
type AuthClaims struct {
	Subject       string
	Email         string
	EmailVerified bool
	Roles         []string
	CampusID      uuid.UUID
	PersonID      uuid.UUID
}

// NewContext stores AuthClaims in the context.
func NewContext(ctx context.Context, claims AuthClaims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// ClaimsFromContext extracts AuthClaims from context.
// Returns zero-value AuthClaims if not present.
func ClaimsFromContext(ctx context.Context) AuthClaims {
	claims, ok := ctx.Value(claimsKey).(AuthClaims)
	if !ok {
		return AuthClaims{}
	}
	return claims
}

// CampusIDFromContext returns the campus_id from auth claims.
func CampusIDFromContext(ctx context.Context) uuid.UUID {
	return ClaimsFromContext(ctx).CampusID
}

// UserIDFromContext returns the Keycloak subject (user ID) from auth claims.
func UserIDFromContext(ctx context.Context) string {
	return ClaimsFromContext(ctx).Subject
}
