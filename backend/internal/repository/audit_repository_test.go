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

func newAuditRepoMock(t *testing.T) (*AuditRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return NewAuditRepository(mock), mock
}

func TestAuditRepository_List(t *testing.T) {
	t.Run("campus-only filter returns items with resolved user_email", func(t *testing.T) {
		repo, mock := newAuditRepoMock(t)
		campusID := uuid.New()
		logID := uuid.New()
		entityID := uuid.New()
		email := "maria@example.com"
		desc := "Updated phone number"
		ip := "192.168.1.1"
		ts := time.Date(2026, 4, 2, 10, 30, 0, 0, time.UTC)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM audit_log a WHERE a.campus_id = $1`)).
			WithArgs(campusID).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

		mock.ExpectQuery(`SELECT a\.id, u\.email, a\.action_type, a\.entity_type, a\.entity_id, a\.description, a\.old_values, a\.new_values, host\(a\.ip_address\), a\.timestamp.*FROM audit_log a.*LEFT JOIN app_user u ON u\.keycloak_subject_id = a\.user_id::text.*WHERE a\.campus_id = \$1.*ORDER BY a\.timestamp DESC.*LIMIT \$2 OFFSET \$3`).
			WithArgs(campusID, 50, 0).
			WillReturnRows(pgxmock.NewRows(
				[]string{"id", "email", "action_type", "entity_type", "entity_id", "description", "old_values", "new_values", "host", "timestamp"},
			).AddRow(logID, &email, "UPDATE", "person", &entityID, &desc, []byte(`{"phone":"x"}`), []byte(`{"phone":"y"}`), &ip, ts))

		result, err := repo.List(context.Background(), domain.AuditLogFilter{
			CampusID: campusID,
			Page:     1,
			PerPage:  50,
		})
		require.NoError(t, err)
		require.Len(t, result.Data, 1)
		assert.Equal(t, logID, result.Data[0].ID)
		require.NotNil(t, result.Data[0].UserEmail)
		assert.Equal(t, email, *result.Data[0].UserEmail)
		assert.Equal(t, "UPDATE", result.Data[0].ActionType)
		assert.JSONEq(t, `{"phone":"y"}`, string(result.Data[0].NewValues))
		assert.Equal(t, 1, result.Pagination.Total)
		assert.Equal(t, 50, result.Pagination.PerPage)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("all optional filters appear in WHERE with correct positional args", func(t *testing.T) {
		repo, mock := newAuditRepoMock(t)
		campusID := uuid.New()
		userID := uuid.New()
		entityType := "person"
		actionType := "UPDATE"
		start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM audit_log a WHERE a\.campus_id = \$1 AND a\.user_id = \$2 AND a\.entity_type = \$3 AND a\.action_type = \$4 AND a\.timestamp >= \$5 AND a\.timestamp < \$6`).
			WithArgs(campusID, userID, entityType, actionType, start, end.AddDate(0, 0, 1)).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

		mock.ExpectQuery(`SELECT a\.id.*FROM audit_log a.*WHERE a\.campus_id = \$1 AND a\.user_id = \$2 AND a\.entity_type = \$3 AND a\.action_type = \$4 AND a\.timestamp >= \$5 AND a\.timestamp < \$6.*LIMIT \$7 OFFSET \$8`).
			WithArgs(campusID, userID, entityType, actionType, start, end.AddDate(0, 0, 1), 50, 0).
			WillReturnRows(pgxmock.NewRows(
				[]string{"id", "email", "action_type", "entity_type", "entity_id", "description", "old_values", "new_values", "host", "timestamp"},
			))

		result, err := repo.List(context.Background(), domain.AuditLogFilter{
			CampusID:   campusID,
			UserID:     &userID,
			EntityType: &entityType,
			ActionType: &actionType,
			Start:      &start,
			End:        &end,
			Page:       1,
			PerPage:    50,
		})
		require.NoError(t, err)
		assert.Empty(t, result.Data)
		assert.Equal(t, 0, result.Pagination.Total)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty result returns non-nil slice", func(t *testing.T) {
		repo, mock := newAuditRepoMock(t)
		campusID := uuid.New()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM audit_log a WHERE a.campus_id = $1`)).
			WithArgs(campusID).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`SELECT a\.id.*FROM audit_log a`).
			WithArgs(campusID, 50, 0).
			WillReturnRows(pgxmock.NewRows(
				[]string{"id", "email", "action_type", "entity_type", "entity_id", "description", "old_values", "new_values", "host", "timestamp"},
			))

		result, err := repo.List(context.Background(), domain.AuditLogFilter{CampusID: campusID, Page: 1, PerPage: 50})
		require.NoError(t, err)
		assert.NotNil(t, result.Data)
		assert.Empty(t, result.Data)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
