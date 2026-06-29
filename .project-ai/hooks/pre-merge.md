# Hook: Pre-Merge Quality Gate

## Purpose

Enforce quality gates before code enters the main branch. This is the final automated quality checkpoint that validates all measurable quality conditions are satisfied.

## Trigger Condition

Before merging any PR into the main branch. Before approving code for integration.

## Status

**Blocking** — Do not merge if any quality gate condition fails.

## Steps

1. **Evaluate New Code Quality Gate**

   Check all conditions from `docs/quality/quality-gates.md` — New Code Quality Gate:

   - [ ] **Bugs**: 0 new reliability issues (unhandled errors, null dereferences, race conditions, incorrect state transitions).
   - [ ] **Vulnerabilities**: 0 new security issues (injection risks, missing auth, exposed PII, insecure patterns).
   - [ ] **Security hotspots reviewed**: 100% of security-sensitive code reviewed (auth, PII, crypto, external input).
   - [ ] **Coverage on new code**: ≥ 80% test coverage on new/changed business logic.
   - [ ] **Integration tests on new boundaries**: every new endpoint, every new API client function, and every new SQL constraint covered per `.project-ai/checklists/integration-tests.md`. Unit tests alone do not satisfy this condition.
   - [ ] **Duplication on new code**: ≤ 3% duplicated lines.
   - [ ] **Maintainability rating**: A — no code smells above threshold.
   - [ ] **Reliability rating**: A — 0 bugs.
   - [ ] **Security rating**: A — 0 vulnerabilities.
   - [ ] **High severity issues**: 0.

2. **Validate complexity thresholds**

   Per `docs/quality/complexity-guidelines.md`:

   - [ ] No Go function exceeds cognitive complexity 25.
   - [ ] No TypeScript function exceeds cognitive complexity 15.
   - [ ] No function exceeds cyclomatic complexity 10.
   - [ ] No Go function exceeds 40 lines.
   - [ ] No TypeScript function exceeds 50 lines.
   - [ ] No nesting deeper than 3 levels.

3. **Validate clean code categories**

   Per `docs/quality/clean-code-guidelines.md`:

   - [ ] **Consistency**: Changed files follow established patterns.
   - [ ] **Intentionality**: Names reveal purpose, no dead code.
   - [ ] **Adaptability**: Dependencies point inward, changes confined to appropriate layer.
   - [ ] **Responsibility**: Each function has a single responsibility.

4. **Verify reviewer agent verdict**

   - [ ] Reviewer agent has evaluated the PR.
   - [ ] Reviewer verdict is APPROVE (not REQUEST_CHANGES or NEEDS_DISCUSSION).
   - [ ] All BLOCKER and MAJOR issues resolved.

5. **Verify security review (if applicable)**

   - [ ] If PR touches security-sensitive areas (per `rules/security-review-triggers.md`), security review is complete.
   - [ ] Security engineer has no CRITICAL or HIGH findings.

6. **Render gate verdict**

   ```
   IF all conditions PASS:
     Gate: PASS — merge is allowed
   ELSE:
     Gate: FAIL — merge is blocked
     List all failing conditions
   ```

## Enforcement Mechanism

- The AI agent must execute this hook before any merge approval.
- If any condition fails, the agent must list all failures and required fixes.
- The agent must invoke the `refactor-for-quality` playbook to guide the developer in resolving failures.
- Quality gate bypass is not permitted without an ADR (see `docs/quality/quality-gates.md` — Handling Failures).

## References

- `docs/quality/quality-gates.md` — Quality gate thresholds and evaluation process
- `docs/quality/quality-profiles.md` — Stack-specific quality standards
- `docs/quality/complexity-guidelines.md` — Complexity thresholds
- `docs/quality/clean-code-guidelines.md` — Clean code categories
- `.project-ai/rules/quality-gates.md` — Quality gates enforcement rule
- `.project-ai/checklists/integration-tests.md` — Integration test mandate (backend + frontend) — blocking
- `CLAUDE.md` — Quality Bar + Integration Test Mandate

## Consequences of Skipping

- Low-quality code enters the main branch, accumulating technical debt.
- Complexity grows unchecked, making future changes expensive and risky.
- Security issues reach production, exposing sensitive PII.
- Duplication creates inconsistency when only one copy is updated.
- Reliability issues cause data corruption in offline-first scenarios.
