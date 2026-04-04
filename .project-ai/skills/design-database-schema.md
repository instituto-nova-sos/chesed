# Skill: Design Database Schema

## Purpose

Design a database schema from requirements, producing table definitions, column specifications, constraints, indexes, foreign keys, and migration plan — before any migration code is written. This is the design counterpart to `review-migration` (which reviews existing migrations) and `add-database-table` playbook (which guides implementation).

## When to Use / Trigger

- During the DESIGN phase of feature delivery, when new tables or columns are needed.
- When a story introduces new domain entities that require persistence.
- When a user says "design schema for feature X" or "what tables do we need for Y".
- Before following the `add-database-table` playbook.

## Role / Expertise

Senior database architect with expertise in:
- PostgreSQL 16 schema design and optimization.
- Normalization and denormalization trade-offs for application databases.
- Index strategy for query-driven design.
- Multi-tenant data isolation via `campus_id` column filtering.
- UUID primary keys for distributed/offline-safe record creation.
- Soft delete patterns with `is_active` boolean flags.
- Migration planning with dependency ordering.

## Inputs

| Input | Required | Source |
|-------|----------|--------|
| Story analysis (from analyze-requirements) | Yes | Prior skill output |
| Domain model | Yes | `docs/04-domain-model.md` |
| Existing data model | Yes | `docs/10-data-model.md` |
| MVP scope (Phase validation) | Yes | `docs/07-mvp-scope.md` |
| API design (for query patterns) | Optional | `docs/11-api-design.md` |

## Process

### Step 1: Identify Entities Requiring Persistence

1. Read the story analysis and domain model.
2. List all entities that need database tables.
3. For each entity, determine if it is:
   - A new table (does not exist in `docs/10-data-model.md`).
   - An extension of an existing table (new columns).
   - A join table (many-to-many relationship).

### Step 2: Validate Against Phase Scope

1. Check `docs/07-mvp-scope.md` for allowed Phase 1 tables:
   `campus`, `person`, `address`, `person_role`, `assisted_profile`, `app_user`, `service_type`, `triage`, `triage_requested_service`, `attendance`, `attendance_transition`, `audit_log`.
2. If any proposed table is NOT in this list, STOP. It belongs to a later phase.
3. For column additions to existing tables, verify the column aligns with Phase 1 requirements.

### Step 3: Design Table Structure

For each table, define:

**Mandatory columns (all entity tables):**
- `id UUID PRIMARY KEY DEFAULT gen_random_uuid()` — UUID primary key.
- `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()` — Creation timestamp.
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()` — Last update timestamp.

**Mandatory columns (all operational tables):**
- `campus_id UUID NOT NULL REFERENCES campus(id)` — Multi-tenant isolation.
- `is_active BOOLEAN NOT NULL DEFAULT TRUE` — Soft delete flag.

**Entity-specific columns:**
- Map domain attributes to PostgreSQL types:
  - `string` → `VARCHAR(n)` or `TEXT`
  - `integer` → `INTEGER` or `BIGINT`
  - `boolean` → `BOOLEAN`
  - `date` → `DATE`
  - `datetime` → `TIMESTAMPTZ`
  - `enum` → `VARCHAR` with CHECK constraint or PostgreSQL ENUM type
  - `json` → `JSONB`
  - `uuid reference` → `UUID REFERENCES table(id)`

**Column constraints:**
- `NOT NULL` for required fields.
- `CHECK` constraints for enums and value ranges.
- `UNIQUE` constraints for natural keys (e.g., document_number within document_type).
- `DEFAULT` values where appropriate.

### Step 4: Define Foreign Key Relationships

For each relationship:
1. Identify the relationship type: one-to-many, many-to-many, one-to-one.
2. Define the foreign key column with appropriate naming: `{referenced_table}_id`.
3. Choose ON DELETE behavior:
   - `RESTRICT` (default) — prevent deletion of referenced record.
   - `CASCADE` — delete dependent records (rare, use with caution).
   - `SET NULL` — set FK to NULL on parent deletion (for optional relationships).
4. Document the relationship direction and cardinality.

### Step 5: Design Indexes

Design indexes based on query patterns from the API design:

**Mandatory indexes:**
- Primary key (automatic).
- All foreign key columns.
- `campus_id` on all operational tables (most queries filter by campus).

**Query-driven indexes:**
- Columns used in WHERE clauses frequently.
- Columns used in ORDER BY for list endpoints.
- Columns used in search/filter operations.
- Composite indexes for common multi-column filters.

**Index naming convention:** `idx_{table}_{column(s)}`

**Special indexes:**
- GIN index for JSONB columns used in queries.
- Trigram index for text search (`pg_trgm` extension) if search is supported.
- Partial index for `WHERE is_active = TRUE` if most queries filter active records.

### Step 6: Plan Migration Sequence

Order migrations by dependency:

1. Tables with no foreign keys first (e.g., `campus`).
2. Tables referenced by others next (e.g., `person`, `service_type`).
3. Tables with foreign keys last (e.g., `triage`, `attendance`).
4. Join tables after both sides exist (e.g., `triage_requested_service`).

For each migration:
- Assign sequential number: `NNNNNN_description.up.sql` / `NNNNNN_description.down.sql`.
- The `.up.sql` creates the table/column/index.
- The `.down.sql` reverses the change exactly (DROP TABLE, DROP COLUMN, DROP INDEX).
- Verify `.down.sql` is safe: dropping a table with data should be acceptable in development.

### Step 7: Define Seed Data

Identify tables that require seed data:
- `campus` — at least one default campus for development.
- `service_type` — fixed set: LEGAL, MEDICAL, NUTRITIONAL, PHYSIOTHERAPY, SOCIAL, EDUCATIONAL, PSYCHOLOGICAL, OTHER.
- Other reference data required for the application to function.

Seed data goes in a separate migration or seed script, not in the table creation migration.

### Step 8: Update Data Model Documentation

Write the complete table DDL in the format used by `docs/10-data-model.md`:

```sql
CREATE TABLE table_name (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- entity columns
    campus_id UUID NOT NULL REFERENCES campus(id),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_table_name_campus_id ON table_name(campus_id);
-- additional indexes
```

## Outputs / Deliverables

1. **Table definitions** — Complete DDL for each new table with all columns, types, and constraints.
2. **Foreign key map** — All relationships with cardinality and ON DELETE behavior.
3. **Index definitions** — All indexes with rationale (query pattern served).
4. **Migration sequence** — Ordered list of migration files with dependencies.
5. **Seed data specification** — Required reference data for each table.
6. **Updates to `docs/10-data-model.md`** — Complete DDL added to the data model document.

## References

| Document | Path | Usage |
|----------|------|-------|
| Domain model | `docs/04-domain-model.md` | Entity definitions and relationships |
| Data model | `docs/10-data-model.md` | Existing schema and conventions |
| MVP scope | `docs/07-mvp-scope.md` | Phase 1 table list validation |
| API design | `docs/11-api-design.md` | Query patterns for index design |
| Implementation guidelines | `docs/15-implementation-guidelines.md` | Naming conventions |

## Constraints / Quality Bar

- All tables must have UUID primary keys (`gen_random_uuid()`).
- All operational tables must have `campus_id`, `is_active`, `created_at`, `updated_at`.
- All foreign keys must have corresponding indexes.
- All timestamps must use `TIMESTAMPTZ` (not `TIMESTAMP`).
- Table names must be valid Phase 1 tables per `docs/07-mvp-scope.md`.
- Every `.up.sql` must have a matching `.down.sql` that fully reverses the change.
- No tables may allow UPDATE or DELETE on `audit_log`.
- Column names use snake_case.
- Enum values use UPPERCASE.

## Interaction with Other Artifacts

- **Invoked by agents**: backend-engineer (primary), tech-lead (design review).
- **Depends on skills**: analyze-requirements (provides story analysis).
- **Feeds into playbooks**: add-database-table (uses schema design as input).
- **Triggers hooks**: pre-migration (validates before creating migration files).
- **Reviewed by skills**: review-migration (validates the resulting migration code).
- **Governed by rules**: phase-boundary (table must be in Phase 1 list), documentation-first (DDL documented before implementation).
