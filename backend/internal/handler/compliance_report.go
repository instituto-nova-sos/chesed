package handler

import (
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

// csvComplianceHeader defines the column order for compliance CSV exports.
var csvComplianceHeader = []string{"metric", "value"}

// ComplianceReportHandler exposes the LGPD compliance report and CSV export.
type ComplianceReportHandler struct {
	svc *service.ComplianceReportService
}

// NewComplianceReportHandler creates a ComplianceReportHandler.
func NewComplianceReportHandler(svc *service.ComplianceReportService) *ComplianceReportHandler {
	return &ComplianceReportHandler{svc: svc}
}

// Report handles GET /reports/compliance.
func (h *ComplianceReportHandler) Report(w http.ResponseWriter, r *http.Request) {
	start, end, ok := parseDateRange(w, r)
	if !ok {
		return
	}

	report, err := h.svc.GetReport(r.Context(), start, end)
	if err != nil {
		h.writeError(w, r, err, "Report")
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// Export handles GET /reports/compliance/export?format=csv.
func (h *ComplianceReportHandler) Export(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}
	if format != "csv" {
		writeError(w, http.StatusBadRequest, "invalid_format", "only csv format is supported")
		return
	}

	start, end, ok := parseDateRange(w, r)
	if !ok {
		return
	}

	filename := fmt.Sprintf("compliance_%s_%s.csv",
		start.Format("2006-01-02"), end.Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	writer := csv.NewWriter(w)
	if err := writer.Write(csvComplianceHeader); err != nil {
		slog.ErrorContext(r.Context(), "complianceReportHandler.Export: header write failed", "error", err.Error())
		return
	}

	err := h.svc.ExportCSV(r.Context(), start, end, func(row domain.ComplianceCSVRow) error {
		return writer.Write([]string{row.Metric, strconv.Itoa(row.Value)})
	})
	writer.Flush()
	if err != nil {
		slog.ErrorContext(r.Context(), "complianceReportHandler.Export: stream failed", "error", err.Error())
		return
	}
	if err := writer.Error(); err != nil {
		slog.ErrorContext(r.Context(), "complianceReportHandler.Export: writer flush failed", "error", err.Error())
	}
}

func (h *ComplianceReportHandler) writeError(w http.ResponseWriter, r *http.Request, err error, op string) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "missing campus context")
	case errors.Is(err, service.ErrInvalidReportRange):
		writeError(w, http.StatusBadRequest, "invalid_range", "invalid date range")
	default:
		slog.ErrorContext(r.Context(), "complianceReportHandler."+op+": failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to build compliance report")
	}
}
