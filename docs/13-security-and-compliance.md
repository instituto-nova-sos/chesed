# 13 - Security and Compliance

## LGPD Compliance

### Overview

Brazil's Lei Geral de Proteção de Dados (LGPD - Law 13.709/2018) governs personal data processing. The system handles sensitive personal data of vulnerable populations, making compliance both a legal requirement and an ethical obligation.

### Legal Basis for Data Processing

| Data Category | Legal Basis (LGPD Art. 7) | Justification |
|---------------|--------------------------|---------------|
| Person basic data (name, document, phone) | Legitimate interest (Art. 7, IX) + Consent (Art. 7, I) | Necessary for NGO service delivery |
| Health data | Explicit consent (Art. 11, I) | Sensitive data requires specific consent |
| Social vulnerability data | Explicit consent (Art. 11, I) | Sensitive data |
| Image/photo data | Explicit consent (Art. 7, I) | Separate consent for image usage |
| Minor data | Guardian consent (Art. 14) | Children and adolescents require special protection |

### Data Subject Rights (LGPD Art. 18)

The system must support:

| Right | Implementation |
|-------|---------------|
| Access (Art. 18, II) | `GET /persons/:id` returns all stored data for a person |
| Correction (Art. 18, III) | `PUT /persons/:id` allows updating personal data |
| Anonymization (Art. 18, IV) | Anonymize or delete sensitive fields while preserving aggregate statistics |
| Deletion (Art. 18, VI) | Logical deletion + anonymization of PII; audit records retained with anonymized references |
| Information about sharing (Art. 18, VII) | System does not share data with third parties |
| Consent revocation (Art. 18, IX) | `PATCH /consents/:id/revoke` revokes active consent |
| Portability (Art. 18, V) | CSV export of person's complete data |

### Consent Management

```
Consent Lifecycle:
  1. Consent presented → Purpose explained (LGPD Art. 8)
  2. Person signs on device → Signature + timestamp captured
  3. Consent stored → Linked to person, type, version, purpose
  4. Consent revoked → Sensitive data anonymized; audit trail preserved
```

**Consent types:**
- `DATA_PROCESSING`: General personal data processing
- `IMAGE_USAGE`: Use of photos/images in publications
- `HEALTH_DATA`: Processing of health-related information
- `MINOR_GUARDIAN`: Guardian consent for minors

**Consent versioning**: When consent terms change, a new version is created. Existing consents remain valid under their original version. New interactions require consent under the current version.

---

## Sensitive Data Classification

### Classification Tiers

| Tier | Data Type | Examples | Protection Level |
|------|-----------|----------|-----------------|
| **Critical** | Health data, social vulnerability | Medical records, income, housing status | Encrypted at rest; explicit consent required; access logged; anonymizable |
| **High** | Personal identification | CPF, SSN, full name, address, phone | Encrypted at rest; access logged; anonymizable |
| **Medium** | Operational data | Attendance records, triage notes | Access controlled by RBAC; audit logged |
| **Low** | Reference data | Service types, campaigns, campus info | Standard access control |

Data classification must be enforced programmatically. Sensitive fields (CPF, health data, income) must have access restricted by role in the service layer, not just the UI.

Consent signatures and health data are classified as CRITICAL and must be encrypted at rest in both server storage and local IndexedDB.

### Data Handling Rules

1. **Critical and High tier data** must be:
   - Encrypted at rest in the database
   - Encrypted in transit (TLS)
   - Encrypted in local storage (IndexedDB) when browser supports Web Crypto API
   - Accessible only to users with explicit permission
   - Logged in the audit trail when accessed
   - Anonymizable upon consent revocation

2. **Audit log data** must be:
   - Append-only (no modifications or deletions)
   - Retained for minimum 5 years
   - Anonymized references when the source record is deleted (replace person name with hash)

---

## Access Logging

### What Gets Logged

| Action | Logged | Details Captured |
|--------|--------|-----------------|
| Login success | Yes | User, IP, user agent, timestamp |
| Login failure | Yes | Email attempted, IP, user agent, timestamp |
| View person record | Yes | User, person_id, timestamp |
| Edit person record | Yes | User, person_id, old values, new values |
| View attendance | Yes | User, attendance_id, timestamp |
| Create/edit attendance | Yes | User, attendance_id, changes |
| Export report | Yes | User, report type, parameters, timestamp |
| Permission change | Yes | User, target user, permissions changed |
| Sync push | Yes | User, device_id, record count |
| Consent creation | Yes | User, person_id, consent type |
| Consent revocation | Yes | User, person_id, consent type, reason |
| Data deletion/anonymization | Yes | User, person_id, fields affected |

### Audit Log Schema

See `10-data-model.md` for the `audit_log` table. Key fields:
- `user_id`: Who performed the action
- `action_type`: What was done (CREATE, READ, UPDATE, DELETE, LOGIN, LOGOUT, EXPORT, PERMISSION_CHANGE)
- `entity_type` + `entity_id`: What was affected
- `old_values` / `new_values`: JSONB diff of changes
- `ip_address`: Source IP (with proxy-aware extraction)
- `user_agent`: Browser/device identification
- `timestamp`: When it happened
- `success`: Whether the action succeeded

---

## Encryption

### In Transit
- TLS 1.3 enforced via reverse proxy (Caddy auto-HTTPS or Nginx with Let's Encrypt)
- HSTS header with 1-year max-age
- No HTTP endpoints (redirect to HTTPS)

### At Rest (Server)
- PostgreSQL: Enable Transparent Data Encryption (TDE) or use encrypted storage volumes
- Object storage (S3): Server-side encryption (SSE-S3 or SSE-KMS)
- Backup files: Encrypted with AES-256

### At Rest (Client)
- IndexedDB: Encrypt sensitive fields (person names, document numbers) using Web Crypto API
- Service Worker cache: App shell only; no PII in cache storage
- JWT tokens: Stored in httpOnly cookies (preferred) or encrypted in IndexedDB

### Key Management
- Database encryption keys managed by cloud provider KMS
- Keycloak realm keys: RS256 signing keys managed by Keycloak with automatic key rotation support. Go API fetches public keys from JWKS endpoint and caches them with TTL.
- Application secrets: Environment variables, never in code

---

## Data Retention Policy

| Data Category | Retention Period | After Expiry | Legal Basis |
|---------------|-----------------|--------------|-------------|
| Person records | 5 years after last activity | Anonymize (keep aggregate stats) | LGPD Art. 15-16 |
| Attendance records | 10 years | Archive (read-only) | Operational + compliance |
| Triage records | 10 years | Archive (read-only) | Linked to attendance |
| Assisted profile (sensitive) | 5 years after last activity | Anonymize | LGPD Art. 18 |
| Consent records | Indefinite | Never deleted | Legal proof of consent |
| Audit logs | 10 years minimum | Archive (read-only) | Compliance requirement |
| Donation records | 10 years | Archive (read-only) | Tax/accounting requirements |
| Campaign records | 5 years after completion | Archive | Operational |
| Login sessions | 90 days | Auto-purge expired sessions | Operational |
| File attachments | Until consent revocation + 30 days | Physical deletion from object storage | LGPD Art. 15-16 |
| Local device data | Until logout or manual clear | Full wipe on logout | Operational |

**Audit logs**: Audit log records must be tamper-resistant (append-only enforcement at database level via triggers or application-level controls).

**LGPD breach notification**: 2 business days to ANPD for incidents involving personal data that may cause significant risk or damage to data subjects (Lei 13.709/2018, Art. 48).

### Anonymization Rules for LGPD Erasure

When a data subject exercises their right to erasure (LGPD Art. 18, V):

1. Anonymize the `person` record: replace `full_name` with 'ANONYMIZED', clear `document_number`, `email`, `phone`, `photo_url`
2. Anonymize the `address` record: clear all fields except `campus_id` (for aggregate stats)
3. Anonymize the `assisted_profile`: clear all sensitive fields (family_composition, income_range, etc.)
4. Keep `attendance` and `triage` records with anonymized person reference for aggregate reporting
5. Keep `audit_log` entries (they reference user_id, not PII directly)
6. Revoke all active `consent` records and log the revocation
7. Log the erasure action in `audit_log` with action_type ANONYMIZE

Complete physical deletion is performed only when legally mandated. Anonymization preserves aggregate data integrity for reporting.

---

## RBAC Security Model

### Permission Matrix

| Resource | Volunteer | Secretary | Professional | Coordinator | Admin |
|----------|-----------|-----------|-------------|-------------|-------|
| Create person | Yes | Yes | Yes | Yes | Yes |
| View person (own campus) | Limited | Yes | Yes | Yes | Yes |
| Edit person | No | Yes | No | Yes | Yes |
| Create triage | Yes | Yes | Yes | Yes | Yes |
| Create attendance | No | Yes | Yes | Yes | Yes |
| Edit attendance | No | No | Own only | Yes | Yes |
| View reports | No | No | No | Yes | Yes |
| Export data | No | No | No | Yes | Yes |
| Manage users | No | No | No | No | Yes |
| View audit logs | No | No | No | No | Yes |
| Manage campaigns | No | No | No | Yes | Yes |
| Manage donations | No | Secretary | No | Yes | Yes |

### Session Security

- Access token TTL: 15 minutes (Keycloak realm setting)
- Refresh token TTL: 7 days (Keycloak realm setting)
- Offline token TTL: 14 days (for field workers with `offline_access` scope)
- SSO Session idle timeout: 30 minutes (Keycloak realm setting)
- MFA: TOTP-based 2FA enforced for ADMIN role via Keycloak conditional authentication flow
- Password policy: Minimum 8 characters, 1 letter, 1 number, password history (3), enforced by Keycloak
- Account lockout: Keycloak brute-force detection — 10 failures, 15-minute lockout (realm setting)
- Concurrent sessions: Allowed, configurable per-client in Keycloak
- Session revocation: Admin can terminate user sessions via Keycloak Admin Console or Admin API

---

## Infrastructure Security

### Network
- Reverse proxy handles TLS termination
- Backend API not directly exposed to internet
- Database accessible only from API server (private network)
- CORS: Explicit origin whitelist

### Application
- Input validation on all endpoints (struct tags + custom validators)
- SQL injection prevention: Parameterized queries (pgx)
- XSS prevention: React auto-escapes output; CSP headers
- CSRF: Not applicable (JWT-based API; no cookies for auth in API)
- Rate limiting: 100 requests/minute per user enforced at reverse proxy level; Keycloak has built-in brute-force protection for login
- File upload validation: Type checking, size limits (10MB), virus scanning (future)
- WAF: Cloudflare (free tier) in front of reverse proxy for DDoS protection and bot mitigation
- Security headers validation: CI pipeline verifies presence of HSTS, CSP, X-Content-Type-Options, X-Frame-Options headers

### Dependencies
- Minimal dependency footprint (Go stdlib-first approach)
- Dependabot or Renovate for automated vulnerability alerts
- No `npm audit` high/critical vulnerabilities in production builds

---

## Centralized Logging

- **Keycloak event logging**: All login/logout/token events logged by Keycloak. Configure Keycloak event listeners to forward events to the application's audit_log or to a centralized log aggregation service.
- **Keycloak Audit Integration (MVP)**: Login, logout, and authentication failure events are forwarded from Keycloak to the application's `audit_log` table by polling the Keycloak Admin Events API from a background Go routine. This approach (Option A) is used in MVP for simplicity. A custom Keycloak SPI Event Listener (Option B) may be implemented in Phase 2 for real-time event forwarding.
- **Application logging**: slog structured JSON logs. For production, ship logs to Grafana Loki (free tier) or similar for centralized log aggregation, search, and alerting.
- **Monitoring**: Grafana dashboards for application metrics, Keycloak metrics (login success/failure rates, token issuance), and infrastructure health.

---

## Multi-Region Compliance Considerations

| Region | Law | Key Requirements | Implementation |
|--------|-----|-----------------|----------------|
| Brazil | LGPD | Consent, data subject rights, breach notification | Primary compliance target |
| USA | CCPA (California) | Notice, opt-out, deletion | Covered by LGPD compliance (LGPD is more restrictive) |
| Europe | GDPR | Consent, DPO, data portability, right to erasure | Largely covered by LGPD compliance; may need DPO designation |

**Data residency**: Campus data should be stored in the region where the campus operates. For MVP, all data in a single region (Brazil). For Phase 3, consider region-specific database instances or schemas.

---

## Incident Response Plan

1. **Detection**: Audit log monitoring for anomalous access patterns
2. **Assessment**: Determine scope (what data, how many people affected)
3. **Containment**: Disable compromised accounts in Keycloak; terminate all active sessions via Keycloak Admin Console; the Go API immediately rejects tokens for disabled users on next JWKS cache refresh
4. **Notification**: LGPD requires notification to ANPD (Autoridade Nacional de Protecao de Dados) within 2 business days for incidents involving personal data that may cause significant risk or damage to data subjects (Lei 13.709/2018, Art. 48). Affected persons must also be notified.
5. **Remediation**: Fix vulnerability; rotate credentials; update security controls
6. **Post-mortem**: Document incident; update security policies; audit log review
