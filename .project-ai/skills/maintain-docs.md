# Skill: Maintain Documentation

## Purpose

Update project documentation after code changes to keep docs in sync with the implementation. Ensures that API changes, database changes, domain model changes, architecture changes, and Keycloak configuration changes are reflected in the corresponding documentation files. Also handles HANDOFF.md updates at session end.

## When to Use / Trigger

- After implementing any API endpoint (update `docs/11-api-design.md`).
- After running a database migration (update `docs/10-data-model.md`).
- After modifying domain entities (update `docs/04-domain-model.md`).
- After changing architecture decisions (update `docs/05-architecture-proposal.md`).
- After modifying Keycloak realm config (update `docs/20-keycloak-configuration.md`).
- At the end of every coding session (update `HANDOFF.md`).
- When a user says "update docs" or "sync documentation".

## Role / Expertise

Technical writer who understands the Chesed documentation structure and can translate code changes into accurate documentation updates.

## Inputs

| Input | Required | Source |
|-------|----------|--------|
| Code changes (git diff or file list) | Yes | Implementation work |
| Current documentation | Yes | `docs/` directory |

## Process

### Step 1: Determine Which Docs Need Updates

Apply these rules based on what changed:

| Change Type | Document to Update |
|-------------|-------------------|
| API endpoint added/modified | `docs/11-api-design.md` |
| Database table added/modified | `docs/10-data-model.md` |
| Domain entity changed | `docs/04-domain-model.md` |
| Architecture decision changed | `docs/05-architecture-proposal.md` |
| Keycloak realm config changed | `docs/20-keycloak-configuration.md` |
| New requirement discovered | `docs/03-requirements-catalog.md` |
| MVP scope clarification | `docs/07-mvp-scope.md` |
| Security concern identified | `docs/13-security-and-compliance.md` or `docs/18-threat-model.md` |
| Offline sync behavior changed | `docs/12-offline-sync-strategy.md` |
| Backlog item completed | `docs/09-backlog.md` (mark as done) |
| Any session | `HANDOFF.md` |

### Step 2: Update API Design (docs/11-api-design.md)

When API endpoints change:

1. Update the endpoint summary table (Method, Path, Auth, Roles, Description).
2. Update or add the request/response JSON examples.
3. Ensure status codes are documented.
4. Verify pagination format is consistent.
5. Note any new query parameters.
6. Update RBAC role requirements if changed.

Format to follow:
```markdown
## Entity Endpoints

| Method | Path | Auth | Roles | Description |
|--------|------|------|-------|-------------|
| POST | `/entity` | Yes | Roles | Description |

#### POST /entity
```json
// Request
{ ... }

// Response 201
{ ... }
```
```

### Step 3: Update Data Model (docs/10-data-model.md)

When database schema changes:

1. Update the table DDL (CREATE TABLE statement).
2. Update the column list with types, constraints, and defaults.
3. Update the index list.
4. Update the Phase 1 vs Phase 2 table categorization if needed.
5. Verify foreign key relationships are documented.

### Step 4: Update Domain Model (docs/04-domain-model.md)

When domain entities change:

1. Update the entity field list.
2. Update entity relationships.
3. Update enum values if changed.
4. Update design decision notes.

### Step 5: Update Keycloak Configuration (docs/20-keycloak-configuration.md)

When Keycloak realm changes:

1. Document the specific configuration change.
2. Ensure `keycloak/realm-export.json` is updated (committed to repo).
3. Update protocol mapper documentation if claims change.
4. Update role or client configuration documentation.

### Step 6: Update Backlog Progress (docs/09-backlog.md)

When stories are completed:

1. Add completion status to the story (e.g., "Status: DONE").
2. Note any deviations from original acceptance criteria.
3. Note any discovered sub-tasks or follow-up work.

### Step 7: Update HANDOFF.md (Every Session)

At the end of every coding session, update `HANDOFF.md` with:

```markdown
# Handoff - [Date]

## Session Summary
Brief description of what was accomplished.

## Files Created
- `/path/to/file.go` -- description

## Files Modified
- `/path/to/file.go` -- what changed

## Decisions Made
- Decision 1: rationale
- Decision 2: rationale

## Current State
- What is working
- What is partially complete
- What is blocked

## Next Steps
1. Immediate next task
2. Follow-up items
3. Open questions

## Open Questions
- Question 1: context
- Question 2: context
```

### Step 8: Cross-Reference Validation

After updating docs, verify consistency:

- API design endpoint list matches implemented routes.
- Data model tables match migration files.
- Domain model entities match Go domain structs.
- Backlog story status matches implementation state.
- Roadmap sprint tasks match completed work.

## Outputs / Deliverables

1. **Updated documentation files** with changes applied.
2. **Change summary**: which docs were updated and why.
3. **Consistency report**: any remaining inconsistencies between code and docs.

## References

| Document | Path | Purpose |
|----------|------|---------|
| Requirements catalog | `docs/03-requirements-catalog.md` | New requirements |
| Domain model | `docs/04-domain-model.md` | Entity changes |
| Architecture | `docs/05-architecture-proposal.md` | Architecture decisions |
| MVP scope | `docs/07-mvp-scope.md` | Scope clarifications |
| Roadmap | `docs/08-roadmap.md` | Sprint progress |
| Backlog | `docs/09-backlog.md` | Story completion |
| Data model | `docs/10-data-model.md` | Schema changes |
| API design | `docs/11-api-design.md` | Endpoint changes |
| Offline sync | `docs/12-offline-sync-strategy.md` | Sync behavior |
| Security | `docs/13-security-and-compliance.md` | Security concerns |
| Keycloak config | `docs/20-keycloak-configuration.md` | IAM changes |

## Constraints / Quality Bar

- Documentation must be updated in the same PR as the code change (not deferred).
- API documentation must exactly match implementation (method, path, request/response shapes, status codes).
- Data model documentation must exactly match migration DDL.
- HANDOFF.md must be updated at every session end.
- All documentation in English (per project language rules).

## Interaction with Other Artifacts

- **Invoked by agents**: all agents at end of implementation tasks.
- **Triggered by skills**: design-backend-feature, design-frontend-feature, review-migration (when deviations found).
- **Feeds into skills**: analyze-requirements (reads updated docs for next story), assess-release-readiness (docs up-to-date check).
