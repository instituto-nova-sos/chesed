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

// seedFinancialDonation inserts a financial donation for a campus and returns
// its id.
func seedFinancialDonation(t *testing.T, h *testHarness, campusID uuid.UUID, donorID *uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := h.pool.Exec(context.Background(), `
		INSERT INTO donation (id, campus_id, donor_person_id, donation_type, amount, currency, registered_by)
		VALUES ($1, $2, $3, 'FINANCIAL', 1250.50, 'BRL', $4)
	`, id, campusID, donorID, uuid.New())
	require.NoError(t, err)
	return id
}

// TestDonationReceipt_IssueIsIdempotent drives the receipt endpoint end to end:
// first call renders + stores the PDF and stamps the donation; second call
// returns the same receipt number without re-issuing (S11.5).
func TestDonationReceipt_IssueIsIdempotent(t *testing.T) {
	store := startDocumentStorage(t)
	h := freshHarnessWithStorage(t, store)

	donorID := seedPerson(t, h, h.campusID, "Maria Silva", "52998224725")
	donationID := seedFinancialDonation(t, h, h.campusID, &donorID)

	// First issuance.
	rec := h.doRequest(h.authedRequest(
		httptest.NewRequest(http.MethodGet, "/api/v1/donations/"+donationID.String()+"/receipt", nil),
		asCoordinator))
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var first struct {
		URL       string `json:"url"`
		ExpiresAt string `json:"expires_at"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &first))
	assert.NotEmpty(t, first.URL)
	assert.NotEmpty(t, first.ExpiresAt)

	// DB is stamped.
	var receiptNumber *string
	var issuedAt *string
	require.NoError(t, h.pool.QueryRow(context.Background(),
		`SELECT receipt_number, receipt_issued_at::text FROM donation WHERE id = $1`, donationID,
	).Scan(&receiptNumber, &issuedAt))
	require.NotNil(t, receiptNumber)
	assert.Contains(t, *receiptNumber, "REC-")
	require.NotNil(t, issuedAt)

	// An issuance audit row exists.
	var issuanceAudits int
	require.NoError(t, h.pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM audit_log WHERE entity_type='donation' AND action_type='UPDATE' AND description='receipt issued'`,
	).Scan(&issuanceAudits))
	assert.Equal(t, 1, issuanceAudits)

	// Second call: same receipt number, no new issuance audit.
	rec2 := h.doRequest(h.authedRequest(
		httptest.NewRequest(http.MethodGet, "/api/v1/donations/"+donationID.String()+"/receipt", nil),
		asCoordinator))
	require.Equal(t, http.StatusOK, rec2.Code)

	var secondNumber *string
	require.NoError(t, h.pool.QueryRow(context.Background(),
		`SELECT receipt_number FROM donation WHERE id = $1`, donationID,
	).Scan(&secondNumber))
	require.NotNil(t, secondNumber)
	assert.Equal(t, *receiptNumber, *secondNumber, "receipt number must be stable across downloads")

	require.NoError(t, h.pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM audit_log WHERE entity_type='donation' AND action_type='UPDATE' AND description='receipt issued'`,
	).Scan(&issuanceAudits))
	assert.Equal(t, 1, issuanceAudits, "second download must not issue a new receipt")
}

// TestDonationReceipt_CampusBoundary proves a receipt cannot be issued for a
// donation in another campus.
func TestDonationReceipt_CampusBoundary(t *testing.T) {
	store := startDocumentStorage(t)
	h := freshHarnessWithStorage(t, store)

	secondCampus := seedSecondCampus(t, h)
	foreignID := seedFinancialDonation(t, h, secondCampus, nil)

	rec := h.doRequest(h.authedRequest(
		httptest.NewRequest(http.MethodGet, "/api/v1/donations/"+foreignID.String()+"/receipt", nil),
		asCoordinator))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// TestDonationReceipt_RequiresCoordinator proves a Secretary cannot download a
// receipt (reads are Coordinator+).
func TestDonationReceipt_RequiresCoordinator(t *testing.T) {
	store := startDocumentStorage(t)
	h := freshHarnessWithStorage(t, store)

	donationID := seedFinancialDonation(t, h, h.campusID, nil)

	rec := h.doRequest(h.authedRequest(
		httptest.NewRequest(http.MethodGet, "/api/v1/donations/"+donationID.String()+"/receipt", nil),
		asSecretary))
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
