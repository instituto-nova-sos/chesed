package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAttendanceRepoMock(t *testing.T) (*AttendanceRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return NewAttendanceRepository(mock), mock
}

// TestAttendanceRepository_List_UsesServiceTypeName documents the contract that
// `service_type` in AttendanceListItem is populated from `service_type.name` —
// not `service_type.code` (which does not exist in migration 000007).
// Regression guard for Sprint 3 bug discovered during Sprint 4 reports work.
func TestAttendanceRepository_List_UsesServiceTypeName(t *testing.T) {
	t.Run("happy path returns service_type from st.name and respects campus scope", func(t *testing.T) {
		repo, mock := newAttendanceRepoMock(t)

		campusID := uuid.New()
		attendanceID := uuid.New()
		personID := uuid.New()
		attendanceDate := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)

		// Count query: 1 record matches the campus
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM attendance a WHERE a.campus_id = $1`)).
			WithArgs(campusID).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

		// List query MUST select st.name (not st.code) and join service_type.
		// The pattern matcher pins the contract; if SQL changes columns this
		// test fails noisily.
		mock.ExpectQuery(`SELECT a\.id, a\.person_id, p\.full_name, st\.name, a\.status, a\.attendance_date.*FROM attendance a.*JOIN service_type st ON st\.id = a\.service_type_id`).
			WithArgs(campusID, 20, 0).
			WillReturnRows(pgxmock.NewRows(
				[]string{"id", "person_id", "full_name", "name", "status", "attendance_date"},
			).AddRow(attendanceID, personID, "Maria Silva", "Consulta Medica", "SCHEDULED", attendanceDate))

		result, err := repo.List(context.Background(), domain.AttendanceFilter{
			CampusID: campusID,
			Page:     1,
			PerPage:  20,
		})
		require.NoError(t, err)
		require.Len(t, result.Data, 1)
		assert.Equal(t, "Consulta Medica", result.Data[0].ServiceType,
			"service_type must be populated from service_type.name, not the non-existent st.code column")
		assert.Equal(t, "Maria Silva", result.Data[0].PersonName)
		assert.Equal(t, "SCHEDULED", result.Data[0].Status)
		assert.Equal(t, 1, result.Pagination.Total)
		assert.Equal(t, 1, result.Pagination.TotalPages)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty result returns valid pagination", func(t *testing.T) {
		repo, mock := newAttendanceRepoMock(t)
		campusID := uuid.New()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM attendance a WHERE a.campus_id = $1`)).
			WithArgs(campusID).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`SELECT a\.id.*FROM attendance a.*JOIN service_type st`).
			WithArgs(campusID, 20, 0).
			WillReturnRows(pgxmock.NewRows(
				[]string{"id", "person_id", "full_name", "name", "status", "attendance_date"},
			))

		result, err := repo.List(context.Background(), domain.AttendanceFilter{
			CampusID: campusID,
			Page:     1,
			PerPage:  20,
		})
		require.NoError(t, err)
		assert.Len(t, result.Data, 0)
		assert.Equal(t, 0, result.Pagination.Total)
		assert.Equal(t, 0, result.Pagination.TotalPages)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("status filter is appended to WHERE clause", func(t *testing.T) {
		repo, mock := newAttendanceRepoMock(t)
		campusID := uuid.New()
		status := "SCHEDULED"

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM attendance a WHERE a\.campus_id = \$1 AND a\.status = \$2`).
			WithArgs(campusID, status).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`SELECT a\.id.*WHERE a\.campus_id = \$1 AND a\.status = \$2.*ORDER BY a\.attendance_date DESC.*LIMIT \$3 OFFSET \$4`).
			WithArgs(campusID, status, 20, 0).
			WillReturnRows(pgxmock.NewRows(
				[]string{"id", "person_id", "full_name", "name", "status", "attendance_date"},
			))

		_, err := repo.List(context.Background(), domain.AttendanceFilter{
			CampusID: campusID,
			Status:   &status,
			Page:     1,
			PerPage:  20,
		})
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("pagination math: page 3 perPage 10 produces OFFSET 20", func(t *testing.T) {
		repo, mock := newAttendanceRepoMock(t)
		campusID := uuid.New()

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM attendance`).
			WithArgs(campusID).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(25))
		mock.ExpectQuery(`SELECT a\.id.*LIMIT \$2 OFFSET \$3`).
			WithArgs(campusID, 10, 20).
			WillReturnRows(pgxmock.NewRows(
				[]string{"id", "person_id", "full_name", "name", "status", "attendance_date"},
			))

		result, err := repo.List(context.Background(), domain.AttendanceFilter{
			CampusID: campusID,
			Page:     3,
			PerPage:  10,
		})
		require.NoError(t, err)
		assert.Equal(t, 3, result.Pagination.TotalPages, "ceil(25/10) = 3")
		assert.Equal(t, 25, result.Pagination.Total)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAttendanceRepository_FindBySyncID(t *testing.T) {
	t.Run("returns attendance scoped to campus", func(t *testing.T) {
		repo, mock := newAttendanceRepoMock(t)
		syncID := uuid.New()
		campusID := uuid.New()
		attendanceID := uuid.New()
		personID := uuid.New()
		serviceTypeID := uuid.New()
		professionalID := uuid.New()
		now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)

		mock.ExpectQuery(`SELECT .*FROM attendance.*WHERE sync_id = \$1 AND campus_id = \$2`).
			WithArgs(syncID, campusID).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "person_id", "triage_id", "campaign_id", "campus_id", "service_type_id", "professional_id",
				"status", "attendance_date", "observations", "recommendations",
				"created_at", "updated_at", "created_by",
			}).AddRow(
				attendanceID, personID, nil, nil, campusID, serviceTypeID, professionalID,
				"SCHEDULED", now, nil, nil,
				now, now, nil,
			))

		a, err := repo.FindBySyncID(context.Background(), syncID, campusID)
		require.NoError(t, err)
		assert.Equal(t, attendanceID, a.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrNotFound on miss", func(t *testing.T) {
		repo, mock := newAttendanceRepoMock(t)
		syncID := uuid.New()
		campusID := uuid.New()

		mock.ExpectQuery(`SELECT .*FROM attendance.*WHERE sync_id = \$1 AND campus_id = \$2`).
			WithArgs(syncID, campusID).
			WillReturnError(pgx.ErrNoRows)

		_, err := repo.FindBySyncID(context.Background(), syncID, campusID)
		assert.True(t, errors.Is(err, domain.ErrNotFound))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAttendanceRepository_ListUpdatedSince(t *testing.T) {
	t.Run("returns delta records campus-scoped, ordered ASC, limit honoured", func(t *testing.T) {
		repo, mock := newAttendanceRepoMock(t)
		campusID := uuid.New()
		since := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
		updatedAt := time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)
		attendanceID := uuid.New()
		personID := uuid.New()
		serviceTypeID := uuid.New()
		professionalID := uuid.New()

		mock.ExpectQuery(`SELECT .*FROM attendance.*WHERE campus_id = \$1 AND updated_at > \$2.*ORDER BY updated_at ASC.*LIMIT \$3`).
			WithArgs(campusID, since, 100).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "person_id", "triage_id", "campaign_id", "campus_id", "service_type_id", "professional_id",
				"status", "attendance_date", "observations", "recommendations",
				"created_at", "updated_at", "created_by",
			}).AddRow(
				attendanceID, personID, nil, nil, campusID, serviceTypeID, professionalID,
				"SCHEDULED", updatedAt, nil, nil,
				updatedAt, updatedAt, nil,
			))

		records, err := repo.ListUpdatedSince(context.Background(), campusID, since, 100)
		require.NoError(t, err)
		require.Len(t, records, 1)
		assert.Equal(t, attendanceID, records[0].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAttendanceRepository_CreateWithSync(t *testing.T) {
	t.Run("INSERT carries sync_id column", func(t *testing.T) {
		repo, mock := newAttendanceRepoMock(t)
		campusID := uuid.New()
		attendanceID := uuid.New()
		personID := uuid.New()
		serviceTypeID := uuid.New()
		professionalID := uuid.New()
		syncID := uuid.New()
		now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)

		mock.ExpectQuery(regexp.MustCompile(`INSERT INTO attendance.*sync_id.*RETURNING created_at, updated_at`).String()).
			WithArgs(
				attendanceID, personID, (*uuid.UUID)(nil), (*uuid.UUID)(nil), campusID, serviceTypeID,
				professionalID, "SCHEDULED", now, (*string)(nil),
				(*string)(nil), (*uuid.UUID)(nil), syncID,
			).
			WillReturnRows(pgxmock.NewRows([]string{"created_at", "updated_at"}).AddRow(now, now))

		a := domain.Attendance{
			ID:             attendanceID,
			PersonID:       personID,
			CampusID:       campusID,
			ServiceTypeID:  serviceTypeID,
			ProfessionalID: professionalID,
			Status:         "SCHEDULED",
			AttendanceDate: now,
		}
		out, err := repo.CreateWithSync(context.Background(), a, syncID)
		require.NoError(t, err)
		assert.Equal(t, attendanceID, out.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
