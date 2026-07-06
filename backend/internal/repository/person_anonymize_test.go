package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/instituto-nova-sos/chesed/internal/domain"
)

func TestPersonRepository_Anonymize(t *testing.T) {
	t.Run("scrubs person PII and its address rows", func(t *testing.T) {
		repo, mock := newPersonRepoMock(t)
		personID := uuid.New()
		campusID := uuid.New()

		mock.ExpectExec(`UPDATE person SET .*full_name = '\[ANONYMIZED\]'.*WHERE id = \$1 AND campus_id = \$2`).
			WithArgs(personID, campusID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`UPDATE address SET .*WHERE person_id = \$1`).
			WithArgs(personID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.Anonymize(context.Background(), personID, campusID)
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrNotFound when the person update affects no rows", func(t *testing.T) {
		repo, mock := newPersonRepoMock(t)
		personID := uuid.New()
		campusID := uuid.New()

		mock.ExpectExec(`UPDATE person SET .*WHERE id = \$1 AND campus_id = \$2`).
			WithArgs(personID, campusID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.Anonymize(context.Background(), personID, campusID)
		assert.True(t, errors.Is(err, domain.ErrNotFound))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("anonymizing two different persons of the same document type does not error", func(t *testing.T) {
		repo, mock := newPersonRepoMock(t)
		campusID := uuid.New()
		first := uuid.New()
		second := uuid.New()

		// The 'ANON-'||id sentinel keeps document_number unique per person, so a
		// second anonymization never trips uq_person_document.
		for _, id := range []uuid.UUID{first, second} {
			mock.ExpectExec(`UPDATE person SET .*document_number = 'ANON-'`).
				WithArgs(id, campusID).
				WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			mock.ExpectExec(`UPDATE address SET .*WHERE person_id = \$1`).
				WithArgs(id).
				WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		}

		require.NoError(t, repo.Anonymize(context.Background(), first, campusID))
		require.NoError(t, repo.Anonymize(context.Background(), second, campusID))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
