# CODEX.md - AI Coding Agent Execution Rules for Chesed

This file provides execution context for any AI coding agent (Codex, Cursor, Copilot, or similar) working on this repository. It is equivalent to `CLAUDE.md` but written to be agent-agnostic.

---

## Project Overview

**Chesed** is a management platform for Instituto Nova SOS, a social NGO. It manages beneficiaries, volunteers, service attendance, campaigns, donations, and consent.

- **Backend**: Go (Golang) — REST API at `backend/`
- **Frontend**: React + TypeScript + Vite — PWA at `frontend/`
- **Database**: PostgreSQL 16
- **IAM**: Keycloak (OIDC) — external identity provider
- **Key feature**: Offline-first mobile PWA for field operations

---

## Before You Start Coding

1. Read the relevant documentation in `docs/`:
   - `03-requirements-catalog.md` — what to build
   - `04-domain-model.md` — entity relationships
   - `07-mvp-scope.md` — what's in/out of scope
   - `09-backlog.md` — stories with acceptance criteria
   - `10-data-model.md` — database schema
   - `11-api-design.md` — API contracts
2. Check `CLAUDE.md` (root) for architectural constraints (they apply to all agents)
3. Run `make test` to ensure the codebase is green before making changes
4. Run `make lint` to check for lint errors

---

## Architecture Rules (Non-Negotiable)

These rules MUST be followed. Violating them will break the system or create security vulnerabilities.

### Data Access
- All queries MUST filter by `campus_id` (from JWT claims)
- All mutations MUST write to the `audit_log` table
- All database access MUST go through repository interfaces
- All records MUST use UUID primary keys

### Security
- All API endpoints MUST have RBAC middleware
- Never hardcode secrets in source code
- Never log PII (names, CPF, phone numbers)
- Never return stack traces in API error responses
- Never create endpoints without authentication (except `/health`). Authentication is handled by Keycloak (OIDC).
- Never write custom login, registration, or password reset endpoints. Keycloak handles all identity flows.
- Never implement custom password hashing or token issuance. The Go API only validates Keycloak-issued tokens.

### Architecture
- Backend handlers call services, services call repositories. Never skip layers.
- Frontend pages compose components and hooks. Components never call APIs directly.
- Offline-created records use client-generated UUIDs and a sync queue.

---

## Tech Stack Reference

### Backend Dependencies
| Package | Purpose |
|---------|---------|
| `go-chi/chi` | HTTP router |
| `jackc/pgx` | PostgreSQL driver |
| `golang-migrate/migrate` | Database migrations |
| `coreos/go-oidc` | OIDC token validation (Keycloak) |
| `go-playground/validator` | Input validation |
| `google/uuid` | UUID generation |
| `log/slog` (stdlib) | Structured logging |

### Frontend Dependencies
| Package | Purpose |
|---------|---------|
| `react` + `react-dom` | UI framework |
| `react-router-dom` | Client-side routing |
| `react-hook-form` | Form management |
| `dexie` | IndexedDB wrapper (offline storage) |
| `tailwindcss` | Utility-first CSS |
| `vite-plugin-pwa` | PWA + Service Worker |
| `recharts` | Charts (Phase 2) |
| `keycloak-js` | OIDC authentication (Keycloak adapter) |
| `react-signature-canvas` | Consent signature capture (Phase 2) |

---

## File Structure

```
backend/
├── cmd/server/main.go           # Entry point
├── internal/
│   ├── config/                  # Env config loading
│   ├── domain/                  # Pure structs (zero deps)
│   ├── handler/                 # HTTP handlers
│   ├── service/                 # Business logic
│   ├── repository/              # SQL data access
│   ├── middleware/              # Auth, audit, CORS
│   └── sync/                   # Offline sync engine
├── migrations/                  # SQL up/down files
└── Makefile

frontend/
├── src/
│   ├── api/                    # API client layer
│   ├── components/             # Reusable UI
│   ├── hooks/                  # Custom hooks
│   ├── offline/                # IndexedDB + sync
│   ├── pages/                  # Route pages
│   ├── store/                  # Global state
│   ├── types/                  # TypeScript interfaces
│   └── utils/                  # Helpers
└── vite.config.ts
```

---

## Common Tasks

### Add a new API endpoint
1. Define the domain struct in `internal/domain/`
2. Add the repository interface method in `internal/service/`
3. Implement the repository in `internal/repository/`
4. Add the service method in `internal/service/`
5. Add the handler in `internal/handler/`
6. Register the route in `cmd/server/main.go`
7. Add RBAC role requirement to the route (roles are extracted from Keycloak token claims)
8. Update `docs/11-api-design.md`
9. Write tests for service and handler

### Add a new database table
1. Create `NNNNNN_description.up.sql` and `NNNNNN_description.down.sql` in `backend/migrations/`
2. Add the domain struct in `internal/domain/`
3. Add the repository implementation
4. Update `docs/10-data-model.md`
5. Run `make migrate-up`

### Add a new React page
1. Create the page component in `src/pages/`
2. Add the route in `App.tsx`
3. Create API functions in `src/api/` if new endpoints are needed
4. Use React Hook Form for forms
5. Add IndexedDB store in `src/offline/` if offline support is needed
6. Ensure mobile-responsive at 320px width

### Add offline support for a feature
1. Add Dexie.js table in `src/offline/db.ts`
2. Create sync queue entry on local save
3. Update `src/offline/queue.ts` to handle the new entity type
4. Test: disconnect network → create record → reconnect → verify sync

---

## Testing

### Run all tests
```bash
# Backend
cd backend && make test

# Frontend
cd frontend && npm test
```

### Write a Go test
```go
func TestPersonService_Create(t *testing.T) {
    tests := []struct {
        name    string
        input   domain.CreatePersonInput
        wantErr bool
    }{
        {
            name:    "valid person",
            input:   domain.CreatePersonInput{FullName: "Maria", DocumentNumber: "123.456.789-00"},
            wantErr: false,
        },
        {
            name:    "missing name",
            input:   domain.CreatePersonInput{},
            wantErr: true,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ...
        })
    }
}
```

---

## Naming Conventions

| Go | TypeScript | Database |
|----|-----------|----------|
| PascalCase types | PascalCase interfaces | snake_case columns |
| camelCase functions | camelCase functions | snake_case tables |
| UPPER_CASE constants | UPPER_CASE constants | UPPER_CASE enums |
| snake_case files | PascalCase components | NNNNNN_name migrations |

---

## Product Scope Guardrails

- Respect roadmap phases. Do not implement Phase 2 features (campaigns, donations, documents, consents, FOLLOW_UP state) during Phase 1.
- Phase 1 tables: campus, person, address, person_role, assisted_profile, app_user, service_type, triage, triage_requested_service, attendance, attendance_transition, audit_log.
- Service types are fixed seed data in MVP. No admin UI for managing them.
- Each person/user belongs to exactly one campus in MVP.
- Validate features exist in `docs/03-requirements-catalog.md` before implementing.
- Check `docs/07-mvp-scope.md` to confirm phase assignment before starting.

---

## Quality Checklist

Before submitting code:

- [ ] `make test` passes
- [ ] `make lint` passes (Go + TypeScript)
- [ ] New business logic has tests
- [ ] Campus scoping applied to queries
- [ ] Audit logging on mutations
- [ ] RBAC middleware on endpoints
- [ ] Mobile-responsive UI (320px minimum)
- [ ] Offline behavior considered
- [ ] API docs updated if endpoints changed
- [ ] No hardcoded secrets
- [ ] No PII in logs

---

## Language Policy

- **Code**: English (variable names, comments, API fields)
- **UI text**: Portuguese (Brazilian) via i18n system
- **Documentation**: English

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
