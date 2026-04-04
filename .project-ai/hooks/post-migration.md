# Hook: Post-Migration Validation

## Purpose

Verify that completed database migrations are correct, reversible, and consistent with domain structs and documentation.

## Trigger Condition

After creating or modifying any SQL migration file in `backend/migrations/`.

## Status

**Non-blocking** — But mandatory before marking the work as complete. Do not close a story or submit for review until all steps pass.

## Steps

1. **Run migration review**
   - Execute the `review-migration` skill against the new migration files.
   - Check for:
     - SQL syntax correctness
     - Proper use of `IF NOT EXISTS` / `IF EXISTS` guards where appropriate
     - Correct data types (UUID, TIMESTAMPTZ, BOOLEAN, TEXT, JSONB)
     - Foreign key constraints with proper ON DELETE behavior
     - Index creation on foreign keys and frequently filtered columns
     - `campus_id` column present on all operational tables

2. **Verify down.sql reverses up.sql**
   - Read both files side by side.
   - Confirm every CREATE in up.sql has a corresponding DROP in down.sql.
   - Confirm every ALTER TABLE ADD COLUMN has a corresponding ALTER TABLE DROP COLUMN.
   - Confirm every CREATE INDEX has a corresponding DROP INDEX.
   - Confirm every CREATE TYPE (ENUM) has a corresponding DROP TYPE.
   - Verify the down.sql handles foreign key dependencies in the correct drop order.
   - Mental test: "If I run up then down, is the database in the exact same state as before?"

3. **Verify domain structs match schema**
   - For each table in the migration, locate the corresponding Go struct in `backend/internal/domain/`.
   - Verify field-by-field alignment:
     - Column name (snake_case) maps to struct field (PascalCase) with correct `db` tag.
     - Column type maps to correct Go type:
       - `UUID` -> `uuid.UUID`
       - `TEXT` / `VARCHAR` -> `string`
       - `BOOLEAN` -> `bool`
       - `TIMESTAMPTZ` -> `time.Time`
       - `INTEGER` -> `int` or `int32`
       - `JSONB` -> `json.RawMessage` or typed struct
       - `ENUM` -> custom string type with constants
     - Nullable columns use pointer types or `pgtype` equivalents.
   - If domain structs do not exist yet, flag them for creation.

4. **Update docs/10-data-model.md if deviations found**
   - If the migration deviates from the documented schema (e.g., additional index, modified constraint, changed default):
     - If intentional: update `docs/10-data-model.md` to match the migration.
     - If unintentional: fix the migration to match the documentation.
   - Verify the ER diagram or table listing in the docs reflects the current state.

## Enforcement Mechanism

- The AI agent must execute this hook after completing any migration file creation.
- All four steps must pass before the story can be marked as complete.
- Domain struct mismatches must be resolved immediately — either create the struct or fix the migration.

## References

- `docs/10-data-model.md` — Database schema documentation
- `docs/04-domain-model.md` — Domain model and relationships
- `backend/internal/domain/` — Go domain struct definitions
- `docs/15-implementation-guidelines.md` — Coding standards for database layer

## Consequences of Skipping

- Irreversible migrations create deployment risk; failed rollbacks can cause production incidents.
- Domain struct mismatches cause runtime panics or silent data corruption when scanning query results.
- Documentation drift makes `docs/10-data-model.md` unreliable for future development.
- Missing indexes cause performance degradation as data grows.
