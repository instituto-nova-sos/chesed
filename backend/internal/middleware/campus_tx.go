package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

// campusGUC is the PostgreSQL session variable read by the RLS policies via
// current_setting('app.current_campus', true). It is set with SET LOCAL so it
// lives only for the request transaction and is cleared on COMMIT/ROLLBACK,
// making pooled-connection reuse across requests safe.
const campusGUC = "app.current_campus"

// campusTx is the per-request transaction the middleware manages. It must be a
// repository.Querier so repositories can run their queries inside it (with the
// campus GUC set), plus the transaction lifecycle methods the middleware owns.
type campusTx interface {
	repository.Querier
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// txBeginner opens a request-scoped campus transaction. It is an interface so
// the middleware's commit/rollback logic can be unit-tested with a fake, while
// production uses a *pgxpool.Pool.
type txBeginner interface {
	BeginCampusTx(ctx context.Context) (campusTx, error)
}

// poolBeginner adapts a *pgxpool.Pool to txBeginner. pgx.Tx already satisfies
// campusTx (it implements Query/QueryRow/Exec/Begin plus Commit/Rollback).
type poolBeginner struct {
	pool *pgxpool.Pool
}

func (p poolBeginner) BeginCampusTx(ctx context.Context) (campusTx, error) {
	return p.pool.Begin(ctx)
}

// CampusTx opens a per-request transaction, sets the campus GUC with SET LOCAL,
// installs the transaction as the request Querier, and commits on a 2xx/3xx
// response or rolls back on a >=400 status or a panic. It MUST run after
// AutoProvision, which resolves the campus into the auth context.
func CampusTx(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return campusTxWith(poolBeginner{pool: pool})
}

func campusTxWith(beginner txBeginner) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			campusID := auth.CampusIDFromContext(r.Context())
			if campusID == (auth.AuthClaims{}).CampusID {
				writeError(w, http.StatusForbidden, "forbidden", "missing campus assignment")
				return
			}

			ctx := r.Context()
			tx, err := beginner.BeginCampusTx(ctx)
			if err != nil {
				slog.ErrorContext(ctx, "middleware.CampusTx: begin failed", "error", err.Error())
				writeError(w, http.StatusInternalServerError, "internal_error", "could not open request transaction")
				return
			}

			// Parameterized SET LOCAL: the campus is a validated UUID, and the
			// value binds as text, so there is no injection surface.
			if _, err := tx.Exec(ctx, "SELECT set_config($1, $2, true)", campusGUC, campusID.String()); err != nil {
				_ = tx.Rollback(ctx)
				slog.ErrorContext(ctx, "middleware.CampusTx: set campus GUC failed", "error", err.Error())
				writeError(w, http.StatusInternalServerError, "internal_error", "could not scope request transaction")
				return
			}

			sw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			ctx = repository.NewQuerierContext(ctx, tx)

			committed := false
			defer func() {
				if p := recover(); p != nil {
					_ = tx.Rollback(ctx)
					panic(p)
				}
				if !committed {
					_ = tx.Rollback(ctx)
				}
			}()

			next.ServeHTTP(sw, r.WithContext(ctx))

			if sw.status >= http.StatusBadRequest {
				_ = tx.Rollback(ctx)
				return
			}
			if err := tx.Commit(ctx); err != nil {
				slog.ErrorContext(ctx, "middleware.CampusTx: commit failed", "error", err.Error())
				return
			}
			committed = true
		})
	}
}

// statusRecorder captures the response status so the middleware can decide
// whether to commit or roll back the request transaction.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if !s.wroteHeader {
		s.status = code
		s.wroteHeader = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if !s.wroteHeader {
		s.wroteHeader = true
	}
	return s.ResponseWriter.Write(b)
}
