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
Matching algorithm: exact match on `document_number` + `document_type`, **scoped to the caller's campus** (a duplicate in another campus is never returned; CLAUDE.md rule #4, threat model T3). Because matches are always same-campus, the campus is implicit and not included in the response. Fuzzy name matching deferred to Phase 2.

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

`document_type` accepts `CPF`, `RG`, `SSN`, `EU_ID`, `PASSPORT`, `OTHER`, with per-type format validation on `document_number`: CPF checksum, SSN pattern, and length + charset for RG, EU_ID, and PASSPORT. A value that fails its type's format check returns `400 validation_error`.

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
  "campus_timezone": "America/Sao_Paulo",
  "roles": ["VOLUNTEER"]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `person_id` | `uuid \| null` | Person ID if linked (may be auto-linked by email on this call) |
| `needs_profile_completion` | `boolean` | `true` if user has no linked person — must complete `/complete-profile` |
| `campus_id` | `uuid \| null` | Campus ID resolved from backend (app_user or person) |
| `campus_timezone` | `string \| null` | IANA timezone of the user's campus (optional; e.g. `America/Sao_Paulo`) |
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
      "timezone": "America/Sao_Paulo",
      "is_active": true,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

The `timezone` field (IANA string) is included in all campus responses (`GET /campuses`, `/campuses/all`, `/campuses/{id}`).

#### POST /campuses
```json
// Request
{
  "name": "New York",
  "region": "USA",
  "city": "New York",
  "state": "NY",
  "country": "USA",
  "timezone": "America/New_York"
}

// Response 201
{ "id": "uuid", "name": "New York", "region": "USA", "timezone": "America/New_York", ... }
```

`timezone` is optional on create and update (IANA string); it defaults to `America/Sao_Paulo` when omitted.

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

## Campaign Endpoints (Phase 2 — Sprint 5)

| Method | Path | Auth | Roles | Description |
|--------|------|------|-------|-------------|
| POST | `/campaigns` | Yes | Coordinator+ | Create campaign |
| GET | `/campaigns` | Yes | All | List campaigns |
| GET | `/campaigns/:id` | Yes | All | Get campaign detail (includes team) |
| PUT | `/campaigns/:id` | Yes | Coordinator+ | Update campaign |
| POST | `/campaigns/:id/team` | Yes | Coordinator+ | Add team member |
| DELETE | `/campaigns/:id/team/:personId` | Yes | Coordinator+ | Remove team member |

All campaign queries are campus-scoped from the caller's token. A campaign id
outside the caller's campus responds 404 (`not_found`) with no existence
disclosure.

#### POST /campaigns
```json
// Request
{
  "name": "March Social Action",
  "description": "Community outreach at Jabaquara",
  "campaign_type": "SOCIAL_ACTION",
  "start_date": "2026-07-10",
  "end_date": "2026-07-12",
  "location_name": "Community Center - Jabaquara",
  "location_address": "Rua Example, 123",
  "coordinator_id": "uuid (optional)"
}

// Response 201 (dates serialize as RFC3339 timestamps; the response is a
// superset of this example — it also carries campus_id and updated_at)
{
  "id": "uuid",
  "name": "March Social Action",
  "campaign_type": "SOCIAL_ACTION",
  "status": "PLANNED",
  "start_date": "2026-07-10T00:00:00Z",
  "end_date": "2026-07-12T00:00:00Z",
  "created_at": "2026-07-01T10:30:00Z"
}
```
`campaign_type` ∈ `SOCIAL_ACTION | EDUCATIONAL | HEALTH | COMMUNITY | OTHER`.
`end_date`, when present, must be ≥ `start_date`. `coordinator_id` must reference
a person in the caller's campus (400 `validation_error` otherwise, generic message).

#### GET /campaigns?status=ACTIVE&page=1&per_page=20
```json
// Response 200
{
  "data": [
    {
      "id": "uuid",
      "name": "March Social Action",
      "campaign_type": "SOCIAL_ACTION",
      "status": "ACTIVE",
      "start_date": "2026-07-10T00:00:00Z",
      "end_date": "2026-07-12T00:00:00Z",
      "location_name": "Community Center - Jabaquara"
    }
  ],
  "pagination": { "page": 1, "per_page": 20, "total": 3, "total_pages": 1 }
}
```
Optional filters: `status`, `campaign_type`.

#### GET /campaigns/:id
```json
// Response 200
{
  "id": "uuid",
  "name": "March Social Action",
  "description": "Community outreach at Jabaquara",
  "campaign_type": "SOCIAL_ACTION",
  "status": "ACTIVE",
  "start_date": "2026-07-10T00:00:00Z",
  "end_date": "2026-07-12T00:00:00Z",
  "location_name": "Community Center - Jabaquara",
  "location_address": "Rua Example, 123",
  "coordinator_id": "uuid",
  "created_at": "2026-07-01T10:30:00Z",
  "updated_at": "2026-07-01T10:30:00Z",
  "team": [
    {
      "person_id": "uuid",
      "person_name": "Maria Silva",
      "role_in_campaign": "VOLUNTEER",
      "assigned_at": "2026-07-01T11:00:00Z"
    }
  ]
}
```

#### PUT /campaigns/:id
Accepts the same body as POST plus `status`
(`PLANNED | ACTIVE | COMPLETED | CANCELLED`). Responds 200 with the updated
detail (without `team`). Invalid enum values or dates respond 400 `validation_error`.

#### POST /campaigns/:id/team
```json
// Request
{ "person_id": "uuid", "role_in_campaign": "VOLUNTEER" }

// Response 201
{
  "person_id": "uuid",
  "person_name": "Maria Silva",
  "role_in_campaign": "VOLUNTEER",
  "assigned_at": "2026-07-01T11:00:00Z"
}
```
`role_in_campaign` ∈ `COORDINATOR | PROFESSIONAL | VOLUNTEER | SUPPORT`.
Duplicate campaign+person responds 409 `duplicate`. A `person_id` not visible in
the caller's campus responds 400 `validation_error` (generic message).

#### DELETE /campaigns/:id/team/:personId
Responds 204 on success, 404 if the assignment does not exist.

Campaign error responses:
| HTTP | Error | When |
|------|-------|------|
| 400 | `invalid_request` | Malformed JSON body |
| 400 | `invalid_id` | Malformed campaign/person id in the path |
| 400 | `validation_error` | Missing/invalid fields, inverted dates, or a reference not resolvable in the caller's campus (generic — no cross-campus disclosure) |
| 400 | `invalid_status` / `invalid_campaign_type` | Unknown list filter value |
| 403 | `forbidden` | Auth token lacks resolvable `campus_id` |
| 404 | `not_found` | Campaign (or assignment) not visible in the caller's campus |
| 409 | `duplicate` | Person already assigned to the campaign |

---

## Public Endpoints (Phase 3 — Sprint 11, S12.1)

**Unauthenticated, internet-facing** surface consumed by the public WordPress site.
These routes are mounted under `/api/v1/public/…` **before** the authenticated route
groups and carry no `Authorization`. They are protected by, in order: a per-IP rate
limiter, a `campus_id` validator (the campus must exist and be active), and a
per-request campus transaction on the **non-owner** `chesed_app` connection whose
`app.current_campus` GUC is the validated `campus_id` — so PostgreSQL RLS enforces
campus isolation as a fail-closed safety net (a handler bug cannot cross campuses).

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/public/campaigns?campus_id=` | No | List a campus's `ACTIVE` campaigns (lean projection) |
| POST | `/public/volunteer-signup` | No | Register a prospective volunteer |

Rate limiting: a single per-IP limiter guards the whole `/public` group, budgeted at
`PUBLIC_RATE_LIMIT_RPM` requests/minute (default 60), keyed on the connection's
RemoteAddr (not a spoofable forwarded header). It is a coarse abuse backstop; the edge
proxy is expected to apply its own per-client limits. Excess requests receive `429` with
`Retry-After`.

CORS: only origins in the configured allowlist (`PUBLIC_CORS_ORIGINS`) receive an
`Access-Control-Allow-Origin`. The WordPress origin is provided via env, not hardcoded.

#### GET /public/campaigns?campus_id=&page=1&per_page=20
Returns only `ACTIVE` campaigns for the given campus, using the same lean projection as
the authenticated campaign list — **no** PII, coordinator, address, or team fields.
```json
// Response 200
{
  "data": [
    {
      "id": "uuid",
      "name": "March Social Action",
      "campaign_type": "SOCIAL_ACTION",
      "status": "ACTIVE",
      "start_date": "2026-07-10T00:00:00Z",
      "end_date": "2026-07-12T00:00:00Z",
      "location_name": "Community Center - Jabaquara"
    }
  ],
  "pagination": { "page": 1, "per_page": 20, "total": 3, "total_pages": 1 }
}
```

#### POST /public/volunteer-signup
Request body:
- `full_name` (string, required, ≤200).
- `email` (string, optional, valid email, ≤255).
- `phone` (string, optional, ≤30).
- `birth_date` (RFC3339 date, optional).
- `referral_source` (string, optional, ≤200).
- `campus_id` (uuid, required) — validated against an existing active campus.

Creates a `person`, a `VOLUNTEER` `person_role`, and a `PENDING` volunteer agreement in
that campus. Minimal PII by design (no document number for a public lead form). Writes an
`audit_log` entry with the campus set, the client IP and user-agent captured, and a null
actor (no authenticated subject).
```json
// Response 201
{ "id": "uuid", "full_name": "Maria", "campus_id": "uuid", "created_at": "2026-07-06T09:00:00Z" }
```

Public error responses:
| HTTP | Error | When |
|------|-------|------|
| 400 | `invalid_request` | Malformed JSON body or missing/invalid `campus_id` |
| 400 | `validation_error` | Missing `full_name` or an invalid optional field |
| 404 | `not_found` | `campus_id` does not match an existing active campus |
| 429 | `rate_limited` | Per-IP rate exceeded (includes `Retry-After`) |

---

## Donation Endpoints (Phase 2 — Sprint 7)

**(Phase 2)** — Implemented in Sprint 7 (E09), except the receipt PDF (Phase 3, S11.5).

| Method | Path | Auth | Roles | Description |
|--------|------|------|-------|-------------|
| POST | `/donations` | Yes | Secretary+ | Create donation |
| PUT | `/donations/:id` | Yes | Secretary+ | Edit donation |
| GET | `/donations` | Yes | Coordinator+ | List donations |
| GET | `/donations/:id` | Yes | Coordinator+ | Get donation detail |
| GET | `/donations/:id/receipt` | Yes | Coordinator+ | Download receipt PDF **(Phase 3 — S11.5)** |

All donation queries and mutations are scoped to the caller's campus (from the
token `campus_id` claim). Every mutation writes an `audit_log` entry
(module `donation`).

#### POST /donations

Request body fields:
- `donation_type` (string, required) — one of `FINANCIAL`, `GOODS`, `SERVICES`.
- `amount` (number, required **only** when `donation_type` is `FINANCIAL`, must be
  `> 0`) — the monetary value; ignored/optional for `GOODS`/`SERVICES`.
- `currency` (string, optional) — one of `BRL`, `USD`, `EUR`; defaults to `"BRL"`. Any other value returns `400 validation_error`. Amounts are stored in the native currency (no FX conversion).
- `item_description` (string, required **only** when `donation_type` is `GOODS` or
  `SERVICES`) — description of the donated goods/services.
- `donor_person_id` (uuid, optional) — the donor; must be visible in the caller's
  campus. A nonexistent or foreign-campus reference returns `400 validation_error`
  with a generic message (no cross-campus existence disclosure, T3). Omit for
  anonymous donations.
- `campaign_id` (uuid, optional) — the campaign to attribute the donation to; same
  campus-scoped validation and generic rejection as `donor_person_id` (RF-56).
- `donation_date` (string `YYYY-MM-DD`, optional) — defaults to the current date.
- `notes` (string, optional).

`receipt_number` and `receipt_issued_at` are **not** accepted in Phase 2 (receipt
issuance is Phase 3, S11.5); they are stored as `NULL`. `registered_by` is set from
the caller's token, never from the body.

Returns `201` with the created donation. Errors: `400 validation_error` (bad type,
missing conditional field, unresolvable donor/campaign), `403` (below Secretary),
`409 conflict` (duplicate `receipt_number`, reserved for Phase 3).

```json
{
  "id": "uuid",
  "donor_person_id": "uuid|null",
  "campaign_id": "uuid|null",
  "campus_id": "uuid",
  "donation_type": "FINANCIAL",
  "amount": 150.00,
  "currency": "BRL",
  "item_description": null,
  "donation_date": "2026-07-05",
  "receipt_number": null,
  "receipt_issued_at": null,
  "notes": null,
  "created_at": "timestamptz",
  "updated_at": "timestamptz"
}
```

#### PUT /donations/:id

Same body as `POST /donations`. Updates the mutable fields
(`donation_type`, `amount`, `currency`, `item_description`, `donor_person_id`,
`campaign_id`, `donation_date`, `notes`). `campus_id`, `registered_by`, and
`created_at` are immutable. Returns `200` with the updated donation; `404` if the
donation is not in the caller's campus.

#### GET /donations

Query params: `page` (default 1), `per_page` (default 20), `donation_type`
(filter, validated against the enum), `campaign_id` (filter, uuid). Returns a
paginated list of donation summaries scoped to the caller's campus, each carrying
the resolved donor and campaign names for display.

```json
{
  "data": [
    {
      "id": "uuid",
      "donation_type": "FINANCIAL",
      "amount": 150.00,
      "currency": "BRL",
      "donation_date": "2026-07-05",
      "donor_name": "Maria Silva|null",
      "campaign_name": "Campanha do Agasalho|null"
    }
  ],
  "pagination": { "page": 1, "per_page": 20, "total": 1, "total_pages": 1 }
}
```

#### GET /donations/:id

Returns the full donation (all fields above) plus resolved `donor_name` and
`campaign_name`. Returns `404` if the donation is not in the caller's campus.

#### GET /donations/:id/receipt · **Phase 3 (Sprint 10 — S11.5)**

Issues (once) and downloads the donation receipt PDF (RF-55). Coordinator+.
On the first call the API renders the PDF, stores it in object storage under
`receipts/{campus_id}/{donation_id}.pdf`, stamps a unique `receipt_number`
(format `REC-{YYYY}-{8-hex}`) and `receipt_issued_at` on the donation, and audits
the issuance (`action_type: UPDATE`, `description: "receipt issued"`). Subsequent
calls are idempotent — they return a fresh presigned URL for the existing receipt
without re-issuing. The response mirrors the document-download shape:

```json
// Response 200
{ "url": "https://storage.example.com/receipts/...&X-Amz-Signature=...", "expires_at": "2026-07-06T12:45:00Z" }
```

The receipt body shows the issuing campus/organization (legal name, CNPJ, address
when present), donor name and document, amount + currency, donation date, and the
receipt number. Errors: `404 not_found` (donation outside the caller's campus),
`403 forbidden` (below Coordinator, audited as `ACCESS_DENIED`).

---

## Consent Endpoints (Phase 2 — Sprint 6)

| Method | Path | Auth | Roles | Description |
|--------|------|------|-------|-------------|
| POST | `/consents` | Yes | All | Create consent record |
| GET | `/persons/:id/consents` | Yes | Secretary+ | List person's consents (active + revoked) |
| PATCH | `/consents/:id/revoke` | Yes | Admin | Revoke consent |

#### POST /consents

Request body fields:
- `person_id` (uuid, required) — must be visible in the caller's campus; a
  nonexistent or foreign-campus reference returns `400 validation_error` with a
  generic message (no cross-campus existence disclosure).
- `consent_type` (string, required) — one of `DATA_PROCESSING`, `IMAGE_USAGE`,
  `HEALTH_DATA`, `MINOR_GUARDIAN`.
- `purpose` (string, required) — the LGPD Art. 7/8 purpose shown to the person
  at collection time (RF-58a).
- `consent_version` (string, optional) — defaults to `"1.0"`.
- `granted_by_person_id` (uuid, optional) — guardian who granted on behalf of a
  minor (`MINOR_GUARDIAN`); same campus-scoped validation as `person_id`.
- `signature_data` (string, optional) — base64 PNG data URL captured by the
  signature pad (RF-57).

```json
// Request
{
  "person_id": "uuid",
  "consent_type": "DATA_PROCESSING",
  "purpose": "Cadastro e acompanhamento de atendimentos",
  "consent_version": "1.0",
  "signature_data": "data:image/png;base64,iVBOR..."
}

// Response 201
{
  "id": "uuid",
  "person_id": "uuid",
  "consent_type": "DATA_PROCESSING",
  "consent_version": "1.0",
  "purpose": "Cadastro e acompanhamento de atendimentos",
  "granted_at": "2026-07-03T14:30:00Z",
  "granted_by_person_id": null,
  "signature_data": "data:image/png;base64,iVBOR...",
  "is_active": true,
  "revoked_at": null,
  "revoked_reason": null
}
```

Error responses:
| HTTP | Error | When |
|------|-------|------|
| 400 | `validation_error` | Missing/invalid field, unknown `consent_type`, or unresolvable person reference (generic message) |
| 409 | `duplicate` | An active consent of the same type already exists for the person (`uq_consent_active_person_type`) |

#### GET /persons/:id/consents

Returns every consent of the person (active and revoked), ordered by
`granted_at` descending — the consent registry required by RF-58b. Responds
`404 not_found` when the person is not visible in the caller's campus.

```json
// Response 200
{ "data": [ { "id": "uuid", "consent_type": "IMAGE_USAGE", "is_active": false,
  "revoked_at": "2026-07-01T10:00:00Z", "revoked_reason": "Pedido do titular",
  "consent_version": "1.0", "purpose": "...", "granted_at": "2026-05-02T09:00:00Z" } ] }
```

#### PATCH /consents/:id/revoke

Request body fields:
- `revoked_reason` (string, required) — recorded in the registry and audit log.

Sets `is_active = false`, `revoked_at = now()`, keeps the row (audit trail per
docs/13). **Sprint 10 (S11.3):** when the revoked consent's type is
`DATA_PROCESSING`, the same request transaction additionally anonymizes the
linked `person` PII and `address` rows (right to erasure, RF-58), sets
`person.anonymized_at`, and audits the anonymization (without recording the
scrubbed PII). Other consent types revoke only. Atomicity is guaranteed by the
per-request campus transaction (any error rolls the revoke back). Responds `200`
with the updated consent. Errors:
| HTTP | Error | When |
|------|-------|------|
| 400 | `validation_error` | Missing reason, or the consent is already revoked |
| 403 | `forbidden` | Caller is not ADMIN (denial audited as `ACCESS_DENIED`) |
| 404 | `not_found` | Consent nonexistent or outside the caller's campus |

---

## Document Endpoints (Phase 2 — Sprint 6)

| Method | Path | Auth | Roles | Description |
|--------|------|------|-------|-------------|
| POST | `/persons/:id/documents` | Yes | Secretary+ | Upload document to a person |
| GET | `/persons/:id/documents` | Yes | Professional+ | List person's documents |
| POST | `/attendances/:id/documents` | Yes | Professional+ | Upload document/exam to an attendance (RF-30) |
| GET | `/attendances/:id/documents` | Yes | Professional+ | List attendance's documents |
| GET | `/documents/:id/download` | Yes | Professional+ | Get presigned download URL |

#### POST /persons/:id/documents · POST /attendances/:id/documents

`Content-Type: multipart/form-data` with fields:
- `document` (file, required) — allowed content types `application/pdf`,
  `image/jpeg`, `image/png`, verified by **magic bytes** (the client-sent
  Content-Type header is not trusted); maximum **10MB** (docs/19).
- `document_type` (string, required) — one of `ID`, `PROOF_OF_RESIDENCE`,
  `MEDICAL_RECORD`, `EXAM`, `CONSENT`, `PHOTO`, `OTHER`.
- `description` (string, optional).

The file is stored in object storage under a UUID-based key
(`documents/{campus_id}/{person_id}/{uuid}{ext}`); the original filename is
kept only as metadata (`file_name`). An attendance upload links the document to
the attendance's person and sets `attendance_id`.

```json
// Response 201
{
  "id": "uuid",
  "person_id": "uuid",
  "attendance_id": null,
  "document_type": "PROOF_OF_RESIDENCE",
  "file_name": "comprovante.pdf",
  "file_size": 182734,
  "mime_type": "application/pdf",
  "description": null,
  "uploaded_at": "2026-07-03T14:30:00Z"
}
```

Error responses:
| HTTP | Error | When |
|------|-------|------|
| 400 | `validation_error` | Missing file/field, disallowed or spoofed content type (magic bytes), file over 10MB, or unknown `document_type` |
| 403 | `forbidden` | Role below the required minimum (denial audited) |
| 404 | `not_found` | Person/attendance nonexistent or outside the caller's campus |

#### GET /persons/:id/documents · GET /attendances/:id/documents

Returns the entity's documents ordered by `uploaded_at` descending, same
response item shape as the 201 above (no download URL — see below). `404` when
the parent entity is not visible in the caller's campus.

#### GET /documents/:id/download

Returns a **time-limited presigned URL** (15 minutes) to the object-storage
key; the API never streams file bytes itself and files are never served from
the application filesystem (docs/19).

```json
// Response 200
{ "url": "https://storage.example.com/chesed-docs/documents/...?X-Amz-...", "expires_at": "2026-07-03T14:45:00Z" }
```

| HTTP | Error | When |
|------|-------|------|
| 404 | `not_found` | Document nonexistent or outside the caller's campus |

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
    For `triage` and `attendance` records, `data` may include an optional
    `campaign_id` (uuid); a value that is nonexistent or not visible in the
    caller's campus yields a per-record `error` result (nothing persisted).
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
    {
      "sync_id": "uuid",
      "status": "conflict",
      "message": "duplicate",
      "server_id": "uuid",
      "server_data": { "full_name": "Maria S.", "phone": "+55 11 90000-0000" },
      "server_updated_at": "2026-07-06T09:00:00Z",
      "conflicting_fields": ["phone"]
    },
    { "sync_id": "uuid", "status": "error",   "message": "invalid person_id: ..." }
  ],
  "server_timestamp": "2026-04-02T10:40:00Z"
}
```

Per-record `status`:
- `created` — new server record (or idempotent return of the existing one).
- `conflict` — DB constraint blocked the write (e.g., duplicate document).
- `error` — payload validation or DB error specific to the record.

On `conflict`, the server MAY return the current server row so the client can offer
field-level resolution (S12.2):
- `server_data` (object, optional) — a **lean, non-PII** projection of the existing
  server record, limited to the same fields the client already holds for that entity.
  It is never the full PII row (it travels to the offline client).
- `server_updated_at` (RFC3339, optional) — the server row's last update timestamp,
  for last-write-wins comparison.
- `conflicting_fields` (string[], optional) — the fields that differ between the
  client payload and the server row. Absent when the server cannot compute a diff;
  the client may diff `server_data` against its local copy instead.

These fields are omitted when no server row is resolvable (e.g., a validation-only
conflict), preserving backward compatibility with the Phase 1 `{status, message}` shape.

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
| GET | `/reports/campaigns/:id` | Yes | Coordinator+ | Campaign metrics | **Phase 2 (Sprint 5)** |
| GET | `/reports/dashboard` | Yes | Coordinator+ | Dashboard KPIs | **Phase 2 (Sprint 8)** |
| GET | `/reports/compliance` | Yes | Coordinator+ | LGPD compliance metrics | **Phase 3 (Sprint 10 — S11.4)** |
| GET | `/reports/compliance/export` | Yes | Coordinator+ | Compliance CSV export | **Phase 3 (Sprint 10 — S11.4)** |

#### GET /reports/attendances?start=2026-01-01&end=2026-03-31

Both `start` and `end` are required `YYYY-MM-DD` dates. The end day is inclusive
(server interprets as `< end + 1 day`). Range may not exceed 366 days. The query
is automatically campus-scoped from the caller's token. `FOLLOW_UP` is reserved
for Phase 2 and will not appear in `by_status` until then.

**Optional filters (Sprint 8, S10.1/S10.4)** — all narrow the same campus-scoped
result set and combine with `AND`:

| Query param | Type | Effect |
|-------------|------|--------|
| `service_type_id` | UUID | Restrict to one service type |
| `campaign_id` | UUID | Restrict to attendances linked to a campaign |
| `professional_id` | UUID | Restrict to one acting professional |

A malformed UUID in any filter returns `400 invalid_filter`. A filter id outside
the caller's campus simply yields empty aggregations (no existence disclosure);
the request still returns `200`.

The response adds a `by_professional` breakdown (Sprint 8) grouping attendances by
the acting professional, ordered by count desc then name asc.

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
  ],
  "by_professional": [
    { "professional_id": "uuid", "professional_name": "Maria Silva", "count": 120 },
    { "professional_id": "uuid", "professional_name": "João Souza", "count": 114 }
  ]
}
```

Error codes: `invalid_range` (missing/inverted/oversize), `invalid_start` /
`invalid_end` (malformed date), `invalid_filter` (malformed filter UUID),
`forbidden` (no campus in token), `range_too_large` (>366 days).

#### GET /reports/dashboard

Campus-scoped snapshot of key operational metrics for the current moment. Takes
no query parameters — all windows are computed server-side relative to the
request date. Coordinator+ only. Replaces the client-side KPI computation that
the dashboard page previously assembled from list endpoints.

- `attendances_this_month` counts attendances with `attendance_date` in the
  current calendar month (campus timezone assumed UTC).
- `attendances_by_status` is an all-time campus snapshot of the current status
  distribution.
- `upcoming_scheduled` counts `SCHEDULED` attendances dated today or later.
- `active_campaigns` counts campaigns with status `ACTIVE`.
- `recent_months` is the last 6 calendar months of attendance volume (oldest
  first), suitable for a trend chart. Months with no attendances appear with
  `count: 0`.

```json
// Response 200
{
  "total_persons": 512,
  "attendances_this_month": 84,
  "upcoming_scheduled": 17,
  "active_campaigns": 3,
  "attendances_by_status": {
    "COMPLETED": 640,
    "SCHEDULED": 40,
    "IN_PROGRESS": 22,
    "CANCELLED": 18
  },
  "recent_months": [
    { "month": "2026-02", "count": 61 },
    { "month": "2026-03", "count": 77 },
    { "month": "2026-04", "count": 70 },
    { "month": "2026-05", "count": 82 },
    { "month": "2026-06", "count": 91 },
    { "month": "2026-07", "count": 84 }
  ]
}
```

Error codes: `forbidden` (no campus in token), `internal_error`.

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

#### GET /reports/campaigns/:id

Campus-scoped campaign metrics. A campaign id outside the caller's campus
responds 404 (`not_found`).

```json
// Response 200
{
  "campaign_id": "uuid",
  "campaign_name": "March Social Action",
  "status": "ACTIVE",
  "period": { "start_date": "2026-07-10", "end_date": "2026-07-12" },
  "triage_count": 42,
  "attendance_total": 61,
  "attendances_by_status": {
    "COMPLETED": 50,
    "SCHEDULED": 4,
    "IN_PROGRESS": 5,
    "CANCELLED": 2
  },
  "team_size": 12
}
```

#### GET /reports/compliance?start=2026-01-01&end=2026-03-31

Campus-scoped LGPD compliance metrics for the period. `start`/`end` are required
`YYYY-MM-DD` dates (well-ordered; unlike the attendance report there is no 366-day
span cap — compliance posture is reviewed over multi-year retention windows).
Coordinator+.

```json
// Response 200
{
  "period": { "start": "2026-01-01", "end": "2026-03-31" },
  "consents_by_type": { "DATA_PROCESSING": 120, "IMAGE_USAGE": 80, "HEALTH_DATA": 15, "MINOR_GUARDIAN": 9 },
  "active_consents": 190,
  "revoked_consents": 34,
  "anonymized_subjects": 12,
  "data_subjects": 640,
  "documents_stored": 210
}
```

Errors: `400 invalid_range` (missing/malformed/reversed dates), `403 forbidden`
(below Coordinator, audited as `ACCESS_DENIED`).

#### GET /reports/compliance/export?format=csv&start=2026-01-01&end=2026-03-31

Returns `Content-Type: text/csv` with `Content-Disposition: attachment` — one
`metric,value` row per compliance metric. Writing the export records an `EXPORT`
audit entry (`entity_type: compliance_report`). `format` must be `csv`
(otherwise `400 invalid_format`).

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

| Method | Path | Auth | Roles | Description | Status |
|--------|------|------|-------|-------------|--------|
| GET | `/audit/logs` | Yes | Admin | Query audit logs | **Phase 3 (Sprint 10 — S11.6)** |

Read-only viewer for compliance teams (RF-53). The `audit_log` table stays
append-only — this endpoint only issues `SELECT`. Optional filters (all combine
with `AND`): `user_id` (uuid), `entity_type`, `action_type`, `start`/`end`
(`YYYY-MM-DD`, inclusive end). Paginated with `page`/`per_page` (default 1/50,
max per_page 100), ordered newest-first. `user_email` is resolved by joining
`app_user` on the Keycloak subject.

**Campus scoping**: results are restricted to the caller's campus at the SQL
layer (system rows with no campus are excluded). `audit_log` is intentionally
excluded from PostgreSQL RLS, so scoping is enforced by the query, not the policy.

Errors: `400` (malformed `user_id`/date), `403 forbidden` (below Admin, recorded
as `ACCESS_DENIED`).

#### GET /audit/logs?user_id=uuid&entity_type=person&action_type=UPDATE&start=2026-01-01&end=2026-03-31&page=1&per_page=50
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

## Admin Endpoints

| Method | Path | Auth | Roles | Description | Status |
|--------|------|------|-------|-------------|--------|
| POST | `/admin/retention/run` | Yes | Admin | Run the data-retention sweep | **Phase 3 (Sprint 10 — S11.7)** |

#### POST /admin/retention/run

Synchronously enforces the LGPD data-retention policy (RNF-01) for the caller's
campus: person records whose last activity predates the retention window (5 years
operational, per docs/13) are anonymized (reusing the S11.3 anonymization), each
action audited. Last activity is the most recent of the person's own row and any
related triage, attendance, or donation — a subject still being assisted is never
anonymized even if their profile row is old. Idempotent — already-anonymized
records are skipped. No request body. Admin only (`403 ACCESS_DENIED` otherwise).
No scheduler infra in the MVP; an external cron may invoke this endpoint.

```json
// Response 200
{ "scanned": 512, "anonymized": 7 }
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
