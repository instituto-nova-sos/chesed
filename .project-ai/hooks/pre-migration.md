# Hook: Pre-Migration Validation

## Purpose

Validate that database migrations are documented, within Phase 1 scope, and properly structured before creation. Prevents undocumented schema changes and Phase 2 table creation.

## Trigger Condition

Before creating any new SQL migration file in `backend/migrations/`.

## Status

**Blocking** — Do not create migration files if any gate fails.

## Steps

1. **Verify table is documented in data model**
   - Open `docs/10-data-model.md` and confirm the table (or column change) is documented.
   - Verify column names, types, constraints, and indexes match the documentation.
   - If the table or column is NOT documented, STOP. Update `docs/10-data-model.md` first with:
     - Table name and purpose
     - All columns with types, nullability, and defaults
     - Primary key and foreign key constraints
     - Indexes
     - Any CHECK constraints or ENUMs

2. **Verify table is in Phase 1 list**
   - Phase 1 tables (exactly 12):
     `campus`, `person`, `address`, `person_role`, `assisted_profile`, `app_user`, `service_type`, `triage`, `triage_requested_service`, `attendance`, `attendance_transition`, `audit_log`.
   - If creating a table not in this list, STOP. It belongs to Phase 2 or later.
   - Cross-reference with `docs/07-mvp-scope.md` to confirm the table supports an in-scope feature.
   - Explicitly forbidden Phase 2 tables: `campaign`, `campaign_team`, `document`, `consent`, `donation`.

3. **If not documented, stop and update docs first**
   - Add the complete table definition to `docs/10-data-model.md`.
   - Include the rationale for the schema design.
   - Verify the schema follows project conventions:
     - UUID primary keys (`id UUID PRIMARY KEY DEFAULT gen_random_uuid()`)
     - `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
     - `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`
     - `is_active BOOLEAN NOT NULL DEFAULT true` (for soft deletes)
     - `campus_id UUID NOT NULL REFERENCES campus(id)` (on all operational tables)
   - Get documentation reviewed before proceeding.

4. **Verify migration numbering**
   - Check existing migration files in `backend/migrations/`.
   - New migration number must be the next sequential number.
   - Format: `NNNNNN_descriptive_name.up.sql` and `NNNNNN_descriptive_name.down.sql`.
   - The descriptive name must clearly indicate the change (e.g., `000003_create_person_table`).

5. **Plan both up.sql and down.sql**
   - Before writing, outline both files:
     - `up.sql`: The forward migration (CREATE TABLE, ALTER TABLE, CREATE INDEX, etc.)
     - `down.sql`: The exact reversal (DROP TABLE, DROP INDEX, ALTER TABLE DROP COLUMN, etc.)
   - Verify the down migration will cleanly reverse the up migration.
   - For tables with foreign key dependencies, plan the creation/drop order.
   - For ENUM types: create in up, drop in down (with CASCADE if needed).

## Enforcement Mechanism

- The AI agent must execute this hook before creating any file in `backend/migrations/`.
- If step 1 or 2 produces a STOP condition, the agent must update documentation first and re-run this hook.
- The agent must never create an up.sql without a corresponding down.sql.

## References

- `docs/10-data-model.md` — Database schema documentation (source of truth)
- `docs/07-mvp-scope.md` — Phase 1 scope and exclusions
- `docs/04-domain-model.md` — Domain model and relationships
- `docs/15-implementation-guidelines.md` — Migration conventions

## Consequences of Skipping

- Undocumented schema changes make `docs/10-data-model.md` unreliable and cause confusion during development.
- Creating Phase 2 tables prematurely adds maintenance burden and may constrain future design decisions.
- Missing down migrations make rollbacks impossible, creating deployment risk.
- Incorrect migration numbering causes `golang-migrate` to fail or apply migrations out of order.
