//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/instituto-nova-sos/chesed/internal/auth"
)

// seededVolunteer bundles the ids a self-service agreement test needs.
type seededVolunteer struct {
	personID  uuid.UUID
	appUserID uuid.UUID
}

// seedVolunteerWithPendingAgreement creates a person, an active VOLUNTEER role, a local
// app_user projection, and a PENDING volunteer_agreement. The app_user is required
// because accepted_by_user/uploaded_by are FKs to app_user.id (regression guard for the
// FK bug where the Keycloak subject was written instead of the app_user id).
func seedVolunteerWithPendingAgreement(t *testing.T, h *testHarness, campusID uuid.UUID) seededVolunteer {
	t.Helper()
	personID := seedPerson(t, h, campusID, "Volunteer Signer", "50060070081")
	roleID := uuid.New()
	_, err := h.pool.Exec(context.Background(), `
		INSERT INTO person_role (id, person_id, role_type, is_active)
		VALUES ($1, $2, 'VOLUNTEER', TRUE)
	`, roleID, personID)
	require.NoError(t, err)

	appUserID := uuid.New()
	_, err = h.pool.Exec(context.Background(), `
		INSERT INTO app_user (id, person_id, email, keycloak_subject_id, access_profile, campus_id)
		VALUES ($1, $2, 'volunteer.signer@chesed.test', $3, 'VOLUNTEER', $4)
	`, appUserID, personID, uuid.New().String(), campusID)
	require.NoError(t, err)

	_, err = h.pool.Exec(context.Background(), `
		INSERT INTO volunteer_agreement (id, person_id, person_role_id, campus_id, status, agreement_version)
		VALUES ($1, $2, $3, $4, 'PENDING', 'v1')
	`, uuid.New(), personID, roleID, campusID)
	require.NoError(t, err)
	return seededVolunteer{personID: personID, appUserID: appUserID}
}

// asVolunteerPerson sets the claims a self-service request carries. AppUserID mirrors what
// AutoProvision enriches in production and is the value FK columns reference.
func asVolunteerPerson(v seededVolunteer) func(*auth.AuthClaims) {
	return func(c *auth.AuthClaims) {
		c.Roles = []string{"VOLUNTEER"}
		c.PersonID = v.personID
		c.AppUserID = v.appUserID
	}
}

// TestAgreementAcceptDigital_PersistsSignature drives the self-service digital accept
// against real Postgres: the drawn signature is stored in signature_data and the row
// flips to ACCEPTED via the DIGITAL method.
func TestAgreementAcceptDigital_PersistsSignature(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()
	v := seedVolunteerWithPendingAgreement(t, h, h.campusID)

	body, _ := json.Marshal(map[string]any{"signature_data": "data:image/png;base64,SIGDATA"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/volunteer-agreement/accept", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := h.doRequest(h.authedRequest(req, asVolunteerPerson(v)))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var status, method, signature string
	var acceptedBy uuid.UUID
	err := h.pool.QueryRow(ctx, `
		SELECT status, signature_method, signature_data, accepted_by_user
		FROM volunteer_agreement WHERE person_id = $1
	`, v.personID).Scan(&status, &method, &signature, &acceptedBy)
	require.NoError(t, err)
	assert.Equal(t, "ACCEPTED", status)
	assert.Equal(t, "DIGITAL", method)
	assert.Equal(t, "data:image/png;base64,SIGDATA", signature)
	assert.Equal(t, v.appUserID, acceptedBy, "accepted_by_user must be the app_user id, not the Keycloak subject")
}

// TestAgreementAcceptDigital_RejectsEmptySignature asserts the mandatory-signature
// contract: an accept with no signature is a 400 and leaves the row PENDING.
func TestAgreementAcceptDigital_RejectsEmptySignature(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()
	v := seedVolunteerWithPendingAgreement(t, h, h.campusID)

	body, _ := json.Marshal(map[string]any{"signature_data": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/volunteer-agreement/accept", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := h.doRequest(h.authedRequest(req, asVolunteerPerson(v)))

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

	var status string
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT status FROM volunteer_agreement WHERE person_id = $1`, v.personID).Scan(&status))
	assert.Equal(t, "PENDING", status)
}

// TestAgreementAcceptDigital_ConflictWhenAlreadyAccepted asserts a second accept is a 409.
func TestAgreementAcceptDigital_ConflictWhenAlreadyAccepted(t *testing.T) {
	h := freshHarness(t)
	v := seedVolunteerWithPendingAgreement(t, h, h.campusID)

	accept := func() *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]any{"signature_data": "data:image/png;base64,SIG"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/volunteer-agreement/accept", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		return h.doRequest(h.authedRequest(req, asVolunteerPerson(v)))
	}

	require.Equal(t, http.StatusOK, accept().Code)
	assert.Equal(t, http.StatusConflict, accept().Code)
}

// TestAgreementAcceptUpload_SelfService drives the self-service file upload path: a
// volunteer uploads their own signed document and the row flips to ACCEPTED via the
// MANUAL_UPLOAD method with a document_path recorded.
func TestAgreementAcceptUpload_SelfService(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()
	v := seedVolunteerWithPendingAgreement(t, h, h.campusID)

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	mh := make(textproto.MIMEHeader)
	mh.Set("Content-Disposition", `form-data; name="document"; filename="termo.pdf"`)
	mh.Set("Content-Type", "application/pdf")
	part, err := w.CreatePart(mh)
	require.NoError(t, err)
	_, err = part.Write([]byte("%PDF-1.4 signed agreement"))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/volunteer-agreement/upload", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := h.doRequest(h.authedRequest(req, asVolunteerPerson(v)))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var status, method string
	var docPath *string
	var uploadedBy uuid.UUID
	err = h.pool.QueryRow(ctx, `
		SELECT status, signature_method, document_path, uploaded_by
		FROM volunteer_agreement WHERE person_id = $1
	`, v.personID).Scan(&status, &method, &docPath, &uploadedBy)
	require.NoError(t, err)
	assert.Equal(t, "ACCEPTED", status)
	assert.Equal(t, "MANUAL_UPLOAD", method)
	require.NotNil(t, docPath)
	assert.NotEmpty(t, *docPath)
	assert.Equal(t, v.appUserID, uploadedBy, "uploaded_by must be the app_user id, not the Keycloak subject")
}
