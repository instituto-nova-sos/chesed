# 17 - Security Test Strategy

## Overview

This document defines the security testing approach for Chesed. The system handles sensitive personal data (health records, CPF numbers, social vulnerability data) of vulnerable populations, making security testing a first-class concern.

---

## Security Testing Layers

```
┌──────────────────────────────────────┐
│  Layer 4: Penetration Testing        │  External assessment (Phase 3)
├──────────────────────────────────────┤
│  Layer 3: Dependency Scanning        │  Automated vulnerability alerts
├──────────────────────────────────────┤
│  Layer 2: Integration Security Tests │  Auth, RBAC, data isolation
├──────────────────────────────────────┤
│  Layer 1: Unit Security Tests        │  Input validation, OIDC, sanitization
└──────────────────────────────────────┘
```

---

## Layer 1: Unit Security Tests

Tests that run on every commit, covering individual security functions.

### Input Validation

```go
// Test that all person inputs are validated
func TestPersonInput_Validation(t *testing.T) {
    tests := []struct {
        name    string
        input   CreatePersonInput
        wantErr string
    }{
        {"empty name", CreatePersonInput{FullName: ""}, "full_name is required"},
        {"XSS in name", CreatePersonInput{FullName: "<script>alert(1)</script>"}, "full_name contains invalid characters"},
        {"SQL in document", CreatePersonInput{DocumentNumber: "'; DROP TABLE person;--"}, "document_number format invalid"},
        {"oversized input", CreatePersonInput{FullName: strings.Repeat("a", 1000)}, "full_name exceeds max length"},
    }
    // ...
}
```

### OIDC Token Validation (Keycloak)

```go
func TestOIDC_ExpiredKeycloakToken(t *testing.T) {
    token := generateExpiredKeycloakToken(t, testKeycloakClaims)
    _, err := oidcValidator.ValidateToken(ctx, token)
    assert.ErrorIs(t, err, ErrTokenExpired)
}

func TestOIDC_InvalidIssuer(t *testing.T) {
    token := generateKeycloakToken(t, withIssuer("https://wrong-issuer.example.com"))
    _, err := oidcValidator.ValidateToken(ctx, token)
    assert.ErrorIs(t, err, ErrInvalidIssuer)
}

func TestOIDC_InvalidAudience(t *testing.T) {
    token := generateKeycloakToken(t, withAudience("wrong-client-id"))
    _, err := oidcValidator.ValidateToken(ctx, token)
    assert.ErrorIs(t, err, ErrInvalidAudience)
}

func TestOIDC_MissingCampusClaim(t *testing.T) {
    token := generateKeycloakToken(t, withoutClaim("campus_id"))
    _, err := oidcValidator.ValidateToken(ctx, token)
    assert.ErrorIs(t, err, ErrMissingCampusClaim)
}

func TestOIDC_MissingRealmRoles(t *testing.T) {
    token := generateKeycloakToken(t, withoutRealmRoles())
    _, err := oidcValidator.ValidateToken(ctx, token)
    assert.ErrorIs(t, err, ErrMissingRealmRoles)
}

func TestOIDC_InvalidSignature(t *testing.T) {
    token := generateKeycloakToken(t, signedWithWrongKey())
    _, err := oidcValidator.ValidateToken(ctx, token)
    assert.Error(t, err)
}

func TestOIDC_MalformedToken(t *testing.T) {
    _, err := oidcValidator.ValidateToken(ctx, "not.a.valid.jwt.token")
    assert.Error(t, err)
}
```

### UUID Validation

```go
func TestSyncID_MustBeValidUUID(t *testing.T) {
    input := SyncPushInput{SyncID: "not-a-uuid"}
    err := validate.Struct(input)
    assert.Error(t, err)
}
```

---

## Layer 2: Integration Security Tests

Tests that verify security behavior across components, using a real database.

### Authentication Tests

| Test Case | Description | Expected |
|-----------|------------|----------|
| Access with valid Keycloak token | Properly signed token with correct issuer, audience, and claims | 200 |
| Access with token from wrong realm/issuer | Token issued by a different Keycloak realm or external IdP | 401 |
| Access with token missing campus_id claim | Valid Keycloak token but without the required `campus_id` custom claim | 401 |
| Access with token for disabled local user (is_active=false) | Valid Keycloak token, but corresponding `app_user` record has `is_active=false` | 401 |
| Access with expired token | Expired Keycloak JWT | 401 |
| Access with no token | Missing Authorization header | 401 |

### RBAC Tests

| Test Case | Description | Expected |
|-----------|------------|----------|
| Volunteer creates triage | Allowed action | 201 |
| Volunteer edits person | Forbidden action | 403 |
| Professional edits own attendance | Allowed action | 200 |
| Professional edits other's attendance | Forbidden action | 403 |
| Coordinator views reports | Allowed action | 200 |
| Secretary views audit logs | Forbidden action | 403 |
| Admin manages users | Allowed action | 200 |

### Campus Isolation Tests

| Test Case | Description | Expected |
|-----------|------------|----------|
| User queries own campus persons | Standard query | Returns only campus-scoped data |
| User queries other campus persons | No explicit campus param | Returns empty (filtered out) |
| User tries to create record in other campus | campus_id mismatch | 403 |
| Admin queries cross-campus | With explicit campus_id param | Returns requested campus data |
| Attendance references person from wrong campus | Cross-campus person_id | 400 (validation error) |

### Data Access Tests

| Test Case | Description | Expected |
|-----------|------------|----------|
| View person creates audit log | GET /persons/:id | Audit entry with action_type=READ |
| Edit person logs old/new values | PUT /persons/:id | Audit entry with JSONB diff |
| Export creates audit log | GET /reports/.../export | Audit entry with action_type=EXPORT |
| Failed access creates audit log | 403 response | Audit entry with success=false |

---

## Layer 3: Dependency Scanning

### Automated Scanning

| Tool | Scope | Trigger |
|------|-------|---------|
| **Dependabot** (GitHub) | Go modules + npm packages | Daily check; auto-creates PRs |
| **govulncheck** | Go vulnerability database | CI pipeline on every PR |
| **npm audit** | Node.js dependency vulnerabilities | CI pipeline on every PR |
| **Trivy** | Docker image vulnerabilities (including Keycloak container image) | CI pipeline on image build |

### CI Integration

```yaml
# In GitHub Actions workflow
- name: Go vulnerability check
  run: go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...

- name: npm audit
  run: cd frontend && npm audit --audit-level=high

- name: Docker image scan (API)
  uses: aquasecurity/trivy-action@master
  with:
    image-ref: 'chesed-api:latest'
    severity: 'HIGH,CRITICAL'

- name: Docker image scan (Keycloak)
  uses: aquasecurity/trivy-action@master
  with:
    image-ref: 'quay.io/keycloak/keycloak:latest'
    severity: 'HIGH,CRITICAL'
```

### SBOM Generation (Phase 2)

- Generate Software Bill of Materials (SBOM) for all container images (API and Keycloak) using Trivy or Syft
- SBOMs are stored as CI artifacts and can be used for compliance audits

### Policy

- **Critical vulnerabilities**: Block merge; fix immediately
- **High vulnerabilities**: Fix within 7 days
- **Medium vulnerabilities**: Fix within 30 days
- **Low vulnerabilities**: Fix at next convenience

---

## Layer 4: Penetration Testing (Phase 3)

### OWASP Top 10 Checklist

| # | Vulnerability | Test Approach | Implementation Defense |
|---|--------------|--------------|----------------------|
| A01 | Broken Access Control | RBAC tests + campus isolation tests | Middleware-enforced RBAC; campus filter on all queries |
| A02 | Cryptographic Failures | Token security tests; TLS verification | Keycloak-managed credentials; RS256 token signatures; JWKS-based validation; TLS 1.3 |
| A03 | Injection | Input validation tests; parameterized query verification | pgx parameterized queries; struct validation |
| A04 | Insecure Design | Architecture review | Threat model (doc 18); defense-in-depth |
| A05 | Security Misconfiguration | Infrastructure scan; header checks | CSP, HSTS, X-Frame-Options headers |
| A06 | Vulnerable Components | Dependency scanning (Layer 3) | Automated scanning + policy |
| A07 | Auth Failures | Authentication test suite | Keycloak brute-force detection; Keycloak-managed token lifecycle; no local credential storage |
| A08 | Software/Data Integrity | Supply chain verification | Lock files; hash verification |
| A09 | Logging Failures | Audit log tests | Comprehensive audit logging |
| A10 | SSRF | API input validation | No user-controlled URLs in backend requests |

### Keycloak-Specific Tests

- Verify that the Keycloak admin console is not publicly accessible (restricted to internal network or VPN)
- Verify that the OIDC discovery endpoint (`/.well-known/openid-configuration`) is properly configured and returns correct issuer, JWKS URI, and supported scopes
- Verify that unused Keycloak flows (e.g., direct access grants in production clients) are disabled

### Manual Testing Scope

When resources allow, conduct manual testing focused on:
1. Authentication bypass attempts
2. RBAC escalation (can a volunteer access admin endpoints?)
3. Campus isolation breach (can campus A user see campus B data?)
4. Offline sync data manipulation (can a crafted sync payload inject data?)
5. File upload abuse (oversized files, malicious content types)
6. Rate limit bypass

---

## Security Headers Validation

The CI pipeline validates that all HTTP responses from the API include the required security headers. This check runs on every PR and blocks merge on failure.

### Required Headers

| Header | Expected Value |
|--------|---------------|
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains` |
| `Content-Security-Policy` | `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'` |
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `Referrer-Policy` | `strict-origin-when-cross-origin` |
| `Permissions-Policy` | `camera=(), microphone=(), geolocation=()` |

### Sample Validation Script

```bash
#!/usr/bin/env bash
# security-headers-check.sh — Run against a live or test server
set -euo pipefail

BASE_URL="${1:-http://localhost:8080}"
ENDPOINT="${BASE_URL}/api/v1/health"
FAILED=0

check_header() {
    local header_name="$1"
    local expected="$2"
    local actual
    actual=$(curl -sI "$ENDPOINT" | grep -i "^${header_name}:" | sed "s/^${header_name}: //i" | tr -d '\r')

    if [[ -z "$actual" ]]; then
        echo "FAIL: Missing header '${header_name}'"
        FAILED=1
    elif [[ "$actual" != *"$expected"* ]]; then
        echo "FAIL: Header '${header_name}' expected to contain '${expected}', got '${actual}'"
        FAILED=1
    else
        echo "PASS: ${header_name}"
    fi
}

check_header "Strict-Transport-Security" "max-age=31536000"
check_header "Content-Security-Policy" "default-src 'self'"
check_header "X-Content-Type-Options" "nosniff"
check_header "X-Frame-Options" "DENY"
check_header "Referrer-Policy" "strict-origin-when-cross-origin"
check_header "Permissions-Policy" "camera=()"

if [[ $FAILED -ne 0 ]]; then
    echo "Security headers check FAILED"
    exit 1
fi

echo "All security headers OK"
```

### CI Integration

```yaml
- name: Security headers check
  run: |
    # Start the API server in test mode
    ./chesed-api &
    sleep 2
    bash scripts/security-headers-check.sh http://localhost:8080
    kill %1
```

---

## Security Test Data

### Test Users for Security Tests

```go
var securityTestUsers = []TestUser{
    {Email: "admin@test.com", Profile: "ADMIN", CampusID: campusA},
    {Email: "coordinator@test.com", Profile: "COORDINATOR", CampusID: campusA},
    {Email: "professional@test.com", Profile: "PROFESSIONAL", CampusID: campusA},
    {Email: "secretary@test.com", Profile: "SECRETARY", CampusID: campusA},
    {Email: "volunteer@test.com", Profile: "VOLUNTEER", CampusID: campusA},
    {Email: "other-campus@test.com", Profile: "COORDINATOR", CampusID: campusB},
}
```

### Test Authentication Setup

Integration tests requiring authentication should use one of the following approaches:

1. **Keycloak Admin API**: Create test users programmatically via the Keycloak Admin REST API before test execution, and clean them up afterward.
2. **Pre-configured test realm**: Use a dedicated test realm with test fixtures exported as JSON (`keycloak/test-realm-export.json`), imported into a test Keycloak instance at CI startup.

Both approaches ensure tests are self-contained and do not depend on manual Keycloak configuration.

### Test Isolation

- Security tests use a dedicated test database
- Each test creates its own users and data
- Tests clean up after themselves (transaction rollback)
- No shared mutable state between tests

---

## Reporting

Security test results are tracked as:
1. **CI metrics**: Pass/fail counts visible in GitHub Actions
2. **Vulnerability reports**: Dependabot alerts in GitHub Security tab
3. **Manual findings**: Documented in `docs/security-findings/` (if any issues found)

### Regression Policy

Every security bug found (in testing or production) becomes a permanent regression test. The test must:
1. Reproduce the vulnerability
2. Verify the fix prevents it
3. Never be removed from the test suite
