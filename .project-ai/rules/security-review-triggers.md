# Rule: Security Review Triggers

## Purpose

Define exactly which code changes require a mandatory security review. Ensure security-sensitive areas are never modified without explicit verification against the project's security requirements and threat model.

## Rule Statement

Security review is mandatory when touching authentication middleware, authorization logic, PII fields, Keycloak configuration, IndexedDB/sync logic, RBAC definitions, or audit logging. The review must be executed and documented before the work is marked complete.

## Trigger Condition

Any time the AI agent creates or modifies files in the areas listed below.

## Enforcement

### Trigger Areas

A security review is required when ANY of the following files or areas are modified:

1. **Authentication middleware**
   - `backend/internal/middleware/auth.go`
   - Any OIDC token validation logic
   - JWKS configuration or endpoint changes
   - Token extraction or claims parsing

2. **Audit middleware and logging**
   - `backend/internal/middleware/audit.go`
   - `audit_log` table schema or repository
   - Any code that writes to or reads from the audit log
   - Changes to what data is captured in audit entries

3. **New API endpoints**
   - Every new endpoint in `backend/internal/handler/`
   - Route registration in the main router setup
   - Changes to existing endpoint authorization requirements

4. **Keycloak configuration**
   - `keycloak/realm-export.json`
   - Keycloak client configuration
   - Role mappings or realm settings
   - `docs/20-keycloak-configuration.md`

5. **IndexedDB and sync logic**
   - `frontend/src/offline/` — All files
   - Dexie.js table definitions and schema changes
   - Sync queue implementation
   - Conflict resolution logic
   - Any code that stores data locally on the device

6. **PII fields**
   - Any code handling these fields: `full_name`, `document_number`, `email`, `phone`, `birth_date`, `address`, `assisted_profile` data
   - Frontend components displaying or collecting PII
   - Backend handlers or services processing PII
   - Database queries returning PII
   - Log statements that might inadvertently include PII

7. **RBAC role requirements**
   - Role-checking middleware configuration
   - Changes to which roles can access which endpoints
   - New role definitions or role hierarchy changes
   - Frontend route guards based on roles

### Review Actions

When a trigger is activated, the AI agent must:

1. **Execute the `review-security` skill** against the changed files.
2. **Check against threat model**: Open `docs/18-threat-model.md` and verify the change does not introduce any documented threat.
3. **Check against security requirements**: Open `docs/13-security-and-compliance.md` and verify compliance.
4. **Check against secure development standard**: Open `docs/19-secure-development-standard.md` for coding practices.
5. **Document findings**: Record any security observations, mitigations applied, or risks accepted.

### Specific Checks by Area

| Area | Key Checks |
|------|-----------|
| Auth middleware | Token validation complete, JWKS endpoint correct, expired tokens rejected, claims extracted properly |
| Audit logging | All mutations logged, old/new values captured, user_id and campus_id present, no PII in log messages |
| New endpoints | RBAC middleware applied, campus_id filtered, input validated, error responses leak no internals |
| Keycloak config | Client secrets not in code, redirect URIs restricted, roles properly mapped |
| IndexedDB/sync | Sensitive data encrypted at rest, sync conflicts handled safely, no PII in browser logs |
| PII handling | No PII in logs, no PII in error responses, PII masked in non-essential displays, access controlled |
| RBAC | Least privilege applied, no role bypass paths, role checks on both frontend and backend |

## Enforcement Mechanism

- The `pre-implement` hook flags security-sensitive changes.
- The `pre-review` hook requires security review completion before marking work done.
- The AI agent must not skip security review by splitting changes across multiple small commits.
- Security findings must be documented in the commit message or `tasks/todo.md`.

## References

- `docs/13-security-and-compliance.md` — Security requirements
- `docs/16-iam-and-access-control.md` — IAM and RBAC definitions
- `docs/17-security-test-strategy.md` — Security testing approach
- `docs/18-threat-model.md` — Identified threats and mitigations
- `docs/19-secure-development-standard.md` — Secure coding practices
- `docs/20-keycloak-configuration.md` — Keycloak setup and configuration

## Consequences of Skipping

- Authentication bypasses can expose the entire system to unauthorized access.
- Missing audit logging violates compliance requirements and makes incident investigation impossible.
- PII leakage in logs or error messages violates privacy requirements (LGPD compliance for Brazilian NGO).
- RBAC gaps allow privilege escalation — a VOLUNTEER accessing ADMIN-only data.
- IndexedDB vulnerabilities can expose sensitive data on shared or stolen devices.
- Keycloak misconfigurations can create authentication backdoors or session hijacking vectors.
