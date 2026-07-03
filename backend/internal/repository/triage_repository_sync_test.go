package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/instituto-nova-sos/chesed/internal/domain"
)

func newTriageRepoMock(t *testing.T) (*TriageRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return NewTriageRepository(mock), mock
}

func TestTriageRepository_FindBySyncID(t *testing.T) {
	t.Run("returns triage scoped to campus", func(t *testing.T) {
		repo, mock := newTriageRepoMock(t)
		syncID := uuid.New()
		campusID := uuid.New()
		triageID := uuid.New()
		personID := uuid.New()
		now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)

		mock.ExpectQuery(`SELECT .*FROM triage.*WHERE sync_id = \$1 AND campus_id = \$2`).
			WithArgs(syncID, campusID).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "person_id", "campaign_id", "campus_id", "main_complaint", "assigned_team",
				"triage_date", "location", "triaged_by", "notes", "is_active",
				"created_at", "updated_at",
			}).AddRow(
				triageID, personID, nil, campusID, "Headache", nil,
				now, nil, personID, nil, true,
				now, now,
			))

		tr, err := repo.FindBySyncID(context.Background(), syncID, campusID)
		require.NoError(t, err)
		assert.Equal(t, triageID, tr.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrNotFound on miss", func(t *testing.T) {
		repo, mock := newTriageRepoMock(t)
		syncID := uuid.New()
		campusID := uuid.New()

		mock.ExpectQuery(`SELECT .*FROM triage.*WHERE sync_id = \$1 AND campus_id = \$2`).
			WithArgs(syncID, campusID).
			WillReturnError(pgx.ErrNoRows)

		_, err := repo.FindBySyncID(context.Background(), syncID, campusID)
		assert.True(t, errors.Is(err, domain.ErrNotFound))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTriageRepository_CreateWithSync(t *testing.T) {
	t.Run("INSERT carries sync_id and inserts requested services", func(t *testing.T) {
		repo, mock := newTriageRepoMock(t)
		campusID := uuid.New()
		triageID := uuid.New()
		personID := uuid.New()
		triagedBy := uuid.New()
		syncID := uuid.New()
		serviceTypeID := uuid.New()
		now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)

		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO triage.*sync_id.*RETURNING created_at, updated_at`).
			WithArgs(
				triageID, personID, (*uuid.UUID)(nil), campusID, "Headache", (*uuid.UUID)(nil),
				now, (*string)(nil), triagedBy, (*string)(nil), true, syncID,
			).
			WillReturnRows(pgxmock.NewRows([]string{"created_at", "updated_at"}).AddRow(now, now))
		mock.ExpectExec(`INSERT INTO triage_requested_service`).
			WithArgs(triageID, serviceTypeID).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectCommit()

		tr := domain.Triage{
			ID:             triageID,
			PersonID:       personID,
			CampusID:       campusID,
			MainComplaint:  "Headache",
			TriageDate:     now,
			TriagedBy:      triagedBy,
			IsActive:       true,
			RequestedTypes: []uuid.UUID{serviceTypeID},
		}
		out, err := repo.CreateWithSync(context.Background(), tr, syncID)
		require.NoError(t, err)
		assert.Equal(t, triageID, out.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTriageRepository_ListUpdatedSince(t *testing.T) {
	t.Run("returns delta records campus-scoped and ordered ASC limited to N", func(t *testing.T) {
		repo, mock := newTriageRepoMock(t)
		campusID := uuid.New()
		since := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
		updatedAt := time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)
		triageID := uuid.New()
		personID := uuid.New()

		mock.ExpectQuery(`SELECT .*FROM triage.*WHERE campus_id = \$1 AND updated_at > \$2.*ORDER BY updated_at ASC.*LIMIT \$3`).
			WithArgs(campusID, since, 100).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "person_id", "campaign_id", "campus_id", "main_complaint", "assigned_team",
				"triage_date", "location", "triaged_by", "notes", "is_active",
				"created_at", "updated_at",
			}).AddRow(
				triageID, personID, nil, campusID, "Headache", nil,
				updatedAt, nil, personID, nil, true,
				updatedAt, updatedAt,
			))

		records, err := repo.ListUpdatedSince(context.Background(), campusID, since, 100)
		require.NoError(t, err)
		require.Len(t, records, 1)
		assert.Equal(t, triageID, records[0].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
