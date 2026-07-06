//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/handler"
	"github.com/instituto-nova-sos/chesed/internal/middleware"
	"github.com/instituto-nova-sos/chesed/internal/repository"
	"github.com/instituto-nova-sos/chesed/internal/service"
)

// appConnString rewrites the owner connection string to log in as the non-owner
// RLS role chesed_app, which migration 000028 creates. Only chesed_app is
// subject to the row-level security policies; the owner role bypasses them.
func appConnString(t *testing.T, ownerConn string) string {
	t.Helper()
	u, err := url.Parse(ownerConn)
	require.NoError(t, err)
	u.User = url.UserPassword("chesed_app", "chesed_app")
	return u.String()
}

// openAppPool connects as chesed_app so queries are subject to RLS. Migration
// 000028 creates the role passwordless (the real password is set by the Postgres
// init script in compose); the testcontainers path has no init script, so we set
// the password here, as the owner, before connecting.
func openAppPool(t *testing.T, h *testHarness) *pgxpool.Pool {
	t.Helper()
	_, err := h.pool.Exec(context.Background(), "ALTER ROLE chesed_app LOGIN PASSWORD 'chesed_app'")
	require.NoError(t, err, "set chesed_app password for the test connection")

	pool, err := pgxpool.New(context.Background(), appConnString(t, h.connStr))
	require.NoError(t, err, "connect as chesed_app")
	t.Cleanup(pool.Close)
	return pool
}

// withCampusGUC runs fn inside a transaction on the app pool with the campus GUC
// set, mirroring what the CampusTx middleware does per request.
func withCampusGUC(t *testing.T, pool *pgxpool.Pool, campusID *uuid.UUID, fn func(tx pgx.Tx)) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	if campusID != nil {
		_, err = tx.Exec(ctx, "SELECT set_config('app.current_campus', $1, true)", campusID.String())
		require.NoError(t, err)
	}
	fn(tx)
}

// TestRLS_CrossCampusReadBlockedWithoutAppFilter proves database-level isolation:
// a bare SELECT with NO campus_id filter still returns only the session campus's
// rows when running as the app role with the GUC set.
func TestRLS_CrossCampusReadBlockedWithoutAppFilter(t *testing.T) {
	h := freshHarness(t)
	defer h.Close()

	campusA := h.campusID
	campusB := seedSecondCampus(t, h)
	seedPerson(t, h, campusA, "Alice A", "11111111111")
	seedPerson(t, h, campusB, "Bob B", "22222222222")

	appPool := openAppPool(t, h)

	withCampusGUC(t, appPool, &campusA, func(tx pgx.Tx) {
		rows, err := tx.Query(context.Background(), `SELECT full_name FROM person`)
		require.NoError(t, err)
		defer rows.Close()

		var names []string
		for rows.Next() {
			var n string
			require.NoError(t, rows.Scan(&n))
			names = append(names, n)
		}
		require.NoError(t, rows.Err())

		assert.Contains(t, names, "Alice A", "session campus rows must be visible")
		assert.NotContains(t, names, "Bob B", "other campus rows must be invisible even without an app-level filter")
	})
}

// TestRLS_UnsetGUCReturnsZeroRows proves fail-closed behavior: with no campus GUC
// set, the app role sees no rows (never all rows).
func TestRLS_UnsetGUCReturnsZeroRows(t *testing.T) {
	h := freshHarness(t)
	defer h.Close()

	seedPerson(t, h, h.campusID, "Alice A", "11111111111")
	appPool := openAppPool(t, h)

	withCampusGUC(t, appPool, nil, func(tx pgx.Tx) {
		var count int
		err := tx.QueryRow(context.Background(), `SELECT count(*) FROM person`).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "an unset campus GUC must fail closed to zero rows")
	})
}

// TestRLS_CrossCampusWriteRejected proves the WITH CHECK clause blocks inserting
// a row for a campus other than the session campus.
func TestRLS_CrossCampusWriteRejected(t *testing.T) {
	h := freshHarness(t)
	defer h.Close()

	campusA := h.campusID
	campusB := seedSecondCampus(t, h)
	appPool := openAppPool(t, h)

	withCampusGUC(t, appPool, &campusA, func(tx pgx.Tx) {
		_, err := tx.Exec(context.Background(),
			`INSERT INTO person (id, full_name, document_type, campus_id, nationality)
			 VALUES ($1, 'Mallory', 'CPF', $2, 'BRA')`,
			uuid.New(), campusB)
		require.Error(t, err, "writing a row for another campus must be rejected by WITH CHECK")
	})
}

// TestRLS_JoinInheritedTableIsolation proves EXISTS-based policies isolate tables
// that carry no campus_id column (address inherits campus via person).
func TestRLS_JoinInheritedTableIsolation(t *testing.T) {
	h := freshHarness(t)
	defer h.Close()

	campusA := h.campusID
	campusB := seedSecondCampus(t, h)
	personA := seedPerson(t, h, campusA, "Alice A", "11111111111")
	personB := seedPerson(t, h, campusB, "Bob B", "22222222222")

	_, err := h.pool.Exec(context.Background(),
		`INSERT INTO address (id, person_id, city) VALUES ($1, $2, 'Sao Paulo')`, uuid.New(), personA)
	require.NoError(t, err)
	_, err = h.pool.Exec(context.Background(),
		`INSERT INTO address (id, person_id, city) VALUES ($1, $2, 'Lisbon')`, uuid.New(), personB)
	require.NoError(t, err)

	appPool := openAppPool(t, h)
	withCampusGUC(t, appPool, &campusA, func(tx pgx.Tx) {
		rows, err := tx.Query(context.Background(), `SELECT city FROM address`)
		require.NoError(t, err)
		defer rows.Close()

		var cities []string
		for rows.Next() {
			var c string
			require.NoError(t, rows.Scan(&c))
			cities = append(cities, c)
		}
		require.NoError(t, rows.Err())

		assert.Contains(t, cities, "Sao Paulo")
		assert.NotContains(t, cities, "Lisbon", "addresses inherit campus isolation via their person")
	})
}

// TestRLS_OwnerBypassesRLS confirms migrations/seed (owner role) still see all
// rows with no GUC — otherwise migrations would break.
func TestRLS_OwnerBypassesRLS(t *testing.T) {
	h := freshHarness(t)
	defer h.Close()

	seedPerson(t, h, h.campusID, "Alice A", "11111111111")
	seedPerson(t, h, seedSecondCampus(t, h), "Bob B", "22222222222")

	var count int
	err := h.pool.QueryRow(context.Background(), `SELECT count(*) FROM person`).Scan(&count)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 2, "the owner role must bypass RLS and see all rows")
}

// buildPreCampusRouter mounts /self-register and /auth/me exactly as production
// does: the app repositories run on the non-owner appPool (RLS-subject), and the
// routes are wrapped in BypassRLS(ownerPool) so these pre-campus flows run on the
// RLS-bypassing owner connection. Auth claims are injected directly (as the OIDC
// middleware would) via the withClaims wrapper.
func buildPreCampusRouter(appPool, ownerPool *pgxpool.Pool, claims auth.AuthClaims) chi.Router {
	auditRepo := repository.NewAuditRepository(ownerPool)
	personRepo := repository.NewPersonRepository(appPool)
	personRoleRepo := repository.NewPersonRoleRepository(ownerPool)
	userRepo := repository.NewUserRepository(ownerPool)
	agreementRepo := repository.NewVolunteerAgreementRepository(ownerPool)
	campusRepo := repository.NewCampusRepository(ownerPool)

	auditSvc := service.NewAuditService(auditRepo)
	selfRegSvc := service.NewSelfRegisterService(personRepo, personRoleRepo, userRepo, agreementRepo, auditSvc)
	onboardingSvc := service.NewOnboardingService(userRepo, personRepo, personRoleRepo, agreementRepo, campusRepo, auditSvc)

	selfRegH := handler.NewSelfRegisterHandler(selfRegSvc)
	onboardingH := handler.NewOnboardingHandler(onboardingSvc)

	injectClaims := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(auth.NewContext(r.Context(), claims)))
		})
	}

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(injectClaims)
		r.Use(middleware.BypassRLS(ownerPool))
		r.Post("/api/v1/self-register", selfRegH.Register)
		r.Get("/api/v1/auth/me", onboardingH.GetStatus)
	})
	return r
}

// TestRLS_SelfRegisterAndOnboardingBypassRLS proves the pre-campus routes work
// when the app connects as the non-owner chesed_app role: without BypassRLS the
// person INSERT would be rejected by WITH CHECK and the onboarding global lookup
// would fail closed to zero rows. Regression guard for BLOCKER-1.
func TestRLS_SelfRegisterAndOnboardingBypassRLS(t *testing.T) {
	h := freshHarness(t)
	defer h.Close()

	appPool := openAppPool(t, h)

	subject := uuid.New().String()
	claims := auth.AuthClaims{
		Subject:       subject,
		Email:         "self-register@chesed.test",
		EmailVerified: true,
		Roles:         []string{"VOLUNTEER"},
		CampusID:      h.campusID,
	}
	router := buildPreCampusRouter(appPool, h.pool, claims)

	// Self-register: the person INSERT (campus_id = h.campusID) must succeed even
	// though the app pool is RLS-subject, because BypassRLS routes it to the owner.
	body, _ := json.Marshal(map[string]any{
		"full_name":     "Self Registrant",
		"role_type":     "VOLUNTEER",
		"campus_id":     h.campusID.String(),
		"document_type": "CPF",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/self-register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, "self-register must succeed as chesed_app via BypassRLS: %s", rec.Body.String())

	// The row must exist in Postgres.
	var count int
	require.NoError(t, h.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM person WHERE full_name = 'Self Registrant'`).Scan(&count))
	assert.Equal(t, 1, count, "self-registered person must persist")

	// Onboarding: the cross-campus email lookup must find the just-created person
	// (fail-closed RLS would return zero rows without BypassRLS).
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meRec := httptest.NewRecorder()
	router.ServeHTTP(meRec, meReq)
	require.Equal(t, http.StatusOK, meRec.Code, meRec.Body.String())

	var status struct {
		CampusID *string `json:"campus_id"`
	}
	require.NoError(t, json.Unmarshal(meRec.Body.Bytes(), &status))
	require.NotNil(t, status.CampusID, "onboarding must resolve the campus via the global lookup (not fail-closed)")
	assert.Equal(t, h.campusID.String(), *status.CampusID)
}
