# Rule: Backlog Traceability

## Purpose

Ensure every code change is traceable to a documented story, technical enabler, or chore. Maintain a clear audit trail from requirements to implementation.

## Rule Statement

Every commit message must reference a story ID from `docs/09-backlog.md`. Branch names must reference the feature area. Non-story work must be tagged explicitly. Feature branches must be scoped to one story or a closely related group.

## Trigger Condition

Every time the AI agent creates a commit, creates a branch, or completes a unit of work.

## Enforcement

### Commit Messages

1. **Format**: `<type>: <story-id> - <short description>`
   - Story work: `feat: S03.1 - add person creation endpoint`
   - Bug fix: `fix: S03.1 - handle duplicate document_number`
   - Refactor: `refactor: S03.1 - extract validation to service layer`
   - Test: `test: S03.1 - add person repository unit tests`
   - Documentation: `docs: S03.1 - update API design for person endpoints`

2. **Non-story work** uses explicit tags:
   - Technical enabler: `chore: TE01 - configure golangci-lint`
   - Infrastructure: `ci: TE02 - add GitHub Actions workflow`
   - General chore: `chore: setup Docker Compose for local development`

3. **Types** (matching project convention):
   - `feat` — New feature implementation
   - `fix` — Bug fix
   - `refactor` — Code restructuring without behavior change
   - `test` — Adding or modifying tests
   - `docs` — Documentation changes
   - `chore` — Maintenance, tooling, configuration
   - `ci` — CI/CD pipeline changes

### Branch Names

4. **Feature branches**: `feat/<area>-<short-description>`
   - Examples: `feat/person-crud`, `feat/triage-workflow`, `feat/auth-middleware`

5. **Fix branches**: `fix/<area>-<short-description>`
   - Examples: `fix/person-validation`, `fix/audit-campus-filter`

6. **Scope**: One branch per story or tightly related story group. Do not bundle unrelated stories in a single branch.

### Traceability Verification

7. Before committing, verify the story ID exists in `docs/09-backlog.md`.
8. If the work does not map to any story, classify it as a technical enabler (TE) or chore and document why.
9. Never use vague commit messages like "fix stuff", "updates", or "WIP" in the main branch.

## Enforcement Mechanism

- The AI agent must include a story ID or TE tag in every commit message.
- The `pre-review` hook verifies commit message format before marking work complete.
- If the agent cannot identify a story ID for the work being done, it must pause and either find the relevant story or create a TE entry.

## References

- `docs/09-backlog.md` — Story definitions and IDs
- `docs/08-roadmap.md` — Sprint assignments
- `CLAUDE.md` — Commit convention section

## Consequences of Skipping

- Untraceable commits make it impossible to understand why a change was made.
- Missing story references break the ability to verify all requirements are implemented.
- Vague commit messages make debugging, rollbacks, and code reviews significantly harder.
- Unscoped branches with mixed stories create merge conflicts and make partial reverts impossible.
