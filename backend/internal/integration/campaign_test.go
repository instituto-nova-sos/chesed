//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/instituto-nova-sos/chesed/internal/auth"
)

func asCoordinator(c *auth.AuthClaims) { c.Roles = []string{"COORDINATOR"} }

// seedPerson inserts a person in the given campus and returns its id. Document
// numbers must be globally unique (uq_person_document), so callers pass one.
func seedPerson(t *testing.T, h *testHarness, campusID uuid.UUID, name, document string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := h.pool.Exec(context.Background(), `
		INSERT INTO person (id, full_name, document_type, document_number, nationality, campus_id)
		VALUES ($1, $2, 'CPF', $3, 'BRA', $4)
	`, id, name, document, campusID)
	require.NoError(t, err)
	return id
}

// seedSecondCampus creates a second campus for cross-campus boundary tests.
func seedSecondCampus(t *testing.T, h *testHarness) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := h.pool.Exec(context.Background(), `
		INSERT INTO campus (id, name, region, is_active)
		VALUES ($1, 'Second Campus', 'BRAZIL', TRUE)
	`, id)
	require.NoError(t, err)
	return id
}

func postJSON(h *testHarness, path string, payload map[string]any, opts ...func(*auth.AuthClaims)) *httptest.ResponseRecorder {
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return h.doRequest(h.authedRequest(req, opts...))
}

// TestCampaignCRUD_HappyPath drives the full documented flow against real
// Postgres: create → list → detail → update, asserting DB state and the
// audit trail at each mutation (S07.1 acceptance criteria).
func TestCampaignCRUD_HappyPath(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	rec := postJSON(h, "/api/v1/campaigns", map[string]any{
		"name":          "March Social Action",
		"campaign_type": "SOCIAL_ACTION",
		"start_date":    "2026-07-10",
		"end_date":      "2026-07-12",
		"location_name": "Community Center",
	}, asCoordinator)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var created struct {
		ID     uuid.UUID `json:"id"`
		Status string    `json:"status"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, "PLANNED", created.Status)

	// DB-level assertion: the row exists, campus-stamped.
	var dbCampus uuid.UUID
	var dbStatus string
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT campus_id, status FROM campaign WHERE id = $1`, created.ID,
	).Scan(&dbCampus, &dbStatus))
	assert.Equal(t, h.campusID, dbCampus)
	assert.Equal(t, "PLANNED", dbStatus)

	// Audit CREATE entry written.
	var auditCount int
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_log WHERE entity_type = 'campaign' AND action_type = 'CREATE'`,
	).Scan(&auditCount))
	assert.Equal(t, 1, auditCount)

	// List includes it.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns", nil)
	rec = h.doRequest(h.authedRequest(req))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "March Social Action")

	// Detail returns the record with an empty team array.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/"+created.ID.String(), nil)
	rec = h.doRequest(h.authedRequest(req))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"team"`)

	// Update transitions status and audits with old values.
	body, _ := json.Marshal(map[string]any{
		"name":          "March Social Action",
		"campaign_type": "SOCIAL_ACTION",
		"start_date":    "2026-07-10",
		"status":        "ACTIVE",
	})
	upReq := httptest.NewRequest(http.MethodPut, "/api/v1/campaigns/"+created.ID.String(), bytes.NewReader(body))
	rec = h.doRequest(h.authedRequest(upReq, asCoordinator))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT status FROM campaign WHERE id = $1`, created.ID,
	).Scan(&dbStatus))
	assert.Equal(t, "ACTIVE", dbStatus)

	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_log WHERE entity_type = 'campaign' AND action_type = 'UPDATE'`,
	).Scan(&auditCount))
	assert.Equal(t, 1, auditCount)
}

// TestCampaign_CampusBoundary proves a campaign in campus B is invisible to a
// campus A caller: absent from the list and 404 on detail/update (rule #4, T3).
func TestCampaign_CampusBoundary(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	secondCampus := seedSecondCampus(t, h)
	foreignID := uuid.New()
	_, err := h.pool.Exec(ctx, `
		INSERT INTO campaign (id, name, campaign_type, start_date, campus_id, status)
		VALUES ($1, 'Foreign Campaign', 'HEALTH', '2026-07-01', $2, 'ACTIVE')
	`, foreignID, secondCampus)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns", nil)
	rec := h.doRequest(h.authedRequest(req))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), "Foreign Campaign")

	req = httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/"+foreignID.String(), nil)
	rec = h.doRequest(h.authedRequest(req))
	assert.Equal(t, http.StatusNotFound, rec.Code)

	body, _ := json.Marshal(map[string]any{
		"name": "Hijack", "campaign_type": "HEALTH", "start_date": "2026-07-01", "status": "CANCELLED",
	})
	upReq := httptest.NewRequest(http.MethodPut, "/api/v1/campaigns/"+foreignID.String(), bytes.NewReader(body))
	rec = h.doRequest(h.authedRequest(upReq, asCoordinator))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// TestCampaign_RBACBoundary proves VOLUNTEER cannot create campaigns
// (Coordinator+ per docs/16 permissions matrix) and the denial is audited.
func TestCampaign_RBACBoundary(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	rec := postJSON(h, "/api/v1/campaigns", map[string]any{
		"name": "Rogue", "campaign_type": "OTHER", "start_date": "2026-07-01",
	}) // default claims: VOLUNTEER
	require.Equal(t, http.StatusForbidden, rec.Code)

	var count int
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM campaign`).Scan(&count))
	assert.Zero(t, count, "no campaign row may exist after a denied create")

	var auditCount int
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_log WHERE action_type = 'ACCESS_DENIED'`,
	).Scan(&auditCount))
	assert.Equal(t, 1, auditCount, "RBAC denial must be audited")
}

// TestCampaign_Validation exercises the documented 400 contract: bad enum and
// inverted dates are rejected without touching the database.
func TestCampaign_Validation(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	rec := postJSON(h, "/api/v1/campaigns", map[string]any{
		"name": "Bad Type", "campaign_type": "PARTY", "start_date": "2026-07-10",
	}, asCoordinator)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = postJSON(h, "/api/v1/campaigns", map[string]any{
		"name": "Bad Dates", "campaign_type": "OTHER",
		"start_date": "2026-07-10", "end_date": "2026-07-01",
	}, asCoordinator)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var count int
	require.NoError(t, h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM campaign`).Scan(&count))
	assert.Zero(t, count)
}

// TestCampaignTeam_Flow covers S07.2 end to end: add → duplicate 409 →
// cross-campus person 400 → remove 204 → remove again 404, with the
// uq_campaign_person constraint enforced by real Postgres.
func TestCampaignTeam_Flow(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	rec := postJSON(h, "/api/v1/campaigns", map[string]any{
		"name": "Team Campaign", "campaign_type": "COMMUNITY", "start_date": "2026-07-10",
	}, asCoordinator)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	memberID := seedPerson(t, h, h.campusID, "Maria Silva", "11122233344")
	teamPath := fmt.Sprintf("/api/v1/campaigns/%s/team", created.ID)

	rec = postJSON(h, teamPath, map[string]any{
		"person_id": memberID.String(), "role_in_campaign": "VOLUNTEER",
	}, asCoordinator)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "Maria Silva")

	// Duplicate assignment → 409 and still a single row.
	rec = postJSON(h, teamPath, map[string]any{
		"person_id": memberID.String(), "role_in_campaign": "SUPPORT",
	}, asCoordinator)
	assert.Equal(t, http.StatusConflict, rec.Code)
	var rows int
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM campaign_team WHERE campaign_id = $1`, created.ID,
	).Scan(&rows))
	assert.Equal(t, 1, rows)

	// Person from another campus → generic 400, nothing persisted.
	secondCampus := seedSecondCampus(t, h)
	foreignPerson := seedPerson(t, h, secondCampus, "Foreign Person", "55566677788")
	rec = postJSON(h, teamPath, map[string]any{
		"person_id": foreignPerson.String(), "role_in_campaign": "VOLUNTEER",
	}, asCoordinator)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.NotContains(t, rec.Body.String(), "Foreign Person",
		"error must not disclose foreign-campus existence")
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM campaign_team WHERE campaign_id = $1`, created.ID,
	).Scan(&rows))
	assert.Equal(t, 1, rows)

	// Detail shows the team member.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/"+created.ID.String(), nil)
	rec = h.doRequest(h.authedRequest(req))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Maria Silva")

	// Remove → 204 and the row is gone.
	delReq := httptest.NewRequest(http.MethodDelete,
		fmt.Sprintf("%s/%s", teamPath, memberID), nil)
	rec = h.doRequest(h.authedRequest(delReq, asCoordinator))
	assert.Equal(t, http.StatusNoContent, rec.Code)
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM campaign_team WHERE campaign_id = $1`, created.ID,
	).Scan(&rows))
	assert.Zero(t, rows)

	// Removing again → 404.
	delReq = httptest.NewRequest(http.MethodDelete,
		fmt.Sprintf("%s/%s", teamPath, memberID), nil)
	rec = h.doRequest(h.authedRequest(delReq, asCoordinator))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
