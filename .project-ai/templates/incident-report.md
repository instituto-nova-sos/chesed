# Template: Incident Report

Use this template to document post-incident findings after a production issue, failed deployment, or data integrity problem.

---

## Incident Report: [Short Title]

**Date**: YYYY-MM-DD
**Severity**: CRITICAL / HIGH / MEDIUM / LOW
**Duration**: [Time from detection to resolution]
**Affected users/features**: [What was impacted]
**Reporter**: [Who detected the issue]

---

### Timeline

| Time | Event |
|------|-------|
| HH:MM | Issue detected: [how it was discovered] |
| HH:MM | Triage started: [who responded] |
| HH:MM | Root cause identified: [brief description] |
| HH:MM | Fix applied: [what was done] |
| HH:MM | Fix verified: [how verification was done] |
| HH:MM | Incident resolved |

---

### Root Cause Analysis

**What happened**: [Factual description of the failure]

**Why it happened**: [Root cause — not symptoms]

**Contributing factors**:
- [Factor 1: e.g., missing test coverage for edge case]
- [Factor 2: e.g., migration tested only with empty database]
- [Factor 3: e.g., no monitoring alert for this failure mode]

---

### Impact Assessment

**Data affected**: [Records corrupted, lost, or exposed — with counts]
**Users impacted**: [Number and type of affected users]
**SLA violation**: [Yes/No — which SLA if yes]
**PII exposure**: [Yes/No — LGPD implications if yes]
**Financial impact**: [If applicable]

---

### Resolution

**Immediate fix**: [What was done to stop the bleeding]
**Permanent fix**: [What code/config change resolves the root cause]
**Rollback performed**: [Yes/No — details if yes]
**Data remediation**: [Steps taken to fix affected data]

---

### Prevention

**Code changes**:
- [ ] [Specific code fix — with PR reference]
- [ ] [Regression test added — with test function name]

**Process changes**:
- [ ] [New rule, hook, or checklist item to prevent recurrence]
- [ ] [Monitoring/alerting addition]

**Documentation updates**:
- [ ] [Docs updated to reflect lessons learned]

---

### Follow-Up Items

| Item | Owner | Deadline | Status |
|------|-------|----------|--------|
| [Backlog story for proper fix] | | | |
| [Monitoring addition] | | | |
| [Process improvement] | | | |

---

### Lessons Learned

**What went well**: [Effective parts of the response]
**What could improve**: [Where the response was slow or ineffective]
**Process gaps identified**: [Missing hooks, rules, or checks that would have prevented this]
