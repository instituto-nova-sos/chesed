# API Review Checklist

Use this checklist when reviewing any API endpoint — new or modified. Every item must pass before the endpoint is approved.

---

## Contract Compliance

- [ ] Endpoint path matches `docs/11-api-design.md` (e.g., `/api/v1/persons`, not `/api/v1/person`)
- [ ] HTTP method matches docs (GET, POST, PUT, PATCH, DELETE)
- [ ] Request body schema matches docs — field names, types, required/optional
- [ ] Response body schema matches docs — field names, types, nesting
- [ ] If endpoint is not in `docs/11-api-design.md`, add it before implementing

## Request Format

- [ ] `Content-Type: application/json` expected and validated
- [ ] Request body validated with `go-playground/validator` before processing
- [ ] `sync_id` field accepted for offline-created records (where applicable per `docs/12-offline-sync-strategy.md`)
- [ ] Query parameters for filtering, sorting, pagination documented and validated

## Response Format

- [ ] Success responses return appropriate status codes:
  - `200 OK` — successful read or update
  - `201 Created` — successful resource creation (with `Location` header if applicable)
  - `204 No Content` — successful delete
- [ ] Error responses return appropriate status codes:
  - `400 Bad Request` — validation error, malformed input
  - `401 Unauthorized` — missing or invalid Keycloak token
  - `403 Forbidden` — valid token but insufficient role/campus
  - `404 Not Found` — resource does not exist (or not in user's campus)
  - `409 Conflict` — duplicate record or state conflict
- [ ] Error response body follows standard format:
  ```json
  {
    "error": "VALIDATION_ERROR",
    "message": "Human-readable description",
    "details": [ ... ]
  }
  ```
- [ ] No PII in error responses (no names, CPF, phone numbers)

## Pagination

- [ ] List endpoints support pagination query parameters: `page`, `per_page`
- [ ] Response includes pagination metadata:
  ```json
  {
    "data": [ ... ],
    "page": 1,
    "per_page": 20,
    "total": 150,
    "total_pages": 8
  }
  ```
- [ ] Default `per_page` is reasonable (e.g., 20) with a maximum cap (e.g., 100)

## Authentication & Authorization

- [ ] RBAC role requirement matches `docs/11-api-design.md`
- [ ] Keycloak OIDC token validated via `coreos/go-oidc` + JWKS
- [ ] RBAC middleware applied on the route (not checked inside handler)
- [ ] `campus_id` extracted from token claims and applied to all queries
- [ ] Resources from other campuses are invisible (return 404, not 403)

## Data Integrity

- [ ] Audit log entry created for all mutations (CREATE, UPDATE, DELETE)
- [ ] Audit log captures: `user_id`, `campus_id`, `action`, `table_name`, `record_id`, `old_values`, `new_values`
- [ ] UUID used for all resource identifiers
- [ ] Timestamps returned in ISO 8601 format with timezone

## Security

- [ ] SQL queries use parameterized placeholders (`$1`, `$2`) — no string interpolation
- [ ] Input length limits enforced (prevent oversized payloads)
- [ ] No sensitive data in URL path or query parameters (use request body for sensitive fields)

---

## How to Use

Run this checklist for every endpoint during code review. Cross-reference against `docs/11-api-design.md` for the authoritative contract.

```
Skill:   review-api-contract (automated contract verification)
Hook:    pre-api-change (run before modifying any endpoint)
Hook:    post-api-change (run after modifying any endpoint)
Agent:   backend-engineer (for implementation), tech-lead (for design review)
```
