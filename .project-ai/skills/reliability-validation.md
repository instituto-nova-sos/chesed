# Skill: Reliability Validation

## Purpose

Validate code for reliability issues including error handling, state consistency, fault tolerance, concurrency safety, and graceful degradation. Ensures the system behaves predictably under normal, degraded, and failure conditions.

## When to Use / Trigger

- After implementing features that involve error handling, state transitions, or external dependencies.
- When reviewing code that handles database transactions, Keycloak integration, or offline sync.
- When a quality gate flags reliability issues.
- Invoked by reviewer agent during PR review.
- Invoked by tech-lead for release gating.

## Role / Expertise

Senior developer with expertise in:
- Error handling patterns in Go and TypeScript.
- Distributed systems reliability (timeouts, retries, circuit breakers).
- Database transaction management.
- Offline-first application resilience.
- State machine correctness (triage and attendance lifecycles).

## Inputs

| Input | Required | Source |
|-------|----------|--------|
| Files to validate | Yes | File paths or git diff |
| Quality profile | Yes | `docs/quality/quality-profiles.md` |
| Domain model | Yes | `docs/04-domain-model.md` |
| Offline sync strategy | If offline code | `docs/12-offline-sync-strategy.md` |

## Process

### 1. Error Handling Validation

#### Go
- [ ] Every error return is checked — no `_` for errors.
- [ ] Errors wrapped with context: `fmt.Errorf("scope.Method: %w", err)`.
- [ ] Custom domain errors defined for expected failures: `ErrNotFound`, `ErrDuplicate`, `ErrInvalidTransition`.
- [ ] Errors propagated up the stack — not swallowed silently.
- [ ] Error types distinguish recoverable from unrecoverable failures.
- [ ] Handlers translate domain errors to appropriate HTTP status codes.
- [ ] No `panic` in production code (except truly unrecoverable startup errors).

#### React/TypeScript
- [ ] All async operations have error handling (try/catch or `.catch()`).
- [ ] No unhandled promise rejections.
- [ ] Error boundaries present for component trees that can fail.
- [ ] Error states displayed to users with actionable messages.
- [ ] Network errors handled with offline-aware fallbacks.
- [ ] Form submission errors displayed without losing user input.

### 2. State Consistency

- [ ] State transitions follow defined lifecycle rules:
  - Triage: documented transitions per `docs/04-domain-model.md`.
  - Attendance: SCHEDULED → IN_PROGRESS → COMPLETED/CANCELLED.
- [ ] Invalid state transitions are rejected with clear error messages.
- [ ] Database transactions used for multi-statement mutations (all-or-nothing).
- [ ] Audit log entries created within the same transaction as the mutation.
- [ ] No partial state left on failure (e.g., record created but audit log missing).
- [ ] Optimistic updates in frontend rolled back on server error.

### 3. Fault Tolerance

- [ ] Timeouts configured on all external calls:
  - Database queries: reasonable timeout via context.
  - Keycloak JWKS fetch: timeout configured.
  - HTTP client calls: timeout set.
- [ ] Transient failures handled appropriately:
  - Database connection: connection pool handles reconnection.
  - Keycloak unavailability: cached JWKS used if available.
- [ ] Critical operations are idempotent where possible (using UUIDs for creation).
- [ ] Sync queue tolerates network interruptions without data loss.

### 4. Concurrency Safety

- [ ] No shared mutable state without synchronization.
- [ ] Database operations use appropriate isolation levels.
- [ ] No race conditions in handler registration or middleware setup.
- [ ] Frontend state management (Zustand/Context) handles concurrent updates.
- [ ] Offline sync queue processes items sequentially to avoid conflicts.

### 5. Graceful Degradation (Frontend)

- [ ] Application remains usable when offline (core flows work).
- [ ] Cached data displayed when network is unavailable.
- [ ] Sync queue persists mutations until connectivity is restored.
- [ ] User is informed of degraded state (offline indicator).
- [ ] No blank screens or crashes when external services are unavailable.

### 6. Input Validation

- [ ] All external input validated at system boundaries (handlers, API client responses).
- [ ] Validation errors return structured error responses.
- [ ] No trust in client-side validation — server always re-validates.
- [ ] Query parameters and path variables validated before use.
- [ ] Request body size limits enforced.

## Outputs / Deliverables

A reliability report with:

1. **Reliability Rating**: A through E (per rating definitions in `docs/quality/quality-gates.md`).
2. **Error Handling Findings**: Unhandled errors, missing context, swallowed errors.
3. **State Consistency Findings**: Invalid transitions possible, partial state risks, missing transactions.
4. **Fault Tolerance Findings**: Missing timeouts, unhandled transient failures.
5. **Concurrency Findings**: Race conditions, unsafe shared state.
6. **Degradation Findings**: Missing offline fallbacks, crash scenarios.
7. **Verdict**: PASS (reliability rating A) or FAIL with required fixes.

## References

| Document | Path | Usage |
|----------|------|-------|
| Quality profiles | `docs/quality/quality-profiles.md` | Reliability requirements |
| Quality gates | `docs/quality/quality-gates.md` | Pass/fail criteria |
| Domain model | `docs/04-domain-model.md` | State transition rules |
| Offline sync | `docs/12-offline-sync-strategy.md` | Degradation requirements |
| Architecture | `docs/05-architecture-proposal.md` | Layer responsibilities |

## Constraints / Quality Bar

- Any unhandled error in Go (using `_`) is a BLOCKER.
- Any missing transaction for multi-statement mutation is a BLOCKER.
- Any invalid state transition allowed by the code is a BLOCKER.
- Missing timeout on external call is a MAJOR issue.
- Missing offline fallback for a core flow is a MAJOR issue.

## Interaction with Other Artifacts

- **Invoked by agents**: reviewer (PR review), tech-lead (release gating), backend-engineer (self-check).
- **Used alongside skills**: review-code (broader review), maintainability-analysis, review-security.
- **Informs playbooks**: implement-with-quality (reliability checks during implementation).
- **Blocks**: pre-merge hook (BLOCKER issues block merge).
