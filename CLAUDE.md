# CLAUDE.md - AI Agent Execution Rules for Chesed

This file defines permanent rules for Claude Code when working on this repository. Follow these rules exactly.

---

## Project Identity

- **Project**: Chesed - Instituto Nova SOS Management Platform
- **Repository**: `instituto-nova-sos/chesed`
- **Backend**: Go (Golang) with chi router, pgx, golang-migrate
- **Frontend**: React + TypeScript + Vite + Tailwind CSS
- **Database**: PostgreSQL 16
- **IAM**: Keycloak (OIDC) — external identity provider
- **Architecture**: API-first, offline-first PWA, mobile-first design

---

## Architecture Guardrails

### MUST follow:
1. Backend code goes in `backend/`. Frontend code goes in `frontend/`. Never mix.
2. All API endpoints are versioned under `/api/v1/`.
3. All database access goes through repository interfaces. Services never import database drivers directly.
4. All data queries MUST include `campus_id` filter from the authenticated user's Keycloak token claims.
5. All data mutations MUST create an audit log entry.
6. All records use UUID primary keys (for offline-safe creation).
7. All timestamps use `timestamptz` in PostgreSQL and `time.Time` in Go.
8. Every SQL migration has both `.up.sql` and `.down.sql` files.
9. All authentication is delegated to Keycloak. The Go API validates Keycloak-issued OIDC tokens via JWKS. No custom login/registration endpoints.
10. User provisioning goes through Keycloak Admin API. The `app_user` table is a local projection, not the source of truth for identity.
11. Keycloak realm configuration changes must be exported to `keycloak/realm-export.json` and committed.
12. The Go API MUST check `email_verified` claim in JWT tokens. Unverified users are rejected with HTTP 403.
13. MFA is mandatory for ADMIN and COORDINATOR roles (TOTP or Email OTP). Other roles can opt-in.

### MUST NOT do:
1. Never use global mutable state or singletons.
2. Never hardcode secrets, API keys, or credentials in source code.
3. Never skip error handling in Go (no `_` for errors).
4. Never use `any` type in TypeScript (strict mode enforced).
5. Never create API endpoints without RBAC middleware.
6. Never modify the audit_log table schema to allow updates or deletes.
7. Never store PII in logs, error messages, or client-visible error responses.
8. Never implement custom login forms, password hashing, or token issuance. All credential handling is Keycloak's responsibility.
9. Never store Keycloak client secrets or admin passwords in source code or Docker images.
10. Never bypass Keycloak for authentication (e.g., creating backdoor endpoints that accept raw credentials).

---

## Documentation-First Workflow

Before implementing any new feature or significant change:

1. Check the relevant doc in `docs/` (requirements, domain model, API design, etc.)
2. If the feature isn't documented, update the docs first, then implement.
3. When changing API endpoints, update `docs/11-api-design.md`.
4. When adding database tables/columns, update `docs/10-data-model.md`.
5. When modifying the domain model, update `docs/04-domain-model.md`.

---

## Code Structure Rules

### Backend (Go)
```
handler: Parse HTTP request → validate input → call service → write HTTP response
service: Business logic → validate rules → call repository → return result
repository: SQL query → scan result → return domain struct
domain: Pure data structs with no behavior or dependencies
middleware: Cross-cutting concerns (auth, audit, CORS, rate limit)
```

- handlers depend on services (never repositories)
- services depend on repository interfaces (defined in service package)
- repositories depend on domain structs and pgx
- domain has zero dependencies

### Frontend (React)
```
pages: Route-level components; compose from components + hooks
components: Reusable UI; no direct API calls
hooks: Custom hooks for auth, offline, sync, API calls
api: HTTP client functions (one file per domain)
offline: IndexedDB logic, sync queue, conflict resolution
types: Shared TypeScript interfaces
```

---

## Implementation Constraints

### Go Code
- Use `slog` for logging (stdlib, structured)
- Use `chi` for HTTP routing
- Use `pgx` for PostgreSQL (not `database/sql` with driver)
- Use `golang-migrate` for migrations (SQL files, not Go code)
- Use `coreos/go-oidc` for Keycloak OIDC token validation (not `golang-jwt` directly for token issuance)
- Use `go-playground/validator` for struct validation
- Test with `testing` + `testify` (table-driven tests)
- Format with `gofmt`; lint with `golangci-lint`

### React Code
- Use Vite for build tooling
- Use React Hook Form for forms
- Use Dexie.js for IndexedDB
- Use Tailwind CSS for styling (no CSS modules, no styled-components)
- Use Recharts for charts (Phase 2)
- Use React Router for routing
- Use Zustand or Context API for global state (no Redux)
- Use Vitest for testing
- Use `keycloak-js` for OIDC authentication flow (no custom login forms)

### Database
- PostgreSQL 16
- UUID primary keys on all tables
- `created_at`, `updated_at` timestamps on all tables
- `is_active` boolean for soft deletes
- `campus_id` on all operational tables
- JSONB for audit log `old_values` / `new_values`
- Indexes on all foreign keys and frequently filtered columns

---

## Product Scope Guardrails

- Respect roadmap phases strictly. Do not implement Phase 2 features (campaigns, donations, documents, consents, FOLLOW_UP state) during Phase 1.
- Phase 1 database tables: campus, person, address, person_role, assisted_profile, app_user, service_type, triage, triage_requested_service, attendance, attendance_transition, audit_log. Do not create Phase 2 tables prematurely.
- Service types are fixed seed data in MVP. Do not build admin UI for service type management.
- Each person/user belongs to exactly one campus in MVP. Do not build multi-campus assignment.
- Always validate a feature exists in `docs/03-requirements-catalog.md` before implementing it.
- Check `docs/07-mvp-scope.md` to confirm a feature belongs in the current phase before starting work.

---

## Quality Bar

Before considering any task complete:

1. All existing tests pass
2. New business logic has unit tests
3. **Integration tests cover every new API endpoint and every new client-server contract** (see "Integration Test Mandate" below)
4. No lint warnings (Go + TypeScript)
5. API responses match the documented format in `docs/11-api-design.md`
6. Mobile-responsive (test at 320px width)
7. Offline behavior considered (will this feature degrade gracefully?)
8. Audit logging present for data mutations
9. Campus scoping applied
10. **HANDOFF.md updated** with task progress (files created/modified, decisions, current state, next steps)

### Integration Test Mandate

Every new feature MUST include integration tests at the layer where production data crosses a process or network boundary. Unit tests prove the code is internally consistent; integration tests prove the contract with the rest of the system holds. Both are required — neither substitutes for the other.

**Backend integration tests** (`backend/internal/integration/`, `//go:build integration`):
- Run the real `chi` router → service → repository stack against a real PostgreSQL container booted by `testcontainers-go`.
- All migrations are applied from disk before each test so schema drift is caught immediately.
- One file per feature surface (`sync_test.go`, `triage_test.go`, etc.); add tests when you add an endpoint, table, or SQL constraint.
- Required scenarios for any new endpoint: happy path with DB-level assertions, the campus scoping boundary, every documented error code, and any uniqueness or transition constraint the SQL layer enforces.
- Run locally: `make test-integration`. Run in CI: the `Integration tests` step in `.github/workflows/backend.yml` (mandatory, blocking).

**Frontend integration tests** (`frontend/src/__integration__/`, suffix `.integration.test.ts(x)`):
- Use MSW (`msw/node`) to intercept `fetch` at the network boundary. This exercises the real `apiClient` → hook → component chain against a realistic HTTP surface without standing up the full stack.
- Required scenarios for any new API surface: happy path, server error mapping (status → `ApiError`), and any query-string or header contract the API client builds.
- One file per surface, co-located with related fixtures. Use the shared `server.ts` for cross-cutting handlers; `.use(...)` per test for case-specific overrides.
- Run locally: `npm run test:integration`. Run in CI: the `Integration tests` step in `.github/workflows/frontend.yml` (mandatory, blocking).

A PR that adds a new endpoint without backend integration tests, or a new API client function without frontend integration tests, fails the pre-merge gate. The reviewer agent must verify the integration test exists and exercises the documented contract — not just that any test exists.

---

## CI/CD Status — PAUSED (as of 2026-06-03)

GitHub Actions workflows are intentionally disabled while the project is in Phase 1 cost-control mode. Files under `.github/workflows/` and `.github/dependabot.yml` are preserved so re-enabling is a one-revert operation when budget allows.

**What this means for AI agents working in this repo:**
- Do NOT silently re-enable `on: push` / `on: pull_request` triggers, restore Dependabot limits, or change CI policy without explicit user approval.
- Quality gates (lint, tests) still apply — they just have to be run **locally** before pushing. Backend: `make test` (and `make test-integration` if integration suite present), `make lint`. Frontend: `npm test`, `npm run lint`, `npm run typecheck`, `npm run build`.
- The `pre-merge` quality gate hook in `.project-ai/hooks/pre-merge.md` is now operator-enforced (run locally), not CI-enforced.

To re-enable in the future:
1. Restore each workflow's `on:` block — every workflow file under `.github/workflows/` has a banner comment showing the original triggers commented out.
2. Restore Dependabot `open-pull-requests-limit` values — the banner in `.github/dependabot.yml` lists the original limits (5 / 5 / 3).
3. Optionally restore branch-protection rules requiring the workflows.

---

## Quality Governance

All code must follow the quality profiles, quality gates, and clean code guidelines defined in `docs/quality/`. These are non-negotiable engineering constraints.

### Quality Profiles

Two profiles enforce stack-specific standards:
- **Backend (Go)**: `docs/quality/quality-profiles.md` — error handling, context propagation, interface design, dependency direction, naming, testing.
- **Frontend (React/TS)**: `docs/quality/quality-profiles.md` — TypeScript strictness, component quality, hooks, forms, styling, authentication.

### Quality Gates (Mandatory)

Per `docs/quality/quality-gates.md`:

**New Code (every PR):**
- 0 bugs, 0 vulnerabilities, security hotspots 100% reviewed
- Test coverage ≥ 80%, duplication ≤ 3%
- Maintainability, reliability, security ratings = A

**Overall Code (sprint release):**
- 0 blocker/high issues, coverage ≥ 70% (tightens to 80%), duplication ≤ 5%
- All ratings = A

Quality gates are enforced by the `pre-merge` hook (blocking) and `pre-release` hook (blocking). No exceptions without an ADR.

### Complexity Thresholds

Per `docs/quality/complexity-guidelines.md`:

| Metric | Go | React/TS |
|--------|-----|----------|
| Cognitive complexity / function | 25 | 15 |
| Cyclomatic complexity / function | 10 | 10 |
| Function length | 40 lines | 50 lines |
| File length | 400 lines | 300 lines |
| Nesting depth | 3 | 3 |
| Parameter count | 5 | 5 |
| Return values | 3 | — |
| Component JSX lines | — | 80 |

### Clean Code Categories

Per `docs/quality/clean-code-guidelines.md`, all code must satisfy:
- **Consistency**: Follow established patterns uniformly.
- **Intentionality**: Names reveal purpose. No dead code.
- **Adaptability**: Dependencies point inward. Changes confined to appropriate layer.
- **Responsibility**: Each function does one thing.

### Software Qualities (Non-Negotiable)

Per `docs/quality/quality-profiles.md`:
- **Security**: OWASP Top 10 protection, Keycloak-only auth, no exposed PII, audit logging.
- **Reliability**: Every error handled, consistent state transitions, graceful degradation.
- **Maintainability**: Low complexity, modular design, minimal duplication.

### AI Agent Responsibility

The AI agent MUST:
- Enforce quality profiles during implementation and review.
- Run quality gate validation before marking work complete.
- Proactively identify and fix quality issues.
- Use the `refactor-for-quality` playbook when quality gates fail.
- Never approve code that violates quality gates.

### Quality Documentation

- Quality profiles: `docs/quality/quality-profiles.md`
- Clean code guidelines: `docs/quality/clean-code-guidelines.md`
- Quality gates: `docs/quality/quality-gates.md`
- Complexity guidelines: `docs/quality/complexity-guidelines.md`

---

## Commit Convention

```
<type>: <short description>

Types: feat, fix, refactor, test, docs, chore, ci
```

---

## Key Documentation References

- Requirements: `docs/03-requirements-catalog.md`
- Domain model: `docs/04-domain-model.md`
- Architecture: `docs/05-architecture-proposal.md`
- MVP scope: `docs/07-mvp-scope.md`
- Backlog: `docs/09-backlog.md`
- Data model: `docs/10-data-model.md`
- API design: `docs/11-api-design.md`
- Offline sync: `docs/12-offline-sync-strategy.md`
- Security: `docs/13-security-and-compliance.md`
- IAM: `docs/16-iam-and-access-control.md`
- Threat model: `docs/18-threat-model.md`
- Implementation: `docs/15-implementation-guidelines.md`
- Keycloak config: `docs/20-keycloak-configuration.md`
- Security tests: `docs/17-security-test-strategy.md`
- Secure dev: `docs/19-secure-development-standard.md`
- Quality profiles: `docs/quality/quality-profiles.md`
- Clean code: `docs/quality/clean-code-guidelines.md`
- Quality gates: `docs/quality/quality-gates.md`
- Complexity: `docs/quality/complexity-guidelines.md`

---

## Language

All code comments, variable names, API fields, and documentation MUST be in English. The UI text (displayed to users) will be in Portuguese (Brazilian) via i18n.

---

## AI-Assisted Delivery Operating Model

This repository includes a project-level AI delivery operating system at `.project-ai/`. It provides skills, agents, hooks, rules, playbooks, templates, checklists, and workflows that orchestrate how work flows through the delivery lifecycle.

### How to Use

- Start every feature with `.project-ai/workflows/feature-delivery.md`
- Follow the relevant playbook for implementation (backend endpoint, frontend page, database table, offline support)
- Run the appropriate checklist before marking work complete
- Use the operating model index at `.project-ai/OPERATING_MODEL.md` to find the right artifact

### Continuous Improvement Mandate

During the development lifecycle, the AI agent MUST continuously evaluate whether the `.project-ai/` tools:

1. **Are sufficient** — Do they cover the current delivery needs?
2. **Need refinement** — Are any artifacts outdated, unclear, or missing steps?
3. **Need consolidation** — Are any artifacts redundant or overlapping?
4. **Need additions** — Are there recurring patterns that would benefit from a new artifact?

When recurring friction, ambiguity, repeated work, or quality gaps are identified, the AI agent SHOULD:

- Propose improvements to existing skills, agents, hooks, rules, playbooks, templates, checklists, or workflows
- Create new artifacts when justified by repeated patterns
- Update existing artifacts when outdated
- Remove artifacts that prove unnecessary

These tools are **living project artifacts** that evolve with the project. Maintaining and improving them is an explicit responsibility during delivery.
