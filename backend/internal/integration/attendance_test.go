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
)

// createAttendanceViaAPI posts a minimal attendance and returns its id.
func createAttendanceViaAPI(t *testing.T, h *testHarness, personID, professionalID, serviceTypeID uuid.UUID) uuid.UUID {
	t.Helper()
	rec := postCampaignJSON(h, "/api/v1/attendances", map[string]any{
		"person_id":       personID.String(),
		"service_type_id": serviceTypeID.String(),
		"professional_id": professionalID.String(),
	}, asProfessional)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	return created.ID
}

// TestAttendanceTransition_Regression pins the transition endpoint end to end
// against real Postgres. Sprint 5 changed the attendance persistence contract
// (campaign_id column); this endpoint previously had no integration coverage
// and a RETURNING/Scan mismatch shipped silently (review Blocker B1).
func TestAttendanceTransition_Regression(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	personID := seedPerson(t, h, h.campusID, "Transition Person", "20230340450")
	professionalID := seedPerson(t, h, h.campusID, "Transition Pro", "20230340451")
	var serviceTypeID uuid.UUID
	require.NoError(t, h.pool.QueryRow(ctx, `SELECT id FROM service_type LIMIT 1`).Scan(&serviceTypeID))

	attendanceID := createAttendanceViaAPI(t, h, personID, professionalID, serviceTypeID)

	body, _ := json.Marshal(map[string]any{"to": "IN_PROGRESS", "reason": "starting"})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/attendances/"+attendanceID.String()+"/transitions", bytes.NewReader(body))
	rec := h.doRequest(h.authedRequest(req, asProfessional))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var status string
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT status FROM attendance WHERE id = $1`, attendanceID).Scan(&status))
	assert.Equal(t, "IN_PROGRESS", status)

	var transitions int
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM attendance_transition WHERE attendance_id = $1`, attendanceID,
	).Scan(&transitions))
	assert.Equal(t, 1, transitions)
}

// TestAttendanceUpdateNotes_Regression pins the notes endpoint end to end
// (same Sprint 5 persistence-contract change, same prior blind spot).
func TestAttendanceUpdateNotes_Regression(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	personID := seedPerson(t, h, h.campusID, "Notes Person", "20230340452")
	professionalID := seedPerson(t, h, h.campusID, "Notes Pro", "20230340453")
	var serviceTypeID uuid.UUID
	require.NoError(t, h.pool.QueryRow(ctx, `SELECT id FROM service_type LIMIT 1`).Scan(&serviceTypeID))

	attendanceID := createAttendanceViaAPI(t, h, personID, professionalID, serviceTypeID)

	body, _ := json.Marshal(map[string]any{"observations": "obs text", "recommendations": "rec text"})
	req := httptest.NewRequest(http.MethodPatch,
		"/api/v1/attendances/"+attendanceID.String()+"/notes", bytes.NewReader(body))
	rec := h.doRequest(h.authedRequest(req, asProfessional))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var obs string
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT observations FROM attendance WHERE id = $1`, attendanceID).Scan(&obs))
	assert.Equal(t, "obs text", obs)
}
