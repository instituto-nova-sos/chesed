package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newComplianceRepoMock(t *testing.T) (*ComplianceReportRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return NewComplianceReportRepository(mock), mock
}

func TestComplianceReportRepository_BuildComplianceReport(t *testing.T) {
	t.Run("aggregates campus-scoped metrics", func(t *testing.T) {
		repo, mock := newComplianceRepoMock(t)
		campusID := uuid.New()
		start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

		// consents by type + active/revoked (single grouped query)
		mock.ExpectQuery(regexp.QuoteMeta(`FROM consent`)).
			WithArgs(campusID, start, end).
			WillReturnRows(pgxmock.NewRows([]string{"consent_type", "count", "active", "revoked"}).
				AddRow("DATA_PROCESSING", 10, 8, 2).
				AddRow("IMAGE_USAGE", 3, 3, 0))

		// anonymized subjects
		mock.ExpectQuery(regexp.QuoteMeta(`FROM person`)).
			WithArgs(campusID).
			WillReturnRows(pgxmock.NewRows([]string{"anonymized", "total"}).AddRow(1, 40))

		// documents stored
		mock.ExpectQuery(regexp.QuoteMeta(`FROM document`)).
			WithArgs(campusID).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(12))

		report, err := repo.BuildComplianceReport(context.Background(), domain.ComplianceReportFilter{
			CampusID: campusID, Start: start, End: end,
		})
		require.NoError(t, err)
		assert.Equal(t, 10, report.ConsentsByType["DATA_PROCESSING"])
		assert.Equal(t, 3, report.ConsentsByType["IMAGE_USAGE"])
		assert.Equal(t, 11, report.ActiveConsents)
		assert.Equal(t, 2, report.RevokedConsents)
		assert.Equal(t, 1, report.AnonymizedSubjects)
		assert.Equal(t, 40, report.DataSubjects)
		assert.Equal(t, 12, report.DocumentsStored)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty consent set yields zeroed report", func(t *testing.T) {
		repo, mock := newComplianceRepoMock(t)
		campusID := uuid.New()
		start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

		mock.ExpectQuery(regexp.QuoteMeta(`FROM consent`)).
			WithArgs(campusID, start, end).
			WillReturnRows(pgxmock.NewRows([]string{"consent_type", "count", "active", "revoked"}))
		mock.ExpectQuery(regexp.QuoteMeta(`FROM person`)).
			WithArgs(campusID).
			WillReturnRows(pgxmock.NewRows([]string{"anonymized", "total"}).AddRow(0, 0))
		mock.ExpectQuery(regexp.QuoteMeta(`FROM document`)).
			WithArgs(campusID).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

		report, err := repo.BuildComplianceReport(context.Background(), domain.ComplianceReportFilter{
			CampusID: campusID, Start: start, End: end,
		})
		require.NoError(t, err)
		assert.Empty(t, report.ConsentsByType)
		assert.Equal(t, 0, report.ActiveConsents)
		assert.Equal(t, 0, report.DataSubjects)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
