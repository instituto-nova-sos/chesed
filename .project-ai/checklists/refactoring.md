# Refactoring Checklist

Use this checklist after any refactoring to ensure safety and correctness. Every item must pass.

---

## Behavior Preservation

- [ ] All existing tests pass **before** refactoring (baseline established)
- [ ] All existing tests pass **after** refactoring (behavior preserved)
- [ ] No test was modified to make it pass (tests validate behavior, not implementation)
- [ ] Manual verification of affected user flows (if applicable)

## Structural Improvement

- [ ] Complexity reduced or stable — no function exceeds thresholds after refactoring
- [ ] Duplication reduced — extracted shared logic to appropriate layer
- [ ] Nesting depth reduced or stable — guard clauses used where applicable
- [ ] File length within limits — large files split by responsibility
- [ ] Naming improved — names are more descriptive and intention-revealing

## Dependencies

- [ ] No new external dependencies introduced (unless justified)
- [ ] No new circular dependencies created
- [ ] Dependency direction preserved (handler → service → repository → domain)
- [ ] No layer violations introduced

## Clean Code

- [ ] **Consistency**: Refactored code follows the same patterns as surrounding code
- [ ] **Intentionality**: Extracted functions have clear, descriptive names
- [ ] **Adaptability**: Refactoring makes future changes easier, not harder
- [ ] **Responsibility**: Each extracted function has a single responsibility

## Coverage

- [ ] Test coverage maintained or improved after refactoring
- [ ] New helper functions have tests if they contain logic (not trivial delegation)
- [ ] No untested code paths introduced

## Documentation

- [ ] Commit message clearly describes the refactoring: `refactor: <description>`
- [ ] No feature changes mixed into the refactoring commit
- [ ] Affected documentation updated if public interfaces changed

---

## Safety Rules

1. **One commit per refactoring step** — makes reverting safe.
2. **No behavior changes during refactoring** — separate refactoring from feature work.
3. **Green tests at every step** — if tests break, revert and try smaller steps.
4. **No feature flags or compatibility shims** — just change the code.

---

## How to Use

Run this checklist after completing any refactoring, whether triggered by quality gate failure or proactive improvement.

```
Playbook: refactor-for-quality (guides the refactoring process)
Skill:    maintainability-analysis (identifies what to refactor)
Skill:    review-code (validates quality after refactoring)
```
