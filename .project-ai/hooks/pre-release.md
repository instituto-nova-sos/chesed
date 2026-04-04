# Hook: Pre-Release Sprint Gate Check

## Purpose

Final validation before tagging a sprint release. Ensures all sprint stories are complete, tested, documented, and ready for deployment.

## Trigger Condition

Before tagging a git release for a sprint milestone. Before declaring a sprint as complete.

## Status

**Blocking** — Do not tag a release or declare sprint complete until all steps pass.

## Steps

1. **Run release readiness assessment**
   - Execute the `assess-release-readiness` skill.
   - Verify the assessment covers:
     - All sprint stories implemented (check `docs/08-roadmap.md` for sprint scope)
     - All acceptance criteria met per `docs/09-backlog.md`
     - No critical bugs open
     - No failing tests
     - No lint warnings

2. **Run sprint release checklist**
   - Execute the `sprint-release` checklist from `.project-ai/checklists/`.
   - Confirm each item passes:
     - All migrations applied cleanly (up and down tested)
     - All API endpoints match documentation
     - All RBAC rules enforced
     - All audit logging in place for mutations
     - All campus scoping verified
     - Offline behavior documented for new pages
     - No hardcoded secrets in codebase

3. **Verify all sprint stories are implemented**
   - Open `docs/08-roadmap.md` and list all stories assigned to the current sprint.
   - Cross-reference with `tasks/todo.md` to confirm each story is marked complete.
   - For each story, verify:
     - Implementation exists in the codebase
     - Tests exist and pass
     - Documentation is current
   - If any story is incomplete, it must be either finished or explicitly deferred to the next sprint (with documentation of the reason).

4. **Run handoff preparation**
   - Execute the `prepare-handoff` skill.
   - Generate or update `HANDOFF.md` with:
     - Sprint summary (what was built)
     - Known issues or limitations
     - Next sprint priorities
     - Technical debt items identified
     - Environment setup notes if changed
   - Ensure `HANDOFF.md` is sufficient for another developer (or AI agent) to continue work.

5. **Enforce Overall Code Quality Gate**
   - Evaluate the entire codebase against the Overall Code Quality Gate from `docs/quality/quality-gates.md`:
     - [ ] 0 blocker severity issues.
     - [ ] 0 high severity issues.
     - [ ] Test coverage (overall) ≥ threshold (70% Sprint 1-2, 80% Sprint 3+).
     - [ ] Duplicated lines (overall) ≤ 5%.
     - [ ] Maintainability rating = A.
     - [ ] 0 reliability issues.
     - [ ] 0 security issues.
     - [ ] Security hotspots reviewed = 100%.
   - If any condition fails, the release is blocked. Resolve failures before proceeding.
   - Reference: `.project-ai/rules/quality-gates.md`

6. **Tag release in git**
   - Create an annotated git tag following the format: `sprint-N-vX.Y.Z`
     - Example: `sprint-1-v0.1.0`, `sprint-2-v0.2.0`
   - Tag message must include:
     - Sprint number and name
     - List of completed stories
     - Any known issues
   - Do NOT tag if any of steps 1-4 failed.

## Enforcement Mechanism

- The AI agent must execute this hook when the user requests a sprint release or when all sprint stories appear complete.
- If any step fails, the agent must report the specific failure and what needs to be resolved.
- Partial releases are not permitted — either all sprint stories pass or the release is deferred.

## References

- `docs/07-mvp-scope.md` — Phase 1 feature scope
- `docs/08-roadmap.md` — Sprint assignments and timeline
- `docs/09-backlog.md` — Story definitions and acceptance criteria
- `docs/11-api-design.md` — API documentation
- `docs/10-data-model.md` — Database schema
- `HANDOFF.md` — Session/sprint handoff document

## Consequences of Skipping

- Releasing incomplete sprints breaks the incremental delivery model and creates hidden debt.
- Missing handoff documentation causes context loss between sessions.
- Untagged releases make rollback impossible during production incidents.
- Skipping the release checklist risks deploying unprotected endpoints or missing audit logging.
