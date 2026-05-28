package repository

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AttendanceRepository handles attendance persistence.
type AttendanceRepository struct {
	pool *pgxpool.Pool
}

// NewAttendanceRepository creates a new AttendanceRepository.
func NewAttendanceRepository(pool *pgxpool.Pool) *AttendanceRepository {
	return &AttendanceRepository{pool: pool}
}

// Create inserts an attendance.
func (r *AttendanceRepository) Create(ctx context.Context, a domain.Attendance) (*domain.Attendance, error) {
	const q = `
		INSERT INTO attendance (id, person_id, triage_id, campus_id, service_type_id,
		                        professional_id, status, attendance_date, observations,
		                        recommendations, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at`

	if err := r.pool.QueryRow(ctx, q,
		a.ID, a.PersonID, a.TriageID, a.CampusID, a.ServiceTypeID,
		a.ProfessionalID, a.Status, a.AttendanceDate, a.Observations,
		a.Recommendations, a.CreatedBy,
	).Scan(&a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, fmt.Errorf("attendanceRepository.Create: %w", err)
	}
	return &a, nil
}

// FindByID returns an attendance by ID scoped to campus.
func (r *AttendanceRepository) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Attendance, error) {
	const q = `
		SELECT id, person_id, triage_id, campus_id, service_type_id, professional_id,
		       status, attendance_date, observations, recommendations,
		       created_at, updated_at, created_by
		FROM attendance
		WHERE id = $1 AND campus_id = $2`

	var a domain.Attendance
	if err := r.pool.QueryRow(ctx, q, id, campusID).Scan(
		&a.ID, &a.PersonID, &a.TriageID, &a.CampusID, &a.ServiceTypeID, &a.ProfessionalID,
		&a.Status, &a.AttendanceDate, &a.Observations, &a.Recommendations,
		&a.CreatedAt, &a.UpdatedAt, &a.CreatedBy,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("attendanceRepository.FindByID: %w", err)
	}
	return &a, nil
}

// FindByIDWithTransitions returns an attendance plus its transition history.
func (r *AttendanceRepository) FindByIDWithTransitions(ctx context.Context, id, campusID uuid.UUID) (*domain.AttendanceDetail, error) {
	att, err := r.FindByID(ctx, id, campusID)
	if err != nil {
		return nil, err
	}

	const q = `
		SELECT id, attendance_id, from_status, to_status, reason, transitioned_by, transitioned_at
		FROM attendance_transition
		WHERE attendance_id = $1
		ORDER BY transitioned_at ASC`

	rows, err := r.pool.Query(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("attendanceRepository.FindByIDWithTransitions: query: %w", err)
	}
	defer rows.Close()

	transitions := []domain.AttendanceTransition{}
	for rows.Next() {
		var t domain.AttendanceTransition
		if err := rows.Scan(&t.ID, &t.AttendanceID, &t.FromStatus, &t.ToStatus,
			&t.Reason, &t.TransitionedBy, &t.TransitionedAt); err != nil {
			return nil, fmt.Errorf("attendanceRepository.FindByIDWithTransitions: scan: %w", err)
		}
		transitions = append(transitions, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attendanceRepository.FindByIDWithTransitions: rows: %w", err)
	}

	return &domain.AttendanceDetail{Attendance: *att, Transitions: transitions}, nil
}

// List returns a paginated list of attendances scoped to campus.
func (r *AttendanceRepository) List(ctx context.Context, filter domain.AttendanceFilter) (*domain.AttendanceListResult, error) {
	args := []any{filter.CampusID}
	where := "a.campus_id = $1"
	if filter.PersonID != nil {
		args = append(args, *filter.PersonID)
		where += fmt.Sprintf(" AND a.person_id = $%d", len(args))
	}
	if filter.Status != nil {
		args = append(args, *filter.Status)
		where += fmt.Sprintf(" AND a.status = $%d", len(args))
	}
	if filter.From != nil {
		args = append(args, *filter.From)
		where += fmt.Sprintf(" AND a.attendance_date >= $%d", len(args))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		where += fmt.Sprintf(" AND a.attendance_date <= $%d", len(args))
	}

	countQuery := "SELECT COUNT(*) FROM attendance a WHERE " + where
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("attendanceRepository.List: count: %w", err)
	}

	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PerPage

	listQuery := fmt.Sprintf(`
		SELECT a.id, a.person_id, p.full_name, st.code, a.status, a.attendance_date
		FROM attendance a
		JOIN person p ON p.id = a.person_id
		JOIN service_type st ON st.id = a.service_type_id
		WHERE %s
		ORDER BY a.attendance_date DESC
		LIMIT $%d OFFSET $%d`, where, len(args)+1, len(args)+2)
	args = append(args, filter.PerPage, offset)

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("attendanceRepository.List: query: %w", err)
	}
	defer rows.Close()

	items := []domain.AttendanceListItem{}
	for rows.Next() {
		var it domain.AttendanceListItem
		if err := rows.Scan(&it.ID, &it.PersonID, &it.PersonName, &it.ServiceType, &it.Status, &it.AttendanceDate); err != nil {
			return nil, fmt.Errorf("attendanceRepository.List: scan: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attendanceRepository.List: rows: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.PerPage)))
	return &domain.AttendanceListResult{
		Data: items,
		Pagination: domain.Pagination{
			Page:       filter.Page,
			PerPage:    filter.PerPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// Transition atomically updates an attendance status and records the transition.
// The from-status check is enforced in SQL to prevent concurrent races.
func (r *AttendanceRepository) Transition(ctx context.Context, t domain.AttendanceTransition) (*domain.Attendance, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("attendanceRepository.Transition: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const updateQuery = `
		UPDATE attendance
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND status = $3
		RETURNING id, person_id, triage_id, campus_id, service_type_id, professional_id,
		          status, attendance_date, observations, recommendations,
		          created_at, updated_at, created_by`

	var a domain.Attendance
	if err := tx.QueryRow(ctx, updateQuery, t.ToStatus, t.AttendanceID, t.FromStatus).Scan(
		&a.ID, &a.PersonID, &a.TriageID, &a.CampusID, &a.ServiceTypeID, &a.ProfessionalID,
		&a.Status, &a.AttendanceDate, &a.Observations, &a.Recommendations,
		&a.CreatedAt, &a.UpdatedAt, &a.CreatedBy,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvalidTransition
		}
		return nil, fmt.Errorf("attendanceRepository.Transition: update: %w", err)
	}

	const insertQuery = `
		INSERT INTO attendance_transition (id, attendance_id, from_status, to_status,
		                                   reason, transitioned_by)
		VALUES ($1, $2, $3, $4, $5, $6)`

	if _, err := tx.Exec(ctx, insertQuery, t.ID, t.AttendanceID, t.FromStatus, t.ToStatus,
		t.Reason, t.TransitionedBy); err != nil {
		return nil, fmt.Errorf("attendanceRepository.Transition: insert: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("attendanceRepository.Transition: commit: %w", err)
	}

	return &a, nil
}

// UpdateNotes updates observations and recommendations for an attendance.
func (r *AttendanceRepository) UpdateNotes(ctx context.Context, id, campusID uuid.UUID, observations, recommendations *string) (*domain.Attendance, error) {
	const q = `
		UPDATE attendance
		SET observations = $1, recommendations = $2, updated_at = NOW()
		WHERE id = $3 AND campus_id = $4
		RETURNING id, person_id, triage_id, campus_id, service_type_id, professional_id,
		          status, attendance_date, observations, recommendations,
		          created_at, updated_at, created_by`

	var a domain.Attendance
	if err := r.pool.QueryRow(ctx, q, observations, recommendations, id, campusID).Scan(
		&a.ID, &a.PersonID, &a.TriageID, &a.CampusID, &a.ServiceTypeID, &a.ProfessionalID,
		&a.Status, &a.AttendanceDate, &a.Observations, &a.Recommendations,
		&a.CreatedAt, &a.UpdatedAt, &a.CreatedBy,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("attendanceRepository.UpdateNotes: %w", err)
	}
	return &a, nil
}
