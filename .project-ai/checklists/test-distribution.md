# Checklist: Test Distribution (The Pyramid)

## Purpose

Keep the test suite shaped like a **pyramid**, not an ice-cream cone. Fast, numerous unit tests at the base; a focused band of integration tests in the middle; a thin slice of slow, full-stack E2E tests at the top. This keeps the suite fast, the failures diagnosable, and the cost of change low. An inverted distribution (many E2E, few unit) makes every change slow to verify and every failure hard to localize.

This checklist complements `rules/tdd-enforcement.md`: TDD says *write the test first*; this checklist says *write it at the right level*.

## Target Ratio

| Tier | Target | What it covers | Boundary exercised |
|------|--------|----------------|--------------------|
| **Unit** | ~60% | Pure logic, branching, validation, state reducers, mappers | None — in-process, mocked dependencies |
| **Integration** | ~30% | Contracts across a process/network boundary | HTTP / SQL (testcontainers) / `fetch` (MSW) |
| **E2E** | ~10% | Critical user journeys, screen → backend → DB | Real browser → real API → real Postgres |

These are targets for the **mix of new tests a feature adds**, not a hard per-PR quota. The shape matters more than the exact percentages.

## Ratio Check (blocking)

- [ ] **Unit share ≥ 50%** of the feature's new tests. If unit tests are below half, the feature is pushing logic that *should* be pure into integration/E2E tiers. **Required action**: extract the branching/validation/transformation logic into pure functions and unit-test those directly, then keep only the contract assertions at the integration tier.
- [ ] **Integration tier present** — every new process/network boundary (endpoint, api-client function, hook that fetches, SQL constraint) has an integration test per `.project-ai/checklists/integration-tests.md`. Integration is the *floor* for boundaries, not a substitute for unit coverage of the logic behind them.
- [ ] **E2E tier is thin** — only the documented critical flow(s) get an E2E test. New E2E specs require justification: they must cover a *journey* (multi-screen, multi-system), not a single function or a single endpoint. Per-function or per-endpoint behavior belongs in unit/integration.
- [ ] **No duplicated assertion across tiers** — a rule already proven by a unit test is not re-proven by an E2E test. Each tier asserts what only that tier can.

## Worked Example — Sync-Drainer Feature

The upcoming offline sync-drainer (Dexie v2 → `useOnlineSync` → pull-merge → conflict UI) decomposes as follows. This is the reference distribution for the pilot slice.

### Unit (~60% — the base)

Pure logic, no boundary, fast:

- [ ] Sync-queue operations: `enqueue` / `dequeue` / ordering / `batch` grouping / `retry` backoff counter — pure functions over plain objects.
- [ ] Conflict-resolution decision function: given local record + server record + versions, returns `keep-local` / `keep-server` / `needs-user` — a pure decision table, every branch covered.
- [ ] Pull-merge reducer: given a server page and local cache, computes the merged set and the conflict list — pure, deterministic.
- [ ] `useOnlineSync` state transitions tested with mocked queue + mocked api-client (React Testing Library): `idle → draining → done`, `idle → draining → error → retry`.
- [ ] Dexie v2 schema/migration mapping logic (version upgrade transform) tested as a pure transform over fixtures.

### Integration (~30% — the middle)

Real boundary, no full stack:

- [ ] **Frontend (MSW)** — `syncPush` / `syncPull` api-client functions: request URL, query string, body shape, `Authorization: Bearer` header, and error-code → `ApiError` mapping. Per `integration-tests.md` (frontend section).
- [ ] **Frontend (MSW)** — `useOnlineSync` against MSW: a queued mutation drains and the handler receives the expected wire payload; a 409 from the server surfaces as a conflict.
- [ ] **Backend (testcontainers)** — the sync push/pull endpoints against real Postgres: happy path with DB assertions, campus scoping boundary, idempotency (same idempotency key → exactly one row), and the SQL uniqueness/constraint the sync relies on. Per `integration-tests.md` (backend section).

### E2E (~10% — the single thin slice)

One Playwright spec, full stack (screen → real API → real Postgres, Keycloak token injected), exactly the critical journey:

- [ ] **The one E2E**: login → create a person while **online** → toggle **offline** → confirm the person stays visible from cache → toggle **online** → the queue drains → assert the row is persisted in Postgres and the UI reflects synced state.

Explicitly **out of E2E** (covered by the tiers above, not re-tested end-to-end): pull-merge internals, conflict-resolution branches, batching, retry/backoff, reports. Driving those through the browser would invert the pyramid.

## How to Use

Run this checklist when scoping a feature's tests (during DESIGN / before writing the first RED test) and again at pre-review. If the ratio check fails, refactor logic out of the upper tiers *before* marking the feature complete — do not "fix" the ratio by deleting integration or E2E tests that cover real boundaries.

## References

- `.project-ai/rules/tdd-enforcement.md` — RED→GREEN→REFACTOR per layer
- `.project-ai/checklists/integration-tests.md` — the integration (middle-tier) mandate
- `.project-ai/checklists/e2e-critical-flows.md` — the E2E (top-tier) sprint gate (created separately)
- `.project-ai/rules/test-coverage-enforcement.md` — per-layer coverage thresholds
- `docs/quality/quality-gates.md` — overall coverage and quality conditions
- `CLAUDE.md` — "Integration Test Mandate"
