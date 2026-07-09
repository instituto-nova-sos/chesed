package handler

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newAgreementHandler(t *testing.T) (*VolunteerAgreementHandler, *mockAgreementRepo) {
	t.Helper()
	agRepo := new(mockAgreementRepo)
	roleRepo := new(mockPersonRoleRepo)
	auditRepo := new(mockAuditRepo)
	auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil).Maybe()
	auditSvc := service.NewAuditService(auditRepo)
	svc := service.NewVolunteerAgreementService(agRepo, roleRepo, auditSvc)
	return NewVolunteerAgreementHandler(svc, t.TempDir()), agRepo
}

func agreementCtx() context.Context {
	claims := auth.AuthClaims{
		Subject:  uuid.New().String(),
		Email:    "volunteer@chesed.test",
		Roles:    []string{"VOLUNTEER"},
		CampusID: uuid.New(),
		PersonID: uuid.New(),
	}
	return auth.NewContext(context.Background(), claims)
}

func TestVolunteerAgreementHandler_Accept(t *testing.T) {
	t.Run("rejects missing signature with 400", func(t *testing.T) {
		h, _ := newAgreementHandler(t)
		req := httptest.NewRequest(http.MethodPost, "/volunteer-agreement/accept",
			strings.NewReader(`{"signature_data":""}`)).WithContext(agreementCtx())
		rec := httptest.NewRecorder()

		h.Accept(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("accepts with signature returns 200", func(t *testing.T) {
		h, agRepo := newAgreementHandler(t)
		ctx := agreementCtx()
		personID := auth.ClaimsFromContext(ctx).PersonID
		pendingID := uuid.New()

		agRepo.On("HasAcceptedAgreement", mock.Anything, personID).Return(false, nil)
		agRepo.On("FindPendingByPersonID", mock.Anything, personID).
			Return(&domain.VolunteerAgreement{ID: pendingID, PersonID: personID, Status: domain.AgreementPending}, nil)
		agRepo.On("AcceptDigital", mock.Anything, pendingID, mock.AnythingOfType("uuid.UUID"),
			mock.Anything, mock.Anything, "data:image/png;base64,SIG").
			Return(&domain.VolunteerAgreement{ID: pendingID, PersonID: personID, Status: domain.AgreementAccepted}, nil)

		req := httptest.NewRequest(http.MethodPost, "/volunteer-agreement/accept",
			strings.NewReader(`{"signature_data":"data:image/png;base64,SIG"}`)).WithContext(ctx)
		rec := httptest.NewRecorder()

		h.Accept(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		agRepo.AssertExpectations(t)
	})
}

// agreementMultipart builds a multipart body whose part carries an explicit Content-Type
// header, so the handler's MIME whitelist can be exercised (including the reject case).
func agreementMultipart(t *testing.T, contentType, filename, content string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="document"; filename="`+filename+`"`)
	h.Set("Content-Type", contentType)
	part, err := w.CreatePart(h)
	require.NoError(t, err)
	_, err = part.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return body, w.FormDataContentType()
}

func TestVolunteerAgreementHandler_AcceptUpload(t *testing.T) {
	t.Run("self-service upload of PDF returns 200", func(t *testing.T) {
		h, agRepo := newAgreementHandler(t)
		ctx := agreementCtx()
		personID := auth.ClaimsFromContext(ctx).PersonID
		pendingID := uuid.New()

		agRepo.On("FindPendingByPersonID", mock.Anything, personID).
			Return(&domain.VolunteerAgreement{ID: pendingID, PersonID: personID, Status: domain.AgreementPending}, nil)
		agRepo.On("AcceptManualUpload", mock.Anything, pendingID, mock.AnythingOfType("string"), mock.AnythingOfType("uuid.UUID")).
			Return(&domain.VolunteerAgreement{ID: pendingID, PersonID: personID, Status: domain.AgreementAccepted}, nil)

		body, ct := agreementMultipart(t, "application/pdf", "termo.pdf", "%PDF-1.4 fake")
		req := httptest.NewRequest(http.MethodPost, "/volunteer-agreement/upload", body).WithContext(ctx)
		req.Header.Set("Content-Type", ct)
		rec := httptest.NewRecorder()

		h.AcceptUpload(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		agRepo.AssertExpectations(t)
	})

	t.Run("rejects invalid MIME type with 400", func(t *testing.T) {
		h, _ := newAgreementHandler(t)
		body, ct := agreementMultipart(t, "text/plain", "termo.txt", "not allowed")
		req := httptest.NewRequest(http.MethodPost, "/volunteer-agreement/upload", body).WithContext(agreementCtx())
		req.Header.Set("Content-Type", ct)
		rec := httptest.NewRecorder()

		h.AcceptUpload(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("rejects request without linked person with 400", func(t *testing.T) {
		h, _ := newAgreementHandler(t)
		claims := auth.AuthClaims{Subject: uuid.New().String(), Roles: []string{"VOLUNTEER"}, CampusID: uuid.New()}
		ctx := auth.NewContext(context.Background(), claims)
		body, ct := agreementMultipart(t, "application/pdf", "termo.pdf", "%PDF-1.4 fake")
		req := httptest.NewRequest(http.MethodPost, "/volunteer-agreement/upload", body).WithContext(ctx)
		req.Header.Set("Content-Type", ct)
		rec := httptest.NewRecorder()

		h.AcceptUpload(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
