# Feature Specification: [Feature Name]

## Metadata

| Field | Value |
|-------|-------|
| **Story Reference** | S0x.x — [Story title from docs/09-backlog.md] |
| **Phase** | 1 |
| **Sprint** | [1-4] |
| **Author** | [Name] |
| **Date** | [YYYY-MM-DD] |
| **Status** | Draft / In Review / Approved / Implemented |

---

## Acceptance Criteria

Copy directly from the story in `docs/09-backlog.md`. Each criterion must be independently verifiable.

- [ ] [Criterion 1 from backlog]
- [ ] [Criterion 2 from backlog]
- [ ] [Criterion 3 from backlog]
- [ ] [Criterion N from backlog]

---

## Affected Tables

Reference: `docs/10-data-model.md`

| Table | Operation | Notes |
|-------|-----------|-------|
| [table_name] | Read / Write / Create | [Specific columns or constraints relevant to this feature] |
| [table_name] | Read | [e.g., "Join for person lookup"] |

Phase 1 tables only: campus, person, address, person_role, assisted_profile, app_user, service_type, triage, triage_requested_service, attendance, attendance_transition, audit_log.

---

## Affected Endpoints

Reference: `docs/11-api-design.md`

| Method | Path | Status | Notes |
|--------|------|--------|-------|
| [GET/POST/PUT/PATCH] | /api/v1/[path] | Existing / New | [Brief note on what this feature needs from the endpoint] |

---

## RBAC

Reference: `docs/16-iam-and-access-control.md`

| Action | Required Role | Middleware |
|--------|--------------|-----------|
| [e.g., Create person] | All authenticated | `middleware.Authenticate` |
| [e.g., Update person] | Secretary+ | `middleware.RequireRole("SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN")` |
| [e.g., View history] | Professional+ | `middleware.RequireRole("PROFESSIONAL", "COORDINATOR", "ADMIN")` |

---

## Offline Support

| Classification | Justification |
|---|---|
| **Fully offline-capable** / **Read-only offline** / **Online-only** | [Reason. Reference RF-46/47/48 from docs/03-requirements-catalog.md if applicable] |

If offline-capable:
- **Local storage**: Dexie table `[table_name]` in `frontend/src/offline/db.ts`
- **Sync entity type**: `[entity_type]` in sync push/pull
- **Encrypted fields**: [List PII fields that must be encrypted in IndexedDB]
- **Conflict strategy**: Last-write-wins (MVP) / [other]

If online-only:
- **Offline behavior**: Show message "Esta funcionalidade requer conexao com a internet"

---

## Security Considerations

Reference: `docs/18-threat-model.md`

| Threat | Relevance | Mitigation |
|--------|-----------|-----------|
| [T1-T12 ID] | [Why this threat applies] | [How the implementation mitigates it] |

Additional checks:
- [ ] No PII in logs or error responses
- [ ] Campus scoping enforced
- [ ] Audit logging for all mutations
- [ ] Input validation (server-side)

---

## Implementation Tasks

### Backend

1. [ ] **Domain**: Create/update struct in `backend/internal/domain/[entity].go`
   - Fields: [list key fields with types]
   - Validator tags: [list constraints]

2. [ ] **Repository**: Implement in `backend/internal/repository/[entity]_repository.go`
   - Methods: [Create, GetByID, List, Update, SoftDelete]
   - Campus scoping: All queries include `campus_id` filter

3. [ ] **Service**: Implement in `backend/internal/service/[entity]_service.go`
   - Business rules: [list key validations or logic]
   - Audit logging: [which mutations are logged]

4. [ ] **Handler**: Implement in `backend/internal/handler/[entity]_handler.go`
   - Endpoints: [list method + path]
   - Request parsing: [JSON body / query params / path params]

5. [ ] **Routes**: Register in `backend/cmd/server/main.go`
   - RBAC: [list role middleware per route]

### Frontend

6. [ ] **Types**: Create `frontend/src/types/[entity].ts`
   - Interfaces: [list interface names]

7. [ ] **API client**: Create `frontend/src/api/[entity]Api.ts`
   - Methods: [list API methods]

8. [ ] **Hooks**: Create `frontend/src/hooks/use[Entity].ts`
   - State management: [loading, error, data, pagination]

9. [ ] **Components**: Create in `frontend/src/components/[entity]/`
   - [ComponentName]: [brief description]
   - Forms: React Hook Form + Zod schema

10. [ ] **Page**: Create `frontend/src/pages/[Entity]/[PageName].tsx`
    - Route: [path]
    - Auth guard: [role requirement]

11. [ ] **Offline** (if applicable): Update `frontend/src/offline/db.ts`
    - Dexie table: [table name and indexes]
    - Sync queue: [entity type]
    - Encryption: [encrypted fields]

### Tests

12. [ ] **Service unit tests**: `backend/internal/service/[entity]_service_test.go`
    - Table-driven tests for: [list scenarios]

13. [ ] **Repository integration tests**: `backend/internal/repository/[entity]_repository_test.go`
    - Campus isolation test
    - Soft delete test

14. [ ] **Frontend hook tests**: `frontend/src/hooks/__tests__/use[Entity].test.ts`

15. [ ] **Frontend form tests**: `frontend/src/components/[entity]/__tests__/[Form].test.tsx`
    - Validation tests
    - Submission tests

### Documentation

16. [ ] **API docs**: Verify `docs/11-api-design.md` matches implementation
17. [ ] **Data model**: Verify `docs/10-data-model.md` matches migrations

---

## Dependencies

| Dependency | Status | Notes |
|---|---|---|
| [e.g., campus table migration] | Done / In Progress / Blocked | [S01.4] |
| [e.g., auth middleware] | Done / In Progress / Blocked | [S02.1] |
| [e.g., Keycloak realm configured] | Done / In Progress / Blocked | [Phase 0] |

---

## Notes

[Any additional context, design decisions, trade-offs, or open questions.]
