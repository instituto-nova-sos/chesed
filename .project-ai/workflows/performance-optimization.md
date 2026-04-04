# Performance Optimization Workflow

Use this workflow when performance issues are identified (via performance-analysis skill, user reports, or monitoring) and need systematic resolution.

---

## When to Use This Workflow

- Performance-analysis skill identifies endpoints or metrics over budget.
- User reports slow behavior in the application.
- Monitoring shows response time degradation.
- Bundle size exceeds budget after new feature additions.
- Offline sync becomes too slow for acceptable UX.

---

## Flow Overview

```
IDENTIFY ──> PROFILE ──> DESIGN FIX ──> IMPLEMENT ──> VERIFY ──> DOCUMENT
    │            │            │              │            │           │
    │            │            │              │            │           └─ performance-report template
    │            │            │              │            └─ Re-measure against budget
    │            │            │              └─ Targeted, minimal fix
    │            │            └─ ADR if architectural change needed
    │            └─ Measure specific bottleneck
    └─ performance-analysis skill
```

---

## Step 1: Identify

1. Run the `performance-analysis` skill to systematically evaluate the system.
2. Prioritize findings by severity:
   - **BLOCKER**: User-facing endpoint > 2x budget.
   - **MAJOR**: User-facing endpoint > budget.
   - **MINOR**: Internal operation > budget.
3. Select the highest-priority finding to address first.

---

## Step 2: Profile

For the selected finding, gather precise measurements:

**Backend (API/Database):**
- Run `EXPLAIN ANALYZE` on the identified query.
- Profile the handler-to-response path to identify which layer is slow.
- Measure with representative data volume (not empty database).
- Check connection pool utilization.

**Frontend:**
- Analyze Vite build output for chunk sizes.
- Profile React component renders with React DevTools.
- Measure Web Vitals with Lighthouse.
- Check for unnecessary re-renders or missing memoization.

**Offline Sync:**
- Measure sync queue processing time with realistic data.
- Profile IndexedDB operations.
- Check for blocking operations on the main thread.

**Output**: Precise measurement of the bottleneck with root cause.

---

## Step 3: Design Fix

1. Design a targeted fix for the identified bottleneck.
2. If the fix is localized (add index, memoize component, optimize query):
   - Proceed directly to implementation.
3. If the fix is architectural (add caching layer, change data access pattern, restructure components):
   - Follow the `architecture-change` workflow.
   - Create an ADR documenting the decision.
4. Estimate the expected improvement.

---

## Step 4: Implement

1. Apply the fix following standard implementation playbooks.
2. Keep changes minimal and focused on the performance issue.
3. Do not combine performance fixes with feature work.
4. Ensure tests still pass (performance fix must not break functionality).

---

## Step 5: Verify

1. Re-measure the exact metric that was over budget.
2. Confirm the metric is now within budget.
3. Run the full test suite to ensure no regressions.
4. If the metric is still over budget, return to Step 2 with deeper profiling.

---

## Step 6: Document

1. Update the performance-report template with before/after measurements.
2. If an index was added, update `docs/10-data-model.md`.
3. If query patterns changed, document the optimization in code comments.
4. Record the optimization for trend tracking across sprints.

---

## Agent Assignments

| Step | Agent |
|------|-------|
| 1. Identify | tech-lead (using performance-analysis skill) |
| 2. Profile (backend) | backend-engineer |
| 2. Profile (frontend) | frontend-engineer |
| 3. Design fix | tech-lead + relevant engineer |
| 4. Implement | backend-engineer or frontend-engineer |
| 5. Verify | tech-lead (re-run performance-analysis) |
| 6. Document | tech-lead (using maintain-docs skill) |

---

## Quick Reference

```
Skills:  performance-analysis, maintain-docs
Rules:   performance-budget
Template: performance-report
Hooks:   post-implement (after fix), pre-review (quality gate)
Agents:  tech-lead (orchestration), backend/frontend-engineer (implementation)
Workflow: architecture-change (if architectural fix needed)
```
