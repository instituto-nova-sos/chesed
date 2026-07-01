# Rule: TDD Enforcement (Red-Green-Refactor)

## Purpose

Mandate strict Test-Driven Development for **all production code**. Every line of production code must be preceded by a failing test that required it. This rule makes the discipline *verifiable*: it defines a commit-ordering protocol that proves the test was written first and seen to fail (RED) before any production code existed (GREEN). Implicit "I wrote tests too" is not sufficient — the evidence must be reconstructable from `git log`.

This rule extends the project's quality governance with an explicit, auditable RED→GREEN→REFACTOR proof.

## Rule Statement

No production code may be written, committed, or merged without a prior failing test that demanded it. The discipline follows the canonical cycle, and the cycle's first two phases must leave a trace in the commit history.

### The Cycle (mandatory for every unit of behavior)

1. **Write a failing test** that captures the desired behavior. Nothing else.
2. **Run the test and SEE it fail (RED)** — for the *expected* reason (assertion failure or missing symbol, not an unrelated compile error elsewhere). Never skip watching it fail.
3. **Write the minimum production code to pass (GREEN)** — no speculative generality, no extra behavior the test does not require.
4. **Run the full suite and confirm everything is green** — the new test and all existing tests.
5. **Refactor while keeping the suite green** — improve names, structure, and duplication without changing behavior; re-run after each refactor.

Production code without a prior failing test is **prohibited**. If you find yourself writing code "to be tested later", stop and write the test first.

## Proof of RED via Commit Ordering

The cycle is enforced through the order and naming of commits. For any given story/feature, scoped by its story ID (e.g. `S03.1`):

1. **The first commit that touches test files MUST precede the first commit that touches production files** for that same story.
   - **Test files** are matched by path/suffix: `*_test.go`, `*.test.ts`, `*.test.tsx`, `*.integration.test.ts`, `*.integration.test.tsx`, and e2e specs under `frontend/e2e/` (`*.spec.ts`).
   - **Production files** are everything else under `backend/` and `frontend/src/` that is not a test file (and not pure config/scaffolding — see Allowed exceptions).
2. **The RED commit contains only failing test files.** It must not include the production code that makes those tests pass. Its message is:
   ```
   test: <story-id> - <desc> (RED)
   ```
3. **The GREEN commit introduces the minimum production code** that turns the RED tests green. Its message is:
   ```
   feat: <story-id> - <desc> (GREEN)
   ```
   (Use `fix:` instead of `feat:` when the story is a bug fix; the `(GREEN)` marker and ordering rule are unchanged.)
4. **Refactor commits are optional** and come last, keeping the suite green. Their message is:
   ```
   refactor: <story-id> - <desc>
   ```

This is unambiguously checkable from `git log --name-only`: for a story's commits in chronological order, the file set of the first commit must consist of test files only, and at least one later commit introduces the matching production files. A production file appearing in (or before) the first test-touching commit for the story is a violation.

### Per-layer application

The cycle repeats once per layer/unit, not once per story. A backend story produces interleaved RED→GREEN pairs for repository, service, and handler; a frontend story produces them per hook/component and for the MSW api-client surface. Each pair still obeys the test-first ordering within the story's commit sequence. See `.project-ai/workflows/feature-delivery.md` Phase 3 (IMPLEMENT) for the layer-by-layer order.

## Allowed Exceptions

A small set of changes may legitimately have no prior failing test. When committing such code, the commit body MUST carry an explicit note (`No-Test-Rationale: <reason>`) so the omission is intentional and auditable:

- **Pure configuration** — `Makefile`, `vite.config.ts`, `tsconfig.json`, `go.mod`, lint/CI config, `package.json` scripts.
- **Scaffolding with no behavior** — empty type declarations, route wiring that only registers an already-tested handler, generated boilerplate.
- **Generated code** — code produced by a generator (codegen, migrations scaffold) that is verified by the generator itself or by downstream integration tests.

Exceptions are narrow. Anything containing a branch, a computation, a validation rule, or a data transformation is **not** an exception and requires a prior failing test.

## Trigger Condition

- Every commit that adds or changes production code under `backend/` or `frontend/src/`.
- Every story moving from `in_progress` toward `done`.
- Pre-review self-check, before a story is marked complete.

## Enforcement Mechanism

- **`pre-review` hook (`.project-ai/hooks/pre-review.md`)** — performs the git-log inspection: for each story touched on the branch, it verifies that the first test-touching commit precedes the first production-touching commit and that the RED commit carried only test files. If the order is violated and no Allowed-exception note is present, the hook **blocks**. The hook configuration (the exact `git log --name-only` parsing and story-scoping logic) is defined separately in that hook file; this rule defines *what* must hold, the hook defines *how* it is checked.
- **`backend-feature-complete.md` / `frontend-feature-complete.md` checklists** — carry a blocking item asserting the RED→GREEN commit order for the feature.
- **Reviewer agent** — confirms the commit narrative tells a coherent RED→GREEN→REFACTOR story during quality validation.

## Handling Failures

If the commit order cannot prove RED-first (e.g. test and production code landed in one commit):

1. Do **not** rewrite history to fake the order — that defeats the audit. Instead, treat it as a process miss.
2. For the next unit of behavior in the story, return to strict cycle: RED commit first, then GREEN.
3. If a whole story shipped without test-first evidence, the reviewer records it as a `REQUEST_CHANGES` finding; the remediation is to re-derive the missing tests and confirm they fail when the production code is reverted (the cross-cutting "tests fail without the production code" check in `integration-tests.md`).

## Consequences of Skipping

- Tests written after the fact codify the implementation's bugs instead of the intended behavior.
- "Green" suites that never went red give false confidence — a test that has never failed may assert nothing.
- Untested branches reach production; regressions go uncaught (see `rules/test-coverage-enforcement.md`).
- The RED→GREEN audit trail is the only durable proof the discipline held while CI is paused.

## References

- `.project-ai/hooks/pre-review.md` — the git-log gate that enforces commit ordering (configured separately)
- `.project-ai/workflows/feature-delivery.md` — Phase 3 IMPLEMENT, test-first per layer
- `.project-ai/checklists/backend-feature-complete.md` — backend TDD blocking item
- `.project-ai/checklists/frontend-feature-complete.md` — frontend TDD blocking item
- `.project-ai/checklists/test-distribution.md` — test pyramid targets (unit/integration/E2E mix)
- `.project-ai/checklists/integration-tests.md` — integration test mandate; "tests fail without the production code" sanity check
- `.project-ai/rules/test-coverage-enforcement.md` — per-layer coverage thresholds
- `.project-ai/playbooks/refactor-for-quality.md` — refactor phase guidance
- `docs/quality/quality-gates.md` — overall quality gate conditions
- `CLAUDE.md` — global TDD rule and "Integration Test Mandate"
