# Quality Gates

Quality gates define the pass/fail conditions that code must satisfy before it is accepted. They provide objective, measurable criteria that eliminate subjective quality judgments.

This project enforces two quality gates:
1. **New Code Quality Gate** — applied to every PR (code changed or added in the PR)
2. **Overall Code Quality Gate** — applied at sprint release (entire codebase)

---

## New Code Quality Gate

Applied to every pull request. All conditions must pass. Any failure blocks the merge.

| Condition | Threshold | Verdict |
|-----------|-----------|---------|
| New bugs (reliability issues) | > 0 | FAIL |
| New vulnerabilities (security issues) | > 0 | FAIL |
| Security hotspots reviewed | < 100% | FAIL |
| Test coverage on new code | < 80% | FAIL |
| Duplicated lines on new code | > 3.0% | FAIL |
| Maintainability rating | worse than A | FAIL |
| Reliability rating | worse than A | FAIL |
| Security rating | worse than A | FAIL |
| High severity issues | > 0 | FAIL |
| Cognitive complexity exceeded | any function above threshold | FAIL |

### Rating Definitions

| Rating | Criteria |
|--------|---------|
| A | 0 issues of the relevant type (bugs for reliability, vulnerabilities for security, code smells for maintainability) |
| B | At least 1 minor issue |
| C | At least 1 major issue |
| D | At least 1 critical issue |
| E | At least 1 blocker issue |

### Issue Severity Definitions

| Severity | Definition | Examples |
|----------|-----------|---------|
| Blocker | Prevents the system from functioning or creates a critical security hole | Authentication bypass, data corruption, crash on startup |
| Critical (High) | Major functionality broken or significant security risk | Missing RBAC on endpoint, unhandled error causing data loss, PII in logs |
| Major | Significant quality problem that should be fixed before release | Function exceeding 2x complexity threshold, missing campus_id filter, duplicated business logic |
| Minor | Quality problem that should be fixed but does not block release | Inconsistent naming, missing test for edge case, minor duplication |
| Info | Suggestion for improvement, no functional impact | Better variable name, alternative pattern, style preference |

---

## Overall Code Quality Gate

Applied at sprint release. Evaluates the entire codebase. All conditions must pass for release.

| Condition | Threshold | Verdict |
|-----------|-----------|---------|
| Blocker severity issues | > 0 | FAIL |
| High severity issues | > 0 | FAIL |
| Test coverage (overall) | < 70% | FAIL |
| Duplicated lines (overall) | > 5.0% | FAIL |
| Maintainability rating | worse than A | FAIL |
| Reliability issues | > 0 | FAIL |
| Security hotspots reviewed | < 100% | FAIL |
| Security issues | > 0 | FAIL |
| Security rating | worse than A | FAIL |

### Coverage Roadmap

Overall coverage thresholds tighten over time:

| Sprint | Minimum Coverage |
|--------|-----------------|
| Sprint 1 | 70% |
| Sprint 2 | 70% |
| Sprint 3 | 80% |
| Sprint 4 | 80% |

New code coverage threshold (80%) remains constant across all sprints.

---

## How Quality Gates Are Enforced

### During Development

| Enforcement Point | Mechanism |
|-------------------|-----------|
| Before coding | `pre-implement` hook assesses expected complexity |
| During implementation | `implement-with-quality` playbook guides quality-aware development |
| Self-review | `pre-review` hook validates quality gate conditions |
| Code review | `review-code` skill evaluates quality profiles and clean code categories |
| Before merge | `pre-merge` hook enforces quality gate pass/fail |
| Sprint release | `pre-release` hook enforces overall code quality gate |

### Agent Responsibilities

| Agent | Quality Gate Role |
|-------|-----------------|
| Reviewer | Primary enforcer. Runs quality gate checks. Issues APPROVE or REQUEST_CHANGES. |
| Tech Lead | Architecture quality and release gating. Blocks release on overall gate failure. |
| Backend Engineer | Self-checks Go code against backend quality profile before requesting review. |
| Frontend Engineer | Self-checks React/TS code against frontend quality profile before requesting review. |
| Security Engineer | Validates security dimension. Reviews all security hotspots. |

---

## Quality Gate Evaluation Process

### Step 1: Identify Changed Code

Determine which files were added or modified in the PR. This is the "new code" scope.

### Step 2: Evaluate New Code Conditions

For each condition in the New Code Quality Gate:

1. **Bugs**: Check for reliability issues — unhandled errors, potential null dereferences, race conditions, incorrect state transitions.
2. **Vulnerabilities**: Check for security issues — injection risks, missing auth, exposed PII, insecure patterns.
3. **Security hotspots**: Identify security-sensitive code (auth, PII, crypto, external input) and verify each has been reviewed.
4. **Coverage**: Verify new business logic has unit tests. Service layer coverage ≥ 80%. Handler coverage ≥ 60%.
5. **Integration tests**: Verify every new HTTP endpoint has a backend integration test (`backend/internal/integration/`, real Postgres via testcontainers-go) and every new API client function has a frontend integration test (`frontend/src/__integration__/`, MSW-backed). Per `.project-ai/checklists/integration-tests.md`. Unit-only coverage does NOT satisfy this condition.
6. **Duplication**: Check for copy-pasted logic. If similar code exists elsewhere, it should be extracted.
7. **Maintainability**: Check cognitive complexity, function length, nesting depth against thresholds. See [`complexity-guidelines.md`](complexity-guidelines.md).
8. **Severity classification**: Classify each finding as Blocker, Critical, Major, Minor, or Info.

### Step 3: Render Verdict

```
IF any condition FAILs:
  Verdict: FAIL — REQUEST_CHANGES
  List all failing conditions with specific findings
ELSE:
  Verdict: PASS — APPROVE
```

### Step 4: Report

Quality gate report format:

```markdown
## Quality Gate: [PASS / FAIL]

### New Code Conditions
| Condition | Status | Details |
|-----------|--------|---------|
| Bugs | PASS/FAIL | [count and locations] |
| Vulnerabilities | PASS/FAIL | [count and locations] |
| Security Hotspots | PASS/FAIL | [reviewed/total] |
| Coverage | PASS/FAIL | [percentage] |
| Integration tests | PASS/FAIL | [new endpoints / new client functions covered] |
| Duplication | PASS/FAIL | [percentage] |
| Maintainability | PASS/FAIL | [rating] |
| Reliability | PASS/FAIL | [rating] |
| Security | PASS/FAIL | [rating] |

### Findings
[List of issues by severity]

### Verdict
[APPROVE / REQUEST_CHANGES with required fixes]
```

---

## Handling Quality Gate Failures

When a quality gate fails:

1. **Do not bypass the gate.** Quality gates are non-negotiable.
2. **Fix the findings.** Use the `refactor-for-quality` playbook for guidance.
3. **Re-evaluate.** Run the quality gate check again after fixes.
4. **Document exceptions.** If a threshold cannot be met due to genuine technical constraints, document the reason in an ADR (Architecture Decision Record) using `.project-ai/templates/adr.md`. This requires Tech Lead approval.

### Escalation Path

```
Developer fixes → Re-check passes → Merge
Developer cannot fix → Tech Lead reviews → ADR exception (rare) → Merge with documented debt
```

Exceptions must be:
- Documented in an ADR
- Time-bounded (fix-by date)
- Tracked as technical debt
- Resolved before the sprint in which they were scheduled

---

## References

| Document | Path |
|----------|------|
| Quality profiles | [`docs/quality/quality-profiles.md`](quality-profiles.md) |
| Complexity guidelines | [`docs/quality/complexity-guidelines.md`](complexity-guidelines.md) |
| Clean code guidelines | [`docs/quality/clean-code-guidelines.md`](clean-code-guidelines.md) |
| Implementation guidelines | `docs/15-implementation-guidelines.md` |
