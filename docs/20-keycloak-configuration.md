# 20. Keycloak Configuration Guide

## Overview

Chesed uses **Keycloak** as its external Identity and Access Management (IAM) provider. Keycloak handles all authentication, credential storage, password policies, MFA, and brute-force protection. The Go API and React PWA integrate via **OpenID Connect (OIDC)**.

This document covers realm setup, client configuration, role mapping, custom claims, MFA, branding, and operational procedures.

---

## Realm Configuration

### Realm: `chesed`

| Setting | Value | Notes |
|---------|-------|-------|
| Realm name | `chesed` | All lowercase |
| Display name | `SOS Gestao - Instituto Nova SOS` | Shown on login page |
| Login with email | Enabled | Users authenticate with email, not username |
| Email as username | Enabled | Simplifies user management |
| Forgot password | Enabled | Built-in password reset flow |
| Remember me | Disabled | Security preference for shared devices |
| Verify email | **Enabled** | Users must verify email before accessing the system |
| User registration | Disabled | Users are created by admins only |
| Edit username | Disabled | Prevents identity confusion |

### SMTP Configuration

Required for email verification, password reset, and email OTP flows.

| Setting | Development (Mailpit) | Production |
|---------|----------------------|------------|
| Host | `mailpit` (Docker service) | SMTP provider (e.g., Amazon SES, SendGrid) |
| Port | `1025` | `587` (STARTTLS) |
| From | `noreply@chesed.test` | `noreply@institutanovasos.org` |
| From Display Name | `Instituto Nova SOS` | `Instituto Nova SOS` |
| Authentication | Disabled | Enabled |
| STARTTLS | Disabled | Enabled |
| Credentials | N/A | From secrets manager |

**Development**: Mailpit captures all emails at `http://localhost:8025`. No real emails are sent.

**Production environment variables**:
```
KC_SMTP_HOST=smtp.sendgrid.net
KC_SMTP_PORT=587
KC_SMTP_FROM=noreply@institutanovasos.org
KC_SMTP_FROM_DISPLAY_NAME=Instituto Nova SOS
KC_SMTP_USER=<from-secrets-manager>
KC_SMTP_PASSWORD=<from-secrets-manager>
KC_SMTP_STARTTLS=true
```

---

## Clients

### Client: `chesed-pwa` (React PWA)

| Setting | Value | Notes |
|---------|-------|-------|
| Client ID | `chesed-pwa` | Referenced in `keycloak-js` init |
| Client Protocol | openid-connect | OIDC |
| Access Type | public | No client secret (PKCE used instead) |
| Standard Flow | Enabled | Authorization Code Flow |
| Direct Access Grants | Disabled | No resource owner password grant |
| Valid Redirect URIs | `http://localhost:5173/*`, `https://chesed.example.com/*` | Per environment |
| Web Origins | `http://localhost:5173`, `https://chesed.example.com` | CORS |
| Post Logout Redirect URIs | Same as redirect URIs | |
| PKCE Code Challenge Method | S256 | Required for public clients |

**Default Client Scopes**: `openid`, `profile`, `email`, `offline_access`

### Client: `chesed-api` (Go API — Admin API Access)

| Setting | Value | Notes |
|---------|-------|-------|
| Client ID | `chesed-api` | Used for Keycloak Admin API calls |
| Client Protocol | openid-connect | |
| Access Type | confidential | Has client secret |
| Service Account | Enabled | For Admin API access |
| Standard Flow | Disabled | Not used for user login |
| Direct Access Grants | Disabled | |

**Service Account Roles**: Assign `realm-management` → `manage-users`, `view-users`, `manage-realm` (for user provisioning and session management).

---

## Realm Roles

| Role | Description | Keycloak Realm Role Name |
|------|-------------|--------------------------|
| Admin | Full system access, all campuses | `ADMIN` |
| Coordinator | Campaign and team management | `COORDINATOR` |
| Professional | Service attendance recording | `PROFESSIONAL` |
| Secretary | Person registration, triage | `SECRETARY` |
| Volunteer | Basic data entry during events | `VOLUNTEER` |

Roles are assigned to users in Keycloak and included in the access token via the `realm_access.roles` claim (default behavior, no extra configuration needed).

---

## Custom Protocol Mappers

Custom claims are added to tokens via protocol mappers. These enable campus-scoped data isolation and person linkage.

### Mapper: `campus_id`

| Setting | Value |
|---------|-------|
| Name | `campus_id` |
| Mapper Type | User Attribute |
| User Attribute | `campus_id` |
| Token Claim Name | `campus_id` |
| Claim JSON Type | String |
| Add to ID token | Yes |
| Add to access token | Yes |
| Add to userinfo | Yes |

### Mapper: `person_id`

| Setting | Value |
|---------|-------|
| Name | `person_id` |
| Mapper Type | User Attribute |
| User Attribute | `person_id` |
| Token Claim Name | `person_id` |
| Claim JSON Type | String |
| Add to ID token | Yes |
| Add to access token | Yes |
| Add to userinfo | Yes |

### Resulting Token Claims

```json
{
  "sub": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "preferred_username": "maria@example.com",
  "email": "maria@example.com",
  "email_verified": true,
  "realm_access": {
    "roles": ["COORDINATOR"]
  },
  "campus_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "person_id": "550e8400-e29b-41d4-a716-446655440000",
  "iss": "https://keycloak.example.com/realms/chesed",
  "aud": "chesed-pwa",
  "exp": 1711988100,
  "iat": 1711987200
}
```

---

## Session and Token Configuration

### Realm Settings → Tokens

| Setting | Value | Path in Keycloak Admin |
|---------|-------|------------------------|
| Access Token Lifespan | 15 minutes | Realm Settings → Tokens |
| Client Session Idle | 30 minutes | Realm Settings → Sessions |
| Client Session Max | 24 hours | Realm Settings → Sessions |
| SSO Session Idle | 30 minutes | Realm Settings → Sessions |
| SSO Session Max | 24 hours | Realm Settings → Sessions |
| Offline Session Idle | 14 days | Realm Settings → Tokens |
| Offline Session Max Lifespan | Enabled, 30 days | Realm Settings → Tokens |
| Default Signature Algorithm | RS256 | Realm Settings → Tokens |

### Refresh Token Configuration

| Setting | Value |
|---------|-------|
| Revoke Refresh Token | Enabled |
| Refresh Token Max Reuse | 0 (single use) |
| Client Offline Session Idle | 14 days |

---

## Password Policy

Configure in: **Authentication → Password Policy**

| Policy | Value | Notes |
|--------|-------|-------|
| Minimum Length | 8 | |
| Digits | 1 | At least one number |
| Lower Case | 1 | At least one lowercase letter |
| Not Username | Enabled | Password cannot be the username |
| Password History | 3 | Cannot reuse last 3 passwords |
| Hashing Algorithm | pbkdf2-sha512 | Keycloak default; secure |

---

## Brute-Force Protection

Configure in: **Realm Settings → Security Defenses → Brute Force Detection**

| Setting | Value |
|---------|-------|
| Enabled | Yes |
| Permanent Lockout | No |
| Max Login Failures | 10 |
| Wait Increment | 900 seconds (15 minutes) |
| Max Failure Wait | 3600 seconds (1 hour) |
| Failure Reset Time | 43200 seconds (12 hours) |
| Quick Login Check | 1000 milliseconds |
| Minimum Quick Login Wait | 60 seconds |

---

## MFA Configuration

### MFA Policy

| Role | Policy | Enforcement |
|------|--------|-------------|
| ADMIN | **Mandatory** | Conditional auth flow requires MFA on every login |
| COORDINATOR | **Mandatory** | Conditional auth flow requires MFA on every login |
| PROFESSIONAL | Optional (opt-in) | Users can enroll via Keycloak Account Console |
| SECRETARY | Optional (opt-in) | Users can enroll via Keycloak Account Console |
| VOLUNTEER | Optional (opt-in) | Users can enroll via Keycloak Account Console |

### Available MFA Methods

Users with mandatory or opt-in MFA can choose between:

1. **TOTP (Authenticator App)** — Google Authenticator, Authy, Microsoft Authenticator, or any TOTP-compatible app. Scans QR code during enrollment.
2. **Email OTP** — One-time code sent to the user's verified email address. Requires SMTP to be configured. Useful for users without smartphones.

### Browser Authentication Flow

Custom flow: `Browser with Conditional MFA`

```
Cookie Authentication (ALTERNATIVE)
Identity Provider Redirector (ALTERNATIVE)
Username/Password Form (REQUIRED)
  ├─ Conditional MFA - ADMIN (CONDITIONAL)
  │   ├─ Condition: User has ADMIN role
  │   └─ Action: Require OTP
  └─ Conditional MFA - COORDINATOR (CONDITIONAL)
      ├─ Condition: User has COORDINATOR role
      └─ Action: Require OTP
```

On first login, ADMIN and COORDINATOR users are prompted to configure their preferred MFA method (TOTP or Email OTP).

**Development note**: `init-realm.sh` switches the browser flow to the built-in `browser` flow (no MFA) for testing. Production must use `Browser with Conditional MFA`.

### TOTP Policy

Configure in: **Authentication → OTP Policy**

| Setting | Value |
|---------|-------|
| OTP Type | Time-Based (TOTP) |
| Algorithm | SHA1 |
| Digits | 6 |
| Period | 30 seconds |
| Look Ahead Window | 1 |
| Initial Counter | 0 |

### Email OTP

Keycloak 26 supports email-based OTP natively when SMTP is configured. Users who cannot install an authenticator app can select "Email" as their MFA method during enrollment. A 6-digit code is sent to their verified email address on each login.

---

## User Provisioning

### Creating a New User

**Via Keycloak Admin Console:**
1. Navigate to **Users → Add User**
2. Set `Email`, `First Name`, `Last Name`
3. Set `Email Verified` = Yes (if email is confirmed)
4. Set custom attributes:
   - `campus_id` = UUID of the user's campus
   - `person_id` = UUID of the linked person record (if exists)
5. Under **Role Mappings**, assign the appropriate realm role
6. Under **Credentials**, set a temporary password with "Temporary" = On
7. The user will be prompted to change their password on first login

**Via Go API (proxying to Keycloak Admin API):**
```
POST /api/v1/users
{
  "email": "maria@example.com",
  "first_name": "Maria",
  "last_name": "Santos",
  "access_profile": "SECRETARY",
  "campus_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "person_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

The Go API:
1. Creates the user in Keycloak via Admin API
2. Sets custom attributes (`campus_id`, `person_id`)
3. Assigns the realm role
4. Sets required action: `UPDATE_PASSWORD`
5. Sends email with temporary credentials (if SMTP configured)
6. Creates the local `app_user` record with `keycloak_subject_id`

### Deactivating a User

1. Disable the user in Keycloak (sets `enabled = false`)
2. Terminate all active sessions via Keycloak Admin API
3. Set `is_active = false` in the local `app_user` table
4. Log the deactivation in the audit trail

---

## Keycloak Login Page Branding

Keycloak supports custom themes for the login page.

### Approach for Chesed

**Phase 1 (MVP)**: Use Keycloak's default theme with minimal customization:
- Set realm display name: "SOS Gestao - Instituto Nova SOS"
- Upload the NGO logo via **Realm Settings → Themes → Login Theme** attributes
- Set `loginTheme = keycloak` (default)

**Phase 2**: Create a custom theme:
1. Create theme directory: `keycloak/themes/chesed/login/`
2. Override `template.ftl` and `login.ftl` for layout
3. Add custom CSS in `resources/css/login.css`
4. Configure colors to match the Tailwind theme
5. Mount the theme volume in Docker Compose

---

## Realm Export and Version Control

### Exporting the Realm

```bash
# From running Keycloak container
docker exec -it chesed-keycloak \
  /opt/keycloak/bin/kc.sh export \
  --realm chesed \
  --file /tmp/realm-export.json \
  --users realm_file

# Copy to project
docker cp chesed-keycloak:/tmp/realm-export.json keycloak/realm-export.json
```

### Import on Startup

The Docker Compose configuration mounts the export file for automatic import:

```yaml
keycloak:
  image: quay.io/keycloak/keycloak:26-alpine
  command: start-dev --import-realm
  volumes:
    - ./keycloak/realm-export.json:/opt/keycloak/data/import/realm.json
```

### Version Control Rules

1. Always commit `keycloak/realm-export.json` after realm changes
2. Never include user passwords or secrets in the export
3. Review realm changes in pull requests like code changes
4. Document the reason for realm changes in commit messages

---

## Audit Event Configuration

### Keycloak Event Listeners

Configure in: **Events → Config**

**Login Events:**
| Event | Logged |
|-------|--------|
| LOGIN | Yes |
| LOGIN_ERROR | Yes |
| LOGOUT | Yes |
| REGISTER | Yes |
| UPDATE_PASSWORD | Yes |
| RESET_PASSWORD | Yes |
| SEND_RESET_PASSWORD | Yes |
| VERIFY_EMAIL | Yes |

**Admin Events:**
- Enable admin event logging: Yes
- Include representation: Yes (logs the payload of admin operations)

### Forwarding to Application Audit Log

**Option A (Recommended for MVP)**: Poll Keycloak Admin Events API periodically from a background Go routine. Map events to the application's `audit_log` table using the Keycloak `sub` (user ID) for correlation.

**Option B (Phase 2)**: Implement a custom Keycloak SPI Event Listener that sends events to the Go API via webhook. More real-time but requires custom Keycloak extension.

---

## Environment-Specific Configuration

### Local Development

```env
KEYCLOAK_URL=http://localhost:8180
KEYCLOAK_REALM=chesed
KEYCLOAK_CLIENT_ID=chesed-pwa
KEYCLOAK_ADMIN_CLIENT_ID=chesed-api
KEYCLOAK_ADMIN_CLIENT_SECRET=<dev-secret>
KEYCLOAK_ADMIN=admin
KEYCLOAK_ADMIN_PASSWORD=admin
```

### Staging / Production

```env
KEYCLOAK_URL=https://auth.chesed.example.com
KEYCLOAK_REALM=chesed
KEYCLOAK_CLIENT_ID=chesed-pwa
KEYCLOAK_ADMIN_CLIENT_ID=chesed-api
KEYCLOAK_ADMIN_CLIENT_SECRET=<from-secrets-manager>
```

**Production hardening:**
- Run Keycloak in `start` mode (not `start-dev`)
- Enable HTTPS via reverse proxy (Cloudflare or Caddy)
- Set `KC_HOSTNAME` to the public Keycloak URL
- Disable Keycloak Admin Console from public access (restrict to internal network)
- Set strong admin credentials (rotated every 6 months)
- Enable Keycloak metrics endpoint for Grafana monitoring
- Configure database connection pooling

---

## Troubleshooting

### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| "Invalid redirect URI" on login | Redirect URI not registered in client | Add the URI to `Valid Redirect URIs` in Keycloak client config |
| "CORS error" on token exchange | Web Origin not configured | Add the frontend origin to `Web Origins` in client config |
| Token validation fails in Go API | JWKS endpoint unreachable | Check `KEYCLOAK_URL` env var; ensure Go API can reach Keycloak |
| `campus_id` missing from token | Protocol mapper not configured | Add User Attribute mapper for `campus_id` (see Protocol Mappers section) |
| User cannot log in after creation | Temporary password not set | Set credentials in Keycloak with "Temporary" = On |
| MFA prompt not appearing for admin | Conditional flow not bound | Bind the custom browser flow in Authentication → Bindings |
| Token expired during offline work | Refresh token TTL too short | Use `offline_access` scope for field workers (14-day TTL) |
| Import fails on startup | Realm already exists | Keycloak skips import if realm exists; delete realm first or use `--override=true` |

### Health Check

The Go API should verify Keycloak connectivity on startup:

```go
// Check OIDC discovery endpoint
resp, err := http.Get(keycloakURL + "/realms/chesed/.well-known/openid-configuration")
if err != nil || resp.StatusCode != 200 {
    log.Fatal("Keycloak is not reachable")
}
```

---

## Keycloak Version Policy

- Use the latest **LTS or stable** release (currently 26.x)
- Pin Docker image by version tag (e.g., `quay.io/keycloak/keycloak:26-alpine`), not `latest`
- Test Keycloak upgrades in staging before production
- Monitor Keycloak security advisories: https://www.keycloak.org/security
- Apply security patches within 7 days of release
- Realm export/import ensures safe migration between versions

---

## References

- Keycloak Documentation: https://www.keycloak.org/documentation
- Keycloak Admin REST API: https://www.keycloak.org/docs-api/latest/rest-api/
- keycloak-js Adapter: https://www.keycloak.org/docs/latest/securing_apps/#_javascript_adapter
- coreos/go-oidc: https://github.com/coreos/go-oidc
- OIDC Specification: https://openid.net/specs/openid-connect-core-1_0.html
- OAuth 2.1 Draft: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-11
