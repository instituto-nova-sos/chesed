# Playbook: Rollback and Hotfix

Emergency procedures for rolling back a failed deployment or applying a critical production fix. This playbook prioritizes speed and safety over completeness.

---

## When to Use

- Deployment verification (post-deploy hook) fails.
- Production monitoring detects critical issues after deployment.
- Users report widespread failures after a release.
- Data integrity issues discovered post-deployment.

---

## Flow Overview

```
INCIDENT ──> ASSESS ──> DECIDE ──> ROLLBACK or HOTFIX ──> VERIFY ──> DOCUMENT ──> FOLLOW-UP
```

---

## Step 1: Assess Severity

Classify the incident:

| Severity | Description | Response Time |
|----------|-------------|---------------|
| **CRITICAL** | Data corruption, security breach, all users affected | Immediate (within minutes) |
| **HIGH** | Major feature broken, significant user impact | Within 1 hour |
| **MEDIUM** | Minor feature broken, workaround available | Within 4 hours |
| **LOW** | Cosmetic issue, no functional impact | Next sprint |

For CRITICAL and HIGH: proceed immediately to Step 2.
For MEDIUM and LOW: follow the `bug-fix-workflow` instead.

---

## Step 2: Decide — Rollback or Hotfix

| Condition | Decision |
|-----------|----------|
| Previous version was stable | **Rollback** |
| Issue is in new code only (not migration) | **Rollback** |
| Database migration is irreversible | **Hotfix** (cannot rollback DB) |
| Issue is in configuration, not code | **Hotfix** (config change) |
| Fix is small and obvious | **Hotfix** (faster than rollback) |
| Root cause unclear | **Rollback** (safer while investigating) |

---

## Step 3A: Rollback Path

1. **Revert to previous Docker image**:
   - Deploy the previous tagged version (e.g., `sprint-N-1-complete`).
   - Do NOT deploy `latest` — use explicit tags.

2. **Rollback database migrations** (if applicable):
   - Run `make migrate-down` to revert the last migration(s).
   - Verify the previous version's code works with the reverted schema.
   - **CAUTION**: Only rollback migrations if data was not yet written to new schema.

3. **Verify Keycloak configuration**:
   - If realm config was changed, revert to previous `realm-export.json`.
   - Restart Keycloak to apply the previous configuration.

4. **Run post-deploy hook**:
   - Execute all post-deployment verification steps.
   - Confirm the previous version is healthy.

---

## Step 3B: Hotfix Path

1. **Create hotfix branch**:
   ```bash
   git checkout -b hotfix/short-description sprint-N-complete
   ```

2. **Apply minimal fix**:
   - Change ONLY what is necessary to fix the issue.
   - No feature additions, no refactoring, no "while I'm here" improvements.
   - Maximum scope: 1-3 files changed.

3. **Fast-track review**:
   - Security-engineer reviews security implications.
   - Reviewer agent runs abbreviated quality check (no full quality gate).
   - Tech-lead approves the hotfix.

4. **Deploy to staging first**:
   - Build Docker image from hotfix branch.
   - Deploy to staging.
   - Run post-deploy hook on staging.

5. **Deploy to production**:
   - Only after staging verification passes.
   - Tag the hotfix: `hotfix-YYYY-MM-DD-description`.
   - Deploy to production.
   - Run post-deploy hook on production.

6. **Merge hotfix back to main**:
   ```bash
   git checkout main
   git merge hotfix/short-description
   ```

---

## Step 4: Verify

1. Run post-deploy hook on the target environment.
2. Verify the specific issue that triggered the incident is resolved.
3. Verify no new issues were introduced.
4. Monitor for 30 minutes for any delayed symptoms.

---

## Step 5: Document

1. Fill the `incident-report` template with:
   - Timeline of events.
   - Root cause analysis.
   - Impact assessment.
   - Resolution details.
   - Prevention measures.

2. Commit the incident report to `docs/incidents/`.

---

## Step 6: Follow-Up

1. **Create backlog story** for proper fix (if hotfix was applied).
2. **Add regression test** that would catch this issue in CI/CD.
3. **Update process** — add hook, rule, or checklist item to prevent recurrence.
4. **Review monitoring** — add alert for this failure mode.
5. **Update `.project-ai/`** artifacts if a process gap was identified.

---

## Safety Rules

- Hotfix must be minimal — no feature additions.
- Hotfix must have at least one reviewer (security-engineer + tech-lead).
- Database rollbacks must be tested in staging first.
- Never rollback a migration if new data was written to the new schema.
- Document everything — even if the fix is "obvious".
- Exception to documentation-first rule: code first, docs immediately after.

---

## Agent Assignments

| Step | Agent |
|------|-------|
| 1. Assess | tech-lead |
| 2. Decide | tech-lead |
| 3A. Rollback | devops-engineer |
| 3B. Hotfix | backend-engineer or frontend-engineer + devops-engineer |
| 4. Verify | devops-engineer |
| 5. Document | tech-lead |
| 6. Follow-up | tech-lead |

---

## Quick Reference

```
Hooks:    post-deploy (verification after rollback/hotfix)
Templates: incident-report
Workflows: hotfix-workflow (detailed hotfix process)
Agents:   tech-lead (decision), devops-engineer (deployment), engineer (fix)
```
