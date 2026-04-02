# CLAUDE.md - AI Agent Execution Rules for Chesed

This file defines permanent rules for Claude Code when working on this repository. Follow these rules exactly.

---

## Project Identity

- **Project**: Chesed - Instituto Nova SOS Management Platform
- **Repository**: `instituto-nova-sos/chesed`
- **Backend**: Go (Golang) with chi router, pgx, golang-migrate
- **Frontend**: React + TypeScript + Vite + Tailwind CSS
- **Database**: PostgreSQL 16
- **Architecture**: API-first, offline-first PWA, mobile-first design

---

## Architecture Guardrails

### MUST follow:
1. Backend code goes in `backend/`. Frontend code goes in `frontend/`. Never mix.
2. All API endpoints are versioned under `/api/v1/`.
3. All database access goes through repository interfaces. Services never import database drivers directly.
4. All data queries MUST include `campus_id` filter from the authenticated user's JWT claims.
5. All data mutations MUST create an audit log entry.
6. All records use UUID primary keys (for offline-safe creation).
7. All timestamps use `timestamptz` in PostgreSQL and `time.Time` in Go.
8. Every SQL migration has both `.up.sql` and `.down.sql` files.

### MUST NOT do:
1. Never use global mutable state or singletons.
2. Never hardcode secrets, API keys, or credentials in source code.
3. Never skip error handling in Go (no `_` for errors).
4. Never use `any` type in TypeScript (strict mode enforced).
5. Never create API endpoints without RBAC middleware.
6. Never modify the audit_log table schema to allow updates or deletes.
7. Never store PII in logs, error messages, or client-visible error responses.

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
- Use `golang-jwt` for JWT handling
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

### Database
- PostgreSQL 16
- UUID primary keys on all tables
- `created_at`, `updated_at` timestamps on all tables
- `is_active` boolean for soft deletes
- `campus_id` on all operational tables
- JSONB for audit log `old_values` / `new_values`
- Indexes on all foreign keys and frequently filtered columns

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

---

## Language

All code comments, variable names, API fields, and documentation MUST be in English. The UI text (displayed to users) will be in Portuguese (Brazilian) via i18n.
