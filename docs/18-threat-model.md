# 18 - Threat Model

## System Context

Chesed is a web application handling sensitive personal data (health records, identification documents, social vulnerability assessments) of people in vulnerable situations. The system operates across multiple church campuses, supports offline field operations via mobile devices, and must comply with LGPD.

---

## Assets to Protect

| Asset | Classification | Impact if Compromised |
|-------|---------------|----------------------|
| Person PII (name, CPF, address, phone) | High | Identity theft; LGPD violation; reputational damage |
| Health and social vulnerability data | Critical | Discrimination; exploitation of vulnerable people |
| Consent records and signatures | High | Legal liability; LGPD violation |
| Authentication credentials | Critical | Unauthorized system access; data breach |
| Audit logs | High | Loss of compliance evidence; cover-up of breaches |
| Attendance records | Medium | Service history tampering; privacy violation |
| Donation records | Medium | Financial fraud; trust erosion |
| Offline device data | High | Unencrypted PII on lost/stolen devices |
| Keycloak realm configuration | High | Identity infrastructure compromise; authentication bypass |
| Keycloak admin credentials | Critical | Complete identity takeover; impersonation of any user |

---

## Threat Actors

| Actor | Motivation | Capability | Likelihood |
|-------|-----------|-----------|------------|
| **Opportunistic attacker** | Data theft, credential stuffing | Low-medium; automated tools | Medium |
| **Malicious insider** | Data access for personal use, grudge | Has valid credentials; knows the system | Low-medium |
| **Lost/stolen device** | N/A (accidental) | Physical access to offline data | Medium (field events) |
| **Social engineering** | Trick volunteer into revealing data | Targets non-technical users | Medium |
| **Targeted attacker** | Specific person's data (e.g., stalking) | Medium; willing to invest effort | Low |
| **Regulatory auditor** | Compliance verification | Full cooperation expected | High (eventually) |

---

## Attack Surface

### External Attack Surface

```
Internet
  │
  ├── HTTPS API (Go server)
  │   ├── Authentication endpoints (login, refresh, password reset)
  │   ├── Data CRUD endpoints (persons, attendances, campaigns)
  │   ├── Sync endpoints (push/pull offline data)
  │   ├── Report/export endpoints
  │   └── File upload/download endpoints
  │
  ├── Keycloak login page (OIDC endpoints)
  │
  ├── PWA static assets (CDN)
  │   └── Service Worker + cached app shell
  │
  └── DNS + TLS termination (reverse proxy)
```

### Internal Attack Surface

```
Authenticated user
  ├── RBAC bypass (access data beyond role)
  ├── Campus isolation bypass (access other campus data)
  ├── Data exfiltration via export/reports
  └── Sync endpoint abuse (inject/modify offline data)

Keycloak Admin Console
  ├── Realm and client configuration changes
  └── User and role management

Keycloak Admin API
  ├── Programmatic access to identity management
  └── Token introspection and revocation

Mobile device
  ├── IndexedDB with PII (unencrypted by default)
  ├── Cached API responses
  └── Stored JWT tokens
```

---

## Threat Scenarios and Mitigations

### T1: Credential Stuffing / Brute Force Login

**Scenario**: Attacker uses stolen credential lists or brute-force against the Keycloak login page.

| Aspect | Detail |
|--------|--------|
| Target | Keycloak login page (OIDC authorization endpoint) |
| Impact | Unauthorized access to user accounts |
| Likelihood | Medium |
| Risk | **Medium** (Keycloak built-in protections significantly reduce risk) |

**Mitigations**:
- Keycloak brute-force detection (built-in, configurable: 10 failures, 15-minute lockout)
- MFA for admin accounts (blocks credential stuffing even with correct password)
- Cloudflare WAF rate limiting on Keycloak login URL
- Keycloak account lockout is automatic, no custom code required
- Generic error messages (no user enumeration)
- Audit logging of all login attempts (Keycloak event logging: IP, user agent)

---

### T2: RBAC Escalation

**Scenario**: A volunteer or secretary discovers API endpoints and attempts to access admin functionality directly.

| Aspect | Detail |
|--------|--------|
| Target | Any restricted API endpoint |
| Impact | Data access beyond authorized scope |
| Likelihood | Low-Medium |

**Mitigations**:
- RBAC middleware on every handler (no endpoint without role check)
- Role checked from JWT claims (not from request body)
- Integration tests verify every role/endpoint combination
- Audit logging of 403 responses

---

### T3: Campus Isolation Breach

**Scenario**: A user from Campus A attempts to view or modify data belonging to Campus B.

| Aspect | Detail |
|--------|--------|
| Target | Any data endpoint with campus_id filter |
| Impact | Privacy violation; data exposure across organizational boundaries |
| Likelihood | Low |

**Mitigations**:
- All queries include `campus_id` filter from JWT (not from user input)
- Repository layer enforces campus filter at the SQL level
- Integration tests with multi-campus test data
- Phase 3: PostgreSQL Row-Level Security as additional layer
- Audit logging of cross-campus access attempts

---

### T4: Offline Device Theft / Loss

**Scenario**: A volunteer's phone is lost or stolen during a field event. The device contains offline data in IndexedDB.

| Aspect | Detail |
|--------|--------|
| Target | IndexedDB on mobile device |
| Impact | PII exposure of beneficiaries |
| Likelihood | Medium (field events in public spaces) |

**Mitigations**:
- Encrypt sensitive fields in IndexedDB using Web Crypto API
- Session timeout: require re-authentication after 30 minutes of inactivity
- "Clear local data" button accessible from login screen
- Automatic data wipe on logout
- Minimize data cached offline (only recent records, not full database)
- Device lock screen (user education; not enforced by app)

---

### T5: Sync Endpoint Data Injection

**Scenario**: An attacker crafts a malicious sync push payload to inject or modify data, bypassing normal validation.

| Aspect | Detail |
|--------|--------|
| Target | `POST /api/v1/sync/push` |
| Impact | Data integrity compromise; fake records |
| Likelihood | Low |

**Mitigations**:
- Full server-side validation on all sync payloads (same rules as direct API)
- Sync records must reference valid campus_id matching the user's JWT
- UUID collision check (sync_id idempotency)
- Rate limiting on sync endpoint
- Audit logging of all sync operations

---

### T6: Data Exfiltration via Reports/Export

**Scenario**: A coordinator exports large datasets to CSV, potentially exfiltrating sensitive data.

| Aspect | Detail |
|--------|--------|
| Target | `GET /api/v1/reports/.../export` |
| Impact | Bulk PII exfiltration |
| Likelihood | Low-Medium |

**Mitigations**:
- Export endpoints restricted to Coordinator+ role
- Every export logged in audit trail (who, when, what parameters)
- Export file does not include Critical-tier data by default (opt-in with additional audit)
- Row count limits on exports (e.g., max 10,000 rows per request)
- Future: anomaly detection on export frequency

---

### T7: JWT Token Theft (XSS)

**Scenario**: An XSS vulnerability allows an attacker to steal JWT tokens from the browser.

| Aspect | Detail |
|--------|--------|
| Target | Client-side token storage |
| Impact | Session hijacking |
| Likelihood | Low (React auto-escapes; CSP headers) |

**Mitigations**:
- Keycloak access tokens are short-lived (15 minutes)
- keycloak-js adapter can use `response_mode=fragment` to avoid tokens in URL
- Store tokens in httpOnly cookies (inaccessible to JavaScript)
- Content Security Policy (CSP) headers to prevent inline scripts
- React auto-escapes all rendered content
- Token binding: Keycloak supports DPoP (Demonstrating Proof of Possession) for token binding (Phase 3)
- Keycloak supports token revocation via Admin API for compromised tokens

---

### T8: Social Engineering of Volunteers

**Scenario**: Someone impersonates an NGO coordinator and asks a volunteer to share login credentials or look up a specific person's data.

| Aspect | Detail |
|--------|--------|
| Target | Non-technical volunteers |
| Impact | Unauthorized data access |
| Likelihood | Medium |

**Mitigations**:
- Role-based data visibility (volunteers see minimal data)
- Audit trail shows who accessed what
- Training materials for volunteers (operational, not in the system)
- No credential sharing; each volunteer has their own account
- Session monitoring for unusual access patterns (Phase 3)

---

### T9: Database Compromise

**Scenario**: Attacker gains access to the PostgreSQL database directly (e.g., through exposed credentials, misconfigured network).

| Aspect | Detail |
|--------|--------|
| Target | PostgreSQL server |
| Impact | Complete data breach |
| Likelihood | Low |

**Mitigations**:
- Database not exposed to public internet (private network only)
- Strong database passwords, rotated periodically
- Encrypted storage volumes
- Network security groups / firewall rules
- Database access requires TLS
- Regular backup verification
- Principle of least privilege for database users (app user has limited grants)

---

### T10: Keycloak Server Compromise

**Scenario**: Attacker gains access to Keycloak admin console or Keycloak database.

| Aspect | Detail |
|--------|--------|
| Target | Keycloak admin console / Keycloak database |
| Impact | Critical — complete identity compromise; ability to impersonate any user, create admin accounts, access all data |
| Likelihood | Low |
| Risk | **Medium** |

**Mitigations**:
- Keycloak admin credentials stored in secrets manager, rotated every 6 months
- Keycloak admin console restricted to internal network (not exposed to public internet)
- Keycloak deployed on private network, accessible only via reverse proxy for user-facing flows
- Keycloak database credentials separate from application database credentials
- Keycloak admin audit events monitored and alerted
- Regular Keycloak version updates (security patches within 7 days of release)
- Principle of least privilege for Keycloak service accounts

---

### T11: Supply Chain Attack

**Scenario**: Compromised dependency in Go modules, npm packages, Docker base images, or Keycloak extensions.

| Aspect | Detail |
|--------|--------|
| Target | Application dependencies and container images |
| Impact | High — code execution, data exfiltration, backdoor installation |
| Likelihood | Low |
| Risk | **Medium** |

**Mitigations**:
- Lock file verification (`go.sum`, `package-lock.json`) enforced in CI
- Docker image pinning by digest (not just tag)
- Trivy scanning on all container images (including Keycloak)
- Dependabot/Renovate for automated dependency update PRs
- Minimal dependency footprint (Go stdlib-first philosophy)
- SBOM (Software Bill of Materials) generation for Docker images (Phase 2)
- No custom Keycloak SPIs or extensions unless security-reviewed
- Approved dependency list maintained in docs/19-secure-development-standard.md

---

### T12: DDoS / Service Availability Attack

**Scenario**: Volumetric or application-layer attack overwhelms the Chesed application or Keycloak during a social action event.

| Aspect | Detail |
|--------|--------|
| Target | Chesed API / Keycloak OIDC endpoints |
| Impact | High — service unavailable during critical field operations |
| Likelihood | Low (NGO is not a typical high-value DDoS target) |
| Risk | **Low-Medium** |

**Mitigations**:
- Cloudflare free tier provides basic DDoS protection and CDN
- Keycloak and Go API behind Cloudflare proxy
- Rate limiting at reverse proxy level (100 req/min per IP)
- Offline-first PWA continues to function during outage (key mitigation)
- Keycloak offline tokens (14-day TTL) allow continued authenticated offline work
- Application designed to degrade gracefully (offline mode is a first-class feature)

---

### T13: Malicious File Upload (Sprint 6 — document/consent surface)

**Scenario**: An authenticated user uploads a crafted file (malware, polyglot,
oversized payload, or a spoofed content type) through the document upload
endpoints, aiming to store hostile content, exhaust storage, or have the file
executed/rendered by another user's browser on download.

| Aspect | Detail |
|--------|--------|
| Target | `POST /persons/:id/documents`, `POST /attendances/:id/documents`, object storage |
| Impact | Medium-High — stored malware distribution, storage exhaustion, XSS via rendered files |
| Likelihood | Low-Medium (authenticated, role-gated surface) |
| Risk | **Medium** |

**Mitigations**:
- Content-type whitelist (`application/pdf`, `image/jpeg`, `image/png`) verified
  by **magic bytes** server-side; the client Content-Type header is never trusted (docs/19)
- 10MB size limit enforced with `http.MaxBytesReader` before parsing
- Stored under UUID-based object keys; original filename kept only as metadata
  (no path traversal / no filename-driven execution)
- Files live in object storage, never on the application filesystem; downloads
  only via time-limited presigned URLs (15 min) — the API never streams bytes
- Uploads are role-gated (Secretary+/Professional+), campus-scoped, and audited
- Virus scanning is documented as a future control (docs/13); risk accepted for
  Phase 2 given the authenticated, low-volume surface

---

## Risk Summary Matrix

| Threat | Likelihood | Impact | Risk | Primary Mitigation |
|--------|-----------|--------|------|-------------------|
| T1: Brute force login | Medium | High | **Medium** | Keycloak brute-force detection + MFA |
| T2: RBAC escalation | Low-Medium | High | **Medium** | Middleware enforcement |
| T3: Campus isolation breach | Low | High | **Medium** | JWT-based campus filter |
| T4: Device theft | Medium | High | **High** | IndexedDB encryption + session timeout |
| T5: Sync injection | Low | Medium | **Low** | Server-side validation |
| T6: Data exfiltration | Low-Medium | Medium | **Medium** | Audit logging + access control |
| T7: XSS token theft | Low | High | **Medium** | httpOnly cookies + CSP |
| T8: Social engineering | Medium | Medium | **Medium** | Role-based data minimization |
| T9: Database compromise | Low | Critical | **Medium** | Network isolation + encryption |
| T10: Keycloak compromise | Low | Critical | **Medium** | Network isolation + secrets management |
| T11: Supply chain attack | Low | High | **Medium** | Lock files + image scanning + minimal deps |
| T12: DDoS / availability | Low | High | **Low-Medium** | Cloudflare + offline-first PWA |
| T13: Malicious file upload | Low-Medium | Medium-High | **Medium** | Magic-byte whitelist + size limit + presigned-only access |

---

## Review Schedule

- **Quarterly**: Review threat model for new threats based on system changes
- **Per phase**: Update when new features (donations, file upload, multi-region) change the attack surface
- **After incidents**: Immediate review and update after any security event
- **After Keycloak version upgrades**: Review for new attack vectors or deprecated security features
