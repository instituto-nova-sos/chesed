package repository

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeQuerier is a distinct Querier implementation used to prove that
// QuerierFrom returns the context-installed value and not the fallback.
type fakeQuerier struct {
	Querier
	id string
}

func TestQuerierFrom_ReturnsFallbackWhenNoContextQuerier(t *testing.T) {
	pool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer pool.Close()

	got := QuerierFrom(context.Background(), pool)

	assert.Same(t, Querier(pool), got, "with no request Querier in context, the fallback pool must be used")
}

func TestQuerierFrom_PrefersContextQuerier(t *testing.T) {
	pool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer pool.Close()

	tx := &fakeQuerier{id: "request-tx"}
	ctx := NewQuerierContext(context.Background(), tx)

	got := QuerierFrom(ctx, pool)

	assert.Same(t, Querier(tx), got, "when a request Querier is installed, it must take precedence over the pool")
}

func TestBase_ResolvesQuerierFromContext(t *testing.T) {
	pool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer pool.Close()

	b := base{pool: pool}

	assert.Same(t, Querier(pool), b.q(context.Background()), "base.q falls back to the pool")

	tx := &fakeQuerier{id: "request-tx"}
	assert.Same(t, Querier(tx), b.q(NewQuerierContext(context.Background(), tx)), "base.q prefers the request Querier")
}
