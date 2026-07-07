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

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/instituto-nova-sos/chesed/internal/handler"
	"github.com/instituto-nova-sos/chesed/internal/middleware"
	"github.com/instituto-nova-sos/chesed/internal/repository"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

// seedCampaignWithStatus inserts a campaign with an explicit lifecycle status so
// tests can prove the public list exposes only ACTIVE campaigns. seedCampaign
// hardcodes 'ACTIVE', so the non-ACTIVE fixtures are inserted here directly,
// mirroring the SQL in campaign_link_test.go / campaign_test.go.
func seedCampaignWithStatus(t *testing.T, h *testHarness, campusID uuid.UUID, name, status string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := h.pool.Exec(context.Background(), `
		INSERT INTO campaign (id, name, campaign_type, start_date, campus_id, status)
		VALUES ($1, $2, 'SOCIAL_ACTION', '2026-07-10', $3, $4)
	`, id, name, campusID, status)
	require.NoError(t, err)
	return id
}

// publicCampaignsResponse mirrors the lean projection the public list returns
// (domain.CampaignListItem inside data). Decoding into this struct confirms the
// shape carries no PII/internal fields; the raw-body assertions below prove the
// hidden fields never leak on the wire.
type publicCampaignsResponse struct {
	Data []struct {
		ID           uuid.UUID `json:"id"`
		Name         string    `json:"name"`
		CampaignType string    `json:"campaign_type"`
		Status       string    `json:"status"`
	} `json:"data"`
}

// TestPublicCampaigns_ReturnsOnlyActive proves GET /public/campaigns exposes
// only ACTIVE campaigns and returns the lean projection with no PII/internal
// fields (S12.1).
func TestPublicCampaigns_ReturnsOnlyActive(t *testing.T) {
	h := freshHarness(t)

	activeID := seedCampaignWithStatus(t, h, h.campusID, "Active Campaign", "ACTIVE")
	seedCampaignWithStatus(t, h, h.campusID, "Planned Campaign", "PLANNED")
	seedCampaignWithStatus(t, h, h.campusID, "Completed Campaign", "COMPLETED")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/campaigns?campus_id="+h.campusID.String(), nil)
	rec := h.doRequest(req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	body := rec.Body.String()
	assert.Contains(t, body, "Active Campaign")
	assert.NotContains(t, body, "Planned Campaign", "non-ACTIVE campaigns must not be exposed publicly")
	assert.NotContains(t, body, "Completed Campaign", "non-ACTIVE campaigns must not be exposed publicly")

	// The lean projection must not leak internal/PII fields onto the wire.
	for _, hidden := range []string{"location_address", "coordinator_id", "description", "team", "created_by"} {
		assert.NotContainsf(t, body, hidden, "public campaign projection must not expose %q", hidden)
	}

	var resp publicCampaignsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 1, "exactly one ACTIVE campaign expected")
	assert.Equal(t, activeID, resp.Data[0].ID)
	assert.Equal(t, "Active Campaign", resp.Data[0].Name)
	assert.Equal(t, "ACTIVE", resp.Data[0].Status)
}

// TestPublicCampaigns_CampusBoundary proves the campus_id filter isolates
// campaigns: an ACTIVE campaign in another campus never appears (rule #4).
func TestPublicCampaigns_CampusBoundary(t *testing.T) {
	h := freshHarness(t)

	seedCampaignWithStatus(t, h, h.campusID, "Home Campaign", "ACTIVE")
	secondCampus := seedSecondCampus(t, h)
	seedCampaignWithStatus(t, h, secondCampus, "Foreign Campaign", "ACTIVE")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/campaigns?campus_id="+h.campusID.String(), nil)
	rec := h.doRequest(req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	body := rec.Body.String()
	assert.Contains(t, body, "Home Campaign")
	assert.NotContains(t, body, "Foreign Campaign", "another campus's ACTIVE campaign must not appear")

	var resp publicCampaignsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "Home Campaign", resp.Data[0].Name)
}

// TestPublicCampaigns_MissingCampusID proves a missing campus_id yields 400.
func TestPublicCampaigns_MissingCampusID(t *testing.T) {
	h := freshHarness(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/campaigns", nil)
	rec := h.doRequest(req)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}

// TestPublicCampaigns_MalformedCampusID proves a non-uuid campus_id yields 400.
func TestPublicCampaigns_MalformedCampusID(t *testing.T) {
	h := freshHarness(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/campaigns?campus_id=not-a-uuid", nil)
	rec := h.doRequest(req)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}

// TestPublicCampaigns_UnknownCampus proves a valid-but-unknown campus_id yields
// 404 (never disclosing which).
func TestPublicCampaigns_UnknownCampus(t *testing.T) {
	h := freshHarness(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/campaigns?campus_id="+uuid.New().String(), nil)
	rec := h.doRequest(req)
	assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
}

// postPublicSignup builds an unauthenticated POST to the volunteer sign-up
// route. Public routes carry no AuthClaims, so the request is sent plain.
func postPublicSignup(payload map[string]any) *http.Request {
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/volunteer-signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// TestPublicVolunteerSignup_HappyPath drives an unauthenticated sign-up and
// asserts, at the DB level, the full side-effect chain: person + VOLUNTEER role
// + PENDING agreement + campus-scoped, actor-less audit entry carrying the
// captured client IP/user-agent. This also proves the nested person-create
// commits under the public campus transaction (S12.1).
func TestPublicVolunteerSignup_HappyPath(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	const (
		fullName  = "Grace Hopper"
		clientIP  = "203.0.113.7"
		userAgent = "ChesedPublicTest/1.0"
	)

	req := postPublicSignup(map[string]any{
		"full_name": fullName,
		"email":     "grace@example.org",
		"phone":     "+5511999998888",
		"campus_id": h.campusID.String(),
	})
	req.Header.Set("X-Forwarded-For", clientIP)
	req.Header.Set("User-Agent", userAgent)

	rec := h.doRequest(req)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var created struct {
		ID       uuid.UUID `json:"id"`
		FullName string    `json:"full_name"`
		CampusID uuid.UUID `json:"campus_id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, fullName, created.FullName)
	assert.Equal(t, h.campusID, created.CampusID)

	// person row exists in the target campus.
	var personID uuid.UUID
	var personCampus uuid.UUID
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT id, campus_id FROM person WHERE full_name = $1`, fullName,
	).Scan(&personID, &personCampus))
	assert.Equal(t, created.ID, personID)
	assert.Equal(t, h.campusID, personCampus)

	// VOLUNTEER person_role for that person. This is the S12.1 contract: a public
	// sign-up MUST produce a VOLUNTEER role for staff to follow up on. It also
	// proves the role INSERT ran inside the same PublicCampusTx transaction that
	// created the person — if the role repository ran on a fresh pool connection
	// it would not see the not-yet-committed person and the FK would fail.
	var roleID uuid.UUID
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT id FROM person_role WHERE person_id = $1 AND role_type = 'VOLUNTEER'`, personID,
	).Scan(&roleID), "public sign-up must create a VOLUNTEER person_role for the new person")

	// PENDING volunteer_agreement for that role (staff completes it later).
	var agreementStatus string
	var agreementCampus uuid.UUID
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT status, campus_id FROM volunteer_agreement WHERE person_role_id = $1`, roleID,
	).Scan(&agreementStatus, &agreementCampus),
		"public sign-up must open a PENDING volunteer_agreement for the new role")
	assert.Equal(t, "PENDING", agreementStatus)
	assert.Equal(t, h.campusID, agreementCampus)

	// audit entry: campus-scoped, no actor (NULL user_id), IP/user-agent captured.
	var (
		auditCampus    uuid.UUID
		auditUserID    *uuid.UUID
		auditIP        string
		auditUserAgent string
		auditModule    string
	)
	require.NoError(t, h.pool.QueryRow(ctx, `
		SELECT campus_id, user_id, host(ip_address), user_agent, module
		FROM audit_log
		WHERE entity_type = 'person' AND entity_id = $1 AND action_type = 'CREATE'
	`, personID).Scan(&auditCampus, &auditUserID, &auditIP, &auditUserAgent, &auditModule))
	assert.Equal(t, h.campusID, auditCampus, "public sign-up audit must be campus-scoped")
	assert.Nil(t, auditUserID, "public sign-up has no authenticated actor")
	assert.Equal(t, clientIP, auditIP, "client IP from X-Forwarded-For must be captured")
	assert.Equal(t, userAgent, auditUserAgent, "user-agent must be captured")
	assert.Equal(t, "public-api", auditModule)
}

// TestPublicVolunteerSignup_ValidationErrors proves malformed sign-up bodies are
// rejected. A missing/absent campus_id is rejected upstream by the validator
// (400 invalid_request); field-level violations are rejected by the service
// (400 validation_error). Both surface as 400.
func TestPublicVolunteerSignup_ValidationErrors(t *testing.T) {
	h := freshHarness(t)

	tests := []struct {
		name    string
		payload map[string]any
	}{
		{
			name:    "missing full_name",
			payload: map[string]any{"email": "x@example.org", "campus_id": h.campusID.String()},
		},
		{
			name:    "invalid email",
			payload: map[string]any{"full_name": "No Email", "email": "not-an-email", "campus_id": h.campusID.String()},
		},
		{
			name:    "missing campus_id",
			payload: map[string]any{"full_name": "No Campus", "email": "x@example.org"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := h.doRequest(postPublicSignup(tt.payload))
			assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		})
	}
}

// TestPublicVolunteerSignup_UnknownCampus proves a valid body with an unknown
// campus_id is rejected with 404 by the campus validator, before any write.
func TestPublicVolunteerSignup_UnknownCampus(t *testing.T) {
	h := freshHarness(t)

	rec := h.doRequest(postPublicSignup(map[string]any{
		"full_name": "Orphan Volunteer",
		"email":     "orphan@example.org",
		"campus_id": uuid.New().String(),
	}))
	assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
}

// buildPublicSignupRouter mounts POST /volunteer-signup with the full public
// middleware chain but a caller-supplied rate limit, so the 429 path can be
// exercised deterministically without touching the shared 600/min harness
// router. It mirrors registerPublicRoutes (and buildRouter) exactly, differing
// only in the rate-limit budget.
func buildPublicSignupRouter(h *testHarness, rpm int) chi.Router {
	auditRepo := repository.NewAuditRepository(h.pool)
	campaignRepo := repository.NewCampaignRepository(h.pool)
	personRepo := repository.NewPersonRepository(h.pool)
	personRoleRepo := repository.NewPersonRoleRepository(h.pool)
	agreementRepo := repository.NewVolunteerAgreementRepository(h.pool)
	campusRepo := repository.NewCampusRepository(h.pool)

	auditSvc := service.NewAuditService(auditRepo)
	publicSvc := service.NewPublicService(campaignRepo, personRepo, personRoleRepo, agreementRepo, auditSvc)
	publicH := handler.NewPublicHandler(publicSvc)

	r := chi.NewRouter()
	r.Route("/api/v1/public", func(r chi.Router) {
		r.Use(middleware.PublicRateLimit(rpm))
		r.Use(middleware.PublicCampusValidator(campusRepo))
		r.Use(middleware.PublicCampusTx(h.pool))
		r.Post("/volunteer-signup", publicH.VolunteerSignup)
	})
	return r
}

// TestPublicVolunteerSignup_RateLimited proves the per-IP throttle returns 429
// once the budget is exhausted. A dedicated router with a budget of 1 makes the
// second request from the same peer address deterministically throttled.
func TestPublicVolunteerSignup_RateLimited(t *testing.T) {
	h := freshHarness(t)
	router := buildPublicSignupRouter(h, 1)

	send := func(n int) *httptest.ResponseRecorder {
		req := postPublicSignup(map[string]any{
			"full_name": fmt.Sprintf("Rate Limited %d", n),
			"email":     fmt.Sprintf("rl%d@example.org", n),
			"campus_id": h.campusID.String(),
		})
		// httptest.NewRequest sets a fixed RemoteAddr (192.0.2.1:1234); both
		// requests share it, so they land in the same rate-limit bucket.
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	first := send(1)
	require.Equal(t, http.StatusCreated, first.Code, first.Body.String())

	second := send(2)
	assert.Equal(t, http.StatusTooManyRequests, second.Code, second.Body.String())
}

// TestPublicSignup_RLSFailsClosed asserts the database-layer safety net the
// public write path relies on, directly (not through HTTP): a person inserted
// under campusA's GUC is visible only under campusA, and an insert whose
// campus_id differs from the GUC is rejected by the WITH CHECK policy. This
// mirrors rls_test.go's cross-campus write/read assertions for the person table
// the public sign-up writes to.
func TestPublicSignup_RLSFailsClosed(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	campusA := h.campusID
	campusB := seedSecondCampus(t, h)
	appPool := openAppPool(t, h)

	volunteerID := uuid.New()

	// (a) INSERT under campusA's GUC succeeds and is visible under campusA.
	withCampusGUC(t, appPool, &campusA, func(tx pgx.Tx) {
		_, err := tx.Exec(ctx,
			`INSERT INTO person (id, full_name, document_type, campus_id, nationality)
			 VALUES ($1, 'RLS Volunteer', 'OTHER', $2, 'BRA')`,
			volunteerID, campusA)
		require.NoError(t, err, "same-campus public volunteer insert must be allowed")

		var count int
		require.NoError(t, tx.QueryRow(ctx,
			`SELECT count(*) FROM person WHERE id = $1`, volunteerID).Scan(&count))
		assert.Equal(t, 1, count, "the row must be visible under its own campus GUC")
	})

	// (b) The same row is INVISIBLE under a different campus GUC.
	withCampusGUC(t, appPool, &campusB, func(tx pgx.Tx) {
		var count int
		require.NoError(t, tx.QueryRow(ctx,
			`SELECT count(*) FROM person WHERE id = $1`, volunteerID).Scan(&count))
		assert.Equal(t, 0, count, "campusA's volunteer must be invisible under campusB's GUC")
	})

	// (c) INSERT of a person for a campus other than the GUC campus is rejected
	// by the WITH CHECK policy (fail-closed).
	withCampusGUC(t, appPool, &campusA, func(tx pgx.Tx) {
		_, err := tx.Exec(ctx,
			`INSERT INTO person (id, full_name, document_type, campus_id, nationality)
			 VALUES ($1, 'Cross Campus Volunteer', 'OTHER', $2, 'BRA')`,
			uuid.New(), campusB)
		require.Error(t, err, "inserting a volunteer for another campus must be rejected by WITH CHECK")
	})
}
