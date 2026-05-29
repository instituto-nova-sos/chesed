package repository

import (
	"context"
	"fmt"

	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReportRepository handles reporting/aggregation queries.
type ReportRepository struct {
	pool *pgxpool.Pool
}

// NewReportRepository creates a ReportRepository.
func NewReportRepository(pool *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{pool: pool}
}

// BuildAttendanceReport assembles the attendance summary report for the period.
// All queries are campus-scoped and use an inclusive date range on attendance_date.
func (r *ReportRepository) BuildAttendanceReport(
	ctx context.Context,
	filter domain.AttendanceReportFilter,
) (*domain.AttendanceReport, error) {
	report := &domain.AttendanceReport{
		Period:        domain.ReportPeriod{Start: filter.Start, End: filter.End},
		ByStatus:      map[string]int{},
		ByServiceType: []domain.ServiceTypeCount{},
		ByMonth:       []domain.MonthCount{},
	}

	if err := r.fetchTotals(ctx, filter, report); err != nil {
		return nil, fmt.Errorf("reportRepository.BuildAttendanceReport: totals: %w", err)
	}
	if err := r.fetchByStatus(ctx, filter, report); err != nil {
		return nil, fmt.Errorf("reportRepository.BuildAttendanceReport: by_status: %w", err)
	}
	if err := r.fetchByServiceType(ctx, filter, report); err != nil {
		return nil, fmt.Errorf("reportRepository.BuildAttendanceReport: by_service_type: %w", err)
	}
	if err := r.fetchByMonth(ctx, filter, report); err != nil {
		return nil, fmt.Errorf("reportRepository.BuildAttendanceReport: by_month: %w", err)
	}

	return report, nil
}

func (r *ReportRepository) fetchTotals(
	ctx context.Context,
	f domain.AttendanceReportFilter,
	report *domain.AttendanceReport,
) error {
	const q = `
		SELECT COUNT(*)::int, COUNT(DISTINCT person_id)::int
		FROM attendance
		WHERE campus_id = $1 AND attendance_date >= $2 AND attendance_date < ($3::date + INTERVAL '1 day')`
	return r.pool.QueryRow(ctx, q, f.CampusID, f.Start, f.End).
		Scan(&report.TotalAttendances, &report.UniquePersons)
}

func (r *ReportRepository) fetchByStatus(
	ctx context.Context,
	f domain.AttendanceReportFilter,
	report *domain.AttendanceReport,
) error {
	const q = `
		SELECT status, COUNT(*)::int
		FROM attendance
		WHERE campus_id = $1 AND attendance_date >= $2 AND attendance_date < ($3::date + INTERVAL '1 day')
		GROUP BY status`
	rows, err := r.pool.Query(ctx, q, f.CampusID, f.Start, f.End)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return err
		}
		report.ByStatus[status] = count
	}
	return rows.Err()
}

func (r *ReportRepository) fetchByServiceType(
	ctx context.Context,
	f domain.AttendanceReportFilter,
	report *domain.AttendanceReport,
) error {
	const q = `
		SELECT st.category, COUNT(*)::int
		FROM attendance a
		JOIN service_type st ON st.id = a.service_type_id
		WHERE a.campus_id = $1 AND a.attendance_date >= $2 AND a.attendance_date < ($3::date + INTERVAL '1 day')
		GROUP BY st.category
		ORDER BY COUNT(*) DESC, st.category ASC`
	rows, err := r.pool.Query(ctx, q, f.CampusID, f.Start, f.End)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.ServiceTypeCount
		if err := rows.Scan(&item.ServiceType, &item.Count); err != nil {
			return err
		}
		report.ByServiceType = append(report.ByServiceType, item)
	}
	return rows.Err()
}

func (r *ReportRepository) fetchByMonth(
	ctx context.Context,
	f domain.AttendanceReportFilter,
	report *domain.AttendanceReport,
) error {
	const q = `
		SELECT to_char(date_trunc('month', attendance_date), 'YYYY-MM') AS month, COUNT(*)::int
		FROM attendance
		WHERE campus_id = $1 AND attendance_date >= $2 AND attendance_date < ($3::date + INTERVAL '1 day')
		GROUP BY month
		ORDER BY month ASC`
	rows, err := r.pool.Query(ctx, q, f.CampusID, f.Start, f.End)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.MonthCount
		if err := rows.Scan(&item.Month, &item.Count); err != nil {
			return err
		}
		report.ByMonth = append(report.ByMonth, item)
	}
	return rows.Err()
}

// StreamAttendancesForCSV iterates attendances in the period, joining person and
// service_type, and invokes emit for each row. Stops on the first emit error.
func (r *ReportRepository) StreamAttendancesForCSV(
	ctx context.Context,
	filter domain.AttendanceReportFilter,
	emit func(domain.AttendanceCSVRow) error,
) error {
	const q = `
		SELECT a.id,
		       a.attendance_date,
		       p.full_name,
		       COALESCE(p.document_number, ''),
		       st.name,
		       a.status,
		       COALESCE(prof.full_name, ''),
		       a.created_at
		FROM attendance a
		JOIN person p ON p.id = a.person_id
		JOIN service_type st ON st.id = a.service_type_id
		LEFT JOIN person prof ON prof.id = a.professional_id
		WHERE a.campus_id = $1
		  AND a.attendance_date >= $2
		  AND a.attendance_date < ($3::date + INTERVAL '1 day')
		ORDER BY a.attendance_date ASC, a.id ASC`

	rows, err := r.pool.Query(ctx, q, filter.CampusID, filter.Start, filter.End)
	if err != nil {
		return fmt.Errorf("reportRepository.StreamAttendancesForCSV: query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var row domain.AttendanceCSVRow
		if err := rows.Scan(
			&row.AttendanceID,
			&row.AttendanceDate,
			&row.PersonName,
			&row.PersonDocument,
			&row.ServiceType,
			&row.Status,
			&row.ProfessionalName,
			&row.CreatedAt,
		); err != nil {
			return fmt.Errorf("reportRepository.StreamAttendancesForCSV: scan: %w", err)
		}
		if err := emit(row); err != nil {
			return fmt.Errorf("reportRepository.StreamAttendancesForCSV: emit: %w", err)
		}
	}
	return rows.Err()
}
