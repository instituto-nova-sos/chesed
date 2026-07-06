package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Querier exposes the subset of *pgxpool.Pool methods used by repositories
// that need transactional + read/write access. Both *pgxpool.Pool and
// pgxmock.PgxPoolIface satisfy this interface, enabling SQL-level unit tests
// without a live Postgres. A pgx.Tx also satisfies it, which is what the
// per-request RLS transaction is carried as.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// querierCtxKey is the context key under which the per-request Querier (the
// RLS transaction) is stored by the campus-tx middleware.
type querierCtxKey struct{}

// NewQuerierContext returns a context carrying q as the request-scoped Querier.
// Queries run via QuerierFrom will then execute on q (the RLS transaction)
// rather than acquiring a fresh, GUC-less connection from the pool.
func NewQuerierContext(ctx context.Context, q Querier) context.Context {
	return context.WithValue(ctx, querierCtxKey{}, q)
}

// QuerierFrom returns the request-scoped Querier installed in ctx, or fallback
// when none is present. This lets repositories run inside the per-request RLS
// transaction in production while unit tests (which pass a plain context) fall
// back to the injected pool or mock.
func QuerierFrom(ctx context.Context, fallback Querier) Querier {
	if q, ok := ctx.Value(querierCtxKey{}).(Querier); ok && q != nil {
		return q
	}
	return fallback
}

// base is embedded by campus-scoped repositories to resolve the correct Querier
// for each call: the per-request RLS transaction when present, otherwise the
// pool the repository was constructed with.
type base struct {
	pool Querier
}

func (b base) q(ctx context.Context) Querier {
	return QuerierFrom(ctx, b.pool)
}
