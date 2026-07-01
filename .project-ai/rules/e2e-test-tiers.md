# Rule: E2E Test Tiers (Smoke vs Full)

## Purpose

The Playwright end-to-end suite (`frontend/e2e/`) drives the BUILT frontend
against the REAL Go API + REAL Postgres (Keycloak mocked, token injected). It is
the top of the test pyramid (`checklists/test-distribution.md`): slow, expensive,
and the only tier that proves screen → network → API → DB holds together.

Because E2E is slow, it runs at **two tiers** with different cadence and scope.
This rule defines what each tier contains, when it runs, and the npm script it
maps to. It exists so "run the E2E tests" is unambiguous — a fast guard on every
merge, a thorough gate at sprint release.

## The Two Tiers

| Tier | Script | Cadence | Scope | Blocking |
|------|--------|---------|-------|----------|
| **SMOKE** | `npm run test:e2e:smoke` | Every merge to `main` (run locally — CI paused) | The single happy-path critical slice. Fast. | Yes — a red smoke blocks the merge. |
| **FULL** | `npm run test:e2e` | Sprint release gate | Every critical flow incl. offline, conflict, reports journeys. | Yes — see `checklists/e2e-critical-flows.md`. |

`test:e2e:smoke` is `playwright test --grep @smoke`; `test:e2e` runs the whole
`frontend/e2e/` suite. Both require the E2E stack up
(`docker compose -f docker-compose.e2e.yml up -d --build`) and a built frontend —
see `docs/e2e-testing.md`.

### SMOKE — fast, every merge, happy path

- **Goal**: catch gross breakage (app won't boot, auth wiring broken, create/list
  path dead, API contract drift the integration tier missed) in the shortest
  possible run.
- **Content**: the narrow critical slice only — login (token injected) → create
  one person online → person visible in list → row exists in Postgres. Tagged
  `@smoke` in the spec title.
- **Constraints**: no more than a couple of specs; must finish in a few minutes;
  no conflict/pull/report journeys; deterministic (per-test campus isolation +
  cleanup, `frontend/e2e/fixtures.ts`).
- **What it is NOT**: a place to grow. New per-feature behavior goes to
  unit/integration tiers, not smoke. Smoke stays thin on purpose.

### FULL — sprint gate, all critical flows

- **Goal**: prove every Phase 1 critical user journey works end-to-end before a
  release is tagged.
- **Content**: the smoke slice PLUS the remaining critical-flow journeys —
  offline create → cache-visible → reconnect → queue drains; conflict surfaced
  to the user on reconnect; report generation render path. Enumerated in
  `checklists/e2e-critical-flows.md`.
- **Constraints**: still a thin band relative to unit/integration
  (`checklists/test-distribution.md`). Each E2E must cover a *journey*
  (multi-screen / multi-system), never a single function or endpoint.

## Tagging Convention

- A spec belongs to SMOKE iff its title contains `@smoke`. The smoke script
  greps that tag.
- Everything in `frontend/e2e/**/*.spec.ts` belongs to FULL.
- Specs (or assertions) for features that are not yet wired into the running app
  MUST use `test.fixme()` / `test.skip()` with an inline reason — never a
  weakened assertion that passes falsely. (Current example: the offline-drainer
  assertions in `sync-smoke.spec.ts` are `test.fixme` until the drainer ships.)

## Trigger Condition

- **SMOKE**: before every merge to `main` (operator-run while CI is paused).
- **FULL**: at sprint release, as part of `checklists/sprint-release.md` /
  `checklists/e2e-critical-flows.md` and the `pre-release` gate.
- Any change under `frontend/e2e/`, the API contract, or the auth/sync wiring.

## Enforcement Mechanism

- **`package.json` scripts** — `test:e2e:smoke` and `test:e2e` are the only
  sanctioned entry points; do not invent ad-hoc invocations.
- **`checklists/e2e-critical-flows.md`** — the blocking sprint gate that requires
  a green FULL run plus the green `auth_middleware_test.go` compensating test.
- **`pre-merge` (operator-enforced)** — smoke must be green before merge.
- **`pre-release`** — full must be green before tagging.

## Relationship to Other Tiers (do not duplicate)

E2E does NOT replace the per-feature mandates:
- **Unit + integration remain mandatory per feature** (`checklists/integration-tests.md`,
  `rules/tdd-enforcement.md`). E2E asserts the journey; the tiers below assert the
  units and the contracts.
- **Keycloak is mocked in E2E.** The real OIDC token-validation path (expired /
  unverified-email-403 / bad-signature-401) is covered by the compensating
  backend integration test `backend/internal/integration/auth_middleware_test.go`.
  A green FULL E2E without that test green does NOT satisfy the gate.

## Consequences of Skipping

- No smoke on merge → a broken auth wire or dead create/list path lands on `main`
  and is discovered only at sprint end, expensive to bisect.
- No full at release → an offline/conflict/report journey ships broken because no
  lower tier exercises the whole screen → DB path.
- Letting smoke grow → the pyramid inverts into an ice-cream cone; every merge
  gets slow and flaky (`checklists/test-distribution.md`).

## References

- `.project-ai/checklists/e2e-critical-flows.md` — the FULL sprint gate + flow list
- `.project-ai/checklists/test-distribution.md` — pyramid targets; E2E is the thin top
- `.project-ai/checklists/integration-tests.md` — the per-feature integration mandate
- `.project-ai/rules/tdd-enforcement.md` — RED→GREEN; e2e specs count as test files
- `docs/e2e-testing.md` — how to run the E2E stack locally
- `frontend/e2e/` — Playwright config, fixtures, smoke spec
- `docker-compose.e2e.yml` — the real API + Postgres + mock OIDC stack
- `backend/internal/integration/auth_middleware_test.go` — compensating real-auth test
