# Quality Profiles

This document defines the quality profiles for the Chesed project. A quality profile is the set of rules, thresholds, and practices that code must satisfy before it is accepted into the codebase.

There are two profiles: **Backend (Go)** and **Frontend (React/TypeScript)**. Both profiles enforce three core software qualities: **Security**, **Reliability**, and **Maintainability**.

---

## Backend Quality Profile (Go)

### Language and Stack

- Go with `chi` router, `pgx`, `golang-migrate`, `slog`, `coreos/go-oidc`
- PostgreSQL 16
- Keycloak (OIDC) for authentication

### Code Standards

| Dimension | Requirement |
|-----------|------------|
| Error handling | Every error must be handled. No `_` for error returns. Errors wrapped with `fmt.Errorf("scope.Method: %w", err)`. |
| Context propagation | All I/O functions accept `context.Context` as first parameter. No `context.Background()` in production code. |
| Interface design | Interfaces defined at the consumption site (service package). Minimal interfaces. |
| Dependency direction | handler → service → repository interface → domain. No reverse imports. No circular dependencies. |
| Naming | Packages: lowercase single word. Exported: PascalCase. Unexported: camelCase. Files: snake_case. Constructors: `NewXxx(deps) *Xxx`. |
| Logging | `slog` only. Structured fields. No PII in logs. Appropriate severity levels. |
| Database | `pgx` only. Parameterized queries. All queries include `campus_id` filter. Transactions for multi-statement mutations. |
| Testing | `testing` + `testify`. Table-driven tests. Integration tests with real PostgreSQL for repositories. |

### Complexity Thresholds

| Metric | Limit |
|--------|-------|
| Cognitive complexity per function | 25 |
| Cyclomatic complexity per function | 10 |
| Function length | 40 lines |
| File length | 400 lines |
| Nesting depth | 3 levels |
| Parameter count | 5 |
| Return values | 3 |

Full details: [`docs/quality/complexity-guidelines.md`](complexity-guidelines.md)

### Clean Code Categories

All Go code must satisfy: **Consistency**, **Intentionality**, **Adaptability**, and **Responsibility**.

Full details: [`docs/quality/clean-code-guidelines.md`](clean-code-guidelines.md)

---

## Frontend Quality Profile (React/TypeScript)

### Language and Stack

- React + TypeScript (strict mode) + Vite
- Tailwind CSS, React Hook Form + Zod, Dexie.js, Zustand
- `keycloak-js` for OIDC authentication

### Code Standards

| Dimension | Requirement |
|-----------|------------|
| TypeScript strictness | No `any` type. No `@ts-ignore` or `@ts-nocheck`. Explicit return types on exported functions. Union types for known string values. |
| Components | Functional only. Under 150 lines. No direct API calls (use hooks). Props interface defined and exported. |
| Hooks | Prefixed with `use`. No side effects outside `useEffect`. Complete dependency arrays. Loading and error states handled. |
| Forms | React Hook Form with Zod resolver. Validation schema co-located with type. Error messages in pt-BR. Accessible fields. |
| Styling | Tailwind CSS only. Mobile-first breakpoints. Responsive at 320px width. |
| Authentication | `keycloak-js` adapter only. Auth context provides user info. API client auto-attaches Bearer token. Protected routes check auth. |
| Testing | Vitest. React Testing Library for components. Hook tests for custom hooks. |

### Complexity Thresholds

| Metric | Limit |
|--------|-------|
| Cognitive complexity per function | 15 |
| Cyclomatic complexity per function | 10 |
| Function length | 50 lines |
| File length | 300 lines |
| Nesting depth | 3 levels |
| Parameter count | 5 |
| Component JSX lines | 80 |

Full details: [`docs/quality/complexity-guidelines.md`](complexity-guidelines.md)

### Clean Code Categories

All React/TypeScript code must satisfy: **Consistency**, **Intentionality**, **Adaptability**, and **Responsibility**.

Full details: [`docs/quality/clean-code-guidelines.md`](clean-code-guidelines.md)

---

## Software Quality Dimensions

Both profiles enforce three non-negotiable software qualities. These are engineering constraints, not aspirational goals.

### Security

Security is embedded in design and implementation decisions.

#### Requirements

- Protection against OWASP Top 10 and API Security Top 10
- Secure authentication and authorization via Keycloak (no custom auth)
- Input validation on all external boundaries
- Output sanitization to prevent injection attacks
- Encryption in transit (TLS) and at rest (IndexedDB encryption for PII)
- Secrets management — no hardcoded credentials anywhere
- Safe error handling — no sensitive data in error responses or logs
- Audit logging for all sensitive operations
- Least privilege principle for RBAC roles
- Campus isolation on all data queries

#### Evaluation Criteria

| Condition | Threshold |
|-----------|-----------|
| High or critical vulnerabilities | 0 |
| Exposed secrets | 0 |
| Insecure access patterns | 0 |
| Security hotspots reviewed | 100% |
| Security rating | A |

#### Violation Examples

- Hardcoded credentials or API keys in source code
- Missing RBAC middleware on an endpoint
- Endpoint accepting requests without token validation
- PII logged in error messages or application logs
- Client-side-only validation without server-side enforcement
- SQL injection via string concatenation in queries
- Missing `campus_id` filter allowing cross-tenant data access

### Reliability

Reliability ensures predictable and correct behavior under all conditions.

#### Requirements

- Every error handled and propagated with context — no silent failures
- Custom domain errors for expected failure modes (`ErrNotFound`, `ErrDuplicate`)
- Retry strategies for transient external failures where appropriate
- Idempotency for critical write operations
- Consistent state transitions (attendance lifecycle, triage states)
- Validation of external dependencies (Keycloak availability, database connectivity)
- Concurrency safety — no race conditions in shared state
- Timeouts on all external calls (database, Keycloak, HTTP)
- Graceful degradation for offline scenarios (frontend)
- Transaction boundaries for multi-statement database mutations

#### Evaluation Criteria

| Condition | Threshold |
|-----------|-----------|
| High severity bugs | 0 |
| Unhandled errors | 0 |
| Inconsistent system states | 0 |
| Reliability rating | A |

#### Violation Examples

- Ignored errors (`_` in Go, unhandled promise rejections in TS)
- Runtime panics from nil pointer dereference
- Race conditions in concurrent handlers
- Inconsistent state after partial failure (e.g., record created but audit log missing)
- Missing timeout on database query allowing indefinite hangs
- Sync queue corruption after network failure

### Maintainability

Maintainability ensures long-term evolvability and developer productivity.

#### Requirements

- Low cognitive complexity (within profile thresholds)
- Modular design — clear separation of concerns across layers
- Consistent naming — aligned with naming conventions in each profile
- Minimal duplication — shared logic extracted to appropriate layer
- Readable, intention-revealing code — code explains what and why
- Appropriate abstraction levels — no premature abstraction, no missing abstraction
- Alignment between code and documentation — API docs, data model docs, domain docs current
- Clean dependency graph — no circular imports, correct layer direction

#### Evaluation Criteria

| Condition | Threshold |
|-----------|-----------|
| Maintainability rating | A |
| Duplicated lines (new code) | < 3% |
| Duplicated lines (overall) | < 5% |
| Technical debt ratio | < 5% |

#### Violation Examples

- Functions exceeding cognitive complexity threshold
- Deep nesting (> 3 levels)
- Duplicated business logic across handlers or services
- God functions with multiple responsibilities
- Unclear naming that requires comments to explain
- Handler containing business logic that belongs in a service
- Component with 300+ lines without extraction

---

## Cross-Quality Enforcement

These qualities are enforced through:

| Mechanism | How |
|-----------|-----|
| Quality Gates | Numeric thresholds block merge on failure. See [`docs/quality/quality-gates.md`](quality-gates.md). |
| Skills | `review-code`, `maintainability-analysis`, `reliability-validation`, `review-security` evaluate compliance. |
| Agents | Reviewer agent blocks PRs on quality gate failure. Tech Lead enforces architecture quality. Security Engineer enforces security dimension. |
| Hooks | `pre-merge` enforces quality gates. `pre-review` validates quality checks completed. `pre-release` enforces full compliance. |
| Checklists | Backend and frontend feature-complete checklists include quality profile items. PR quality checklist validates all dimensions. |
| Playbooks | `implement-with-quality` ensures quality-aware implementation. `refactor-for-quality` addresses violations. |

**No feature is complete if it violates any of these qualities.**

---

## References

| Document | Path |
|----------|------|
| Clean code guidelines | [`docs/quality/clean-code-guidelines.md`](clean-code-guidelines.md) |
| Quality gates | [`docs/quality/quality-gates.md`](quality-gates.md) |
| Complexity guidelines | [`docs/quality/complexity-guidelines.md`](complexity-guidelines.md) |
| Implementation guidelines | `docs/15-implementation-guidelines.md` |
| Security and compliance | `docs/13-security-and-compliance.md` |
| Secure development standard | `docs/19-secure-development-standard.md` |
| Project rules | `CLAUDE.md` |
