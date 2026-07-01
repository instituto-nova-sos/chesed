# Backend Feature Complete Checklist

Use this checklist before marking any backend feature as done. Every item must pass.

---

## Domain Layer

- [ ] Domain struct defined in `internal/domain/` with `validator` tags (go-playground/validator)
- [ ] UUID primary key field (`ID uuid.UUID`)
- [ ] `CreatedAt`, `UpdatedAt` as `time.Time` (maps to `timestamptz`)
- [ ] `IsActive bool` for soft-delete support
- [ ] `CampusID uuid.UUID` present on all operational structs
- [ ] No dependencies on other packages (pure data structs)

## Repository Layer

- [ ] Repository interface defined in `internal/service/` (dependency inversion)
- [ ] Repository implementation in `internal/repository/` using `pgx`
- [ ] All queries include `campus_id` filter (from authenticated user's token claims)
- [ ] SQL uses parameterized queries only (`$1`, `$2` — never string interpolation)
- [ ] `context.Context` is the first parameter on all repository methods
- [ ] Proper error wrapping with `fmt.Errorf("repo.Method: %w", err)`
- [ ] Repository unit tests use `pgxmock` to pin the SQL contract (column names, WHERE clauses, ORDER BY, parameter positions)
- [ ] **Integration test** in `backend/internal/integration/` exercises the repository against a real PostgreSQL via `testcontainers-go`. Mandatory per `.project-ai/checklists/integration-tests.md`

## Service Layer

- [ ] Service struct in `internal/service/` depends on repository interface (not implementation)
- [ ] Business logic and validation rules live here (not in handler or repository)
- [ ] Error wrapping with `fmt.Errorf` and `%w` verb for error chain
- [ ] `context.Context` is the first parameter on all service methods
- [ ] No direct database driver imports (`pgx` never appears in service code)
- [ ] Table-driven unit tests with `testing` + `testify`

## Handler Layer

- [ ] Handler in `internal/handler/` with request parsing, input validation, response formatting
- [ ] Request body decoded and validated before calling service
- [ ] Response formatted as JSON matching `docs/11-api-design.md`
- [ ] Error responses follow standard format: `{ "error": "<code>", "message": "<description>", "details": ... }`
- [ ] Proper HTTP status codes used:
  - `201 Created` for resource creation
  - `200 OK` for read and update
  - `204 No Content` for delete
  - `400 Bad Request` for validation errors
  - `401 Unauthorized` for missing/invalid token
  - `403 Forbidden` for insufficient role
  - `404 Not Found` for missing resources
  - `409 Conflict` for duplicate/conflict
- [ ] Pagination follows format: `page`, `per_page`, `total`, `total_pages`

## Routing & Middleware

- [ ] Route registered in `cmd/server/main.go` under `/api/v1/`
- [ ] RBAC middleware applied (role requirement matches `docs/11-api-design.md`)
- [ ] No unprotected routes (except `/health`)

## Audit & Logging

- [ ] Audit log entries created for all mutations (CREATE, UPDATE, DELETE)
- [ ] Audit log includes: `user_id`, `campus_id`, `action`, `table_name`, `record_id`, `old_values`, `new_values`
- [ ] `slog` structured logging used throughout (no `fmt.Println`, no `log.Println`)
- [ ] No PII in log output (no names, CPF, phone numbers, addresses)

## Code Quality

- [ ] **TDD**: test was written first and seen to fail before implementation (verifiable via RED→GREEN commit order; see `rules/tdd-enforcement.md`)
- [ ] No `_` for errors — all errors are handled or explicitly documented
- [ ] `context.Context` propagated as first parameter in all function chains
- [ ] `make test` (unit tests) passes with zero failures
- [ ] `make test-integration` (real Postgres via testcontainers-go) passes with zero failures
- [ ] `make lint` passes with zero warnings (golangci-lint)
- [ ] No `TODO`, `FIXME`, or `HACK` comments left unresolved

## Quality Profile Compliance

Per `docs/quality/quality-profiles.md` (Go Backend) and `docs/quality/complexity-guidelines.md`:

- [ ] Cognitive complexity per function ≤ 25
- [ ] Cyclomatic complexity per function ≤ 10
- [ ] Function length ≤ 40 lines
- [ ] File length ≤ 400 lines
- [ ] Nesting depth ≤ 3 levels
- [ ] Parameter count ≤ 5
- [ ] Return values ≤ 3
- [ ] Duplicated lines on new code ≤ 3%
- [ ] Clean code categories satisfied:
  - [ ] Consistency — patterns match sibling files
  - [ ] Intentionality — names reveal purpose, no dead code
  - [ ] Adaptability — dependencies point inward
  - [ ] Responsibility — each function has single responsibility

## Documentation Sync

- [ ] If new endpoint: documented in `docs/11-api-design.md`
- [ ] If new table/column: documented in `docs/10-data-model.md`
- [ ] If domain model change: documented in `docs/04-domain-model.md`

---

## How to Use

Run through this checklist item by item. If any item fails, fix it before proceeding.

```
Skill:   review-code (for implementation quality)
Hook:    pre-review (automated checks before marking complete)
Agent:   backend-engineer (for implementation), tech-lead (for design review)
```
