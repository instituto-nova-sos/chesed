# Skill: Review Security

## Purpose

Perform a security review on code changes that touch authentication, PII handling, RBAC, data access patterns, or offline storage. Validate against the project's threat model (T1-T12), OWASP Top 10, and LGPD requirements.

## When to Use / Trigger

- Any PR that modifies `middleware/auth.go`, `middleware/rbac.go`, or `middleware/audit.go`.
- Any PR that handles PII fields (document_number, phone, email, health data, social observations).
- Any PR that adds or modifies API endpoints.
- Any PR that changes offline storage (IndexedDB/Dexie.js).
- Any PR that modifies Keycloak configuration.
- When a user says "security review" or "check security for this change".
- Before any release (invoked by assess-release-readiness skill).

## Role / Expertise

Security engineer specializing in:
- OIDC/OAuth 2.0 with Keycloak.
- RBAC enforcement patterns.
- LGPD compliance for PII handling.
- OWASP Top 10 web application security.
- Offline-first PWA security (IndexedDB, service workers).

## Inputs

| Input | Required | Source |
|-------|----------|--------|
| Code diff or files under review | Yes | Git diff or file paths |
| Threat model | Yes | `docs/18-threat-model.md` |
| Security requirements | Yes | `docs/13-security-and-compliance.md` |
| IAM design | Yes | `docs/16-iam-and-access-control.md` |
| Security test strategy | Yes | `docs/17-security-test-strategy.md` |

## Process

### Step 1: Check Authentication (Threat T1, T7)

- [ ] No custom login forms, password hashing, or token issuance in application code.
- [ ] All authentication delegated to Keycloak via keycloak-js (frontend) and coreos/go-oidc (backend).
- [ ] OIDC token validation uses JWKS endpoint (not hardcoded keys).
- [ ] Token issuer validated against `KEYCLOAK_REALM_URL`.
- [ ] Token audience validated against `chesed-api`.
- [ ] Access token expiry is short (15 minutes).
- [ ] No refresh tokens stored in JavaScript-accessible storage.
- [ ] No `/auth/*` endpoints in the Go API.

### Step 2: Check RBAC (Threat T2)

- [ ] Every endpoint has RBAC middleware applied.
- [ ] Roles are extracted from JWT `realm_access.roles` claim (not from request body).
- [ ] Role hierarchy respected: ADMIN > COORDINATOR > PROFESSIONAL > SECRETARY > VOLUNTEER.
- [ ] 403 responses logged in audit trail.
- [ ] No hardcoded role bypass or admin backdoors.
- [ ] Role checks documented per endpoint in `docs/11-api-design.md`.

### Step 3: Check Campus Isolation (Threat T3)

- [ ] All data queries include `campus_id` filter from JWT claims.
- [ ] `campus_id` is NEVER accepted from client input for data filtering.
- [ ] Repository layer enforces campus filter at SQL level.
- [ ] Cross-campus access attempts are logged.
- [ ] Admin cross-campus access is explicitly documented and audited.

### Step 4: Check Offline Security (Threat T4)

- [ ] Sensitive fields encrypted in IndexedDB using Web Crypto API.
- [ ] Session timeout re-authentication after 30 minutes of inactivity.
- [ ] "Clear local data" accessible from login screen.
- [ ] Automatic data wipe on logout (keycloak.logout() clears IndexedDB).
- [ ] Minimal data cached offline (recent records only, not full database).
- [ ] No PII stored in localStorage (only IndexedDB with encryption).

### Step 5: Check Sync Endpoint Security (Threat T5)

- [ ] Full server-side validation on all sync payloads.
- [ ] Sync records validate `campus_id` matches JWT claims.
- [ ] `sync_id` used for idempotency (UUID collision check).
- [ ] Rate limiting on sync endpoints.
- [ ] Sync operations create audit log entries.

### Step 6: Check Data Exfiltration Controls (Threat T6)

- [ ] Export/report endpoints restricted to Coordinator+ role.
- [ ] Every export logged in audit trail.
- [ ] Row count limits on exports.
- [ ] Critical-tier data excluded from exports by default.

### Step 7: Check XSS/Injection Prevention (Threat T7, OWASP A03)

- [ ] All SQL queries use parameterized statements (pgx prepared statements).
- [ ] No string concatenation in SQL queries.
- [ ] Input validation on all user-supplied data (go-playground/validator).
- [ ] React auto-escapes rendered content (no `dangerouslySetInnerHTML`).
- [ ] Content Security Policy headers configured.

### Step 8: Check PII Handling (LGPD)

- [ ] No PII in log messages (`slog` fields must not include document_number, phone, email, health data).
- [ ] No PII in error responses.
- [ ] PII fields identified per `docs/13-security-and-compliance.md` data classification:
  - Critical: health data, social vulnerability, income, housing, special needs.
  - High: CPF/SSN, full name, address, phone, email.
- [ ] Audit logging for all PII access and mutations.
- [ ] No PII in URL paths or query parameters (use request body for searches).

### Step 9: Check Audit Logging

- [ ] All data mutations (CREATE, UPDATE, DELETE) create audit log entries.
- [ ] Audit log captures: entity_type, entity_id, action, old_values, new_values, performed_by, campus_id.
- [ ] Audit log table is append-only (no UPDATE or DELETE operations on audit_log).
- [ ] PII in audit log old_values/new_values is acceptable (needed for LGPD compliance).
- [ ] Failed authorization attempts (403) logged.

### Step 10: Check Keycloak Configuration (Threat T8-T12)

- [ ] Brute-force protection enabled (10 failures, 15-minute lockout).
- [ ] MFA required for ADMIN role.
- [ ] Password policy enforced (minimum length, complexity).
- [ ] Client secrets not in source code or Docker images.
- [ ] Realm configuration exported to `keycloak/realm-export.json`.
- [ ] Admin console access restricted.

## Outputs / Deliverables

A security review report with:

1. **Risk Summary**: HIGH / MEDIUM / LOW findings count.
2. **Findings** (per category):
   - Finding ID (SEC-001, SEC-002, ...).
   - Severity: CRITICAL / HIGH / MEDIUM / LOW.
   - Threat reference: T1-T12 or OWASP category.
   - Description: what is wrong.
   - Location: file path and line range.
   - Recommendation: specific fix.
3. **Blockers**: Any CRITICAL or HIGH finding that must be fixed before merge.
4. **Passed Checks**: List of categories that passed review.

## References

| Document | Path | Usage |
|----------|------|-------|
| Threat model | `docs/18-threat-model.md` | T1-T12 threat scenarios |
| Security and compliance | `docs/13-security-and-compliance.md` | LGPD, data classification |
| IAM and access control | `docs/16-iam-and-access-control.md` | Keycloak config, RBAC |
| Security test strategy | `docs/17-security-test-strategy.md` | Test patterns |
| Secure development standard | `docs/19-secure-development-standard.md` | Coding rules |
| Keycloak configuration | `docs/20-keycloak-configuration.md` | Realm settings |

### Software Quality: Security Dimension

This skill validates the **Security** dimension defined in `docs/quality/quality-profiles.md`. Security findings directly impact the Security rating in the quality gate:

- Security rating A = 0 vulnerabilities → quality gate PASS
- Any vulnerability → Security rating degrades → quality gate FAIL

Quality gate reference: `docs/quality/quality-gates.md`

## Constraints / Quality Bar

- Any CRITICAL finding is a merge blocker.
- Any HIGH finding must have a remediation plan before merge.
- Custom authentication code is an automatic CRITICAL finding.
- PII in logs is an automatic HIGH finding.
- Missing RBAC on an endpoint is an automatic HIGH finding.
- Missing audit logging on a mutation is a MEDIUM finding.
- Missing campus_id filtering is a HIGH finding.

## Interaction with Other Artifacts

- **Invoked by agents**: security-engineer (primary), tech-lead (gate check).
- **Used alongside skills**: review-code (code quality), review-api-contract (API conformance).
- **Blocks**: assess-release-readiness (any CRITICAL/HIGH finding blocks release).
