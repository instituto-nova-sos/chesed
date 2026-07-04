//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcminio "github.com/testcontainers/testcontainers-go/modules/minio"

	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/service"
	"github.com/instituto-nova-sos/chesed/internal/storage"
)

// startDocumentStorage boots a MinIO container and returns a real
// ObjectStorage implementation bound to it (same approach as storage_test.go).
func startDocumentStorage(t *testing.T) service.ObjectStorage {
	t.Helper()
	ctx := context.Background()

	minioC, err := tcminio.Run(ctx, "minio/minio:latest")
	require.NoError(t, err, "start minio container")
	t.Cleanup(func() { _ = minioC.Terminate(context.Background()) })

	endpoint, err := minioC.ConnectionString(ctx)
	require.NoError(t, err)

	store, err := storage.NewMinioStorage(ctx, storage.Config{
		Endpoint:  endpoint,
		Bucket:    "chesed-docs-test",
		AccessKey: minioC.Username,
		SecretKey: minioC.Password,
		UseSSL:    false,
	})
	require.NoError(t, err)
	return store
}

func documentMultipart(t *testing.T, fileName string, content []byte, docType string) (*bytes.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("document", fileName)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.WriteField("document_type", docType))
	require.NoError(t, w.Close())
	return bytes.NewReader(buf.Bytes()), w.FormDataContentType()
}

func integrationPDF() []byte {
	return append([]byte("%PDF-1.7\n"), bytes.Repeat([]byte("chesed"), 32)...)
}

func postDocument(t *testing.T, h *testHarness, path, fileName string, content []byte, docType string, opts ...func(*auth.AuthClaims)) *httptest.ResponseRecorder {
	t.Helper()
	body, contentType := documentMultipart(t, fileName, content, docType)
	req := newRequest(t, http.MethodPost, path, body)
	req.Header.Set("Content-Type", contentType)
	return h.doRequest(h.authedRequest(req, opts...))
}

// TestDocumentUpload_PersonHappyPath drives multipart HTTP → magic-byte sniff
// → object storage → Postgres, then proves the presigned download serves the
// exact stored bytes (S08.2 happy path with DB, audit, and storage asserts).
func TestDocumentUpload_PersonHappyPath(t *testing.T) {
	store := startDocumentStorage(t)
	h := freshHarnessWithStorage(t, store)
	ctx := context.Background()

	personID := seedPerson(t, h, h.campusID, "Doc Person", "40230340470")
	content := integrationPDF()

	rec := postDocument(t, h,
		"/api/v1/persons/"+personID.String()+"/documents",
		"comprovante.pdf", content, "PROOF_OF_RESIDENCE", asSecretary)

	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
	var created struct {
		ID       uuid.UUID `json:"id"`
		FileName string    `json:"file_name"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, "comprovante.pdf", created.FileName)
	assert.NotContains(t, rec.Body.String(), "file_path", "storage key must not leak")

	// DB row is campus-scoped with metadata persisted.
	var (
		dbCampus   uuid.UUID
		dbType     string
		dbFileName string
		dbFilePath string
	)
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT campus_id, document_type, file_name, file_path FROM document WHERE id = $1`,
		created.ID,
	).Scan(&dbCampus, &dbType, &dbFileName, &dbFilePath))
	assert.Equal(t, h.campusID, dbCampus)
	assert.Equal(t, "PROOF_OF_RESIDENCE", dbType)
	assert.Equal(t, "comprovante.pdf", dbFileName)
	assert.Contains(t, dbFilePath, "documents/"+h.campusID.String()+"/"+personID.String()+"/")

	// Audit CREATE entry written.
	var auditCount int
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_log WHERE entity_type = 'document' AND entity_id = $1 AND action_type = 'CREATE'`,
		created.ID,
	).Scan(&auditCount))
	assert.Equal(t, 1, auditCount)

	// The object really exists in MinIO: the download endpoint's presigned URL
	// serves the exact uploaded bytes.
	dlRec := h.doRequest(h.authedRequest(
		getRequest(t, "/api/v1/documents/"+created.ID.String()+"/download"),
		asProfessional))
	require.Equal(t, http.StatusOK, dlRec.Code, "body=%s", dlRec.Body.String())
	var dl struct {
		URL string `json:"url"`
	}
	require.NoError(t, json.Unmarshal(dlRec.Body.Bytes(), &dl))
	resp, err := http.Get(dl.URL)
	require.NoError(t, err)
	gotBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, content, gotBytes)
}

// TestDocumentUpload_AttendanceHappyPath links the document to an attendance
// and its person through the real stack.
func TestDocumentUpload_AttendanceHappyPath(t *testing.T) {
	store := startDocumentStorage(t)
	h := freshHarnessWithStorage(t, store)
	ctx := context.Background()

	personID := seedPerson(t, h, h.campusID, "Doc Att Person", "40230340471")
	professionalID := seedPerson(t, h, h.campusID, "Doc Att Pro", "40230340472")
	var serviceTypeID uuid.UUID
	require.NoError(t, h.pool.QueryRow(ctx, `SELECT id FROM service_type LIMIT 1`).Scan(&serviceTypeID))
	attendanceID := createAttendanceViaAPI(t, h, personID, professionalID, serviceTypeID)

	rec := postDocument(t, h,
		"/api/v1/attendances/"+attendanceID.String()+"/documents",
		"exame.pdf", integrationPDF(), "EXAM", asProfessional)

	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())

	var dbPerson, dbAttendance uuid.UUID
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT person_id, attendance_id FROM document WHERE attendance_id = $1`, attendanceID,
	).Scan(&dbPerson, &dbAttendance))
	assert.Equal(t, personID, dbPerson)
	assert.Equal(t, attendanceID, dbAttendance)

	// The attendance list endpoint returns it.
	listRec := h.doRequest(h.authedRequest(
		getRequest(t, "/api/v1/attendances/"+attendanceID.String()+"/documents"),
		asProfessional))
	require.Equal(t, http.StatusOK, listRec.Code)
	var list struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &list))
	assert.Len(t, list.Data, 1)
}

// TestDocumentUpload_SpoofedContentRejected proves the magic-byte gate (T13):
// a text payload with a .pdf name and pdf-declared part type persists nothing
// anywhere.
func TestDocumentUpload_SpoofedContentRejected(t *testing.T) {
	store := startDocumentStorage(t)
	h := freshHarnessWithStorage(t, store)
	ctx := context.Background()

	personID := seedPerson(t, h, h.campusID, "Doc Spoof Person", "40230340473")

	rec := postDocument(t, h,
		"/api/v1/persons/"+personID.String()+"/documents",
		"malicious.pdf", []byte("#!/bin/sh\necho pwned"), "OTHER", asSecretary)

	require.Equal(t, http.StatusBadRequest, rec.Code, "body=%s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), "validation_error")

	var count int
	require.NoError(t, h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM document`).Scan(&count))
	assert.Equal(t, 0, count, "no document row may persist for rejected content")
}

// TestDocumentUpload_UnknownTypeRejected covers the document_type enum gate.
func TestDocumentUpload_UnknownTypeRejected(t *testing.T) {
	store := startDocumentStorage(t)
	h := freshHarnessWithStorage(t, store)

	personID := seedPerson(t, h, h.campusID, "Doc Type Person", "40230340474")

	rec := postDocument(t, h,
		"/api/v1/persons/"+personID.String()+"/documents",
		"doc.pdf", integrationPDF(), "SELFIE", asSecretary)

	require.Equal(t, http.StatusBadRequest, rec.Code, "body=%s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), "validation_error")
}

// TestDocumentUpload_CrossCampusPersonNotFound proves the campus boundary on
// the person path reference (404, no disclosure).
func TestDocumentUpload_CrossCampusPersonNotFound(t *testing.T) {
	store := startDocumentStorage(t)
	h := freshHarnessWithStorage(t, store)

	secondCampus := seedSecondCampus(t, h)
	foreignPerson := seedPerson(t, h, secondCampus, "Foreign Doc Person", "40230340475")

	rec := postDocument(t, h,
		"/api/v1/persons/"+foreignPerson.String()+"/documents",
		"doc.pdf", integrationPDF(), "OTHER", asSecretary)

	require.Equal(t, http.StatusNotFound, rec.Code, "body=%s", rec.Body.String())
}

// TestDocumentRBAC covers the role boundaries of the document surface:
// volunteers are fully denied (audited), secretaries may upload person
// documents but not attendance documents.
func TestDocumentRBAC(t *testing.T) {
	store := startDocumentStorage(t)
	h := freshHarnessWithStorage(t, store)
	ctx := context.Background()

	personID := seedPerson(t, h, h.campusID, "Doc RBAC Person", "40230340476")

	// VOLUNTEER (default claims) → 403 + ACCESS_DENIED audit.
	rec := postDocument(t, h,
		"/api/v1/persons/"+personID.String()+"/documents",
		"doc.pdf", integrationPDF(), "OTHER")
	require.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())

	var denied int
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_log WHERE action_type = 'ACCESS_DENIED'`,
	).Scan(&denied))
	assert.GreaterOrEqual(t, denied, 1, "volunteer denial must be audited")

	// SECRETARY on the attendance surface → 403 (Professional+ only).
	rec = postDocument(t, h,
		"/api/v1/attendances/"+uuid.New().String()+"/documents",
		"doc.pdf", integrationPDF(), "EXAM", asSecretary)
	require.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
}

// TestDocumentList_CrossCampusNotFound proves the campus boundary on both
// list GETs: a person or attendance of another campus yields 404 with no
// existence disclosure (review remediation — mandate gap).
func TestDocumentList_CrossCampusNotFound(t *testing.T) {
	store := startDocumentStorage(t)
	h := freshHarnessWithStorage(t, store)
	ctx := context.Background()

	personID := seedPerson(t, h, h.campusID, "Doc List Person", "40230340478")
	professionalID := seedPerson(t, h, h.campusID, "Doc List Pro", "40230340479")
	var serviceTypeID uuid.UUID
	require.NoError(t, h.pool.QueryRow(ctx, `SELECT id FROM service_type LIMIT 1`).Scan(&serviceTypeID))
	attendanceID := createAttendanceViaAPI(t, h, personID, professionalID, serviceTypeID)

	secondCampus := seedSecondCampus(t, h)
	asForeignCampus := func(c *auth.AuthClaims) { c.CampusID = secondCampus }

	rec := h.doRequest(h.authedRequest(
		getRequest(t, "/api/v1/persons/"+personID.String()+"/documents"),
		asProfessional, asForeignCampus))
	require.Equal(t, http.StatusNotFound, rec.Code, "person documents list must be campus-scoped; body=%s", rec.Body.String())

	rec = h.doRequest(h.authedRequest(
		getRequest(t, "/api/v1/attendances/"+attendanceID.String()+"/documents"),
		asProfessional, asForeignCampus))
	require.Equal(t, http.StatusNotFound, rec.Code, "attendance documents list must be campus-scoped; body=%s", rec.Body.String())
}

// TestDocumentDownload_CrossCampusNotFound proves the campus boundary on the
// download endpoint.
func TestDocumentDownload_CrossCampusNotFound(t *testing.T) {
	store := startDocumentStorage(t)
	h := freshHarnessWithStorage(t, store)

	personID := seedPerson(t, h, h.campusID, "Doc DL Person", "40230340477")
	rec := postDocument(t, h,
		"/api/v1/persons/"+personID.String()+"/documents",
		"doc.pdf", integrationPDF(), "OTHER", asSecretary)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	secondCampus := seedSecondCampus(t, h)
	dlRec := h.doRequest(h.authedRequest(
		getRequest(t, "/api/v1/documents/"+created.ID.String()+"/download"),
		asProfessional,
		func(c *auth.AuthClaims) { c.CampusID = secondCampus }))
	require.Equal(t, http.StatusNotFound, dlRec.Code, "body=%s", dlRec.Body.String())
}
