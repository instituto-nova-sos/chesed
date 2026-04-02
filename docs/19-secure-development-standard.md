# 19 - Secure Development Standard

## Purpose

This document defines the security practices that MUST be followed during development of Chesed. It applies to all contributors — human and AI agents alike.

---

## Secure Coding Rules

### 1. Input Validation

**Rule**: Never trust client input. Validate all inputs on the server side.

| Input Type | Validation | Go Implementation |
|-----------|-----------|-------------------|
| String fields | Max length, allowed characters, trim whitespace | `validator:"required,max=200"` |
| Email | RFC 5322 format | `validator:"required,email"` |
| UUID | Valid UUID v4 format | `validator:"required,uuid4"` |
| Dates | Valid date, reasonable range | Custom validator |
| Enums | Must match defined set | `validator:"oneof=M F OTHER PREFER_NOT_TO_SAY"` |
| Document numbers (CPF) | Format + checksum validation | Custom CPF validator |
| Free text (observations) | Max length, strip HTML tags | `validator:"max=5000"` + sanitization |
| File uploads | Type whitelist, size limit, content verification | Middleware |

**Forbidden patterns:**
```go
// NEVER: String concatenation in SQL
query := "SELECT * FROM person WHERE name = '" + name + "'"

// ALWAYS: Parameterized queries
query := "SELECT * FROM person WHERE name = $1"
rows, err := db.Query(ctx, query, name)
```

### 2. Authentication Security

**Rule**: Authentication logic must be centralized in middleware, never duplicated in handlers.

- All passwords hashed with bcrypt (minimum cost 12)
- JWT signed with HS256 using a secret from environment variables
- No sensitive data in JWT payload (no passwords, no PII beyond what's needed for auth)
- Refresh tokens stored hashed in the database (not in plaintext)
- Generic error messages for auth failures (no user enumeration)

```go
// NEVER: Different error messages for "user not found" vs "wrong password"
if user == nil {
    return errors.New("user not found")  // WRONG: reveals existence
}

// ALWAYS: Same error for all auth failures
return errors.New("invalid credentials")
```

### 3. Authorization Enforcement

**Rule**: Every handler must declare and enforce required roles.

```go
// Route registration with RBAC
r.With(middleware.RequireRole("COORDINATOR", "ADMIN")).
    Get("/reports/attendances", handler.GetAttendanceReport)
```

- Role is extracted from JWT claims, never from request body or query params
- Campus scoping is applied automatically by middleware
- No "admin backdoors" or hidden endpoints

### 4. Data Protection

**Rule**: PII must never appear in logs, error responses, or debug output.

```go
// NEVER: Log PII
slog.Info("created person", "name", person.FullName, "cpf", person.DocumentNumber)

// ALWAYS: Log identifiers only
slog.Info("created person", "person_id", person.ID, "campus_id", person.CampusID)
```

**Error responses** must not include:
- Stack traces
- Database error messages
- Internal field names
- PII from the request

```go
// NEVER: Expose internal errors
http.Error(w, err.Error(), 500)

// ALWAYS: Generic error with structured logging
slog.Error("failed to create person", "error", err, "person_id", input.ID)
writeJSON(w, 500, ErrorResponse{Error: "internal_error", Message: "An unexpected error occurred"})
```

### 5. SQL Security

**Rule**: Always use parameterized queries. Never build SQL with string concatenation.

- Use `pgx` parameterized queries (`$1`, `$2`, etc.)
- Never use `fmt.Sprintf` to build SQL
- Use query builder patterns for dynamic filters (safe construction)
- Test that special characters in input don't break queries

### 6. File Upload Security

**Rule**: Validate file uploads before storage.

| Check | Implementation |
|-------|---------------|
| File type | Whitelist: `image/jpeg`, `image/png`, `application/pdf` |
| File size | Maximum 10MB |
| Content verification | Read magic bytes; don't trust Content-Type header |
| File name | Sanitize; replace with UUID-based name |
| Storage | Object storage (S3); never on application server filesystem |
| Access | Presigned URLs with expiration; never serve directly |

### 7. CORS Configuration

**Rule**: CORS must explicitly whitelist allowed origins.

```go
cors := middleware.CORS{
    AllowedOrigins:   []string{"https://chesed.pages.dev", "http://localhost:5173"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
    AllowedHeaders:   []string{"Authorization", "Content-Type"},
    ExposedHeaders:   []string{"X-Request-ID"},
    AllowCredentials: true,
    MaxAge:           3600,
}
```

Never use `AllowedOrigins: ["*"]` in production.

### 8. HTTP Security Headers

The reverse proxy or Go middleware must set:

```
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 0
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: camera=(), microphone=(), geolocation=()
```

---

## Secret Management

### Rules
1. **No secrets in source code** — ever. Not even "development" secrets.
2. **No secrets in Docker images** — use environment variables at runtime.
3. **No secrets in git history** — if accidentally committed, rotate immediately.
4. **`.env` files are gitignored** — only `.env.example` (with placeholder values) is committed.

### Required Secrets

| Secret | Source | Rotation |
|--------|--------|----------|
| `JWT_SECRET` | Random 64+ characters | Every 6 months |
| `DB_PASSWORD` | Strong random password | Every 6 months |
| `S3_ACCESS_KEY` / `S3_SECRET_KEY` | Cloud provider IAM | Every 12 months |
| `ENCRYPTION_KEY` (Phase 2) | Random 32 bytes | Every 12 months |

### `.env.example`

```bash
# Application
APP_ENV=development
APP_PORT=8080
APP_LOG_LEVEL=info

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=chesed
DB_USER=chesed_user
DB_PASSWORD=CHANGE_ME
DB_SSL_MODE=disable

# JWT
JWT_SECRET=CHANGE_ME_TO_RANDOM_64_CHARS

# Storage
STORAGE_TYPE=local
S3_ENDPOINT=
S3_BUCKET=
S3_ACCESS_KEY=
S3_SECRET_KEY=

# CORS
CORS_ORIGINS=http://localhost:5173
```

---

## Dependency Management

### Rules
1. **Minimal dependencies**: Prefer Go stdlib over third-party packages
2. **Lock versions**: `go.sum` and `package-lock.json` must be committed
3. **Automated scanning**: Dependabot or Renovate enabled
4. **No vulnerable dependencies**: CI blocks merge if `govulncheck` or `npm audit` finds high/critical issues

### Approved Dependencies

Only the following third-party dependencies are pre-approved. Adding new ones requires justification.

**Go:**
- `go-chi/chi` — HTTP router
- `jackc/pgx` — PostgreSQL driver
- `golang-migrate/migrate` — Migrations
- `golang-jwt/jwt` — JWT
- `go-playground/validator` — Validation
- `google/uuid` — UUID
- `stretchr/testify` — Test assertions

**React:**
- `react`, `react-dom`, `react-router-dom`
- `react-hook-form`, `zod`
- `dexie` — IndexedDB
- `tailwindcss`
- `vite`, `vite-plugin-pwa`
- `recharts` — Charts
- `react-signature-canvas` — Signature capture

Adding anything outside this list requires a note in the PR explaining why.

---

## CI Security Gates

The CI pipeline must enforce these gates before allowing merge:

| Gate | Tool | Blocks Merge |
|------|------|-------------|
| Go lint | `golangci-lint` | Yes |
| Go tests | `go test ./...` | Yes |
| Go vulnerability scan | `govulncheck` | Yes (high/critical) |
| TypeScript lint | `eslint` | Yes |
| TypeScript tests | `vitest` | Yes |
| npm audit | `npm audit --audit-level=high` | Yes (high/critical) |
| Docker image scan | `trivy` | Yes (critical) |
| Secret detection | `trufflehog` or `gitleaks` | Yes |

---

## Secure Review Checklist

For every PR that touches security-sensitive code (auth, RBAC, data access, sync, encryption):

- [ ] Input validation on all new fields
- [ ] Parameterized queries (no string concatenation in SQL)
- [ ] No PII in logs or error responses
- [ ] RBAC enforced on new endpoints
- [ ] Campus scoping on new queries
- [ ] Audit log entries for new data mutations
- [ ] Secrets loaded from environment, not hardcoded
- [ ] New dependencies justified and scanned
- [ ] Error responses are generic (no internal details exposed)
- [ ] Tests cover both authorized and unauthorized access attempts

---

## Incident Handling for Developers

If you discover a security issue during development:

1. **Do not commit the vulnerability** to a public branch
2. **Document the finding** with reproduction steps
3. **Fix before merge** — security issues are never "known tech debt"
4. **Add a regression test** that prevents the issue from recurring
5. **Update the threat model** (`docs/18-threat-model.md`) if the finding reveals a new attack vector

If you accidentally commit a secret:
1. **Rotate the secret immediately** — assume it's compromised
2. **Do not rely on git force-push** to remove it (it may already be cached/cloned)
3. **Update `.env.example`** if the variable was missing
4. **Add to `.gitignore`** if the file pattern was missing
