# Definition of Done (DoD)

The Definition of Done is a **blocking** gate. Work is not "done" because code
exists — it is done when the behavior is **verified by running the app**, the
tests prove the contract, and the documentation matches reality.

DoD applies at three levels: per **Story**, per **Feature**, and per **Sprint**.
Each level builds on the one below it. This checklist is wired into the
`pre-review` hook and into `make deliver` — the autonomous delivery pipeline must
not print `READY-FOR-PR` unless the applicable DoD level passes.

> Reviewer note: the `make deliver` gate and the `pre-review` hook treat this
> checklist as blocking. A story/feature/sprint that does not satisfy its DoD
> level fails the gate; there is no "code merged, criteria checked later".

---

## Level 1 — Story DoD

A single story (e.g. `S05.3`) is done when **every one of its Given/When/Then
acceptance criteria has been verified by running the app**, not merely by reading
the code.

- [ ] Every acceptance criterion in the story (`docs/09-backlog.md`) is satisfied and was **observed running** — backend endpoint exercised (real request/response), or frontend behavior reproduced in the browser, online and offline as the criterion specifies.
- [ ] Each "Given/When/Then" scenario has a corresponding automated test (unit, integration, or E2E) at the appropriate layer. "It works on my screen" is not sufficient on its own — but neither is a green test without having run the behavior.
- [ ] Error and boundary scenarios in the criteria (e.g. `403` cross-campus, `400` validation, `409` illegal transition, idempotent duplicate `sync_id`) are demonstrated, not assumed.
- [ ] Offline behavior described in the story's `offline:` field is verified (record persists locally, queue drains, or the online-only message shows).
- [ ] Audit logging and campus scoping are confirmed for any data mutation in the story.
- [ ] The story `status:` in `docs/09-backlog.md` is moved to `done` (source of truth); the board is regenerated with `make status`.

---

## Level 2 — Feature DoD

A feature is a story or a tightly related group delivered together. In addition
to every constituent story passing Level 1:

- [ ] All unit tests pass (backend `make test`, frontend `npm test`).
- [ ] All integration tests pass for every new endpoint and every new client-server contract (backend `make test-integration`, frontend `npm run test:integration`) — per the Integration Test Mandate in `CLAUDE.md`.
- [ ] Lint is clean: Go (`make lint`, golangci-lint) and TypeScript (`npm run lint`, `npm run typecheck`) with zero warnings.
- [ ] Coverage meets the layer thresholds in `.project-ai/rules/test-coverage-enforcement.md`.
- [ ] Documentation updated to match the implementation: `docs/11-api-design.md` for endpoint changes, `docs/10-data-model.md` for schema changes, `docs/04-domain-model.md` for domain changes, `docs/12-offline-sync-strategy.md` for offline changes.
- [ ] `make validate-backlog` passes (status enum, `depends_on`, `covers_requirements` integrity).
- [ ] The feature's acceptance criteria are confirmed end-to-end one more time at the feature boundary (not just per isolated story).
- [ ] Build succeeds for both stacks (`make build`, `npm run build`).

---

## Level 3 — Sprint DoD

A sprint is done when the whole increment is releasable:

- [ ] Every sprint story in `docs/08-roadmap.md` is `done` in the backlog (verified via `make status`) — or **explicitly deferred with rationale** recorded. No silent partials.
- [ ] **No orphan deferrals**: any deferred story names the sprint/phase it moves to and the reason; deferral is a decision, not an accident.
- [ ] The Requirement Coverage Audit passes for the sprint's target requirements (see `.project-ai/rules/backlog-traceability.md`): no in-scope Phase 1 RF/RNF is left uncovered.
- [ ] All Level 2 (Feature) gates pass for the sprint as a whole.
- [ ] Sprint-level checklist (`.project-ai/checklists/sprint-release.md`) passes.
- [ ] Critical-flow E2E and the auth-integration test are green (where applicable to the sprint).
- [ ] `HANDOFF.md` is updated with the sprint summary: what was built, key decisions, known issues/tech debt, and recommended next steps (narrative only — status lives in the backlog).

---

## How to Use

- Run **Level 1** as each story is finished — before moving its status to `done`.
- Run **Level 2** before opening a feature for review / before `make deliver` prints `READY-FOR-PR`.
- Run **Level 3** at sprint end, alongside `.project-ai/checklists/sprint-release.md`.

If any item fails, the work is not done. Fix it (use the `refactor-for-quality`
playbook for quality failures) and re-run the level.

## References

- `docs/09-backlog.md` — acceptance criteria and story status (source of truth)
- `.project-ai/rules/ready-definition.md` — entry gate (counterpart to this exit gate)
- `.project-ai/rules/backlog-traceability.md` — coverage audit and source-of-truth rules
- `.project-ai/rules/test-coverage-enforcement.md` — per-layer coverage thresholds
- `.project-ai/checklists/sprint-release.md` — sprint release checklist
- `CLAUDE.md` — Quality Bar and Integration Test Mandate
