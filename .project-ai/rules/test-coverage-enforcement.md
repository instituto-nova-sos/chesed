# Rule: Test Coverage Enforcement

## Purpose

Enforce minimum test coverage thresholds per architectural layer, ensuring that critical business logic, data access, and user interactions are adequately tested. Extends the general quality-gates rule with layer-specific granularity.

## Rule Statement

Every code change must meet the following minimum test coverage thresholds by architectural layer. Coverage is measured by branch coverage (not just line coverage) to ensure decision paths are tested.

## Thresholds

### Backend (Go)

| Layer | Coverage Target | Test Type | Rationale |
|-------|----------------|-----------|-----------|
| Service layer | ≥ 90% branch | Unit tests (mocked repos) | Contains business logic and validation |
| Handler layer | ≥ 80% branch | Unit tests (mocked services) | Request parsing, response formatting, error mapping |
| Repository layer | All CRUD tested | Integration tests (real PostgreSQL) | Query correctness, campus isolation, pagination |
| Middleware | ≥ 80% branch | Unit tests | Auth validation, RBAC enforcement |

### Frontend (React/TypeScript)

| Layer | Coverage Target | Test Type | Rationale |
|-------|----------------|-----------|-----------|
| Custom hooks | ≥ 80% branch | Unit tests (React Testing Library) | State management and API interaction logic |
| Form components | Validation + submission | Integration tests | User-facing input validation |
| API client functions | ≥ 70% branch | Unit tests (mocked HTTP) | Request/response handling |
| Utility functions | ≥ 90% branch | Unit tests | Pure logic must be fully tested |

### Cross-Cutting

| Area | Requirement | Test Type |
|------|-------------|-----------|
| Authentication | Valid, expired, missing token scenarios | Integration tests |
| RBAC | Each role tested against protected endpoints | Integration tests |
| Campus isolation | Cross-campus access denied | Integration tests |
| Audit logging | Mutations create audit entries | Integration tests |
| Offline sync | Queue, retry, conflict resolution | Unit + integration tests |

## Trigger Condition

- Every pull request: new code coverage evaluated against thresholds.
- Every sprint release: overall coverage evaluated.
- Coverage thresholds tighten over time:
  - Sprint 1-2: overall ≥ 70%
  - Sprint 3+: overall ≥ 80%

## Enforcement Mechanism

- **Pre-merge hook**: Blocks merge if new code coverage < 80% overall.
- **Reviewer agent**: Evaluates layer-specific coverage during PR review.
- **QA engineer agent**: Runs `execute-test-plan` skill to validate test completeness.
- **CI/CD pipeline**: Generates coverage reports per test run.

## Handling Failures

1. Identify which layer is below threshold.
2. Use `execute-test-plan` skill to find specific untested code paths.
3. Add tests for the highest-priority gaps (BLOCKER severity first).
4. Re-run coverage analysis.
5. If a threshold genuinely cannot be met (e.g., generated code, infrastructure glue):
   - Document exception in PR description.
   - Get tech-lead approval.
   - Exclude specific files via coverage configuration (not by lowering thresholds).

## Consequences of Skipping

- Business logic bugs reach production undetected.
- Regressions introduced by future changes go uncaught.
- Campus isolation failures expose PII across tenants (LGPD violation risk).
- State machine bugs cause data corruption in triage/attendance workflows.
- Offline sync bugs cause data loss or duplication.

## References

- `docs/quality/quality-gates.md` — Overall quality gate conditions
- `docs/quality/quality-profiles.md` — Stack-specific testing requirements
- `docs/17-security-test-strategy.md` — Security test requirements
- `docs/15-implementation-guidelines.md` — Test conventions
