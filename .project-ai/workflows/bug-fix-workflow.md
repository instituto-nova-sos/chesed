# Bug Fix Workflow

Use this workflow for non-emergency bug fixes discovered during development or testing. This is a lightweight alternative to the full feature-delivery workflow, focused on rapid reproduction, root cause analysis, and verified fixes with regression tests.

---

## When to Use This Workflow

- Bug discovered during sprint development or testing.
- Bug reported by a user or stakeholder.
- Test failure that indicates a real code defect (not a flaky test).
- Regression from a previous change.

**Do NOT use this workflow for:**
- Production emergencies — use the `hotfix-workflow` instead.
- Feature requests disguised as bugs — use the `feature-delivery` workflow.
- Infrastructure issues — coordinate with devops-engineer agent.

---

## Flow Overview

```
BUG REPORTED ──> REPRODUCE ──> ROOT CAUSE ──> DESIGN FIX ──> IMPLEMENT + REGRESSION TEST ──> REVIEW ──> MERGE
     │               │             │              │                    │                        │         │
     │               │             │              │                    │                        │         └─ pre-merge hook
     │               │             │              │                    │                        └─ review-code skill
     │               │             │              │                    └─ post-implement hook
     │               │             │              └─ Minimal fix approach
     │               │             └─ Identify exact cause and affected code
     │               └─ Create failing test that demonstrates the bug
     └─ Document symptoms and conditions
```

---

## Step 1: Document the Bug

Record the bug with sufficient detail:

```markdown
### Bug: [Short description]

**Symptoms**: What is happening (observed behavior)
**Expected**: What should happen (expected behavior)
**Steps to reproduce**: Numbered steps to trigger the bug
**Environment**: Which context (backend/frontend, specific endpoint, browser)
**Severity**: BLOCKER / MAJOR / MINOR
**Affected story**: [Story ID from backlog, if applicable]
```

---

## Step 2: Reproduce

1. Follow the reproduction steps to confirm the bug.
2. **Write a failing test** that demonstrates the bug:
   - The test must FAIL on the current code (proving the bug exists).
   - The test must describe the expected correct behavior.
   - This test becomes the regression test — it must PASS after the fix.
3. If the bug cannot be reproduced, investigate further before proceeding:
   - Check logs for error patterns.
   - Check for environment-specific conditions (data state, race conditions).
   - If still not reproducible, document findings and close as "cannot reproduce".

**Output**: A failing test case.

---

## Step 3: Root Cause Analysis

1. Trace the code path from the reproduction scenario.
2. Identify the exact point of failure:
   - Wrong conditional logic?
   - Missing validation?
   - Incorrect SQL query?
   - State machine transition error?
   - Missing campus_id filter?
   - Race condition?
3. Document the root cause clearly:
   - Which file and function contains the bug.
   - Why the current code produces the wrong behavior.
   - What the correct behavior should be.
4. Check if the same pattern exists elsewhere (same bug in other places).

**Output**: Root cause statement with file:line reference.

---

## Step 4: Design Fix

1. Design the minimal fix that corrects the root cause.
2. **Minimal means**: Change only what is necessary to fix the bug. No refactoring, no feature additions, no "while I'm here" improvements.
3. Consider side effects:
   - Does this fix change any API behavior?
   - Does this fix affect other features?
   - Does this fix require a migration?
4. If the fix is complex (touches multiple files, changes behavior significantly), consider using the `feature-delivery` workflow instead.

**Output**: Fix approach (1-3 sentences describing the change).

---

## Step 5: Implement Fix + Regression Test

1. Apply the fix to the identified code.
2. Run the failing test from Step 2 — it must now PASS.
3. Run the full test suite — no existing tests should break.
4. If the same bug pattern exists elsewhere (from Step 3), fix all instances.
5. Run the `post-implement` hook to verify quality.

**Implementation rules:**
- The regression test from Step 2 is non-negotiable — it must exist.
- The fix commit must reference the bug description.
- No unrelated changes in the fix commit.

---

## Step 6: Review

1. Run `pre-review` hook (tests, lint, quality gate).
2. Run `review-code` skill focused on:
   - Does the fix address the root cause (not just symptoms)?
   - Is the regression test sufficient (covers the exact bug scenario)?
   - Are there any side effects on other functionality?
   - Is the fix minimal and focused?
3. Standard quality gates apply — the fix must meet the same quality bar as feature code.

---

## Step 7: Merge

1. Run `pre-merge` hook (quality gate enforcement).
2. Merge the fix.
3. Commit message format:
   ```
   fix: [short description of what was fixed]

   Root cause: [brief root cause explanation]
   Regression test: [test function name]
   ```

---

## Agent Assignments

| Step | Agent |
|------|-------|
| 1. Document | tech-lead (triage) |
| 2. Reproduce | backend-engineer or frontend-engineer (depending on bug location) |
| 3. Root cause | backend-engineer or frontend-engineer |
| 4. Design fix | backend-engineer or frontend-engineer + tech-lead (if complex) |
| 5. Implement | backend-engineer or frontend-engineer |
| 6. Review | reviewer + tech-lead |
| 7. Merge | tech-lead |

---

## Key Differences from Feature Delivery

| Aspect | Feature Delivery | Bug Fix |
|--------|-----------------|---------|
| Starting point | Backlog story | Bug report |
| Design phase | Full feature spec | Minimal fix approach |
| First code written | Feature code | Failing regression test |
| Scope | May be large | Must be minimal |
| Documentation | Full docs update | Only if fix changes documented behavior |
| Quality gates | Full gates | Same gates (no exceptions) |

---

## Quick Reference

```
Skills:  review-code
Hooks:   post-implement, pre-review, pre-merge
Agents:  tech-lead (triage), backend/frontend-engineer (fix), reviewer (review)
Rules:   quality-gates, test-coverage-enforcement
Commit:  fix: <description>
```
