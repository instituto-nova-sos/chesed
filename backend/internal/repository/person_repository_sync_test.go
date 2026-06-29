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

func newPersonRepoMock(t *testing.T) (*PersonRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return NewPersonRepository(mock), mock
}

func TestPersonRepository_FindBySyncID(t *testing.T) {
	t.Run("returns person scoped to campus", func(t *testing.T) {
		repo, mock := newPersonRepoMock(t)
		syncID := uuid.New()
		campusID := uuid.New()
		personID := uuid.New()
		createdAt := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)

		mock.ExpectQuery(`SELECT .*FROM person.*WHERE sync_id = \$1 AND campus_id = \$2`).
			WithArgs(syncID, campusID).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "full_name", "birth_date", "document_type", "document_number",
				"gender", "email", "phone", "photo_url", "referral_source",
				"nationality", "campus_id", "is_active", "created_at", "updated_at", "created_by",
			}).AddRow(
				personID, "Maria Silva", nil, "CPF", nil,
				nil, nil, nil, nil, nil,
				"BRA", campusID, true, createdAt, createdAt, nil,
			))

		p, err := repo.FindBySyncID(context.Background(), syncID, campusID)
		require.NoError(t, err)
		assert.Equal(t, personID, p.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrNotFound when no row matches", func(t *testing.T) {
		repo, mock := newPersonRepoMock(t)
		syncID := uuid.New()
		campusID := uuid.New()

		mock.ExpectQuery(`SELECT .*FROM person.*WHERE sync_id = \$1 AND campus_id = \$2`).
			WithArgs(syncID, campusID).
			WillReturnError(pgx.ErrNoRows)

		_, err := repo.FindBySyncID(context.Background(), syncID, campusID)
		assert.True(t, errors.Is(err, domain.ErrNotFound))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPersonRepository_ListUpdatedSince(t *testing.T) {
	t.Run("returns delta records ordered by updated_at ASC limited to N", func(t *testing.T) {
		repo, mock := newPersonRepoMock(t)
		campusID := uuid.New()
		since := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
		updatedAt := time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)
		personID := uuid.New()

		mock.ExpectQuery(`SELECT .*FROM person.*WHERE campus_id = \$1 AND updated_at > \$2.*ORDER BY updated_at ASC.*LIMIT \$3`).
			WithArgs(campusID, since, 100).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "full_name", "birth_date", "document_type", "document_number",
				"gender", "email", "phone", "photo_url", "referral_source",
				"nationality", "campus_id", "is_active", "created_at", "updated_at", "created_by",
			}).AddRow(
				personID, "Maria Silva", nil, "CPF", nil,
				nil, nil, nil, nil, nil,
				"BRA", campusID, true, updatedAt, updatedAt, nil,
			))

		records, err := repo.ListUpdatedSince(context.Background(), campusID, since, 100)
		require.NoError(t, err)
		require.Len(t, records, 1)
		assert.Equal(t, personID, records[0].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty slice when no records match", func(t *testing.T) {
		repo, mock := newPersonRepoMock(t)
		campusID := uuid.New()
		since := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

		mock.ExpectQuery(`SELECT .*FROM person.*WHERE campus_id = \$1 AND updated_at > \$2`).
			WithArgs(campusID, since, 100).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "full_name", "birth_date", "document_type", "document_number",
				"gender", "email", "phone", "photo_url", "referral_source",
				"nationality", "campus_id", "is_active", "created_at", "updated_at", "created_by",
			}))

		records, err := repo.ListUpdatedSince(context.Background(), campusID, since, 100)
		require.NoError(t, err)
		assert.Empty(t, records)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPersonRepository_CreateWithSync(t *testing.T) {
	t.Run("inserts person with sync_id column", func(t *testing.T) {
		repo, mock := newPersonRepoMock(t)
		campusID := uuid.New()
		personID := uuid.New()
		syncID := uuid.New()
		now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)

		var (
			nilTime *time.Time
			nilStr  *string
			nilUUID *uuid.UUID
		)
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO person.*sync_id.*RETURNING created_at, updated_at`).
			WithArgs(
				personID, "Maria", nilTime, "CPF", nilStr,
				nilStr, nilStr, nilStr, nilStr, nilStr,
				"BRA", campusID, true, nilUUID, syncID,
			).
			WillReturnRows(pgxmock.NewRows([]string{"created_at", "updated_at"}).AddRow(now, now))
		mock.ExpectCommit()

		p := domain.Person{
			ID:           personID,
			FullName:     "Maria",
			DocumentType: "CPF",
			Nationality:  "BRA",
			CampusID:     campusID,
			IsActive:     true,
		}
		out, err := repo.CreateWithSync(context.Background(), p, nil, syncID)
		require.NoError(t, err)
		assert.Equal(t, personID, out.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
