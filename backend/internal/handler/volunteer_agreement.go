package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

const maxUploadSize = 10 << 20 // 10 MB

var allowedMimeTypes = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
}

// VolunteerAgreementHandler handles HTTP requests for volunteer agreements.
type VolunteerAgreementHandler struct {
	svc       *service.VolunteerAgreementService
	uploadDir string
}

// NewVolunteerAgreementHandler creates a new VolunteerAgreementHandler.
func NewVolunteerAgreementHandler(svc *service.VolunteerAgreementService, uploadDir string) *VolunteerAgreementHandler {
	return &VolunteerAgreementHandler{svc: svc, uploadDir: uploadDir}
}

// GetText handles GET /volunteer-agreement/text.
func (h *VolunteerAgreementHandler) GetText(w http.ResponseWriter, r *http.Request) {
	version := r.URL.Query().Get("version")
	text, resolvedVersion := h.svc.GetAgreementText(version)

	writeJSON(w, http.StatusOK, map[string]string{
		"text":    text,
		"version": resolvedVersion,
	})
}

// Accept handles POST /volunteer-agreement/accept.
func (h *VolunteerAgreementHandler) Accept(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims.PersonID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "no_person", "user has no linked person")
		return
	}

	var body struct {
		SignatureData string `json:"signature_data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	ip := extractIP(r)
	userAgent := r.UserAgent()

	agreement, err := h.svc.AcceptDigital(r.Context(), claims.PersonID, ip, userAgent, body.SignatureData)
	if err != nil {
		if errors.Is(err, domain.ErrValidation) {
			writeError(w, http.StatusBadRequest, "validation", "signature is required")
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "no pending agreement found")
			return
		}
		if errors.Is(err, domain.ErrAgreementExists) {
			writeError(w, http.StatusConflict, "already_accepted", "agreement already accepted")
			return
		}
		slog.ErrorContext(r.Context(), "agreementHandler.Accept: failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to accept agreement")
		return
	}

	writeJSON(w, http.StatusOK, agreement)
}

// Reject handles POST /volunteer-agreement/reject.
func (h *VolunteerAgreementHandler) Reject(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims.PersonID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "no_person", "user has no linked person")
		return
	}

	var body struct {
		Reason *string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	agreement, err := h.svc.Reject(r.Context(), claims.PersonID, body.Reason)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "no pending agreement found")
			return
		}
		slog.ErrorContext(r.Context(), "agreementHandler.Reject: failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to reject agreement")
		return
	}

	writeJSON(w, http.StatusOK, agreement)
}

// GetPersonAgreement handles GET /persons/{id}/agreement.
func (h *VolunteerAgreementHandler) GetPersonAgreement(w http.ResponseWriter, r *http.Request) {
	personID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid person ID format")
		return
	}

	agreements, err := h.svc.GetByPersonID(r.Context(), personID)
	if err != nil {
		slog.ErrorContext(r.Context(), "agreementHandler.GetPersonAgreement: failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get agreements")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"agreements": agreements})
}

// Upload handles POST /persons/{id}/agreement/upload (coordinator uploads for a person).
func (h *VolunteerAgreementHandler) Upload(w http.ResponseWriter, r *http.Request) {
	personID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid person ID format")
		return
	}
	h.handleUpload(w, r, personID)
}

// AcceptUpload handles POST /volunteer-agreement/upload (volunteer uploads their own
// signed agreement). The person is taken from the token, never from the request, so a
// volunteer can only submit their own agreement.
func (h *VolunteerAgreementHandler) AcceptUpload(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims.PersonID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "no_person", "user has no linked person")
		return
	}
	h.handleUpload(w, r, claims.PersonID)
}

// handleUpload parses a multipart agreement document, persists the file, and marks the
// agreement ACCEPTED via manual upload. Shared by the coordinator and self-service paths.
func (h *VolunteerAgreementHandler) handleUpload(w http.ResponseWriter, r *http.Request, personID uuid.UUID) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "file_too_large", "file exceeds 10MB limit")
		return
	}

	file, header, err := r.FormFile("document")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing_file", "document file is required")
		return
	}
	defer file.Close()

	// Validate MIME type
	contentType := header.Header.Get("Content-Type")
	if !allowedMimeTypes[contentType] {
		writeError(w, http.StatusBadRequest, "invalid_file_type", "only PDF, JPEG, and PNG files are accepted")
		return
	}

	ext := resolveExtension(contentType, header.Filename)
	claims := auth.ClaimsFromContext(r.Context())
	dirPath := filepath.Join(h.uploadDir, claims.CampusID.String(), personID.String())

	filePath, err := saveUploadedFile(file, dirPath, ext)
	if err != nil {
		slog.ErrorContext(r.Context(), "agreementHandler.handleUpload: save failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to save document")
		return
	}

	agreement, err := h.svc.UploadManual(r.Context(), personID, filePath)
	if err != nil {
		slog.ErrorContext(r.Context(), "agreementHandler.handleUpload: service failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to process agreement")
		return
	}

	writeJSON(w, http.StatusOK, agreement)
}

// DownloadDocument handles GET /persons/{id}/agreement/document.
func (h *VolunteerAgreementHandler) DownloadDocument(w http.ResponseWriter, r *http.Request) {
	personID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid person ID format")
		return
	}

	agreements, err := h.svc.GetByPersonID(r.Context(), personID)
	if err != nil {
		slog.ErrorContext(r.Context(), "agreementHandler.DownloadDocument: failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get agreement")
		return
	}

	documentPath := findAcceptedDocumentPath(agreements)
	if documentPath == "" {
		writeError(w, http.StatusNotFound, "not_found", "no uploaded agreement document found")
		return
	}

	// Validate path is within upload directory (prevent path traversal)
	absPath, err := filepath.Abs(documentPath)
	if err != nil || !strings.HasPrefix(absPath, h.uploadDir) {
		writeError(w, http.StatusForbidden, "forbidden", "invalid document path")
		return
	}

	ext := filepath.Ext(documentPath)
	if ct := contentTypeForExt(ext); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"agreement%s\"", ext))
	http.ServeFile(w, r, absPath)
}

// resolveExtension picks a file extension from the filename, falling back to
// one derived from the validated MIME type.
func resolveExtension(contentType, filename string) string {
	if ext := filepath.Ext(filename); ext != "" {
		return ext
	}
	switch contentType {
	case "application/pdf":
		return ".pdf"
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	default:
		return ""
	}
}

// saveUploadedFile writes the uploaded file under dirPath with a random name
// and returns the resulting path.
func saveUploadedFile(file io.Reader, dirPath, ext string) (string, error) {
	if err := os.MkdirAll(dirPath, 0o750); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}
	filePath := filepath.Join(dirPath, uuid.New().String()+ext)
	dst, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("copy: %w", err)
	}
	return filePath, nil
}

// findAcceptedDocumentPath returns the document path of the first accepted
// agreement that has an uploaded document, or "" if none.
func findAcceptedDocumentPath(agreements []domain.VolunteerAgreement) string {
	for _, a := range agreements {
		if a.Status == domain.AgreementAccepted && a.DocumentPath != nil {
			return *a.DocumentPath
		}
	}
	return ""
}

// contentTypeForExt maps a file extension to its HTTP content type.
func contentTypeForExt(ext string) string {
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	default:
		return ""
	}
}

func extractIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.SplitN(forwarded, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	// Strip port from RemoteAddr
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}
