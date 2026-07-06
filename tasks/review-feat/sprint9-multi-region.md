# Critical Review — Sprint 9 (Multi-Region and Data Segregation)

**Branch:** `feat/sprint9-multi-region` (8 commits) · **Base:** `main`
**Scope reviewed:** `git diff main...HEAD` — RLS (9.1), intl document types (9.2), multi-currency (9.3), campus timezone (9.4), plus review remediation (commits `ad9b18b`, `24eb719`, `b678aea`)
**Reviewer:** Independent critical reviewer (blocking gate)
**Status:** Re-reviewed after remediation. Original Blocker + 2 Majors verified resolved against the diff.

---

## Quality Gate

| Gate | Result | Notes |
|------|--------|-------|
| New-code bugs = 0 | **PASS** | BLOCKER-1 (pre-campus routes broken under RLS) resolved by the `BypassRLS` owner-pool path; proven by a production-faithful integration test. |
| Security rating A | PASS | RLS design sound and fail-closed; no secrets committed; `set_config` parameterized; startup guard warns if the app role bypasses RLS. Excluded tables now enumerated with rationale. |
| Reliability rating A | PASS | `/self-register` and `/auth/me` proven functional as the non-owner `chesed_app` role. |
| Test coverage new code | PASS | Unit (`bypass_rls_test`, `config_test`), the isolated RLS policy suite, AND a production-faithful pre-campus regression test (`TestRLS_SelfRegisterAndOnboardingBypassRLS`) that wires app repos on the `chesed_app` pool + `BypassRLS(owner)`. |
| Duplication ≤ 3% | PASS | |
| Migrations up/down symmetric | PASS | 27/28/29/30 symmetric and idempotent where needed. |

**Gate verdict: PASS.**

---

## Clean Code

- **Intentionality / naming:** Excellent throughout. `BypassRLS`, `warnIfRLSBypassed`, `AdminDatabaseURL`, `buildPreCampusRouter`, `base.q(ctx)`, `QuerierFrom` all reveal intent. Comments are used only where behavior is non-obvious (GUC lifecycle, owner-bypass rationale, pre-campus routing, idempotent CHECK swap).
- **Consistency:** The remediation reuses the existing querier-context abstraction — `BypassRLS` installs the owner Querier via the same `NewQuerierContext` used by `CampusTx`, so `r.q(ctx)` transparently resolves to the right connection on both route groups. No parallel/ad-hoc path was invented. Very clean.
- **Adaptability:** `AdminDatabaseURL` defaults to `DATABASE_URL`, so single-role dev/test keeps working; RLS-enabled deployments set the owner URL explicitly. Dependencies point inward.
- **Responsibility:** Each middleware does one thing; `warnIfRLSBypassed` is a self-contained observability aid.

The earlier consistency defect (undocumented `volunteer_agreement` exclusion) is resolved: the docs now enumerate every excluded table and scope the completeness claim to the covered operational tables.

---

## Complexity

All new/changed files within thresholds (Go: func ≤ 40 lines, file ≤ 400, cyclomatic ≤ 10, cognitive ≤ 25):

| File | Lines | Verdict |
|------|-------|---------|
| `middleware/bypass_rls.go` | 24 | OK — single trivial middleware |
| `middleware/campus_tx.go` | 129 | OK |
| `cmd/server/main.go` | +45 | OK — `warnIfRLSBypassed` ~18 lines, cyclomatic ~3 |
| `config/config.go` | +10 | OK |
| `utils/document.go` | 44 | OK |
| `repository/person_repository.go` | 572 | Over the 400 file threshold, but pre-existing (Sprint 9 only swapped `r.pool` → `r.q(ctx)`); not charged against this sprint. |

Frontend files all within TS thresholds.

---

## Issues by Severity

### BLOCKER — none (BLOCKER-1 RESOLVED)

**BLOCKER-1 (RESOLVED) — RLS + non-owner app role broke `/self-register` and `/auth/me`.**
Root cause was: the app connects as the RLS-subject role `chesed_app`, but `/self-register` (person INSERT → rejected by `WITH CHECK` with an unset GUC) and `/auth/me` (global email lookup → fail-closed to zero rows) run outside `CampusTx`, so no campus GUC exists.

**Fix (verified in diff):**
- New `internal/middleware/bypass_rls.go`: `BypassRLS(admin)` installs the owner (RLS-bypassing) Querier into the request context via the same `NewQuerierContext` mechanism, so `r.q(ctx)` resolves to the owner connection on these routes. No transaction/GUC needed — the owner bypasses RLS.
- `registerAuthOnlyRoutes` (`/self-register`, `/auth/me`, `/campuses`) now `r.Use(middleware.BypassRLS(adminPool))`.
- `main.go` opens a second owner pool from new config `AdminDatabaseURL` (`ADMIN_DATABASE_URL`, defaults to `DATABASE_URL` for single-role dev/test). Wired to the **owner** role in all compose files: dev/e2e `chesed:chesed`, prod `${DATABASE_URL}` (owner) while the app uses `${APP_DATABASE_URL}` (`chesed_app`).
- `TestRLS_SelfRegisterAndOnboardingBypassRLS` builds the pre-campus router with `personRepo` on the **non-owner `chesed_app` pool** wrapped in `BypassRLS(ownerPool)`, and asserts `/self-register` → 201 with the row persisted in Postgres, and `/auth/me` → campus resolved via the global lookup. This is the production-faithful coverage MAJOR-2 asked for and directly guards the regression.

**Verification performed:** `go build ./...` (0), `go vet` on middleware+config (0), `gofmt -l` clean, `go test ./internal/middleware/... ./internal/config/...` pass, integration package compiles under the `integration` build tag (new test + helpers resolve). Coordinator reports the full serial integration suite green (incl. 6 RLS scenarios + this regression) and 6/6 e2e smoke on the rebuilt stack with the app booting as `chesed_app`. *Confidence: 9/10.*

### MAJOR — none (both RESOLVED)

**MAJOR-1 (RESOLVED) — `volunteer_agreement` (and other tables) undocumented as RLS-excluded.**
`docs/10-data-model.md` and `docs/18-threat-model.md` now enumerate every excluded table — `audit_log`, `volunteer_agreement`, `person_role`, `app_user`, `service_type`, `campus` — each with a rationale, and explicitly scope the completeness guarantee ("a repository that forgets its `WHERE campus_id` clause cannot leak") to the **covered operational tables**. `volunteer_agreement` access is transitively campus-bounded through an already-resolved RLS-protected person, and it is written on the pre-campus self-register path (owner connection still writes `campus_id` explicitly, so integrity holds). The claim is now accurate. Verified the agreement routes only touch RLS-excluded tables (`volunteer_agreement`, `person_role`) on the raw pool, so no fail-closed hazard on those routes. *Confidence: 8/10.*

**MAJOR-2 (RESOLVED) — production RLS wiring not exercised end-to-end.**
`TestRLS_SelfRegisterAndOnboardingBypassRLS` now exercises the exact production configuration for the pre-campus routes (app repos on `chesed_app` + `BypassRLS(owner)`). The isolated `rls_test.go` policy suite (correctly connecting as the non-owner role) already proved the policies. Together they cover both the policy layer and the wiring. *Confidence: 9/10.*

### MINOR

**MINOR-1 — `clearBaseConfigEnv` (config_test.go) does not clear `ADMIN_DATABASE_URL`.** Benign in practice: `t.Setenv` restores env between subtests and the default-case subtest runs before any that set the override, so the "defaults to DATABASE_URL" assertion is not polluted. Still worth adding `ADMIN_DATABASE_URL` to the cleared set for robustness against future reordering / ambient env. Non-blocking.

**MINOR-2 — `RequireAgreement` runs inside `CampusTx` but its repos use the raw pool** (carried from the original review). Functionally harmless (those tables are RLS-excluded), but the agreement-gate reads execute on a different connection than the request tx. Note-worthy, not blocking.

### SUGGESTION

- **S-1:** `openAppPool` sets the `chesed_app` test password to a literal — fine for tests; the comment already explains it mirrors the dev default.
- **S-3:** Add a test for the `formatDateTime` / `useCampusTimezone` browser-zone fallback path.

---

## Positive Practices

- Remediation reuses the querier-context abstraction rather than inventing a new code path — the owner pool is injected the same way the RLS transaction is, so repositories stay agnostic.
- RLS design remains textbook: non-owner role, `set_config(..., is_local => true)`, `current_setting(..., true)` fail-closed, `WITH CHECK` writes, EXISTS policies for join-inherited children, `audit_log` excluded with rationale.
- `warnIfRLSBypassed` adds startup observability so a future misconfiguration (app reconnecting as owner, silently disabling RLS) is detectable — closes original S-2.
- Production-faithful regression test connects app repos to the **non-owner** pool; the isolated RLS suite connects as the non-owner role — both avoid the RLS false-pass trap.
- `ADMIN_DATABASE_URL` default (falls back to `DATABASE_URL`) is safe for single-role setups and explicitly overridden to the owner in every RLS-enabled environment; config test covers default + override.
- e2e donation smoke now selects USD and asserts persisted `currency = 'USD'` — genuine multi-currency end-to-end proof.
- Migrations 27–30 symmetric; `000027` genuinely idempotent. Docs (10/11/13/18) kept in sync with the implementation.

---

## Verdict

APPROVE
