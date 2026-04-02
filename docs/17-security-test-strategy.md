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
│  Layer 1: Unit Security Tests        │  Input validation, crypto, sanitization
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

### Password Hashing

```go
func TestPasswordHashing(t *testing.T) {
    password := "secure_password_123"
    hash, err := HashPassword(password)
    assert.NoError(t, err)
    assert.NotEqual(t, password, hash)
    assert.True(t, CheckPassword(password, hash))
    assert.False(t, CheckPassword("wrong_password", hash))
}
```

### JWT Token Security

```go
func TestJWT_ExpiredToken(t *testing.T) {
    token := generateToken(userID, -1*time.Hour) // Already expired
    _, err := ValidateToken(token)
    assert.ErrorIs(t, err, ErrTokenExpired)
}

func TestJWT_TamperedToken(t *testing.T) {
    token := generateToken(userID, 15*time.Minute)
    tampered := token[:len(token)-5] + "XXXXX"
    _, err := ValidateToken(tampered)
    assert.Error(t, err)
}

func TestJWT_WrongSigningKey(t *testing.T) {
    token := generateTokenWithKey(userID, 15*time.Minute, "wrong_key")
    _, err := ValidateToken(token)
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
| Login with valid credentials | Email + password match | 200 + tokens |
| Login with wrong password | Correct email, wrong password | 401 |
| Login with non-existent email | Unknown email | 401 (same error as wrong password) |
| Login with deactivated account | Valid credentials, is_active=false | 401 |
| Access with expired token | Expired JWT | 401 |
| Access with no token | Missing Authorization header | 401 |
| Refresh with valid refresh token | Non-expired refresh | 200 + new access token |
| Refresh with revoked token | Previously revoked | 401 |
| Account lockout after 10 failures | 10 wrong passwords | 429 (locked) |

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
| **Trivy** | Docker image vulnerabilities | CI pipeline on image build |

### CI Integration

```yaml
# In GitHub Actions workflow
- name: Go vulnerability check
  run: go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...

- name: npm audit
  run: cd frontend && npm audit --audit-level=high

- name: Docker image scan
  uses: aquasecurity/trivy-action@master
  with:
    image-ref: 'chesed-api:latest'
    severity: 'HIGH,CRITICAL'
```

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
| A02 | Cryptographic Failures | Token security tests; TLS verification | bcrypt passwords; JWT with HS256; TLS 1.3 |
| A03 | Injection | Input validation tests; parameterized query verification | pgx parameterized queries; struct validation |
| A04 | Insecure Design | Architecture review | Threat model (doc 18); defense-in-depth |
| A05 | Security Misconfiguration | Infrastructure scan; header checks | CSP, HSTS, X-Frame-Options headers |
| A06 | Vulnerable Components | Dependency scanning (Layer 3) | Automated scanning + policy |
| A07 | Auth Failures | Authentication test suite | Lockout, token expiry, secure password storage |
| A08 | Software/Data Integrity | Supply chain verification | Lock files; hash verification |
| A09 | Logging Failures | Audit log tests | Comprehensive audit logging |
| A10 | SSRF | API input validation | No user-controlled URLs in backend requests |

### Manual Testing Scope

When resources allow, conduct manual testing focused on:
1. Authentication bypass attempts
2. RBAC escalation (can a volunteer access admin endpoints?)
3. Campus isolation breach (can campus A user see campus B data?)
4. Offline sync data manipulation (can a crafted sync payload inject data?)
5. File upload abuse (oversized files, malicious content types)
6. Rate limit bypass

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
