# Skill: Validate Acceptance Criteria

## Purpose

Evaluate whether acceptance criteria for a backlog story are complete, testable, unambiguous, and implementable given the current architecture. Produces a validation report with pass/fail per criterion and improvement suggestions.

## When to Use / Trigger

- After the `refine-requirements` skill produces a story.
- Before sprint planning — validate all stories in the upcoming sprint.
- When a user says "check these acceptance criteria" or "are these criteria testable?".
- During the PLAN phase of feature delivery.

## Role / Expertise

QA-minded product analyst who evaluates acceptance criteria for testability, completeness, and implementability.

## Inputs

| Input | Required | Source |
|-------|----------|--------|
| Story with acceptance criteria | Yes | `docs/09-backlog.md` or refine-requirements output |
| Requirements catalog | Yes | `docs/03-requirements-catalog.md` |
| API design | Optional | `docs/11-api-design.md` |
| Data model | Optional | `docs/10-data-model.md` |
| Architecture | Optional | `docs/05-architecture-proposal.md` |

## Process

### Step 1: Completeness Check

For the story's acceptance criteria as a whole:
1. Are all happy path scenarios covered?
2. Are key error scenarios covered (validation errors, auth failures, not found)?
3. Is the RBAC requirement covered (which roles can/cannot access)?
4. Is campus isolation covered (user sees only their campus data)?
5. Is audit logging covered (mutations create audit entries)?
6. Is offline behavior covered (if applicable)?

### Step 2: Individual Criterion Validation

For EACH acceptance criterion, evaluate:

| Check | PASS Condition |
|-------|---------------|
| **Testable** | Can be verified by an automated test (unit, integration, or E2E) |
| **Specific** | Uses concrete values, not vague terms ("appropriate", "correct", "valid") |
| **Single behavior** | Describes exactly one observable behavior |
| **Independent** | Can be verified without depending on other criteria's state |
| **Phase-aligned** | Does not reference Phase 2/3 capabilities |
| **Implementable** | Can be implemented with the current architecture and tech stack |

### Step 3: Ambiguity Detection

Flag any criterion that contains:
- Subjective terms: "user-friendly", "fast", "appropriate", "reasonable".
- Missing specifics: "proper error handling" (which errors? what response?).
- Implicit assumptions: "works correctly" (define "correctly").
- Multiple behaviors: "creates the record and sends a notification" (split into two criteria).
- Undefined references: "follows the standard format" (which standard? specify).

### Step 4: Gap Identification

Compare acceptance criteria against standard checks for Chesed features:

| Standard Check | Present? | Required? |
|---------------|----------|-----------|
| Happy path CRUD operations | | If API endpoint |
| Validation error handling | | If user input |
| Authentication (401) scenario | | Always |
| Authorization (403) scenario | | Always |
| Not found (404) scenario | | If GET/PUT/DELETE |
| Duplicate/conflict (409) scenario | | If CREATE |
| Campus isolation verification | | Always |
| Audit log creation | | If mutation |
| Pagination behavior | | If list endpoint |
| Offline behavior | | If frontend page |

### Step 5: Generate Validation Report

```markdown
## Acceptance Criteria Validation Report

**Story**: [STORY-NNN] [Title]
**Overall Verdict**: PASS / FAIL / NEEDS REVISION

### Per-Criterion Results

| # | Criterion Summary | Testable | Specific | Single | Independent | Phase-aligned | Verdict |
|---|------------------|----------|----------|--------|-------------|---------------|---------|
| 1 | | ✓/✗ | ✓/✗ | ✓/✗ | ✓/✗ | ✓/✗ | PASS/FAIL |
| 2 | | ✓/✗ | ✓/✗ | ✓/✗ | ✓/✗ | ✓/✗ | PASS/FAIL |

### Ambiguities Found
- Criterion N: "[vague term]" — suggest replacing with "[specific term]"

### Missing Criteria
- [Standard check] is not covered — suggest adding: "Given [context], when [action], then [result]"

### Suggested Improvements
- [Specific rewording suggestions for failing criteria]
```

## Outputs / Deliverables

1. **Validation report** with per-criterion pass/fail and overall verdict.
2. **Ambiguity flags** with specific terms to replace.
3. **Missing criteria** with suggested additions.
4. **Improvement suggestions** with concrete rewording proposals.

## References

| Document | Path | Usage |
|----------|------|-------|
| Requirements | `docs/03-requirements-catalog.md` | Requirement context |
| API design | `docs/11-api-design.md` | Endpoint behavior expectations |
| Data model | `docs/10-data-model.md` | Entity constraints |
| IAM | `docs/16-iam-and-access-control.md` | RBAC requirements |

## Constraints / Quality Bar

- Every criterion must be evaluated — no criterion skipped.
- FAIL verdict if any criterion fails the "Testable" check.
- FAIL verdict if authentication/authorization scenarios are missing.
- FAIL verdict if campus isolation is not covered (for data-accessing features).
- Suggestions must be concrete (provide the exact rewording, not just "make it more specific").

## Interaction with Other Artifacts

- **Invoked by agents**: product-analyst (primary), tech-lead (sprint planning validation), qa-engineer (test design input).
- **Depends on skills**: refine-requirements (produces the story being validated).
- **Feeds into skills**: analyze-requirements (validated story ready for task breakdown), design-test-plan (validated criteria become test cases).
- **Governed by rules**: phase-boundary (criteria must not reference future phases).
