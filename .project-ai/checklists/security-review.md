# Security Review Checklist

Use this checklist for any security-sensitive change. Triggered by the `security-review-triggers` rule or manually for high-risk features. Every item must pass.

---

## Authentication (Keycloak Only)

- [ ] No custom login forms, password hashing, or token issuance in codebase
- [ ] All authentication delegated to Keycloak (OIDC)
- [ ] Keycloak token validation uses `coreos/go-oidc` with JWKS endpoint
- [ ] Token signature verified against Keycloak's public keys (not a shared secret)
- [ ] Token expiration and issuer claims validated
- [ ] No backdoor endpoints that accept raw credentials

## Authorization (RBAC)

- [ ] RBAC middleware applied on all endpoints (no unprotected routes except `/health`)
- [ ] Role requirements match `docs/11-api-design.md` and `docs/16-iam-and-access-control.md`
- [ ] Roles enforced: ADMIN, COORDINATOR, PROFESSIONAL, SECRETARY, VOLUNTEER
- [ ] `campus_id` extracted from Keycloak token claims in every query
- [ ] Cross-campus data access is impossible (queries always filter by `campus_id`)
- [ ] Resource-not-found returned (404) instead of forbidden (403) for cross-campus resources (prevents enumeration)

## Data Protection — Backend

- [ ] No PII in `slog` structured logging output (no names, CPF, phone, address, email)
- [ ] No PII in error responses returned to clients
- [ ] No PII in URL paths or query parameters
- [ ] SQL queries use parameterized placeholders only (`pgx` with `$1`, `$2`)
- [ ] No string concatenation or interpolation in SQL queries
- [ ] Input validation on all handler inputs (`go-playground/validator`)
- [ ] Input length limits enforced to prevent oversized payloads

## Data Protection — Frontend

- [ ] IndexedDB encryption for sensitive offline data using Web Crypto API (AES-256-GCM)
- [ ] Encryption keys derived from user session, not hardcoded
- [ ] Sensitive data cleared from IndexedDB on logout
- [ ] No sensitive data stored in `localStorage` or `sessionStorage`
- [ ] No sensitive data in browser console logs (`console.log` removed)

## Secrets Management

- [ ] No hardcoded secrets, API keys, or credentials in source code
- [ ] No secrets in Docker images or `docker-compose.yml` files
- [ ] All secrets injected via environment variables at runtime
- [ ] No Keycloak client secrets or admin passwords in source code
- [ ] `.env` files listed in `.gitignore`

## Audit Trail

- [ ] Audit log entries created for all data mutations (CREATE, UPDATE, DELETE)
- [ ] Audit log captures: `user_id`, `campus_id`, `action`, `table_name`, `record_id`, `old_values` (JSONB), `new_values` (JSONB)
- [ ] Audit log table schema does not allow UPDATE or DELETE operations
- [ ] Audit log entries include timestamp with timezone

## Infrastructure & Configuration

- [ ] CORS configured to restrict allowed origins (not wildcard `*` in production)
- [ ] HTTPS enforced in production (TLS termination at reverse proxy or load balancer)
- [ ] Keycloak realm configuration changes exported to `keycloak/realm-export.json` and committed
- [ ] Rate limiting configured for authentication-related endpoints
- [ ] Security headers set: `X-Content-Type-Options`, `X-Frame-Options`, `Strict-Transport-Security`

## Dependency Security

- [ ] No known critical vulnerabilities in Go dependencies (`go list -m -json all`)
- [ ] No known critical vulnerabilities in npm dependencies (`npm audit`)
- [ ] Dependencies pinned to specific versions (no floating ranges for security-critical packages)

---

## Severity Classification

When findings are identified, classify them:

| Severity | Definition | Action |
|----------|-----------|--------|
| **CRITICAL** | Exploitable vulnerability, data breach risk | Block release. Fix immediately. |
| **HIGH** | Security control missing or bypassed | Block release. Fix before merge. |
| **MEDIUM** | Defense-in-depth gap, hardening issue | Fix within current sprint. |
| **LOW** | Best practice deviation, minor improvement | Track and fix in next sprint. |

---

## How to Use

Run this checklist for any change that touches authentication, authorization, data access, secrets, or audit logging. See `docs/18-threat-model.md` for threat context.

```
Skill:   review-security (automated security analysis)
Playbook: conduct-security-review (full security review process)
Hook:    pre-review (includes security checks for flagged changes)
Agent:   security-engineer (for deep security review)

Reference docs:
  - docs/13-security-and-compliance.md
  - docs/16-iam-and-access-control.md
  - docs/17-security-test-strategy.md
  - docs/18-threat-model.md
  - docs/19-secure-development-standard.md
  - docs/20-keycloak-configuration.md
```
