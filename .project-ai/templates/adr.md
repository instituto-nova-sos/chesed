# ADR-[NNN]: [Title]

## Metadata

| Field | Value |
|-------|-------|
| **ADR Number** | ADR-[NNN] |
| **Date** | [YYYY-MM-DD] |
| **Status** | Proposed / Accepted / Deprecated / Superseded by ADR-[NNN] |
| **Deciders** | [Names or roles involved in the decision] |

---

## Context

Describe the technical or architectural issue that requires a decision. Include:

- What problem or need triggered this decision
- What constraints exist (budget, timeline, team skills, NGO operational requirements)
- What is the current state (if changing an existing approach)
- Reference relevant Chesed docs if applicable:
  - Architecture: `docs/05-architecture-proposal.md`
  - Requirements: `docs/03-requirements-catalog.md`
  - Security: `docs/13-security-and-compliance.md`
  - MVP scope: `docs/07-mvp-scope.md`

---

## Decision

State the decision clearly and concisely. Use active voice.

Example: "We will use Keycloak as the identity provider for all authentication and authorization, replacing the custom JWT implementation."

If the decision involves configuration or code patterns, include a brief example:

```
[Code or configuration snippet illustrating the decision]
```

---

## Consequences

### What becomes easier

- [Positive consequence 1]
- [Positive consequence 2]

### What becomes harder

- [Negative consequence or trade-off 1]
- [Negative consequence or trade-off 2]

### Risks

- [Risk 1 and how it will be monitored/mitigated]

### Impact on existing code

- [Files, modules, or patterns affected]
- [Migration effort required, if any]

---

## Alternatives Considered

### Alternative 1: [Name]

**Description**: [Brief description of the alternative]

**Rejection reason**: [Why this was not chosen]

| Pros | Cons |
|------|------|
| [Pro 1] | [Con 1] |
| [Pro 2] | [Con 2] |

### Alternative 2: [Name]

**Description**: [Brief description of the alternative]

**Rejection reason**: [Why this was not chosen]

| Pros | Cons |
|------|------|
| [Pro 1] | [Con 1] |
| [Pro 2] | [Con 2] |

### Alternative 3: [Name] (if applicable)

**Description**: [Brief description]

**Rejection reason**: [Why this was not chosen]

---

## References

- [Link or path to relevant documentation, e.g., `docs/05-architecture-proposal.md`]
- [Link to relevant external resource, RFC, or library documentation]
- [Related ADR, e.g., "Supersedes ADR-001" or "Builds on ADR-005"]
- [Relevant story or epic from `docs/09-backlog.md`]
