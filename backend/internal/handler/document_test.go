package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

type mockDocumentRepo struct{ mock.Mock }

func (m *mockDocumentRepo) Create(ctx context.Context, d domain.Document) (*domain.Document, error) {
	args := m.Called(ctx, d)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Document), args.Error(1)
}

func (m *mockDocumentRepo) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Document, error) {
	args := m.Called(ctx, id, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Document), args.Error(1)
}

func (m *mockDocumentRepo) ListByPerson(ctx context.Context, personID, campusID uuid.UUID) ([]domain.Document, error) {
	args := m.Called(ctx, personID, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Document), args.Error(1)
}

func (m *mockDocumentRepo) ListByAttendance(ctx context.Context, attendanceID, campusID uuid.UUID) ([]domain.Document, error) {
	args := m.Called(ctx, attendanceID, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Document), args.Error(1)
}

type mockDocumentPersonRepo struct{ mock.Mock }

func (m *mockDocumentPersonRepo) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Person, error) {
	args := m.Called(ctx, id, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Person), args.Error(1)
}

type mockDocumentAttendanceRepo struct{ mock.Mock }

func (m *mockDocumentAttendanceRepo) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Attendance, error) {
	args := m.Called(ctx, id, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Attendance), args.Error(1)
}

type mockObjectStorage struct{ mock.Mock }

func (m *mockObjectStorage) Put(ctx context.Context, key, contentType string, size int64, r io.Reader) error {
	args := m.Called(ctx, key, contentType, size, r)
	return args.Error(0)
}

func (m *mockObjectStorage) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	args := m.Called(ctx, key, expiry)
	return args.String(0), args.Error(1)
}

func newDocumentTestHandler() (*DocumentHandler, *mockDocumentRepo, *mockDocumentPersonRepo, *mockObjectStorage, *mockAuditRepo) {
	repo := new(mockDocumentRepo)
	personRepo := new(mockDocumentPersonRepo)
	attendanceRepo := new(mockDocumentAttendanceRepo)
	store := new(mockObjectStorage)
	aRepo := new(mockAuditRepo)
	svc := service.NewDocumentService(repo, personRepo, attendanceRepo, store, service.NewAuditService(aRepo))
	return NewDocumentHandler(svc), repo, personRepo, store, aRepo
}

func documentRouter(h *DocumentHandler) chi.Router {
	r := chi.NewRouter()
	r.Post("/persons/{id}/documents", h.UploadForPerson)
	r.Get("/persons/{id}/documents", h.ListByPerson)
	r.Post("/attendances/{id}/documents", h.UploadForAttendance)
	r.Get("/attendances/{id}/documents", h.ListByAttendance)
	r.Get("/documents/{id}/download", h.Download)
	return r
}

func documentClaims(campusID uuid.UUID) auth.AuthClaims {
	return auth.AuthClaims{
		Subject:  uuid.New().String(),
		Email:    "tester@example.com",
		Roles:    []string{"SECRETARY"},
		CampusID: campusID,
	}
}

// multipartBody builds a multipart form with a "document" file part carrying
// the given bytes plus the document_type field.
func multipartBody(t *testing.T, fileName string, content []byte, docType string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("document", fileName)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.WriteField("document_type", docType))
	require.NoError(t, w.Close())
	return &buf, w.FormDataContentType()
}

func pdfBytes() []byte {
	return append([]byte("%PDF-1.7\n"), bytes.Repeat([]byte("a"), 64)...)
}

func doDocumentRequest(h *DocumentHandler, req *http.Request, campusID uuid.UUID) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req = req.WithContext(auth.NewContext(req.Context(), documentClaims(campusID)))
	documentRouter(h).ServeHTTP(rec, req)
	return rec
}

func TestDocumentHandler_UploadForPerson_Created(t *testing.T) {
	h, repo, personRepo, store, aRepo := newDocumentTestHandler()
	campusID := uuid.New()
	personID := uuid.New()

	personRepo.On("FindByID", mock.Anything, personID, campusID).
		Return(&domain.Person{ID: personID, CampusID: campusID}, nil).Once()
	store.On("Put", mock.Anything, mock.Anything, "application/pdf", mock.Anything, mock.Anything).
		Return(nil).Once()
	repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Document")).
		Return(&domain.Document{
			ID: uuid.New(), PersonID: personID, CampusID: campusID,
			DocumentType: "EXAM", FileName: "exame.pdf", FilePath: "documents/x/y/z.pdf",
		}, nil).Once()
	aRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil).Once()

	body, contentType := multipartBody(t, "exame.pdf", pdfBytes(), "EXAM")
	req := httptest.NewRequest(http.MethodPost, "/persons/"+personID.String()+"/documents", body)
	req.Header.Set("Content-Type", contentType)

	rec := doDocumentRequest(h, req, campusID)

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "exame.pdf", resp["file_name"])
	_, hasFilePath := resp["file_path"]
	assert.False(t, hasFilePath, "storage key must never leak in responses")
	repo.AssertExpectations(t)
	store.AssertExpectations(t)
}

func TestDocumentHandler_UploadForPerson_SpoofedContentTypeRejected(t *testing.T) {
	h, repo, personRepo, store, _ := newDocumentTestHandler()
	campusID := uuid.New()
	personID := uuid.New()

	personRepo.On("FindByID", mock.Anything, personID, campusID).
		Return(&domain.Person{ID: personID, CampusID: campusID}, nil).Maybe()

	// Declared as a PDF by filename, but the bytes are plain text: the
	// magic-byte sniff must win over any client-declared type (docs/19).
	body, contentType := multipartBody(t, "notes.pdf", []byte("just some text, no pdf magic"), "EXAM")
	req := httptest.NewRequest(http.MethodPost, "/persons/"+personID.String()+"/documents", body)
	req.Header.Set("Content-Type", contentType)

	rec := doDocumentRequest(h, req, campusID)

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "validation_error")
	store.AssertNotCalled(t, "Put", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestDocumentHandler_UploadForPerson_MissingFileRejected(t *testing.T) {
	h, _, _, _, _ := newDocumentTestHandler()
	campusID := uuid.New()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	require.NoError(t, w.WriteField("document_type", "EXAM"))
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, "/persons/"+uuid.New().String()+"/documents", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())

	rec := doDocumentRequest(h, req, campusID)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}

func TestDocumentHandler_UploadForPerson_UnknownDocumentTypeRejected(t *testing.T) {
	h, repo, personRepo, store, _ := newDocumentTestHandler()
	campusID := uuid.New()
	personID := uuid.New()

	personRepo.On("FindByID", mock.Anything, personID, campusID).
		Return(&domain.Person{ID: personID, CampusID: campusID}, nil).Maybe()

	body, contentType := multipartBody(t, "exame.pdf", pdfBytes(), "SELFIE")
	req := httptest.NewRequest(http.MethodPost, "/persons/"+personID.String()+"/documents", body)
	req.Header.Set("Content-Type", contentType)

	rec := doDocumentRequest(h, req, campusID)

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "validation_error")
	store.AssertNotCalled(t, "Put", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestDocumentHandler_UploadForPerson_OversizeRejected(t *testing.T) {
	h, repo, _, store, _ := newDocumentTestHandler()
	campusID := uuid.New()
	personID := uuid.New()

	oversize := append([]byte("%PDF-1.7\n"), bytes.Repeat([]byte("a"), 11<<20)...)
	body, contentType := multipartBody(t, "huge.pdf", oversize, "EXAM")
	req := httptest.NewRequest(http.MethodPost, "/persons/"+personID.String()+"/documents", body)
	req.Header.Set("Content-Type", contentType)

	rec := doDocumentRequest(h, req, campusID)

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	store.AssertNotCalled(t, "Put", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestDocumentHandler_Download_ReturnsPresignedURL(t *testing.T) {
	h, repo, _, store, _ := newDocumentTestHandler()
	campusID := uuid.New()
	docID := uuid.New()

	repo.On("FindByID", mock.Anything, docID, campusID).
		Return(&domain.Document{ID: docID, CampusID: campusID, FilePath: "documents/k.pdf"}, nil).Once()
	store.On("PresignGet", mock.Anything, "documents/k.pdf", 15*time.Minute).
		Return("https://storage.example/presigned", nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/documents/"+docID.String()+"/download", nil)
	rec := doDocumentRequest(h, req, campusID)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var resp struct {
		URL       string    `json:"url"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "https://storage.example/presigned", resp.URL)
	assert.WithinDuration(t, time.Now().Add(15*time.Minute), resp.ExpiresAt, 10*time.Second)
}

func TestDocumentHandler_Download_NotFound(t *testing.T) {
	h, repo, _, _, _ := newDocumentTestHandler()
	campusID := uuid.New()
	docID := uuid.New()

	repo.On("FindByID", mock.Anything, docID, campusID).Return(nil, domain.ErrNotFound).Once()

	req := httptest.NewRequest(http.MethodGet, "/documents/"+docID.String()+"/download", nil)
	rec := doDocumentRequest(h, req, campusID)

	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
}

func TestDocumentHandler_ListByPerson_Envelope(t *testing.T) {
	h, repo, personRepo, _, _ := newDocumentTestHandler()
	campusID := uuid.New()
	personID := uuid.New()

	personRepo.On("FindByID", mock.Anything, personID, campusID).
		Return(&domain.Person{ID: personID, CampusID: campusID}, nil).Once()
	repo.On("ListByPerson", mock.Anything, personID, campusID).
		Return([]domain.Document{{ID: uuid.New(), PersonID: personID, FileName: "a.pdf"}}, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/persons/"+personID.String()+"/documents", nil)
	rec := doDocumentRequest(h, req, campusID)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var resp struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "a.pdf", resp.Data[0]["file_name"])
}
