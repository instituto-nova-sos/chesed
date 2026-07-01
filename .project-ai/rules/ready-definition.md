# Rule: Definition of Ready

## Purpose

Define the entry gate for implementation. A story must be **ready** before it can
move to `in_progress`. This prevents starting work that is under-specified, blocked
on unmet dependencies, or missing the data/API substrate it needs — the most
common cause of mid-task rework.

This rule is the counterpart to `.project-ai/checklists/definition-of-done.md`:
"ready" gates entry into `in_progress`; "done" gates exit. Status values live in
`docs/09-backlog.md` (single source of truth).

## Rule Statement

A story may transition from `ready` (or `backlog`) to `in_progress` **only when ALL
of the following hold**:

1. **Acceptance criteria are in Given/When/Then form.** Each scenario is concrete
   and testable (a specific input, action, and observable outcome). Vague criteria
   like "works correctly" do not satisfy this and block readiness.
2. **All `depends_on` stories are `done`.** Check the `depends_on` field in the
   backlog; every referenced story must have `status: done`. Never start a story
   whose dependency graph is not yet satisfied.
3. **The technical substrate exists for the story's layer:**
   - **Backend stories** — the required database tables and migrations exist
     (applied), and the domain/repository scaffolding the story builds on is in
     place. A backend story that needs a not-yet-created table is **not ready**.
   - **Frontend stories** — the API endpoints the story consumes are **both
     documented** (in `docs/11-api-design.md`) **and implemented**. A frontend
     story that calls an endpoint that does not yet exist is **not ready**.
4. **No open questions.** There are no unresolved ambiguities, pending product
   decisions, or undecided trade-offs that would change the implementation. Open
   questions must be resolved (or explicitly scoped out) before work starts.

A story failing any condition stays `ready`/`backlog`/`blocked` and must not be
picked up. If a dependency or substrate is missing, address that first.

## The `ready` State

In the status enum (`backlog | ready | in_progress | review | blocked | done`),
`ready` is the staging state: criteria are written and well-formed, but one or
more entry conditions (typically a `depends_on` story or an API/table dependency)
may still be pending. A story is only **eligible to start** when it is `ready`
**and** all four conditions above are simultaneously true. Use `blocked` when a
dependency is known to be unmet and the story cannot proceed.

## Trigger Condition

Before moving any story to `in_progress` and beginning implementation work.

## Status

**Blocking** — Do not begin implementation if any readiness condition fails.

## Enforcement

- The AI agent verifies all four conditions against `docs/09-backlog.md` (and
  `docs/11-api-design.md` / migrations as relevant) before starting a story.
- This rule is the conceptual basis for the pre-implementation gate: the
  `pre-implement` hook already checks story existence, phase scope, sprint order,
  dependencies, and substrate (tables/endpoints). This rule names the **state
  transition** those checks gate — `ready → in_progress` — and adds the explicit
  "criteria in Given/When/Then" and "no open questions" conditions.
- If a story is not ready, the agent reports which condition failed and what must
  happen first (resolve a question, finish a dependency, create a migration,
  implement/document an endpoint) rather than starting partial work.

## References

- `docs/09-backlog.md` — story status, `depends_on`, acceptance criteria (source of truth)
- `docs/11-api-design.md` — API contracts that frontend stories depend on
- `.project-ai/hooks/pre-implement.md` — pre-implementation gate that enforces these checks in practice
- `.project-ai/checklists/definition-of-done.md` — the exit gate (counterpart to this entry gate)
- `.project-ai/rules/backlog-traceability.md` — required metadata and source-of-truth rules

## Consequences of Skipping

- Starting an under-specified story produces code that does not match intent and
  must be reworked once the criteria are clarified.
- Starting a story whose dependencies are not `done` causes integration failures
  and forces context-switching back to the unfinished prerequisite.
- Building a frontend story against a non-existent endpoint wastes effort on mocks
  that diverge from the real contract.
