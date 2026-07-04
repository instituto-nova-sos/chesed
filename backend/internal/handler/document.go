package handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

// maxDocumentUploadBytes is the docs/19 upload ceiling (10MB); the request
// body limit adds headroom for the multipart envelope.
const (
	maxDocumentUploadBytes = 10 << 20
	multipartOverheadBytes = 1 << 20
	sniffLen               = 512
)

// allowedDocumentContent whitelists the magic-byte-sniffed content types
// accepted for upload (docs/19 §6 — the client-declared type is not trusted).
var allowedDocumentContent = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
}

// DocumentHandler handles HTTP requests for document attachments.
type DocumentHandler struct {
	svc *service.DocumentService
}

// NewDocumentHandler creates a new DocumentHandler.
func NewDocumentHandler(svc *service.DocumentService) *DocumentHandler {
	return &DocumentHandler{svc: svc}
}

// documentListResponse is the documented list envelope (docs/11).
type documentListResponse struct {
	Data []domain.Document `json:"data"`
}

// UploadForPerson handles POST /persons/{id}/documents.
func (h *DocumentHandler) UploadForPerson(w http.ResponseWriter, r *http.Request) {
	h.upload(w, r, h.svc.UploadForPerson)
}

// UploadForAttendance handles POST /attendances/{id}/documents.
func (h *DocumentHandler) UploadForAttendance(w http.ResponseWriter, r *http.Request) {
	h.upload(w, r, h.svc.UploadForAttendance)
}

// upload implements the shared multipart flow: body limit → parse → magic-byte
// sniff → service. The target function binds the parent entity (person or
// attendance) resolved from the {id} path parameter.
func (h *DocumentHandler) upload(w http.ResponseWriter, r *http.Request, target func(ctx context.Context, id uuid.UUID, in service.UploadDocumentInput) (*domain.Document, error)) {
	parentID, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid ID format")
		return
	}

	in, ok := h.parseUploadForm(w, r)
	if !ok {
		return
	}

	doc, err := target(r.Context(), parentID, in)
	if err != nil {
		h.writeDocumentError(w, r, err, "upload document")
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

// parseUploadForm enforces the size ceiling, extracts the file and fields,
// and sniffs the real content type. It writes the error response itself and
// returns ok=false on rejection.
func (h *DocumentHandler) parseUploadForm(w http.ResponseWriter, r *http.Request) (service.UploadDocumentInput, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxDocumentUploadBytes+multipartOverheadBytes))
	if err := r.ParseMultipartForm(maxDocumentUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "file too large or malformed multipart body")
		return service.UploadDocumentInput{}, false
	}

	file, header, err := r.FormFile("document")
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "document file is required")
		return service.UploadDocumentInput{}, false
	}

	if header.Size > maxDocumentUploadBytes {
		writeError(w, http.StatusBadRequest, "validation_error", "file exceeds the 10MB limit")
		return service.UploadDocumentInput{}, false
	}

	contentType, ok := sniffDocumentContent(file)
	if !ok {
		writeError(w, http.StatusBadRequest, "validation_error", "unsupported file content")
		return service.UploadDocumentInput{}, false
	}

	var description *string
	if v := r.FormValue("description"); v != "" {
		description = &v
	}

	return service.UploadDocumentInput{
		DocumentType: r.FormValue("document_type"),
		Description:  description,
		FileName:     header.Filename,
		ContentType:  contentType,
		Size:         header.Size,
		Content:      file,
	}, true
}

// sniffDocumentContent detects the file's real content type from its first
// bytes and rewinds the reader. The client-declared Content-Type header is
// deliberately ignored (docs/19, threat T13).
func sniffDocumentContent(file io.ReadSeeker) (string, bool) {
	head := make([]byte, sniffLen)
	n, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", false
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", false
	}
	// DetectContentType may append parameters (e.g. "; charset=utf-8").
	contentType := http.DetectContentType(head[:n])
	if i := strings.IndexByte(contentType, ';'); i >= 0 {
		contentType = strings.TrimSpace(contentType[:i])
	}
	return contentType, allowedDocumentContent[contentType]
}

// ListByPerson handles GET /persons/{id}/documents.
func (h *DocumentHandler) ListByPerson(w http.ResponseWriter, r *http.Request) {
	h.list(w, r, h.svc.ListByPerson)
}

// ListByAttendance handles GET /attendances/{id}/documents.
func (h *DocumentHandler) ListByAttendance(w http.ResponseWriter, r *http.Request) {
	h.list(w, r, h.svc.ListByAttendance)
}

func (h *DocumentHandler) list(w http.ResponseWriter, r *http.Request, target func(ctx context.Context, id uuid.UUID) ([]domain.Document, error)) {
	parentID, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid ID format")
		return
	}

	docs, err := target(r.Context(), parentID)
	if err != nil {
		h.writeDocumentError(w, r, err, "list documents")
		return
	}
	writeJSON(w, http.StatusOK, documentListResponse{Data: docs})
}

// Download handles GET /documents/{id}/download.
func (h *DocumentHandler) Download(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid document ID format")
		return
	}

	dl, err := h.svc.GetDownloadURL(r.Context(), id)
	if err != nil {
		h.writeDocumentError(w, r, err, "generate download URL")
		return
	}
	writeJSON(w, http.StatusOK, dl)
}

// writeDocumentError maps domain sentinel errors to the documented HTTP
// contract; messages stay generic (T3/T13 — no disclosure).
func (h *DocumentHandler) writeDocumentError(w http.ResponseWriter, r *http.Request, err error, action string) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "missing campus context")
	case errors.Is(err, domain.ErrValidation):
		writeError(w, http.StatusBadRequest, "validation_error", "invalid reference or field values")
	default:
		slog.ErrorContext(r.Context(), "documentHandler: failed", "action", action, "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to "+action)
	}
}
