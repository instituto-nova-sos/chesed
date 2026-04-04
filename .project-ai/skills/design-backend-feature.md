# Skill: Design Backend Feature

## Purpose

Design a complete Go backend feature following the handler -> service -> repository layered architecture. Produces domain structs, repository interfaces, service logic, HTTP handlers, route registration with RBAC, audit log entries, and test case specifications.

## When to Use / Trigger

- After a story has been analyzed (via analyze-requirements skill).
- When a user says "design backend for story X" or "implement the Go side of feature Y".
- Before writing any backend code for a new feature.

## Role / Expertise

Go backend engineer with deep knowledge of:
- chi router for HTTP routing.
- pgx for PostgreSQL queries with context propagation.
- coreos/go-oidc for Keycloak OIDC token validation.
- Table-driven testing with testify.
- Clean architecture: handler -> service -> repository -> domain.

## Inputs

| Input | Required | Source |
|-------|----------|--------|
| Story analysis (from analyze-requirements) | Yes | Prior skill output |
| Table DDL for affected tables | Yes | `docs/10-data-model.md` |
| API endpoint contract | Yes | `docs/11-api-design.md` |
| RBAC role requirements | Yes | `docs/11-api-design.md` |

## Process

### Step 1: Define Domain Structs

Location: `backend/internal/domain/`

1. Read the table DDL from `docs/10-data-model.md`.
2. Create Go structs that mirror the database schema.
3. Use `time.Time` for all `timestamptz` columns.
4. Use `uuid.UUID` (or `string`) for UUID columns.
5. Use pointer types for nullable columns.
6. Add JSON tags matching the API response field names from `docs/11-api-design.md`.
7. Add `validate` struct tags for input validation (go-playground/validator).

```go
// Example: backend/internal/domain/person.go
type Person struct {
    ID             string     `json:"id"`
    FullName       string     `json:"full_name" validate:"required,max=200"`
    BirthDate      *time.Time `json:"birth_date,omitempty"`
    DocumentType   string     `json:"document_type" validate:"required,oneof=CPF SSN EU_ID PASSPORT OTHER"`
    DocumentNumber *string    `json:"document_number,omitempty" validate:"omitempty,max=30"`
    CampusID       string     `json:"campus_id"`
    IsActive       bool       `json:"is_active"`
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
}
```

Also define request/response DTOs:
- `CreateXxxInput` -- validated input struct for POST.
- `UpdateXxxInput` -- validated input struct for PUT/PATCH.
- `XxxFilter` -- query parameters for list endpoints (includes `CampusID` always).
- `PaginatedResult[T]` -- generic paginated response.

### Step 2: Define Repository Interface

Location: `backend/internal/service/` (interfaces at consumption site per project rules).

1. Define the repository interface in the service package (not the repository package).
2. Methods accept `context.Context` as first parameter.
3. Methods return `(result, error)` tuples.
4. Include `campusID` parameter on all query methods.

```go
// Example: backend/internal/service/person_service.go
type PersonRepository interface {
    Create(ctx context.Context, person *domain.Person) error
    GetByID(ctx context.Context, id string, campusID string) (*domain.Person, error)
    List(ctx context.Context, filter domain.PersonFilter) (*domain.PaginatedResult[domain.Person], error)
    Update(ctx context.Context, person *domain.Person) error
    CheckDuplicate(ctx context.Context, documentType, documentNumber string) ([]domain.DuplicateMatch, error)
}
```

### Step 3: Implement Repository

Location: `backend/internal/repository/`

1. Implement the interface using pgx.
2. All queries MUST include `WHERE campus_id = $N` filter (except for duplicate checks which are cross-campus by design).
3. Use parameterized queries (never string concatenation).
4. Use `pgx.CollectRows` for list queries.
5. Return `domain.ErrNotFound` for missing records.
6. Wrap errors with `fmt.Errorf("personRepo.Create: %w", err)`.

### Step 4: Implement Service Layer

Location: `backend/internal/service/`

1. Service struct receives repository interface via constructor injection.
2. Business logic goes here: validation, authorization checks, workflow rules.
3. For mutations, prepare audit log data (entity type, action, old/new values).
4. Return domain-level errors (not HTTP errors).
5. Never import pgx or any database driver.

```go
type PersonService struct {
    repo     PersonRepository
    auditLog AuditLogRepository
    logger   *slog.Logger
}

func NewPersonService(repo PersonRepository, auditLog AuditLogRepository, logger *slog.Logger) *PersonService {
    return &PersonService{repo: repo, auditLog: auditLog, logger: logger}
}
```

Audit log entry format (per `docs/10-data-model.md` audit_log table):
- `entity_type`: "person", "triage", "attendance", etc.
- `entity_id`: UUID of the affected record.
- `action`: "CREATE", "UPDATE", "DELETE", "ACCESS".
- `old_values`: JSONB of previous state (for updates).
- `new_values`: JSONB of new state.
- `performed_by`: UUID from JWT claims.
- `campus_id`: from JWT claims.

### Step 5: Implement HTTP Handler

Location: `backend/internal/handler/`

1. Parse HTTP request (path params, query params, JSON body).
2. Validate input using go-playground/validator.
3. Extract authenticated user context (campus_id, user_id, roles) from request context (set by auth middleware).
4. Call service method.
5. Map service result to HTTP response.
6. Use consistent status codes per `docs/11-api-design.md`:
   - 201 for successful creation.
   - 200 for successful retrieval/update.
   - 400 for validation errors.
   - 401 for missing/invalid token.
   - 403 for insufficient role.
   - 404 for not found.
   - 409 for conflict (duplicate).
   - 500 for unexpected errors.
7. Error response format: `{ "error": { "code": "VALIDATION_ERROR", "message": "...", "details": [...] } }`.
8. Never expose internal error details or PII in error responses.

### Step 6: Register Routes

Location: `backend/internal/handler/routes.go` (or similar).

1. Group under `/api/v1/`.
2. Apply auth middleware to all routes.
3. Apply RBAC middleware per endpoint using role constants.
4. RBAC role hierarchy: ADMIN > COORDINATOR > PROFESSIONAL > SECRETARY > VOLUNTEER.

```go
r.Route("/api/v1", func(r chi.Router) {
    r.Use(middleware.AuthMiddleware(oidcValidator))

    r.Route("/persons", func(r chi.Router) {
        r.With(middleware.RequireRole("VOLUNTEER", "SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN")).Post("/", handler.CreatePerson)
        r.With(middleware.RequireRole("VOLUNTEER", "SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN")).Get("/", handler.ListPersons)
        r.With(middleware.RequireRole("SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN")).Put("/{id}", handler.UpdatePerson)
    })
})
```

### Step 7: Define Test Cases

1. **Service layer unit tests** (table-driven, mocked repository):
   - Happy path for each method.
   - Validation failure cases.
   - Business rule violations (e.g., duplicate document).
   - Campus scoping enforcement.
2. **Repository integration tests** (real PostgreSQL via Docker):
   - CRUD operations against test database.
   - Campus isolation verification.
   - Pagination behavior.
3. **Handler tests** (httptest, mocked service):
   - Correct status codes per scenario.
   - Request parsing and validation.
   - Error response format.

## Outputs / Deliverables

1. **Domain struct definitions** with JSON and validation tags.
2. **Repository interface** with all required methods.
3. **Repository implementation** with pgx queries and campus_id filtering.
4. **Service implementation** with business logic and audit logging.
5. **Handler implementation** with request parsing, validation, and response formatting.
6. **Route registration** with RBAC middleware.
7. **Test case specifications** organized by layer.
8. **File manifest**: exact file paths for all files to create/modify.

## References

| Document | Path | Usage |
|----------|------|-------|
| Data model | `docs/10-data-model.md` | Table DDL for domain structs and queries |
| API design | `docs/11-api-design.md` | Endpoint contracts, status codes, response format |
| Implementation guidelines | `docs/15-implementation-guidelines.md` | Code patterns and approved dependencies |
| IAM and access control | `docs/16-iam-and-access-control.md` | RBAC roles, token claims |
| Security test strategy | `docs/17-security-test-strategy.md` | Security test patterns |

## Constraints / Quality Bar

- Handler depends on service only (never on repository directly).
- Service depends on repository interface (defined in service package).
- Repository depends on domain structs and pgx only.
- Domain has zero dependencies.
- All errors handled explicitly (no `_` for errors).
- All I/O functions take `context.Context` as first parameter.
- All queries include `campus_id` filter from JWT claims.
- All mutations create audit log entries.
- No global mutable state or singletons.
- No PII in log messages or error responses.
- Use `slog` for structured logging.
- Use `fmt.Errorf("context: %w", err)` for error wrapping.

## Interaction with Other Artifacts

- **Invoked by agents**: backend-engineer, tech-lead.
- **Depends on skills**: analyze-requirements (provides story analysis).
- **Feeds into skills**: design-test-plan (test specifications), review-code (review target).
- **Triggers skills**: maintain-docs (after implementation, update API and data model docs).
