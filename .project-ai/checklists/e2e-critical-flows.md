# Checklist: E2E Critical Flows (Sprint Release Gate)

## Purpose

The end-to-end suite (`frontend/e2e/`, Playwright → built frontend → REAL Go API →
REAL Postgres, Keycloak mocked) is the only tier that proves a whole user journey
holds together across the wire. This checklist is the **sprint release gate** for
that tier: before a sprint is tagged, every Phase 1 critical flow must have a
green FULL E2E run, and the compensating real-auth integration test must be green.

This checklist is **blocking** at the `pre-release` gate. It is the top-tier
counterpart to `checklists/integration-tests.md` (per-feature, pre-merge).

## Trigger

Run this at sprint end, as part of `checklists/sprint-release.md`, and whenever a
critical flow's screens, the API contract behind it, or the auth/sync wiring
changed during the sprint.

## Pre-conditions (must hold before the gate counts)

- [ ] **Stack is the real thing** — `docker compose -f docker-compose.e2e.yml up -d --build`
      brings up Postgres + the Go API (migrations applied from disk) + the mock
      OIDC issuer. No app code is stubbed; only Keycloak is mocked
      (`frontend/e2e/mock-oidc`).
- [ ] **Frontend is built, not dev-served** — Playwright drives the production
      build via `vite preview` on port 5173 (the API's CORS allow-list origin).
- [ ] **Determinism** — each test uses per-test campus isolation + cleanup
      (`frontend/e2e/fixtures.ts`): a unique Keycloak subject, a seeded `app_user`
      on the deterministic test campus, and teardown that deletes the test's data
      by unique prefix. No test depends on another's residue.

## The Gate (blocking)

- [ ] **FULL E2E green** — `npm run test:e2e` passes with zero failures against the
      real stack. (SMOKE alone does NOT satisfy this gate — see
      `rules/e2e-test-tiers.md`.)
- [ ] **Compensating real-auth test green** — `backend/internal/integration/auth_middleware_test.go`
      passes (`make test-integration`). Because E2E mocks Keycloak, this test is
      the ONLY proof the real OIDC token-validation path works: valid → pass,
      expired → 401, missing `email_verified` → 403, bad signature → 401. A green
      FULL E2E without this test green does not pass the gate.
- [ ] **No silently-skipped critical flow** — any `test.fixme`/`test.skip` on a
      critical-flow assertion is listed below with its activating condition, and
      the gate owner has confirmed the gap is a not-yet-built feature, not a
      regression being hidden.
- [ ] **No weakened assertions** — no critical-flow test asserts something trivially
      true to go green. Each flow asserts the screen state AND the DB/API state.

## Phase 1 Critical Flows

Each flow below needs a green FULL E2E (or an explicit, justified `test.fixme`
with its activating feature named). These are the journeys whose breakage would
make the platform unusable in the field.

1. **Authenticated entry** — token injected (Keycloak mocked) → app boots
   authenticated → lands in the app shell. *(Covered by `sync-smoke.spec.ts`
   `@smoke`; auth validity itself covered by the compensating backend test.)*
2. **Person create → list → persist (online)** — login → create a person online →
   person visible in the list → row exists in Postgres on the scoped campus.
   *(Covered by `sync-smoke.spec.ts` `@smoke`.)*
3. **Person offline-create → cache-visible → reconnect → queue drains** — create a
   person while offline → it stays visible from the IndexedDB cache → reconnect →
   the sync queue drains to `/api/v1/sync/push` → row reaches Postgres.
   *(Written in `sync-smoke.spec.ts` as `test.fixme`; **activates when the
   frontend sync drainer ships** — see `tasks/todo.md`. Until then this flow is a
   known, documented gap, NOT a passing test.)*
4. **Triage create for a person** — open a person → create a triage with requested
   services → triage persisted and visible. *(Add a `triage` E2E when the triage
   create journey is promoted to a critical flow; backend contract already covered
   by integration tests.)*
5. **Attendance lifecycle** — create an attendance from a triage → transition its
   state → audit trail + state persisted. *(E2E added when promoted; backend
   covered by integration tests.)*
6. **Reports render** — a COORDINATOR/ADMIN opens reports → the attendance summary
   renders from the live API. *(Report internals stay in integration/unit per
   `checklists/test-distribution.md`; the E2E only asserts the render path.)*

> Scope discipline: flows 4–6 are listed so the gate is complete as those journeys
> mature. Adding their E2E is gated by `rules/e2e-test-tiers.md` (a journey, not a
> single endpoint) and `checklists/test-distribution.md` (keep the top thin). Do
> NOT add an E2E that merely re-tests one endpoint already covered by integration.

## On Failure

- A red FULL E2E **blocks the release**. Triage: is it a real regression, a flaky
  selector/timing issue, or an environment problem (stack not up)? Fix the cause;
  do not retry-until-green or delete the failing assertion.
- If a critical flow cannot be made green because its feature is incomplete, the
  sprint either descopes that flow (documented in HANDOFF.md) or does not release.
  Marking a real regression as `test.fixme` to pass the gate is prohibited.

## How to Use

Run at sprint release per `checklists/sprint-release.md` and the `pre-release`
hook. See `docs/e2e-testing.md` for the exact local run sequence. CI is paused
(CLAUDE.md), so this gate is operator-enforced locally until CI is restored.

## References

- `.project-ai/rules/e2e-test-tiers.md` — SMOKE vs FULL tier definitions + scripts
- `.project-ai/checklists/sprint-release.md` — the umbrella sprint gate (names this list)
- `.project-ai/checklists/test-distribution.md` — keep E2E the thin top of the pyramid
- `.project-ai/checklists/integration-tests.md` — the per-feature (pre-merge) mandate
- `docs/e2e-testing.md` — local run instructions
- `frontend/e2e/` — Playwright config, fixtures, specs
- `docker-compose.e2e.yml` — real API + Postgres + mock OIDC issuer
- `backend/internal/integration/auth_middleware_test.go` — compensating real-auth test
