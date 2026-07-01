//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// TestSyncPush_PersonHappyPath drives the full HTTP → service → repository →
// real Postgres path for a single offline-created person record. It asserts
// both the wire-level response shape and the persisted row, so a regression
// in either layer trips the test.
func TestSyncPush_PersonHappyPath(t *testing.T) {
	h := freshHarness(t)

	syncID := uuid.New()
	payload := domain.SyncPushRequest{Records: []domain.SyncPushRecord{{
		EntityType: domain.SyncEntityPerson,
		SyncID:     syncID,
		Data: map[string]any{
			"full_name":     "Maria Integration",
			"document_type": "CPF",
			"nationality":   "BRA",
		},
	}}}

	rec := h.doRequest(h.authedRequest(postJSON(t, "/api/v1/sync/push", payload)))

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var resp domain.SyncPushResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Len(t, resp.Results, 1)
	assert.Equal(t, domain.SyncStatusCreated, resp.Results[0].Status)
	require.NotNil(t, resp.Results[0].ServerID)
	assert.True(t, withinSeconds(resp.ServerTimestamp, 5))

	// DB row exists with the matching sync_id and the auto-resolved campus.
	var (
		dbName     string
		dbCampusID uuid.UUID
		dbSyncID   uuid.UUID
	)
	err := h.pool.QueryRow(context.Background(),
		`SELECT full_name, campus_id, sync_id FROM person WHERE sync_id = $1`, syncID,
	).Scan(&dbName, &dbCampusID, &dbSyncID)
	require.NoError(t, err, "expected one person row keyed by sync_id")
	assert.Equal(t, "Maria Integration", dbName)
	assert.Equal(t, h.campusID, dbCampusID)
	assert.Equal(t, syncID, dbSyncID)
}

// TestSyncPush_IdempotentRePush proves the documented behaviour that
// re-pushing the same sync_id never creates a duplicate row, regardless of
// how many retries the client makes. Without this guarantee, mobile clients
// would flood the DB on flaky networks.
func TestSyncPush_IdempotentRePush(t *testing.T) {
	h := freshHarness(t)

	syncID := uuid.New()
	body := domain.SyncPushRequest{Records: []domain.SyncPushRecord{{
		EntityType: domain.SyncEntityPerson,
		SyncID:     syncID,
		Data: map[string]any{
			"full_name":     "Carlos Repush",
			"document_type": "CPF",
			"nationality":   "BRA",
		},
	}}}

	// First push: creates.
	rec1 := h.doRequest(h.authedRequest(postJSON(t, "/api/v1/sync/push", body)))
	require.Equal(t, http.StatusOK, rec1.Code, "body=%s", rec1.Body.String())
	var resp1 domain.SyncPushResponse
	require.NoError(t, json.NewDecoder(rec1.Body).Decode(&resp1))
	require.NotNil(t, resp1.Results[0].ServerID)
	firstServerID := *resp1.Results[0].ServerID

	// Second push with identical sync_id: returns the same server_id, no new row.
	rec2 := h.doRequest(h.authedRequest(postJSON(t, "/api/v1/sync/push", body)))
	require.Equal(t, http.StatusOK, rec2.Code)
	var resp2 domain.SyncPushResponse
	require.NoError(t, json.NewDecoder(rec2.Body).Decode(&resp2))
	require.NotNil(t, resp2.Results[0].ServerID)
	assert.Equal(t, domain.SyncStatusCreated, resp2.Results[0].Status,
		"idempotent re-push reports created (existing record returned)")
	assert.Equal(t, firstServerID, *resp2.Results[0].ServerID,
		"server_id must be stable across re-pushes")

	// Exactly one row in DB for this sync_id.
	var count int
	require.NoError(t, h.pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM person WHERE sync_id = $1`, syncID,
	).Scan(&count))
	assert.Equal(t, 1, count, "duplicate sync_id must not create a second row")
}

// TestSyncPush_BatchTooLargeRejected pins the 50-record cap. The strategy
// doc and API spec both reference this number; we lock it in a test so a
// future change requires explicit doc + test updates.
func TestSyncPush_BatchTooLargeRejected(t *testing.T) {
	h := freshHarness(t)

	records := make([]domain.SyncPushRecord, domain.MaxSyncBatchSize+1)
	for i := range records {
		records[i] = domain.SyncPushRecord{
			EntityType: domain.SyncEntityPerson,
			SyncID:     uuid.New(),
			Data: map[string]any{
				"full_name":     "Batch Person",
				"document_type": "CPF",
				"nationality":   "BRA",
			},
		}
	}
	body := domain.SyncPushRequest{Records: records}

	rec := h.doRequest(h.authedRequest(postJSON(t, "/api/v1/sync/push", body)))
	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code, "body=%s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), "batch_too_large")

	// Belt-and-suspenders: no rows should have been inserted, even partially.
	var count int
	require.NoError(t, h.pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM person`,
	).Scan(&count))
	assert.Equal(t, 0, count, "oversize batch is rejected atomically")
}

// TestSyncPush_DuplicateSyncIDUniqueConstraint guards the DB-level
// constraint added in migration 000019. A race where two callers attempt
// to insert with the same sync_id (different rows) must be blocked by the
// unique partial index regardless of what the service layer does.
func TestSyncPush_DuplicateSyncIDUniqueConstraint(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	// Manually insert a person with a chosen sync_id.
	syncID := uuid.New()
	personID1 := uuid.New()
	_, err := h.pool.Exec(ctx, `
		INSERT INTO person (id, full_name, document_type, nationality, campus_id, sync_id)
		VALUES ($1, $2, 'CPF', 'BRA', $3, $4)
	`, personID1, "First Person", h.campusID, syncID)
	require.NoError(t, err)

	// Attempting a second insert with the same sync_id must fail with 23505.
	personID2 := uuid.New()
	_, err = h.pool.Exec(ctx, `
		INSERT INTO person (id, full_name, document_type, nationality, campus_id, sync_id)
		VALUES ($1, $2, 'CPF', 'BRA', $3, $4)
	`, personID2, "Second Person", h.campusID, syncID)
	require.Error(t, err, "uq_person_sync_id must reject duplicate sync_id")
	assert.Contains(t, err.Error(), "23505", "expected unique_violation SQLSTATE")
}

// TestSyncPull_CampusScopedDelta proves the multi-tenant boundary holds at
// the SQL level. A second campus's records must never leak into the first
// campus's pull response — even when their updated_at timestamps qualify.
func TestSyncPull_CampusScopedDelta(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	// Create a second campus and a person in each campus.
	secondCampusID := uuid.New()
	_, err := h.pool.Exec(ctx, `
		INSERT INTO campus (id, name, region, is_active)
		VALUES ($1, 'Second Campus', 'BRAZIL', TRUE)
	`, secondCampusID)
	require.NoError(t, err)

	// document_number must be distinct because uq_person_document uses
	// NULLS NOT DISTINCT — two NULLs would collide.
	personA := uuid.New()
	_, err = h.pool.Exec(ctx, `
		INSERT INTO person (id, full_name, document_type, document_number, nationality, campus_id)
		VALUES ($1, 'Person A', 'CPF', '11111111111', 'BRA', $2)
	`, personA, h.campusID)
	require.NoError(t, err)

	personB := uuid.New()
	_, err = h.pool.Exec(ctx, `
		INSERT INTO person (id, full_name, document_type, document_number, nationality, campus_id)
		VALUES ($1, 'Person B', 'CPF', '22222222222', 'BRA', $2)
	`, personB, secondCampusID)
	require.NoError(t, err)

	// Pull as campus A: must see Person A and NOT Person B.
	since := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	q := url.Values{}
	q.Set("since", since)
	q.Set("entity_types", domain.SyncEntityPerson)

	req := h.authedRequest(getRequest(t, fmtSyncURL("/api/v1/sync/pull", q.Encode())))
	rec := h.doRequest(req)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var resp domain.SyncPullResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	ids := map[uuid.UUID]struct{}{}
	for _, r := range resp.Records {
		ids[r.ID] = struct{}{}
	}
	assert.Contains(t, ids, personA, "campus A pull must include its own records")
	assert.NotContains(t, ids, personB, "campus B records must not leak into campus A pull")
}

// TestSyncPull_DeltaCursor exercises the has_more + next_since cursor flow
// end-to-end. We pre-seed records spanning a time window, pull with a
// cursor mid-window, and assert only later records come back.
func TestSyncPull_DeltaCursor(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	older := time.Now().Add(-2 * time.Hour).UTC()
	newer := time.Now().Add(-30 * time.Minute).UTC()

	// document_number values must be distinct (uq_person_document is NULLS NOT DISTINCT)
	idOlder := uuid.New()
	_, err := h.pool.Exec(ctx, `
		INSERT INTO person (id, full_name, document_type, document_number, nationality, campus_id, updated_at)
		VALUES ($1, 'Older', 'CPF', '33333333333', 'BRA', $2, $3)
	`, idOlder, h.campusID, older)
	require.NoError(t, err)

	idNewer := uuid.New()
	_, err = h.pool.Exec(ctx, `
		INSERT INTO person (id, full_name, document_type, document_number, nationality, campus_id, updated_at)
		VALUES ($1, 'Newer', 'CPF', '44444444444', 'BRA', $2, $3)
	`, idNewer, h.campusID, newer)
	require.NoError(t, err)

	// Pull with a cursor that falls between the two rows.
	since := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)
	q := url.Values{}
	q.Set("since", since)
	q.Set("entity_types", domain.SyncEntityPerson)

	rec := h.doRequest(h.authedRequest(getRequest(t, fmtSyncURL("/api/v1/sync/pull", q.Encode()))))
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var resp domain.SyncPullResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	ids := map[uuid.UUID]struct{}{}
	for _, r := range resp.Records {
		ids[r.ID] = struct{}{}
	}
	assert.Contains(t, ids, idNewer, "post-cursor row must appear")
	assert.NotContains(t, ids, idOlder, "pre-cursor row must be excluded")
}

// TestSyncPull_MissingSinceReturns400 documents the wire-level rejection
// of a malformed pull request. The frontend drainer relies on this 400
// code path to surface a clear failure rather than silently retrying.
func TestSyncPull_MissingSinceReturns400(t *testing.T) {
	h := freshHarness(t)
	rec := h.doRequest(h.authedRequest(getRequest(t, "/api/v1/sync/pull?entity_types=person")))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, strings.ToLower(rec.Body.String()), "since")
}

// TestSyncPush_NoCampusReturns403 confirms the multi-tenant guard kicks in
// even when a token is otherwise valid. Without a resolved campus, the
// service refuses the whole batch — the test pins that behaviour against
// the real router.
func TestSyncPush_NoCampusReturns403(t *testing.T) {
	h := freshHarness(t)

	body := domain.SyncPushRequest{Records: []domain.SyncPushRecord{{
		EntityType: domain.SyncEntityPerson,
		SyncID:     uuid.New(),
		Data: map[string]any{
			"full_name":     "No Campus",
			"document_type": "CPF",
			"nationality":   "BRA",
		},
	}}}

	req := postJSON(t, "/api/v1/sync/push", body)
	req = req.WithContext(auth.NewContext(req.Context(), auth.AuthClaims{
		Subject:       uuid.New().String(),
		Email:         "nocampus@chesed.test",
		EmailVerified: true,
		Roles:         []string{"VOLUNTEER"},
		CampusID:      uuid.Nil, // explicit zero — simulates a missing campus claim
	}))
	rec := h.doRequest(req)

	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
}

// TestSyncPush_CrossCampusPersonReferenceRejected proves the multi-tenant
// reference guard (security Finding 3, threat T3): a client authenticated for
// campus B cannot attach a triage to a person that lives in campus A. Before
// the fix, the push stamped the caller's campus onto a triage referencing a
// foreign person_id, silently linking cross-campus data.
func TestSyncPush_CrossCampusPersonReferenceRejected(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	// h.campusID is campus A; create campus B for the caller.
	campusB := uuid.New()
	_, err := h.pool.Exec(ctx, `
		INSERT INTO campus (id, name, region, is_active)
		VALUES ($1, 'Campus B', 'BRAZIL', TRUE)
	`, campusB)
	require.NoError(t, err)

	// The person exists only in campus A.
	personA := uuid.New()
	_, err = h.pool.Exec(ctx, `
		INSERT INTO person (id, full_name, document_type, document_number, nationality, campus_id)
		VALUES ($1, 'Person A', 'CPF', '55555555555', 'BRA', $2)
	`, personA, h.campusID)
	require.NoError(t, err)

	syncID := uuid.New()
	payload := domain.SyncPushRequest{Records: []domain.SyncPushRecord{{
		EntityType: domain.SyncEntityTriage,
		SyncID:     syncID,
		Data: map[string]any{
			"person_id":      personA.String(),
			"main_complaint": "cross-campus attempt",
		},
	}}}

	// Push as campus B, referencing campus A's person.
	req := h.authedRequest(postJSON(t, "/api/v1/sync/push", payload), func(c *auth.AuthClaims) {
		c.CampusID = campusB
	})
	rec := h.doRequest(req)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var resp domain.SyncPushResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Len(t, resp.Results, 1)
	assert.Equal(t, domain.SyncStatusError, resp.Results[0].Status,
		"cross-campus person reference must be rejected")

	// No triage row must have been written for this sync_id.
	var count int
	err = h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM triage WHERE sync_id = $1`, syncID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "no triage row may be persisted for a cross-campus reference")
}

// --- helpers ----------------------------------------------------------------

func postJSON(t *testing.T, path string, payload any) *http.Request {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	r := newRequest(t, http.MethodPost, path, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func getRequest(t *testing.T, path string) *http.Request {
	t.Helper()
	return newRequest(t, http.MethodGet, path, nil)
}

func newRequest(t *testing.T, method, path string, body interface {
	Read(p []byte) (n int, err error)
}) *http.Request {
	t.Helper()
	if body == nil {
		req, err := http.NewRequest(method, path, http.NoBody)
		require.NoError(t, err)
		return req
	}
	req, err := http.NewRequest(method, path, body)
	require.NoError(t, err)
	return req
}
