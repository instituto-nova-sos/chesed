# Playbook: Prepare Sprint Delivery

## Purpose

End-to-end checklist for validating a sprint is complete, tested, secure, and ready for delivery. Covers story verification, testing, security review, documentation, and tagging.

---

## Context

Chesed follows the roadmap in `docs/08-roadmap.md`:
- **Phase 1 sprints**: Sprint 1 (Auth & Infrastructure), Sprint 2 (Person Management), Sprint 3 (Triage & Attendance), Sprint 4 (Reports & Polish)
- **Sprint duration**: 2 weeks
- **Backlog reference**: `docs/09-backlog.md` (stories with acceptance criteria)

---

## Steps

### Step 1: Story Verification

Open `docs/09-backlog.md` and identify all stories assigned to this sprint. For each story:

1. Read the acceptance criteria
2. Verify each criterion is implemented (code exists, endpoint works, UI matches)
3. If a criterion is not met, document what is missing

Sprint-to-story mapping (from `docs/08-roadmap.md`):

| Sprint | Stories |
|---|---|
| Sprint 1 (Auth & Infrastructure) | S01.1-S01.7, S02.1-S02.5 |
| Sprint 2 (Person Management) | S03.1-S03.7 |
| Sprint 3 (Triage & Attendance) | S04.1-S04.7, S05.1-S05.3 |
| Sprint 4 (Reports & Polish) | S06.1-S06.3 |

For each story, check:
- [ ] Backend endpoint implemented and tested
- [ ] Frontend page/component implemented
- [ ] Offline support added (if applicable per `docs/07-mvp-scope.md`)
- [ ] RBAC enforced per `docs/16-iam-and-access-control.md`
- [ ] Audit logging present for data mutations

### Step 2: Run Full Test Suite

```bash
# Backend tests
cd backend && go test ./... -v -count=1

# Frontend tests
cd frontend && npx vitest run

# Check test coverage (informational)
cd backend && go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out
cd frontend && npx vitest run --coverage
```

Coverage targets:
- Service layer: 80% minimum
- Handler layer: 60% minimum
- Repository layer: integration tests covering CRUD + campus isolation
- Frontend hooks: 70% minimum
- Frontend forms: validation + submission tests

Fix all test failures before proceeding. Do not skip or mark tests as `t.Skip()` for delivery.

### Step 3: Run Linters

```bash
# Go linter
cd backend && golangci-lint run ./...

# Go format check
cd backend && gofmt -l .
# Should output nothing (all files formatted)

# TypeScript type check
cd frontend && npx tsc --noEmit

# ESLint
cd frontend && npx eslint src/ --max-warnings 0
```

Fix all lint warnings. The quality bar requires zero warnings for delivery.

### Step 4: Security Review

Run the security review playbook (`playbooks/conduct-security-review.md`) for any security-sensitive changes in this sprint.

Security-sensitive changes include:
- New API endpoints (all of them — every endpoint needs RBAC)
- Changes to auth/RBAC middleware
- Changes to Keycloak configuration
- New database tables or columns containing PII
- Sync endpoint changes
- Report/export endpoint changes
- Offline storage changes

Output: A completed `templates/security-review-report.md` for the sprint.

Minimum checks even for non-security sprints:
- [ ] No new endpoints without RBAC middleware
- [ ] No PII in log statements
- [ ] Campus scoping on all new queries
- [ ] Audit logging on all new mutations

### Step 5: Documentation Verification

Verify all relevant docs are up to date:

| Doc | When to update |
|---|---|
| `docs/11-api-design.md` | Any new or changed endpoint |
| `docs/10-data-model.md` | Any new table, column, or constraint change |
| `docs/04-domain-model.md` | Any new entity or relationship |
| `docs/16-iam-and-access-control.md` | Any RBAC or Keycloak change |
| `docs/12-offline-sync-strategy.md` | Any offline/sync behavior change |
| `docs/08-roadmap.md` | Mark completed tasks as Done |
| `keycloak/realm-export.json` | Any Keycloak realm configuration change |

Verification:
```bash
# Check if API docs match implemented endpoints
# For each route in the router, verify it exists in docs/11-api-design.md
grep -rn "r\.\(Get\|Post\|Put\|Patch\|Delete\)" backend/cmd/server/ backend/internal/handler/
```

### Step 6: Offline Scenario Verification

For features marked as offline-capable in `docs/07-mvp-scope.md`:

| Feature | Offline test |
|---|---|
| Person registration | Create person offline, reconnect, verify sync |
| Triage creation | Create triage offline, reconnect, verify sync |
| Attendance recording | Create attendance offline, reconnect, verify sync |
| Person search | View cached list offline |
| Service type catalog | View cached service types offline |

Test procedure for each:
1. Log in and load data (populate local cache)
2. Go offline (DevTools > Network > Offline)
3. Perform the operation
4. Verify local storage (IndexedDB has the record)
5. Verify sync status shows "Pendente"
6. Go online
7. Verify sync completes and status changes to "Sincronizado"
8. Verify server has the record (check via API or database)

### Step 7: Mobile Responsiveness Check

Test key pages at 320px width (minimum supported mobile width per RNF-11, RNF-13):

Pages to test:
- [ ] Person list page
- [ ] Person create/edit form
- [ ] Person detail page
- [ ] Triage create form
- [ ] Attendance list page
- [ ] Attendance detail page
- [ ] Report page (if applicable in this sprint)

For each page verify:
- No horizontal scrolling
- All text is readable (minimum 14px body text)
- Buttons are tappable (minimum 44x44px touch target)
- Forms are usable (inputs are full-width, labels visible)
- Navigation is accessible (hamburger menu or bottom nav)

Tool: Browser DevTools > Toggle device toolbar > Set width to 320px.

### Step 8: Sprint Release Checklist

Final pre-delivery checklist:

**Code quality:**
- [ ] All tests pass (backend + frontend)
- [ ] Zero lint warnings (Go + TypeScript)
- [ ] No `TODO` or `FIXME` comments for sprint deliverables (search: `grep -rn "TODO\|FIXME" backend/ frontend/src/`)
- [ ] No `console.log` statements in frontend production code
- [ ] No `fmt.Println` in backend production code (use `slog` instead)

**Security:**
- [ ] Security review report completed for this sprint
- [ ] No secrets in source code
- [ ] All endpoints have RBAC middleware
- [ ] Campus scoping on all queries
- [ ] Audit logging on all mutations

**Documentation:**
- [ ] API docs match implementation
- [ ] Data model docs match migrations
- [ ] Roadmap updated with completed tasks

**Functionality:**
- [ ] All sprint stories meet acceptance criteria
- [ ] Offline features tested (disconnect/reconnect cycle)
- [ ] Mobile responsiveness verified at 320px

### Step 9: Prepare Handoff Summary

Write a brief summary of what was delivered in this sprint. Include:

1. **Stories completed**: List of S0x.x story IDs with one-line descriptions
2. **Endpoints added/changed**: List of HTTP method + path
3. **Database migrations**: List of migration files
4. **Known issues**: Any accepted trade-offs or deferred items
5. **Configuration changes**: Any new environment variables, Keycloak changes, or Docker changes

### Step 10: Create Git Tag

```bash
# Format: v<phase>.<sprint>.<patch>
# Example: v1.1.0 for Phase 1, Sprint 1, initial release

git tag -a v1.1.0 -m "Phase 1 Sprint 1: Auth and Infrastructure

Stories completed:
- S01.1: Go project skeleton
- S01.2: React project skeleton
- S01.3: Docker Compose environment
- S01.4: Database migration framework
- S01.5: CI/CD pipeline
- S01.6: Service type seed data
- S02.1: OIDC token validation middleware
- S02.2: RBAC middleware
- S02.3: Local user auto-provisioning
- S02.4: Audit logging middleware
- S02.5: Campus-scoped data access
"

git push origin v1.1.0
```

Tag naming convention:
- `v1.1.0` = Phase 1, Sprint 1, release 0
- `v1.2.0` = Phase 1, Sprint 2, release 0
- `v1.2.1` = Phase 1, Sprint 2, hotfix 1

---

## Checklist

- [ ] All sprint stories verified against acceptance criteria
- [ ] Full test suite passes (backend + frontend)
- [ ] Zero lint warnings
- [ ] Security review completed
- [ ] All docs up to date
- [ ] Offline scenarios tested
- [ ] Mobile responsiveness verified at 320px
- [ ] Release checklist completed
- [ ] Handoff summary written
- [ ] Git tag created
