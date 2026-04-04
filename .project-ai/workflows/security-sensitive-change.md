# Security-Sensitive Change Workflow

Use this workflow for any change that affects authentication, authorization, data access controls, secrets management, audit logging, or data protection. This workflow wraps the standard feature delivery with additional security gates.

---

## When to Use This Workflow

This workflow is triggered by the `security-review-triggers` rule. Apply it when a change touches any of:

- Authentication flow (Keycloak OIDC token handling)
- Authorization logic (RBAC middleware, role checks)
- Data access patterns (campus scoping, query filters)
- Secrets or credentials (environment variables, Keycloak config)
- Audit logging (audit_log table, mutation tracking)
- PII handling (person, address, assisted_profile data)
- Offline data encryption (IndexedDB, Web Crypto API)
- CORS configuration
- Input validation or sanitization
- SQL query construction
- New external API integrations
- Keycloak realm or client configuration changes

---

## Flow Overview

```
TRIGGER ──> EARLY REVIEW ──> IMPLEMENT ──> SECURITY REVIEW ──> REMEDIATE ──> RE-VERIFY ──> UPDATE DOCS
    │            │                │              │                  │             │              │
    │            │                │              │                  │             │              └─ Security docs
    │            │                │              │                  │             └─ Re-run review
    │            │                │              │                  └─ Fix findings
    │            │                │              └─ conduct-security-review playbook
    │            │                └─ feature-delivery.md (IMPLEMENT phase)
    │            └─ security-engineer agent
    └─ security-review-triggers rule
```

---

## Step 1: Trigger Identification

When a change is identified as security-sensitive (by the `security-review-triggers` rule or manual assessment):

1. **Tag the change** as security-sensitive in the feature spec or task description
2. **Identify the threat category** from `docs/18-threat-model.md`:
   - Authentication bypass
   - Authorization escalation
   - Data leakage (PII exposure)
   - Injection (SQL, XSS)
   - Audit trail tampering
   - Offline data exposure
   - Cross-campus data access
3. **Assess initial risk level**:
   - **HIGH**: Directly handles auth, secrets, or PII access controls
   - **MEDIUM**: Modifies data queries, validation, or audit logging
   - **LOW**: Tangentially affects security (e.g., new endpoint with standard RBAC)

**Output**: Tagged change with threat category and risk level.

---

## Step 2: Early Design Review

**Goal**: Catch security design flaws before implementation (shift-left).

### Process

1. **Engage the security-engineer agent** with:
   - The feature spec or change description
   - The identified threat category
   - The proposed technical approach

2. **Security-engineer reviews against**:
   - `docs/13-security-and-compliance.md` — compliance requirements
   - `docs/16-iam-and-access-control.md` — IAM design
   - `docs/18-threat-model.md` — known threats and mitigations
   - `docs/19-secure-development-standard.md` — secure coding rules
   - `docs/20-keycloak-configuration.md` — Keycloak setup

3. **Review focus areas**:

   | Area | Key Questions |
   |------|--------------|
   | **Auth** | Is Keycloak the only auth mechanism? No custom token issuance? |
   | **RBAC** | Are role requirements correct? Is the principle of least privilege applied? |
   | **Data access** | Is campus_id filtering applied? Can cross-campus access occur? |
   | **Input** | Are all inputs validated? Are SQL queries parameterized? |
   | **Secrets** | Are any secrets introduced? How are they managed? |
   | **PII** | Does this change how PII is stored, transmitted, or logged? |
   | **Offline** | Is sensitive data encrypted in IndexedDB? Cleared on logout? |

4. **Early review outcomes**:

   | Outcome | Action |
   |---------|--------|
   | **APPROVED** | Proceed to implementation |
   | **DESIGN CHANGES NEEDED** | Revise design, re-submit for early review |
   | **BLOCKED** | Fundamental security concern — escalate, do not proceed |

**Output**: Early security review approval (or required design changes).

---

## Step 3: Implementation

Follow the standard `feature-delivery.md` workflow (IMPLEMENT phase) with these additional constraints:

### Security Implementation Rules

1. **Authentication**: Use only `coreos/go-oidc` for token validation. Never create custom login endpoints.

2. **Authorization**: Apply RBAC middleware at the route level in `cmd/server/main.go`. Never check roles inside handler logic.

3. **Data access**: Every SQL query that touches operational data must include `WHERE campus_id = $N` with the value from the authenticated user's token claims.

4. **Input validation**: Every handler must validate input using `go-playground/validator` before calling the service layer.

5. **SQL safety**: Use `pgx` parameterized queries exclusively. Never concatenate user input into SQL strings.

6. **Logging**: Use `slog` structured logging. Never log PII fields (name, CPF, phone, email, address).

7. **Error responses**: Return generic error messages to clients. Never expose internal details, stack traces, or PII.

8. **Secrets**: Inject via environment variables. Never commit to source code, Docker images, or compose files.

9. **Offline encryption**: Use Web Crypto API with AES-256-GCM for sensitive IndexedDB data. Derive keys from user session.

10. **Keycloak changes**: Export realm configuration to `keycloak/realm-export.json` and commit.

---

## Step 4: Security Review

**Goal**: Thorough security analysis of the implemented change.

### Process

1. **Run the `conduct-security-review` playbook**, which:
   - Analyzes the code diff for security patterns
   - Checks against `security-review.md` checklist
   - Identifies potential vulnerabilities

2. **Run the `review-security` skill** for automated analysis:
   - Token validation correctness
   - RBAC middleware coverage
   - Campus scoping in queries
   - PII in logs/errors
   - SQL injection vectors
   - Secret exposure

3. **Fill the security-review-report template** with:

   ```markdown
   ## Security Review Report

   ### Change Summary
   [What was changed and why]

   ### Threat Category
   [From docs/18-threat-model.md]

   ### Findings

   | # | Severity | Finding | Location | Recommendation |
   |---|----------|---------|----------|----------------|
   | 1 | HIGH/MEDIUM/LOW | ... | file:line | ... |

   ### Checklist Results
   [Reference security-review.md — all items PASS/FAIL]

   ### Verdict
   APPROVED / APPROVED WITH CONDITIONS / REMEDIATION REQUIRED
   ```

### Severity Definitions

| Severity | Definition | SLA |
|----------|-----------|-----|
| **CRITICAL** | Exploitable vulnerability, data breach risk | Block merge. Fix immediately. |
| **HIGH** | Security control missing or bypassed | Block merge. Fix before merge. |
| **MEDIUM** | Defense-in-depth gap | Fix within current sprint. |
| **LOW** | Best practice deviation | Track, fix in next sprint. |

---

## Step 5: Remediate Findings

For each finding with severity MEDIUM or above:

1. **Fix the issue** in code
2. **Add a test** that would catch regression
3. **Document the fix** in the security review report
4. **Update the threat model** (`docs/18-threat-model.md`) if a new threat was identified

For LOW severity findings:
- Create a tracked item for the next sprint
- Document in HANDOFF.md

---

## Step 6: Re-Verify

After remediation:

1. **Re-run the `review-security` skill** on the updated code
2. **Re-check failed items** from the `security-review.md` checklist
3. **Confirm all CRITICAL and HIGH findings are resolved**
4. **Security-engineer agent gives final approval**

### Final Approval Gate

```
All CRITICAL findings: RESOLVED     [required]
All HIGH findings:     RESOLVED     [required]
All MEDIUM findings:   RESOLVED or TRACKED  [required]
All LOW findings:      RESOLVED or TRACKED  [acceptable]
```

Only proceed when the security-engineer agent confirms: **APPROVED**.

---

## Step 7: Update Security Documentation

Update the relevant security documents based on what was changed:

| What Changed | Documents to Update |
|-------------|-------------------|
| Auth flow | `docs/16-iam-and-access-control.md`, `docs/20-keycloak-configuration.md` |
| RBAC roles/permissions | `docs/16-iam-and-access-control.md`, `docs/11-api-design.md` |
| New threat identified | `docs/18-threat-model.md` |
| New security control | `docs/13-security-and-compliance.md` |
| New security test | `docs/17-security-test-strategy.md` |
| Secure coding pattern | `docs/19-secure-development-standard.md` |
| Keycloak config | `docs/20-keycloak-configuration.md`, `keycloak/realm-export.json` |

Commit with:
```
docs: update security docs for <change description>
```

---

## Agent Assignments

| Step | Primary Agent | Supporting Agent |
|------|--------------|-----------------|
| 1. Trigger | tech-lead (or any agent that identifies the trigger) | — |
| 2. Early review | security-engineer | tech-lead |
| 3. Implement | backend-engineer / frontend-engineer | security-engineer (advisory) |
| 4. Security review | security-engineer | tech-lead |
| 5. Remediate | backend-engineer / frontend-engineer | security-engineer (verify fix) |
| 6. Re-verify | security-engineer | — |
| 7. Update docs | tech-lead | security-engineer (review) |

---

## Quick Reference

```
Skills:    review-security, review-code, maintain-docs
Playbooks: conduct-security-review
Hooks:     pre-implement (flags security-sensitive changes)
           pre-review (includes security checks)
Checklists: security-review.md
Templates:  security-review-report
Agents:     security-engineer (primary), tech-lead (oversight)
            backend-engineer / frontend-engineer (implementation)

Key docs:
  - docs/13-security-and-compliance.md
  - docs/16-iam-and-access-control.md
  - docs/17-security-test-strategy.md
  - docs/18-threat-model.md
  - docs/19-secure-development-standard.md
  - docs/20-keycloak-configuration.md
```
