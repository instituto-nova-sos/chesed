# Skill: Design API Contract

## Purpose

Design a complete REST API contract from requirements, producing endpoint definitions, request/response schemas, status codes, RBAC roles, pagination format, and error formats — all before any implementation begins. This is the creative counterpart to `review-api-contract` (which validates existing contracts against the spec).

## When to Use / Trigger

- During the DESIGN phase of feature delivery, after story analysis is complete.
- When a new domain resource needs API endpoints (CRUD or specialized operations).
- When a user says "design API for feature X" or "define endpoints for resource Y".
- Before writing any handler code for a new resource.

## Role / Expertise

Senior API architect with expertise in:
- RESTful API design conventions and resource modeling.
- JSON schema design with consistent naming (snake_case).
- Pagination patterns for list endpoints.
- RBAC role-based access control per endpoint.
- Error response standardization.
- Campus-scoped multi-tenant API design.

## Inputs

| Input | Required | Source |
|-------|----------|--------|
| Story analysis (from analyze-requirements) | Yes | Prior skill output |
| Domain model | Yes | `docs/04-domain-model.md` |
| Existing API conventions | Yes | `docs/11-api-design.md` |
| Data model (table DDL) | Yes | `docs/10-data-model.md` |
| RBAC role hierarchy | Yes | `docs/16-iam-and-access-control.md` |
| Existing endpoints | Yes | `docs/11-api-design.md` (to avoid conflicts) |

## Process

### Step 1: Identify Resources and Operations

1. Read the story analysis and identify the domain entities involved.
2. Map each entity to a REST resource (noun, plural form).
3. Identify the required operations:
   - **CRUD**: Create, Read (single + list), Update, Delete (soft delete via `is_active`).
   - **Specialized**: State transitions, search, export, batch operations.
4. Check `docs/11-api-design.md` for existing endpoints on the same resource — extend, don't duplicate.

### Step 2: Define Endpoint Paths

Follow existing conventions from `docs/11-api-design.md`:

| Operation | Method | Path Pattern |
|-----------|--------|-------------|
| Create | POST | `/api/v1/{resources}` |
| List (paginated) | GET | `/api/v1/{resources}?page=1&per_page=20` |
| Get by ID | GET | `/api/v1/{resources}/{id}` |
| Update | PUT | `/api/v1/{resources}/{id}` |
| Soft delete | PATCH | `/api/v1/{resources}/{id}/deactivate` |
| State transition | POST | `/api/v1/{resources}/{id}/transitions` |
| Search | GET | `/api/v1/{resources}/search?q=term` |
| Sub-resources | GET | `/api/v1/{resources}/{id}/{sub-resources}` |

Rules:
- All paths versioned under `/api/v1/`.
- Resource names are plural, lowercase, hyphenated (e.g., `service-types`, `person-roles`).
- Path parameters use `{id}` or `{descriptive_id}` format.
- No verbs in paths except for specialized actions (`/transitions`, `/deactivate`, `/export`).

### Step 3: Design Request Schemas

For each POST/PUT endpoint, define the request body:

1. Read the domain entity attributes from `docs/10-data-model.md`.
2. Include only client-provided fields (exclude server-generated: `id`, `created_at`, `updated_at`, `campus_id`).
3. Mark required fields explicitly.
4. Use snake_case for all field names.
5. For offline-creatable entities (person, triage, attendance), include `sync_id` (client-generated UUID).
6. Validate field types match PostgreSQL column types:
   - `timestamptz` → ISO 8601 string
   - `uuid` → string (UUID format)
   - `boolean` → boolean
   - `integer` → number
   - `text` / `varchar` → string with max length

**Request schema format:**
```json
{
  "full_name": "string (required, max 200)",
  "birth_date": "string (optional, ISO 8601 date)",
  "document_type": "string (required, enum: CPF|SSN|EU_ID|PASSPORT|OTHER)",
  "document_number": "string (optional, max 30)",
  "sync_id": "string (optional, UUID for offline creation)"
}
```

### Step 4: Design Response Schemas

**Single resource response:**
```json
{
  "id": "uuid",
  "full_name": "string",
  "birth_date": "string|null",
  "campus_id": "uuid",
  "is_active": true,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Paginated list response (mandatory for all list endpoints):**
```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 150,
    "total_pages": 8
  }
}
```

**Error response (standard format):**
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Human-readable description",
    "details": [
      {
        "field": "document_number",
        "message": "Required when document_type is CPF"
      }
    ]
  }
}
```

Rules:
- All field names use snake_case.
- Nullable fields return `null`, not omitted.
- Timestamps use ISO 8601 with timezone (UTC).
- IDs are always UUID strings.
- No PII in error messages or error details.

### Step 5: Define Status Codes

Map each endpoint to its status codes:

| Scenario | Code | When |
|----------|------|------|
| Successful creation | 201 | POST that creates a resource |
| Successful retrieval | 200 | GET, PUT, PATCH |
| No content | 204 | DELETE (if used) |
| Validation error | 400 | Invalid request body, missing required fields |
| Unauthorized | 401 | Missing or invalid Keycloak token |
| Forbidden | 403 | Valid token but insufficient role |
| Not found | 404 | Resource does not exist or is in another campus |
| Conflict | 409 | Duplicate resource (e.g., duplicate document number) |
| Internal error | 500 | Unexpected server error (never expose internals) |

### Step 6: Assign RBAC Roles

For each endpoint, assign minimum required role from the hierarchy:
`ADMIN > COORDINATOR > PROFESSIONAL > SECRETARY > VOLUNTEER`

Guidelines from `docs/16-iam-and-access-control.md`:
- **Read operations**: Generally accessible to all authenticated roles (Volunteer+).
- **Create operations**: Depends on entity sensitivity (Person: Volunteer+, Attendance: Secretary+).
- **Update operations**: Generally Secretary+ or Professional+.
- **Admin operations**: User management, reports export = Coordinator+ or Admin only.
- **State transitions**: Depend on the transition type (triage resolution: Professional+).

Document as a roles array per endpoint: `["SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN"]`.

### Step 7: Validate Campus Scoping

For every endpoint, verify:
- `campus_id` is extracted from JWT claims, NOT from request body or query parameters.
- List endpoints filter by campus_id from JWT.
- Detail endpoints verify the resource belongs to the user's campus.
- Create endpoints set campus_id from JWT, not client input.
- No cross-campus data access (exception: Admin with explicit cross-campus permission).

### Step 8: Document Query Parameters

For list endpoints, define supported query parameters:
- `page` (integer, default: 1) — pagination page number.
- `per_page` (integer, default: 20, max: 100) — items per page.
- `sort_by` (string, default varies) — field to sort by.
- `sort_order` (string, default: "asc") — "asc" or "desc".
- `search` or `q` (string, optional) — text search across relevant fields.
- Entity-specific filters (e.g., `is_active`, `role`, `service_type`).

### Step 9: Write to API Design Document

Output the complete endpoint specification in the format used by `docs/11-api-design.md`:

For each endpoint:
```markdown
### [Operation Name]

**Endpoint**: `METHOD /api/v1/path`
**Roles**: [ROLE1, ROLE2, ...]
**Description**: Brief description of what this endpoint does.

**Request Body** (if applicable):
| Field | Type | Required | Validation |
|-------|------|----------|------------|
| field_name | string | Yes | max 200 |

**Query Parameters** (if applicable):
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | integer | 1 | Page number |

**Response** (status code):
```json
{ ... }
```

**Error Codes**:
| Code | Status | Description |
|------|--------|-------------|
| VALIDATION_ERROR | 400 | Invalid input |
```

## Outputs / Deliverables

1. **Complete endpoint specifications** for all operations on the resource.
2. **Request schemas** with field types, requirements, and validation rules.
3. **Response schemas** with exact JSON structure.
4. **Status code mapping** for all scenarios.
5. **RBAC role assignments** per endpoint.
6. **Query parameters** for list and search endpoints.
7. **Updates to `docs/11-api-design.md`** with the new endpoints.

## References

| Document | Path | Usage |
|----------|------|-------|
| Domain model | `docs/04-domain-model.md` | Entity definitions |
| API design | `docs/11-api-design.md` | Existing conventions and endpoints |
| Data model | `docs/10-data-model.md` | Table DDL for field types |
| IAM | `docs/16-iam-and-access-control.md` | RBAC roles and hierarchy |
| Requirements | `docs/03-requirements-catalog.md` | Functional requirements for the feature |

## Constraints / Quality Bar

- All endpoints must be versioned under `/api/v1/`.
- All field names must use snake_case.
- All list endpoints must use the standard pagination wrapper.
- All error responses must use the standard error format.
- All endpoints must have RBAC role assignments.
- All endpoints must enforce campus_id scoping from JWT.
- No PII may appear in error messages or error details.
- No undocumented endpoints — everything must be specified before implementation.
- Request schemas must not accept `id`, `campus_id`, `created_at`, `updated_at` from clients.

## Interaction with Other Artifacts

- **Invoked by agents**: tech-lead (design phase), backend-engineer (pre-implementation).
- **Depends on skills**: analyze-requirements (provides story analysis).
- **Feeds into skills**: design-backend-feature (consumes endpoint specs), review-api-contract (validates implementation).
- **Triggers hooks**: pre-api-change (when modifying existing endpoints).
- **Governed by rules**: documentation-first (API must be documented before implementation), api-versioning-strategy (breaking changes).
