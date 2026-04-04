# Security Review Report

## Metadata

| Field | Value |
|-------|-------|
| **Feature/Change** | [Brief description of what was reviewed] |
| **Date** | [YYYY-MM-DD] |
| **Reviewer** | [Name or AI agent identifier] |
| **Sprint** | [Sprint number] |
| **Verdict** | **PASS** / **PASS WITH FINDINGS** / **FAIL** |

---

## Scope

### Changed Files

| File | Category | Security Relevance |
|------|----------|-------------------|
| [file path] | [Handler / Service / Repository / Middleware / Migration / Frontend / Keycloak] | [Critical / High / Medium / Low] |

### Affected Endpoints

| Method | Path | Roles | Change Type |
|--------|------|-------|-------------|
| [GET/POST/PUT/PATCH/DELETE] | /api/v1/[path] | [Role list] | New / Modified |

### Affected Data Flows

- [ ] User input -> API -> Database
- [ ] Database -> API -> Client response
- [ ] Keycloak -> API (token validation)
- [ ] Client -> IndexedDB (offline storage)
- [ ] IndexedDB -> API (sync push)
- [ ] API -> Audit log

---

## Findings

| # | Severity | Category | Finding | Remediation | Status |
|---|----------|----------|---------|-------------|--------|
| 1 | [Critical/High/Medium/Low/Info] | [OWASP category or Chesed-specific] | [Description of the finding] | [Required action] | [Open / Fixed / Accepted Risk] |
| 2 | | | | | |
| 3 | | | | | |

Severity definitions:
- **Critical**: Immediate exploitation risk, data breach possible. Must fix before merge.
- **High**: Significant security weakness. Must fix before sprint delivery.
- **Medium**: Security concern that should be addressed. Fix within next sprint.
- **Low**: Minor hardening opportunity. Track in backlog.
- **Info**: Observation or recommendation. No action required.

---

## OWASP Top 10 Check

| # | Category | Status | Notes |
|---|----------|--------|-------|
| A01 | Broken Access Control | PASS / FAIL / N/A | [e.g., "All endpoints have RBAC middleware. Campus scoping verified."] |
| A02 | Cryptographic Failures | PASS / FAIL / N/A | [e.g., "No secrets in source code. IndexedDB encryption in place."] |
| A03 | Injection | PASS / FAIL / N/A | [e.g., "All queries parameterized. No string concatenation in SQL."] |
| A04 | Insecure Design | PASS / FAIL / N/A | [e.g., "Business logic enforces authorization. Audit logging present."] |
| A05 | Security Misconfiguration | PASS / FAIL / N/A | [e.g., "CORS restricts origins. No debug mode in production config."] |
| A06 | Vulnerable Components | PASS / FAIL / N/A | [e.g., "go mod tidy clean. npm audit shows 0 high/critical."] |
| A07 | Auth Failures | PASS / FAIL / N/A | [e.g., "Auth delegated to Keycloak. JWKS validation in place."] |
| A08 | Data Integrity Failures | PASS / FAIL / N/A | [e.g., "Sync push validates server-side. Audit log is append-only."] |
| A09 | Logging Failures | PASS / FAIL / N/A | [e.g., "Mutations audit-logged. No PII in log output."] |
| A10 | SSRF | PASS / FAIL / N/A | [e.g., "No user-controlled URLs in server-side requests."] |

---

## Chesed-Specific Checks

### Campus Data Isolation (T3)

| Check | Status | Evidence |
|-------|--------|----------|
| All repository queries include `campus_id` in WHERE clause | PASS / FAIL | [File:line references] |
| campus_id sourced from JWT claims (middleware context), not request body | PASS / FAIL | [File:line references] |
| List endpoints filter by campus_id | PASS / FAIL | [File:line references] |
| Integration test verifies cross-campus isolation | PASS / FAIL | [Test file reference] |

### Audit Logging

| Check | Status | Evidence |
|-------|--------|----------|
| All POST/PUT/PATCH/DELETE handlers create audit log entries | PASS / FAIL | [File:line references] |
| Audit log includes: entity_type, entity_id, action, user_id, campus_id | PASS / FAIL | [Audit entry example] |
| Audit log table has no UPDATE or DELETE operations in code | PASS / FAIL | [Grep result] |
| old_values captured for UPDATE operations | PASS / FAIL | [File:line references] |

### No PII in Logs

| Check | Status | Evidence |
|-------|--------|----------|
| slog calls do not include: full_name, document_number, email, phone, address | PASS / FAIL | [Grep result] |
| Error responses do not include PII | PASS / FAIL | [Handler review] |
| Stack traces not exposed in production error responses | PASS / FAIL | [Error handler review] |

### No Custom Authentication

| Check | Status | Evidence |
|-------|--------|----------|
| No login, register, or password endpoints in the API | PASS / FAIL | [Route registration review] |
| No password hashing or credential storage in application code | PASS / FAIL | [Grep result] |
| No custom JWT issuance (all tokens from Keycloak) | PASS / FAIL | [Code review] |
| keycloak-js used for frontend authentication flow | PASS / FAIL | [Frontend auth code review] |

### Keycloak Token Validation

| Check | Status | Evidence |
|-------|--------|----------|
| Token validated via coreos/go-oidc using JWKS endpoint | PASS / FAIL | [Middleware code reference] |
| Audience claim (aud) validated as "chesed-api" | PASS / FAIL | [Middleware code reference] |
| Token expiration (exp) checked | PASS / FAIL | [Built into go-oidc] |
| Issuer (iss) validated against configured Keycloak realm URL | PASS / FAIL | [Middleware code reference] |

### IndexedDB Encryption (T4)

| Check | Status | Evidence |
|-------|--------|----------|
| Sensitive fields encrypted before IndexedDB storage | PASS / FAIL / N/A | [Encryption code reference] |
| AES-256-GCM algorithm used via Web Crypto API | PASS / FAIL / N/A | [Encryption code reference] |
| Encryption key is per-user (not global) | PASS / FAIL / N/A | [Key derivation code] |
| Logout clears all IndexedDB data | PASS / FAIL / N/A | [Logout handler code] |
| Session timeout after 30 minutes of inactivity | PASS / FAIL / N/A | [Session management code] |

### Sync Endpoint Security (T5)

| Check | Status | Evidence |
|-------|--------|----------|
| Sync push validates all fields server-side | PASS / FAIL / N/A | [Sync handler code] |
| Sync push enforces campus_id from JWT (not payload) | PASS / FAIL / N/A | [Sync handler code] |
| Sync push checks sync_id for idempotency | PASS / FAIL / N/A | [Sync handler code] |
| Rate limiting on sync endpoint | PASS / FAIL / N/A | [Middleware config] |

---

## Threat Model Coverage

Reference: `docs/18-threat-model.md`

| Threat | Relevant | Mitigated | Notes |
|--------|----------|-----------|-------|
| T1: Credential stuffing | Yes / No | Yes / No / Partial | [Keycloak brute-force protection] |
| T2: RBAC escalation | Yes / No | Yes / No / Partial | [Middleware on all endpoints] |
| T3: Campus isolation breach | Yes / No | Yes / No / Partial | [campus_id in all queries] |
| T4: Offline device theft | Yes / No | Yes / No / Partial | [IndexedDB encryption] |
| T5: Sync data injection | Yes / No | Yes / No / Partial | [Server-side validation] |
| T6: Data exfiltration via reports | Yes / No | Yes / No / Partial | [Role restriction + audit] |
| T7: IDOR | Yes / No | Yes / No / Partial | [Campus scoping prevents cross-tenant] |
| T8: XSS | Yes / No | Yes / No / Partial | [React auto-escapes, no dangerouslySetInnerHTML] |
| T9: CSRF | Yes / No | Yes / No / Partial | [Bearer token auth, no cookies] |
| T10: SQL injection | Yes / No | Yes / No / Partial | [Parameterized queries] |
| T11: JWT manipulation | Yes / No | Yes / No / Partial | [JWKS signature validation] |
| T12: Keycloak misconfiguration | Yes / No | Yes / No / Partial | [Realm export reviewed] |

---

## Verdict

**Overall: PASS / PASS WITH FINDINGS / FAIL**

### Summary

[1-3 sentence summary of the review outcome.]

### Required Actions Before Merge

| # | Action | Severity | Assigned To |
|---|--------|----------|-------------|
| 1 | [Required action] | [Critical/High] | [Person] |

### Recommended Actions (Non-Blocking)

| # | Action | Severity | Timeline |
|---|--------|----------|----------|
| 1 | [Recommended improvement] | [Medium/Low] | [Next sprint / Backlog] |
