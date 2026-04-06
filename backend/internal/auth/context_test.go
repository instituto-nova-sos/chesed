package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewContextAndClaimsFromContext(t *testing.T) {
	campusID := uuid.New()
	personID := uuid.New()

	claims := AuthClaims{
		Subject:  "keycloak-subject-id",
		Email:    "user@example.com",
		Roles:    []string{"ADMIN", "COORDINATOR"},
		CampusID: campusID,
		PersonID: personID,
	}

	ctx := NewContext(context.Background(), claims)
	got := ClaimsFromContext(ctx)

	assert.Equal(t, claims.Subject, got.Subject)
	assert.Equal(t, claims.Email, got.Email)
	assert.Equal(t, claims.Roles, got.Roles)
	assert.Equal(t, claims.CampusID, got.CampusID)
	assert.Equal(t, claims.PersonID, got.PersonID)
}

func TestClaimsFromContext_EmptyContext(t *testing.T) {
	got := ClaimsFromContext(context.Background())

	assert.Equal(t, "", got.Subject)
	assert.Equal(t, "", got.Email)
	assert.Nil(t, got.Roles)
	assert.Equal(t, uuid.Nil, got.CampusID)
	assert.Equal(t, uuid.Nil, got.PersonID)
}

func TestCampusIDFromContext(t *testing.T) {
	t.Run("with claims", func(t *testing.T) {
		campusID := uuid.New()
		ctx := NewContext(context.Background(), AuthClaims{CampusID: campusID})
		assert.Equal(t, campusID, CampusIDFromContext(ctx))
	})

	t.Run("empty context", func(t *testing.T) {
		assert.Equal(t, uuid.Nil, CampusIDFromContext(context.Background()))
	})
}

func TestUserIDFromContext(t *testing.T) {
	t.Run("with claims", func(t *testing.T) {
		ctx := NewContext(context.Background(), AuthClaims{Subject: "user-123"})
		assert.Equal(t, "user-123", UserIDFromContext(ctx))
	})

	t.Run("empty context", func(t *testing.T) {
		assert.Equal(t, "", UserIDFromContext(context.Background()))
	})
}
