//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedAddress attaches a primary address with PII to a person so anonymization
// has something to scrub.
func seedAddress(t *testing.T, h *testHarness, personID uuid.UUID) {
	t.Helper()
	_, err := h.pool.Exec(context.Background(), `
		INSERT INTO address (id, person_id, street, number, neighborhood, city, state, zip_code, country, is_primary)
		VALUES ($1, $2, 'Rua das Flores', '100', 'Centro', 'São Paulo', 'SP', '01000-000', 'BRA', TRUE)
	`, uuid.New(), personID)
	require.NoError(t, err)
}

// seedActiveConsent inserts an active consent of the given type for a person,
// returning its id.
func seedActiveConsent(t *testing.T, h *testHarness, personID, campusID uuid.UUID, consentType string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := h.pool.Exec(context.Background(), `
		INSERT INTO consent (id, person_id, consent_type, purpose, campus_id)
		VALUES ($1, $2, $3, 'Registration and follow-up', $4)
	`, id, personID, consentType, campusID)
	require.NoError(t, err)
	return id
}

// TestConsentRevoke_AnonymizesPerson covers S11.3: revoking a DATA_PROCESSING
// consent scrubs the linked person's PII and addresses in the same request,
// stamps anonymized_at, and audits the erasure.
func TestConsentRevoke_AnonymizesPerson(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	personID := seedPerson(t, h, h.campusID, "Maria Silva", "11122233344")
	seedAddress(t, h, personID)
	consentID := seedActiveConsent(t, h, personID, h.campusID, "DATA_PROCESSING")

	rec := revokeConsentJSON(h, consentID.String(), map[string]any{"revoked_reason": "Data subject request"}, asAdmin)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// Person PII is scrubbed and anonymized_at is set.
	var (
		fullName    string
		documentNum *string
		email       *string
		phone       *string
		anonymized  *string
	)
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT full_name, document_number, email, phone, anonymized_at::text FROM person WHERE id = $1`, personID,
	).Scan(&fullName, &documentNum, &email, &phone, &anonymized))
	assert.Equal(t, "[ANONYMIZED]", fullName)
	require.NotNil(t, documentNum)
	assert.Contains(t, *documentNum, "ANON-")
	assert.Nil(t, email)
	assert.Nil(t, phone)
	require.NotNil(t, anonymized, "anonymized_at must be set")

	// Address is scrubbed.
	var street, city *string
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT street, city FROM address WHERE person_id = $1`, personID,
	).Scan(&street, &city))
	assert.Nil(t, street)
	assert.Nil(t, city)

	// Consent row is revoked.
	var active bool
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT is_active FROM consent WHERE id = $1`, consentID,
	).Scan(&active))
	assert.False(t, active)

	// A person-anonymization audit entry exists.
	var auditCount int
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_log WHERE entity_type = 'person' AND entity_id = $1 AND description LIKE 'person anonymized%'`, personID,
	).Scan(&auditCount))
	assert.Equal(t, 1, auditCount)

	// The former name is no longer searchable (search_vector refreshed by trigger).
	var searchHits int
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM person WHERE id = $1 AND search_vector @@ plainto_tsquery('portuguese', 'Maria Silva')`, personID,
	).Scan(&searchHits))
	assert.Equal(t, 0, searchHits)
}

// TestConsentRevoke_NonDataProcessingKeepsPII covers S11.3: revoking a consent
// of any type other than DATA_PROCESSING revokes the consent but leaves the
// person's PII intact.
func TestConsentRevoke_NonDataProcessingKeepsPII(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	personID := seedPerson(t, h, h.campusID, "João Souza", "55566677788")
	consentID := seedActiveConsent(t, h, personID, h.campusID, "IMAGE_USAGE")

	rec := revokeConsentJSON(h, consentID.String(), map[string]any{"revoked_reason": "No longer consents to images"}, asAdmin)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var (
		fullName   string
		anonymized *string
	)
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT full_name, anonymized_at::text FROM person WHERE id = $1`, personID,
	).Scan(&fullName, &anonymized))
	assert.Equal(t, "João Souza", fullName)
	assert.Nil(t, anonymized, "non-DATA_PROCESSING revocation must not anonymize")
}

// TestConsentRevoke_CrossCampus covers S11.3: a consent whose person lives in
// another campus is not visible to the caller — revocation returns 404 and no
// PII is scrubbed.
func TestConsentRevoke_CrossCampus(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	otherCampus := seedSecondCampus(t, h)
	foreignPerson := seedPerson(t, h, otherCampus, "Foreign Person", "99988877766")
	foreignConsent := seedActiveConsent(t, h, foreignPerson, otherCampus, "DATA_PROCESSING")

	// Caller is in the harness default campus; the consent is in otherCampus.
	rec := revokeConsentJSON(h, foreignConsent.String(), map[string]any{"revoked_reason": "attempt"}, asAdmin)
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())

	var fullName string
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT full_name FROM person WHERE id = $1`, foreignPerson,
	).Scan(&fullName))
	assert.Equal(t, "Foreign Person", fullName, "foreign person PII must be untouched")
}
