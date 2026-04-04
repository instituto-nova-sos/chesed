# Playbook: Implement with Quality

End-to-end guide for implementing features while maintaining quality compliance. This playbook wraps the standard implementation flow with quality checkpoints at every stage.

---

## When to Use

- Every feature implementation (backend, frontend, or full-stack).
- When implementing changes that must pass quality gates before merge.
- As a complement to `implement-backend-endpoint` and `implement-frontend-page` playbooks.

---

## Flow Overview

```
QUALITY PROFILE → COMPLEXITY BUDGET → IMPLEMENT → SELF-CHECK → QUALITY GATE → REVIEW
```

---

## Step 1: Review Quality Profile

Before writing any code, review the applicable quality profile from `docs/quality/quality-profiles.md`:

- **Backend (Go)**: Error handling, context propagation, interface design, dependency direction, naming, logging, database access, testing patterns.
- **Frontend (React/TS)**: TypeScript strictness, component quality, hooks, forms, styling, authentication, testing patterns.

Internalize the thresholds from `docs/quality/complexity-guidelines.md`:

| Metric | Go | React/TS |
|--------|-----|----------|
| Cognitive complexity | ≤ 25 | ≤ 15 |
| Cyclomatic complexity | ≤ 10 | ≤ 10 |
| Function length | ≤ 40 | ≤ 50 |
| File length | ≤ 400 | ≤ 300 |
| Nesting depth | ≤ 3 | ≤ 3 |

---

## Step 2: Set Complexity Budget

Before coding, estimate the complexity of the feature:

1. **Identify the functions** you will create or modify.
2. **Estimate cognitive complexity** for each. If any function is likely to exceed the threshold, plan the decomposition upfront:
   - Which helper functions to extract.
   - Which logic to move to a different layer.
   - Which conditions to simplify with guard clauses.
3. **Check file sizes** of files you will modify. If adding code would exceed file length limits, plan extraction of existing code into separate files.

---

## Step 3: Implement Following Clean Code Guidelines

During implementation, continuously apply the four clean code categories from `docs/quality/clean-code-guidelines.md`:

### Consistency
- Match patterns used in sibling files (naming, structure, error handling).
- Follow the established constructor, handler, service, repository patterns.
- Use the same import ordering and formatting.

### Intentionality
- Choose names that reveal purpose. No generic names (`data`, `result`, `temp`).
- Delete dead code immediately. No commented-out blocks.
- Write comments only for "why," never for "what."

### Adaptability
- Keep dependencies pointing inward (handler → service → repository → domain).
- Inject dependencies via constructors.
- Isolate external concerns behind interfaces.

### Responsibility
- Each function does one thing. One reason to change.
- Handlers: parse, validate, call service, respond. Nothing else.
- Services: business logic. No HTTP or SQL.
- Components: render UI. No API calls.

---

## Step 4: Write Tests Alongside Code

Do not defer testing. Write tests as you implement:

1. **Service layer**: Table-driven tests for every method. Cover happy path, validation failures, business rule violations, dependency failures.
2. **Repository layer**: Integration tests with real PostgreSQL.
3. **Handler layer**: `httptest` tests for status codes and response format.
4. **Frontend hooks**: Vitest tests for data fetching and state management.
5. **Frontend forms**: React Testing Library tests for validation and submission.

Coverage target: ≥ 80% on new code (per quality gate).

---

## Step 5: Self-Check Against Quality Gates

Before requesting review, run a self-check against the New Code Quality Gate from `docs/quality/quality-gates.md`:

- [ ] 0 new bugs (reliability issues)
- [ ] 0 new vulnerabilities (security issues)
- [ ] All security-sensitive code reviewed
- [ ] Coverage on new code ≥ 80%
- [ ] Duplication on new code ≤ 3%
- [ ] All functions within complexity thresholds
- [ ] Clean code categories satisfied

Check each changed function against complexity limits:
- Run through cognitively. Count branches, nesting, conditions.
- If a function feels complex, it probably exceeds the threshold. Refactor before proceeding.

---

## Step 6: Run Quality Validation

Execute the following skills:

1. **`review-code`**: Full code review with quality profile compliance, clean code assessment, complexity check, and quality gate validation.
2. **`maintainability-analysis`** (if significant new code): Coupling, cohesion, complexity scoring, duplication detection.
3. **`reliability-validation`** (if error handling or state transitions involved): Error recovery, state consistency, fault tolerance.

Fix all BLOCKER and MAJOR issues before proceeding.

---

## Step 7: Request Review

With all quality checks passing:

1. Run the `pre-review` hook for final automated validation.
2. Run the `pre-merge` hook for quality gate enforcement.
3. Submit for review by the reviewer agent.
4. Address any REQUEST_CHANGES feedback.

---

## Quality Failure Recovery

If quality gates fail at any step:

1. Identify which conditions failed.
2. Follow the `refactor-for-quality` playbook for guidance on:
   - Reducing complexity (extract functions, simplify conditions, use guard clauses).
   - Improving coverage (add missing test cases).
   - Removing duplication (extract shared logic).
   - Fixing reliability issues (add error handling, fix state transitions).
3. Re-run quality checks after fixes.

---

## References

| Artifact | Path | Usage |
|----------|------|-------|
| Quality profiles | `docs/quality/quality-profiles.md` | Stack-specific standards |
| Complexity guidelines | `docs/quality/complexity-guidelines.md` | Thresholds |
| Clean code guidelines | `docs/quality/clean-code-guidelines.md` | Categories |
| Quality gates | `docs/quality/quality-gates.md` | Pass/fail criteria |
| Backend endpoint playbook | `.project-ai/playbooks/implement-backend-endpoint.md` | Implementation steps |
| Frontend page playbook | `.project-ai/playbooks/implement-frontend-page.md` | Implementation steps |
| Refactoring playbook | `.project-ai/playbooks/refactor-for-quality.md` | Failure recovery |
