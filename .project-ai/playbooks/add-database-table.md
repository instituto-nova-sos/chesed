# Playbook: Add Database Table

## Purpose

End-to-end guide for adding a new database table to the Chesed PostgreSQL schema, including migrations, domain structs, and repository implementation.

---

## Prerequisites

- The table is in the Phase 1 list (`docs/07-mvp-scope.md`): campus, person, address, person_role, assisted_profile, app_user, service_type, triage, triage_requested_service, attendance, attendance_transition, audit_log
- If the table is NOT in the Phase 1 list, STOP. Do not create Phase 2 tables prematurely.

---

## Steps

### Step 1: Verify Phase Membership

Open `docs/07-mvp-scope.md` and confirm the table is listed under "Phase 1 Tables (MVP)".

Phase 1 tables:
- `campus`, `person`, `address`, `person_role`, `assisted_profile`
- `app_user`, `service_type`
- `triage`, `triage_requested_service`
- `attendance`, `attendance_transition`
- `audit_log`

Phase 2 tables (DO NOT create in Phase 1):
- `campaign`, `campaign_team`, `document`, `consent`, `donation`

If the table is Phase 2, stop here and document the deferral reason.

### Step 2: Verify or Create DDL in Data Model Doc

Open `docs/10-data-model.md` and verify the table DDL exists with:

- UUID primary key: `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
- Timestamps: `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`, `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- Soft delete: `is_active BOOLEAN NOT NULL DEFAULT TRUE`
- Campus scoping: `campus_id UUID NOT NULL REFERENCES campus(id)` (on all operational tables)
- CHECK constraints for enum-like columns
- Foreign key references with appropriate ON DELETE behavior
- Indexes on all foreign keys and frequently filtered columns

If the DDL is missing or incomplete, update `docs/10-data-model.md` first (documentation-first workflow).

### Step 3: Create Up Migration

File: `backend/migrations/NNNNNN_create_<table_name>.up.sql`

Determine the next migration number by checking existing files in `backend/migrations/`. Use a sequential 6-digit number (e.g., `000001`, `000002`).

Template:

```sql
-- Migration: Create <table_name> table
-- Phase 1 table per docs/10-data-model.md

CREATE TABLE <table_name> (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Domain-specific columns
    <column_name>   <TYPE> [NOT NULL] [DEFAULT <value>] [CHECK (...)],

    -- Foreign keys
    campus_id       UUID NOT NULL REFERENCES campus(id),
    <fk_column>     UUID [NOT NULL] REFERENCES <parent_table>(id),

    -- Standard columns
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes on foreign keys
CREATE INDEX idx_<table>_campus ON <table_name>(campus_id);
CREATE INDEX idx_<table>_<fk> ON <table_name>(<fk_column>);

-- Additional indexes for common query patterns
CREATE INDEX idx_<table>_<column> ON <table_name>(<column>) WHERE is_active = TRUE;

-- Trigger for updated_at (if using trigger approach)
CREATE TRIGGER set_<table>_updated_at
    BEFORE UPDATE ON <table_name>
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

Concrete example for the `person` table:

```sql
CREATE TABLE person (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name         VARCHAR(200) NOT NULL,
    birth_date        DATE,
    document_type     VARCHAR(20) NOT NULL DEFAULT 'CPF'
                      CHECK (document_type IN ('CPF', 'SSN', 'EU_ID', 'PASSPORT', 'OTHER')),
    document_number   VARCHAR(30),
    gender            VARCHAR(20) CHECK (gender IN ('M', 'F', 'OTHER', 'PREFER_NOT_TO_SAY')),
    email             VARCHAR(255),
    phone             VARCHAR(30),
    photo_url         VARCHAR(500),
    referral_source   VARCHAR(200),
    campus_id         UUID NOT NULL REFERENCES campus(id),
    is_active         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by        UUID,
    search_vector     TSVECTOR
);

CREATE INDEX idx_person_campus ON person(campus_id);
CREATE INDEX idx_person_document ON person(document_type, document_number) WHERE is_active = TRUE;
CREATE INDEX idx_person_search ON person USING GIN(search_vector);
CREATE INDEX idx_person_full_name ON person(full_name) WHERE is_active = TRUE;
```

Rules for the up migration:
- Always use `gen_random_uuid()` for UUID defaults (PostgreSQL 16 built-in)
- Always use `TIMESTAMPTZ` (never `TIMESTAMP`)
- All VARCHAR columns must have explicit length limits
- CHECK constraints for enum-like columns (match values in `docs/10-data-model.md`)
- Foreign keys must reference existing tables (ensure migration ordering)
- Create indexes on all foreign key columns
- Create filtered indexes with `WHERE is_active = TRUE` for commonly queried columns

### Step 4: Create Down Migration

File: `backend/migrations/NNNNNN_create_<table_name>.down.sql`

```sql
-- Rollback: Drop <table_name> table

DROP TABLE IF EXISTS <table_name>;
```

If the up migration created indexes explicitly (outside the table definition), drop them too, though `DROP TABLE` cascades to dependent indexes:

```sql
DROP INDEX IF EXISTS idx_<table>_campus;
DROP INDEX IF EXISTS idx_<table>_<fk>;
DROP TABLE IF EXISTS <table_name>;
```

If this table is referenced by other tables via foreign keys, the down migration must account for dependency order. Drop dependent tables first or use `CASCADE` (with caution).

### Step 5: Create Domain Struct

File: `backend/internal/domain/<entity>.go`

Map the SQL table to a Go struct:

| SQL Type | Go Type |
|----------|---------|
| `UUID` | `uuid.UUID` |
| `VARCHAR(N)` | `string` (required) or `*string` (nullable) |
| `TEXT` | `string` or `*string` |
| `BOOLEAN` | `bool` |
| `DATE` | `*time.Time` (nullable) or `time.Time` |
| `TIMESTAMPTZ` | `time.Time` |
| `INTEGER` | `int` or `int32` |
| `JSONB` | `json.RawMessage` or a typed struct |

```go
package domain

import (
    "time"
    "github.com/google/uuid"
)

type Person struct {
    ID             uuid.UUID  `json:"id" db:"id"`
    FullName       string     `json:"full_name" db:"full_name" validate:"required,max=200"`
    BirthDate      *time.Time `json:"birth_date,omitempty" db:"birth_date"`
    DocumentType   string     `json:"document_type" db:"document_type" validate:"required,oneof=CPF SSN EU_ID PASSPORT OTHER"`
    DocumentNumber *string    `json:"document_number,omitempty" db:"document_number" validate:"omitempty,max=30"`
    Gender         *string    `json:"gender,omitempty" db:"gender" validate:"omitempty,oneof=M F OTHER PREFER_NOT_TO_SAY"`
    Email          *string    `json:"email,omitempty" db:"email" validate:"omitempty,email,max=255"`
    Phone          *string    `json:"phone,omitempty" db:"phone" validate:"omitempty,max=30"`
    PhotoURL       *string    `json:"photo_url,omitempty" db:"photo_url"`
    ReferralSource *string    `json:"referral_source,omitempty" db:"referral_source"`
    CampusID       uuid.UUID  `json:"campus_id" db:"campus_id" validate:"required"`
    IsActive       bool       `json:"is_active" db:"is_active"`
    CreatedAt      time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
    CreatedBy      *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
}
```

Rules:
- JSON tags use snake_case (matching API contract in `docs/11-api-design.md`)
- `validate` tags from `go-playground/validator` for all constrained fields
- Nullable columns use pointer types (`*string`, `*time.Time`, `*uuid.UUID`)
- `omitempty` on optional JSON fields

### Step 6: Implement CRUD Repository

File: `backend/internal/repository/<entity>_repository.go`

Implement the standard CRUD operations using pgx:

```go
package repository

import (
    "context"
    "fmt"
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/instituto-nova-sos/chesed/internal/domain"
)

type PersonRepository struct {
    pool *pgxpool.Pool
}

func NewPersonRepository(pool *pgxpool.Pool) *PersonRepository {
    return &PersonRepository{pool: pool}
}

func (r *PersonRepository) Create(ctx context.Context, p *domain.Person) error {
    query := `INSERT INTO person (id, full_name, birth_date, document_type, document_number,
              gender, email, phone, photo_url, referral_source, campus_id, created_by)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
              RETURNING created_at, updated_at, is_active`

    err := r.pool.QueryRow(ctx, query,
        p.ID, p.FullName, p.BirthDate, p.DocumentType, p.DocumentNumber,
        p.Gender, p.Email, p.Phone, p.PhotoURL, p.ReferralSource, p.CampusID, p.CreatedBy,
    ).Scan(&p.CreatedAt, &p.UpdatedAt, &p.IsActive)
    if err != nil {
        return fmt.Errorf("repository.Person.Create: %w", err)
    }
    return nil
}

func (r *PersonRepository) GetByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Person, error) {
    query := `SELECT id, full_name, birth_date, document_type, document_number,
              gender, email, phone, photo_url, referral_source, campus_id,
              is_active, created_at, updated_at, created_by
              FROM person WHERE id = $1 AND campus_id = $2 AND is_active = TRUE`
    // ... Scan into domain.Person
}

func (r *PersonRepository) List(ctx context.Context, campusID uuid.UUID, filter domain.PersonFilter) ([]domain.Person, int, error) {
    // Always filter by campus_id
    // Return total count for pagination
}

func (r *PersonRepository) Update(ctx context.Context, p *domain.Person) error {
    query := `UPDATE person SET full_name = $1, birth_date = $2, ...
              updated_at = NOW()
              WHERE id = $3 AND campus_id = $4 AND is_active = TRUE`
    // ...
}

func (r *PersonRepository) SoftDelete(ctx context.Context, id, campusID uuid.UUID) error {
    query := `UPDATE person SET is_active = FALSE, updated_at = NOW()
              WHERE id = $1 AND campus_id = $2`
    // ...
}
```

Key rules:
- Every query includes `campus_id` in WHERE
- Read queries include `is_active = TRUE` (unless explicitly querying inactive)
- Use `RETURNING` to get server-generated values
- Use `context.Context` as first parameter
- Wrap all errors with `fmt.Errorf("repository.<Entity>.<Method>: %w", err)`
- Never use string concatenation for SQL (always parameterized queries)

### Step 7: Define Repository Interface

File: `backend/internal/service/<entity>_service.go`

Define the interface at the consumption site (Go convention):

```go
package service

type PersonRepository interface {
    Create(ctx context.Context, p *domain.Person) error
    GetByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Person, error)
    List(ctx context.Context, campusID uuid.UUID, filter domain.PersonFilter) ([]domain.Person, int, error)
    Update(ctx context.Context, p *domain.Person) error
    SoftDelete(ctx context.Context, id, campusID uuid.UUID) error
}
```

### Step 8: Run Migration Locally

```bash
cd backend && make migrate-up
```

Verify:
- Migration applies without errors
- Table exists with correct columns, types, and constraints
- Indexes are created
- Down migration works: `make migrate-down` then `make migrate-up` again

### Step 9: Write Repository Integration Test

File: `backend/internal/repository/<entity>_repository_test.go`

Test against a real PostgreSQL instance (test database). Verify:

1. **Create**: Insert a record, verify it exists
2. **Read by ID**: Retrieve by ID with correct campus_id
3. **Campus isolation**: Insert with campus A, query with campus B returns no rows
4. **List with pagination**: Insert multiple records, verify pagination
5. **Update**: Modify a field, verify the change persists
6. **Soft delete**: Set is_active = false, verify record no longer returned by default queries
7. **Constraint violations**: Insert with invalid CHECK value, verify error

```go
func TestPersonRepository_CampusIsolation(t *testing.T) {
    // Create person in campus A
    person := &domain.Person{CampusID: campusA.ID, FullName: "Test"}
    err := repo.Create(ctx, person)
    require.NoError(t, err)

    // Query with campus B should return nil
    result, err := repo.GetByID(ctx, person.ID, campusB.ID)
    assert.Nil(t, result)
}
```

### Step 10: Review Migration Quality

Self-review checklist for the migration:

- [ ] Table name matches `docs/10-data-model.md` exactly
- [ ] All columns from the data model doc are present
- [ ] UUID primary key with `gen_random_uuid()`
- [ ] `created_at` and `updated_at` with `TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- [ ] `is_active BOOLEAN NOT NULL DEFAULT TRUE`
- [ ] `campus_id` with foreign key to campus table (operational tables)
- [ ] CHECK constraints match the data model doc values exactly
- [ ] VARCHAR lengths match the data model doc
- [ ] Indexes on all foreign key columns
- [ ] Down migration reverses the up migration cleanly
- [ ] Migration number is sequential and does not conflict with existing migrations

---

## Checklist

- [ ] Table confirmed in Phase 1 scope (`docs/07-mvp-scope.md`)
- [ ] DDL documented in `docs/10-data-model.md`
- [ ] Up migration created with all required columns, constraints, indexes
- [ ] Down migration created (DROP TABLE IF EXISTS)
- [ ] Domain struct created with JSON, db, and validator tags
- [ ] Repository implements CRUD with campus_id scoping
- [ ] Repository interface defined in service package
- [ ] Migration runs successfully (up and down)
- [ ] Repository integration tests pass
- [ ] Campus isolation tested
- [ ] Soft delete behavior tested
