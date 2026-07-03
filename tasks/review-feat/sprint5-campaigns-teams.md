# Critical Review — feat/sprint5-campaigns-teams

Produced by the autonomous-critical-review skill (independent reviewer agent
over `git diff main...HEAD`). Consumed by `make deliver` step 6
(`deliver-review-gate`). The authoritative verdict is the LAST
`### Verdict:` heading in this file.

---

## Cycle 1 (2026-07-02) — summary

Initial verdict: **REQUEST_CHANGES** (superseded by cycle 2 below).

Findings: **B1** attendance `Transition`/`UpdateNotes` RETURNING/Scan mismatch
(13 columns vs 14 destinations — every transition and notes update 500'd; zero
covering tests at any layer); **B2** frontend Integration Test Mandate violated
for `getCampaign`/`updateCampaign`/`addTeamMember`; **M1** committed docs/11
contradicted the shipped error contract (422/conflict vs 400
`validation_error`/409 `duplicate`) with the fix sitting uncommitted; **M2**
campaign edit silently wiped `coordinator_id` (full-replace PUT, form dropped
the field); **M3** team picker capped at the first 20 persons (server-side
search unwired); minors **N1–N5** (harness role drift, missing stale-response
guard, no client-side date validation, selection cleared on failed add,
cosmetic docs drift).

Remediation applied under TDD as commits `e159612` (RED) → `2024e66` (GREEN) →
`8f62eb2` (docs).

---

## Cycle 2 (2026-07-02) — Remediation Verification

**Branch**: `feat/sprint5-campaigns-teams` (12 commits ahead of `main`, 65 files, +4,940/−77)
**Remediation pair verified**: `e159612` (test/RED) → `2024e66` (fix/GREEN) → `8f62eb2` (docs)

## Quality Gate: PASS

### Conditions
| Condition | Status | Details |
|-----------|--------|---------|
| Bugs | PASS | 0 — cycle-1 B1 (RETURNING/Scan mismatch) fixed and pinned at two test layers |
| Vulnerabilities | PASS | 0 — campus scoping and audit logging verified on all campaign mutations; cross-campus probes return generic 400/404 (no existence disclosure) |
| Security Hotspots | PASS | Reviewed: RBAC role sets, campus boundary tests, error-message genericity — all confirmed in code and integration tests |
| Coverage | PASS | Every new endpoint has real-Postgres integration coverage; the previously uncovered `POST /{id}/transitions` and `PATCH /{id}/notes` blind spot is closed. Full suites: backend unit green, backend integration green (40.8s, Docker, migrations from disk), frontend 161/161 across 30 files |
| Duplication | PASS | ≤ 3% — remediation reused existing harness/MSW helpers; no copy-paste blocks introduced |
| Maintainability | PASS | A — all remediated files within thresholds (see Complexity Report) |
| Reliability | PASS | A — every error path handled; N2 stale-response guard added; N4 removes the failure-path state wipe |
| Security | PASS | A |
| High severity issues | PASS | 0 |

Gates run locally (CI paused): `go build ./...`, `go test ./internal/...`,
`go test -tags integration ./internal/integration/` (Docker present, real
Postgres), `make lint` (clean), `npm run typecheck` (clean), `npm run lint`
(0 errors, 49 warnings = pre-existing baseline, none in branch files),
`npx vitest run` (161 passed).

## Clean Code Assessment
| Category | Status | Findings |
|----------|--------|----------|
| Consistency | PASS | Harness role wiring now mirrors `cmd/server/main.go` verbatim; pgxmock contract tests follow the established regression pattern; MSW tests follow the shared `server.ts` + per-test `.use(...)` convention |
| Intentionality | PASS | Remediation comments encode non-obvious decisions only (M2 echo rationale, N2 guard rationale, harness-fidelity note) |
| Adaptability | PASS | `onSearchTextChange` added as an optional prop — no breaking change to other `SearchableSelect` call sites; `useTeamActions` keeps UI state out of the data hook |
| Responsibility | PASS | `handleAdd` now returns `Promise<boolean>` so the form owns its own reset decision — correct layering |

## Complexity Report
No function in the remediated or branch-new files exceeds thresholds.
Spot-measured: `CampaignFormPage.tsx` 197 lines, `CampaignDetailPage.tsx` 222
lines, `SearchableSelect.tsx` 252 lines (all < 300); `campaign_service.go` 383
lines, `attendance_repository.go` 330 lines (< 400); all functions within
length/nesting/cyclomatic limits. One note: `campaign_service_test.go` is 415
lines — over the 400-line Go file guideline, but it is a test file organized as
table-driven cases (SUGGESTION only).

## Remediation Verification

Each cycle-1 finding was confirmed **in code**, not from commit messages:

**B1 — FIXED, coverage genuine (both layers).**
`backend/internal/repository/attendance_repository.go`: `Transition` and
`UpdateNotes` RETURNING lists now include `campaign_id` in position 4; both are
byte-consistent with the 14 Scan destinations in identical order.
- **pgxmock contract tests** (`attendance_repository_test.go:256–328`) pin the
  regex `RETURNING id, person_id, triage_id, campaign_id, campus_id, …`.
  Re-run **at the RED commit `e159612` in a throwaway worktree**: both fail
  with exactly the original defect. At HEAD they pass. The RED→GREEN pair is
  genuine, not retro-fitted.
- **Real-Postgres integration tests**
  (`backend/internal/integration/attendance_test.go`) exercise
  `POST /api/v1/attendances/{id}/transitions` and
  `PATCH /api/v1/attendances/{id}/notes` through the full
  chi→service→repository stack with DB-level assertions.

**B2 — FIXED.** `campaigns.integration.test.tsx` now covers at the MSW
boundary: `getCampaign` (detail incl. `team`), `updateCampaign` (PUT body
asserted to carry `status` + `coordinator_id`), `addTeamMember` (201 happy path
+ second call → 409 mapped to `ApiError{status: 409}`).

**M1 — FIXED.** `docs/11-api-design.md` campaign section (committed in
`8f62eb2`) documents 400 `validation_error` and 409 `duplicate`; no
`422`/`conflict` remnants (grep-verified). The new error-code table matches the
handler's emitted codes one-for-one.

**M2 — FIXED.** `CampaignFormPage.tsx`: `toFormValues` carries
`coordinator_id` and `toPayload` echoes it, so the full-replace PUT can no
longer wipe it. Test asserts the round-trip; fails at the RED commit, passes at
HEAD.

**M3 — FIXED.** `SearchableSelect` gained the optional `onSearchTextChange`
prop; `AddMemberForm` wires it to `usePersons().search` (debounced server-side
person fetch). Test types into the picker and asserts `search('Mar')`.

**N1 — FIXED.** Harness role sets verified side-by-side against
`cmd/server/main.go` — exact mirror, with a comment pinning the fidelity
requirement. **N2 — FIXED** (`requestSeq` stale-response guard in
`useCampaign`). **N3 — FIXED** (client-side `end_date >= start_date` rule +
test). **N4 — FIXED** (picker selection cleared only on `ok === true`).
**N5 — FIXED** (RFC3339 response examples + campaign error-code table).

**New-issue spot checks (no findings):** the
`item?.scrollIntoView?.({ block: 'nearest' })` optional-call guard is a safe
jsdom accommodation with no browser behavior change; the harness route
additions introduced no drift from production; audit logging present on all
four campaign mutations with non-failing error handling; campus scoping present
in every campaign repository query.

## Issues

**BLOCKER**: none.
**MAJOR**: none.
**MINOR**: none.
**SUGGESTION**:
1. `backend/internal/service/campaign_service_test.go` (415 lines) slightly
   exceeds the 400-line Go file guideline; consider splitting team-assignment
   cases into a sibling `_test.go` file at the next touch. Non-blocking.
2. Frontend lint baseline remains at 49 warnings (pre-existing, e.g.
   `authStore.ts:38` complexity 13); none originate on this branch, but a
   follow-up debt item to burn the baseline down would keep the gate honest.

## TDD Narrative

The branch preserves strict RED→GREEN ordering across all four pairs:
`90fade8`→`6dd3f47` (S07.1+S07.2 API), `470d970`→`369c23d` (S07.3+S07.4
linking/metrics), `be38c8f`→`c006898` (S07.5+S07.4 frontend), and the
remediation pair `e159612`→`2024e66`. The remediation RED commit was
independently proven red: executed at `e159612` in an isolated worktree, both
pgxmock column-contract tests fail with the exact production defect, and the
frontend M2/M3 wiring is demonstrably absent at that commit. The GREEN commit
contains only the minimum code to satisfy the failing tests plus the reviewed
N-fixes, each of which also has a test written in the RED commit. The docs
commit (`8f62eb2`) correctly carries no code.

## Verification Evidence

- `go build ./...` — clean
- `go test ./internal/... -count=1` — all packages pass
- `go test -tags integration ./internal/integration/ -count=1` — **ok, 40.8s**
  (real Postgres via testcontainers, migrations applied from disk; includes the
  new attendance regression, campaign CRUD/team, campaign-link, and
  pre-existing sync suites)
- RED proof: contract tests **fail at `e159612`**, pass at HEAD
- `make lint` (golangci-lint) — clean
- `npm run typecheck` — clean; `npm run lint` — 0 errors / 49 pre-existing
  warnings; `npx vitest run` — 30 files, 161 tests, all pass

### Verdict: APPROVE
