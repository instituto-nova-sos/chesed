# Rule: Documentation-First Development

## Purpose

Ensure that documentation is the source of truth and always precedes or accompanies implementation. Prevent undocumented features, API drift, and knowledge loss.

## Rule Statement

Before implementing any feature, verify it is fully documented. Implementation must not proceed until the relevant documentation exists and is accurate.

## Trigger Condition

Every time the AI agent begins work on a new feature, modifies an API endpoint, changes a database schema, or alters the domain model.

## Enforcement

### Before Implementation

1. **API endpoint not in `docs/11-api-design.md`?**
   - STOP. Document the endpoint first: method, path, request/response schemas, status codes, RBAC roles.
   - Resume implementation only after documentation is written.

2. **Database table or column not in `docs/10-data-model.md`?**
   - STOP. Document the table first: columns, types, constraints, indexes, relationships.
   - Resume implementation only after documentation is written.

3. **Domain concept not in `docs/04-domain-model.md`?**
   - STOP. Document the entity, its attributes, and its relationships first.
   - Resume implementation only after documentation is written.

4. **Feature not in `docs/03-requirements-catalog.md`?**
   - STOP. Verify the feature belongs in the project scope. If yes, add it to the requirements catalog.
   - Resume implementation only after documentation is written.

### After Implementation

5. **Implementation deviates from documentation?**
   - Update the documentation to match the implementation (if the deviation is intentional).
   - Fix the implementation to match the documentation (if the deviation is unintentional).
   - Never leave a known discrepancy between docs and code.

6. **Session end or context switch?**
   - Update `HANDOFF.md` with current progress, decisions made, and next steps.
   - Ensure anyone (human or AI) can pick up where you left off.

### Exception

- **Critical hotfixes**: When a production bug requires immediate code changes, implementation may proceed first. However, documentation must be updated immediately after the fix is applied — within the same session, not deferred.

## Enforcement Mechanism

- The AI agent must check documentation existence as the first action in every implementation task.
- If documentation is missing, the agent must create it before writing any implementation code.
- At the end of every session, the agent must verify `HANDOFF.md` is current.
- The `pre-implement` and `pre-api-change` hooks enforce this rule automatically.

## References

- `docs/03-requirements-catalog.md` — Requirements definitions
- `docs/04-domain-model.md` — Domain entities and relationships
- `docs/10-data-model.md` — Database schema
- `docs/11-api-design.md` — API endpoint contracts
- `HANDOFF.md` — Session handoff document

## Consequences of Skipping

- Undocumented APIs create integration failures when frontend and backend develop in parallel.
- Undocumented schema changes cause confusion and inconsistency in domain structs.
- Missing handoff documentation causes context loss, duplicated work, and contradictory implementations across sessions.
- The documentation-first approach is the foundation of the entire project workflow — skipping it undermines every other quality gate.
