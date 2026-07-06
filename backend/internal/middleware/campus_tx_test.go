package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/repository"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTx records the campus GUC argument and whether Commit/Rollback ran. It
// embeds campusTx so it satisfies the interface; only the methods the
// middleware exercises are implemented.
type fakeTx struct {
	campusTx
	setCampus  string
	committed  bool
	rolledBack bool
	execErr    error
}

func (t *fakeTx) Exec(_ context.Context, _ string, args ...any) (pgconn.CommandTag, error) {
	if t.execErr != nil {
		return pgconn.CommandTag{}, t.execErr
	}
	// set_config(name, value, is_local): the campus is the second bound arg.
	if len(args) == 2 {
		if s, ok := args[1].(string); ok {
			t.setCampus = s
		}
	}
	return pgconn.CommandTag{}, nil
}
func (t *fakeTx) Commit(_ context.Context) error   { t.committed = true; return nil }
func (t *fakeTx) Rollback(_ context.Context) error { t.rolledBack = true; return nil }

// fakeBeginner hands out a single fakeTx as the request transaction.
type fakeBeginner struct {
	tx       *fakeTx
	beginErr error
}

func (b *fakeBeginner) BeginCampusTx(_ context.Context) (campusTx, error) {
	if b.beginErr != nil {
		return nil, b.beginErr
	}
	return b.tx, nil
}

func serve(mw func(http.Handler) http.Handler, campus uuid.UUID, next http.HandlerFunc) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/persons", nil)
	ctx := auth.NewContext(req.Context(), auth.AuthClaims{CampusID: campus})
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req.WithContext(ctx))
	return rec
}

func TestCampusTx_SetsGUCAndInstallsQuerier(t *testing.T) {
	campus := uuid.New()
	tx := &fakeTx{}
	mw := campusTxWith(&fakeBeginner{tx: tx})

	var installed repository.Querier
	serve(mw, campus, func(w http.ResponseWriter, r *http.Request) {
		installed = repository.QuerierFrom(r.Context(), nil)
		w.WriteHeader(http.StatusOK)
	})

	assert.Equal(t, campus.String(), tx.setCampus, "GUC must be set to the request campus")
	assert.NotNil(t, installed, "the tx must be installed as the request Querier")
	assert.True(t, tx.committed, "a 2xx response must commit")
	assert.False(t, tx.rolledBack)
}

func TestCampusTx_RollsBackOnErrorStatus(t *testing.T) {
	tx := &fakeTx{}
	mw := campusTxWith(&fakeBeginner{tx: tx})

	serve(mw, uuid.New(), func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	assert.True(t, tx.rolledBack, "a >=400 response must roll back")
	assert.False(t, tx.committed)
}

func TestCampusTx_RollsBackOnPanic(t *testing.T) {
	tx := &fakeTx{}
	mw := campusTxWith(&fakeBeginner{tx: tx})

	assert.Panics(t, func() {
		serve(mw, uuid.New(), func(http.ResponseWriter, *http.Request) {
			panic("boom")
		})
	})
	assert.True(t, tx.rolledBack, "a panic must roll back")
}

func TestCampusTx_RejectsMissingCampus(t *testing.T) {
	tx := &fakeTx{}
	mw := campusTxWith(&fakeBeginner{tx: tx})

	rec := serve(mw, uuid.Nil, func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler must not run without a campus")
	})

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.False(t, tx.committed)
}

func TestCampusTx_BeginError(t *testing.T) {
	mw := campusTxWith(&fakeBeginner{beginErr: errors.New("pool exhausted")})

	rec := serve(mw, uuid.New(), func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler must not run when the tx cannot begin")
	})

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}
