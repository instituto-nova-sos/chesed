//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCampusTimezone_DefaultApplied verifies migration 000030 gives new campuses
// the America/Sao_Paulo default when the timezone is omitted, and that an
// explicit IANA timezone round-trips through the repository (Sprint 9.4).
func TestCampusTimezone_DefaultApplied(t *testing.T) {
	h := freshHarness(t)
	defer h.Close()

	repo := repository.NewCampusRepository(h.pool)
	ctx := context.Background()

	t.Run("omitted timezone defaults to America/Sao_Paulo", func(t *testing.T) {
		created, err := repo.Create(ctx, domain.Campus{
			ID:       uuid.New(),
			Name:     "Default TZ Campus",
			Region:   "BRAZIL",
			Country:  "BRA",
			IsActive: true,
		})
		require.NoError(t, err)
		assert.Equal(t, "America/Sao_Paulo", created.Timezone)

		found, err := repo.FindByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, "America/Sao_Paulo", found.Timezone)
	})

	t.Run("explicit timezone round-trips", func(t *testing.T) {
		created, err := repo.Create(ctx, domain.Campus{
			ID:       uuid.New(),
			Name:     "Lisbon Campus",
			Region:   "EUROPE",
			Country:  "PRT",
			Timezone: "Europe/Lisbon",
			IsActive: true,
		})
		require.NoError(t, err)
		assert.Equal(t, "Europe/Lisbon", created.Timezone)

		found, err := repo.FindByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, "Europe/Lisbon", found.Timezone)
	})
}
