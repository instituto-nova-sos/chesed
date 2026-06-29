//go:build integration

// Package integration_test owns end-to-end tests that exercise the real
// HTTP stack against a real PostgreSQL instance.
//
// These tests are gated behind the `integration` build tag so the standard
// `go test ./...` flow stays fast and Docker-free. Run them via:
//
//	make test-integration
//
// or directly:
//
//	go test -tags integration ./internal/integration/...
//
// Each test spins up an ephemeral Postgres container via testcontainers-go,
// applies all repository migrations from disk, and exercises the real
// chi router + service + repository stack. Auth middleware is bypassed —
// tests inject a synthetic AuthClaims context to focus the assertion
// surface on the application behaviour rather than the Keycloak handshake.
package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/handler"
	"github.com/instituto-nova-sos/chesed/internal/repository"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

// testHarness wires a real Postgres + the production stack so each test can
// exercise the HTTP surface end-to-end without mocks.
type testHarness struct {
	t        *testing.T
	pool     *pgxpool.Pool
	router   chi.Router
	campusID uuid.UUID
	cleanup  func()
}

// newHarness boots a fresh Postgres container, applies all migrations,
// seeds the default campus from migration 000011, and returns a fully
// wired chi router with the sync routes mounted.
func newHarness(t *testing.T) *testHarness {
	t.Helper()
	ctx := context.Background()

	pgC, err := tcpg.Run(ctx,
		"postgres:16-alpine",
		tcpg.WithDatabase("chesed_test"),
		tcpg.WithUsername("test"),
		tcpg.WithPassword("test"),
		tcpg.BasicWaitStrategies(),
	)
	require.NoError(t, err, "start postgres container")

	connStr, err := pgC.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	migrationsURL := "file://" + migrationsPath(t)
	m, err := migrate.New(migrationsURL, connStr)
	require.NoError(t, err)
	require.NoError(t, m.Up(), "apply migrations")
	srcErr, dbErr := m.Close()
	require.NoError(t, srcErr)
	require.NoError(t, dbErr)

	campusID := seedDefaultCampus(t, pool)

	router := buildRouter(pool)

	return &testHarness{
		t:        t,
		pool:     pool,
		router:   router,
		campusID: campusID,
		cleanup: func() {
			pool.Close()
			_ = pgC.Terminate(context.Background())
		},
	}
}

func (h *testHarness) Close() { h.cleanup() }

// migrationsPath walks up from the test file to locate backend/migrations.
// Using runtime.Caller keeps the path stable regardless of where go test
// is invoked from.
func migrationsPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	// internal/integration/harness_test.go → backend/migrations
	dir := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	return filepath.Join(dir, "migrations")
}

// seedDefaultCampus returns the campus_id created by migration 000011. The
// migration inserts a single fixed-name campus; we look it up rather than
// hard-coding the UUID so renames upstream still work.
func seedDefaultCampus(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(),
		`SELECT id FROM campus ORDER BY created_at LIMIT 1`,
	).Scan(&id)
	require.NoError(t, err, "seed campus from migration 000011")
	return id
}

// buildRouter mounts only the routes under test. We skip auth middleware
// because integration tests inject AuthClaims directly via context; the
// OIDC handshake is exercised by the Session 9 manual verification flow.
func buildRouter(pool *pgxpool.Pool) chi.Router {
	auditRepo := repository.NewAuditRepository(pool)
	personRepo := repository.NewPersonRepository(pool)
	triageRepo := repository.NewTriageRepository(pool)
	attendanceRepo := repository.NewAttendanceRepository(pool)

	auditSvc := service.NewAuditService(auditRepo)
	syncSvc := service.NewSyncService(personRepo, triageRepo, attendanceRepo, auditSvc)

	syncH := handler.NewSyncHandler(syncSvc)

	r := chi.NewRouter()
	r.Route("/api/v1/sync", func(r chi.Router) {
		r.Post("/push", syncH.Push)
		r.Get("/pull", syncH.Pull)
	})
	return r
}

// authedRequest attaches a synthetic AuthClaims context so the service can
// resolve campus_id without going through the Keycloak middleware chain.
func (h *testHarness) authedRequest(req *http.Request, opts ...func(*auth.AuthClaims)) *http.Request {
	claims := auth.AuthClaims{
		Subject:       uuid.New().String(),
		Email:         "tester@chesed.test",
		EmailVerified: true,
		Roles:         []string{"VOLUNTEER"},
		CampusID:      h.campusID,
	}
	for _, opt := range opts {
		opt(&claims)
	}
	return req.WithContext(auth.NewContext(req.Context(), claims))
}

// doRequest executes the request against the harness router and returns the
// recorder. Helper exists so individual tests stay focused on assertions.
func (h *testHarness) doRequest(req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.router.ServeHTTP(rec, req)
	return rec
}

// withinSeconds is a small helper for assertions where the server timestamp
// is fresh — keeps test code readable without taking a dependency on tzdata.
func withinSeconds(actual time.Time, seconds int) bool {
	delta := time.Since(actual)
	if delta < 0 {
		delta = -delta
	}
	return delta < time.Duration(seconds)*time.Second
}

// freshHarness is a small wrapper that registers cleanup automatically and
// returns ready-to-use plumbing. Use this in tests instead of newHarness
// directly to avoid forgetting Close().
func freshHarness(t *testing.T) *testHarness {
	t.Helper()
	h := newHarness(t)
	t.Cleanup(h.Close)
	return h
}

// fmtSyncURL is a tiny escape hatch for tests that need to assemble URL
// query strings without importing net/url at every call site.
func fmtSyncURL(path, query string) string {
	if query == "" {
		return path
	}
	return fmt.Sprintf("%s?%s", path, query)
}
