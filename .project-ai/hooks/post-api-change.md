# Hook: Post-API Change Sync

## Purpose

Ensure documentation, RBAC middleware, audit logging, and campus scoping remain in sync after any API endpoint is created or modified.

## Trigger Condition

After completing implementation of any API endpoint change — new endpoint, modified endpoint, or removed endpoint in `backend/internal/handler/`.

## Status

**Non-blocking** — But mandatory before marking the work as complete. Do not close a story or submit for review until all steps pass.

## Steps

1. **Run API contract review**
   - Execute the `review-api-contract` skill.
   - Compare the implemented endpoint (method, path, request/response schemas, status codes) against `docs/11-api-design.md`.
   - List any deviations found.

2. **Update docs/11-api-design.md if implementation differs**
   - If the implementation intentionally deviates from the original documented contract (e.g., added a field, changed a status code), update the documentation to match.
   - If the deviation was unintentional, fix the implementation to match the documentation.
   - Document the reason for any intentional changes in the commit message.

3. **Verify RBAC middleware is registered**
   - Confirm the endpoint's route registration includes the RBAC middleware with the correct role set.
   - Verify the roles match what is documented in `docs/11-api-design.md` and `docs/16-iam-and-access-control.md`.
   - Check that the middleware extracts and validates the `campus_id` claim from the Keycloak token.
   - Test mentally: "If a VOLUNTEER hits a COORDINATOR-only endpoint, will they get a 403?"

4. **Verify audit logging is present**
   - For all data mutation endpoints (POST, PUT, PATCH, DELETE):
     - Confirm the handler or service layer creates an audit log entry.
     - Verify the audit log captures: `user_id`, `action`, `entity_type`, `entity_id`, `old_values`, `new_values`, `campus_id`, `timestamp`.
   - For read-only endpoints (GET): audit logging is optional but campus scoping is still mandatory.
   - Reference: `docs/13-security-and-compliance.md` for audit requirements.

5. **Verify campus scoping is applied**
   - Confirm every database query in the endpoint's call chain includes a `WHERE campus_id = $N` filter.
   - The `campus_id` must come from the authenticated user's Keycloak token claims, never from request parameters.
   - Verify that list endpoints filter by campus_id.
   - Verify that single-resource endpoints check campus_id ownership before returning data.

## Enforcement Mechanism

- The AI agent must execute this hook after completing any handler implementation.
- All five steps must pass before the story can be marked as complete in `tasks/todo.md`.
- Deviations found in step 1 must be resolved (either fix code or update docs) before proceeding.

## References

- `docs/11-api-design.md` — API endpoint contracts
- `docs/16-iam-and-access-control.md` — RBAC role definitions
- `docs/13-security-and-compliance.md` — Audit logging requirements
- `docs/04-domain-model.md` — Campus scoping rules

## Consequences of Skipping

- Documentation drift makes `docs/11-api-design.md` unreliable, breaking the documentation-first workflow.
- Missing RBAC middleware leaves endpoints unprotected — a critical security vulnerability.
- Missing audit logging violates compliance requirements and makes incident investigation impossible.
- Missing campus scoping allows cross-campus data access — a data isolation violation.
