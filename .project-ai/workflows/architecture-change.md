# Architecture Change Workflow

Use this workflow when a proposed change affects the system's architecture — technology choices, structural patterns, integration boundaries, or cross-cutting concerns. The goal is to make deliberate, documented decisions before writing code.

---

## When to Use This Workflow

Triggers:
- New external dependency or integration
- Change to the layered architecture (handler -> service -> repository -> domain)
- New cross-cutting concern (caching, rate limiting, new middleware)
- Change to authentication/authorization flow
- Change to the offline sync strategy
- New database technology or access pattern
- Deviation from `docs/05-architecture-proposal.md`

If in doubt, use this workflow. It is cheaper to write an ADR than to undo an architectural mistake.

---

## Flow Overview

```
QUESTION ──> RESEARCH ──> DRAFT ADR ──> REVIEW ──> ACCEPT ──> UPDATE DOCS ──> IMPLEMENT
    │            │            │            │           │            │              │
    │            │            │            │           │            │              └─ feature-delivery.md
    │            │            │            │           │            └─ maintain-docs skill
    │            │            │            │           └─ Record decision
    │            │            │            └─ tech-lead agent
    │            │            └─ ADR template
    │            └─ docs/05-architecture-proposal.md
    └─ Identify the architectural question
```

---

## Step 1: Identify the Question

Clearly articulate the architectural question in one sentence.

**Format**: "Should we [proposed change] to [achieve goal], or [alternative approach]?"

**Examples**:
- "Should we add Redis for session caching, or continue with Keycloak-managed sessions?"
- "Should we use server-sent events for real-time updates, or polling?"
- "Should we split the person service into separate services for assisted vs. staff?"

**Output**: A single, clear question statement.

---

## Step 2: Research

Gather context from existing documentation and codebase.

### Required Reading

1. `docs/05-architecture-proposal.md` — current architecture decisions and rationale
2. `docs/07-mvp-scope.md` — what is in scope for the current phase
3. `docs/15-implementation-guidelines.md` — existing implementation constraints
4. Any domain-specific docs relevant to the change

### Research Questions

- Does the current architecture already address this? (If yes, follow existing guidance.)
- What are the constraints from `CLAUDE.md`? (Technology choices are often fixed.)
- What is the impact on existing code? (How many files/packages affected?)
- What is the impact on the offline-first PWA strategy?
- What are the security implications? (Consult `docs/18-threat-model.md` if relevant.)

**Output**: Research summary with findings from each document consulted.

---

## Step 3: Draft ADR (Architecture Decision Record)

Fill the ADR template (`templates/adr.md`) with the following structure:

```markdown
# ADR-NNN: [Title]

## Status
PROPOSED

## Date
YYYY-MM-DD

## Context
[What is the situation? Why does a decision need to be made?]

## Decision Drivers
- [Driver 1: constraint, requirement, or goal]
- [Driver 2]
- [Driver 3]

## Options Considered

### Option A: [Name]
- Description: ...
- Pros: ...
- Cons: ...
- Impact on existing code: ...

### Option B: [Name]
- Description: ...
- Pros: ...
- Cons: ...
- Impact on existing code: ...

### Option C: [Name] (if applicable)
- Description: ...
- Pros: ...
- Cons: ...
- Impact on existing code: ...

## Decision
[Which option was chosen and why]

## Consequences
- [Positive consequence 1]
- [Positive consequence 2]
- [Negative consequence / trade-off 1]
- [Migration/refactoring required]

## Affected Documentation
- [List of docs that need updating if this decision is accepted]
```

**Output**: Completed ADR document.

---

## Step 4: Review with Tech Lead

Submit the ADR for review by the `tech-lead` agent.

### Review Criteria

The tech-lead agent evaluates:

1. **Alignment**: Does the decision align with `docs/05-architecture-proposal.md`?
2. **Scope**: Does the decision respect phase boundaries from `docs/07-mvp-scope.md`?
3. **Simplicity**: Is this the simplest solution that meets the requirement?
4. **Reversibility**: Can this decision be reversed later if needed?
5. **Impact**: What is the blast radius if this goes wrong?
6. **Security**: Does this introduce new attack surface? (Involve `security-engineer` agent if yes.)
7. **Constraints**: Does this violate any MUST/MUST NOT rules from `CLAUDE.md`?

### Review Outcomes

| Outcome | Next Step |
|---------|-----------|
| **APPROVED** | Proceed to Step 5 |
| **APPROVED WITH CONDITIONS** | Address conditions, update ADR, proceed to Step 5 |
| **NEEDS REVISION** | Revise ADR based on feedback, return to Step 3 |
| **REJECTED** | Document rejection reason in ADR, status = REJECTED, stop |

---

## Step 5: Accept Decision

1. Update ADR status from `PROPOSED` to `ACCEPTED`
2. Store the ADR in `docs/adrs/` (create directory if it does not exist)
3. Name the file: `docs/adrs/ADR-NNN-short-title.md`
4. Commit the ADR:
   ```
   docs: ADR-NNN <short title>
   ```

---

## Step 6: Update Affected Documentation

Run the `maintain-docs` skill to identify all documents that need updating based on the decision.

### Commonly Affected Documents

| Decision Area | Documents to Update |
|--------------|-------------------|
| Architecture pattern | `docs/05-architecture-proposal.md` |
| API design | `docs/11-api-design.md` |
| Data model | `docs/10-data-model.md` |
| Offline strategy | `docs/12-offline-sync-strategy.md` |
| Security | `docs/13-security-and-compliance.md`, `docs/18-threat-model.md` |
| IAM/Auth | `docs/16-iam-and-access-control.md`, `docs/20-keycloak-configuration.md` |
| Implementation | `docs/15-implementation-guidelines.md` |
| Project rules | `CLAUDE.md` (if new constraints are introduced) |

Commit documentation updates:
```
docs: update docs for ADR-NNN <short title>
```

---

## Step 7: Implement

Follow the standard `feature-delivery.md` workflow for implementation.

The ADR serves as the design document — reference it in the feature spec and code review.

---

## Agent Assignments

| Step | Agent |
|------|-------|
| 1. Identify | tech-lead |
| 2. Research | tech-lead (delegate to subagents for parallel research) |
| 3. Draft ADR | tech-lead |
| 4. Review | tech-lead (involve security-engineer if security-related) |
| 5. Accept | tech-lead |
| 6. Update docs | tech-lead (using maintain-docs skill) |
| 7. Implement | backend-engineer / frontend-engineer (per feature-delivery.md) |

---

## Quick Reference

```
Skills:  maintain-docs, review-security (if security-related)
Agents:  tech-lead (primary), security-engineer (if security-related)
Template: adr (ADR template)
Storage: docs/adrs/ADR-NNN-short-title.md
Commit:  docs: ADR-NNN <short title>
```
