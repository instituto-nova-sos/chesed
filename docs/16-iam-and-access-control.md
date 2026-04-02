# 16 - IAM and Access Control

## Overview

Chesed implements Identity and Access Management through JWT-based authentication with Role-Based Access Control (RBAC), scoped by campus. This document defines the complete IAM model.

---

## Identity Model

### Person → User Relationship

Not every person in the system is a user. The identity model is:

```
Person (anyone known to the system)
  └── User account (optional — only for people who log in)
       ├── access_profile (role)
       └── campus_id (data scope)
```

- A **Person** is registered when they interact with the NGO as beneficiary, volunteer, or professional
- A **User** account is created only when that person needs system access
- One person → at most one user account
- User accounts are always linked to a person record

### Authentication

| Method | Description | Endpoint |
|--------|------------|----------|
| Email + Password | Primary authentication method | `POST /api/v1/auth/login` |
| JWT Access Token | Short-lived bearer token for API access | 15-minute TTL |
| JWT Refresh Token | Long-lived token for session continuity | 7-day TTL, stored hashed in DB |
| Password Reset | Email-based secure token | `POST /api/v1/auth/forgot-password` |

### Token Claims

The JWT access token contains:

```json
{
  "sub": "<user_id>",
  "person_id": "<person_id>",
  "email": "<email>",
  "profile": "<access_profile>",
  "campus_id": "<campus_id>",
  "iat": 1711987200,
  "exp": 1711988100
}
```

These claims are extracted by the auth middleware and made available to all handlers for authorization and data scoping.

---

## Access Profiles (Roles)

### Role Hierarchy

```
ADMIN
  └── COORDINATOR
       └── PROFESSIONAL
            └── SECRETARY
                 └── VOLUNTEER
```

Each higher role implicitly has all permissions of lower roles, plus additional capabilities.

### Role Definitions

| Role | Description | Typical User |
|------|------------|-------------|
| **ADMIN** | Full system access; manages users, permissions, audit logs, and all campuses | IT administrator, organization leadership |
| **COORDINATOR** | Manages campaigns, teams, reports; full operational access within campus | Team leader, department head |
| **PROFESSIONAL** | Records and manages own attendance; views assigned persons | Doctor, lawyer, social worker, psychologist |
| **SECRETARY** | Registers persons, creates triages, manages scheduling | Administrative staff |
| **VOLUNTEER** | Creates triages and basic data entry during events | Community volunteer |

### Permission Matrix

| Resource | VOLUNTEER | SECRETARY | PROFESSIONAL | COORDINATOR | ADMIN |
|----------|-----------|-----------|-------------|-------------|-------|
| **Person** |
| Create person | Yes | Yes | Yes | Yes | Yes |
| View person (own campus) | Name only | Full | Full | Full | Full + cross-campus |
| Edit person | No | Yes | No | Yes | Yes |
| Delete/anonymize person | No | No | No | No | Yes |
| **Triage** |
| Create triage | Yes | Yes | Yes | Yes | Yes |
| View triages | Own only | Campus | Campus | Campus | All |
| **Attendance** |
| Create attendance | No | Yes | Yes | Yes | Yes |
| Edit attendance | No | No | Own only | Campus | All |
| View attendance history | No | Campus | Own + assigned | Campus | All |
| Transition status | No | No | Own only | Campus | All |
| **Campaign** |
| Create/edit campaign | No | No | No | Yes | Yes |
| View campaigns | Read-only | Read-only | Read-only | Full | Full |
| Manage team assignment | No | No | No | Yes | Yes |
| **Reports** |
| View reports | No | No | No | Yes | Yes |
| Export data (CSV) | No | No | No | Yes | Yes |
| View dashboard | No | Basic | Basic | Full | Full |
| **Donations** |
| Create donation | No | Yes | No | Yes | Yes |
| View donations | No | Own entries | No | Campus | All |
| Issue receipt | No | No | No | Yes | Yes |
| **Users** |
| Create/edit users | No | No | No | No | Yes |
| Deactivate users | No | No | No | No | Yes |
| Assign roles | No | No | No | No | Yes |
| **Audit** |
| View audit logs | No | No | No | No | Yes |
| **Consent** |
| Create consent | Yes | Yes | Yes | Yes | Yes |
| View consents | No | Campus | Own assigned | Campus | All |
| Revoke consent | No | No | No | No | Yes |

---

## Campus Data Isolation

### How It Works

1. Every user is assigned to a `campus_id` at account creation
2. The JWT token includes `campus_id` in its claims
3. The auth middleware extracts `campus_id` and injects it into the request context
4. Every repository query includes `WHERE campus_id = $campus_id`
5. Admin users can optionally pass `?campus_id=<uuid>` to query other campuses

### Implementation Pattern (Go)

```go
// Middleware extracts campus from JWT
func CampusMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims := auth.ClaimsFromContext(r.Context())
        ctx := context.WithValue(r.Context(), campusKey, claims.CampusID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Repository always filters by campus
func (r *PersonRepo) List(ctx context.Context, filter PersonFilter) ([]Person, error) {
    campusID := CampusFromContext(ctx)
    query := `SELECT * FROM person WHERE campus_id = $1 AND ...`
    // ...
}
```

### Cross-Campus Access

- Only ADMIN profile can query across campuses
- Cross-campus queries require explicit `campus_id` parameter
- All cross-campus access is logged in audit trail with `cross_campus=true` flag

---

## Session Management

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| Access token TTL | 15 minutes | Short-lived to limit exposure if stolen |
| Refresh token TTL | 7 days | Balances security with UX for multi-day events |
| Concurrent sessions | Allowed | Users may access from phone + desktop |
| Token storage | httpOnly cookie (preferred) or encrypted IndexedDB | Prevents XSS token theft |
| Refresh token storage | Hashed in `refresh_token` database table | Enables server-side revocation |

### Session Lifecycle

```
Login → Access token + Refresh token issued
  │
  ├── API call → Access token in Authorization header
  │   └── Token valid? → Process request
  │   └── Token expired? → Client uses refresh token
  │       └── Refresh valid? → New access token issued
  │       └── Refresh expired? → Redirect to login
  │
  └── Logout → Refresh token revoked in DB
```

### Token Revocation

- **Per-session**: Revoke specific refresh token on logout
- **Per-user**: Revoke all refresh tokens (force re-login on all devices)
- **Global**: Emergency revocation by changing JWT signing key (affects all users)

---

## Account Security

### Password Policy
- Minimum 8 characters
- At least 1 letter and 1 number
- Passwords hashed with bcrypt (cost factor 12)
- No password reuse check (MVP; add in Phase 2)

### Account Lockout
- After 10 consecutive failed login attempts: lock for 15 minutes
- Lockout events logged in audit trail
- Admin can unlock accounts manually

### Password Recovery
1. User requests reset via `POST /api/v1/auth/forgot-password`
2. System generates secure random token (32 bytes, hex-encoded)
3. Token stored hashed in database with 1-hour TTL
4. Email sent with reset link (Phase 2; MVP provides admin-initiated reset)
5. User submits new password via `POST /api/v1/auth/reset-password`
6. All existing refresh tokens for the user are revoked

---

## Audit Integration

Every authentication and authorization event is logged:

| Event | Audit Action | Details |
|-------|-------------|---------|
| Successful login | `LOGIN` | User, IP, user agent |
| Failed login | `LOGIN` (success=false) | Email attempted, IP, user agent |
| Token refresh | Not logged | High frequency; would pollute audit log |
| Logout | `LOGOUT` | User, session |
| Access denied (wrong role) | `READ` (success=false) | User, resource, required role |
| Cross-campus access | `READ` | User, target campus |
| Permission change | `PERMISSION_CHANGE` | Admin, target user, old/new role |
| Account lockout | `LOGIN` (success=false) | User, lockout duration |
| Password change | `UPDATE` | User (no password values logged) |

---

## Future Considerations (Phase 3+)

- **Multi-factor authentication (MFA)**: TOTP-based 2FA for admin accounts
- **OAuth/OIDC**: Social login for volunteer accounts (Google, Microsoft)
- **API keys**: For WordPress portal and external integrations
- **Fine-grained permissions**: Permission overrides per user beyond role defaults
- **Temporary access**: Time-limited access for event-specific volunteers
