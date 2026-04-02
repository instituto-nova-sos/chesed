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

Mobile device
  ├── IndexedDB with PII (unencrypted by default)
  ├── Cached API responses
  └── Stored JWT tokens
```

---

## Threat Scenarios and Mitigations

### T1: Credential Stuffing / Brute Force Login

**Scenario**: Attacker uses stolen credential lists or brute-force against the login endpoint.

| Aspect | Detail |
|--------|--------|
| Target | `POST /api/v1/auth/login` |
| Impact | Unauthorized access to user accounts |
| Likelihood | Medium |

**Mitigations**:
- Account lockout after 10 failed attempts (15-minute cooldown)
- Rate limiting: 20 login attempts per IP per hour
- bcrypt with cost factor 12 (slow hash)
- Generic error messages (no user enumeration)
- Audit logging of all login attempts (IP, user agent)

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
- Store tokens in httpOnly cookies (inaccessible to JavaScript)
- Content Security Policy (CSP) headers to prevent inline scripts
- React auto-escapes all rendered content
- Short access token TTL (15 minutes limits exposure window)
- Token binding to IP/user-agent (optional, Phase 2)

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

## Risk Summary Matrix

| Threat | Likelihood | Impact | Risk | Primary Mitigation |
|--------|-----------|--------|------|-------------------|
| T1: Brute force login | Medium | High | **High** | Lockout + rate limiting |
| T2: RBAC escalation | Low-Medium | High | **Medium** | Middleware enforcement |
| T3: Campus isolation breach | Low | High | **Medium** | JWT-based campus filter |
| T4: Device theft | Medium | High | **High** | IndexedDB encryption + session timeout |
| T5: Sync injection | Low | Medium | **Low** | Server-side validation |
| T6: Data exfiltration | Low-Medium | Medium | **Medium** | Audit logging + access control |
| T7: XSS token theft | Low | High | **Medium** | httpOnly cookies + CSP |
| T8: Social engineering | Medium | Medium | **Medium** | Role-based data minimization |
| T9: Database compromise | Low | Critical | **Medium** | Network isolation + encryption |

---

## Review Schedule

- **Quarterly**: Review threat model for new threats based on system changes
- **Per phase**: Update when new features (donations, file upload, multi-region) change the attack surface
- **After incidents**: Immediate review and update after any security event
