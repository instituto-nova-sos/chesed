# Hook: Pre-Implementation Gate Check

## Purpose

Validate that all prerequisites are met before starting any feature implementation. This prevents wasted effort on undocumented, out-of-scope, or out-of-phase work.

## Trigger Condition

Before writing any implementation code (backend handler, service, repository, frontend page, component, or database migration) for a new feature or story.

## Status

**Blocking** — Do not proceed with implementation if any gate fails.

## Steps

1. **Verify story exists in backlog**
   - Open `docs/09-backlog.md` and confirm the story ID exists.
   - If the story is not listed, STOP. Add the story to the backlog first with proper acceptance criteria.

2. **Verify feature is in Phase 1 scope**
   - Open `docs/07-mvp-scope.md` and confirm the feature is NOT in the "Won't Have in MVP" list.
   - Cross-reference with `docs/03-requirements-catalog.md` to confirm the requirement exists.
   - If the feature is Phase 2 or beyond, STOP. Do not implement.

3. **Check sprint assignment**
   - Open `docs/08-roadmap.md` and confirm which sprint the story belongs to.
   - If implementing out of sprint order, flag the dependency risk and get explicit approval.
   - Sprint order: Sprint 1 (Auth & Infra) -> Sprint 2 (Person Mgmt) -> Sprint 3 (Triage & Attendance) -> Sprint 4 (Sync, Reports & Polish).

4. **Check dependencies are met**
   - Verify that prerequisite stories from earlier sprints are implemented and tested.
   - For backend features: confirm the database tables exist (migrations applied).
   - For frontend features: confirm the API endpoints exist and are documented.
   - For API features: confirm the domain structs and repository layer exist.

5. **Run requirements analysis**
   - Execute the `analyze-requirements` skill against the story.
   - Confirm acceptance criteria are clear and testable.
   - Identify which layers are affected: domain, repository, service, handler, frontend.

6. **Flag security-sensitive changes**
   - If the story touches any of these areas, flag for security review:
     - Authentication or authorization middleware
     - PII fields (full_name, document_number, email, phone, birth_date, address, assisted_profile)
     - Keycloak configuration
     - Audit logging
     - IndexedDB or sync logic
   - Reference: `docs/13-security-and-compliance.md`, `docs/18-threat-model.md`.

7. **Assess complexity expectations**
   - Before writing code, estimate the complexity of the planned change.
   - Reference thresholds from `docs/quality/complexity-guidelines.md`:
     - Go: cognitive complexity ≤ 25 per function, cyclomatic ≤ 10, function length ≤ 40 lines.
     - TypeScript: cognitive complexity ≤ 15 per function, cyclomatic ≤ 10, function length ≤ 50 lines.
   - If the feature will likely require functions exceeding these thresholds, plan the decomposition upfront.
   - Identify which existing files may exceed file length limits (Go: 400, TS: 300) after the change, and plan extraction.

8. **Verify new tables are in Phase 1 list**
   - If the story requires new database tables, confirm they are in the Phase 1 list:
     `campus`, `person`, `address`, `person_role`, `assisted_profile`, `app_user`, `service_type`, `triage`, `triage_requested_service`, `attendance`, `attendance_transition`, `audit_log`.
   - If a table is not in this list, STOP. It belongs to a later phase.

## Enforcement Mechanism

- The AI agent must execute this hook automatically before any `feat:` or `refactor:` implementation work.
- If any step produces a STOP condition, the agent must report which gate failed and what action is needed before proceeding.
- The agent must not bypass this hook by splitting work into smaller tasks that individually appear non-blocking.

## References

- `docs/03-requirements-catalog.md` — Full requirements list
- `docs/07-mvp-scope.md` — Phase 1 scope and exclusions
- `docs/08-roadmap.md` — Sprint assignments
- `docs/09-backlog.md` — Story definitions and acceptance criteria
- `docs/10-data-model.md` — Database schema documentation
- `docs/13-security-and-compliance.md` — Security requirements

## Consequences of Skipping

- Implementing undocumented features leads to scope creep and untraceable requirements.
- Implementing Phase 2 features wastes effort and creates maintenance burden.
- Missing dependency checks cause integration failures and rework.
- Skipping security flagging may introduce vulnerabilities in PII handling or authentication.
