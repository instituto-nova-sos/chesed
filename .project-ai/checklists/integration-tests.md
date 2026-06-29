# Checklist: Integration Tests

## Purpose

Every feature that crosses a process boundary — HTTP, SQL, or fetch — needs integration tests. Unit tests prove the code is internally consistent; integration tests prove the contract with the rest of the system holds. Both are required.

This checklist is **blocking** at the pre-merge gate. The reviewer agent must verify each item before approval.

## Trigger

Use this checklist when a PR:
- Adds or modifies any HTTP endpoint under `/api/v1/*`
- Adds or modifies any SQL migration (especially: constraints, indexes, new columns the code reads/writes)
- Adds or modifies any function in `frontend/src/api/*` that wraps a backend call
- Adds or modifies any hook in `frontend/src/hooks/*` that fetches from the API
- Adds or modifies any service-layer method that orchestrates multiple repositories

## Backend Integration Tests

Location: `backend/internal/integration/`
Build tag: `//go:build integration`
Runner: `make test-integration`
CI step: `.github/workflows/backend.yml` → `Integration tests`

- [ ] **Harness use** — Test uses `freshHarness(t)` from `harness_test.go`. No bypassing the real chi router or repository stack.
- [ ] **Happy path** — At least one test exercises the full HTTP → service → repository → real Postgres flow and asserts BOTH the HTTP response shape AND the persisted database state.
- [ ] **Campus scoping** — A test inserts data in two campuses and proves the second campus's data does NOT leak into a request scoped to the first.
- [ ] **Every documented error code** — For each error response listed in `docs/11-api-design.md`, a test reproduces the precondition and asserts the HTTP status and error code.
- [ ] **SQL constraints** — For each new UNIQUE, CHECK, FK, or partial index added in a migration, a test attempts the constraint violation and asserts the expected SQLSTATE (e.g., `23505` for unique violation).
- [ ] **Migration round-trip** — The migration `.up.sql` applies cleanly during harness boot. If the migration is destructive or non-trivial, add a test that calls `.down.sql` and asserts the schema reverts.
- [ ] **Idempotency** — For any endpoint with documented idempotency semantics (sync push, retried writes), a test calls the endpoint twice with the same idempotency key and asserts exactly one DB row exists.
- [ ] **Auth boundary** — A test confirms missing or zero campus_id returns 403, regardless of role.

## Frontend Integration Tests

Location: `frontend/src/__integration__/`
File suffix: `*.integration.test.ts(x)`
Runner: `npm run test:integration`
CI step: `.github/workflows/frontend.yml` → `Integration tests`

- [ ] **MSW boundary** — Test uses the shared `server` from `__integration__/server.ts` and imports `./setup` to install the lifecycle hooks. No `vi.mock('../api/...')` patterns — that would unit-test the hook, not integration-test the contract.
- [ ] **Happy path** — At least one test exercises the API client → hook → component chain (or API client → assertion) against an MSW handler that returns a contract-shaped response.
- [ ] **Wire contract assertions** — A test inspects the intercepted request (URL, query string, method, body, headers) and asserts each field the backend expects. Catches API-client serialization drift.
- [ ] **Bearer token** — At least one test asserts the `Authorization: Bearer <token>` header is present on requests that require auth.
- [ ] **Error mapping** — For each documented error code, an MSW handler returns that status and the test asserts `ApiError` is thrown with the expected `.status` and `.body`.
- [ ] **Hook state transitions** — If the new code is a hook, tests assert `isLoading` / `error` / data state transitions, not just terminal values.
- [ ] **Cleanup** — Tests do not leak handlers. The shared `setup.ts` calls `server.resetHandlers()` between tests; per-test `server.use(...)` is preferred over `server.listen({...})` overrides.

## Cross-Cutting

- [ ] **Tests fail without the production code** — Comment out the new endpoint / API function and confirm at least one integration test goes RED. (Sanity-check that the test actually exercises the new code.)
- [ ] **No live-network calls** — Tests must not call real Keycloak, real backend, or any external service. Backend integration tests use `testcontainers-go` Postgres. Frontend integration tests use MSW. Auth tokens are stubbed.
- [ ] **CI gates updated** — If the new feature lives under a path filter not currently covered by `backend.yml` or `frontend.yml`, the workflow `paths:` list is updated so the integration tests actually run on the PR.
- [ ] **Documentation updated** — If the new endpoint is in `docs/11-api-design.md`, the documented contract (status codes, request/response shape) matches what the integration test asserts.

## Failure Mode

If any item above is unchecked, the pre-merge gate fails. The PR cannot merge until:
- The missing tests are written, OR
- An ADR is filed explaining why integration tests are not feasible for this surface (extremely rare — typically only for vendor SDK bootstrap or one-time data scripts).

## References

- `CLAUDE.md` — Quality Bar item #3 and "Integration Test Mandate" section
- `backend/internal/integration/harness_test.go` — backend harness reference
- `frontend/src/__integration__/server.ts` — MSW server reference
- `.project-ai/hooks/pre-merge.md` — pre-merge gate that enforces this checklist
- `.project-ai/workflows/feature-delivery.md` — where integration tests live in the feature lifecycle
