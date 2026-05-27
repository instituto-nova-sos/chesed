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

	ip := extractIP(r)
	userAgent := r.UserAgent()

	agreement, err := h.svc.AcceptDigital(r.Context(), claims.PersonID, ip, userAgent)
	if err != nil {
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

// Upload handles POST /persons/{id}/agreement/upload.
func (h *VolunteerAgreementHandler) Upload(w http.ResponseWriter, r *http.Request) {
	personID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid person ID format")
		return
	}

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

	// Determine file extension
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		switch contentType {
		case "application/pdf":
			ext = ".pdf"
		case "image/jpeg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		}
	}

	// Create upload directory
	claims := auth.ClaimsFromContext(r.Context())
	dirPath := filepath.Join(h.uploadDir, claims.CampusID.String(), personID.String())
	if err := os.MkdirAll(dirPath, 0o750); err != nil {
		slog.ErrorContext(r.Context(), "agreementHandler.Upload: mkdir failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to save document")
		return
	}

	// Save file
	fileName := uuid.New().String() + ext
	filePath := filepath.Join(dirPath, fileName)

	dst, err := os.Create(filePath)
	if err != nil {
		slog.ErrorContext(r.Context(), "agreementHandler.Upload: create file failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to save document")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		slog.ErrorContext(r.Context(), "agreementHandler.Upload: copy failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to save document")
		return
	}

	agreement, err := h.svc.UploadManual(r.Context(), personID, filePath)
	if err != nil {
		slog.ErrorContext(r.Context(), "agreementHandler.Upload: service failed", "error", err.Error())
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

	// Find the accepted agreement with a document
	var documentPath string
	for _, a := range agreements {
		if a.Status == domain.AgreementAccepted && a.DocumentPath != nil {
			documentPath = *a.DocumentPath
			break
		}
	}

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
	switch ext {
	case ".pdf":
		w.Header().Set("Content-Type", "application/pdf")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"agreement%s\"", ext))
	http.ServeFile(w, r, absPath)
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
