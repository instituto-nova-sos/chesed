# Rule: Dependency Management

## Purpose

Ensure all external dependencies are evaluated, justified, and approved before being added to the project. Prevent dependency bloat, license conflicts, and supply chain vulnerabilities.

## Rule Statement

No new external dependency may be added to `go.mod` or `package.json` without explicit justification and tech-lead approval. Each dependency must be evaluated against the criteria below.

## Evaluation Criteria

| Criterion | Requirement | How to Check |
|-----------|-------------|-------------|
| **License compatibility** | MIT, BSD-2, BSD-3, Apache 2.0, ISC preferred. GPL/LGPL require ADR. | Check LICENSE file in dependency repo |
| **Maintenance status** | Active maintenance within last 12 months | Check last commit date, open issues response time |
| **Security history** | No unpatched CRITICAL/HIGH CVEs | Run `govulncheck` (Go) or `npm audit` (frontend) |
| **Popularity/trust** | Reasonable adoption (stars, downloads, known maintainers) | Check GitHub stars, npm weekly downloads |
| **Size impact** | Proportional to functionality gained | Check module size, transitive dependencies count |
| **Alternative in stack** | Prefer existing approved dependencies over new ones | Check if approved stack already covers the need |

### Approved Dependencies (Pre-Approved)

**Backend (Go):**
- `go-chi/chi` — HTTP routing
- `jackc/pgx` — PostgreSQL driver
- `golang-migrate/migrate` — Database migrations
- `coreos/go-oidc` — OIDC token validation
- `go-playground/validator` — Struct validation
- `stretchr/testify` — Test assertions and mocks
- `google/uuid` — UUID generation

**Frontend (React/TypeScript):**
- `react`, `react-dom` — UI framework
- `react-router-dom` — Client routing
- `react-hook-form` + `@hookform/resolvers` — Form management
- `zod` — Schema validation
- `keycloak-js` — OIDC authentication
- `dexie` — IndexedDB abstraction
- `tailwindcss` — Utility-first CSS
- `vitest` + `@testing-library/react` — Testing
- `recharts` — Charts (Phase 2)

Dependencies on this list do not require additional justification.

## Trigger Condition

- Any `go get` command adding a new module to `go.mod`.
- Any `npm install` adding a new package to `package.json`.
- Any PR that modifies `go.mod`, `go.sum`, `package.json`, or `package-lock.json` with new entries.

## Enforcement Mechanism

- **Pre-review hook**: Checks for new entries in `go.mod` or `package.json`. If found, requires justification.
- **Reviewer agent**: Verifies justification exists and evaluation criteria are met.
- **CI/CD pipeline**: Runs `govulncheck` and `npm audit` on every PR.
- **Tech-lead agent**: Approves or rejects non-pre-approved dependencies.

## Justification Format

When adding a new dependency, include in the PR description:

```markdown
### New Dependency: {package-name}

- **Purpose**: Why this dependency is needed
- **License**: MIT/BSD/Apache/other
- **Maintenance**: Last commit date, maintainer activity
- **Alternatives considered**: Why existing stack doesn't cover this need
- **Size impact**: Module size, number of transitive dependencies
- **Security**: govulncheck/npm audit result
```

## Consequences of Skipping

- License-incompatible dependencies create legal exposure.
- Unmaintained dependencies become security liabilities.
- Unnecessary dependencies increase attack surface and build times.
- Transitive dependency bloat complicates vulnerability management.

## References

- `CLAUDE.md` — Approved technology stack (Implementation Constraints)
- `docs/05-architecture-proposal.md` — Architecture decisions and approved libraries
- `docs/13-security-and-compliance.md` — Security requirements
- `docs/14-deployment-strategy.md` — CI/CD security scanning
