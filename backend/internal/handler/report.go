package handler

import (
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

// csvAttendanceHeader defines the column order for attendance CSV exports.
var csvAttendanceHeader = []string{
	"attendance_id",
	"attendance_date",
	"person_name",
	"person_document",
	"service_type",
	"status",
	"professional_name",
	"created_at",
}

// ReportHandler exposes attendance summary and CSV export endpoints.
type ReportHandler struct {
	svc *service.ReportService
}

// NewReportHandler creates a ReportHandler.
func NewReportHandler(svc *service.ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

// AttendanceSummary handles GET /reports/attendances.
func (h *ReportHandler) AttendanceSummary(w http.ResponseWriter, r *http.Request) {
	start, end, ok := parseReportRange(w, r)
	if !ok {
		return
	}

	report, err := h.svc.GetAttendanceReport(r.Context(), start, end)
	if err != nil {
		h.writeReportError(w, r, err, "AttendanceSummary")
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// AttendanceExport handles GET /reports/attendances/export?format=csv.
// Only `format=csv` is supported in Phase 1. Other formats return 400.
func (h *ReportHandler) AttendanceExport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}
	if format != "csv" {
		writeError(w, http.StatusBadRequest, "invalid_format", "only csv format is supported")
		return
	}

	start, end, ok := parseReportRange(w, r)
	if !ok {
		return
	}

	filename := fmt.Sprintf("attendances_%s_%s.csv",
		start.Format("2006-01-02"), end.Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	writer := csv.NewWriter(w)
	if err := writer.Write(csvAttendanceHeader); err != nil {
		slog.ErrorContext(r.Context(), "reportHandler.AttendanceExport: header write failed", "error", err.Error())
		return
	}

	err := h.svc.StreamAttendanceCSV(r.Context(), start, end, func(row domain.AttendanceCSVRow) error {
		return writer.Write([]string{
			row.AttendanceID.String(),
			row.AttendanceDate.UTC().Format(time.RFC3339),
			row.PersonName,
			row.PersonDocument,
			row.ServiceType,
			row.Status,
			row.ProfessionalName,
			row.CreatedAt.UTC().Format(time.RFC3339),
		})
	})
	writer.Flush()
	if err != nil {
		// Headers already sent; log and end the stream.
		slog.ErrorContext(r.Context(), "reportHandler.AttendanceExport: stream failed", "error", err.Error())
		return
	}
	if err := writer.Error(); err != nil {
		slog.ErrorContext(r.Context(), "reportHandler.AttendanceExport: writer flush failed", "error", err.Error())
	}
}

func parseReportRange(w http.ResponseWriter, r *http.Request) (time.Time, time.Time, bool) {
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	if startStr == "" || endStr == "" {
		writeError(w, http.StatusBadRequest, "invalid_range", "start and end are required")
		return time.Time{}, time.Time{}, false
	}
	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_start", "start must be YYYY-MM-DD")
		return time.Time{}, time.Time{}, false
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_end", "end must be YYYY-MM-DD")
		return time.Time{}, time.Time{}, false
	}
	if end.Before(start) {
		writeError(w, http.StatusBadRequest, "invalid_range", "end must be on or after start")
		return time.Time{}, time.Time{}, false
	}

	const maxDays = 366
	if end.Sub(start)/(24*time.Hour) > maxDays {
		writeError(w, http.StatusBadRequest, "range_too_large",
			"range may not exceed "+strconv.Itoa(maxDays)+" days")
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

// CampaignMetrics handles GET /reports/campaigns/{id}.
func (h *ReportHandler) CampaignMetrics(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid campaign ID format")
		return
	}

	metrics, err := h.svc.GetCampaignMetrics(r.Context(), id)
	if err != nil {
		h.writeReportError(w, r, err, "CampaignMetrics")
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

func (h *ReportHandler) writeReportError(w http.ResponseWriter, r *http.Request, err error, op string) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "missing campus context")
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "campaign not found")
	case errors.Is(err, service.ErrInvalidReportRange):
		writeError(w, http.StatusBadRequest, "invalid_range", "invalid date range")
	default:
		slog.ErrorContext(r.Context(), "reportHandler."+op+": failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to build report")
	}
}
