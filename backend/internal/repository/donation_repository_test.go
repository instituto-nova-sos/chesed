package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDonationRepoMock(t *testing.T) (*DonationRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return NewDonationRepository(mock), mock
}

// TestDonationRepository_FindByID_SelectsDonorDocument pins the receipt
// requirement that FindByID resolves the donor's document number (S11.5).
func TestDonationRepository_FindByID_SelectsDonorDocument(t *testing.T) {
	repo, mock := newDonationRepoMock(t)

	id := uuid.New()
	campusID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT d\.id,.*p\.full_name, p\.document_number, c\.name.*FROM donation d`).
		WithArgs(id, campusID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "donor_person_id", "campaign_id", "campus_id", "donation_type",
			"amount", "currency", "item_description", "donation_date",
			"receipt_number", "receipt_issued_at", "notes", "registered_by",
			"created_at", "updated_at", "full_name", "document_number", "name",
		}).AddRow(
			id, nil, nil, campusID, "FINANCIAL",
			nil, "BRL", nil, now,
			nil, nil, nil, uuid.New(),
			now, now, ptr("Maria Silva"), ptr("529.982.247-25"), nil,
		))

	detail, err := repo.FindByID(context.Background(), id, campusID)
	require.NoError(t, err)
	require.NotNil(t, detail.DonorDocument)
	assert.Equal(t, "529.982.247-25", *detail.DonorDocument)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDonationRepository_MarkReceiptIssued(t *testing.T) {
	id := uuid.New()
	campusID := uuid.New()
	issuedAt := time.Now()

	t.Run("stamps the receipt on an unissued donation", func(t *testing.T) {
		repo, mock := newDonationRepoMock(t)
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE donation SET receipt_number = $3, receipt_issued_at = $4, updated_at = NOW() WHERE id = $1 AND campus_id = $2 AND receipt_number IS NULL`)).
			WithArgs(id, campusID, "REC-2026-1A2B3C4D", issuedAt).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.MarkReceiptIssued(context.Background(), id, campusID, "REC-2026-1A2B3C4D", issuedAt)
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("already issued affects zero rows and maps to ErrDuplicate", func(t *testing.T) {
		repo, mock := newDonationRepoMock(t)
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE donation SET receipt_number = $3, receipt_issued_at = $4, updated_at = NOW() WHERE id = $1 AND campus_id = $2 AND receipt_number IS NULL`)).
			WithArgs(id, campusID, "REC-2026-XXXX", issuedAt).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.MarkReceiptIssued(context.Background(), id, campusID, "REC-2026-XXXX", issuedAt)
		assert.ErrorIs(t, err, domain.ErrDuplicate)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("unique violation maps to ErrDuplicate", func(t *testing.T) {
		repo, mock := newDonationRepoMock(t)
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE donation SET receipt_number = $3, receipt_issued_at = $4, updated_at = NOW() WHERE id = $1 AND campus_id = $2 AND receipt_number IS NULL`)).
			WithArgs(id, campusID, "REC-DUP", issuedAt).
			WillReturnError(&pgconn.PgError{Code: "23505"})

		err := repo.MarkReceiptIssued(context.Background(), id, campusID, "REC-DUP", issuedAt)
		assert.ErrorIs(t, err, domain.ErrDuplicate)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDonationRepository_FindReceiptData(t *testing.T) {
	repo, mock := newDonationRepoMock(t)
	id := uuid.New()
	campusID := uuid.New()
	now := time.Now()
	amount := 100.0

	mock.ExpectQuery(`FROM donation d.*JOIN campus cp ON cp\.id = d\.campus_id`).
		WithArgs(id, campusID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "donor_person_id", "campaign_id", "campus_id", "donation_type",
			"amount", "currency", "item_description", "donation_date",
			"receipt_number", "receipt_issued_at", "notes", "registered_by",
			"created_at", "updated_at",
			"full_name", "document_number",
			"campus_name", "legal_name", "cnpj", "address_line", "city_name", "state_code", "zip_code",
		}).AddRow(
			id, nil, nil, campusID, "FINANCIAL",
			&amount, "BRL", nil, now,
			nil, nil, nil, uuid.New(),
			now, now,
			ptr("Maria Silva"), ptr("529.982.247-25"),
			"Campus Central", ptr("Instituto Nova SOS"), ptr("12.345.678/0001-90"),
			ptr("Rua X, 1"), ptr("São Paulo"), ptr("SP"), ptr("01234-567"),
		))

	data, err := repo.FindReceiptData(context.Background(), id, campusID)
	require.NoError(t, err)
	assert.Equal(t, "Campus Central", data.CampusName)
	require.NotNil(t, data.IssuerName)
	assert.Equal(t, "Instituto Nova SOS", *data.IssuerName)
	require.NotNil(t, data.DonorDocument)
	assert.Equal(t, "529.982.247-25", *data.DonorDocument)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDonationRepository_FindReceiptData_NotFound(t *testing.T) {
	repo, mock := newDonationRepoMock(t)
	id := uuid.New()
	campusID := uuid.New()

	mock.ExpectQuery(`FROM donation d.*JOIN campus cp`).
		WithArgs(id, campusID).
		WillReturnError(pgx.ErrNoRows)

	_, err := repo.FindReceiptData(context.Background(), id, campusID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func ptr(s string) *string { return &s }
