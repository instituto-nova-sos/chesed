package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// --- Mocks --------------------------------------------------------------------

type MockDocumentRepo struct{ mock.Mock }

func (m *MockDocumentRepo) Create(ctx context.Context, d domain.Document) (*domain.Document, error) {
	args := m.Called(ctx, d)
	if v, ok := args.Get(0).(*domain.Document); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDocumentRepo) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Document, error) {
	args := m.Called(ctx, id, campusID)
	if v, ok := args.Get(0).(*domain.Document); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDocumentRepo) ListByPerson(ctx context.Context, personID, campusID uuid.UUID) ([]domain.Document, error) {
	args := m.Called(ctx, personID, campusID)
	if v, ok := args.Get(0).([]domain.Document); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDocumentRepo) ListByAttendance(ctx context.Context, attendanceID, campusID uuid.UUID) ([]domain.Document, error) {
	args := m.Called(ctx, attendanceID, campusID)
	if v, ok := args.Get(0).([]domain.Document); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

type MockDocumentPersonRepo struct{ mock.Mock }

func (m *MockDocumentPersonRepo) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Person, error) {
	args := m.Called(ctx, id, campusID)
	if v, ok := args.Get(0).(*domain.Person); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

type MockDocumentAttendanceRepo struct{ mock.Mock }

func (m *MockDocumentAttendanceRepo) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Attendance, error) {
	args := m.Called(ctx, id, campusID)
	if v, ok := args.Get(0).(*domain.Attendance); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

type MockObjectStorage struct{ mock.Mock }

func (m *MockObjectStorage) Put(ctx context.Context, key, contentType string, size int64, r io.Reader) error {
	args := m.Called(ctx, key, contentType, size, r)
	return args.Error(0)
}

func (m *MockObjectStorage) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	args := m.Called(ctx, key, expiry)
	return args.String(0), args.Error(1)
}

// --- Helpers ------------------------------------------------------------------

func newDocumentService(t *testing.T) (*DocumentService, *MockDocumentRepo, *MockDocumentPersonRepo, *MockDocumentAttendanceRepo, *MockObjectStorage, *MockAuditRepository) {
	t.Helper()
	repo := new(MockDocumentRepo)
	personRepo := new(MockDocumentPersonRepo)
	attendanceRepo := new(MockDocumentAttendanceRepo)
	store := new(MockObjectStorage)
	auditRepo := new(MockAuditRepository)
	svc := NewDocumentService(repo, personRepo, attendanceRepo, store, NewAuditService(auditRepo))
	return svc, repo, personRepo, attendanceRepo, store, auditRepo
}

func validUploadInput() UploadDocumentInput {
	return UploadDocumentInput{
		DocumentType: "PROOF_OF_RESIDENCE",
		FileName:     "comprovante.pdf",
		ContentType:  "application/pdf",
		Size:         42,
		Content:      bytes.NewReader(bytes.Repeat([]byte("x"), 42)),
	}
}

// --- UploadForPerson ------------------------------------------------------------

func TestDocumentService_UploadForPerson_HappyPath(t *testing.T) {
	svc, repo, personRepo, _, store, auditRepo := newDocumentService(t)
	campusID := uuid.New()
	personID := uuid.New()
	ctx := ctxWithCampus(t, campusID)

	personRepo.On("FindByID", ctx, personID, campusID).
		Return(&domain.Person{ID: personID, CampusID: campusID}, nil).Once()
	store.On("Put", ctx,
		mock.MatchedBy(func(key string) bool {
			return strings.HasPrefix(key, "documents/"+campusID.String()+"/"+personID.String()+"/") &&
				strings.HasSuffix(key, ".pdf")
		}),
		"application/pdf", int64(42), mock.Anything,
	).Return(nil).Once()
	repo.On("Create", ctx, mock.MatchedBy(func(d domain.Document) bool {
		return d.PersonID == personID && d.CampusID == campusID &&
			d.AttendanceID == nil && d.DocumentType == "PROOF_OF_RESIDENCE" &&
			d.FileName == "comprovante.pdf" && strings.HasSuffix(d.FilePath, ".pdf")
	})).Return(&domain.Document{ID: uuid.New(), PersonID: personID, CampusID: campusID}, nil).Once()
	auditRepo.On("Create", ctx, mock.AnythingOfType("domain.AuditLog")).Return(nil).Once()

	created, err := svc.UploadForPerson(ctx, personID, validUploadInput())
	require.NoError(t, err)
	require.NotNil(t, created)
	repo.AssertExpectations(t)
	store.AssertExpectations(t)
	auditRepo.AssertExpectations(t)
}

func TestDocumentService_UploadForPerson_CrossCampusNotFound(t *testing.T) {
	svc, repo, personRepo, _, store, _ := newDocumentService(t)
	campusID := uuid.New()
	personID := uuid.New()
	ctx := ctxWithCampus(t, campusID)

	personRepo.On("FindByID", ctx, personID, campusID).Return(nil, domain.ErrNotFound).Once()

	_, err := svc.UploadForPerson(ctx, personID, validUploadInput())
	assert.True(t, errors.Is(err, domain.ErrNotFound), "path reference must map to not found, got %v", err)
	store.AssertNotCalled(t, "Put", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestDocumentService_UploadForPerson_InvalidTypeRejected(t *testing.T) {
	svc, _, _, _, store, _ := newDocumentService(t)
	ctx := ctxWithCampus(t, uuid.New())

	in := validUploadInput()
	in.DocumentType = "SELFIE"

	_, err := svc.UploadForPerson(ctx, uuid.New(), in)
	require.Error(t, err)
	store.AssertNotCalled(t, "Put", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestDocumentService_UploadForPerson_DisallowedContentTypeRejected(t *testing.T) {
	svc, _, personRepo, _, store, _ := newDocumentService(t)
	campusID := uuid.New()
	personID := uuid.New()
	ctx := ctxWithCampus(t, campusID)

	personRepo.On("FindByID", ctx, personID, campusID).
		Return(&domain.Person{ID: personID, CampusID: campusID}, nil).Maybe()

	in := validUploadInput()
	in.ContentType = "text/plain; charset=utf-8"

	_, err := svc.UploadForPerson(ctx, personID, in)
	assert.True(t, errors.Is(err, domain.ErrValidation), "expected ErrValidation, got %v", err)
	store.AssertNotCalled(t, "Put", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestDocumentService_UploadForPerson_StorageFailureAbortsInsert(t *testing.T) {
	svc, repo, personRepo, _, store, _ := newDocumentService(t)
	campusID := uuid.New()
	personID := uuid.New()
	ctx := ctxWithCampus(t, campusID)

	personRepo.On("FindByID", ctx, personID, campusID).
		Return(&domain.Person{ID: personID, CampusID: campusID}, nil).Once()
	store.On("Put", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("connection refused")).Once()

	_, err := svc.UploadForPerson(ctx, personID, validUploadInput())
	require.Error(t, err)
	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// --- UploadForAttendance ---------------------------------------------------------

func TestDocumentService_UploadForAttendance_HappyPath(t *testing.T) {
	svc, repo, _, attendanceRepo, store, auditRepo := newDocumentService(t)
	campusID := uuid.New()
	personID := uuid.New()
	attendanceID := uuid.New()
	ctx := ctxWithCampus(t, campusID)

	attendanceRepo.On("FindByID", ctx, attendanceID, campusID).
		Return(&domain.Attendance{ID: attendanceID, PersonID: personID, CampusID: campusID}, nil).Once()
	store.On("Put", ctx, mock.Anything, "application/pdf", int64(42), mock.Anything).Return(nil).Once()
	repo.On("Create", ctx, mock.MatchedBy(func(d domain.Document) bool {
		return d.PersonID == personID && d.AttendanceID != nil && *d.AttendanceID == attendanceID
	})).Return(&domain.Document{ID: uuid.New(), PersonID: personID, CampusID: campusID}, nil).Once()
	auditRepo.On("Create", ctx, mock.AnythingOfType("domain.AuditLog")).Return(nil).Once()

	created, err := svc.UploadForAttendance(ctx, attendanceID, validUploadInput())
	require.NoError(t, err)
	require.NotNil(t, created)
	repo.AssertExpectations(t)
}

func TestDocumentService_UploadForAttendance_CrossCampusNotFound(t *testing.T) {
	svc, repo, _, attendanceRepo, store, _ := newDocumentService(t)
	campusID := uuid.New()
	attendanceID := uuid.New()
	ctx := ctxWithCampus(t, campusID)

	attendanceRepo.On("FindByID", ctx, attendanceID, campusID).Return(nil, domain.ErrNotFound).Once()

	_, err := svc.UploadForAttendance(ctx, attendanceID, validUploadInput())
	assert.True(t, errors.Is(err, domain.ErrNotFound))
	store.AssertNotCalled(t, "Put", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// --- Listing ---------------------------------------------------------------------

func TestDocumentService_ListByPerson_ResolvesPersonFirst(t *testing.T) {
	svc, repo, personRepo, _, _, _ := newDocumentService(t)
	campusID := uuid.New()
	personID := uuid.New()
	ctx := ctxWithCampus(t, campusID)

	personRepo.On("FindByID", ctx, personID, campusID).Return(nil, domain.ErrNotFound).Once()

	_, err := svc.ListByPerson(ctx, personID)
	assert.True(t, errors.Is(err, domain.ErrNotFound))
	repo.AssertNotCalled(t, "ListByPerson", mock.Anything, mock.Anything, mock.Anything)
}

func TestDocumentService_ListByPerson_ReturnsDocuments(t *testing.T) {
	svc, repo, personRepo, _, _, _ := newDocumentService(t)
	campusID := uuid.New()
	personID := uuid.New()
	ctx := ctxWithCampus(t, campusID)

	personRepo.On("FindByID", ctx, personID, campusID).
		Return(&domain.Person{ID: personID, CampusID: campusID}, nil).Once()
	repo.On("ListByPerson", ctx, personID, campusID).
		Return([]domain.Document{{ID: uuid.New(), PersonID: personID}}, nil).Once()

	docs, err := svc.ListByPerson(ctx, personID)
	require.NoError(t, err)
	assert.Len(t, docs, 1)
}

// --- Download --------------------------------------------------------------------

func TestDocumentService_GetDownloadURL_HappyPath(t *testing.T) {
	svc, repo, _, _, store, _ := newDocumentService(t)
	campusID := uuid.New()
	docID := uuid.New()
	ctx := ctxWithCampus(t, campusID)

	repo.On("FindByID", ctx, docID, campusID).
		Return(&domain.Document{ID: docID, CampusID: campusID, FilePath: "documents/a/b/c.pdf"}, nil).Once()
	store.On("PresignGet", ctx, "documents/a/b/c.pdf", 15*time.Minute).
		Return("https://storage.example/presigned", nil).Once()

	before := time.Now()
	dl, err := svc.GetDownloadURL(ctx, docID)
	require.NoError(t, err)
	assert.Equal(t, "https://storage.example/presigned", dl.URL)
	assert.WithinDuration(t, before.Add(15*time.Minute), dl.ExpiresAt, 5*time.Second)
}

func TestDocumentService_GetDownloadURL_NotFound(t *testing.T) {
	svc, repo, _, _, store, _ := newDocumentService(t)
	campusID := uuid.New()
	docID := uuid.New()
	ctx := ctxWithCampus(t, campusID)

	repo.On("FindByID", ctx, docID, campusID).Return(nil, domain.ErrNotFound).Once()

	_, err := svc.GetDownloadURL(ctx, docID)
	assert.True(t, errors.Is(err, domain.ErrNotFound))
	store.AssertNotCalled(t, "PresignGet", mock.Anything, mock.Anything, mock.Anything)
}
