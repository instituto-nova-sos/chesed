//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// backdatePerson forces a person's updated_at into the past so the retention
// sweep considers it expired.
func backdatePerson(t *testing.T, h *testHarness, personID uuid.UUID, interval string) {
	t.Helper()
	_, err := h.pool.Exec(context.Background(),
		`UPDATE person SET updated_at = NOW() - $2::interval WHERE id = $1`, personID, interval)
	require.NoError(t, err)
}

// TestRetention_AnonymizesExpiredOnly proves the admin retention sweep
// anonymizes persons past the retention window and leaves recent ones intact
// (S11.7).
func TestRetention_AnonymizesExpiredOnly(t *testing.T) {
	h := freshHarness(t)

	oldID := seedPerson(t, h, h.campusID, "Old Person", "52998224725")
	recentID := seedPerson(t, h, h.campusID, "Recent Person", "15350946056")
	backdatePerson(t, h, oldID, "6 years")

	rec := h.doRequest(h.authedRequest(
		httptest.NewRequest(http.MethodPost, "/api/v1/admin/retention/run", nil),
		asAdmin))
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var summary struct {
		Scanned    int `json:"scanned"`
		Anonymized int `json:"anonymized"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &summary))
	assert.Equal(t, 1, summary.Scanned)
	assert.Equal(t, 1, summary.Anonymized)

	// Old person is anonymized.
	var oldName string
	var oldAnon *string
	require.NoError(t, h.pool.QueryRow(context.Background(),
		`SELECT full_name, anonymized_at::text FROM person WHERE id = $1`, oldID,
	).Scan(&oldName, &oldAnon))
	assert.Equal(t, "[ANONYMIZED]", oldName)
	require.NotNil(t, oldAnon)

	// Recent person is untouched.
	var recentName string
	var recentAnon *string
	require.NoError(t, h.pool.QueryRow(context.Background(),
		`SELECT full_name, anonymized_at::text FROM person WHERE id = $1`, recentID,
	).Scan(&recentName, &recentAnon))
	assert.Equal(t, "Recent Person", recentName)
	assert.Nil(t, recentAnon)

	// Second run is idempotent — nothing left to anonymize.
	rec2 := h.doRequest(h.authedRequest(
		httptest.NewRequest(http.MethodPost, "/api/v1/admin/retention/run", nil),
		asAdmin))
	require.Equal(t, http.StatusOK, rec2.Code)
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &summary))
	assert.Equal(t, 0, summary.Anonymized, "already-anonymized persons must be skipped")
}

// TestRetention_RequiresAdmin proves a non-admin cannot trigger the sweep.
func TestRetention_RequiresAdmin(t *testing.T) {
	h := freshHarness(t)

	rec := h.doRequest(h.authedRequest(
		httptest.NewRequest(http.MethodPost, "/api/v1/admin/retention/run", nil),
		asCoordinator))
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
