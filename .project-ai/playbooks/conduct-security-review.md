# Playbook: Conduct Security Review

## Purpose

Structured security review process for Chesed changes. Covers OWASP Top 10, Chesed-specific threats (T1-T12 from `docs/18-threat-model.md`), Keycloak integration, campus isolation, and offline data protection.

---

## When to Run

- Before merging any PR that touches: authentication/authorization, data access patterns, sync endpoints, report/export endpoints, Keycloak configuration, new API endpoints
- As part of sprint delivery (see `playbooks/prepare-sprint-delivery.md`)
- After any security-related bug fix

---

## Steps

### Step 1: Define Scope

Identify what changed:

```bash
# List changed files since the base branch
git diff --name-only main...HEAD

# Or for a specific commit range
git diff --name-only <commit-range>
```

Categorize changed files:

| Category | File patterns | Security relevance |
|---|---|---|
| Auth middleware | `backend/internal/middleware/auth*.go` | Critical |
| RBAC middleware | `backend/internal/middleware/rbac*.go` | Critical |
| Handlers | `backend/internal/handler/*.go` | High (input validation, response data) |
| Services | `backend/internal/service/*.go` | High (business logic, authorization) |
| Repositories | `backend/internal/repository/*.go` | High (SQL injection, campus scoping) |
| Migrations | `backend/migrations/*.sql` | High (schema security, constraints) |
| Keycloak config | `keycloak/realm-export.json` | Critical |
| Frontend auth | `frontend/src/hooks/useAuth.ts`, `frontend/src/components/auth/*` | High |
| Offline storage | `frontend/src/offline/*` | High (PII in IndexedDB) |
| API client | `frontend/src/api/*` | Medium (token handling) |
| Sync endpoints | `backend/internal/handler/sync*.go` | High (data injection) |

### Step 2: Map to Threat Model

Reference: `docs/18-threat-model.md`

For each changed area, identify relevant threat scenarios:

| Threat ID | Name | Relevant when changing... |
|---|---|---|
| **T1** | Credential stuffing / brute force | Keycloak config, login flow |
| **T2** | RBAC escalation | Middleware, handler route registration, role checks |
| **T3** | Campus isolation breach | Repository queries, service campus_id handling |
| **T4** | Offline device theft | IndexedDB storage, encryption, session timeout |
| **T5** | Sync endpoint data injection | Sync push handler, validation |
| **T6** | Data exfiltration via reports | Report/export endpoints, CSV generation |
| **T7** | IDOR (Insecure Direct Object Reference) | Any endpoint with path params (`:id`) |
| **T8** | XSS (Cross-Site Scripting) | Frontend rendering of user-provided data |
| **T9** | CSRF (Cross-Site Request Forgery) | Mutation endpoints, token handling |
| **T10** | SQL injection | Repository queries, dynamic filters |
| **T11** | JWT manipulation | Token validation middleware |
| **T12** | Keycloak misconfiguration | Realm export, client settings |

### Step 3: OWASP Top 10 Check

For each relevant OWASP category, verify mitigations are in place:

#### A01: Broken Access Control
- [ ] Every endpoint has RBAC middleware (no unprotected routes under `/api/v1`)
- [ ] Role requirements match `docs/16-iam-and-access-control.md` permission matrix
- [ ] Path parameter IDs are validated (UUID format) before database queries
- [ ] Campus ID comes from JWT claims, never from request body/params
- [ ] Users cannot access resources from other campuses

Verification:
```go
// Check route registration in cmd/server/main.go or router file
// Every route under /api/v1 must have middleware.Authenticate
// Mutation routes must have middleware.RequireRole(...)
```

#### A02: Cryptographic Failures
- [ ] No secrets in source code (check for hardcoded keys, passwords, tokens)
- [ ] Keycloak client secret loaded from environment variable
- [ ] IndexedDB sensitive fields encrypted with AES-256-GCM
- [ ] HTTPS enforced in production (TLS termination at reverse proxy)

#### A03: Injection
- [ ] All SQL queries use parameterized queries (`$1`, `$2`, never string concatenation)
- [ ] User input is validated before reaching the repository layer
- [ ] No `fmt.Sprintf` used to build SQL queries
- [ ] JSONB fields are marshaled via `json.Marshal`, not string concatenation

Verification:
```bash
# Search for potential SQL injection patterns in repository files
grep -rn "fmt.Sprintf.*SELECT\|fmt.Sprintf.*INSERT\|fmt.Sprintf.*UPDATE\|fmt.Sprintf.*DELETE" backend/internal/repository/
# This should return zero results
```

#### A04: Insecure Design
- [ ] Business logic enforces authorization (service layer checks, not just middleware)
- [ ] Audit logging present for all data mutations
- [ ] Rate limiting on sensitive endpoints (login, sync push, export)

#### A05: Security Misconfiguration
- [ ] Keycloak realm settings follow `docs/20-keycloak-configuration.md`
- [ ] CORS policy restricts allowed origins
- [ ] Error responses do not leak stack traces or internal details
- [ ] Debug mode disabled in production configuration

#### A06: Vulnerable and Outdated Components
- [ ] Go dependencies are up to date (`go list -m -u all`)
- [ ] npm dependencies checked for known vulnerabilities (`npm audit`)

#### A07: Identification and Authentication Failures
- [ ] Authentication delegated entirely to Keycloak (no custom login endpoints)
- [ ] JWT validation uses JWKS (not hardcoded public key)
- [ ] Token expiration is checked (`exp` claim)
- [ ] Audience claim (`aud`) is validated

#### A08: Software and Data Integrity Failures
- [ ] Sync push validates all fields server-side (same rules as direct API)
- [ ] Audit log table has no UPDATE or DELETE operations
- [ ] Migration files are reviewed for schema integrity

#### A09: Security Logging and Monitoring Failures
- [ ] All data mutations create audit log entries
- [ ] Authentication failures are logged (via Keycloak event logging)
- [ ] 403 (Forbidden) responses are logged with user context
- [ ] Logs contain NO PII (no names, CPFs, emails, phone numbers)

#### A10: Server-Side Request Forgery (SSRF)
- [ ] No endpoints accept user-provided URLs for server-side fetching
- [ ] Keycloak Admin API calls use configured base URL only

### Step 4: Keycloak-Specific Checks

Reference: `docs/16-iam-and-access-control.md`, `docs/20-keycloak-configuration.md`

- [ ] Token validation uses `coreos/go-oidc` with Keycloak's JWKS endpoint
- [ ] No custom token issuance (all tokens come from Keycloak)
- [ ] No backdoor endpoints that accept raw credentials
- [ ] Keycloak client secret is NOT in source code or Docker images
- [ ] Frontend uses `keycloak-js` for the OIDC Authorization Code Flow with PKCE
- [ ] Realm export (`keycloak/realm-export.json`) does not contain secrets
- [ ] Custom protocol mappers (campus_id, person_id) are correctly configured
- [ ] Brute-force protection is enabled (10 failures, 15-minute lockout)
- [ ] MFA is configured for ADMIN and COORDINATOR roles

### Step 5: RBAC Verification

Reference: `docs/16-iam-and-access-control.md` permission matrix

For each endpoint in the change set, verify the role requirement matches the docs:

| Endpoint pattern | Required role | Verify in middleware |
|---|---|---|
| `POST /persons` | All authenticated | `middleware.Authenticate` |
| `PUT /persons/:id` | Secretary+ | `middleware.RequireRole("SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN")` |
| `GET /persons/:id/history` | Professional+ | `middleware.RequireRole("PROFESSIONAL", "COORDINATOR", "ADMIN")` |
| `POST /attendances` | Secretary+ | `middleware.RequireRole("SECRETARY", ...)` |
| `PATCH /attendances/:id/transition` | Professional+ | `middleware.RequireRole("PROFESSIONAL", ...)` |
| `GET /reports/*` | Coordinator+ | `middleware.RequireRole("COORDINATOR", "ADMIN")` |
| `POST /users` | Admin | `middleware.RequireRole("ADMIN")` |
| `GET /audit/logs` | Admin | `middleware.RequireRole("ADMIN")` |

### Step 6: Data Access and Campus Scoping

- [ ] Every repository query includes `WHERE campus_id = $N` with the value from JWT claims
- [ ] The campus_id is extracted from the middleware context, not from the request body
- [ ] List endpoints filter by campus_id (no cross-campus data leakage in paginated results)
- [ ] Update/delete operations verify campus_id ownership before modifying

Test approach:
```go
// Repository integration test
// Insert record with campus_id = A
// Attempt to read/update with campus_id = B
// Assert: operation returns not found / no rows affected
```

### Step 7: PII Protection

- [ ] Log statements (`slog.Info`, `slog.Error`, etc.) do not include: full_name, document_number, email, phone, address, health data
- [ ] Error responses to the client do not include PII
- [ ] Stack traces are not exposed in production error responses
- [ ] CSV/report exports are role-restricted and audit-logged
- [ ] IndexedDB stores PII only in encrypted form

Search for PII leakage:
```bash
# Check Go log statements for PII field names
grep -rn "slog\.\(Info\|Error\|Warn\|Debug\)" backend/internal/ | grep -i "name\|email\|phone\|document\|cpf\|address"
# This should return zero results (or only field names in struct definitions, not log calls)
```

### Step 8: Offline Security

Reference: `docs/18-threat-model.md` (T4: Offline Device Theft)

- [ ] Sensitive fields are encrypted before IndexedDB storage (AES-256-GCM)
- [ ] Encryption key is derived per-user (not a global key)
- [ ] Session timeout after 30 minutes of inactivity
- [ ] Logout clears all IndexedDB data
- [ ] "Clear local data" button available on the login screen
- [ ] Offline token TTL is 14 days maximum
- [ ] Sync queue entries do not contain PII in plaintext

### Step 9: Fill Security Review Report

Create a report using `templates/security-review-report.md`. Include:
- All findings (even low severity)
- OWASP check results
- Chesed-specific check results
- Verdict: PASS / PASS WITH FINDINGS / FAIL

---

## Quick Reference: Common Pitfalls

1. **Missing RBAC on new route**: Every new route must have `middleware.RequireRole(...)` or at minimum `middleware.Authenticate`
2. **Campus ID from request body**: Campus ID must come from JWT claims via middleware context, never from user input
3. **PII in error messages**: `fmt.Errorf("person %s not found", person.FullName)` leaks PII. Use IDs only: `fmt.Errorf("person %s not found", personID)`
4. **SQL injection via dynamic filters**: Building WHERE clauses with string concatenation for search/filter parameters
5. **Unencrypted PII in IndexedDB**: Storing document_number or health data without encryption
6. **Missing audit log**: Every POST, PUT, PATCH, DELETE must create an audit log entry
7. **Hardcoded Keycloak secret**: Client secrets must come from environment variables

---

## Checklist

- [ ] Scope defined (changed files, endpoints, data flows)
- [ ] Threat model scenarios mapped (T1-T12)
- [ ] OWASP Top 10 checks completed (A01-A10)
- [ ] Keycloak integration verified
- [ ] RBAC matches permission matrix
- [ ] Campus scoping enforced in all queries
- [ ] No PII in logs or error responses
- [ ] Offline data encrypted
- [ ] Audit logging present for mutations
- [ ] Security review report filled
