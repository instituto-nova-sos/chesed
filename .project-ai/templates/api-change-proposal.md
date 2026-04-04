# API Change Proposal: [Brief Description]

## Metadata

| Field | Value |
|-------|-------|
| **Endpoint** | [METHOD] /api/v1/[path] |
| **Story Reference** | S0x.x — [Story title from docs/09-backlog.md] |
| **Author** | [Name] |
| **Date** | [YYYY-MM-DD] |
| **Status** | Proposed / Approved / Implemented / Rejected |

---

## Current Contract

From `docs/11-api-design.md`:

**Request**:
```
[METHOD] /api/v1/[path]
Authorization: Bearer <keycloak_access_token>
Content-Type: application/json
```

```json
{
  "current_field_1": "value",
  "current_field_2": "value"
}
```

**Response** ([status code]):
```json
{
  "current_response_field_1": "value",
  "current_response_field_2": "value"
}
```

If this is a new endpoint, write: "New endpoint. No current contract exists."

---

## Proposed Change

Describe what is changing and why. Be specific:

- [ ] New endpoint (not previously documented)
- [ ] New request field(s)
- [ ] Changed request field(s) (type change, constraint change)
- [ ] Removed request field(s)
- [ ] New response field(s)
- [ ] Changed response field(s)
- [ ] Changed HTTP method or path
- [ ] Changed role requirements
- [ ] Changed error codes

**Motivation**: [Why this change is needed. Reference the story or requirement.]

---

## Request Schema

```json
{
  "field_name": "type — description (required/optional)",
  "full_name": "string — Person's full legal name (required, max 200 chars)",
  "birth_date": "string — ISO 8601 date, e.g., '1985-03-15' (optional)",
  "document_type": "string — One of: CPF, SSN, EU_ID, PASSPORT, OTHER (required)",
  "document_number": "string — Document number, max 30 chars (optional)",
  "sync_id": "string — UUID for offline-created records (optional)"
}
```

Validation rules:
- [Field]: [Validation rule, e.g., "required, max 200 characters"]
- [Field]: [Validation rule, e.g., "must be valid UUID"]
- [Field]: [Validation rule, e.g., "must be one of: CPF, SSN, EU_ID, PASSPORT, OTHER"]

---

## Response Schema

### Success Response

**Status**: [201 Created / 200 OK / 204 No Content]

```json
{
  "id": "uuid",
  "field_1": "value",
  "field_2": "value",
  "campus_id": "uuid",
  "created_at": "2026-04-02T10:30:00Z"
}
```

For list endpoints, use the standard pagination wrapper:

```json
{
  "data": [
    { "id": "uuid", "field_1": "value" }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 42,
    "total_pages": 3
  }
}
```

### Error Responses

| Status | Code | When |
|--------|------|------|
| 400 | `VALIDATION_ERROR` | Request body fails validation |
| 401 | `UNAUTHORIZED` | Missing or invalid Keycloak token |
| 403 | `FORBIDDEN` | User lacks required role |
| 404 | `NOT_FOUND` | Resource does not exist or belongs to different campus |
| 409 | `CONFLICT` | Duplicate record (e.g., duplicate sync_id or document) |
| 422 | `BUSINESS_RULE_VIOLATION` | Business logic rejection (e.g., invalid state transition) |
| 500 | `INTERNAL_ERROR` | Unexpected server error |

Error response format:
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Human-readable message (no PII)"
  }
}
```

---

## RBAC

| Role | Access |
|------|--------|
| VOLUNTEER | [Allowed / Denied] |
| SECRETARY | [Allowed / Denied] |
| PROFESSIONAL | [Allowed / Denied] |
| COORDINATOR | [Allowed / Denied] |
| ADMIN | [Allowed / Denied] |

Middleware: `middleware.RequireRole([list of allowed roles])`

Reference: `docs/16-iam-and-access-control.md` permission matrix.

---

## Breaking Change Assessment

| Question | Answer |
|----------|--------|
| Is this a breaking change? | Yes / No |
| Does it remove or rename fields? | Yes / No |
| Does it change field types? | Yes / No |
| Does it change required/optional status? | Yes / No |
| Does it change the URL path? | Yes / No |

If **yes** to any of the above:

**Migration plan**:
1. [Step 1: e.g., "Deploy backend with both old and new field names accepted"]
2. [Step 2: e.g., "Update frontend to use new field names"]
3. [Step 3: e.g., "Remove old field name support in next sprint"]

**Affected clients**:
- [ ] React frontend
- [ ] Mobile offline sync queue (existing queued records may use old format)
- [ ] External integrations (if any)

---

## Offline Impact

| Question | Answer |
|----------|--------|
| Is this endpoint used in sync push? | Yes / No |
| Is this endpoint used in sync pull? | Yes / No |
| Does the change affect IndexedDB schema? | Yes / No |
| Does the change affect sync queue format? | Yes / No |

If yes:

**Sync push impact**: [Describe how `POST /api/v1/sync/push` payload changes for this entity type]

**Sync pull impact**: [Describe how `GET /api/v1/sync/pull` response changes for this entity type]

**IndexedDB migration**: [Describe Dexie version upgrade needed, if any]

**Backward compatibility**: [Can existing offline records sync after this change? If not, describe the migration path]

---

## Implementation Notes

[Any additional technical notes, edge cases, or implementation guidance.]

### Backend changes
- [ ] Domain struct: `backend/internal/domain/[file]`
- [ ] Repository: `backend/internal/repository/[file]`
- [ ] Service: `backend/internal/service/[file]`
- [ ] Handler: `backend/internal/handler/[file]`
- [ ] Migration (if schema change): `backend/migrations/[file]`

### Frontend changes
- [ ] Types: `frontend/src/types/[file]`
- [ ] API client: `frontend/src/api/[file]`
- [ ] Offline schema (if applicable): `frontend/src/offline/db.ts`

### Documentation updates
- [ ] `docs/11-api-design.md` — Update endpoint contract
- [ ] `docs/10-data-model.md` — Update if schema changed
