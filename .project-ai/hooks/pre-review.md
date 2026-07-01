# Hook: Pre-Review Self-Check

## Purpose

Comprehensive self-review checklist before considering any work complete. Ensures quality bar is met across testing, linting, documentation, security, and compliance.

## Trigger Condition

Before marking any story, task, or feature as complete. Before writing a "done" status in `tasks/todo.md`. Before creating a pull request.

## Status

**Blocking** — Do not mark work as complete until all applicable steps pass.

## Steps

1. **All tests pass**
   - Run `go test ./...` in `backend/` — zero failures.
   - Run `npx vitest run` in `frontend/` — zero failures.
   - New business logic must have unit tests with table-driven test patterns.
   - Coverage: all happy paths and key error paths tested.
   - If tests fail, fix them before proceeding. Do not mark as "known failure."

2. **No lint warnings**
   - Run `golangci-lint run ./...` in `backend/` — zero warnings.
   - Run `npx eslint .` in `frontend/` — zero warnings.
   - Run `npx tsc --noEmit` in `frontend/` — zero type errors.
   - No `any` type usage in TypeScript (strict mode enforced).
   - No ignored errors (`_`) in Go.

3. **Run quality gate validation**
   - Evaluate changed code against the New Code Quality Gate from `docs/quality/quality-gates.md`:
     - 0 new bugs (reliability issues).
     - 0 new vulnerabilities (security issues).
     - Coverage on new code ≥ 80%.
     - Duplication on new code ≤ 3%.
     - Maintainability rating = A.
     - Reliability rating = A.
     - Security rating = A.
   - Check complexity thresholds from `docs/quality/complexity-guidelines.md` on all changed functions.
   - If any condition fails, fix before proceeding. Do not mark work as complete with failing quality gates.

4. **Run the appropriate checklist**
   - If backend work: verify handler -> service -> repository layering is correct.
   - If frontend work: verify component -> hook -> API layer separation.
   - If migration work: verify up/down pair and domain struct alignment.
   - If full-stack: run all applicable checklists.
   - Reference the relevant checklist in `.project-ai/checklists/`.

5. **Run API contract review if API changed**
   - If any handler or route was modified, execute the `review-api-contract` skill.
   - Verify implementation matches `docs/11-api-design.md`.
   - Verify error response format is consistent.
   - Verify pagination format for list endpoints.

6. **Run security review if security-sensitive**
   - If the change touches any security-sensitive area (see `rules/security-review-triggers.md`), execute the `review-security` skill.
   - Areas requiring review: auth middleware, PII fields, Keycloak config, IndexedDB, sync logic, RBAC, audit logging.
   - Document any security findings and their resolution.

7. **Verify documentation is updated**
   - If API changed: `docs/11-api-design.md` reflects current implementation.
   - If schema changed: `docs/10-data-model.md` reflects current schema.
   - If domain model changed: `docs/04-domain-model.md` reflects current model.
   - If new patterns introduced: `docs/15-implementation-guidelines.md` updated.
   - `HANDOFF.md` updated with session progress.

8. **Verify commit messages follow convention**
   - Format: `<type>: <story-id> - <short description>`
   - Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `ci`
   - Story ID references backlog (e.g., S03.1) or tagged as TE/chore.
   - Each commit is atomic — one logical change per commit.

9. **TDD commit-order gate (RED before GREEN)** — **AUTHORITATIVE, BLOCKING**

   This is the authoritative enforcement of `.project-ai/rules/tdd-enforcement.md`
   (the `make deliver` heuristic is convenience only). For the feature branch,
   verify via `git log --name-only` that the **first commit touching test files
   precedes the first commit touching production files**. A production file landing
   in — or before — the first test-touching commit is a violation: it means code
   was written without a prior failing test.

   - **Scope**: the branch's own commits, in chronological order. Resolve the base
     with `git merge-base main HEAD`, then walk `git rev-list --reverse <base>..HEAD`.
     Per story (story ID in the commit subject, e.g. `S03.1`), the ordering must
     hold within that story's commit subsequence; for a single-story branch the
     whole branch is one subsequence.

   - **File classification (exact suffix / path matching)**:
     - **Test files** (extended regex):
       `(_test\.go|\.test\.ts|\.test\.tsx|\.integration\.test\.ts|\.integration\.test\.tsx)$`
       plus e2e specs under `frontend/e2e/` matching `\.spec\.ts$`.
     - **Production files**: any path under `backend/` or `frontend/src/` that is
       **not** a test file and **not** an Allowed Exception (pure config /
       scaffolding / generated code, per `tdd-enforcement.md`). A commit carrying a
       `No-Test-Rationale:` body line is treated as an allowed exception and does
       not count as a production-first violation.

   - **Check**: walk the commits in order; record the index of the first commit
     whose file set intersects the test regex (`first_test`) and the first commit
     whose file set contains a production file (`first_prod`). The gate PASSES when
     `first_test` is empty, or `first_prod` is empty, or `first_test <= first_prod`.
     The gate also requires the RED commit to carry **only** test files (no
     production file in the first test-touching commit).

   - **Failure message** (block, do not mark complete):
     ```
     TDD ORDER VIOLATION (pre-review): production code committed before a failing test.
       Story:            <story-id or branch>
       First test commit:  <sha/index or "none">
       First prod commit:  <sha/index> — <offending file(s)>
     Per .project-ai/rules/tdd-enforcement.md the RED test commit must precede the
     GREEN production commit. Do NOT rewrite history to fake the order. Treat as a
     process miss: re-derive the missing test, confirm it fails when the production
     code is reverted, and record the gap for the reviewer (REQUEST_CHANGES).
     ```

   - If the order cannot be proven RED-first, this hook **blocks**. Allowed
     exceptions (config/scaffolding/generated) must carry a `No-Test-Rationale:`
     note in the commit body to be skipped.

## Enforcement Mechanism

- The AI agent must execute this hook before any "task complete" declaration.
- If any step fails, the agent must fix the issue and re-run the failing step.
- The agent must not use hedging language ("should work", "probably fine") — all steps must demonstrably pass.

## References

- `docs/11-api-design.md` — API contracts
- `docs/10-data-model.md` — Database schema
- `docs/04-domain-model.md` — Domain model
- `docs/13-security-and-compliance.md` — Security requirements
- `docs/15-implementation-guidelines.md` — Coding standards
- `.project-ai/rules/security-review-triggers.md` — Security review criteria

## Consequences of Skipping

- Failing tests in the main branch block all other development.
- Lint warnings accumulate into unmaintainable code.
- Undocumented API changes break frontend development and integration.
- Missing security reviews leave vulnerabilities undetected.
- Inconsistent commit messages make git history unusable for debugging and auditing.
