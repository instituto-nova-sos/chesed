# Skill: Review Migration

## Purpose

Review database migration files for correctness, completeness, and conformance with the project's data model documentation. Ensures migrations are safe, reversible, and consistent with the documented schema.

## When to Use / Trigger

- When new migration files are added under `backend/migrations/`.
- When a user says "review migration" or "check the database migration".
- After adding or modifying any database table.
- Before running migrations in a shared environment.

## Role / Expertise

Database engineer familiar with:
- PostgreSQL 16 DDL and constraints.
- golang-migrate SQL migration format.
- The Chesed data model from `docs/10-data-model.md`.
- Safe migration practices (no data loss, reversible changes).

## Inputs

| Input | Required | Source |
|-------|----------|--------|
| Migration files (up.sql and down.sql) | Yes | `backend/migrations/` |
| Data model documentation | Yes | `docs/10-data-model.md` |
| Phase 1 table list | Yes | `docs/07-mvp-scope.md`, `docs/08-roadmap.md` |

## Process

### Step 1: File Structure Validation

- [ ] Both `.up.sql` and `.down.sql` files exist for each migration.
- [ ] File naming follows golang-migrate convention: `NNNNNN_description.up.sql` / `NNNNNN_description.down.sql`.
- [ ] Migration numbers are sequential with no gaps.
- [ ] Files are located in `backend/migrations/`.

### Step 2: Phase Boundary Validation

Phase 1 allowed tables:
- `campus`
- `person`
- `address`
- `person_role`
- `assisted_profile`
- `app_user`
- `service_type`
- `triage`
- `triage_requested_service`
- `attendance`
- `attendance_transition`
- `audit_log`

Phase 2 tables (must NOT appear in Phase 1 migrations):
- `campaign`
- `campaign_team`
- `document`
- `consent`
- `donation`

- [ ] Migration does not create Phase 2 tables during Phase 1.
- [ ] Migration does not reference Phase 2 tables in foreign keys.

### Step 3: Schema Conformance

For each table in the migration, compare against `docs/10-data-model.md`:

- [ ] Table name matches documentation.
- [ ] All columns present with correct types.
- [ ] `id UUID PRIMARY KEY DEFAULT gen_random_uuid()` on all tables.
- [ ] `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()` present.
- [ ] `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()` present.
- [ ] `is_active BOOLEAN NOT NULL DEFAULT TRUE` on applicable tables.
- [ ] `campus_id UUID NOT NULL REFERENCES campus(id)` on all operational tables.
- [ ] CHECK constraints match documented values (e.g., role_type IN, status IN, gender IN).
- [ ] Foreign key constraints match relationships in `docs/04-domain-model.md`.
- [ ] UNIQUE constraints match documentation.
- [ ] Column types match: VARCHAR lengths, TEXT vs VARCHAR, nullable vs NOT NULL.

### Step 4: Index Validation

- [ ] Indexes on all foreign key columns.
- [ ] Indexes on frequently filtered columns (campus_id, status, date fields).
- [ ] Composite indexes where documented (e.g., `document_type + document_number`).
- [ ] GIN index on `search_vector` for full-text search.
- [ ] Partial indexes where appropriate (e.g., `WHERE sync_id IS NOT NULL`).

Cross-reference with documented indexes in `docs/10-data-model.md`:
- `idx_person_campus ON person(campus_id)`
- `idx_person_name ON person(full_name)`
- `idx_person_document ON person(document_type, document_number)`
- `idx_person_search ON person USING GIN(search_vector)`
- `idx_address_person ON address(person_id)`
- `idx_person_role_person ON person_role(person_id)`
- `idx_person_role_type ON person_role(role_type)`
- `idx_user_email ON app_user(email)`
- `idx_user_campus ON app_user(campus_id)`
- `idx_user_keycloak_sub ON app_user(keycloak_subject_id)`
- `idx_triage_person ON triage(person_id)`
- `idx_triage_campus ON triage(campus_id)`
- `idx_triage_date ON triage(triage_date)`
- `idx_triage_sync ON triage(sync_id) WHERE sync_id IS NOT NULL`
- `idx_attendance_person ON attendance(person_id)`
- `idx_attendance_professional ON attendance(professional_id)`
- `idx_attendance_campus ON attendance(campus_id)`
- `idx_attendance_status ON attendance(status)`
- `idx_attendance_date ON attendance(attendance_date)`
- `idx_attendance_sync ON attendance(sync_id) WHERE sync_id IS NOT NULL`
- `idx_transition_attendance ON attendance_transition(attendance_id)`

### Step 5: Down Migration Validation

- [ ] Down migration reverses the up migration completely.
- [ ] Tables dropped in reverse order of creation (to respect foreign keys).
- [ ] Indexes dropped before tables (or cascaded).
- [ ] Seed data removed if inserted in up migration.
- [ ] Down migration is safe to run (does not drop tables that may have data in production unless explicitly intended).

### Step 6: Audit Log Protection

- [ ] Audit log table is created as append-only (no UPDATE or DELETE triggers).
- [ ] No migration modifies the audit_log schema to allow updates or deletes.
- [ ] Audit log columns match: id, entity_type, entity_id, action, old_values (JSONB), new_values (JSONB), performed_by, campus_id, ip_address, user_agent, created_at.

### Step 7: Seed Data Validation

If the migration includes seed data:
- [ ] Service types seeded: LEGAL, MEDICAL, NUTRITIONAL, PHYSIOTHERAPY, SOCIAL, EDUCATIONAL, PSYCHOLOGICAL, OTHER.
- [ ] Seed data uses deterministic UUIDs (for idempotency) or INSERT ... ON CONFLICT DO NOTHING.
- [ ] Default campus created if applicable.

### Step 8: Safety Checks

- [ ] No `DROP TABLE` without `IF EXISTS`.
- [ ] No `ALTER TABLE ... DROP COLUMN` on tables with production data (Phase 1 can be lenient if pre-production).
- [ ] No `TRUNCATE` on any table.
- [ ] `ON DELETE CASCADE` used judiciously (only on child tables like address, person_role, triage_requested_service, attendance_transition).
- [ ] No destructive changes to the audit_log table.

## Outputs / Deliverables

A migration review report with:

1. **File inventory**: List of migration files reviewed.
2. **Schema conformance**: Per-table pass/fail against documentation.
3. **Index coverage**: Missing or extra indexes.
4. **Down migration**: Reversibility assessment.
5. **Issues** (per file):
   - Severity: BLOCKER / MAJOR / MINOR.
   - Description: what is wrong.
   - Fix: specific correction.
6. **Verdict**: APPROVE / REQUEST_CHANGES.

## References

| Document | Path | Usage |
|----------|------|-------|
| Data model | `docs/10-data-model.md` | Source of truth for table schemas |
| Domain model | `docs/04-domain-model.md` | Entity relationships |
| MVP scope | `docs/07-mvp-scope.md` | Phase 1 table list |
| Roadmap | `docs/08-roadmap.md` | Phase boundaries |

## Constraints / Quality Bar

- Every migration must have both up and down files.
- Schema must match `docs/10-data-model.md` exactly.
- Phase 2 tables in Phase 1 migrations is a BLOCKER.
- Missing UUID primary key is a BLOCKER.
- Missing campus_id on operational table is a BLOCKER.
- Missing timestamps (created_at, updated_at) is a MAJOR issue.
- Missing indexes on foreign keys is a MAJOR issue.
- Audit log modifications are a BLOCKER.

## Interaction with Other Artifacts

- **Invoked by agents**: backend-engineer (before running migrations), tech-lead (review gate).
- **Triggers skills**: maintain-docs (if migration diverges from docs, update `docs/10-data-model.md`).
- **Blocks**: assess-release-readiness (migration issues block release).
