# Hook: Post-Implementation Verification

## Purpose

Automated verification checkpoint after implementation is complete but before marking work for review. Ensures code is tested, linted, and documentation-ready before entering the review pipeline.

## Trigger Condition

After completing implementation of a story — all code is written, tests are written, but before invoking the `pre-review` hook or marking the story for review.

## Status

**Non-blocking** — but mandatory before proceeding to review. Findings must be addressed before the `pre-review` hook is executed.

## Steps

1. **Run test suite**
   - Execute `go test -race -count=1 ./...` for backend changes.
   - Execute `npm test` for frontend changes.
   - Record pass/fail count and any failures.
   - If any tests fail, STOP. Fix failures before proceeding.

2. **Run linters**
   - Execute `golangci-lint run ./...` for Go code.
   - Execute `npx eslint .` for TypeScript code.
   - Record warning/error count.
   - If any lint errors exist, STOP. Fix errors before proceeding.

3. **Quick quality gate assessment**
   - Identify all files changed in the current story implementation.
   - For each changed file, check:
     - File length within limits (Go: 400, TS: 300 lines).
     - No function exceeds complexity thresholds.
     - No obvious duplication of logic.
   - If any file exceeds thresholds, flag for refactoring before review.

4. **Verify documentation sync**
   - Run `maintain-docs` skill in check mode (identify, don't update).
   - If new endpoints were added: verify `docs/11-api-design.md` is updated.
   - If new tables/columns were added: verify `docs/10-data-model.md` is updated.
   - If domain model changed: verify `docs/04-domain-model.md` is updated.
   - If any docs are out of sync, update them before proceeding.

5. **Check dependency changes**
   - If `go.mod` or `package.json` was modified:
     - List new dependencies added.
     - Flag for dependency-management rule review.
   - If no dependency-management justification exists, note it.

6. **Generate implementation summary**
   - List all files created or modified.
   - List all tests added (function names and file paths).
   - List any documentation updates made.
   - Note any deviations from the original design.
   - This summary feeds into the `pre-review` hook and reviewer agent.

## Enforcement Mechanism

- The AI agent must execute this hook automatically after completing implementation code.
- All findings from this hook must be resolved before executing the `pre-review` hook.
- If the hook identifies issues, the agent must fix them and re-run the hook.
- The implementation summary is passed to the reviewer agent for context.

## References

- `docs/quality/quality-gates.md` — Quality gate thresholds
- `docs/quality/complexity-guidelines.md` — File and function complexity limits
- `docs/quality/quality-profiles.md` — Stack-specific quality requirements

## Consequences of Skipping

- Broken tests reach the review stage, wasting reviewer time.
- Lint errors accumulate and become harder to fix in bulk.
- Documentation drift causes confusion between spec and implementation.
- Quality gate failures at pre-merge are more expensive to fix than at post-implement.
- Undocumented dependency additions bypass evaluation.
