# 11 - API Design

## Base URL

```
/api/v1
```

All endpoints are versioned under `/api/v1`. Future breaking changes will use `/api/v2`.

---

## Authentication

### Overview

Authentication is handled externally by **Keycloak** using the **OpenID Connect (OIDC)** protocol. The Chesed API does not implement login, token issuance, or credential management. Instead, it validates access tokens issued by Keycloak.

- **Protocol**: Authorization Code Flow with PKCE (for the React SPA)
- **Keycloak realm URL**: Configurable via `KEYCLOAK_REALM_URL` environment variable
- **Token validation**: The Go API validates the `access_token` signature using Keycloak's JWKS endpoint (`{KEYCLOAK_REALM_URL}/protocol/openid-connect/certs`)
- **Header**: `Authorization: Bearer <keycloak_access_token>`

### Expected Access Token Claims

The API expects the following claims in the Keycloak-issued JWT access token:

```json
{
  "sub": "<keycloak-user-id>",
  "email": "<email>",
  "realm_access": { "roles": ["COORDINATOR"] },
  "campus_id": "<uuid>",
  "person_id": "<uuid>",
  "preferred_username": "<email>",
  "iss": "https://keycloak.example.com/realms/chesed",
  "aud": "chesed-api"
}
```

Custom claims (`campus_id`, `person_id`) are mapped via Keycloak protocol mappers from user attributes.

### Authentication Flows (Handled by Keycloak)

The following authentication flows are **not** implemented as API endpoints. They are handled entirely by Keycloak and the frontend `keycloak-js` SDK:

| Flow | Mechanism |
|------|-----------|
| **Login** | React SPA redirects to Keycloak login page via `keycloak-js` SDK |
| **Token refresh** | Handled by `keycloak-js` SDK (silent refresh via hidden iframe) |
| **Password reset** | Keycloak built-in "Forgot Password" flow (configured in realm settings) |
| **Logout** | Keycloak end-session endpoint (front-channel logout); React calls `keycloak.logout()` |

No `/auth/*` endpoints exist in the Chesed API.

---

## Person Endpoints

| Method | Path | Auth | Roles | Description |
|--------|------|------|-------|-------------|
| POST | `/persons` | Yes | All | Create person |
| GET | `/persons` | Yes | All | List/search persons |
| GET | `/persons/:id` | Yes | All | Get person detail |
| PUT | `/persons/:id` | Yes | Secretary+ | Update person |
| GET | `/persons/:id/history` | Yes | Professional+ | Get person history |
| POST | `/persons/:id/roles` | Yes | Coordinator+ | Add role |
| PATCH | `/persons/:id/roles/:roleId` | Yes | Coordinator+ | Activate/deactivate role |
| GET | `/persons/check-duplicate` | Yes | All | Check for duplicates |
| GET | `/persons/:id/agreement` | Yes | Coordinator+ | Get person's volunteer agreements |
| POST | `/persons/:id/agreement/upload` | Yes | Coordinator+ | Upload signed agreement document (multipart) |
| GET | `/persons/:id/agreement/document` | Yes | Coordinator+ | Download uploaded agreement document |

#### Duplicate Detection
```
GET /api/v1/persons/check-duplicate?document_number=123.456.789-00&document_type=CPF

Response 200:
{
  "has_duplicates": true,
  "matches": [
    {
      "id": "uuid",
      "full_name": "Maria Santos",
      "document_number": "123.456.789-00",
      "campus": "Sao Paulo",
      "match_type": "exact_document"
    }
  ]
}

Response 200 (no duplicates):
{
  "has_duplicates": false,
  "matches": []
}
```
Matching algorithm: exact match on `document_number` + `document_type`. Fuzzy name matching deferred to Phase 2.

#### POST /persons
```json
// Request
{
  "full_name": "João Carlos da Silva",
  "birth_date": "1985-03-15",
  "document_type": "CPF",
  "document_number": "123.456.789-00",
  "gender": "M",
  "email": "joao@email.com",
  "phone": "+55 11 99999-0000",
  "referral_source": "Campaign March 2026",
  "address": {
    "street": "Rua das Flores",
    "number": "123",
    "neighborhood": "Centro",
    "city": "São Paulo",
    "state": "SP",
    "zip_code": "01000-000",
    "country": "BRA"
  },
  "sync_id": "uuid (optional, for offline-created records)"
}

// Response 201
{
  "id": "uuid",
  "full_name": "João Carlos da Silva",
  "document_type": "CPF",
  "document_number": "123.456.789-00",
  "campus_id": "uuid",
  "created_at": "2026-04-02T10:30:00Z"
}
```

#### GET /persons?q=joão&page=1&per_page=20&agreement_status=with_agreement

**Additional query parameters:**
- `agreement_status` — Filter by volunteer agreement status. Values: `with_agreement` (accepted), `without_agreement` (pending or no agreement), `rejected`.


```json
// Response 200
{
  "data": [
    {
      "id": "uuid",
      "full_name": "João Carlos da Silva",
      "document_number": "123.456.789-00",
      "phone": "+55 11 99999-0000",
      "roles": ["ASSISTED", "VOLUNTEER"],
      "is_active": true
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 42,
    "total_pages": 3
  }
}
```

#### GET /persons/:id/history
```json
// Response 200
{
  "person": { "id": "uuid", "full_name": "..." },
  "history": [
    {
      "type": "triage",
      "id": "uuid",
      "date": "2026-03-15T14:00:00Z",
      "summary": "Initial assessment - Legal aid requested",
      "campaign": "March Social Action"
    },
    {
      "type": "attendance",
      "id": "uuid",
      "date": "2026-03-15T15:30:00Z",
      "summary": "Legal consultation - Completed",
      "service_type": "Legal",
      "professional": "Dr. Ana Santos",
      "status": "COMPLETED"
    }
  ]
}
```

---

## Onboarding Endpoints

| Method | Path | Auth | Roles | Description |
|--------|------|------|-------|-------------|
| GET | `/auth/me` | Yes | All | Returns onboarding status for authenticated user |

#### GET /auth/me

Returns the onboarding state of the authenticated user. Auto-links pre-created persons by email when applicable.

Middleware: `OIDCAuth` + `AutoProvision` (no agreement requirement).

```json
// Response 200
{
  "person_id": "uuid-or-null",
  "needs_profile_completion": false,
  "needs_agreement": false,
  "roles": ["VOLUNTEER"]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `person_id` | `uuid \| null` | Person ID if linked (may be auto-linked by email on this call) |
| `needs_profile_completion` | `boolean` | `true` if user has no linked person — must complete `/complete-profile` |
| `campus_id` | `uuid \| null` | Campus ID resolved from backend (app_user or person) |
| `needs_campus_assignment` | `boolean` | `true` if user has no campus — must select during profile completion |
| `needs_agreement` | `boolean` | `true` if user has active VOLUNTEER role but no accepted agreement |
| `roles` | `string[]` | Keycloak realm roles from token |

---

## Campus Endpoints

| Method | Path | Auth | Roles | Description |
|--------|------|------|-------|-------------|
| GET | `/campuses` | Yes | All | Returns active campuses (for onboarding selector) |
| GET | `/campuses/all` | Yes | ADMIN | Returns all campuses including inactive |
| GET | `/campuses/{id}` | Yes | ADMIN | Returns campus by ID |
| POST | `/campuses` | Yes | ADMIN | Creates a new campus |
| PUT | `/campuses/{id}` | Yes | ADMIN | Updates an existing campus |

#### GET /campuses
```json
// Response 200
{
  "data": [
    {
      "id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
      "name": "São Paulo",
      "region": "BRAZIL",
      "city": "São Paulo",
      "state": "SP",
      "country": "BRA",
      "is_active": true,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

#### POST /campuses
```json
// Request
{
  "name": "New York",
  "region": "USA",
  "city": "New York",
  "state": "NY",
  "country": "USA"
}

// Response 201
{ "id": "uuid", "name": "New York", "region": "USA", ... }
```

---

## Volunteer Agreement Endpoints

| Method | Path | Auth | Roles | Description |
|--------|------|------|-------|-------------|
| GET | `/volunteer-agreement/text` | Yes | All | Returns current agreement text and version |
| POST | `/volunteer-agreement/accept` | Yes | All | Digital acceptance of agreement (self-service) |
| POST | `/volunteer-agreement/reject` | Yes | All | Rejection with optional reason |

#### GET /volunteer-agreement/text
```json
// Response 200
{
  "version": "1.0",
  "title": "Termo de Adesão ao Voluntariado",
  "content": "Full agreement text in HTML or markdown...",
  "effective_date": "2026-01-01"
}
```

#### POST /volunteer-agreement/accept
```json
// Request (no body required — person identified from JWT claims)

// Response 200
{
  "id": "uuid",
  "person_id": "uuid",
  "status": "ACCEPTED",
  "signature_method": "DIGITAL",
  "agreement_version": "1.0",
  "accepted_at": "2026-04-10T14:30:00Z"
}
```

#### POST /volunteer-agreement/reject
```json
// Request
{
  "reason": "I do not agree with the terms (optional)"
}

// Response 200
{
  "id": "uuid",
  "person_id": "uuid",
  "status": "REJECTED",
  "rejected_at": "2026-04-10T14:30:00Z"
}
```

#### GET /persons/:id/agreement
```json
// Response 200 (Coordinator+ only)
{
  "person_id": "uuid",
  "agreements": [
    {
      "id": "uuid",
      "person_role_id": "uuid",
      "role_type": "VOLUNTEER",
      "status": "ACCEPTED",
      "signature_method": "DIGITAL",
      "agreement_version": "1.0",
      "accepted_at": "2026-04-10T14:30:00Z",
      "has_document": false
    }
  ]
}
```

#### POST /persons/:id/agreement/upload
```
Content-Type: multipart/form-data (Coordinator+ only)

Fields:
- file: PDF or image file of the signed agreement
- notes: Optional text notes

Response 200:
{
  "id": "uuid",
  "status": "ACCEPTED",
  "signature_method": "MANUAL_UPLOAD",
  "document_path": "/agreements/uuid/document.pdf",
  "uploaded_at": "2026-04-10T14:30:00Z"
}
```

#### GET /persons/:id/agreement/document
```
Response 200 (Coordinator+ only):
Content-Type: application/pdf (or image/*)
Content-Disposition: attachment; filename="agreement-<person_name>.pdf"

Binary file content
```

---

## Service Type Endpoints

| Method | Path | Auth | Roles | Description |
|--------|------|------|-------|-------------|
| GET | `/service-types` | Yes | All | List available service types |

#### Service Types
```
GET /api/v1/service-types

Response 200:
{
  "data": [
    { "id": "uuid", "name": "Legal Aid", "category": "LEGAL", "is_active": true },
    { "id": "uuid", "name": "Medical Consultation", "category": "MEDICAL", "is_active": true }
  ]
}
```
Available to all authenticated users. Read-only in Phase 1.

---

## Triage Endpoints

| Method | Path | Auth | Roles | Description |
|--------|------|------|-------|-------------|
| POST | `/triages` | Yes | All | Create triage |
| GET | `/triages` | Yes | All | List triages |
| GET | `/triages/:id` | Yes | All | Get triage detail |

#### POST /triages
```json
// Request
{
  "person_id": "uuid",
  "campaign_id": "uuid (optional)",
  "main_complaint": "Needs legal assistance for housing issue",
  "requested_service_type_ids": ["uuid", "uuid"],
  "location": "Community Center - Jabaquara",
  "notes": "Referred by neighbor",
  "sync_id": "uuid (optional)"
}

// Response 201
{
  "id": "uuid",
  "person_id": "uuid",
  "triage_date": "2026-04-02T10:30:00Z",
  "triaged_by": "uuid"
}
```

---

## Attendance Endpoints

| Method | Path | Auth | Roles | Description |
|--------|------|------|-------|-------------|
| POST | `/attendances` | Yes | Secretary+ | Create attendance |
| GET | `/attendances` | Yes | All | List attendances |
| GET | `/attendances/:id` | Yes | All | Get attendance detail |
| PATCH | `/attendances/:id` | Yes | Professional+ | Update attendance |
| PATCH | `/attendances/:id/transition` | Yes | Professional+ | Change workflow status |

#### POST /attendances
```json
// Request
{
  "person_id": "uuid",
  "triage_id": "uuid (optional)",
  "campaign_id": "uuid (optional)",
  "service_type_id": "uuid",
  "professional_id": "uuid",
  "attendance_date": "2026-04-02T15:00:00Z",
  "observations": "Initial consultation",
  "sync_id": "uuid (optional)"
}

// Response 201
{
  "id": "uuid",
  "status": "SCHEDULED",
  "created_at": "2026-04-02T10:30:00Z"
}
```

#### PATCH /attendances/:id/transition
```json
// Request
{
  "to_status": "IN_PROGRESS",
  "reason": "Starting consultation"
}

// Response 200
{
  "id": "uuid",
  "status": "IN_PROGRESS",
  "transition": {
    "from_status": "SCHEDULED",
    "to_status": "IN_PROGRESS",
    "transitioned_at": "2026-04-02T15:05:00Z"
  }
}
```

**Allowed transitions:**
```
SCHEDULED → IN_PROGRESS
SCHEDULED → CANCELLED
IN_PROGRESS → COMPLETED
IN_PROGRESS → FOLLOW_UP
IN_PROGRESS → CANCELLED
FOLLOW_UP → IN_PROGRESS
FOLLOW_UP → COMPLETED
COMPLETED → FOLLOW_UP (reopen)
```

---

## Campaign Endpoints (Phase 2)

**(Phase 2)** — These endpoints are implemented in Phase 2.

| Method | Path | Auth | Roles | Description |
|--------|------|------|-------|-------------|
| POST | `/campaigns` | Yes | Coordinator+ | Create campaign |
| GET | `/campaigns` | Yes | All | List campaigns |
| GET | `/campaigns/:id` | Yes | All | Get campaign detail |
| PUT | `/campaigns/:id` | Yes | Coordinator+ | Update campaign |
| POST | `/campaigns/:id/team` | Yes | Coordinator+ | Add team member |
| DELETE | `/campaigns/:id/team/:personId` | Yes | Coordinator+ | Remove team member |

---

## Donation Endpoints (Phase 2)

**(Phase 2)** — These endpoints are implemented in Phase 2.

| Method | Path | Auth | Roles | Description |
|--------|------|------|-------|-------------|
| POST | `/donations` | Yes | Secretary+ | Create donation |
| GET | `/donations` | Yes | Coordinator+ | List donations |
| GET | `/donations/:id` | Yes | Coordinator+ | Get donation detail |
| GET | `/donations/:id/receipt` | Yes | Coordinator+ | Download receipt PDF |

---

## Consent Endpoints (Phase 2)

**(Phase 2)** — These endpoints are implemented in Phase 2.

| Method | Path | Auth | Roles | Description |
|--------|------|------|-------|-------------|
| POST | `/consents` | Yes | All | Create consent record |
| GET | `/persons/:id/consents` | Yes | Secretary+ | List person's consents |
| PATCH | `/consents/:id/revoke` | Yes | Admin | Revoke consent |

---

## Document Endpoints (Phase 2)

**(Phase 2)** — These endpoints are implemented in Phase 2.

| Method | Path | Auth | Roles | Description |
|--------|------|------|-------|-------------|
| POST | `/persons/:id/documents` | Yes | Secretary+ | Upload document |
| GET | `/persons/:id/documents` | Yes | Professional+ | List person's documents |
| GET | `/documents/:id/download` | Yes | Professional+ | Get presigned download URL |

---

## Sync Endpoints

| Method | Path | Auth | Roles | Description | Status |
|--------|------|------|-------|-------------|--------|
| POST | `/sync/push` | Yes | All | Upload offline-created records | **Phase 1 (Sprint 4)** |
| GET | `/sync/pull` | Yes | All | Fetch records updated since timestamp | **Phase 1 (Sprint 4)** |
| GET | `/sync/status` | Yes | All | Get sync health and pending count | Phase 2 |

Push is idempotent by `sync_id` — re-pushing the same `sync_id` returns the
existing server record without re-creating it. Per-record errors do not abort
the batch; the response contains a per-record `results` array. Batch-level
errors (oversize > 50 records, missing campus context) reject the entire request.

Pull is a delta query bounded by an internal page size (default 100 records
per entity). `has_more = true` signals there are more records past
`next_since`; the client should re-issue the pull using `next_since` as the
new cursor.

#### POST /sync/push

Request body fields:
- `device_id` (uuid, optional) — informational; not used for server logic in MVP.
- `records[]` — up to 50 entries. Larger batches return `413 batch_too_large`.
  - `entity_type` (string, required) — one of `person`, `triage`, `attendance`.
  - `sync_id` (uuid, required) — client-generated UUIDv4; idempotency key.
  - `data` (object, required) — entity payload matching that entity's create input.
  - `created_at` (RFC3339, optional) — local creation timestamp; informational.

```json
// Request
{
  "device_id": "uuid",
  "records": [
    {
      "entity_type": "person",
      "sync_id": "uuid",
      "data": { "full_name": "Maria", "document_type": "CPF", "nationality": "BRA" },
      "created_at": "2026-04-02T10:30:00Z"
    },
    {
      "entity_type": "triage",
      "sync_id": "uuid",
      "data": { "person_id": "uuid", "main_complaint": "Dor de cabeça" }
    }
  ]
}

// Response 200
{
  "results": [
    { "sync_id": "uuid", "status": "created", "server_id": "uuid" },
    { "sync_id": "uuid", "status": "conflict", "message": "duplicate" },
    { "sync_id": "uuid", "status": "error",   "message": "invalid person_id: ..." }
  ],
  "server_timestamp": "2026-04-02T10:40:00Z"
}
```

Per-record `status`:
- `created` — new server record (or idempotent return of the existing one).
- `conflict` — DB constraint blocked the write (e.g., duplicate document).
- `error` — payload validation or DB error specific to the record.

Error responses:
| HTTP | Error | When |
|------|-------|------|
| 400 | `invalid_request` | Malformed JSON body |
| 400 | `invalid_entity_types` | A record references an unknown `entity_type` |
| 400 | `invalid_request` | Missing `sync_id` or empty `data` |
| 403 | `forbidden` | Auth token lacks resolvable `campus_id` |
| 413 | `batch_too_large` | `records.length > 50` |

#### GET /sync/pull?since=&entity_types=

Query parameters:
- `since` (RFC3339, required) — return records with `updated_at > since`.
- `entity_types` (CSV, optional) — subset of `person,triage,attendance`. Defaults to all three. Duplicate or unknown values return `400 invalid_entity_types`.

```json
// Response 200
{
  "records": [
    {
      "entity_type": "person",
      "id": "uuid",
      "data": { /* full person fields */ },
      "updated_at": "2026-04-02T10:30:00Z"
    }
  ],
  "server_timestamp": "2026-04-02T10:40:00Z",
  "has_more": false
}
```

When `has_more = true`, the response includes a `next_since` timestamp:

```json
{
  "records": [ /* exactly 100 records */ ],
  "server_timestamp": "2026-04-02T10:40:00Z",
  "has_more": true,
  "next_since": "2026-04-02T10:39:50Z"
}
```

Error responses:
| HTTP | Error | When |
|------|-------|------|
| 400 | `missing_since` | `since` query param not provided |
| 400 | `invalid_since` | `since` is not a valid RFC3339 timestamp |
| 400 | `invalid_entity_types` | Unknown or duplicate entity type |
| 403 | `forbidden` | Auth token lacks resolvable `campus_id` |

---

## Report Endpoints

| Method | Path | Auth | Roles | Description | Status |
|--------|------|------|-------|-------------|--------|
| GET | `/reports/attendances` | Yes | Coordinator+ | Attendance summary | **Phase 1 (Sprint 4)** |
| GET | `/reports/attendances/export` | Yes | Coordinator+ | CSV export | **Phase 1 (Sprint 4)** |
| GET | `/reports/campaigns/:id` | Yes | Coordinator+ | Campaign metrics | Phase 2 |
| GET | `/reports/dashboard` | Yes | Coordinator+ | Dashboard KPIs | Phase 2 |

#### GET /reports/attendances?start=2026-01-01&end=2026-03-31

Both `start` and `end` are required `YYYY-MM-DD` dates. The end day is inclusive
(server interprets as `< end + 1 day`). Range may not exceed 366 days. The query
is automatically campus-scoped from the caller's token. `FOLLOW_UP` is reserved
for Phase 2 and will not appear in `by_status` until then.

```json
// Response 200
{
  "period": { "start": "2026-01-01", "end": "2026-03-31" },
  "total_attendances": 234,
  "unique_persons": 187,
  "by_status": {
    "COMPLETED": 198,
    "SCHEDULED": 10,
    "IN_PROGRESS": 20,
    "CANCELLED": 6
  },
  "by_service_type": [
    { "service_type": "LEGAL", "count": 45 },
    { "service_type": "MEDICAL", "count": 78 }
  ],
  "by_month": [
    { "month": "2026-01", "count": 72 },
    { "month": "2026-02", "count": 85 },
    { "month": "2026-03", "count": 77 }
  ]
}
```

Error codes: `invalid_range` (missing/inverted/oversize), `invalid_start` /
`invalid_end` (malformed date), `forbidden` (no campus in token),
`range_too_large` (>366 days).

#### GET /reports/attendances/export?start=2026-01-01&end=2026-03-31&format=csv

Streams `Content-Type: text/csv; charset=utf-8` with one row per attendance.
`format` defaults to `csv` if omitted; other values return `400 invalid_format`.
Response sets `Content-Disposition: attachment; filename="attendances_<start>_<end>.csv"`.

CSV columns (header included on the first line):

```
attendance_id,attendance_date,person_name,person_document,service_type,status,professional_name,created_at
```

Dates are emitted as RFC3339 UTC. `person_document` and `professional_name`
are empty strings when not set.

---

## User Management Endpoints

| Method | Path | Auth | Roles | Description |
|--------|------|------|-------|-------------|
| POST | `/users` | Yes | Admin | Create user in Keycloak + local app_user |
| GET | `/users` | Yes | Admin | List users |
| PATCH | `/users/:id` | Yes | Admin | Update user profile/role |
| PATCH | `/users/:id/deactivate` | Yes | Admin | Deactivate user in Keycloak + local app_user |
| GET | `/users/me` | Yes | All | Get current user profile |

**Keycloak Admin API Integration**:
- `POST /users` creates the user in Keycloak first (via Keycloak Admin REST API), then stores the local `app_user` record with the returned `keycloak_subject_id`. If Keycloak user creation fails, no local record is created.
- `PATCH /users/:id/deactivate` disables the user in Keycloak (sets `enabled: false` via Admin API) and sets `is_active = FALSE` locally.
- The API authenticates to Keycloak Admin API using a **service account** (client credentials grant) configured via `KEYCLOAK_ADMIN_CLIENT_ID` and `KEYCLOAK_ADMIN_CLIENT_SECRET` environment variables.

---

## Audit Endpoints

| Method | Path | Auth | Roles | Description |
|--------|------|------|-------|-------------|
| GET | `/audit/logs` | Yes | Admin | Query audit logs |

#### GET /audit/logs?user_id=uuid&entity_type=person&start=2026-01-01&end=2026-03-31&page=1
```json
{
  "data": [
    {
      "id": "uuid",
      "user_email": "maria@example.com",
      "action_type": "UPDATE",
      "entity_type": "person",
      "entity_id": "uuid",
      "description": "Updated phone number",
      "old_values": { "phone": "+55 11 88888-0000" },
      "new_values": { "phone": "+55 11 99999-0000" },
      "ip_address": "192.168.1.1",
      "timestamp": "2026-04-02T10:30:00Z"
    }
  ],
  "pagination": { "page": 1, "per_page": 50, "total": 1234 }
}
```

---

## Common Patterns

### Pagination
All list endpoints support `?page=1&per_page=20`. Default: page=1, per_page=20. Max per_page=100.

### Filtering
Filter parameters are query strings: `?status=COMPLETED&service_type_id=uuid&start=2026-01-01`

### Sorting
`?sort=created_at&order=desc`. Default sort varies by endpoint.

### Error Responses
```json
// 400 Bad Request
{ "error": "validation_error", "details": [{ "field": "email", "message": "Invalid email format" }] }

// 401 Unauthorized
{ "error": "unauthorized", "message": "Invalid or expired token" }

// 403 Forbidden
{ "error": "forbidden", "message": "Insufficient permissions" }

// 404 Not Found
{ "error": "not_found", "message": "Person not found" }

// 409 Conflict
{ "error": "duplicate", "message": "Person with this document already exists", "existing_id": "uuid" }
```

### Campus Scoping
All data queries are automatically filtered by the user's `campus_id` from the Keycloak access token claims. Admin users can add `?campus_id=uuid` to query other campuses.
