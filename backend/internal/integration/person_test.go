//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/instituto-nova-sos/chesed/internal/repository"
)

// TestCheckDuplicate_CampusScoped proves the duplicate-detection query honours
// the campus boundary (CLAUDE.md rule #4, threat model T3). A caller in campus A
// must never learn that a person with a given document exists in campus B — the
// query is the layer where the multi-tenant boundary is enforced, so we exercise
// it directly against real Postgres.
func TestCheckDuplicate_CampusScoped(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()
	repo := repository.NewPersonRepository(h.pool)

	// A second campus with a person carrying a specific document.
	secondCampusID := uuid.New()
	_, err := h.pool.Exec(ctx, `
		INSERT INTO campus (id, name, region, is_active)
		VALUES ($1, 'Second Campus', 'BRAZIL', TRUE)
	`, secondCampusID)
	require.NoError(t, err)

	otherCampusPerson := uuid.New()
	_, err = h.pool.Exec(ctx, `
		INSERT INTO person (id, full_name, document_type, document_number, nationality, campus_id)
		VALUES ($1, 'Campus B Person', 'CPF', '99988877766', 'BRA', $2)
	`, otherCampusPerson, secondCampusID)
	require.NoError(t, err)

	// A caller in campus A checking that same document must get NO match: the
	// person lives in another campus and must remain invisible.
	result, err := repo.CheckDuplicate(ctx, "CPF", "99988877766", h.campusID)
	require.NoError(t, err)
	assert.False(t, result.HasDuplicates,
		"a document held only in campus B must not be reported to a campus A caller")
	assert.Empty(t, result.Matches)
}

// TestCheckDuplicate_SameCampusMatch proves the query still finds a genuine
// same-campus duplicate — the campus filter must not blind it to its own campus.
func TestCheckDuplicate_SameCampusMatch(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()
	repo := repository.NewPersonRepository(h.pool)

	personID := uuid.New()
	_, err := h.pool.Exec(ctx, `
		INSERT INTO person (id, full_name, document_type, document_number, nationality, campus_id)
		VALUES ($1, 'Same Campus Person', 'CPF', '12312312399', 'BRA', $2)
	`, personID, h.campusID)
	require.NoError(t, err)

	result, err := repo.CheckDuplicate(ctx, "CPF", "12312312399", h.campusID)
	require.NoError(t, err)
	require.True(t, result.HasDuplicates, "a same-campus document match must be detected")
	require.Len(t, result.Matches, 1)
	assert.Equal(t, personID, result.Matches[0].ID)
	assert.Equal(t, "exact_document", result.Matches[0].MatchType)
}
