package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetentionRepository_ListExpiredPersonIDs(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	repo := NewRetentionRepository(mock)

	campusID := uuid.New()
	cutoff := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	id1, id2 := uuid.New(), uuid.New()

	mock.ExpectQuery(`SELECT id FROM person\s+WHERE campus_id = \$1\s+AND anonymized_at IS NULL\s+AND updated_at < \$2`).
		WithArgs(campusID, cutoff).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(id1).AddRow(id2))

	ids, err := repo.ListExpiredPersonIDs(context.Background(), campusID, cutoff)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{id1, id2}, ids)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRetentionRepository_ListExpiredPersonIDs_Empty(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	repo := NewRetentionRepository(mock)

	campusID := uuid.New()
	cutoff := time.Now()

	mock.ExpectQuery(`SELECT id FROM person`).
		WithArgs(campusID, cutoff).
		WillReturnRows(pgxmock.NewRows([]string{"id"}))

	ids, err := repo.ListExpiredPersonIDs(context.Background(), campusID, cutoff)
	require.NoError(t, err)
	assert.Empty(t, ids)
	require.NoError(t, mock.ExpectationsWereMet())
}
