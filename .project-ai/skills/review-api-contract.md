# Skill: Review API Contract

## Purpose

Validate that code changes (handlers, routes, request/response types) conform to the API contract documented in `docs/11-api-design.md`. Produces a conformance report identifying deviations between implementation and specification.

## When to Use / Trigger

- After implementing or modifying any HTTP handler or route.
- During code review of backend PRs that touch `handler/` or `middleware/`.
- When a user says "check API conformance" or "review API contract for endpoint X".
- Before marking any backend story as complete.

## Role / Expertise

API design reviewer who validates REST conventions, schema conformance, RBAC enforcement, and campus scoping against the project's documented API specification.

## Inputs

| Input | Required | Source |
|-------|----------|--------|
| Code files being reviewed | Yes | `backend/internal/handler/`, route registration files |
| API specification | Yes | `docs/11-api-design.md` |
| Domain struct definitions | Yes | `backend/internal/domain/` |

## Process

### Step 1: Identify Affected Endpoints

1. Read the changed handler files.
2. Map each handler function to its registered route (method + path).
3. Cross-reference with the endpoint table in `docs/11-api-design.md`.

### Step 2: Validate HTTP Method and Path

For each endpoint, verify:
- Method matches (GET, POST, PUT, PATCH, DELETE).
- Path follows `/api/v1/{resource}` or `/api/v1/{resource}/{id}` pattern.
- Path parameters use correct names (`:id`, `:roleId`, etc.).
- No unversioned endpoints.

### Step 3: Validate Request Schema

For POST/PUT/PATCH endpoints:
- JSON body fields match the documented request schema.
- Required fields are enforced.
- Field names use snake_case (matching the API spec).
- `sync_id` field is present for offline-creatable entities (person, triage, attendance).
- No undocumented fields accepted without API doc update.

### Step 4: Validate Response Schema

For each endpoint:
- Response body structure matches documented shape.
- Field names use snake_case.
- List endpoints use pagination wrapper: `{ data: [...], pagination: { page, per_page, total, total_pages } }`.
- Create endpoints return 201 with the created resource.
- Detail endpoints return 200 with the full resource.
- Error responses use format: `{ "error": { "code": "...", "message": "...", "details": [...] } }`.

### Step 5: Validate Status Codes

| Scenario | Expected Code |
|----------|--------------|
| Successful creation | 201 |
| Successful retrieval | 200 |
| Successful update | 200 |
| Validation error | 400 |
| Missing/invalid token | 401 |
| Insufficient role | 403 |
| Resource not found | 404 |
| Duplicate/conflict | 409 |
| Server error | 500 |

### Step 6: Validate RBAC

For each endpoint:
- RBAC middleware is applied (not optional).
- Allowed roles match the "Roles" column in the API spec endpoint table.
- Role hierarchy is respected: ADMIN > COORDINATOR > PROFESSIONAL > SECRETARY > VOLUNTEER.
- Endpoint-specific role restrictions from `docs/11-api-design.md`:
  - Person creation: All roles.
  - Person update: Secretary+.
  - Person history: Professional+.
  - Role management: Coordinator+.
  - Attendance creation: Secretary+.
  - Attendance update/transition: Professional+.
  - User management: Admin only.
  - Reports export: Coordinator+.

### Step 7: Validate Campus Scoping

- All list queries include `campus_id` filter derived from JWT claims.
- All detail queries verify the resource belongs to the user's campus.
- `campus_id` is NEVER accepted from the request body or query parameter for filtering (always from JWT).
- Admin users may have cross-campus access (documented exception).

### Step 8: Validate Authentication

- All endpoints require authentication (no public endpoints except health check).
- Auth middleware is applied at the router group level.
- Token validation uses Keycloak JWKS endpoint.
- Expected claims extracted: `sub`, `realm_access.roles`, `campus_id`, `person_id`.

## Outputs / Deliverables

A conformance report with:

1. **Summary**: Total endpoints reviewed, pass/fail count.
2. **Per-endpoint report**:
   - Endpoint: `METHOD /path`
   - Status: PASS / DEVIATION / MISSING
   - Deviations (if any): field name mismatch, wrong status code, missing RBAC, etc.
   - Recommendation: specific fix required.
3. **Undocumented endpoints**: Any implemented endpoints not in the API spec (require doc update or removal).
4. **Missing implementations**: Any documented endpoints not yet implemented.

## References

| Document | Path | Usage |
|----------|------|-------|
| API design | `docs/11-api-design.md` | Source of truth for endpoint contracts |
| IAM and access control | `docs/16-iam-and-access-control.md` | RBAC roles and token claims |
| Implementation guidelines | `docs/15-implementation-guidelines.md` | Handler patterns |

## Constraints / Quality Bar

- Zero deviations from the API spec is the target. Any deviation must be either fixed in code or documented as an intentional API change (with doc update).
- RBAC middleware must be present on every endpoint (no exceptions except `/api/v1/health`).
- Campus scoping must be present on every data endpoint.
- Error responses must never contain PII or internal stack traces.
- Pagination format must be consistent across all list endpoints.

## Interaction with Other Artifacts

- **Invoked by agents**: tech-lead (quality gate), backend-engineer (self-review).
- **Used alongside skills**: review-code (broader code review), review-security (security focus).
- **Triggers skills**: maintain-docs (if intentional API changes need documentation).
