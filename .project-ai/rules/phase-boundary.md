# Rule: Phase Boundary Enforcement

## Purpose

Prevent scope creep by strictly enforcing the boundary between Phase 1 (MVP) and Phase 2. Ensure no Phase 2 features, tables, or states are implemented during Phase 1 development.

## Rule Statement

Before implementing any feature, check `docs/07-mvp-scope.md`. Do NOT create Phase 2 database tables, implement Phase 2 states, or build Phase 2 UI features. Phase 1 scope is fixed and non-negotiable.

## Trigger Condition

Every time the AI agent begins implementing a feature, creating a database migration, or adding a new UI page.

## Enforcement

### Phase 1 Database Tables (Exactly 12)

The following tables — and ONLY these tables — may be created in Phase 1:

1. `campus`
2. `person`
3. `address`
4. `person_role`
5. `assisted_profile`
6. `app_user`
7. `service_type`
8. `triage`
9. `triage_requested_service`
10. `attendance`
11. `attendance_transition`
12. `audit_log`

### Explicitly Forbidden in Phase 1

**Phase 2 tables — Do NOT create:**
- `campaign`
- `campaign_team`
- `document`
- `consent`
- `donation`

**Phase 2 features — Do NOT implement:**
- `FOLLOW_UP` state in triage or attendance workflows
- Service type admin UI (service types are fixed seed data in MVP)
- Multi-campus user assignment (each person/user belongs to exactly one campus)
- Campaign management
- Document management
- Donation tracking
- Consent management
- Advanced reporting dashboards (Recharts is Phase 2)

### Verification Checklist

Before starting any implementation:

1. Open `docs/07-mvp-scope.md` and check the "Won't Have in MVP" section.
2. If the feature appears in that section, STOP immediately.
3. If creating a new table, verify it is one of the 12 listed above.
4. If adding a new state to a workflow, verify it is documented as a Phase 1 state.
5. If adding a new UI page, verify the feature it supports is in Phase 1 scope.

### Gray Area Resolution

If uncertain whether a feature belongs in Phase 1 or Phase 2:
- Check `docs/03-requirements-catalog.md` for the requirement's phase tag.
- Check `docs/09-backlog.md` for the story's sprint assignment.
- If still unclear, default to NOT implementing it and flag the ambiguity.

## Enforcement Mechanism

- The `pre-implement` hook checks Phase 1 scope automatically.
- The `pre-migration` hook verifies tables against the Phase 1 list.
- The AI agent must refuse to implement any feature identified as Phase 2, even if explicitly asked, and must explain why.
- If the user insists on a Phase 2 feature, the agent must document the scope deviation and its implications before proceeding.

## References

- `docs/07-mvp-scope.md` — MVP scope definition and exclusions
- `docs/03-requirements-catalog.md` — Requirements with phase assignments
- `docs/08-roadmap.md` — Sprint-to-phase mapping
- `docs/09-backlog.md` — Story definitions
- `docs/10-data-model.md` — Database schema (Phase 1 tables only)

## Consequences of Skipping

- Phase 2 tables created prematurely may constrain future design decisions when Phase 2 requirements are fully analyzed.
- Implementing Phase 2 features diverts effort from completing Phase 1, delaying the MVP launch.
- Partial Phase 2 implementations create maintenance burden — half-built features that must be maintained but deliver no value.
- Scope creep is the single largest risk to MVP delivery timelines.
