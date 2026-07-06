//go:build integration

package integration_test

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// seedConsentRow inserts a consent directly for report aggregation tests.
func seedConsentRow(t *testing.T, h *testHarness, personID, campusID uuid.UUID, consentType string, active bool) {
	t.Helper()
	_, err := h.pool.Exec(context.Background(), `
		INSERT INTO consent (person_id, consent_type, purpose, is_active, revoked_at, campus_id)
		VALUES ($1, $2, 'report seed', $3, CASE WHEN $3 THEN NULL ELSE NOW() END, $4)
	`, personID, consentType, active, campusID)
	require.NoError(t, err)
}

// TestComplianceReport_HappyPath covers S11.4: a coordinator gets campus-scoped
// LGPD metrics — consent posture, anonymized subjects, data subjects, documents.
func TestComplianceReport_HappyPath(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	p1 := seedPerson(t, h, h.campusID, "Subject One", "90010010011")
	p2 := seedPerson(t, h, h.campusID, "Subject Two", "90010010012")
	anon := seedPerson(t, h, h.campusID, "Anon Subject", "90010010013")
	_, err := h.pool.Exec(ctx, `UPDATE person SET anonymized_at = NOW() WHERE id = $1`, anon)
	require.NoError(t, err)

	// Two persons hold distinct consent types; one is revoked.
	seedConsentRow(t, h, p1, h.campusID, "DATA_PROCESSING", true)
	seedConsentRow(t, h, p1, h.campusID, "IMAGE_USAGE", false)
	seedConsentRow(t, h, p2, h.campusID, "DATA_PROCESSING", true)

	// One active document for the campus.
	_, err = h.pool.Exec(ctx, `
		INSERT INTO document (person_id, campus_id, document_type, file_name, file_path, uploaded_by, is_active)
		VALUES ($1, $2, 'ID', 'id.pdf', 'documents/x/id.pdf', $3, TRUE)
	`, p1, h.campusID, uuid.New())
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/reports/compliance?start=2020-01-01&end=2030-12-31", nil)
	rec := h.doRequest(h.authedRequest(req, asCoordinator))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var report domain.ComplianceReport
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &report))
	assert.Equal(t, 2, report.ConsentsByType["DATA_PROCESSING"])
	assert.Equal(t, 1, report.ConsentsByType["IMAGE_USAGE"])
	assert.Equal(t, 2, report.ActiveConsents)
	assert.Equal(t, 1, report.RevokedConsents)
	assert.Equal(t, 1, report.AnonymizedSubjects)
	assert.Equal(t, 3, report.DataSubjects)
	assert.Equal(t, 1, report.DocumentsStored)
}

// TestComplianceReport_CampusBoundary proves foreign-campus data never leaks
// into the report.
func TestComplianceReport_CampusBoundary(t *testing.T) {
	h := freshHarness(t)

	local := seedPerson(t, h, h.campusID, "Local Subject", "90010010021")
	seedConsentRow(t, h, local, h.campusID, "DATA_PROCESSING", true)

	secondCampus := seedSecondCampus(t, h)
	foreign := seedPerson(t, h, secondCampus, "Foreign Subject", "90010010022")
	seedConsentRow(t, h, foreign, secondCampus, "DATA_PROCESSING", true)
	seedConsentRow(t, h, foreign, secondCampus, "IMAGE_USAGE", true)

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/reports/compliance?start=2020-01-01&end=2030-12-31", nil)
	rec := h.doRequest(h.authedRequest(req, asCoordinator))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var report domain.ComplianceReport
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &report))
	assert.Equal(t, 1, report.ConsentsByType["DATA_PROCESSING"], "only campus A consents count")
	assert.Zero(t, report.ConsentsByType["IMAGE_USAGE"], "campus B consent excluded")
	assert.Equal(t, 1, report.DataSubjects, "only campus A subjects count")
}

// TestComplianceReport_ExportWritesAuditAndRBAC covers the CSV export (EXPORT
// audit) and the 403 for a user below coordinator.
func TestComplianceReport_ExportWritesAuditAndRBAC(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	p := seedPerson(t, h, h.campusID, "Export Subject", "90010010031")
	seedConsentRow(t, h, p, h.campusID, "DATA_PROCESSING", true)

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/reports/compliance/export?format=csv&start=2020-01-01&end=2030-12-31", nil)
	rec := h.doRequest(h.authedRequest(req, asCoordinator))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/csv")

	reader := csv.NewReader(strings.NewReader(rec.Body.String()))
	records, err := reader.ReadAll()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(records), 2)
	assert.Equal(t, []string{"metric", "value"}, records[0])

	var exportAudits int
	require.NoError(t, h.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_log
		WHERE action_type = 'EXPORT' AND entity_type = 'compliance_report'
	`).Scan(&exportAudits))
	assert.Equal(t, 1, exportAudits, "CSV export must write an EXPORT audit entry")

	// Secretary is below coordinator → 403.
	req = httptest.NewRequest(http.MethodGet,
		"/api/v1/reports/compliance?start=2020-01-01&end=2030-12-31", nil)
	rec = h.doRequest(h.authedRequest(req, func(c *auth.AuthClaims) { c.Roles = []string{"SECRETARY"} }))
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
