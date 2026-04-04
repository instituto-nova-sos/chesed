# Rule: Quality Gates Enforcement

## Purpose

Ensure all code meets measurable quality standards before acceptance. This rule unifies complexity control, duplication thresholds, coverage requirements, and quality ratings into a single enforcement mechanism aligned with the project's quality gates.

## Rule Statement

No code is accepted into the codebase if it fails any condition in the applicable quality gate. Quality gates are non-negotiable. Exceptions require an Architecture Decision Record (ADR) with Tech Lead approval.

## Trigger Condition

- Every pull request (New Code Quality Gate)
- Every sprint release (Overall Code Quality Gate)

## Enforcement

### New Code Quality Gate (Every PR)

The following conditions apply to all code changed or added in a PR:

| Condition | Threshold | Action on Failure |
|-----------|-----------|-------------------|
| New bugs | > 0 | Block merge. Fix all reliability issues. |
| New vulnerabilities | > 0 | Block merge. Fix all security issues. |
| Security hotspots reviewed | < 100% | Block merge. Review all security-sensitive code. |
| Test coverage on new code | < 80% | Block merge. Add tests for uncovered business logic. |
| Duplicated lines on new code | > 3% | Block merge. Extract shared logic. |
| Maintainability rating | worse than A | Block merge. Reduce complexity or improve structure. |
| Reliability rating | worse than A | Block merge. Fix error handling and state management. |
| Security rating | worse than A | Block merge. Fix security findings. |
| High severity issues | > 0 | Block merge. Fix all high severity issues. |
| Cognitive complexity exceeded | any function above threshold | Block merge. Refactor function. |

### Overall Code Quality Gate (Sprint Release)

The following conditions apply to the entire codebase at release time:

| Condition | Threshold | Action on Failure |
|-----------|-----------|-------------------|
| Blocker severity issues | > 0 | Block release. |
| High severity issues | > 0 | Block release. |
| Test coverage (overall) | < 70% (Sprint 1-2), < 80% (Sprint 3+) | Block release. |
| Duplicated lines (overall) | > 5% | Block release. |
| Maintainability rating | worse than A | Block release. |
| Reliability issues | > 0 | Block release. |
| Security issues | > 0 | Block release. |
| Security hotspots reviewed | < 100% | Block release. |

### Complexity Thresholds

These are part of the maintainability gate:

| Metric | Go | React/TS |
|--------|-----|----------|
| Cognitive complexity / function | 25 | 15 |
| Cyclomatic complexity / function | 10 | 10 |
| Function length | 40 lines | 50 lines |
| File length | 400 lines | 300 lines |
| Nesting depth | 3 | 3 |
| Parameter count | 5 | 5 |

Full details: `docs/quality/complexity-guidelines.md`

## Enforcement Mechanism

### Hooks

| Hook | Gate Enforced |
|------|--------------|
| `pre-implement` | Assesses expected complexity before coding |
| `pre-review` | Validates quality gate conditions after implementation |
| `pre-merge` | Enforces New Code Quality Gate — blocking |
| `pre-release` | Enforces Overall Code Quality Gate — blocking |

### Skills

| Skill | Gate Aspect |
|-------|-----------|
| `review-code` | All dimensions — primary quality evaluation |
| `maintainability-analysis` | Maintainability rating, complexity, duplication |
| `reliability-validation` | Reliability rating, error handling, state consistency |
| `review-security` | Security rating, vulnerabilities, hotspots |
| `design-test-plan` | Coverage requirements |

### Agents

| Agent | Enforcement Role |
|-------|-----------------|
| Reviewer | Primary enforcer. Runs quality gate checks on every PR. Issues APPROVE or REQUEST_CHANGES. |
| Tech Lead | Release gatekeeper. Blocks release on overall gate failure. |
| Security Engineer | Security dimension enforcer. Reviews all security hotspots. |

## Handling Failures

1. **Fix the findings.** Follow the `refactor-for-quality` playbook.
2. **Re-evaluate.** Run quality gate check again.
3. **Escalate only if necessary.** If a threshold cannot be met due to genuine technical constraints:
   - Document in an ADR
   - Get Tech Lead approval
   - Set a fix-by date
   - Track as technical debt

**Bypassing quality gates without an ADR is a rule violation.**

## Consequences of Skipping

- Uncontrolled complexity accumulates, making future changes expensive and risky
- Low coverage allows regressions to reach production undetected
- Duplication causes inconsistent behavior when only one copy is updated
- Security gaps create vulnerabilities in a system handling sensitive PII (LGPD)
- Reliability issues cause data corruption or loss in offline-first scenarios

## References

- `docs/quality/quality-gates.md` — Full gate definitions and evaluation process
- `docs/quality/quality-profiles.md` — Quality profiles per stack
- `docs/quality/complexity-guidelines.md` — Complexity thresholds and tooling
- `docs/quality/clean-code-guidelines.md` — Clean code categories
