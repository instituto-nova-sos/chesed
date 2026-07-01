# E2E Testing (Playwright, real API + Postgres, mocked Keycloak)

This guide explains how to run the Chesed end-to-end suite locally. The E2E tier
drives the **built** frontend with Playwright against a **real** Go API and
**real** PostgreSQL. **Keycloak is mocked** — a lightweight OIDC issuer signs a
token that the production token-validation middleware accepts, and the Playwright
fixture injects it so the app boots authenticated with no Keycloak login page.

For where E2E sits relative to unit/integration tests, see
`.project-ai/checklists/test-distribution.md`. For the SMOKE vs FULL tiers, see
`.project-ai/rules/e2e-test-tiers.md`.

> **CI is paused** (see `CLAUDE.md`). Everything below is run **locally** by the
> operator. There is no CI E2E job yet; re-enabling CI is a separate, approved
> change.

---

## Architecture

```
┌─────────────────────────────┐        ┌──────────────────────────────────────┐
│  Playwright (host)          │        │  docker-compose.e2e.yml              │
│                             │        │                                      │
│  vite preview  :5173 ───────┼──CORS──┼─► e2e-api (Go)        :8081          │
│  (built dist, the app)      │        │     │  validates Bearer token        │
│                             │        │     ▼  via OIDC JWKS                  │
│  fixtures.ts:               │        │   mock-oidc (issuer)  :8181          │
│   - mints token via mock    │◄───────┼─────┘  discovery + JWKS + /e2e/token  │
│   - intercepts OIDC hops    │        │                                      │
│   - seeds/cleans app_user   │──pg────┼─► e2e-db (Postgres)   :5433          │
│     + persons in Postgres   │        │     (migrations applied from disk)   │
└─────────────────────────────┘        └──────────────────────────────────────┘
```

Why these specific choices:

- **Served on port 5173** — the Go API's CORS allow-list is hardcoded to
  `http://localhost:5173` (`backend/cmd/server/main.go`). The preview MUST serve
  there or every cross-origin API call is blocked.
- **API on 8081, DB on 5433, mock issuer on 8181** — offset from the dev stack
  (`docker-compose.yml`: 8080 / 5432 / 8180) so the E2E stack can run alongside
  the dev stack without port clashes.
- **Mock Keycloak** — `frontend/e2e/mock-oidc` publishes an OIDC discovery doc +
  JWKS and mints RS256 tokens. The API fetches that JWKS and validates tokens
  through the SAME `middleware.OIDCAuth` code that runs in production. The token
  carries `email_verified=true`, a realm role, and `campus_id` per
  `docs/20-keycloak-configuration.md`.
- **The real OIDC failure paths** (expired / unverified-email-403 /
  bad-signature-401) that the mock does NOT exercise are covered by the
  compensating backend test `backend/internal/integration/auth_middleware_test.go`.

---

## Prerequisites (one-time)

1. **Docker Desktop** running (the E2E stack and the backend's testcontainers
   integration tests both need it).
2. **Install frontend dependencies** — the E2E deps (`@playwright/test`, `pg`)
   were added to `frontend/package.json`:
   ```bash
   cd frontend
   npm install
   ```
3. **Install the Playwright browser** (Chromium only — the suite uses one
   project). This downloads a browser binary and may take a minute:
   ```bash
   cd frontend
   npx playwright install chromium
   ```

---

## Running the suite

### 1. Bring up the real stack

From the repository root:

```bash
docker compose -f docker-compose.e2e.yml up -d --build
```

This builds and starts:
- `e2e-db` (Postgres 16) on host port **5433**,
- `e2e-migrate` (runs all migrations from disk, then exits),
- `mock-oidc` (the Keycloak stand-in) on host port **8181**,
- `e2e-api` (the Go API) on host port **8081**.

Wait until the API is healthy:

```bash
docker compose -f docker-compose.e2e.yml ps
curl -fsS http://localhost:8081/health   # expect a 200
curl -fsS http://localhost:8181/health   # mock issuer up
```

### 2. Run Playwright

Playwright's `webServer` config **builds** the frontend and serves it with
`vite preview` on port 5173 automatically — you do **not** need to build or serve
manually. From `frontend/`:

```bash
# SMOKE — fast, happy-path critical slice (every-merge guard):
npm run test:e2e:smoke

# FULL — all critical flows (sprint release gate):
npm run test:e2e
```

The build is pointed at the E2E API via `VITE_API_BASE_URL=http://localhost:8081/api/v1`
(injected by `playwright.config.ts`). To point at a different stack, override:

```bash
E2E_API_BASE_URL=http://localhost:8081/api/v1 \
E2E_OIDC_TOKEN_URL=http://localhost:8181/e2e/token \
E2E_DATABASE_URL=postgres://chesed:chesed@localhost:5433/chesed?sslmode=disable \
npm run test:e2e:smoke
```

### 3. Tear down

```bash
docker compose -f docker-compose.e2e.yml down -v
```

The `-v` drops the Postgres volume so the next run starts from a clean,
freshly-migrated database.

---

## Test isolation and cleanup

Each test (see `frontend/e2e/fixtures.ts`):

1. Gets a **unique Keycloak subject** and a **unique data prefix**.
2. **Seeds** an `app_user` row (subject, `ADMIN`, `campus_id` = the default campus
   from migration `000011`) directly in Postgres. This is required because the API
   resolves `campus_id` from `app_user`, not from the JWT — without a provisioned
   user, `AutoProvision` returns `403 missing campus assignment`.
3. **Authenticates** by intercepting the two OIDC hops keycloak-js performs
   (authorize redirect → callback with code; token exchange → minted token set),
   so the real keycloak-js flow runs against the mock issuer.
4. **Cleans up** in `afterEach`: deletes every `person` whose name starts with the
   test's unique prefix, then the seeded `app_user`.

There is **no dedicated cleanup API endpoint** in the backend, so teardown talks
to Postgres directly (published on 5433) and scopes deletes by the per-test name
prefix. If a cleanup endpoint is added later, swap the SQL deletes in
`fixtures.ts` for API calls.

---

## What the smoke spec covers (and what it does not)

`frontend/e2e/sync-smoke.spec.ts` (`@smoke`):

- **Active**: login (token injected) → create one person **online** → person
  visible in the list → row exists in Postgres on the scoped campus.
- **`test.fixme` (intentionally not run yet)**: the offline slice — go offline,
  create a person, confirm it stays visible from the IndexedDB cache, reconnect,
  and assert the sync queue drains to `/api/v1/sync/push`. These assertions are
  written but **skipped with a reason** because the frontend **sync drainer is not
  wired into the running app yet** (the list reads straight from the API, and no
  background drainer flushes the Dexie queue). They activate when the drainer
  ships (see `tasks/todo.md`). We do not fake a passing offline assertion.

Pull, conflict resolution, batching, and reports stay in the integration/unit
tiers — not the E2E — per `.project-ai/checklists/test-distribution.md`.

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| CORS error on API calls in the test | Preview not on port 5173 | Let Playwright's `webServer` manage the preview; don't serve manually on another port. |
| `403 missing campus assignment` | `app_user` not seeded / DB not reachable on 5433 | Confirm `e2e-db` is up and `E2E_DATABASE_URL` points at port 5433; the fixture seeds per test. |
| `401 invalid or expired token` | API can't reach the mock issuer's JWKS, or issuer URL mismatch | Confirm `mock-oidc` is healthy; `KEYCLOAK_URL` in the compose points at `http://mock-oidc:8080`. |
| keycloak-js "Invalid nonce" | Token's `nonce` ≠ the one keycloak-js generated | The fixture captures the authorize-request `nonce` and passes it to `/e2e/token`; ensure both OIDC `page.route` handlers are installed before `page.goto('/')`. |
| Playwright can't find a browser | `npx playwright install chromium` not run | Run the install step in Prerequisites. |
| `connect ECONNREFUSED` to Postgres | Stack not up, or `down -v` cleared it | `docker compose -f docker-compose.e2e.yml up -d --build` and wait for health. |
| Stale UI / old behavior in the test | A pre-built `dist` shadowing the fresh build | The config rebuilds every run; if you served manually, remove `frontend/dist` and re-run. |

---

## Files

- `docker-compose.e2e.yml` — Postgres + migrate + mock OIDC + Go API (real stack).
- `frontend/playwright.config.ts` — Chromium project, `@smoke` grep, `webServer`
  that builds + previews on 5173 pointed at the E2E API.
- `frontend/e2e/fixtures.ts` — token injection (mock Keycloak), per-test campus
  seeding + cleanup.
- `frontend/e2e/sync-smoke.spec.ts` — the `@smoke` critical slice.
- `frontend/e2e/mock-oidc/` — the test-only OIDC issuer (discovery + JWKS + mint).
- `.project-ai/rules/e2e-test-tiers.md` — SMOKE vs FULL.
- `.project-ai/checklists/e2e-critical-flows.md` — the sprint release gate.
- `backend/internal/integration/auth_middleware_test.go` — compensating real-auth test.
