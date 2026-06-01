package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Querier exposes the subset of *pgxpool.Pool methods used by repositories
// that need transactional + read/write access. Both *pgxpool.Pool and
// pgxmock.PgxPoolIface satisfy this interface, enabling SQL-level unit tests
// without a live Postgres.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}
