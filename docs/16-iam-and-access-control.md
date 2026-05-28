# 16. IAM and Access Control

## Overview

Chesed delegates all identity and credential management to **Keycloak**, an open-source Identity and Access Management (IAM) platform. The Go API acts as an OIDC Resource Server that validates Keycloak-issued JWTs. The React frontend uses the OIDC Authorization Code Flow with PKCE to authenticate users. Role-Based Access Control (RBAC) is enforced through Keycloak realm roles, and campus-level data isolation is driven by a custom Keycloak user attribute mapped into token claims.

This document is the single source of truth for authentication, authorization, session management, and account security in the Chesed platform.

---

## 1. IAM Provider Decision

### Why Keycloak

| Criterion | Keycloak | Auth0 | AWS Cognito | Custom JWT |
|-----------|----------|-------|-------------|------------|
| License cost | Free (Apache 2.0) | Free tier limited; paid per MAU | Free tier limited; paid per MAU | None (dev cost only) |
| Self-hosted | Yes | No (SaaS only) | No (AWS only) | Yes |
| OIDC / OAuth 2.0 | Full standard | Full standard | Partial | Must implement |
| MFA support | TOTP, WebAuthn, SMS | Yes | Yes | Must implement |
| Brute-force protection | Built-in | Built-in | Built-in | Must implement |
| Password policies | Configurable | Configurable | Configurable | Must implement |
| User federation (LDAP) | Yes | Enterprise only | No | No |
| Identity brokering (Google, Microsoft) | Yes | Yes | Yes | Must implement |
| Admin console | Built-in web UI | SaaS dashboard | AWS Console | Must build |
| Community & docs | Large, mature | Commercial | AWS docs | N/A |
| Vendor lock-in | None | High | High | None |
| Maintenance burden | Medium (self-hosted) | Low (managed) | Low (managed) | Very High |

**Decision**: Keycloak is selected because it is open-source, self-hosted (zero licensing cost), fully OIDC-compliant, and fits the budget constraints of an NGO. It eliminates the need to build and maintain authentication, password management, MFA, and brute-force protection from scratch.

### Deployment

- Keycloak runs as a Docker container alongside the application stack.
- A dedicated realm named `chesed` isolates all configuration.
- PostgreSQL is used as the Keycloak database (can share the same PostgreSQL instance with a separate database, or use a dedicated instance).

---

## 2. Identity Model

### Keycloak User to Application User Mapping

```
Keycloak User (source of truth for identity and credentials)
  │
  │  keycloak_subject_id (sub claim)
  │
  ▼
app_user (local projection — application-specific metadata)
  │
  │  person_id (FK, optional)
  │
  ▼
Person (domain entity — may or may not have a system account)
```

**Key principles:**

- **Keycloak is the source of truth** for identity, credentials, and authentication state. The application never stores passwords or manages login directly.
- **`app_user`** is a local projection table that maps a Keycloak subject (`sub` claim) to application-specific data: `person_id`, `campus_id`, `is_active`, and timestamps.
- **Not all persons have system accounts.** A Person is anyone known to the NGO (beneficiary, volunteer, professional). An `app_user` record exists only for people who need system access.
- **One Keycloak user maps to one `app_user`, which maps to at most one Person.**
- **No `keycloak_user_id` on the `person` table.** The link between a Keycloak identity and a Person is established exclusively through the `app_user` join table (`app_user.keycloak_subject_id` + `app_user.person_id`). This avoids duplicating identity references and keeps the Person entity independent of the IAM provider.

### app_user Table

```sql
CREATE TABLE app_user (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    keycloak_subject_id   VARCHAR(255) NOT NULL UNIQUE,
    person_id             UUID UNIQUE REFERENCES person(id),
    email                 VARCHAR(255) NOT NULL,
    campus_id             UUID NOT NULL REFERENCES campus(id),
    is_active             BOOLEAN NOT NULL DEFAULT TRUE,
    last_login            TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_app_user_keycloak_sub ON app_user(keycloak_subject_id);
CREATE INDEX idx_app_user_campus ON app_user(campus_id);
```

---

## 3. Authentication Architecture

### OIDC Authorization Code Flow with PKCE

The application uses the standard OIDC Authorization Code Flow with Proof Key for Code Exchange (PKCE), which is the recommended flow for public clients (SPAs).

```
┌──────────┐       ┌──────────────┐       ┌──────────┐
│  React   │       │   Keycloak   │       │  Go API  │
│ Frontend │       │    Server    │       │  Backend │
└────┬─────┘       └──────┬───────┘       └────┬─────┘
     │                     │                    │
     │  1. User clicks     │                    │
     │     "Login"         │                    │
     │                     │                    │
     │  2. Redirect to     │                    │
     │     Keycloak with   │                    │
     │     code_challenge  │                    │
     │ ──────────────────► │                    │
     │                     │                    │
     │  3. User enters     │                    │
     │     credentials     │                    │
     │     (+ MFA if       │                    │
     │      required)      │                    │
     │                     │                    │
     │  4. Redirect back   │                    │
     │     with auth code  │                    │
     │ ◄────────────────── │                    │
     │                     │                    │
     │  5. Exchange code   │                    │
     │     + code_verifier │                    │
     │     for tokens      │                    │
     │ ──────────────────► │                    │
     │                     │                    │
     │  6. Receive         │                    │
     │     access_token +  │                    │
     │     refresh_token   │                    │
     │ ◄────────────────── │                    │
     │                     │                    │
     │  7. API request with│                    │
     │     Authorization:  │                    │
     │     Bearer <token>  │                    │
     │ ─────────────────────────────────────► │
     │                     │                    │
     │                     │  8. Validate JWT   │
     │                     │     via JWKS       │
     │                     │ ◄────────────────  │
     │                     │ ────────────────►  │
     │                     │                    │
     │  9. API response    │                    │
     │ ◄───────────────────────────────────── │
     │                     │                    │
```

### Responsibility Boundaries

| Concern | Owner | Details |
|---------|-------|---------|
| Login page / UI | Keycloak | Themed to match Chesed branding |
| Credential validation | Keycloak | Email + password verification |
| Password hashing | Keycloak | bcrypt by default |
| MFA enforcement | Keycloak | Conditional authentication flows |
| Brute-force protection | Keycloak | Account lockout after failed attempts |
| Password reset | Keycloak | Built-in "forgot password" email flow |
| Token issuance | Keycloak | JWT access tokens + opaque refresh tokens |
| Token validation | Go API | JWKS-based signature verification |
| Authorization (RBAC) | Go API | Role extraction from token claims |
| Campus data scoping | Go API | campus_id extraction from token claims |
| Audit logging (business) | Go API | All data mutations logged |
| Audit logging (auth events) | Keycloak | Login, logout, failed attempts |

**The Go API NEVER handles credentials.** It does not have login, register, forgot-password, or reset-password endpoints. All credential operations go through Keycloak directly.

---

## 4. Token Strategy

### Token Types

| Token | Issuer | Format | TTL | Storage | Purpose |
|-------|--------|--------|-----|---------|---------|
| Access token | Keycloak | JWT (signed RS256) | 15 minutes | In-memory only | API authorization |
| Refresh token | Keycloak | Opaque | 24 hours | keycloak-js manages internally | Silent token renewal |
| Offline token | Keycloak | Opaque | 14 days | IndexedDB (encrypted) | Field workers without connectivity |
| ID token | Keycloak | JWT | 15 minutes | In-memory only | User profile display (not sent to API) |

### Access Token Claims

The Keycloak-issued JWT access token contains standard OIDC claims plus custom claims added via protocol mappers:

```json
{
  "iss": "https://auth.chesed.org/realms/chesed",
  "sub": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "aud": "chesed-api",
  "exp": 1711988100,
  "iat": 1711987200,
  "email": "maria@example.com",
  "email_verified": true,
  "preferred_username": "maria.silva",
  "given_name": "Maria",
  "family_name": "Silva",
  "realm_access": {
    "roles": ["COORDINATOR"]
  },
  "person_id": "c56a4180-65aa-42ec-a945-5fd21dec0538"
}
```

**Note:** `campus_id` is NOT in the JWT token. It is resolved from the backend database (`app_user.campus_id` or `person.campus_id`) by the `AutoProvision` middleware and `GET /api/v1/auth/me` endpoint.

### Custom Protocol Mappers (Keycloak Configuration)

| Mapper Name | Type | User Attribute | Token Claim | Claim Type |
|-------------|------|---------------|-------------|------------|
| person_id | User Attribute | person_id | person_id | String |

These mappers are configured in the Keycloak `chesed-custom-claims` client scope to inject application-specific attributes into the access token.

### Offline Token for Field Workers

Field workers operating in areas without reliable internet connectivity use the `offline_access` scope to obtain an offline token:

- Requested by adding `scope: "openid offline_access"` to the keycloak-js init.
- Offline tokens survive Keycloak restarts and are valid for 14 days.
- When connectivity is restored, the offline token is exchanged for a fresh access token.
- Offline tokens can be revoked via the Keycloak Admin API if a device is lost.

---

## 5. RBAC Model

### Roles

Five roles are defined as **Keycloak realm roles** in the `chesed` realm:

| Role | Description | Typical User |
|------|------------|-------------|
| **ADMIN** | Full system access; manages users, audit logs, and all campuses | IT administrator, organization leadership |
| **COORDINATOR** | Manages campaigns, teams, reports; full operational access within campus | Team leader, department head |
| **PROFESSIONAL** | Records and manages own attendances; views assigned persons | Doctor, lawyer, social worker, psychologist |
| **SECRETARY** | Registers persons, creates triages, manages scheduling | Administrative staff |
| **VOLUNTEER** | Creates triages and basic data entry during events | Community volunteer |

### Role Hierarchy

```
ADMIN
  └── COORDINATOR
       └── PROFESSIONAL
            └── SECRETARY
                 └── VOLUNTEER
```

Each higher role inherits all permissions of lower roles plus additional capabilities. Role hierarchy is enforced in the Go API middleware, not in Keycloak (Keycloak assigns a single role per user; the API interprets the hierarchy).

### Role Extraction

Roles are extracted from the `realm_access.roles` array in the JWT access token. The Go middleware reads this claim and determines the user's effective permissions.

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
| **Volunteer Agreement** |
| Accept/reject own agreement | Yes | Yes | Yes | Yes | Yes |
| View person's agreements | No | No | No | Yes | Yes |
| Upload signed agreement | No | No | No | Yes | Yes |
| Download agreement document | No | No | No | Yes | Yes |
| **Consent** |
| Create consent | Yes | Yes | Yes | Yes | Yes |
| View consents | No | Campus | Own assigned | Campus | All |
| Revoke consent | No | No | No | No | Yes |

### Volunteer Agreement Access Restriction

Beyond RBAC role checks, volunteers must accept the **Volunteer Agreement** before accessing platform features. This is enforced by the `RequireAgreement` middleware.

**Middleware chain order:**

```
OIDC Token Validation → AutoProvision (app_user) → RequireAgreement → RBAC (RequireRole) → Handler
```

**How it works:**

1. After the user is authenticated and the `app_user` record is provisioned, `RequireAgreement` checks whether the user's person record has an accepted `volunteer_agreement`.
2. If the agreement is not accepted, all API requests (except agreement-related routes) return HTTP 403 with a specific error code indicating agreement is required.
3. The frontend detects this error and redirects the user to the agreement acceptance flow.

**Exempt routes (not subject to RequireAgreement):**

| Route | Purpose |
|-------|---------|
| `GET /api/v1/volunteer-agreement/text` | Fetch agreement text for display |
| `POST /api/v1/volunteer-agreement/accept` | Accept the agreement |
| `POST /api/v1/volunteer-agreement/reject` | Reject the agreement |
| `GET /api/v1/users/me` | Fetch current user profile |
| `POST /api/v1/self-register` | Self-registration flow |

**Rejection behavior:**

- If a volunteer rejects the agreement, the rejection is stored in the `volunteer_agreement` table with an optional reason and timestamp.
- The person record remains visible to coordinators for follow-up.
- The rejected user cannot access platform features until they accept the agreement in a subsequent attempt.
- Coordinators can view rejection status via `GET /api/v1/persons/{id}/agreement`.

**Manual upload flow (Coordinator):**

- For volunteers who sign a physical document, a coordinator can upload the signed agreement via `POST /api/v1/persons/{id}/agreement/upload`.
- This creates an `ACCEPTED` agreement with `signature_method = MANUAL_UPLOAD`.
- The uploaded document is stored at `document_path` and can be downloaded via `GET /api/v1/persons/{id}/agreement/document`.

---

## 6. Campus Data Isolation

### How It Works

1. `campus_id` is stored as a **Keycloak user attribute** on each user.
2. A **protocol mapper** includes `campus_id` in the JWT access token claims.
3. The Go **auth middleware** extracts `campus_id` from the token and injects it into the request context.
4. Every **repository query** includes `WHERE campus_id = $campus_id` to enforce data isolation.
5. ADMIN users can optionally pass `?campus_id=<uuid>` to query other campuses; all cross-campus access is logged.

### Go Middleware Implementation

```go
// CampusMiddleware extracts campus_id from JWT claims and injects into context.
func CampusMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims := auth.ClaimsFromContext(r.Context())
        if claims.CampusID == uuid.Nil {
            http.Error(w, `{"error":"forbidden","message":"Missing campus_id claim"}`, http.StatusForbidden)
            return
        }
        ctx := context.WithValue(r.Context(), campusKey, claims.CampusID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Repository Pattern

```go
func (r *PersonRepo) List(ctx context.Context, filter PersonFilter) ([]Person, error) {
    campusID := campus.FromContext(ctx)
    query := `SELECT id, full_name, email, phone, is_active
              FROM person
              WHERE campus_id = $1 AND is_active = TRUE
              ORDER BY full_name`
    rows, err := r.pool.Query(ctx, query, campusID)
    if err != nil {
        return nil, fmt.Errorf("person list: %w", err)
    }
    defer rows.Close()
    // scan rows...
}
```

### Cross-Campus Access

- Only ADMIN role can query across campuses.
- Cross-campus queries require an explicit `campus_id` query parameter.
- All cross-campus access is logged in the audit trail with `cross_campus=true`.

### Phase 3: PostgreSQL Row-Level Security

In Phase 3, campus isolation will be reinforced at the database level using PostgreSQL RLS policies. The Go API will set `SET LOCAL app.campus_id = '<uuid>'` on each transaction, and RLS policies will automatically filter rows.

---

## 7. Session Management

### Session Parameters

All session parameters are configured as **Keycloak realm settings** in the `chesed` realm:

| Parameter | Value | Keycloak Setting |
|-----------|-------|-----------------|
| Access token TTL | 15 minutes | Realm > Tokens > Access Token Lifespan |
| Refresh token TTL | 24 hours | Realm > Tokens > Client Session Max |
| Offline token idle | 14 days | Realm > Tokens > Offline Session Idle |
| Offline token max | 30 days | Realm > Tokens > Offline Session Max |
| SSO session idle | 30 minutes | Realm > Sessions > SSO Session Idle |
| SSO session max | 24 hours | Realm > Sessions > SSO Session Max |
| Concurrent sessions | Unlimited | Default (multi-device support) |

### Session Lifecycle

```
User opens app
  │
  ├── No token in memory → Redirect to Keycloak login
  │     │
  │     ├── User authenticates (credentials + optional MFA)
  │     │
  │     ├── Keycloak redirects back with authorization code
  │     │
  │     └── keycloak-js exchanges code for tokens
  │           ├── access_token (15 min)
  │           ├── refresh_token (24 hours)
  │           └── id_token (15 min)
  │
  ├── Token in memory → API call with Authorization: Bearer <access_token>
  │     │
  │     ├── Token valid → Process request
  │     │
  │     └── Token expired → keycloak-js silent refresh
  │           │
  │           ├── Refresh valid → New access_token issued
  │           │
  │           └── Refresh expired → Redirect to Keycloak login
  │
  └── Logout → keycloak-js.logout()
        │
        ├── Front-channel logout: Redirect to Keycloak /logout endpoint
        ├── Keycloak invalidates all sessions for the user
        └── React clears local state
```

### Token Revocation

| Scope | Method |
|-------|--------|
| Single session | `keycloak-js.logout()` or Keycloak Admin API: `DELETE /admin/realms/chesed/users/{id}/sessions/{sessionId}` |
| All sessions for a user | Keycloak Admin API: `POST /admin/realms/chesed/users/{id}/logout` |
| Specific offline token | Keycloak Admin API: revoke offline session |
| Emergency (all users) | Rotate the realm signing keys in Keycloak (invalidates all tokens) |

---

## 8. Account Security

### Password Policy

Configured in Keycloak under Realm > Authentication > Password Policy:

| Policy | Value |
|--------|-------|
| Minimum length | 8 characters |
| Minimum digits | 1 |
| Minimum lowercase characters | 1 |
| Password history | 3 (cannot reuse last 3 passwords) |
| Not username | Enabled |
| Not email | Enabled |

Keycloak handles all password hashing internally (bcrypt/PBKDF2).

### Account Lockout

Configured in Keycloak under Realm > Security Defenses > Brute Force Detection:

| Parameter | Value |
|-----------|-------|
| Enabled | Yes |
| Max login failures | 10 |
| Wait increment (seconds) | 900 (15 minutes) |
| Quick login check (ms) | 1000 |
| Minimum quick login wait (seconds) | 60 |
| Max wait (seconds) | 900 |
| Failure reset time (seconds) | 43200 (12 hours) |

Lockout events are recorded in the Keycloak event log and can be forwarded to the application audit trail.

### Password Recovery

Keycloak provides a built-in "forgot password" flow:

1. User clicks "Forgot password" on the Keycloak login page.
2. Keycloak sends a password reset email with a secure, time-limited link.
3. User sets a new password via the Keycloak-hosted form.
4. All existing sessions for the user are invalidated.

No application endpoints are needed for password recovery.

**Email configuration**: Keycloak is configured with SMTP settings (e.g., Amazon SES, SendGrid, or a self-hosted SMTP server) to send password reset and account verification emails.

### Email Verification

Email verification is enforced at three layers:

1. **Keycloak (primary)**: `verifyEmail: true` in realm configuration. Keycloak adds `VERIFY_EMAIL` as a required action on new accounts. Users must click a verification link sent via SMTP before completing login.
2. **Go API (defense-in-depth)**: The OIDC middleware checks the `email_verified` claim in the JWT. Requests with `email_verified: false` are rejected with HTTP 403.
3. **React Frontend (UX gate)**: The `EmailVerifiedGuard` component checks `email_verified` from the Keycloak token. Users with unverified emails are redirected to `/email-verification`, which shows a guidance screen explaining how to verify their email. All other routes (including `/complete-profile`) are blocked until verification is complete.

**SMTP Configuration**: Required for email verification and password reset. In development, Mailpit (local SMTP trap) is used. In production, configure an external SMTP provider (Amazon SES, SendGrid, or self-hosted) via Keycloak Admin Console.

| Environment Variable | Description | Example |
|---------------------|-------------|---------|
| `KC_SMTP_HOST` | SMTP server hostname | `smtp.sendgrid.net` |
| `KC_SMTP_PORT` | SMTP port (587 for TLS) | `587` |
| `KC_SMTP_FROM` | Sender email address | `noreply@institutanovasos.org` |
| `KC_SMTP_FROM_DISPLAY_NAME` | Sender display name | `Instituto Nova SOS` |
| `KC_SMTP_USER` | SMTP username | (from secrets manager) |
| `KC_SMTP_PASSWORD` | SMTP password | (from secrets manager) |
| `KC_SMTP_STARTTLS` | Enable STARTTLS | `true` |

### Multi-Factor Authentication (MFA)

MFA is configured via Keycloak **conditional authentication flows** with two methods available:

- **TOTP** (Google Authenticator, Authy, etc.) — time-based one-time passwords
- **Email OTP** — one-time code sent to verified email address (requires SMTP)

Users choose their preferred method during MFA enrollment.

| Role | MFA Policy | Available Methods |
|------|-----------|-------------------|
| ADMIN | **Mandatory** | TOTP or Email OTP |
| COORDINATOR | **Mandatory** | TOTP or Email OTP |
| PROFESSIONAL | Optional (opt-in) | TOTP or Email OTP |
| SECRETARY | Optional (opt-in) | TOTP or Email OTP |
| VOLUNTEER | Optional (opt-in) | TOTP or Email OTP |

Implementation in Keycloak:
1. Custom browser flow "Browser with Conditional MFA" checks realm roles.
2. ADMIN and COORDINATOR users are required to enroll in MFA on first login.
3. Other users can opt-in via the Keycloak Account Console (`/realms/chesed/account`).
4. Email OTP is available as an alternative to TOTP for users who cannot install authenticator apps.

**Development note**: MFA is disabled in dev mode by `init-realm.sh` (switches to built-in `browser` flow). Production must use the custom `Browser with Conditional MFA` flow.

### Account Lifecycle

| Event | Action |
|-------|--------|
| Account creation | Admin creates user in Keycloak (via Admin Console or Go API proxy); Keycloak sends email verification link |
| Email verification | User clicks link in email; Keycloak marks `emailVerified: true` |
| Password setup | User sets password on first login via Keycloak |
| MFA enrollment | ADMIN/COORDINATOR prompted to configure TOTP or Email OTP; other roles can opt-in |
| First API access | `app_user` record auto-created via `keycloak_subject_id` mapping |
| Role change | Admin updates realm role in Keycloak; change reflected in next token issuance |
| Deactivation | Admin disables user in Keycloak + sets `is_active = false` in `app_user`; all sessions revoked |
| Reactivation | Admin enables user in Keycloak + sets `is_active = true` in `app_user` |

### Onboarding Decision Tree

After authentication, the system determines the user's onboarding state via `GET /api/v1/auth/me`. The frontend uses the response to route users:

| Case | Condition | Behavior |
|------|-----------|----------|
| **Email not verified** | `email_verified = false` in Keycloak token | Frontend blocks all routes, shows `/email-verification` guidance screen. Backend rejects with HTTP 403. |
| **Email verified, no Person record** | `email_verified = true`, no person found by email in the same campus | Frontend redirects to `/complete-profile` for self-registration. |
| **Email verified, Person exists (pre-created)** | `email_verified = true`, person found by email in the same campus | Backend auto-links `app_user.person_id` to the existing person. Frontend grants direct access with RBAC. |
| **Email verified, Person linked, Volunteer without agreement** | Person has active VOLUNTEER role but no accepted agreement | Frontend redirects to `/volunteer-agreement`. |

### Person Auto-Linking by Email

When a user logs in for the first time and their `app_user` has no linked `person_id`, the `GET /api/v1/auth/me` endpoint checks if a person record with the same email exists in the user's campus. If found, the system automatically links the `app_user` to the existing person (sets `app_user.person_id`). This supports the workflow where an admin pre-creates a person record before the Keycloak user is created.

The link is established exclusively through the `app_user` join table — no `keycloak_user_id` field is added to the `person` table. This keeps the Person entity independent of the IAM provider.

---

## 9. Go Backend Integration

### OIDC Middleware

The Go API validates Keycloak-issued JWTs using the `coreos/go-oidc` library with JWKS-based signature verification.

```go
package auth

import (
    "context"
    "fmt"
    "net/http"
    "strings"

    "github.com/coreos/go-oidc/v3/oidc"
)

// Claims represents the relevant claims extracted from a Keycloak JWT.
type Claims struct {
    Subject   string   `json:"sub"`
    Email     string   `json:"email"`
    CampusID  string   `json:"campus_id"`
    PersonID  string   `json:"person_id"`
    Roles     []string `json:"-"` // extracted from realm_access.roles
}

// RealmAccess represents the realm_access claim structure in Keycloak tokens.
type RealmAccess struct {
    Roles []string `json:"roles"`
}

// TokenClaims is the full set of claims parsed from the JWT.
type TokenClaims struct {
    Claims
    RealmAccess RealmAccess `json:"realm_access"`
}

// NewOIDCMiddleware creates an HTTP middleware that validates Keycloak JWTs.
func NewOIDCMiddleware(issuerURL, clientID string) (func(http.Handler) http.Handler, error) {
    ctx := context.Background()

    provider, err := oidc.NewProvider(ctx, issuerURL)
    if err != nil {
        return nil, fmt.Errorf("oidc provider: %w", err)
    }

    verifier := provider.Verifier(&oidc.Config{
        ClientID: clientID,
    })

    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            rawToken := extractBearerToken(r)
            if rawToken == "" {
                http.Error(w, `{"error":"unauthorized","message":"Missing bearer token"}`, http.StatusUnauthorized)
                return
            }

            idToken, err := verifier.Verify(r.Context(), rawToken)
            if err != nil {
                http.Error(w, `{"error":"unauthorized","message":"Invalid or expired token"}`, http.StatusUnauthorized)
                return
            }

            var tc TokenClaims
            if err := idToken.Claims(&tc); err != nil {
                http.Error(w, `{"error":"unauthorized","message":"Failed to parse token claims"}`, http.StatusUnauthorized)
                return
            }
            tc.Roles = tc.RealmAccess.Roles

            ctx := context.WithValue(r.Context(), claimsKey, &tc.Claims)
            ctx = context.WithValue(ctx, rolesKey, tc.Roles)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }, nil
}

func extractBearerToken(r *http.Request) string {
    auth := r.Header.Get("Authorization")
    if !strings.HasPrefix(auth, "Bearer ") {
        return ""
    }
    return strings.TrimPrefix(auth, "Bearer ")
}
```

### JWKS Caching

The `coreos/go-oidc` library handles JWKS caching automatically:
- On first request, it fetches the JWKS from `{issuerURL}/.well-known/openid-configuration` and caches the keys.
- Keys are refreshed when a token presents an unknown `kid` (key ID).
- No manual caching code is needed.

### RBAC Middleware

```go
// RequireRole returns middleware that checks if the user has the required role
// or a higher role in the hierarchy.
func RequireRole(minRole string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            roles := RolesFromContext(r.Context())
            if !hasMinimumRole(roles, minRole) {
                http.Error(w, `{"error":"forbidden","message":"Insufficient permissions"}`, http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r.WithContext(r.Context()))
        })
    }
}

// roleHierarchy defines role levels (higher number = more privileges).
var roleHierarchy = map[string]int{
    "VOLUNTEER":    1,
    "SECRETARY":    2,
    "PROFESSIONAL": 3,
    "COORDINATOR":  4,
    "ADMIN":        5,
}

func hasMinimumRole(userRoles []string, minRole string) bool {
    minLevel, ok := roleHierarchy[minRole]
    if !ok {
        return false
    }
    for _, role := range userRoles {
        if level, exists := roleHierarchy[role]; exists && level >= minLevel {
            return true
        }
    }
    return false
}
```

### Route Registration Example

```go
r := chi.NewRouter()

oidcMiddleware, err := auth.NewOIDCMiddleware(
    "https://auth.chesed.org/realms/chesed",
    "chesed-api",
)
if err != nil {
    log.Fatalf("failed to create OIDC middleware: %v", err)
}

r.Route("/api/v1", func(r chi.Router) {
    r.Use(oidcMiddleware)
    r.Use(auth.CampusMiddleware)

    r.Route("/persons", func(r chi.Router) {
        r.Get("/", personHandler.List)             // All roles
        r.Post("/", personHandler.Create)           // All roles
        r.With(auth.RequireRole("SECRETARY")).Put("/{id}", personHandler.Update)
    })

    r.Route("/users", func(r chi.Router) {
        r.Use(auth.RequireRole("ADMIN"))
        r.Get("/", userHandler.List)
        r.Post("/", userHandler.Create)
        r.Patch("/{id}", userHandler.Update)
    })

    r.Get("/users/me", userHandler.Me)              // All roles
})
```

### Claims Context Helpers

```go
type contextKey string

const (
    claimsKey contextKey = "claims"
    rolesKey  contextKey = "roles"
    campusKey contextKey = "campus"
)

func ClaimsFromContext(ctx context.Context) *Claims {
    c, _ := ctx.Value(claimsKey).(*Claims)
    return c
}

func RolesFromContext(ctx context.Context) []string {
    r, _ := ctx.Value(rolesKey).([]string)
    return r
}
```

---

## 10. React Frontend Integration

### keycloak-js Adapter Initialization

```typescript
// src/auth/keycloak.ts
import Keycloak from "keycloak-js";

const keycloak = new Keycloak({
  url: import.meta.env.VITE_KEYCLOAK_URL,       // e.g. "https://auth.chesed.org"
  realm: import.meta.env.VITE_KEYCLOAK_REALM,   // "chesed"
  clientId: import.meta.env.VITE_KEYCLOAK_CLIENT_ID, // "chesed-pwa"
});

export default keycloak;
```

### App Initialization

```typescript
// src/main.tsx
import keycloak from "./auth/keycloak";

keycloak
  .init({
    onLoad: "login-required",
    pkceMethod: "S256",
    checkLoginIframe: false,
    silentCheckSsoRedirectUri:
      window.location.origin + "/silent-check-sso.html",
  })
  .then((authenticated) => {
    if (authenticated) {
      // Start silent token refresh
      setInterval(() => {
        keycloak.updateToken(60).catch(() => {
          keycloak.logout();
        });
      }, 30_000);

      // Render the app
      const root = ReactDOM.createRoot(document.getElementById("root")!);
      root.render(<App />);
    }
  })
  .catch((err) => {
    console.error("Keycloak init failed:", err);
  });
```

### API Client with Token Injection

```typescript
// src/api/client.ts
import keycloak from "../auth/keycloak";

const API_BASE = import.meta.env.VITE_API_URL;

export async function apiFetch(
  path: string,
  options: RequestInit = {}
): Promise<Response> {
  // Refresh token if it expires within 30 seconds
  await keycloak.updateToken(30);

  const headers = new Headers(options.headers);
  headers.set("Authorization", `Bearer ${keycloak.token}`);
  headers.set("Content-Type", "application/json");

  return fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  });
}
```

### Protected Route Pattern

```typescript
// src/components/ProtectedRoute.tsx
import { Navigate } from "react-router-dom";
import keycloak from "../auth/keycloak";

interface ProtectedRouteProps {
  children: React.ReactNode;
  minRole: string;
}

const ROLE_HIERARCHY: Record<string, number> = {
  VOLUNTEER: 1,
  SECRETARY: 2,
  PROFESSIONAL: 3,
  COORDINATOR: 4,
  ADMIN: 5,
};

function hasMinimumRole(minRole: string): boolean {
  const roles: string[] = keycloak.realmAccess?.roles ?? [];
  const minLevel = ROLE_HIERARCHY[minRole] ?? Infinity;
  return roles.some((r) => (ROLE_HIERARCHY[r] ?? 0) >= minLevel);
}

export function ProtectedRoute({ children, minRole }: ProtectedRouteProps) {
  if (!keycloak.authenticated) {
    keycloak.login();
    return null;
  }
  if (!hasMinimumRole(minRole)) {
    return <Navigate to="/unauthorized" replace />;
  }
  return <>{children}</>;
}
```

### Logout

```typescript
// Front-channel logout: redirects to Keycloak to end the SSO session.
function handleLogout() {
  keycloak.logout({
    redirectUri: window.location.origin,
  });
}
```

### Frontend Route Guard Hierarchy

The frontend enforces a layered guard chain that mirrors the backend middleware:

```
ProtectedRoute (isAuthenticated?)
  └─ EmailVerifiedGuard (email_verified in token?)
       └─ OnboardingGuard (calls GET /auth/me)
            ├─ needs_profile_completion? → redirect /complete-profile
            ├─ needs_agreement? → redirect /volunteer-agreement
            └─ OK → render main app
```

Route groups:
1. `/email-verification` — `ProtectedRoute` only (must be reachable with unverified email)
2. `/complete-profile`, `/volunteer-agreement` — `ProtectedRoute` + `EmailVerifiedGuard`
3. Main app routes — `ProtectedRoute` + `EmailVerifiedGuard` + `OnboardingGuard`

### Extracting User Info from Token

```typescript
// src/hooks/useCurrentUser.ts
import keycloak from "../auth/keycloak";

interface CurrentUser {
  sub: string;
  email: string;
  campusId: string;
  personId: string;
  roles: string[];
  fullName: string;
}

export function useCurrentUser(): CurrentUser {
  const parsed = keycloak.tokenParsed;
  return {
    sub: parsed?.sub ?? "",
    email: parsed?.email ?? "",
    campusId: parsed?.campus_id ?? "",
    personId: parsed?.person_id ?? "",
    roles: keycloak.realmAccess?.roles ?? [],
    fullName: `${parsed?.given_name ?? ""} ${parsed?.family_name ?? ""}`.trim(),
  };
}
```

---

## 11. User Provisioning Flow

### Creating a New User

```
Admin initiates user creation
  │
  ├── Option A: Keycloak Admin Console (direct)
  │     Admin logs into Keycloak Admin Console and creates the user manually.
  │
  └── Option B: Go API proxy (programmatic)
        Admin calls POST /api/v1/users which proxies to the Keycloak Admin API.
        │
        ▼
  Keycloak creates user with:
    - email
    - first name / last name
    - realm role (ADMIN, COORDINATOR, PROFESSIONAL, SECRETARY, VOLUNTEER)
    - user attribute: campus_id = "<uuid>"
    - user attribute: person_id = "<uuid>" (if linked to a person)
    - required action: UPDATE_PASSWORD
        │
        ▼
  Keycloak sends "Set Password" email to the user
        │
        ▼
  User clicks the link and sets their password
        │
        ▼
  User logs in for the first time
        │
        ▼
  Go API receives the first request with the new access token
    - Extracts keycloak_subject_id (sub claim)
    - Checks if app_user record exists
    - If not: auto-creates app_user with keycloak_subject_id, email, campus_id, person_id
    - Logs LOGIN event in audit_log
```

### Go API User Provisioning (Auto-Create on First Login)

```go
// EnsureAppUser creates the local app_user record on first login if it does not exist.
func (s *UserService) EnsureAppUser(ctx context.Context, claims *auth.Claims) (*AppUser, error) {
    user, err := s.repo.FindByKeycloakSubject(ctx, claims.Subject)
    if err != nil && !errors.Is(err, ErrNotFound) {
        return nil, fmt.Errorf("find app_user: %w", err)
    }
    if user != nil {
        return user, nil
    }

    newUser := &AppUser{
        ID:                  uuid.New(),
        KeycloakSubjectID:   claims.Subject,
        Email:               claims.Email,
        CampusID:            uuid.MustParse(claims.CampusID),
        PersonID:            parseOptionalUUID(claims.PersonID),
        IsActive:            true,
    }
    if err := s.repo.Create(ctx, newUser); err != nil {
        return nil, fmt.Errorf("create app_user: %w", err)
    }

    s.audit.Log(ctx, AuditEntry{
        Action:     "CREATE",
        EntityType: "app_user",
        EntityID:   newUser.ID,
        Description: "Auto-created app_user on first login",
    })

    return newUser, nil
}
```

### Deactivating a User

1. Admin disables the user in Keycloak (via Admin Console or API).
2. Admin sets `is_active = false` on the `app_user` record via `PATCH /api/v1/users/{id}/deactivate`.
3. Keycloak revokes all active sessions for the user.
4. The user can no longer authenticate.

---

## 12. Audit Integration

### Dual Audit Strategy

Chesed maintains two complementary audit trails:

| Source | Scope | Storage | Retention |
|--------|-------|---------|-----------|
| Keycloak event log | Authentication events (login, logout, failed attempts, password changes, MFA events) | Keycloak database | 90 days (configurable) |
| Application audit_log | Business operations (CRUD on persons, attendances, donations, etc.) | Application PostgreSQL | 5 years minimum |

### Keycloak Event Configuration

Enable the following in Keycloak under Realm > Events:

**Login Events:**
- `LOGIN`, `LOGIN_ERROR`
- `LOGOUT`, `LOGOUT_ERROR`
- `REGISTER`
- `UPDATE_PASSWORD`
- `SEND_RESET_PASSWORD`
- `RESET_PASSWORD`
- `UPDATE_TOTP`, `REMOVE_TOTP`

**Admin Events:**
- All enabled (user creation, role assignment, user disable, etc.)
- Include representation: Yes

### Application Audit Log

Every business operation logged by the Go API includes the Keycloak subject ID for traceability:

| Event | Audit Action | Details |
|-------|-------------|---------|
| First login (app_user created) | `CREATE` | keycloak_subject_id, email, campus_id |
| View person record | `READ` | user sub, person_id |
| Create/update attendance | `CREATE` / `UPDATE` | user sub, attendance_id, old/new values |
| Cross-campus access | `READ` | user sub, target campus_id, `cross_campus=true` |
| Permission change (via Keycloak proxy) | `PERMISSION_CHANGE` | admin sub, target user, old/new role |
| User deactivation | `UPDATE` | admin sub, target user, `is_active: false` |
| Data export | `EXPORT` | user sub, report type, parameters |

### Unified Audit Trail

For compliance reporting, a unified view can be constructed by:
1. Querying the application `audit_log` table for business events.
2. Querying the Keycloak Admin API for authentication events: `GET /admin/realms/chesed/events`.
3. Correlating records via the Keycloak `sub` (matches `app_user.keycloak_subject_id`).

---

## 13. Keycloak Realm Configuration Summary

### Clients

| Client ID | Type | Purpose |
|-----------|------|---------|
| `chesed-pwa` | Public (SPA) | React frontend; Authorization Code + PKCE |
| `chesed-api` | Bearer-only | Go API; validates tokens (does not initiate login) |
| `chesed-api` | Confidential | Go API service account for Keycloak Admin API calls (user provisioning) |

### Client Scopes

| Scope | Claims Added |
|-------|-------------|
| `openid` | sub, email, preferred_username |
| `profile` | given_name, family_name |
| `chesed-custom` | campus_id, person_id (via protocol mappers) |
| `offline_access` | Enables offline tokens for field workers |

### Realm Roles

`ADMIN`, `COORDINATOR`, `PROFESSIONAL`, `SECRETARY`, `VOLUNTEER`

### Required Actions

| Action | When |
|--------|------|
| `UPDATE_PASSWORD` | On first login (set by admin during user creation) |
| `CONFIGURE_TOTP` | On first login for ADMIN users |
| `VERIFY_EMAIL` | On account creation |

---

## 14. Future Considerations

### Phase 2

- **SSO with Google/Microsoft**: Configure Keycloak Identity Brokering to allow volunteer accounts to log in with Google or Microsoft. The identity provider is configured in Keycloak; no application code changes required.

### Phase 3

- **Fine-grained authorization**: Use Keycloak Authorization Services (UMA 2.0) for resource-level permissions that go beyond role-based access (e.g., per-record ownership, delegation).
- **Service accounts for WordPress API**: Create a confidential client in Keycloak for the WordPress portal integration, using Client Credentials flow for server-to-server communication.
- **DPoP token binding**: Implement Demonstrating Proof-of-Possession (DPoP) to bind access tokens to the client's cryptographic key, preventing token theft and replay attacks.
- **PostgreSQL RLS**: Enforce campus isolation at the database level using Row-Level Security policies, as described in Section 6.

---

## Appendix: API Endpoints Removed

The following endpoints from the original design are **no longer part of the Go API**, as their functionality is handled entirely by Keycloak:

| Removed Endpoint | Replacement |
|-----------------|-------------|
| `POST /api/v1/auth/login` | Keycloak OIDC login via keycloak-js |
| `POST /api/v1/auth/refresh` | keycloak-js `updateToken()` |
| `POST /api/v1/auth/forgot-password` | Keycloak built-in forgot password |
| `POST /api/v1/auth/reset-password` | Keycloak built-in password reset |
| `POST /api/v1/auth/logout` | keycloak-js `logout()` |

The `GET /api/v1/users/me` endpoint remains. It returns the current user's application profile by looking up the `app_user` record via the `sub` claim from the JWT.
