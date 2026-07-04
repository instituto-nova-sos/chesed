//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/instituto-nova-sos/chesed/internal/auth"
)

func asSecretary(c *auth.AuthClaims) { c.Roles = []string{"SECRETARY"} }
func asAdmin(c *auth.AuthClaims)     { c.Roles = []string{"ADMIN"} }

func postConsentJSON(h *testHarness, payload map[string]any, opts ...func(*auth.AuthClaims)) *httptest.ResponseRecorder {
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/consents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return h.doRequest(h.authedRequest(req, opts...))
}

func revokeConsentJSON(h *testHarness, id string, payload map[string]any, opts ...func(*auth.AuthClaims)) *httptest.ResponseRecorder {
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/consents/"+id+"/revoke", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return h.doRequest(h.authedRequest(req, opts...))
}

func listConsents(h *testHarness, personID string, opts ...func(*auth.AuthClaims)) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/persons/"+personID+"/consents", nil)
	return h.doRequest(h.authedRequest(req, opts...))
}

func validConsentPayload(personID uuid.UUID) map[string]any {
	return map[string]any{
		"person_id":      personID.String(),
		"consent_type":   "DATA_PROCESSING",
		"purpose":        "Registration and attendance follow-up",
		"signature_data": "data:image/png;base64,iVBOR",
	}
}

// TestConsentCreate_HappyPath covers S08.3: any authenticated role registers a
// consent; the row persists campus-stamped with defaults and an audit CREATE
// entry is written.
func TestConsentCreate_HappyPath(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	personID := seedPerson(t, h, h.campusID, "Consent Holder", "10020030041")

	rec := postConsentJSON(h, validConsentPayload(personID)) // default claims: VOLUNTEER
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var created struct {
		ID             uuid.UUID  `json:"id"`
		ConsentVersion string     `json:"consent_version"`
		IsActive       bool       `json:"is_active"`
		GrantedAt      time.Time  `json:"granted_at"`
		RevokedAt      *time.Time `json:"revoked_at"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, "1.0", created.ConsentVersion, "consent_version defaults to 1.0")
	assert.True(t, created.IsActive)
	assert.True(t, withinSeconds(created.GrantedAt, 10), "granted_at must be fresh")
	assert.Nil(t, created.RevokedAt)

	var dbCampus uuid.UUID
	var dbActive bool
	var dbSignature *string
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT campus_id, is_active, signature_data FROM consent WHERE id = $1`, created.ID,
	).Scan(&dbCampus, &dbActive, &dbSignature))
	assert.Equal(t, h.campusID, dbCampus)
	assert.True(t, dbActive)
	require.NotNil(t, dbSignature)
	assert.Equal(t, "data:image/png;base64,iVBOR", *dbSignature)

	var auditCount int
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_log WHERE entity_type = 'consent' AND action_type = 'CREATE'`,
	).Scan(&auditCount))
	assert.Equal(t, 1, auditCount)
}

// TestConsentCreate_DuplicateAndRegrant proves the uq_consent_active_person_type
// contract end to end: a second active grant is a 409 both at the API and at
// the SQL layer, and re-granting after a revocation creates a new row (S08.3,
// S08.4 re-grant criterion).
func TestConsentCreate_DuplicateAndRegrant(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	personID := seedPerson(t, h, h.campusID, "Duplicate Holder", "10020030042")

	rec := postConsentJSON(h, validConsentPayload(personID))
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	rec = postConsentJSON(h, validConsentPayload(personID))
	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "duplicate")

	// Direct-SQL proof the partial unique index blocks a second active row.
	_, err := h.pool.Exec(ctx, `
		INSERT INTO consent (person_id, consent_type, purpose, campus_id)
		VALUES ($1, 'DATA_PROCESSING', 'direct insert', $2)
	`, personID, h.campusID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "uq_consent_active_person_type")

	var rows int
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM consent WHERE person_id = $1`, personID,
	).Scan(&rows))
	assert.Equal(t, 1, rows)

	// Revoke, then re-grant the same type: accepted as a new row.
	rec = revokeConsentJSON(h, created.ID.String(), map[string]any{"revoked_reason": "Data subject request"}, asAdmin)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = postConsentJSON(h, validConsentPayload(personID))
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var activeRows int
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM consent WHERE person_id = $1`, personID).Scan(&rows))
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM consent WHERE person_id = $1 AND is_active`, personID).Scan(&activeRows))
	assert.Equal(t, 2, rows, "revoked row is preserved and re-grant adds a new one")
	assert.Equal(t, 1, activeRows)
}

// TestConsentCreate_Validation exercises the documented 400 contract: missing
// purpose and cross-campus person/guardian references persist nothing and stay
// generic (T3 — no foreign-campus existence disclosure).
func TestConsentCreate_Validation(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	personID := seedPerson(t, h, h.campusID, "Local Holder", "10020030043")
	secondCampus := seedSecondCampus(t, h)
	foreignPerson := seedPerson(t, h, secondCampus, "Foreign Holder", "10020030044")

	payload := validConsentPayload(personID)
	delete(payload, "purpose")
	rec := postConsentJSON(h, payload)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "validation_error")

	rec = postConsentJSON(h, validConsentPayload(foreignPerson))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "validation_error")
	assert.NotContains(t, rec.Body.String(), "Foreign Holder",
		"error must not disclose foreign-campus existence")

	payload = validConsentPayload(personID)
	payload["granted_by_person_id"] = foreignPerson.String()
	rec = postConsentJSON(h, payload)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "validation_error")

	var count int
	require.NoError(t, h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM consent`).Scan(&count))
	assert.Zero(t, count, "nothing may persist after rejected creates")
}

// TestConsentList_RegistryAndRBAC covers the RF-58b registry read: active and
// revoked rows return ordered by granted_at desc for SECRETARY+, while a
// VOLUNTEER gets 403 with an ACCESS_DENIED audit row (docs/16).
func TestConsentList_RegistryAndRBAC(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	personID := seedPerson(t, h, h.campusID, "Registry Holder", "10020030045")

	// Older revoked consent inserted directly so the ordering is deterministic.
	olderID := uuid.New()
	_, err := h.pool.Exec(ctx, `
		INSERT INTO consent (id, person_id, consent_type, purpose, granted_at,
		                     is_active, revoked_at, revoked_reason, campus_id)
		VALUES ($1, $2, 'IMAGE_USAGE', 'old image consent', NOW() - INTERVAL '30 days',
		        FALSE, NOW() - INTERVAL '10 days', 'Data subject request', $3)
	`, olderID, personID, h.campusID)
	require.NoError(t, err)

	rec := postConsentJSON(h, validConsentPayload(personID))
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	rec = listConsents(h, personID.String(), asSecretary)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var resp struct {
		Data []struct {
			ID            uuid.UUID `json:"id"`
			ConsentType   string    `json:"consent_type"`
			IsActive      bool      `json:"is_active"`
			GrantedAt     time.Time `json:"granted_at"`
			RevokedReason *string   `json:"revoked_reason"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 2, "registry returns active and revoked rows")
	assert.Equal(t, "DATA_PROCESSING", resp.Data[0].ConsentType, "newest grant first")
	assert.True(t, resp.Data[0].IsActive)
	assert.Equal(t, olderID, resp.Data[1].ID)
	assert.False(t, resp.Data[1].IsActive)
	require.NotNil(t, resp.Data[1].RevokedReason)
	assert.True(t, resp.Data[0].GrantedAt.After(resp.Data[1].GrantedAt))

	// VOLUNTEER is denied and the denial is audited.
	rec = listConsents(h, personID.String()) // default claims: VOLUNTEER
	assert.Equal(t, http.StatusForbidden, rec.Code)

	var auditCount int
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_log WHERE action_type = 'ACCESS_DENIED'`,
	).Scan(&auditCount))
	assert.Equal(t, 1, auditCount, "RBAC denial must be audited")
}

// TestConsentRevoke_Flow covers S08.4: an ADMIN revoke preserves the row with
// is_active=false, revoked_at and reason set, audits old/new values, and a
// second revoke is a 400 without further changes.
func TestConsentRevoke_Flow(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	personID := seedPerson(t, h, h.campusID, "Revoke Holder", "10020030046")
	rec := postConsentJSON(h, validConsentPayload(personID))
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	rec = revokeConsentJSON(h, created.ID.String(), map[string]any{"revoked_reason": "Data subject request"}, asAdmin)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var revoked struct {
		IsActive      bool       `json:"is_active"`
		RevokedAt     *time.Time `json:"revoked_at"`
		RevokedReason *string    `json:"revoked_reason"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &revoked))
	assert.False(t, revoked.IsActive)
	require.NotNil(t, revoked.RevokedAt)
	require.NotNil(t, revoked.RevokedReason)
	assert.Equal(t, "Data subject request", *revoked.RevokedReason)

	var dbActive bool
	var dbRevokedAt *time.Time
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT is_active, revoked_at FROM consent WHERE id = $1`, created.ID,
	).Scan(&dbActive, &dbRevokedAt))
	assert.False(t, dbActive)
	require.NotNil(t, dbRevokedAt)

	var auditCount int
	require.NoError(t, h.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_log
		WHERE entity_type = 'consent' AND action_type = 'UPDATE'
		  AND old_values IS NOT NULL AND new_values IS NOT NULL
	`).Scan(&auditCount))
	assert.Equal(t, 1, auditCount, "revoke must audit UPDATE with old and new values")

	// Double revoke → 400 and nothing changes.
	rec = revokeConsentJSON(h, created.ID.String(), map[string]any{"revoked_reason": "again"}, asAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "validation_error")

	var dbReason string
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT revoked_reason FROM consent WHERE id = $1`, created.ID,
	).Scan(&dbReason))
	assert.Equal(t, "Data subject request", dbReason, "second revoke must not overwrite the registry")
}

// TestConsentRevoke_RBACAndCampusBoundary proves revoke is ADMIN-only (denials
// audited) and a foreign-campus consent is a 404 with no state change.
func TestConsentRevoke_RBACAndCampusBoundary(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	personID := seedPerson(t, h, h.campusID, "Boundary Holder", "10020030047")
	rec := postConsentJSON(h, validConsentPayload(personID))
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	rec = revokeConsentJSON(h, created.ID.String(), map[string]any{"revoked_reason": "coup"}, asCoordinator)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	var auditCount int
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_log WHERE action_type = 'ACCESS_DENIED'`,
	).Scan(&auditCount))
	assert.Equal(t, 1, auditCount, "RBAC denial must be audited")

	var dbActive bool
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT is_active FROM consent WHERE id = $1`, created.ID,
	).Scan(&dbActive))
	assert.True(t, dbActive, "denied revoke must not change state")

	// Foreign-campus consent → 404 for an ADMIN of campus A.
	secondCampus := seedSecondCampus(t, h)
	foreignPerson := seedPerson(t, h, secondCampus, "Foreign Boundary", "10020030048")
	foreignConsent := uuid.New()
	_, err := h.pool.Exec(ctx, `
		INSERT INTO consent (id, person_id, consent_type, purpose, campus_id)
		VALUES ($1, $2, 'DATA_PROCESSING', 'foreign consent', $3)
	`, foreignConsent, foreignPerson, secondCampus)
	require.NoError(t, err)

	rec = revokeConsentJSON(h, foreignConsent.String(), map[string]any{"revoked_reason": "cross"}, asAdmin)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT is_active FROM consent WHERE id = $1`, foreignConsent,
	).Scan(&dbActive))
	assert.True(t, dbActive)
}

// TestConsentList_CrossCampusNotFound proves the campus boundary on the
// consent registry GET: a person of another campus yields 404 with no
// existence disclosure (review remediation — mandate gap).
func TestConsentList_CrossCampusNotFound(t *testing.T) {
	h := freshHarness(t)

	personID := seedPerson(t, h, h.campusID, "Consent List Person", "50230340480")
	secondCampus := seedSecondCampus(t, h)

	rec := h.doRequest(h.authedRequest(
		getRequest(t, "/api/v1/persons/"+personID.String()+"/consents"),
		asSecretary,
		func(c *auth.AuthClaims) { c.CampusID = secondCampus }))
	require.Equal(t, http.StatusNotFound, rec.Code,
		"consent registry must be campus-scoped; body=%s", rec.Body.String())
}
