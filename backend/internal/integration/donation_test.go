//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/instituto-nova-sos/chesed/internal/auth"
)

// putDonationJSON issues an authenticated PUT with a JSON body.
func putDonationJSON(h *testHarness, path string, payload map[string]any, opts ...func(*auth.AuthClaims)) *httptest.ResponseRecorder {
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return h.doRequest(h.authedRequest(req, opts...))
}

// TestDonationCRUD_HappyPath drives the documented flow against real Postgres:
// create (GOODS as secretary) → create (FINANCIAL w/ donor) → list/detail as
// coordinator → update, asserting DB state and the audit trail (E09 / S09.1).
func TestDonationCRUD_HappyPath(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	rec := postCampaignJSON(h, "/api/v1/donations", map[string]any{
		"donation_type":    "GOODS",
		"item_description": "10 winter blankets",
		"donation_date":    "2026-07-05",
	}, asSecretary)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var goods struct {
		ID uuid.UUID `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &goods))

	// DB-level assertion: campus-stamped, registered_by set.
	var dbCampus, dbRegisteredBy uuid.UUID
	var dbType string
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT campus_id, registered_by, donation_type FROM donation WHERE id = $1`, goods.ID,
	).Scan(&dbCampus, &dbRegisteredBy, &dbType))
	assert.Equal(t, h.campusID, dbCampus)
	assert.NotEqual(t, uuid.Nil, dbRegisteredBy)
	assert.Equal(t, "GOODS", dbType)

	var createAudit int
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_log WHERE entity_type = 'donation' AND action_type = 'CREATE'`,
	).Scan(&createAudit))
	assert.Equal(t, 1, createAudit)

	// FINANCIAL donation with an in-campus donor.
	donorID := seedPerson(t, h, h.campusID, "Generous Donor", "90080070060")
	rec = postCampaignJSON(h, "/api/v1/donations", map[string]any{
		"donation_type":   "FINANCIAL",
		"amount":          500.50,
		"donor_person_id": donorID.String(),
	}, asSecretary)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var financial struct {
		ID uuid.UUID `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &financial))

	// List as coordinator includes both donations.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/donations", nil)
	rec = h.doRequest(h.authedRequest(req, asCoordinator))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "GOODS")
	assert.Contains(t, rec.Body.String(), "FINANCIAL")

	// Detail as coordinator resolves donor name.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/donations/"+financial.ID.String(), nil)
	rec = h.doRequest(h.authedRequest(req, asCoordinator))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "Generous Donor")

	// Update the GOODS donation (secretary+).
	rec = putDonationJSON(h, "/api/v1/donations/"+goods.ID.String(), map[string]any{
		"donation_type":    "GOODS",
		"item_description": "25 winter blankets",
		"notes":            "quantity corrected",
	}, asSecretary)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var dbItem string
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT item_description FROM donation WHERE id = $1`, goods.ID,
	).Scan(&dbItem))
	assert.Equal(t, "25 winter blankets", dbItem)

	var updateAudit int
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_log WHERE entity_type = 'donation' AND action_type = 'UPDATE'`,
	).Scan(&updateAudit))
	assert.Equal(t, 1, updateAudit)
}

// seedDonation inserts a donation directly for boundary tests.
func seedDonation(t *testing.T, h *testHarness, campusID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := h.pool.Exec(context.Background(), `
		INSERT INTO donation (id, campus_id, donation_type, item_description, registered_by)
		VALUES ($1, $2, 'GOODS', 'Foreign goods', $3)
	`, id, campusID, uuid.New())
	require.NoError(t, err)
	return id
}

// TestDonation_CampusBoundary proves a donation in campus B is invisible to a
// campus A caller: absent from the list, 404 on detail/update (rule #4, T3).
func TestDonation_CampusBoundary(t *testing.T) {
	h := freshHarness(t)

	secondCampus := seedSecondCampus(t, h)
	foreignID := seedDonation(t, h, secondCampus)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/donations", nil)
	rec := h.doRequest(h.authedRequest(req, asCoordinator))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), "Foreign goods")

	req = httptest.NewRequest(http.MethodGet, "/api/v1/donations/"+foreignID.String(), nil)
	rec = h.doRequest(h.authedRequest(req, asCoordinator))
	assert.Equal(t, http.StatusNotFound, rec.Code)

	rec = putDonationJSON(h, "/api/v1/donations/"+foreignID.String(), map[string]any{
		"donation_type":    "GOODS",
		"item_description": "hijack",
	}, asSecretary)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// TestDonation_RBACBoundary proves VOLUNTEER cannot create donations
// (Secretary+ per docs/16) and a below-coordinator role cannot read them
// (Coordinator+ per docs/11); the create denial is audited.
func TestDonation_RBACBoundary(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	rec := postCampaignJSON(h, "/api/v1/donations", map[string]any{
		"donation_type":    "GOODS",
		"item_description": "rogue goods",
	}) // default claims: VOLUNTEER
	require.Equal(t, http.StatusForbidden, rec.Code)

	var count int
	require.NoError(t, h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM donation`).Scan(&count))
	assert.Zero(t, count, "no donation row may exist after a denied create")

	var auditCount int
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_log WHERE action_type = 'ACCESS_DENIED'`,
	).Scan(&auditCount))
	assert.Equal(t, 1, auditCount, "RBAC denial must be audited")

	// A below-coordinator role (SECRETARY) cannot read the donation list.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/donations", nil)
	rec = h.doRequest(h.authedRequest(req, asSecretary))
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// TestDonation_Validation exercises the documented 400 contract: bad type and a
// FINANCIAL donation missing its amount are rejected without touching the DB.
func TestDonation_Validation(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	rec := postCampaignJSON(h, "/api/v1/donations", map[string]any{
		"donation_type":    "CRYPTO",
		"item_description": "bad type",
	}, asSecretary)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = postCampaignJSON(h, "/api/v1/donations", map[string]any{
		"donation_type": "FINANCIAL",
	}, asSecretary)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var count int
	require.NoError(t, h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM donation`).Scan(&count))
	assert.Zero(t, count)
}

// TestDonation_Currency proves the BRL/USD/EUR allow-list is enforced at the API
// layer (validator) and that accepted currencies persist (Sprint 9.3).
func TestDonation_Currency(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	for _, cur := range []string{"BRL", "USD", "EUR"} {
		rec := postCampaignJSON(h, "/api/v1/donations", map[string]any{
			"donation_type": "FINANCIAL",
			"amount":        10.0,
			"currency":      cur,
		}, asSecretary)
		require.Equal(t, http.StatusCreated, rec.Code, "currency %s must be accepted: %s", cur, rec.Body.String())
	}

	rec := postCampaignJSON(h, "/api/v1/donations", map[string]any{
		"donation_type": "FINANCIAL",
		"amount":        10.0,
		"currency":      "GBP",
	}, asSecretary)
	assert.Equal(t, http.StatusBadRequest, rec.Code, "an unlisted currency must be rejected")

	var count int
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM donation WHERE currency IN ('BRL','USD','EUR')`).Scan(&count))
	assert.Equal(t, 3, count, "the three valid-currency donations must persist")
}

// TestDonation_CampaignLink proves a same-campus campaign link persists and a
// foreign-campus link is rejected generically without disclosure (S09.2, T3).
func TestDonation_CampaignLink(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	campaignID := seedCampaign(t, h, h.campusID, "Donation Drive")
	rec := postCampaignJSON(h, "/api/v1/donations", map[string]any{
		"donation_type":    "GOODS",
		"item_description": "linked goods",
		"campaign_id":      campaignID.String(),
	}, asSecretary)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	var linked uuid.UUID
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT campaign_id FROM donation WHERE id = $1`, created.ID,
	).Scan(&linked))
	assert.Equal(t, campaignID, linked)

	// Foreign-campus campaign → generic 400, no disclosure, no extra row.
	secondCampus := seedSecondCampus(t, h)
	foreignCampaign := seedCampaign(t, h, secondCampus, "Foreign Drive")
	rec = postCampaignJSON(h, "/api/v1/donations", map[string]any{
		"donation_type":    "GOODS",
		"item_description": "rejected link",
		"campaign_id":      foreignCampaign.String(),
	}, asSecretary)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.NotContains(t, rec.Body.String(), "Foreign Drive",
		"error must not disclose foreign-campus existence")

	var count int
	require.NoError(t, h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM donation`).Scan(&count))
	assert.Equal(t, 1, count, "no donation row may persist after a rejected campaign link")
}

// TestDonation_AnonymousDonor proves a donation without a donor persists with a
// NULL donor_person_id.
func TestDonation_AnonymousDonor(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	rec := postCampaignJSON(h, "/api/v1/donations", map[string]any{
		"donation_type":    "GOODS",
		"item_description": "anonymous goods",
	}, asSecretary)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	var donor *uuid.UUID
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT donor_person_id FROM donation WHERE id = $1`, created.ID,
	).Scan(&donor))
	assert.Nil(t, donor, "anonymous donation must have a NULL donor_person_id")
}
