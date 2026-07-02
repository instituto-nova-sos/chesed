//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/instituto-nova-sos/chesed/internal/auth"
)

func asProfessional(c *auth.AuthClaims) { c.Roles = []string{"PROFESSIONAL"} }

// seedCampaign inserts a campaign in the given campus and returns its id.
func seedCampaign(t *testing.T, h *testHarness, campusID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := h.pool.Exec(context.Background(), `
		INSERT INTO campaign (id, name, campaign_type, start_date, campus_id, status)
		VALUES ($1, $2, 'SOCIAL_ACTION', '2026-07-10', $3, 'ACTIVE')
	`, id, name, campusID)
	require.NoError(t, err)
	return id
}

// TestTriageCampaignLink_OwnCampus proves POST /triages persists a same-campus
// campaign link (S07.3).
func TestTriageCampaignLink_OwnCampus(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	campaignID := seedCampaign(t, h, h.campusID, "Linked Campaign")
	personID := seedPerson(t, h, h.campusID, "Triaged Person", "10120230340")

	rec := postCampaignJSON(h, "/api/v1/triages", map[string]any{
		"person_id":      personID.String(),
		"campaign_id":    campaignID.String(),
		"main_complaint": "needs assistance",
	}, asProfessional)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var linked uuid.UUID
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT campaign_id FROM triage WHERE person_id = $1`, personID,
	).Scan(&linked))
	assert.Equal(t, campaignID, linked)
}

// TestTriageCampaignLink_CrossCampus proves a foreign-campus campaign
// reference is rejected with a generic 400 and persists nothing (T3).
func TestTriageCampaignLink_CrossCampus(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	secondCampus := seedSecondCampus(t, h)
	foreignCampaign := seedCampaign(t, h, secondCampus, "Foreign Campaign")
	personID := seedPerson(t, h, h.campusID, "Triaged Person", "10120230341")

	rec := postCampaignJSON(h, "/api/v1/triages", map[string]any{
		"person_id":      personID.String(),
		"campaign_id":    foreignCampaign.String(),
		"main_complaint": "needs assistance",
	}, asProfessional)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "Foreign Campaign")

	var count int
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM triage WHERE person_id = $1`, personID,
	).Scan(&count))
	assert.Zero(t, count, "no triage row may persist after a rejected campaign link")
}

// TestAttendanceCampaignLink covers both directions for attendances: a
// same-campus link persists; a foreign link is rejected without a row.
func TestAttendanceCampaignLink(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	campaignID := seedCampaign(t, h, h.campusID, "Attendance Campaign")
	personID := seedPerson(t, h, h.campusID, "Attended Person", "10120230342")
	professionalID := seedPerson(t, h, h.campusID, "Professional", "10120230343")

	var serviceTypeID uuid.UUID
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT id FROM service_type LIMIT 1`).Scan(&serviceTypeID))

	rec := postCampaignJSON(h, "/api/v1/attendances", map[string]any{
		"person_id":       personID.String(),
		"campaign_id":     campaignID.String(),
		"service_type_id": serviceTypeID.String(),
		"professional_id": professionalID.String(),
	}, asProfessional)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var linked uuid.UUID
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT campaign_id FROM attendance WHERE person_id = $1`, personID,
	).Scan(&linked))
	assert.Equal(t, campaignID, linked)

	// Foreign campaign → 400 and no second row.
	secondCampus := seedSecondCampus(t, h)
	foreignCampaign := seedCampaign(t, h, secondCampus, "Foreign Campaign")
	rec = postCampaignJSON(h, "/api/v1/attendances", map[string]any{
		"person_id":       personID.String(),
		"campaign_id":     foreignCampaign.String(),
		"service_type_id": serviceTypeID.String(),
		"professional_id": professionalID.String(),
	}, asProfessional)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var count int
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM attendance WHERE person_id = $1`, personID,
	).Scan(&count))
	assert.Equal(t, 1, count)
}

// TestCampaignForeignKey proves migration 000023 enforces referential
// integrity on the previously bare campaign_id columns.
func TestCampaignForeignKey(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	personID := seedPerson(t, h, h.campusID, "FK Person", "10120230344")
	_, err := h.pool.Exec(ctx, `
		INSERT INTO triage (person_id, campus_id, main_complaint, triaged_by, campaign_id)
		VALUES ($1, $2, 'x', $3, $4)
	`, personID, h.campusID, uuid.New(), uuid.New())
	require.Error(t, err, "a dangling campaign_id must violate the new foreign key")
	assert.Contains(t, err.Error(), "23503")
}

// TestCampaignMetrics_Report covers S07.4: seeded links roll up into the
// campus-scoped metrics endpoint; foreign campaigns 404; volunteers 403.
func TestCampaignMetrics_Report(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	campaignID := seedCampaign(t, h, h.campusID, "Metrics Campaign")
	personID := seedPerson(t, h, h.campusID, "Metrics Person", "10120230345")
	memberID := seedPerson(t, h, h.campusID, "Team Member", "10120230346")

	var serviceTypeID uuid.UUID
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT id FROM service_type LIMIT 1`).Scan(&serviceTypeID))

	_, err := h.pool.Exec(ctx, `
		INSERT INTO triage (person_id, campus_id, main_complaint, triaged_by, campaign_id)
		VALUES ($1, $2, 'metrics triage', $3, $4)
	`, personID, h.campusID, uuid.New(), campaignID)
	require.NoError(t, err)

	_, err = h.pool.Exec(ctx, `
		INSERT INTO attendance (person_id, campus_id, service_type_id, professional_id, status, campaign_id)
		VALUES ($1, $2, $3, $4, 'COMPLETED', $5), ($1, $2, $3, $4, 'SCHEDULED', $5)
	`, personID, h.campusID, serviceTypeID, memberID, campaignID)
	require.NoError(t, err)

	_, err = h.pool.Exec(ctx, `
		INSERT INTO campaign_team (campaign_id, person_id, role_in_campaign)
		VALUES ($1, $2, 'VOLUNTEER')
	`, campaignID, memberID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/campaigns/"+campaignID.String(), nil)
	rec := h.doRequest(h.authedRequest(req, asCoordinator))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	body := rec.Body.String()
	assert.Contains(t, body, `"triage_count":1`)
	assert.Contains(t, body, `"attendance_total":2`)
	assert.Contains(t, body, `"COMPLETED":1`)
	assert.Contains(t, body, `"team_size":1`)

	// Foreign campaign → 404 (no existence disclosure).
	secondCampus := seedSecondCampus(t, h)
	foreignCampaign := seedCampaign(t, h, secondCampus, "Foreign Metrics")
	req = httptest.NewRequest(http.MethodGet, "/api/v1/reports/campaigns/"+foreignCampaign.String(), nil)
	rec = h.doRequest(h.authedRequest(req, asCoordinator))
	assert.Equal(t, http.StatusNotFound, rec.Code)

	// VOLUNTEER → 403 (Coordinator+ per docs/11).
	req = httptest.NewRequest(http.MethodGet, "/api/v1/reports/campaigns/"+campaignID.String(), nil)
	rec = h.doRequest(h.authedRequest(req))
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
