# Autonomous Delivery Workflow

End-to-end autonomous loop that takes a feature branch from "code committed" to
"ready-for-PR" **without human intervention** — and stops at the single permitted
boundary: **before `git push`**. It chains every local quality gate via
`make deliver`, runs a **blocking critical-review gate**, and — on
`REQUEST_CHANGES` — auto-remediates under TDD for **up to 3 cycles** before
escalating to a human.

This workflow is the autonomous tail of `feature-delivery.md`. Phases 1–5 of
feature delivery (PLAN → DESIGN → IMPLEMENT → VERIFY → DOCUMENT) are unchanged;
this workflow is the single gate that produces the review file and the
`READY-FOR-PR` state.

---

## Hard Push Boundary (the whole point)

This workflow, `make deliver`, and every artifact it invokes **NEVER** run:

- `git push`
- `gh pr create`
- `gh pr merge`

The GitHub PAT has **no push / PR / merge permission**, and it will not be granted.
Delivery is **complete** when all of the following hold on the local branch:

1. Commits exist locally on the feature branch (RED→GREEN→REFACTOR order).
2. `tasks/review-<branch>.md` exists with verdict **`APPROVE`**.
3. `make deliver` printed the **`READY-FOR-PR`** banner.

The final step prints the **suggested** push command for a human to run; the agent
does not run it:

```
! git push -u origin <branch>
```

The leading `!` is a deliberate reminder that this line is human-operated, not an
agent action.

---

## Flow Overview

```
make deliver ──> [gates 1-5] ──> CRITICAL REVIEW (gate 6) ──> DoD (gate 7) ──> READY-FOR-PR
                                      │
                            REQUEST_CHANGES / NEEDS_DISCUSSION
                                      │
                                      v
                         refactor-for-quality (under TDD)
                                      │
                                      v
                             re-run make deliver
                                      │
                       (autonomous, up to 3 cycles, no human approval per cycle)
                                      │
                       not converged after 3 ──> record IMPASSE ──> STOP for human
```

---

## Preconditions

- Work is on a feature branch (not `main`); the implementation followed
  `feature-delivery.md` Phases 1–5, including test-first commits per
  `.project-ai/rules/tdd-enforcement.md`.
- The branch's commits are already made locally. (This workflow does not author
  the feature; it gates it.)

---

## Step 1 — Run the pipeline: `make deliver`

`make deliver` runs every local gate, fail-fast, in this order (see the root
`Makefile` for the authoritative recipe and which steps are Docker-gated):

1. `make validate-backlog` — backlog metadata integrity (real-shell).
2. TDD commit-order gate — best-effort heuristic in `make`; the **authoritative**
   RED→GREEN gate is `.project-ai/hooks/pre-review.md` (run it before opening the PR).
3. Backend: `build` + `lint` + `test`, then `test-integration` (**Docker-gated**:
   SKIPPED-NEEDS-DOCKER when Docker is absent; real failures fail the pipeline when
   Docker is present — this also covers `auth_middleware_test`).
4. Frontend: `typecheck` + `lint` + `test` + `test:integration` + `test:coverage`
   + `build` (real-shell).
5. `test:e2e:smoke` — Playwright against the real compose stack (**Docker-gated**:
   SKIPPED-NEEDS-DOCKER when Docker is absent; the **sprint gate requires it green**).
6. **Critical-review gate** — consumes `tasks/review-<branch>.md` (see Step 2).
7. **DoD gate** — the applicable Definition of Done level
   (`.project-ai/checklists/definition-of-done.md`), agent-verified.

On all green, the pipeline prints `READY-FOR-PR` plus the suggested push command
and **stops**.

---

## Step 2 — Critical-review gate (blocking)

Before gate 6 can pass, produce the verdict file by running the
**autonomous-critical-review** skill (`.project-ai/skills/autonomous-critical-review.md`):

1. The skill drives the `reviewer` agent (`.project-ai/agents/reviewer.md`) over
   `git diff main...HEAD` and the commit narrative `git log main..HEAD`.
2. It writes `tasks/review-<branch>.md` in the reviewer's report format
   (Quality Gate table, Clean Code, Complexity, Issues by severity, Verdict).
3. `make deliver`'s `deliver-review-gate` target greps that file:
   - **`APPROVE`** → gate passes, pipeline continues to the DoD gate.
   - **missing file / `REQUEST_CHANGES` / `NEEDS_DISCUSSION`** → gate FAILS, pipeline
     stops. Go to Step 3.

The verdict file is **regenerated on every `make deliver` run**, so the gate always
reflects the current diff.

---

## Step 3 — Auto-remediation loop (autonomous, up to 3 cycles)

> **User decision (precedence):** auto-remediation is autonomous up to **3 cycles**
> with **no human approval per cycle**. The push boundary is hard.

When the critical-review gate returns `REQUEST_CHANGES` (or `NEEDS_DISCUSSION`),
the orchestrator does the following **without asking for human approval**:

For `cycle` in 1..3:

1. **Read the required fixes** from `tasks/review-<branch>.md`.
2. **Apply `refactor-for-quality`** (`.project-ai/playbooks/refactor-for-quality.md`)
   to resolve the findings, **under TDD**: for any behavior change, first write a
   failing test that captures the intended fix, see it fail (**RED**), then make it
   pass (**GREEN**), then refactor green — committing in RED→GREEN order per
   `.project-ai/rules/tdd-enforcement.md`. Pure refactors (no behavior change) keep
   the suite green throughout and commit as `refactor:`.
3. **Re-run `make deliver`** (which regenerates the verdict file and re-runs every
   gate).
4. If the verdict is now `APPROVE` and all gates pass → the pipeline reaches
   `READY-FOR-PR`. **Done.** Exit the loop.
5. Otherwise continue to the next cycle.

If the loop completes **3 cycles without converging to APPROVE**:

1. **Do not** push, open a PR, or merge. **Do not** weaken any gate to force a pass.
2. **Record the impasse** in `tasks/review-<branch>.md`: append an `## IMPASSE`
   section listing the unresolved findings, what was attempted each cycle, and the
   specific decision a human must make.
3. **Stop** and surface the impasse for human decision. The branch stays as local
   commits; nothing is pushed.

---

## Step 4 — Stop-point (auditable)

On success, the only outputs are local and auditable:

- Local commits on the feature branch (RED→GREEN→REFACTOR).
- `tasks/review-<branch>.md` with `### Verdict: APPROVE`.
- The `READY-FOR-PR` banner from `make deliver`, including the **suggested**
  (not executed) `! git push -u origin <branch>`.

A human reviews these artifacts and runs the push. The agent's responsibility ends
here.

---

## Failure Handling Summary

| Situation | Action |
|-----------|--------|
| A real-shell gate (backlog, backend build/lint/test, frontend, present-Docker integration/e2e) fails | Pipeline stops; fix the underlying issue under TDD; re-run `make deliver`. |
| Docker absent | `test-integration` and `test:e2e:smoke` print SKIPPED-NEEDS-DOCKER and continue; the **sprint gate still requires them green** — run with Docker before release. |
| Review verdict ≠ APPROVE | Auto-remediate (Step 3), up to 3 cycles. |
| 3 cycles, still not APPROVE | Record `## IMPASSE` in the verdict file; STOP for human decision. No push. |
| Any temptation to push/PR/merge | Prohibited. Print the suggested push command only. |

---

## References

| Artifact | Path | Usage |
|----------|------|-------|
| Delivery pipeline | `Makefile` (`deliver`) | The single command this workflow drives |
| Critical-review skill | `.project-ai/skills/autonomous-critical-review.md` | Produces the verdict file |
| Reviewer agent | `.project-ai/agents/reviewer.md` | Verdict format and authority |
| Refactor playbook | `.project-ai/playbooks/refactor-for-quality.md` | Auto-remediation steps |
| TDD rule | `.project-ai/rules/tdd-enforcement.md` | RED→GREEN order during remediation |
| Pre-review hook | `.project-ai/hooks/pre-review.md` | Authoritative TDD commit-order gate |
| Pre-merge hook | `.project-ai/hooks/pre-merge.md` | APPROVE sourced from the verdict file |
| Definition of Done | `.project-ai/checklists/definition-of-done.md` | DoD gate (step 7) |
| Feature delivery | `.project-ai/workflows/feature-delivery.md` | Phases 1–5; this is its autonomous tail |
| Push boundary | `CLAUDE.md` — "Autonomous Delivery & Push Boundary" | Project-level rule |
