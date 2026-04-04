# Skill: Refine Requirements

## Purpose

Transform a vague business request or rough story description into a properly structured backlog story with clear acceptance criteria, requirement traceability (RF-XX), scope validation, and implementation readiness assessment.

## When to Use / Trigger

- When a stakeholder provides an informal feature request.
- When a backlog story has incomplete or vague acceptance criteria.
- When a user says "add this feature" or "we need X" without formal structure.
- During backlog grooming sessions.
- Before the `analyze-requirements` skill (which assumes a well-formed story exists).

## Role / Expertise

Product analyst who structures business needs into implementable, testable backlog stories following the project's requirements conventions.

## Inputs

| Input | Required | Source |
|-------|----------|--------|
| Raw business request or rough description | Yes | User input or stakeholder communication |
| Product vision | Yes | `docs/01-product-vision.md` |
| Requirements catalog | Yes | `docs/03-requirements-catalog.md` |
| MVP scope | Yes | `docs/07-mvp-scope.md` |
| Existing backlog | Yes | `docs/09-backlog.md` |
| Domain model | Optional | `docs/04-domain-model.md` |

## Process

### Step 1: Understand the Request

1. Read the raw business request.
2. Identify the core user need (Who needs what? Why?).
3. Determine which user role(s) this serves (volunteer, secretary, professional, coordinator, admin).
4. Check if this request maps to an existing RF-XX requirement in `docs/03-requirements-catalog.md`.
5. If no existing requirement, assess whether it belongs in the product scope per `docs/01-product-vision.md`.

### Step 2: Validate Scope

1. Check `docs/07-mvp-scope.md`:
   - Is this feature in Phase 1 scope? → Proceed.
   - Is this feature explicitly in Phase 2/3? → Document but do not create implementation story.
   - Is this feature not mentioned? → Evaluate fit with Phase 1 goals, discuss with tech-lead.
2. If the request introduces a new domain entity, check if it's in the Phase 1 table list.
3. If scope is unclear, flag for stakeholder discussion before proceeding.

### Step 3: Structure the Story

Write the story in this format:

```markdown
### [STORY-NNN] [Title]

**Requirement**: RF-XX (from docs/03-requirements-catalog.md)
**Phase**: Phase 1 / Phase 2 / Phase 3
**Sprint**: Sprint N (suggested)
**Priority**: Must / Should / Nice-to-Have
**Complexity**: S / M / L / XL

**As a** [role],
**I want to** [action],
**So that** [benefit].

**Acceptance Criteria:**
1. Given [precondition], when [action], then [expected result].
2. Given [precondition], when [action], then [expected result].
3. [Error scenario]: Given [precondition], when [invalid action], then [error handling].

**Security considerations:**
- PII involved: [Yes/No — which fields]
- RBAC: [Which roles can access]
- Audit logging: [Which mutations need auditing]

**Offline behavior:**
- Category: [A: Full offline / B: Read-only offline / C: Online-only]
- Degradation: [What happens when offline]

**Dependencies:**
- [STORY-XXX] must be complete first (reason)
```

### Step 4: Write Acceptance Criteria

For each acceptance criterion:
1. Use Given/When/Then format for clarity.
2. Include at least one happy path scenario.
3. Include at least one error/edge case scenario.
4. Use concrete values (not "appropriate" or "valid" — specify exactly).
5. Ensure each criterion is independently testable.
6. Verify criteria don't reference Phase 2/3 capabilities.

### Step 5: Assess Readiness

Evaluate the story against the INVEST checklist:
- [ ] Independent: Can be completed without in-progress dependencies.
- [ ] Negotiable: Details can be refined during implementation.
- [ ] Valuable: Delivers clear user or business value.
- [ ] Estimable: Complexity can be assessed (S/M/L/XL).
- [ ] Small: Can be completed within one sprint.
- [ ] Testable: All acceptance criteria can be automated.

If any INVEST criterion fails, refine further or split the story.

## Outputs / Deliverables

1. **Structured story** in the format above, ready for addition to `docs/09-backlog.md`.
2. **Scope validation** confirming the feature belongs in the current phase.
3. **RF-XX mapping** linking the story to documented requirements.
4. **Readiness assessment** against INVEST criteria.
5. **Recommendation**: Ready for sprint planning / Needs more refinement / Out of scope.

## References

| Document | Path | Usage |
|----------|------|-------|
| Product vision | `docs/01-product-vision.md` | Scope validation |
| Requirements | `docs/03-requirements-catalog.md` | RF-XX mapping |
| MVP scope | `docs/07-mvp-scope.md` | Phase validation |
| Backlog | `docs/09-backlog.md` | Existing stories context |
| Domain model | `docs/04-domain-model.md` | Entity context |

## Constraints / Quality Bar

- Every story must reference at least one RF-XX requirement code.
- Every acceptance criterion must be independently testable.
- No acceptance criteria may reference Phase 2/3 capabilities (if story is Phase 1).
- Stories must not duplicate existing backlog entries.
- Complexity estimates must be provided (S/M/L/XL).
- Security and offline considerations must be documented.

## Interaction with Other Artifacts

- **Invoked by agents**: product-analyst (primary), tech-lead (backlog grooming).
- **Feeds into skills**: analyze-requirements (consumes the refined story), validate-acceptance-criteria (validates the criteria).
- **Governed by rules**: phase-boundary (scope validation), documentation-first (requirements before implementation).
