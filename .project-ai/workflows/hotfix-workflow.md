# Hotfix Workflow

Use this workflow for emergency production fixes that cannot wait for normal sprint delivery. This is a fast-track process with abbreviated review requirements.

---

## When to Use This Workflow

- Production deployment has failed and rollback is not possible.
- Critical bug in production affecting users.
- Security vulnerability requiring immediate patch.
- Data integrity issue requiring immediate code fix.

**Do NOT use this workflow for:**
- Non-critical bugs — use `bug-fix-workflow`.
- New features — use `feature-delivery` workflow.
- Issues that can wait for the next sprint.

---

## Flow Overview

```
INCIDENT ──> ASSESS ──> HOTFIX BRANCH ──> MINIMAL FIX ──> ABBREVIATED REVIEW ──>
    ──> STAGING DEPLOY ──> VERIFY ──> PRODUCTION DEPLOY ──> POST-DEPLOY ──>
    ──> INCIDENT REPORT ──> MERGE TO MAIN ──> BACKLOG STORY
```

---

## Step 1: Incident Detection

Document immediately:
- What is failing (symptoms).
- Who is affected (scope).
- When it started (timeline).
- Severity classification (CRITICAL/HIGH).

---

## Step 2: Assess

1. Confirm rollback is not viable (per `rollback-and-hotfix` playbook Step 2).
2. Identify the root cause.
3. Estimate the fix scope (must be minimal — 1-3 files).
4. If fix scope is large (> 5 files), consider partial rollback + staged fix.

---

## Step 3: Create Hotfix Branch

```bash
git checkout -b hotfix/YYYY-MM-DD-description <last-release-tag>
```

The branch MUST be based on the last release tag, not on main HEAD (which may have unreleased changes).

---

## Step 4: Implement Minimal Fix

**Rules:**
- Change ONLY what is necessary to fix the issue.
- No refactoring.
- No feature additions.
- No dependency updates (unless the dependency IS the issue).
- Must include a regression test.
- Must not break existing tests.

---

## Step 5: Abbreviated Review

This is NOT a full quality gate review. The review focuses on:

| Check | Required | Reviewer |
|-------|----------|----------|
| Fix addresses root cause | Yes | tech-lead |
| No security implications | Yes | security-engineer |
| Regression test exists | Yes | reviewer |
| Existing tests pass | Yes | CI/CD |
| No unrelated changes | Yes | reviewer |

**Skipped for hotfix:**
- Full quality gate evaluation.
- Coverage threshold enforcement.
- Maintainability analysis.
- Full documentation update.

These are deferred to the follow-up backlog story.

---

## Step 6: Deploy to Staging

1. Build Docker image from hotfix branch.
2. Deploy to staging environment.
3. Run `post-deploy` hook on staging.
4. Verify the specific issue is resolved in staging.
5. If staging verification fails, return to Step 4.

---

## Step 7: Deploy to Production

1. Deploy the verified staging image to production.
2. Run `post-deploy` hook on production.
3. Monitor for 30 minutes.
4. If production verification fails, initiate rollback.

---

## Step 8: Post-Incident

1. **Write incident report** using `incident-report` template.
2. **Merge hotfix to main**:
   ```bash
   git checkout main
   git merge hotfix/YYYY-MM-DD-description
   git tag hotfix-YYYY-MM-DD-description
   ```
3. **Create backlog story** for proper fix (if hotfix was a band-aid).
4. **Add prevention measures** (monitoring, tests, process changes).
5. **Update `.project-ai/`** if a process gap was identified.

---

## Agent Assignments

| Step | Agent |
|------|-------|
| 1. Detection | Any (escalate to tech-lead) |
| 2. Assess | tech-lead |
| 3. Branch | devops-engineer |
| 4. Fix | backend-engineer or frontend-engineer |
| 5. Review | security-engineer + reviewer (abbreviated) |
| 6. Staging | devops-engineer |
| 7. Production | devops-engineer |
| 8. Post-incident | tech-lead |

---

## Key Differences from Normal Delivery

| Aspect | Normal Delivery | Hotfix |
|--------|----------------|--------|
| Branch base | main HEAD | Last release tag |
| Scope | Full feature | Minimal fix only |
| Review | Full quality gate | Abbreviated (security + basic) |
| Documentation | Documentation-first | Code-first, docs after |
| Testing | Full test plan | Regression test only |
| Deployment | End of sprint | Immediate |
| Follow-up | None needed | Backlog story required |

---

## Quick Reference

```
Playbook:  rollback-and-hotfix (decision framework)
Template:  incident-report (post-incident documentation)
Hooks:     post-deploy (verification after deployment)
Agents:    tech-lead (orchestration), security-engineer (review), devops-engineer (deployment)
Commit:    fix: [hotfix] <description>
Tag:       hotfix-YYYY-MM-DD-description
```
