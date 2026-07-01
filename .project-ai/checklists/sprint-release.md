# Sprint Release Checklist

Use this checklist at the end of each sprint before tagging a release. Every item must pass.

---

## Story Completion

- [ ] All sprint stories from `docs/08-roadmap.md` implemented and verified
- [ ] Each story passes its corresponding feature checklist (backend-feature-complete / frontend-feature-complete)
- [ ] No stories partially complete — either fully done or explicitly deferred to next sprint
- [ ] Deferred stories documented with rationale in HANDOFF.md

## Test Suite

- [ ] All backend tests pass: `make test` (zero failures)
- [ ] All frontend tests pass: `npm test` (zero failures)
- [ ] No skipped tests without documented justification
- [ ] New business logic has unit tests (service layer coverage)
- [ ] New form interactions have React Testing Library tests
- [ ] Repository integration tests run against real PostgreSQL
- [ ] E2E critical flows pass per `e2e-critical-flows.md` (FULL `npm run test:e2e` green + `auth_middleware_test.go` green)

## Code Quality

- [ ] Go linter passes: `make lint` (golangci-lint, zero warnings)
- [ ] TypeScript linter passes: ESLint (zero warnings)
- [ ] No `TODO`, `FIXME`, or `HACK` comments left unresolved
- [ ] No `console.log` statements in frontend production code
- [ ] No `_` for errors in Go code

## Security

- [ ] Security review completed for all security-sensitive changes (use `security-review.md` checklist)
- [ ] No hardcoded secrets in committed code
- [ ] No PII in logs or error responses
- [ ] RBAC middleware verified on all new endpoints
- [ ] Keycloak realm changes exported to `keycloak/realm-export.json` if modified

## Documentation Sync

- [ ] `docs/11-api-design.md` matches all implemented endpoints
- [ ] `docs/10-data-model.md` matches all database migrations
- [ ] `docs/04-domain-model.md` matches domain structs (if changed)
- [ ] `docs/12-offline-sync-strategy.md` updated for new offline features (if applicable)
- [ ] All new endpoints documented in API design doc
- [ ] All new tables/columns documented in data model doc

## Offline & Responsiveness

- [ ] Offline scenarios tested for all offline-capable features
- [ ] Graceful degradation verified — cached data shown or clear offline message
- [ ] Mobile responsiveness verified at 320px width
- [ ] Touch targets meet minimum 44x44px

## Migration Integrity

- [ ] All migrations have both `.up.sql` and `.down.sql` files
- [ ] Migrations run cleanly from scratch (`migrate up` on empty database)
- [ ] Migrations are reversible (`migrate down` then `migrate up` produces same schema)
- [ ] No data-destructive migrations without explicit approval

## Quality Gates

Per `docs/quality/quality-gates.md` — Overall Code Quality Gate:

- [ ] 0 blocker severity issues in the codebase
- [ ] 0 high severity issues in the codebase
- [ ] Test coverage (overall) ≥ threshold (70% Sprint 1-2, 80% Sprint 3+)
- [ ] Duplicated lines (overall) ≤ 5%
- [ ] Maintainability rating = A
- [ ] 0 reliability issues
- [ ] 0 security issues
- [ ] Security hotspots reviewed = 100%

## Release Artifacts

- [ ] HANDOFF.md updated with sprint summary:
  - What was built
  - Key decisions made
  - Known issues or tech debt
  - Recommended next steps
- [ ] Git tag created for sprint milestone (e.g., `sprint-1-complete`, `sprint-2-complete`)
- [ ] Commit history is clean and follows convention: `<type>: <description>`

---

## Sprint-Specific Gates

### Sprint 1 — Auth & Infrastructure
- [ ] Keycloak realm configured and `keycloak/realm-export.json` committed
- [ ] OIDC token validation working end-to-end
- [ ] RBAC middleware functional for all five roles
- [ ] Database migrations for core tables applied
- [ ] Health check endpoint (`/health`) responding
- [ ] Docker Compose development environment functional

### Sprint 2 — Person Management
- [ ] CRUD for persons with campus scoping
- [ ] Address management linked to persons
- [ ] Person role assignment working
- [ ] Assisted profile creation and management
- [ ] Search/filter persons by name, CPF, role

### Sprint 3 — Triage & Attendance
- [ ] Triage creation with requested services
- [ ] Triage lifecycle state machine (PENDING -> SCHEDULED -> IN_PROGRESS -> COMPLETED / CANCELLED)
- [ ] Attendance creation linked to triage
- [ ] Attendance state transitions with audit trail
- [ ] Service type seed data loaded

### Sprint 4 — Sync, Reports & Polish
- [ ] Offline data sync functional (Dexie.js queue + conflict resolution)
- [ ] Basic reports/dashboards rendering
- [ ] PWA manifest and service worker configured
- [ ] End-to-end user flows tested
- [ ] Performance acceptable on mobile devices

---

## How to Use

Run this checklist at sprint end. Use the `prepare-sprint-delivery` playbook to automate verification steps.

```
Playbook: prepare-sprint-delivery (automates test/lint/doc checks)
Skill:    assess-release-readiness (evaluates overall sprint quality)
Skill:    prepare-handoff (generates HANDOFF.md content)
Hook:     pre-release (final automated gate before tagging)
Agent:    tech-lead (for release approval)
```
