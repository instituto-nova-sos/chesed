package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/jackc/pgx/v5"
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
		Period:         domain.ReportPeriod{Start: filter.Start, End: filter.End},
		ByStatus:       map[string]int{},
		ByServiceType:  []domain.ServiceTypeCount{},
		ByMonth:        []domain.MonthCount{},
		ByProfessional: []domain.ProfessionalCount{},
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
	if err := r.fetchByProfessional(ctx, filter, report); err != nil {
		return nil, fmt.Errorf("reportRepository.BuildAttendanceReport: by_professional: %w", err)
	}

	return report, nil
}

// applyFilters appends the optional attendance filters to base, keeping campus
// ($1), start ($2) and end ($3) as the leading positional args. The `a` alias
// is assumed for the attendance table in the caller's query.
func applyFilters(base string, f domain.AttendanceReportFilter) (string, []any) {
	args := []any{f.CampusID, f.Start, f.End}
	for _, opt := range []struct {
		column string
		value  *uuid.UUID
	}{
		{"a.service_type_id", f.ServiceTypeID},
		{"a.campaign_id", f.CampaignID},
		{"a.professional_id", f.ProfessionalID},
	} {
		if opt.value == nil {
			continue
		}
		args = append(args, *opt.value)
		base += fmt.Sprintf(" AND %s = $%d", opt.column, len(args))
	}
	return base, args
}

// scanStatusCounts drains (status, count) rows into dst and returns their sum.
func scanStatusCounts(rows pgx.Rows, dst map[string]int) (int, error) {
	defer rows.Close()
	total := 0
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return 0, err
		}
		dst[status] = count
		total += count
	}
	return total, rows.Err()
}

func (r *ReportRepository) fetchTotals(
	ctx context.Context,
	f domain.AttendanceReportFilter,
	report *domain.AttendanceReport,
) error {
	q, args := applyFilters(`
		SELECT COUNT(*)::int, COUNT(DISTINCT a.person_id)::int
		FROM attendance a
		WHERE a.campus_id = $1 AND a.attendance_date >= $2 AND a.attendance_date < ($3::date + INTERVAL '1 day')`, f)
	return r.pool.QueryRow(ctx, q, args...).
		Scan(&report.TotalAttendances, &report.UniquePersons)
}

func (r *ReportRepository) fetchByStatus(
	ctx context.Context,
	f domain.AttendanceReportFilter,
	report *domain.AttendanceReport,
) error {
	q, args := applyFilters(`
		SELECT a.status, COUNT(*)::int
		FROM attendance a
		WHERE a.campus_id = $1 AND a.attendance_date >= $2 AND a.attendance_date < ($3::date + INTERVAL '1 day')`, f)
	q += ` GROUP BY a.status`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return err
	}
	_, err = scanStatusCounts(rows, report.ByStatus)
	return err
}

func (r *ReportRepository) fetchByServiceType(
	ctx context.Context,
	f domain.AttendanceReportFilter,
	report *domain.AttendanceReport,
) error {
	q, args := applyFilters(`
		SELECT st.category, COUNT(*)::int
		FROM attendance a
		JOIN service_type st ON st.id = a.service_type_id
		WHERE a.campus_id = $1 AND a.attendance_date >= $2 AND a.attendance_date < ($3::date + INTERVAL '1 day')`, f)
	q += ` GROUP BY st.category ORDER BY COUNT(*) DESC, st.category ASC`
	rows, err := r.pool.Query(ctx, q, args...)
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
	q, args := applyFilters(`
		SELECT to_char(date_trunc('month', a.attendance_date), 'YYYY-MM') AS month, COUNT(*)::int
		FROM attendance a
		WHERE a.campus_id = $1 AND a.attendance_date >= $2 AND a.attendance_date < ($3::date + INTERVAL '1 day')`, f)
	q += ` GROUP BY month ORDER BY month ASC`
	rows, err := r.pool.Query(ctx, q, args...)
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

func (r *ReportRepository) fetchByProfessional(
	ctx context.Context,
	f domain.AttendanceReportFilter,
	report *domain.AttendanceReport,
) error {
	q, args := applyFilters(`
		SELECT prof.id, prof.full_name, COUNT(*)::int
		FROM attendance a
		JOIN person prof ON prof.id = a.professional_id
		WHERE a.campus_id = $1 AND a.attendance_date >= $2 AND a.attendance_date < ($3::date + INTERVAL '1 day')`, f)
	q += ` GROUP BY prof.id, prof.full_name ORDER BY COUNT(*) DESC, prof.full_name ASC`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.ProfessionalCount
		if err := rows.Scan(&item.ProfessionalID, &item.ProfessionalName, &item.Count); err != nil {
			return err
		}
		report.ByProfessional = append(report.ByProfessional, item)
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

// BuildCampaignMetrics aggregates per-campaign counters. The campaign lookup
// itself enforces the campus boundary: a campaign outside the caller's campus
// yields ErrNotFound with no existence disclosure.
func (r *ReportRepository) BuildCampaignMetrics(ctx context.Context, campaignID, campusID uuid.UUID) (*domain.CampaignMetrics, error) {
	m := &domain.CampaignMetrics{AttendancesByStatus: map[string]int{}}

	const campaignQuery = `
		SELECT id, name, status, start_date, end_date
		FROM campaign
		WHERE id = $1 AND campus_id = $2`
	if err := r.pool.QueryRow(ctx, campaignQuery, campaignID, campusID).Scan(
		&m.CampaignID, &m.CampaignName, &m.Status, &m.Period.StartDate, &m.Period.EndDate,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("reportRepository.BuildCampaignMetrics: campaign: %w", err)
	}

	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*)::int FROM triage WHERE campaign_id = $1 AND campus_id = $2`,
		campaignID, campusID,
	).Scan(&m.TriageCount); err != nil {
		return nil, fmt.Errorf("reportRepository.BuildCampaignMetrics: triages: %w", err)
	}

	if err := r.fetchCampaignAttendances(ctx, campaignID, campusID, m); err != nil {
		return nil, err
	}

	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*)::int FROM campaign_team WHERE campaign_id = $1`,
		campaignID,
	).Scan(&m.TeamSize); err != nil {
		return nil, fmt.Errorf("reportRepository.BuildCampaignMetrics: team: %w", err)
	}

	return m, nil
}

// BuildDashboard assembles the campus-scoped operational snapshot. Every query
// is filtered by campusID; time windows are computed server-side relative to now().
func (r *ReportRepository) BuildDashboard(ctx context.Context, campusID uuid.UUID) (*domain.DashboardMetrics, error) {
	m := &domain.DashboardMetrics{
		AttendancesByStatus: map[string]int{},
		RecentMonths:        []domain.MonthCount{},
	}

	if err := r.fetchDashboardCounters(ctx, campusID, m); err != nil {
		return nil, fmt.Errorf("reportRepository.BuildDashboard: counters: %w", err)
	}
	if err := r.fetchDashboardByStatus(ctx, campusID, m); err != nil {
		return nil, fmt.Errorf("reportRepository.BuildDashboard: by_status: %w", err)
	}
	if err := r.fetchDashboardRecentMonths(ctx, campusID, m); err != nil {
		return nil, fmt.Errorf("reportRepository.BuildDashboard: recent_months: %w", err)
	}

	return m, nil
}

func (r *ReportRepository) fetchDashboardCounters(ctx context.Context, campusID uuid.UUID, m *domain.DashboardMetrics) error {
	const q = `
		SELECT
			(SELECT COUNT(*)::int FROM person WHERE campus_id = $1 AND is_active),
			(SELECT COUNT(*)::int FROM attendance
			   WHERE campus_id = $1 AND attendance_date >= date_trunc('month', now())
			     AND attendance_date < date_trunc('month', now()) + INTERVAL '1 month'),
			(SELECT COUNT(*)::int FROM attendance
			   WHERE campus_id = $1 AND status = 'SCHEDULED' AND attendance_date >= current_date),
			(SELECT COUNT(*)::int FROM campaign WHERE campus_id = $1 AND status = 'ACTIVE')`
	return r.pool.QueryRow(ctx, q, campusID).Scan(
		&m.TotalPersons, &m.AttendancesThisMonth, &m.UpcomingScheduled, &m.ActiveCampaigns,
	)
}

func (r *ReportRepository) fetchDashboardByStatus(ctx context.Context, campusID uuid.UUID, m *domain.DashboardMetrics) error {
	rows, err := r.pool.Query(ctx,
		`SELECT status, COUNT(*)::int FROM attendance WHERE campus_id = $1 GROUP BY status`,
		campusID,
	)
	if err != nil {
		return err
	}
	_, err = scanStatusCounts(rows, m.AttendancesByStatus)
	return err
}

func (r *ReportRepository) fetchDashboardRecentMonths(ctx context.Context, campusID uuid.UUID, m *domain.DashboardMetrics) error {
	const q = `
		SELECT to_char(months.month, 'YYYY-MM'), COALESCE(counts.count, 0)::int
		FROM generate_series(
			date_trunc('month', now()) - INTERVAL '5 months',
			date_trunc('month', now()),
			INTERVAL '1 month'
		) AS months(month)
		LEFT JOIN (
			SELECT date_trunc('month', attendance_date) AS month, COUNT(*) AS count
			FROM attendance
			WHERE campus_id = $1
			GROUP BY date_trunc('month', attendance_date)
		) AS counts ON counts.month = months.month
		ORDER BY months.month ASC`
	rows, err := r.pool.Query(ctx, q, campusID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.MonthCount
		if err := rows.Scan(&item.Month, &item.Count); err != nil {
			return err
		}
		m.RecentMonths = append(m.RecentMonths, item)
	}
	return rows.Err()
}

func (r *ReportRepository) fetchCampaignAttendances(ctx context.Context, campaignID, campusID uuid.UUID, m *domain.CampaignMetrics) error {
	rows, err := r.pool.Query(ctx,
		`SELECT status, COUNT(*)::int
		 FROM attendance
		 WHERE campaign_id = $1 AND campus_id = $2
		 GROUP BY status`,
		campaignID, campusID,
	)
	if err != nil {
		return fmt.Errorf("reportRepository.BuildCampaignMetrics: attendances: %w", err)
	}
	total, err := scanStatusCounts(rows, m.AttendancesByStatus)
	if err != nil {
		return fmt.Errorf("reportRepository.BuildCampaignMetrics: scan: %w", err)
	}
	m.AttendanceTotal = total
	return nil
}
