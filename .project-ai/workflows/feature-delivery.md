# Feature Delivery Workflow

End-to-end workflow for delivering a feature from story to production-ready code. This is the primary workflow for all Phase 1 sprint work.

---

## Flow Overview

```
STORY ──> PLAN ──> DESIGN ──> IMPLEMENT ──> VERIFY ──> QUALITY VALIDATE ──> DOCUMENT ──> DELIVER
  │         │        │           │             │              │           │
  │         │        │           │             │              │           └─ prepare-sprint-delivery
  │         │        │           │             │              └─ maintain-docs skill
  │         │        │           │             └─ quality gates + reviewer agent
  │         │        │           │             └─ review-code skill + checklists
  │         │        │           └─ playbooks (backend/frontend/offline)
  │         │        └─ design skills (backend/frontend/offline/security)
  │         └─ analyze-requirements skill + feature-spec template
  └─ Read from docs/09-backlog.md or docs/08-roadmap.md
```

---

## Phase 1: PLAN

**Goal**: Understand the story and produce a feature specification.

### Steps

1. **Read the story** from `docs/09-backlog.md` or the current sprint section in `docs/08-roadmap.md`.

2. **Run pre-implement hook** to validate preconditions:
   - Feature exists in `docs/03-requirements-catalog.md`
   - Feature belongs to current phase per `docs/07-mvp-scope.md`
   - No Phase 2 features being built prematurely

3. **Run analyze-requirements skill** to:
   - Identify affected domain entities
   - List API endpoints needed
   - Identify database changes
   - Flag security-sensitive aspects
   - Determine offline requirements

4. **Fill the feature-spec template** with:
   - Story reference
   - Acceptance criteria
   - API endpoints (method, path, request/response)
   - Database changes (tables, columns, migrations)
   - RBAC requirements (which roles can access)
   - Offline behavior (what degrades, what works offline)
   - Security considerations

### Decision Gate

> Does the feature spec cover all acceptance criteria? Are there any open questions?
> If yes to open questions: resolve before proceeding.

---

## Phase 2: DESIGN

**Goal**: Produce technical design for backend, frontend, and supporting concerns.

### Steps (run relevant tracks in parallel)

| Track | Condition | Action |
|-------|-----------|--------|
| **API Contract** | Feature has new API endpoints | Run `design-api-contract` skill |
| **Database** | Feature needs new tables/columns | Run `design-database-schema` skill |
| **Backend** | Feature has API endpoints | Run `design-backend-feature` skill |
| **Frontend** | Feature has UI | Run `design-frontend-feature` skill |
| **Offline** | Feature must work offline | Run `design-offline-support` skill |
| **Security** | Feature is security-sensitive | Run `review-security` skill early (shift-left) |

### Backend Design Output
- Domain struct definition
- Repository interface
- Service method signatures
- Handler request/response types
- Route registration with middleware

### Frontend Design Output
- TypeScript interfaces
- Component tree (page -> components -> hooks)
- Form schema (Zod)
- API client function signatures
- Offline storage schema (if applicable)

### Decision Gate

> Does the design satisfy all acceptance criteria from the spec?
> Are RBAC roles, campus scoping, and audit logging accounted for?

---

## Phase 3: IMPLEMENT

**Goal**: Drive the code into existence test-first. Every layer follows the Red-Green-Refactor cycle from `.project-ai/rules/tdd-enforcement.md`: write the failing test, run it and **see it fail (RED)**, write the minimum code to pass (**GREEN**), confirm the full suite is green, then refactor while staying green. No production code is written before a test that requires it. The RED→GREEN order must be visible in the commit history (test commit `test: <story-id> - <desc> (RED)` before the production commit `feat: <story-id> - <desc> (GREEN)`).

### Steps

1. **Database first** (if needed):
   - Run `pre-migration` hook
   - Follow `add-database-table` playbook
   - Create `.up.sql` and `.down.sql` migration files
   - Run `post-migration` hook to verify
   - Migrations are schema, not behavior — the behavior that reads/writes the new schema is driven by the layer tests below, and any new SQL constraint is proven RED-first by an integration test before the constraint exists (see step 2).

2. **Backend implementation — test-first per layer**:
   - Run `pre-api-change` hook (if modifying endpoints)
   - Follow `implement-backend-endpoint` playbook, but for **each layer** run a full RED→GREEN→REFACTOR cycle before moving to the next:
     ```
     for each layer in [repository, service, handler]:
       write test → run → RED → implement minimum → run → GREEN → refactor (stay green)
     then: route registration (thin wiring; covered by the integration test, not a unit test)
     ```
     - **Repository**: write the `pgxmock` test pinning the SQL contract (columns, WHERE/campus filter, parameter positions) → RED → implement the query → GREEN.
     - **Service**: write the table-driven `testify` test for business rules and validation (mocked repository) → RED → implement the logic → GREEN.
     - **Handler**: write the test for request parsing, validation, status codes, and error mapping (mocked service) → RED → implement the handler → GREEN.
   - **Integration test written before (or together with) the endpoint going live**: add the test in `backend/internal/integration/` (build tag `integration`) that drives the real chi router → service → repository → real Postgres via testcontainers-go. Write it so it is RED until the route is wired, then GREEN. Mandatory per `.project-ai/checklists/integration-tests.md` — happy path with DB assertions, campus scoping, every documented error code, every SQL constraint.

3. **Frontend implementation — test-first per unit**:
   - Follow `implement-frontend-page` playbook, running a RED→GREEN→REFACTOR cycle for **each hook and component**:
     ```
     for each unit in [api client, hooks, components, forms]:
       write Vitest/RTL test → run → RED → implement minimum → run → GREEN → refactor (stay green)
     then: page composition + route registration (thin wiring)
     ```
     - **Hooks**: write the Vitest test for loading/error/success state transitions (mocked api-client) → RED → implement the hook → GREEN.
     - **Components / forms**: write the React Testing Library test for interactions and validation behavior → RED → implement → GREEN.
   - **MSW integration test written before the api-client function**: add the test in `frontend/src/__integration__/` (suffix `.integration.test.ts(x)`) that exercises the real `apiClient` + hook against MSW. Write it RED (the api-client function does not exist yet), then make it GREEN by implementing the function. Mandatory per `.project-ai/checklists/integration-tests.md` — happy path with wire-contract assertions, error mapping, Bearer token presence.

4. **Offline support** (if applicable):
   - Follow `add-offline-support` playbook
   - Drive the sync queue and Dexie schema test-first: write the unit tests for enqueue/dequeue/batch/retry and the schema/migration transform → RED → implement in `src/offline/` → GREEN. Keep most of this logic pure so it sits in the unit tier (see `.project-ai/checklists/test-distribution.md`).

### Implementation Order (test-first)

Each box is a RED→GREEN→REFACTOR cycle: the test is written and seen to fail before the code in that box exists.

```
Database Migration (schema only)
       │
       v
Backend, per layer (test → RED → code → GREEN → refactor):
   Repository → Service → Handler → Route
   └─ Integration test (testcontainers) written before the endpoint goes live
       │
       v
Frontend, per unit (test → RED → code → GREEN → refactor):
   API Client → Hooks → Components → Page
   └─ MSW integration test written before the api-client function
       │
       v
Offline Support — sync queue & Dexie schema driven test-first (if applicable)
       │
       v
Post-Implement Hook (tests, lint, quality assessment, docs check, HANDOFF.md update)
```

> Test distribution: aim for ~60% unit / ~30% integration / ~10% E2E across the tests this feature adds (`.project-ai/checklists/test-distribution.md`). Most behavior is proven at the unit tier; integration tests prove the boundaries; the single E2E slice proves the critical journey.

---

## Phase 3.5: POST-IMPLEMENT

**Goal**: Automated verification before entering review pipeline.

### Steps

1. **Run post-implement hook**:
   - Run full test suite.
   - Run all linters.
   - Quick quality gate assessment on changed files.
   - Verify documentation is updated.
   - Check for new dependency additions.
   - Generate implementation summary for reviewer.
   - **Update HANDOFF.md** with completed task, files created/modified, decisions, and current state.

2. All findings must be addressed before proceeding to VERIFY.

---

## Phase 4: VERIFY

**Goal**: Confirm the implementation meets all quality standards.

### Steps

1. **Run automated hooks**:
   - `post-api-change` hook (if endpoints were modified)
   - `post-migration` hook (if migrations were added)

2. **Run review-code skill** to check:
   - Code structure follows project conventions
   - Error handling is complete
   - Context propagation is correct
   - Logging follows standards
   - Quality profile compliance
   - Clean code categories
   - Complexity thresholds

3. **Run the appropriate checklist(s)**:
   - `backend-feature-complete.md` for backend changes
   - `frontend-feature-complete.md` for frontend changes
   - `api-review.md` for new/modified endpoints

4. **Security review** (if security-sensitive):
   - Run `review-security` skill
   - Complete `security-review.md` checklist
   - Follow `security-sensitive-change.md` workflow if needed

5. **Run pre-review hook** for final automated gate:
   - `make test` (Go unit, fast)
   - `make test-integration` (Go integration, real Postgres via testcontainers — **mandatory**)
   - `npm test` (React unit)
   - `npm run test:integration` (React integration via MSW — **mandatory**)
   - `make lint` (golangci-lint)
   - ESLint (TypeScript)
   - Quality gate validation
   - **`integration-tests.md` checklist** completed — every new endpoint and every new API client function has a passing integration test

### Verification Matrix

| Change Type | Required Checks |
|-------------|----------------|
| Backend only | backend-feature-complete + api-review + integration-tests + make test + make test-integration + make lint |
| Frontend only | frontend-feature-complete + integration-tests + npm test + npm run test:integration + eslint |
| Full stack | All checklists (including integration-tests) + all unit and integration tests + all lints |
| Security-sensitive | Above + security-review checklist |
| Database change | Above + migration up/down test + integration test exercising new constraint or column |

---

## Phase 4.5: QUALITY VALIDATE

**Goal**: Enforce quality gates before code enters the main branch.

### Steps

1. **Run quality gate evaluation**:
   - Evaluate all changed code against New Code Quality Gate from `docs/quality/quality-gates.md`
   - All conditions must pass (0 bugs, 0 vulnerabilities, coverage ≥ 80%, duplication ≤ 3%, ratings = A)

2. **Run maintainability analysis** (for significant changes):
   - Execute `maintainability-analysis` skill
   - Check complexity, duplication, coupling, cohesion, naming quality
   - Verify clean code categories: Consistency, Intentionality, Adaptability, Responsibility

3. **Run reliability validation** (if error handling or state transitions involved):
   - Execute `reliability-validation` skill
   - Verify error recovery, state consistency, fault tolerance

4. **Invoke reviewer agent**:
   - Reviewer evaluates PR against quality gate
   - Issues APPROVE or REQUEST_CHANGES verdict
   - All BLOCKER and MAJOR issues must be resolved

5. **Run pre-merge hook**:
   - Final quality gate enforcement — blocking
   - All conditions must pass for merge

### Quality Gate Failure Recovery

If quality gate fails:
- Follow `refactor-for-quality` playbook
- Run `refactoring` checklist after fixes
- Re-evaluate quality gate
- Repeat until all conditions pass

### Single Autonomous Gate: `make deliver`

Steps 1–5 above are not run ad hoc at delivery time — they are chained, fail-fast,
into the **single autonomous gate** `make deliver` (root `Makefile`). That command
runs backlog validation, the TDD commit-order gate, the backend and frontend
gates, the E2E smoke flow, the **blocking critical-review gate** (which **produces**
`tasks/review-<branch>.md` via the `autonomous-critical-review` skill — the reviewer
verdict is now a file, not just a consulted opinion), and the DoD gate. On a
`REQUEST_CHANGES` verdict, the auto-remediation loop in
`.project-ai/workflows/autonomous-delivery.md` runs `refactor-for-quality` under TDD
and re-runs `make deliver` autonomously, up to 3 cycles. Do not re-implement these
gates here — invoke `make deliver` and follow `autonomous-delivery.md`.

---

## Phase 5: DOCUMENT

**Goal**: Keep documentation in sync with implementation.

### Steps

1. **Run maintain-docs skill** to identify docs that need updating.

2. **Update affected documentation**:
   - New endpoint: update `docs/11-api-design.md`
   - New table/column: update `docs/10-data-model.md`
   - Domain change: update `docs/04-domain-model.md`
   - Offline feature: update `docs/12-offline-sync-strategy.md`
   - Security change: update relevant security docs (13, 16, 17, 18, 19)

3. **Commit with traceable message**:
   ```
   feat: <short description of feature>
   ```

---

## Phase 6: DELIVER

**Goal**: Take the feature from "committed" to "ready-for-PR" through the single
autonomous gate, and stop at the permitted boundary — **before `git push`**.

### Steps

1. **Run the single autonomous gate: `make deliver`.**
   This is the one command that delivers. It chains every local gate
   (backlog → TDD order → backend → frontend → E2E smoke → critical review → DoD),
   **produces** the reviewer verdict file `tasks/review-<branch>.md` via the
   `autonomous-critical-review` skill, and — on all green — prints the
   `READY-FOR-PR` banner with the **suggested** push command. Follow
   `.project-ai/workflows/autonomous-delivery.md` for the end-to-end loop.

2. **On `REQUEST_CHANGES`, auto-remediate (no human approval per cycle).**
   The orchestrator applies `refactor-for-quality` under TDD and re-runs
   `make deliver`, autonomously, **up to 3 cycles**. If it does not converge to
   `APPROVE` after 3 cycles, the impasse is recorded in `tasks/review-<branch>.md`
   and the run **stops for human decision** — nothing is pushed.

3. **Push boundary (hard).** This workflow, `make deliver`, and every artifact it
   invokes **NEVER** run `git push`, `gh pr create`, or `gh pr merge` — the PAT has
   no such permission. Delivery is complete at: local commits + `APPROVE` in
   `tasks/review-<branch>.md` + the `READY-FOR-PR` banner. A human runs the
   suggested `! git push -u origin <branch>`. See `CLAUDE.md` —
   "Autonomous Delivery & Push Boundary".

4. **Sprint end (when the increment is releasable).** Run `assess-release-readiness`,
   `prepare-handoff` (update `HANDOFF.md`), and complete the `sprint-release.md`
   checklist. Tag the release locally (`git tag sprint-N-complete`); the tag, like
   the branch, is pushed by a human.

---

## Agent Assignments

| Phase | Primary Agent | Supporting Agent |
|-------|--------------|-----------------|
| PLAN | tech-lead | product-analyst (requirements refinement) |
| DESIGN (API/schema) | backend-engineer | tech-lead |
| DESIGN (backend) | backend-engineer | tech-lead |
| DESIGN (frontend) | frontend-engineer | tech-lead |
| DESIGN (security) | security-engineer | tech-lead |
| IMPLEMENT (backend) | backend-engineer | — |
| IMPLEMENT (frontend) | frontend-engineer | — |
| IMPLEMENT (infra) | devops-engineer | backend-engineer, frontend-engineer |
| POST-IMPLEMENT | — (automated) | — |
| VERIFY | tech-lead | security-engineer (if needed), qa-engineer (test validation) |
| QUALITY VALIDATE | reviewer | tech-lead, qa-engineer, security-engineer (if needed) |
| DOCUMENT | tech-lead | — |
| DELIVER | tech-lead | devops-engineer (deployment) |

---

## Quick Reference: Skills, Hooks, Playbooks

| Artifact | Used In |
|----------|---------|
| **Skills**: analyze-requirements, design-backend-feature, design-frontend-feature, design-offline-support, review-security, review-code, review-api-contract, maintain-docs, assess-release-readiness, prepare-handoff | Throughout |
| **Hooks**: pre-implement, pre-api-change, post-api-change, pre-migration, post-migration, pre-review, pre-release | PLAN, IMPLEMENT, VERIFY, DELIVER |
| **Playbooks**: add-database-table, implement-backend-endpoint, implement-frontend-page, add-offline-support, prepare-sprint-delivery, conduct-security-review | IMPLEMENT, VERIFY, DELIVER |
| **Checklists**: backend-feature-complete, frontend-feature-complete, api-review, security-review, sprint-release | VERIFY, DELIVER |
| **Templates**: feature-spec, adr | PLAN, architecture decisions |
