# Skill: Prepare Handoff

## Purpose

Prepare a session handoff document that captures everything needed for the next agent or human session to continue work without context loss. Updates HANDOFF.md with what was done, files created/modified, decisions made, next steps, and open questions. Verifies git log matches documented changes.

## When to Use / Trigger

- At the end of every coding session.
- When a user says "prepare handoff", "session end", or "wrap up".
- Before switching to a different feature or sprint.

## Role / Expertise

Project coordinator who understands the full project context and can summarize work clearly for continuity.

## Inputs

| Input | Required | Source |
|-------|----------|--------|
| Git log since session start | Yes | `git log --oneline` |
| Git diff summary | Yes | `git diff --stat` or review of committed changes |
| Session context (what was requested/accomplished) | Yes | Conversation history |
| Current state of HANDOFF.md | Optional | `HANDOFF.md` in project root |

## Process

### Step 1: Gather Session Changes

Run the following to understand what changed:

```bash
# Recent commits (this session)
git log --oneline -20

# Files changed (uncommitted)
git status

# Diff of uncommitted changes
git diff --stat
```

### Step 2: Build Files Inventory

For each file created or modified during the session:

```markdown
## Files Created
- `backend/internal/domain/person.go` -- Person domain struct with JSON and validation tags
- `backend/internal/service/person_service.go` -- Person service with CRUD business logic
- `backend/migrations/000002_create_person.up.sql` -- Person table migration

## Files Modified
- `backend/internal/handler/routes.go` -- Added person routes with RBAC middleware
- `docs/11-api-design.md` -- Updated person endpoint response examples
```

### Step 3: Document Decisions

Capture any architectural or design decisions made during the session:

```markdown
## Decisions Made
- Used tsvector for person name search instead of LIKE queries (performance at scale)
- Person duplicate detection is cross-campus by design (document uniqueness is global)
- Chose to keep address as separate table rather than embedding in person (normalization + history)
```

Include rationale for each decision so the next session understands the "why".

### Step 4: Assess Current State

Describe what is working, partially complete, and blocked:

```markdown
## Current State
### Working
- Person CRUD API (create, read, update, list) with tests passing
- RBAC middleware enforcing role-based access
- Campus isolation on all person queries

### Partially Complete
- Person search: basic query working, tsvector index not yet populated
- Frontend person form: layout done, validation not connected

### Blocked
- User auto-provisioning: waiting for Keycloak realm configuration
```

### Step 5: Define Next Steps

Prioritized list of what to do next:

```markdown
## Next Steps
1. **Immediate**: Complete person search with tsvector population trigger
2. **Next**: Implement person role management API (S03.6)
3. **Follow-up**: Connect frontend form validation to Zod schemas
4. **Deferred**: Optimize person list query for large datasets
```

### Step 6: List Open Questions

Any unresolved questions that need answers:

```markdown
## Open Questions
1. Should duplicate detection on person creation be blocking or advisory? (Currently advisory -- shows warning but allows save)
2. How should we handle the case where a person is created offline on two devices with the same CPF? (Sync conflict or merge?)
3. Do volunteers need to see assisted profiles or just basic person data?
```

### Step 7: Verify Git Consistency

Cross-reference the documented changes with git:

```bash
# Verify all documented files were committed
git log --name-only --oneline -5

# Check for uncommitted changes that should be documented
git status
```

- [ ] All created files listed in HANDOFF.md are committed.
- [ ] All modified files listed in HANDOFF.md reflect actual changes.
- [ ] No uncommitted work left undocumented.
- [ ] Commit messages follow convention: `<type>: <description>`.

### Step 8: Write HANDOFF.md

Location: `HANDOFF.md` in project root.

```markdown
# Handoff - YYYY-MM-DD

## Session Summary
[1-3 sentences describing what was accomplished]

## Sprint Context
Sprint [N]: [Sprint name] -- [stories worked on]

## Files Created
- [list with descriptions]

## Files Modified
- [list with what changed]

## Decisions Made
- [list with rationale]

## Current State
### Working
- [list]

### Partially Complete
- [list]

### Blocked
- [list with reason]

## Test Status
- Backend: [pass/fail count]
- Frontend: [pass/fail count]
- Lint: [clean/warnings]

## Next Steps
1. [prioritized list]

## Open Questions
1. [list with context]

## References
- Stories: [S03.1, S03.2, ...]
- Commits: [short SHAs]
```

### Step 9: Update tasks/lessons.md (if applicable)

If any mistakes were made during the session that should be captured as lessons:

```markdown
# Lesson: [date]
## Pattern: [what went wrong]
## Rule: [how to prevent it next time]
## Context: [project-specific details]
```

## Outputs / Deliverables

1. **Updated HANDOFF.md** in project root.
2. **Updated tasks/lessons.md** (if corrections were made during session).
3. **Verification report**: git log matches documented changes.

## References

| Document | Path | Usage |
|----------|------|-------|
| Roadmap | `docs/08-roadmap.md` | Sprint context |
| Backlog | `docs/09-backlog.md` | Story references |
| HANDOFF.md | `HANDOFF.md` | Previous session state |

## Constraints / Quality Bar

- HANDOFF.md must be updated at every session end (non-negotiable).
- All files listed in HANDOFF.md must exist in the repo.
- Git log must corroborate documented changes.
- Next steps must be actionable (not vague).
- Open questions must include enough context for someone without session history.
- Commit messages must follow convention: `<type>: <short description>`.

## Interaction with Other Artifacts

- **Invoked by agents**: all agents at session end.
- **Depends on**: all work done during the session.
- **Feeds into**: next session's context (the new agent reads HANDOFF.md first).
- **Complements**: maintain-docs skill (HANDOFF covers session state; maintain-docs covers specification docs).
