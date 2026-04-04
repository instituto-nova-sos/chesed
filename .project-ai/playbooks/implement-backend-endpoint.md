# Playbook: Implement Backend Endpoint

## Purpose

End-to-end guide for implementing a new REST API endpoint in the Chesed Go backend. Covers domain modeling through testing and documentation verification.

---

## Prerequisites

- The endpoint is documented in `docs/11-api-design.md`
- The backing database table(s) exist and are in the Phase 1 list (`docs/07-mvp-scope.md`)
- The endpoint's RBAC requirements are defined in `docs/16-iam-and-access-control.md`

---

## Steps

### Step 1: Review API Contract and Data Model

Read the endpoint specification before writing any code.

- Open `docs/11-api-design.md` and locate the endpoint. Note:
  - HTTP method and path (e.g., `POST /api/v1/persons`)
  - Required and optional request fields
  - Response schema (status code, JSON structure, pagination format)
  - Role requirements (e.g., "Secretary+", "Coordinator+", "All")
  - Error cases
- Open `docs/10-data-model.md` and identify the backing table(s). Note:
  - Column names, types, constraints, CHECK values
  - Foreign keys and required indexes
  - Whether `campus_id`, `is_active`, `created_at`, `updated_at` are present (they must be)

### Step 2: Create or Update Domain Struct

File: `backend/internal/domain/<entity>.go`

- Define a Go struct matching the database table columns
- Use proper Go types: `uuid.UUID` for UUIDs, `time.Time` for timestamps, `*string` for nullable strings
- Add JSON tags matching the API response field names from `docs/11-api-design.md`
- Add `validate` tags from `go-playground/validator` for required fields and constraints
- If the endpoint accepts a request body, create a separate `Create<Entity>Request` or `Update<Entity>Request` struct

```go
package domain

import (
    "time"
    "github.com/google/uuid"
)

type Person struct {
    ID             uuid.UUID  `json:"id"`
    FullName       string     `json:"full_name" validate:"required,max=200"`
    BirthDate      *time.Time `json:"birth_date,omitempty"`
    DocumentType   string     `json:"document_type" validate:"required,oneof=CPF SSN EU_ID PASSPORT OTHER"`
    DocumentNumber *string    `json:"document_number,omitempty" validate:"omitempty,max=30"`
    Gender         *string    `json:"gender,omitempty" validate:"omitempty,oneof=M F OTHER PREFER_NOT_TO_SAY"`
    Email          *string    `json:"email,omitempty" validate:"omitempty,email,max=255"`
    Phone          *string    `json:"phone,omitempty" validate:"omitempty,max=30"`
    CampusID       uuid.UUID  `json:"campus_id" validate:"required"`
    IsActive       bool       `json:"is_active"`
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
}

type CreatePersonRequest struct {
    FullName       string  `json:"full_name" validate:"required,max=200"`
    BirthDate      *string `json:"birth_date,omitempty"`
    DocumentType   string  `json:"document_type" validate:"required,oneof=CPF SSN EU_ID PASSPORT OTHER"`
    DocumentNumber *string `json:"document_number,omitempty"`
    Gender         *string `json:"gender,omitempty" validate:"omitempty,oneof=M F OTHER PREFER_NOT_TO_SAY"`
    Email          *string `json:"email,omitempty" validate:"omitempty,email"`
    Phone          *string `json:"phone,omitempty"`
    SyncID         *string `json:"sync_id,omitempty"`
}
```

### Step 3: Define Repository Interface

File: `backend/internal/service/<entity>_service.go`

Define the repository interface at the consumption site (the service package), not in the repository package. This follows Go's implicit interface satisfaction and keeps the service testable.

```go
package service

import (
    "context"
    "github.com/google/uuid"
    "github.com/instituto-nova-sos/chesed/internal/domain"
)

type PersonRepository interface {
    Create(ctx context.Context, person *domain.Person) error
    GetByID(ctx context.Context, id uuid.UUID, campusID uuid.UUID) (*domain.Person, error)
    List(ctx context.Context, campusID uuid.UUID, filter domain.PersonFilter) ([]domain.Person, int, error)
    Update(ctx context.Context, person *domain.Person) error
}
```

Key rules:
- `context.Context` is always the first parameter
- `campusID uuid.UUID` is always included in read/write operations (campus scoping)
- Return `error` as the last return value
- Use domain structs, not database-specific types

### Step 4: Implement Repository

File: `backend/internal/repository/<entity>_repository.go`

- Use `pgx` for all database access (not `database/sql`)
- Every query MUST include `WHERE campus_id = $N` for campus scoping
- Every query on active records MUST include `WHERE is_active = TRUE` unless explicitly querying inactive records
- Use parameterized queries (never string concatenation for SQL)
- Scan results into domain structs
- Wrap database errors with context using `fmt.Errorf("repository.Person.Create: %w", err)`

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

func (r *PersonRepository) GetByID(ctx context.Context, id uuid.UUID, campusID uuid.UUID) (*domain.Person, error) {
    query := `SELECT id, full_name, birth_date, document_type, document_number,
              gender, email, phone, campus_id, is_active, created_at, updated_at
              FROM person WHERE id = $1 AND campus_id = $2 AND is_active = TRUE`

    var p domain.Person
    err := r.pool.QueryRow(ctx, query, id, campusID).Scan(
        &p.ID, &p.FullName, &p.BirthDate, &p.DocumentType, &p.DocumentNumber,
        &p.Gender, &p.Email, &p.Phone, &p.CampusID, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
    )
    if err != nil {
        return nil, fmt.Errorf("repository.Person.GetByID: %w", err)
    }
    return &p, nil
}
```

### Step 5: Implement Service

File: `backend/internal/service/<entity>_service.go`

- Contains all business logic and validation
- Depends on the repository interface (defined in the same package)
- Never imports `pgx`, `pgxpool`, or any database driver
- Validates input using `go-playground/validator`
- Wraps errors with business context using `fmt.Errorf`
- For mutations: calls audit log service to record the change

```go
package service

import (
    "context"
    "fmt"
    "github.com/go-playground/validator/v10"
    "github.com/instituto-nova-sos/chesed/internal/domain"
)

type PersonService struct {
    repo     PersonRepository
    audit    AuditService
    validate *validator.Validate
}

func NewPersonService(repo PersonRepository, audit AuditService) *PersonService {
    return &PersonService{
        repo:     repo,
        audit:    audit,
        validate: validator.New(),
    }
}

func (s *PersonService) Create(ctx context.Context, req domain.CreatePersonRequest, campusID, userID uuid.UUID) (*domain.Person, error) {
    if err := s.validate.Struct(req); err != nil {
        return nil, fmt.Errorf("service.Person.Create: validation failed: %w", err)
    }

    person := &domain.Person{
        ID:             uuid.New(),
        FullName:       req.FullName,
        DocumentType:   req.DocumentType,
        DocumentNumber: req.DocumentNumber,
        CampusID:       campusID,
        IsActive:       true,
    }

    if err := s.repo.Create(ctx, person); err != nil {
        return nil, fmt.Errorf("service.Person.Create: %w", err)
    }

    // Audit log for data mutation
    s.audit.Log(ctx, domain.AuditEntry{
        EntityType: "person",
        EntityID:   person.ID,
        Action:     "CREATE",
        UserID:     userID,
        CampusID:   campusID,
        NewValues:  person,
    })

    return person, nil
}
```

### Step 6: Implement Handler

File: `backend/internal/handler/<entity>_handler.go`

- Parse and decode the HTTP request (path params, query params, JSON body)
- Extract `campus_id` and `user_id` from the request context (set by auth middleware)
- Call the service layer (never the repository directly)
- Format the response according to `docs/11-api-design.md`
- Use the standard error response format for all errors
- Log errors with `slog`

```go
package handler

import (
    "encoding/json"
    "log/slog"
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/instituto-nova-sos/chesed/internal/middleware"
    "github.com/instituto-nova-sos/chesed/internal/service"
)

type PersonHandler struct {
    service *service.PersonService
}

func NewPersonHandler(svc *service.PersonService) *PersonHandler {
    return &PersonHandler{service: svc}
}

func (h *PersonHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req domain.CreatePersonRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    campusID := middleware.CampusIDFromContext(r.Context())
    userID := middleware.UserIDFromContext(r.Context())

    person, err := h.service.Create(r.Context(), req, campusID, userID)
    if err != nil {
        slog.Error("failed to create person", "error", err)
        writeError(w, http.StatusInternalServerError, "Failed to create person")
        return
    }

    writeJSON(w, http.StatusCreated, person)
}
```

Standard error response format (from `docs/11-api-design.md`):

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Human-readable message (no PII)"
  }
}
```

### Step 7: Register Route with RBAC Middleware

File: `backend/cmd/server/main.go` (or router setup file)

Register the route under `/api/v1` with the appropriate RBAC middleware. Match roles from the endpoint table in `docs/11-api-design.md`.

```go
r.Route("/api/v1", func(r chi.Router) {
    r.Use(middleware.Authenticate) // All /api/v1 routes require auth

    // Person endpoints
    r.Route("/persons", func(r chi.Router) {
        r.With(middleware.RequireRole("VOLUNTEER", "SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN")).Post("/", personHandler.Create)
        r.With(middleware.RequireRole("VOLUNTEER", "SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN")).Get("/", personHandler.List)
        r.With(middleware.RequireRole("SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN")).Put("/{id}", personHandler.Update)
    })
})
```

Role hierarchy reference (`docs/16-iam-and-access-control.md`):
- "All" = all authenticated roles
- "Secretary+" = SECRETARY, PROFESSIONAL, COORDINATOR, ADMIN
- "Professional+" = PROFESSIONAL, COORDINATOR, ADMIN
- "Coordinator+" = COORDINATOR, ADMIN
- "Admin" = ADMIN only

### Step 8: Verify Audit Logging

For every data mutation (POST, PUT, PATCH, DELETE):

- The service layer must create an audit log entry
- The entry must include: `entity_type`, `entity_id`, `action` (CREATE/UPDATE/DELETE), `user_id`, `campus_id`, `old_values` (for updates), `new_values`
- The `audit_log` table is append-only (no updates or deletes)
- Verify no PII appears in log messages or error responses

### Step 9: Write Tests

**Service unit tests** (file: `backend/internal/service/<entity>_service_test.go`):
- Use table-driven tests with `testing` + `testify`
- Mock the repository interface
- Test happy path, validation failures, repository errors
- Test campus scoping (service must pass campusID to repository)

```go
func TestPersonService_Create(t *testing.T) {
    tests := []struct {
        name    string
        req     domain.CreatePersonRequest
        wantErr bool
    }{
        {
            name:    "valid request",
            req:     domain.CreatePersonRequest{FullName: "Test Person", DocumentType: "CPF"},
            wantErr: false,
        },
        {
            name:    "missing full_name",
            req:     domain.CreatePersonRequest{DocumentType: "CPF"},
            wantErr: true,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup mock repo, call service, assert
        })
    }
}
```

**Repository integration tests** (file: `backend/internal/repository/<entity>_repository_test.go`):
- Use a real PostgreSQL test database
- Test CRUD operations
- Verify campus scoping (insert with campus A, query with campus B returns nothing)
- Verify soft delete (is_active = false hides records)

### Step 10: Verify Documentation Accuracy

- Re-read `docs/11-api-design.md` and confirm:
  - Request/response fields match the implementation exactly
  - Allowed roles match the middleware configuration
  - Error codes and formats are consistent
- If you changed the contract, update the doc first, then confirm the implementation matches

### Step 11: Run Pre-Review Checks

```bash
# Run Go tests
cd backend && go test ./...

# Run linter
cd backend && golangci-lint run

# Format
cd backend && gofmt -w .

# Verify the endpoint responds correctly (manual or integration test)
curl -X POST http://localhost:8080/api/v1/persons \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"full_name": "Test Person", "document_type": "CPF"}'
```

---

## Checklist

- [ ] Endpoint documented in `docs/11-api-design.md`
- [ ] Domain struct created with JSON + validator tags
- [ ] Repository interface defined in service package
- [ ] Repository implementation uses pgx with campus_id in WHERE
- [ ] Service implements business logic with validation
- [ ] Handler parses request, calls service, formats response per API doc
- [ ] Route registered with correct RBAC middleware
- [ ] Audit log entry created for mutations
- [ ] Service unit tests (table-driven)
- [ ] Repository integration test
- [ ] No PII in logs or error responses
- [ ] All existing tests pass
- [ ] No lint warnings
