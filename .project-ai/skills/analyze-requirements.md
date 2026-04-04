# Skill: Analyze Requirements

## Purpose

Break a backlog story into implementable development tasks. Cross-reference the requirements catalog, MVP scope, roadmap, data model, and API design to produce an ordered, dependency-aware task list with full traceability to project artifacts.

## When to Use / Trigger

- A new backlog story is picked for implementation (e.g., "Implement S03.1 - Create person").
- A user says "analyze story X" or "break down feature Y".
- Before starting any non-trivial implementation work (3+ files affected).

## Inputs

| Input | Required | Source |
|-------|----------|--------|
| Story ID or feature description | Yes | `docs/09-backlog.md` or user request |
| Sprint context | Optional | `docs/08-roadmap.md` |

## Process

### Step 1: Locate the Story

1. Open `docs/09-backlog.md` and find the story by ID (e.g., S03.1).
2. Read the full acceptance criteria.
3. Note the epic (E01-E06) and target sprint.

### Step 2: Validate Phase Scope

1. Open `docs/07-mvp-scope.md`.
2. Confirm the feature is listed under "Must Have" for the current phase.
3. Check the "Won't Have in MVP" list to ensure no Phase 2/3 features are being pulled in.
4. If the story references Phase 2 tables (campaign, campaign_team, document, consent, donation), STOP and flag a phase boundary violation.

### Step 3: Map to Requirements

1. Open `docs/03-requirements-catalog.md`.
2. Identify all RF-XX requirements the story satisfies.
3. List any non-functional requirements (RNF-XX) that apply.
4. Note any ambiguity resolutions or design decisions documented in the requirements.

### Step 4: Identify Affected Data Model Artifacts

1. Open `docs/10-data-model.md`.
2. List all tables the story reads from or writes to.
3. For each table, note: primary key, required columns, foreign keys, check constraints, indexes.
4. Determine if a new migration is needed or if existing schema covers the story.
5. Phase 1 tables only: campus, person, address, person_role, assisted_profile, app_user, service_type, triage, triage_requested_service, attendance, attendance_transition, audit_log.

### Step 5: Map API Endpoints

1. Open `docs/11-api-design.md`.
2. Identify all endpoints the story requires (method, path, request/response schemas).
3. Note RBAC roles required per endpoint.
4. Note pagination format: `{ data: [...], pagination: { page, per_page, total, total_pages } }`.
5. Note error response format and status codes.
6. Confirm `campus_id` scoping is applied.

### Step 6: Assess Offline Requirements

1. Open `docs/12-offline-sync-strategy.md`.
2. Determine if the feature must work offline (person creation, triage, attendance).
3. If yes, note: Dexie.js table, sync queue entry format, client UUID generation, LWW conflict handling.
4. Reference data (service_type) is pre-cached, not created offline.

### Step 7: Assess Security Considerations

1. Open `docs/18-threat-model.md` -- identify relevant threats (T1-T12).
2. Open `docs/13-security-and-compliance.md` -- check LGPD data classification tier for affected data.
3. Verify: RBAC middleware required, audit logging for mutations, no PII in logs, campus_id filtering from JWT.

### Step 8: Generate Task List

Produce an ordered task list following the implementation dependency chain:

```
1. Migration (if needed)       -- schema changes first
2. Domain structs              -- pure data types
3. Repository interface + impl -- database access
4. Service layer               -- business logic
5. Handler + routes            -- HTTP layer with RBAC
6. Audit log integration       -- mutation logging
7. Backend unit tests          -- table-driven tests
8. Backend integration tests   -- repository tests against PostgreSQL
9. Frontend TypeScript types   -- shared interfaces
10. Frontend API client        -- HTTP functions
11. Frontend hooks             -- data fetching + offline
12. Frontend components        -- UI elements
13. Frontend page              -- route-level composition
14. Offline support (if applicable)
15. Frontend tests             -- Vitest + RTL
16. Documentation updates      -- API, data model, domain model docs
```

## Outputs / Deliverables

A structured analysis containing:

1. **Story Summary**: ID, title, epic, sprint, requirements covered.
2. **Affected Tables**: List with column details relevant to the story.
3. **Affected Endpoints**: Method, path, roles, request/response shapes.
4. **Offline Requirements**: Yes/No with Dexie schema if applicable.
5. **Security Considerations**: Threats, LGPD classification, RBAC, audit.
6. **Ordered Task List**: Each task with:
   - File path(s) to create or modify.
   - Dependencies on other tasks.
   - Estimated complexity (S/M/L).
7. **Open Questions**: Anything ambiguous that needs clarification before implementation.

## References

| Document | Path | Usage |
|----------|------|-------|
| Requirements catalog | `docs/03-requirements-catalog.md` | RF-XX traceability |
| Domain model | `docs/04-domain-model.md` | Entity relationships |
| MVP scope | `docs/07-mvp-scope.md` | Phase boundary validation |
| Roadmap | `docs/08-roadmap.md` | Sprint context |
| Backlog | `docs/09-backlog.md` | Story definitions |
| Data model | `docs/10-data-model.md` | Table DDL |
| API design | `docs/11-api-design.md` | Endpoint contracts |
| Offline sync | `docs/12-offline-sync-strategy.md` | Offline behavior |
| Security | `docs/13-security-and-compliance.md` | LGPD, data classification |
| Threat model | `docs/18-threat-model.md` | T1-T12 threats |

## Constraints / Quality Bar

- Every task in the output must trace back to a documented requirement (RF-XX or RNF-XX).
- No task may reference Phase 2 tables or features.
- All mutations must include an audit log task.
- All endpoints must include RBAC middleware.
- All queries must include campus_id filtering.
- If offline support is required, it must be a separate task with Dexie schema details.

## Interaction with Other Artifacts

- **Used by agents**: tech-lead (planning), backend-engineer (before implementation), frontend-engineer (before implementation).
- **Feeds into skills**: design-backend-feature, design-frontend-feature, design-offline-support.
- **Updates**: maintain-docs skill should be invoked after implementation to update affected documentation.
