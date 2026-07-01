# Rule: Backlog Traceability

## Purpose

Ensure every code change is traceable to a documented story, technical enabler, or chore. Maintain a clear audit trail from requirements to implementation.

## Rule Statement

Every commit message must reference a story ID from `docs/09-backlog.md`. Branch names must reference the feature area. Non-story work must be tagged explicitly. Feature branches must be scoped to one story or a closely related group.

## Source of Truth

1. **`docs/09-backlog.md` is the SINGLE SOURCE OF TRUTH for story status.** Story state (`backlog | ready | in_progress | review | blocked | done`), dependencies, requirement coverage, sizing, and acceptance criteria live in the backlog and nowhere else. When status changes, update the backlog — not a side document.
2. **`tasks/STATUS.md` is GENERATED, never hand-edited.** It is produced by `scripts/generate-status.sh` via `make status` and is git-ignored. Treat any manual edit to it as a defect; it will be overwritten on the next generation. If the board looks wrong, fix the backlog and regenerate.
3. **`HANDOFF.md` is narrative only.** It records session history, decisions, and next steps in prose. It MUST NOT be used as a status board or a status source of truth — derive status from the backlog, not from HANDOFF.

## Required Story Metadata (Phase 1)

Every Phase 1 story (epics E01–E06) that is annotated for delivery MUST carry, at minimum:

- `status:` — one of `backlog | ready | in_progress | review | blocked | done`.
- `covers_requirements:` — a non-empty list of real RF/RNF IDs from `docs/03-requirements-catalog.md`. Every Phase 1 story must trace to at least one requirement.
- `depends_on:` — the list of story IDs that must be `done` before this story can enter `in_progress` (may be empty `[]`).

Recommended companions: `parallel_with:`, `size:` (S|M|L), and `offline:` (one line on offline behavior). Acceptance criteria for Phase 1 stories MUST be written in Given/When/Then form. Phase 2/3 stories defer detailed criteria to phase kickoff per the phase-boundary rule.

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

## Requirement Coverage Audit

A backlog that drifts from the requirements catalog silently drops scope. To prevent this, run a coverage audit at three checkpoints:

- **Sprint planning** — before committing a sprint, confirm the sprint's stories collectively cover their target requirements.
- **Sprint release** — before tagging a sprint (see `.project-ai/checklists/sprint-release.md`), confirm no Phase 1 requirement targeted by the sprint is left uncovered.
- **Phase boundary** — before closing Phase 1, confirm **every Phase 1 RF/RNF is covered by at least one story** via `covers_requirements`.

Audit rule: **every Phase 1 functional and non-functional requirement that is in MVP scope must be referenced by `covers_requirements` on at least one story.** A requirement with zero covering stories is a coverage gap and blocks the checkpoint until either a story is added or the requirement is explicitly deferred (with rationale) per the phase-boundary rule.

Mechanical checks are provided by `make validate-backlog`, which fails (exit 1) when:

- a `status:` value is outside the enum,
- a `depends_on:` ID does not exist as a story in `docs/09-backlog.md`, or
- a `covers_requirements:` ID does not exist in `docs/03-requirements-catalog.md`.

`make validate-backlog` validates referential integrity; the human/agent reviewer performs the inverse coverage check (every in-scope requirement has ≥1 story) at the checkpoints above. Regenerate the board with `make status` before reviewing.

## Enforcement Mechanism

- The AI agent must include a story ID or TE tag in every commit message.
- The `pre-review` hook verifies commit message format before marking work complete.
- The AI agent runs `make validate-backlog` before marking PM work complete and at every checkpoint above; a non-zero exit blocks progress.
- If the agent cannot identify a story ID for the work being done, it must pause and either find the relevant story or create a TE entry.

## References

- `docs/09-backlog.md` — Story definitions, status (source of truth), and IDs
- `docs/03-requirements-catalog.md` — RF/RNF IDs for the coverage audit
- `docs/08-roadmap.md` — Sprint assignments
- `scripts/generate-status.sh` / `make status` — generated `tasks/STATUS.md`
- `make validate-backlog` — referential-integrity check for backlog metadata
- `.project-ai/rules/ready-definition.md` — when a story may enter `in_progress`
- `.project-ai/checklists/definition-of-done.md` — completion gate per story/feature/sprint
- `CLAUDE.md` — Commit convention section

## Consequences of Skipping

- Untraceable commits make it impossible to understand why a change was made.
- Missing story references break the ability to verify all requirements are implemented.
- Vague commit messages make debugging, rollbacks, and code reviews significantly harder.
- Unscoped branches with mixed stories create merge conflicts and make partial reverts impossible.
