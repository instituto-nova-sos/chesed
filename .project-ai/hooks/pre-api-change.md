# Hook: Pre-API Change Validation

## Purpose

Ensure all API endpoint modifications are documented before implementation begins. Prevents undocumented contracts, missing RBAC definitions, and inconsistent error handling.

## Trigger Condition

Before creating, modifying, or removing any file in:
- `backend/internal/handler/`
- `backend/internal/middleware/` (route-related changes)
- Any router registration in the main application setup

## Status

**Blocking** — Do not modify API endpoints if gates fail.

## Steps

1. **Read current API contract**
   - Open `docs/11-api-design.md` and locate the endpoint being changed.
   - Note the documented: HTTP method, path, request schema, response schema, status codes, RBAC roles.
   - If making a new endpoint, verify the endpoint pattern follows `/api/v1/{resource}` convention.

2. **If endpoint is not documented, stop and document first**
   - STOP implementation.
   - Add the endpoint definition to `docs/11-api-design.md` with:
     - HTTP method and path
     - Request body schema (with field types and validation rules)
     - Response body schema (with field types)
     - Success and error status codes
     - Required RBAC roles
     - Query parameters (for list endpoints: pagination, filtering, sorting)
   - Get documentation reviewed before proceeding to implementation.

3. **Verify request/response schemas**
   - Confirm the Go request struct matches the documented request schema field-by-field.
   - Confirm the Go response struct matches the documented response schema field-by-field.
   - Verify all fields use the correct JSON naming convention (snake_case).
   - Verify UUID fields are typed as `uuid.UUID` in Go and documented as `string (UUID)` in docs.
   - Verify timestamp fields use `time.Time` in Go and ISO 8601 format in docs.

4. **Verify RBAC requirement is documented**
   - Confirm which roles can access this endpoint (ADMIN, COORDINATOR, PROFESSIONAL, SECRETARY, VOLUNTEER).
   - Cross-reference with `docs/16-iam-and-access-control.md` for role permission matrix.
   - Verify the middleware chain will include the RBAC check.
   - No endpoint may exist without an RBAC requirement.

5. **Verify error response format**
   - All error responses must follow the standard format:
     ```json
     {
       "error": {
         "code": "ERROR_CODE",
         "message": "Human-readable message"
       }
     }
     ```
   - Verify the following status codes are handled:
     - `400` — Validation errors
     - `401` — Missing or invalid token
     - `403` — Insufficient role/permissions
     - `404` — Resource not found
     - `409` — Conflict (e.g., duplicate)
     - `500` — Internal server error (no details leaked)
   - Verify no PII is included in error messages.

## Enforcement Mechanism

- The AI agent must execute this hook before writing or modifying any handler code.
- If the endpoint is not in `docs/11-api-design.md`, the agent must create the documentation entry first and present it for review.
- The agent must not create "temporary" or "draft" endpoints that bypass documentation.

## References

- `docs/11-api-design.md` — API endpoint contracts (source of truth)
- `docs/16-iam-and-access-control.md` — RBAC role definitions and permission matrix
- `docs/05-architecture-proposal.md` — Architecture patterns and conventions
- `docs/15-implementation-guidelines.md` — Coding standards for handlers

## Consequences of Skipping

- Undocumented endpoints become invisible technical debt that breaks contract-first development.
- Missing RBAC definitions leave endpoints unprotected, violating security requirements.
- Inconsistent error formats break frontend error handling and degrade user experience.
- Schema mismatches between docs and code cause integration failures during frontend development.
