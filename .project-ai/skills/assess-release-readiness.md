# Skill: Assess Release Readiness

## Purpose

Evaluate whether a sprint is ready for release by verifying all acceptance criteria are met, tests pass, documentation is current, security review is complete, offline behavior is tested, and mobile responsiveness is verified. Produces a go/no-go decision with blockers.

## When to Use / Trigger

- At the end of a sprint (Sprint 1-4).
- When a user says "are we ready to release?" or "sprint readiness check".
- Before deploying to staging or production.

## Role / Expertise

Release manager verifying quality across all dimensions: functionality, testing, security, documentation, accessibility, and offline support.

## Inputs

| Input | Required | Source |
|-------|----------|--------|
| Sprint number and scope | Yes | `docs/08-roadmap.md` |
| Story list for the sprint | Yes | `docs/09-backlog.md` |
| Test results | Yes | CI/CD output or local test run |
| Git log for the sprint | Yes | `git log` |

## Process

### Step 1: Verify Sprint Scope Completion

Read the sprint task list from `docs/08-roadmap.md`:

**Sprint 1 (Auth & Infrastructure):**
- [ ] 1.1: Database migrations (campus, person, address, person_role, assisted_profile, app_user, service_type, audit_log)
- [ ] 1.2: OIDC token validation middleware (coreos/go-oidc + Keycloak JWKS)
- [ ] 1.3: Local user auto-provisioning
- [ ] 1.4: RBAC middleware
- [ ] 1.5: Audit logging middleware
- [ ] 1.6: Campus-scoped data access
- [ ] 1.7: Seed data (service types, default campus)
- [ ] 1.8: React OIDC integration (keycloak-js)
- [ ] 1.9: React layout shell
- [ ] 1.10: React auth context
- [ ] 1.11: MFA for ADMIN role in Keycloak
- [ ] 1.12: Keycloak realm configuration as code

**Sprint 2 (Person Management):**
- [ ] 2.1: Person API CRUD
- [ ] 2.2: Person search (name, CPF, tsvector)
- [ ] 2.3: Duplicate detection
- [ ] 2.4: Person role management API
- [ ] 2.5: React person list page
- [ ] 2.6: React person form
- [ ] 2.7: React person detail page
- [ ] 2.8: IndexedDB with Dexie.js (person store)

**Sprint 3 (Triage & Attendance):**
- [ ] 3.1: Triage API (CRUD)
- [ ] 3.2: Attendance API (CRUD + transitions)
- [ ] 3.3: Attendance workflow state machine
- [ ] 3.4: React triage form
- [ ] 3.5: React attendance list and detail
- [ ] 3.6: React person history timeline
- [ ] 3.7: IndexedDB for triage and attendance

**Sprint 4 (Sync, Reports & Polish):**
- [ ] 4.1: Sync push endpoint
- [ ] 4.2: Sync pull endpoint
- [ ] 4.3: Frontend sync engine
- [ ] 4.4: Conflict detection and resolution UI
- [ ] 4.5: Basic reports (attendance count by period)
- [ ] 4.6: CSV export
- [ ] 4.7: UI polish and mobile responsiveness
- [ ] 4.8: End-to-end testing

For each task: verify code exists, review acceptance criteria, confirm tests cover it.

### Step 2: Run All Tests

```bash
# Backend
cd backend && go test ./... -v

# Frontend
cd frontend && npm run test

# Lint
cd backend && golangci-lint run
cd frontend && npm run lint
```

- [ ] All Go tests pass (0 failures).
- [ ] All frontend tests pass (0 failures).
- [ ] No Go lint warnings (golangci-lint).
- [ ] No TypeScript/ESLint warnings.

### Step 3: Verify Documentation Currency

- [ ] `docs/11-api-design.md` matches all implemented endpoints.
- [ ] `docs/10-data-model.md` matches all migration files in `backend/migrations/`.
- [ ] `docs/04-domain-model.md` matches Go domain structs.
- [ ] `docs/09-backlog.md` sprint stories marked as complete.
- [ ] `docs/08-roadmap.md` sprint tasks checked off.
- [ ] `HANDOFF.md` reflects current state.

### Step 4: Security Review Checklist

- [ ] Security review (review-security skill) completed for the sprint.
- [ ] No CRITICAL or HIGH security findings outstanding.
- [ ] RBAC middleware present on all endpoints.
- [ ] Campus isolation enforced on all data queries.
- [ ] Audit logging active for all mutations.
- [ ] No PII in logs or error responses.
- [ ] No hardcoded secrets in codebase.
- [ ] Keycloak realm configuration committed (`keycloak/realm-export.json`).

### Step 5: Offline Behavior Testing

- [ ] Person creation works offline (saves to IndexedDB).
- [ ] Triage creation works offline.
- [ ] Attendance creation works offline.
- [ ] Sync queue processes when online.
- [ ] Conflict detection shows user-visible indicator.
- [ ] Token refresh before sync works correctly.
- [ ] Data wipe on logout clears IndexedDB.

### Step 6: Mobile Responsiveness Testing

- [ ] All pages render correctly at 320px width.
- [ ] Navigation (navbar/sidebar) is usable on mobile.
- [ ] Forms are usable on mobile (fields don't overflow).
- [ ] Tables/lists have mobile-friendly layout.
- [ ] Touch targets are minimum 44px.
- [ ] No horizontal scrolling on mobile.

### Step 7: Cross-Browser Verification

- [ ] Chrome (latest): functional.
- [ ] Safari (latest): functional (important for iOS PWA).
- [ ] Firefox (latest): functional.
- [ ] Mobile Chrome (Android): functional.
- [ ] Mobile Safari (iOS): functional.

### Step 8: Generate Verdict

Classify issues:
- **BLOCKER**: Must fix before release (test failures, security critical/high, broken core flow).
- **KNOWN_ISSUE**: Can release with documented workaround.
- **DEFERRED**: Acceptable for next sprint.

## Outputs / Deliverables

A release readiness report:

```markdown
# Sprint [N] Release Readiness

## Verdict: GO / NO-GO

## Sprint Scope
- [x] Task 1 (complete)
- [x] Task 2 (complete)
- [ ] Task 3 (BLOCKER: reason)

## Test Results
- Backend: X passed, Y failed
- Frontend: X passed, Y failed
- Lint: clean / N warnings

## Security
- Review status: complete / pending
- Outstanding findings: none / list

## Offline
- Tested: yes / no
- Issues: none / list

## Mobile
- Tested at 320px: yes / no
- Issues: none / list

## Documentation
- Up to date: yes / no
- Gaps: none / list

## Blockers
1. Blocker description + owner + ETA

## Known Issues (ship with)
1. Issue description + workaround

## Recommendation
GO: All criteria met, ready to deploy.
NO-GO: N blockers must be resolved. ETA: X days.
```

## References

| Document | Path | Usage |
|----------|------|-------|
| Roadmap | `docs/08-roadmap.md` | Sprint task list |
| Backlog | `docs/09-backlog.md` | Story acceptance criteria |
| API design | `docs/11-api-design.md` | API conformance |
| Data model | `docs/10-data-model.md` | Schema conformance |
| Security | `docs/13-security-and-compliance.md` | Security requirements |
| Threat model | `docs/18-threat-model.md` | Security checklist |
| Offline sync | `docs/12-offline-sync-strategy.md` | Offline behavior |
| MVP scope | `docs/07-mvp-scope.md` | Feature scope |

## Constraints / Quality Bar

- Any test failure is a BLOCKER.
- Any CRITICAL security finding is a BLOCKER.
- Any HIGH security finding without remediation plan is a BLOCKER.
- Missing RBAC on an endpoint is a BLOCKER.
- Documentation out of sync with code is a BLOCKER.
- Mobile layout broken at 320px is a BLOCKER.
- Offline core flows broken is a BLOCKER.

## Interaction with Other Artifacts

- **Invoked by agents**: tech-lead (release gate).
- **Depends on skills**: review-code, review-security, review-api-contract, review-migration (all must have passed).
- **Feeds into**: release decision and HANDOFF.md update.
