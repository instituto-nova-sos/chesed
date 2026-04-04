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
3. No lint warnings (Go + TypeScript)
4. API responses match the documented format in `docs/11-api-design.md`
5. Mobile-responsive (test at 320px width)
6. Offline behavior considered (will this feature degrade gracefully?)
7. Audit logging present for data mutations
8. Campus scoping applied

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
