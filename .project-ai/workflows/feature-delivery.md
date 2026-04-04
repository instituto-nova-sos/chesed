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
| **Backend** | Feature has API endpoints | Run `design-backend-feature` skill |
| **Frontend** | Feature has UI | Run `design-frontend-feature` skill |
| **Database** | Feature needs new tables/columns | Follow `add-database-table` playbook design steps |
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

**Goal**: Write the code following the design.

### Steps

1. **Database first** (if needed):
   - Run `pre-migration` hook
   - Follow `add-database-table` playbook
   - Create `.up.sql` and `.down.sql` migration files
   - Run `post-migration` hook to verify

2. **Backend implementation**:
   - Run `pre-api-change` hook (if modifying endpoints)
   - Follow `implement-backend-endpoint` playbook:
     ```
     domain struct -> repository -> service -> handler -> route
     ```
   - Write unit tests alongside service layer
   - Write integration tests for repository

3. **Frontend implementation**:
   - Follow `implement-frontend-page` playbook:
     ```
     types -> api client -> hooks -> components -> page -> route
     ```
   - Write Vitest tests for hooks
   - Write React Testing Library tests for forms

4. **Offline support** (if applicable):
   - Follow `add-offline-support` playbook
   - Implement Dexie.js storage in `src/offline/`
   - Implement sync queue for offline mutations

### Implementation Order

```
Database Migration
       │
       v
Backend: Domain -> Repository -> Service -> Handler -> Route
       │
       v
Frontend: Types -> API Client -> Hooks -> Components -> Page
       │
       v
Offline Support (if applicable)
```

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
   - `make test` (Go)
   - `npm test` (React)
   - `make lint` (golangci-lint)
   - ESLint (TypeScript)
   - Quality gate validation

### Verification Matrix

| Change Type | Required Checks |
|-------------|----------------|
| Backend only | backend-feature-complete + api-review + make test + make lint |
| Frontend only | frontend-feature-complete + npm test + eslint |
| Full stack | All checklists + all tests + all lints |
| Security-sensitive | Above + security-review checklist |
| Database change | Above + migration up/down test |

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

## Phase 6: DELIVER (Sprint End)

**Goal**: Prepare the sprint for release.

### Steps

1. **Run prepare-sprint-delivery playbook**:
   - Verify all sprint stories complete
   - Run full test suite
   - Run full lint suite
   - Check documentation sync

2. **Run assess-release-readiness skill**:
   - Evaluate overall sprint quality
   - Identify any gaps or risks
   - Confirm all checklist items pass

3. **Run prepare-handoff skill**:
   - Generate HANDOFF.md content
   - Summarize what was built, decisions made, known issues
   - Recommend next steps

4. **Complete sprint-release checklist** (`sprint-release.md`)

5. **Tag the release**:
   ```
   git tag sprint-N-complete
   ```

---

## Agent Assignments

| Phase | Primary Agent | Supporting Agent |
|-------|--------------|-----------------|
| PLAN | tech-lead | — |
| DESIGN (backend) | backend-engineer | tech-lead |
| DESIGN (frontend) | frontend-engineer | tech-lead |
| DESIGN (security) | security-engineer | tech-lead |
| IMPLEMENT (backend) | backend-engineer | — |
| IMPLEMENT (frontend) | frontend-engineer | — |
| VERIFY | tech-lead | security-engineer (if needed) |
| QUALITY VALIDATE | reviewer | tech-lead, security-engineer (if needed) |
| DOCUMENT | tech-lead | — |
| DELIVER | tech-lead | — |

---

## Quick Reference: Skills, Hooks, Playbooks

| Artifact | Used In |
|----------|---------|
| **Skills**: analyze-requirements, design-backend-feature, design-frontend-feature, design-offline-support, review-security, review-code, review-api-contract, maintain-docs, assess-release-readiness, prepare-handoff | Throughout |
| **Hooks**: pre-implement, pre-api-change, post-api-change, pre-migration, post-migration, pre-review, pre-release | PLAN, IMPLEMENT, VERIFY, DELIVER |
| **Playbooks**: add-database-table, implement-backend-endpoint, implement-frontend-page, add-offline-support, prepare-sprint-delivery, conduct-security-review | IMPLEMENT, VERIFY, DELIVER |
| **Checklists**: backend-feature-complete, frontend-feature-complete, api-review, security-review, sprint-release | VERIFY, DELIVER |
| **Templates**: feature-spec, adr | PLAN, architecture decisions |
