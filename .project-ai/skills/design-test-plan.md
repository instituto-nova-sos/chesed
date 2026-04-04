# Skill: Design Test Plan

## Purpose

Design comprehensive test cases following the project's testing pyramid. Produces test specifications for backend unit tests (table-driven, service layer), backend integration tests (repository against real PostgreSQL), frontend component tests (Vitest + React Testing Library), and offline behavior tests.

## When to Use / Trigger

- After designing a backend or frontend feature (via design-backend-feature or design-frontend-feature skills).
- When a user says "write tests for feature X" or "design test plan for story Y".
- Before marking any story as implementation-complete.

## Role / Expertise

QA engineer with expertise in:
- Go table-driven tests with testify assertions.
- PostgreSQL integration tests with Docker containers.
- Vitest + React Testing Library for frontend testing.
- Security test patterns from `docs/17-security-test-strategy.md`.
- Offline/sync behavior testing.

## Inputs

| Input | Required | Source |
|-------|----------|--------|
| Feature design (backend and/or frontend) | Yes | Prior skill output |
| Story acceptance criteria | Yes | `docs/09-backlog.md` |
| Security test patterns | Optional | `docs/17-security-test-strategy.md` |

## Process

### Step 1: Backend Service Layer Unit Tests

Location: `backend/internal/service/*_test.go`

Pattern: Table-driven tests with mocked repository.

For each service method, define test cases covering:

```go
func TestPersonService_Create(t *testing.T) {
    tests := []struct {
        name      string
        input     domain.CreatePersonInput
        mockSetup func(repo *MockPersonRepository)
        wantErr   error
        wantAudit bool
    }{
        {
            name:  "valid person creates successfully",
            input: validPersonInput(),
            mockSetup: func(repo *MockPersonRepository) {
                repo.On("CheckDuplicate", mock.Anything, "CPF", "123.456.789-00").Return(nil, nil)
                repo.On("Create", mock.Anything, mock.Anything).Return(nil)
            },
            wantErr:   nil,
            wantAudit: true,
        },
        // ... more cases
    }
}
```

Required test categories per service method:
1. **Happy path**: Valid input, all dependencies succeed.
2. **Validation failures**: Missing required fields, invalid formats, out-of-range values.
3. **Business rule violations**: Duplicate documents, invalid state transitions, unauthorized access.
4. **Dependency failures**: Repository returns error, audit log fails.
5. **Campus scoping**: Verify campus_id is passed to repository calls.
6. **Audit logging**: Verify audit log is called with correct entity_type, action, old/new values.

### Step 2: Backend Repository Integration Tests

Location: `backend/internal/repository/*_test.go`

Pattern: Tests run against a real PostgreSQL instance (Docker in CI, local in dev).

```go
func TestPersonRepository_Create_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    db := setupTestDB(t) // Connect to test PostgreSQL, run migrations
    repo := repository.NewPersonRepository(db)

    // Test CRUD operations
    // Test campus isolation
    // Test pagination
    // Test search with tsvector
}
```

Required integration test categories:
1. **CRUD**: Create, read, update for each entity.
2. **Campus isolation**: Insert records for campus A and B; query with campus A filter; verify only campus A records returned.
3. **Pagination**: Insert N records; verify page size, total count, total pages.
4. **Search**: Full-text search returns matching records.
5. **Constraints**: Unique constraint violations return appropriate errors.
6. **Concurrent access**: Optimistic locking behavior (if implemented).

### Step 3: Backend Handler Tests

Location: `backend/internal/handler/*_test.go`

Pattern: Use `httptest` with mocked service.

```go
func TestCreatePersonHandler(t *testing.T) {
    tests := []struct {
        name       string
        body       string
        authClaims middleware.AuthClaims
        mockSetup  func(svc *MockPersonService)
        wantStatus int
        wantBody   string
    }{
        {
            name:       "valid request returns 201",
            body:       `{"full_name":"Test","document_type":"CPF"}`,
            authClaims: middleware.AuthClaims{CampusID: "campus-1", Roles: []string{"SECRETARY"}},
            wantStatus: http.StatusCreated,
        },
        {
            name:       "invalid JSON returns 400",
            body:       `{invalid`,
            wantStatus: http.StatusBadRequest,
        },
        // ... more cases
    }
}
```

Required handler test categories:
1. **Status codes**: Correct code for each scenario (201, 200, 400, 401, 403, 404, 409, 500).
2. **Request parsing**: Valid JSON, invalid JSON, missing fields.
3. **Response format**: Matches API spec (field names, pagination wrapper, error format).
4. **RBAC**: Verify 403 for insufficient roles.
5. **Error responses**: Format matches `{ "error": { "code": "...", "message": "...", "details": [...] } }`.

### Step 4: Security Tests

Reference: `docs/17-security-test-strategy.md`

Layer 1 (Unit):
- Input validation against XSS, SQL injection, oversized inputs.
- OIDC token validation: expired, wrong issuer, wrong audience, missing campus_id, missing roles.

Layer 2 (Integration):
- RBAC matrix: every role x every endpoint combination.
- Campus isolation: cross-campus data access attempts return 403 or empty results.
- Audit logging: mutations create audit entries; failed access logged.

```go
func TestRBACMatrix(t *testing.T) {
    endpoints := []struct {
        method string
        path   string
        allowedRoles []string
    }{
        {"POST", "/api/v1/persons", []string{"VOLUNTEER", "SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN"}},
        {"PUT", "/api/v1/persons/uuid", []string{"SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN"}},
        {"POST", "/api/v1/persons/uuid/roles", []string{"COORDINATOR", "ADMIN"}},
        // ... all endpoints from docs/11-api-design.md
    }

    roles := []string{"VOLUNTEER", "SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN"}

    for _, ep := range endpoints {
        for _, role := range roles {
            // Test: role allowed -> 2xx; role not allowed -> 403
        }
    }
}
```

### Step 5: Frontend Component Tests

Location: `frontend/src/components/**/*.test.tsx` (co-located)

Pattern: Vitest + React Testing Library.

```typescript
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { PersonForm } from './PersonForm';

describe('PersonForm', () => {
  it('renders all required fields', () => {
    render(<PersonForm onSubmit={vi.fn()} />);
    expect(screen.getByLabelText(/nome completo/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/cpf/i)).toBeInTheDocument();
  });

  it('shows validation errors for empty required fields', async () => {
    render(<PersonForm onSubmit={vi.fn()} />);
    fireEvent.click(screen.getByRole('button', { name: /salvar/i }));
    await waitFor(() => {
      expect(screen.getByText(/nome completo obrigatorio/i)).toBeInTheDocument();
    });
  });

  it('calls onSubmit with valid data', async () => {
    const onSubmit = vi.fn();
    render(<PersonForm onSubmit={onSubmit} />);
    // Fill fields, submit, verify onSubmit called with correct data
  });
});
```

Required frontend test categories:
1. **Render**: Component renders without errors.
2. **Form validation**: Required fields, format validation, error messages (pt-BR).
3. **User interaction**: Click, type, select; verify correct behavior.
4. **Loading states**: Skeleton/spinner shown while data loads.
5. **Error states**: Error message displayed on API failure.
6. **Role-based UI**: Elements hidden/shown based on user role.
7. **Mobile responsiveness**: Not tested via RTL (manual or visual regression).

### Step 6: Offline Behavior Tests

For features that support offline operation:

```typescript
describe('PersonForm - Offline', () => {
  beforeEach(() => {
    // Mock navigator.onLine = false
  });

  it('saves to IndexedDB when offline', async () => {
    // Submit form -> verify Dexie insert called
    // Verify syncStatus = 'pending'
    // Verify sync queue entry created
  });

  it('generates client-side UUID for offline records', async () => {
    // Submit form -> verify id is a valid UUID
  });

  it('shows sync pending indicator', async () => {
    // Create offline record -> verify pending badge visible
  });
});
```

### Step 7: Edge Cases Checklist

For every feature, explicitly test:
- [ ] Empty list (zero records).
- [ ] Single record.
- [ ] Maximum page size.
- [ ] Unicode characters in text fields (Portuguese accented characters: a, e, i, c).
- [ ] Maximum field lengths.
- [ ] Concurrent modifications (optimistic locking).
- [ ] Network timeout during sync.
- [ ] Token expiry mid-operation.
- [ ] Boundary dates (birth_date in future, very old dates).

### Step 8: Test Quality Criteria

Beyond coverage numbers, tests must meet quality standards:

- [ ] **Independence**: Each test runs in isolation. No shared mutable state between tests. No test depends on another test's execution.
- [ ] **Determinism**: Tests produce the same result on every run. No time-dependent assertions, no random data without seeding.
- [ ] **Meaningful assertions**: Tests assert behavior, not implementation. Assert on output/side-effects, not internal state.
- [ ] **Boundary value analysis**: Edge cases tested — empty inputs, maximum values, off-by-one boundaries, nil/null.
- [ ] **Error path coverage**: Failure scenarios tested with equal rigor as happy paths. Service error handling has explicit test cases.
- [ ] **Test naming**: Test names describe the scenario and expected outcome: `Test_Create_DuplicateDocument_ReturnsError`.
- [ ] **No test pollution**: Tests clean up after themselves. Database tests use transactions rolled back after each test.

### Coverage Thresholds (from Quality Gates)

| Scope | Threshold | Source |
|-------|-----------|--------|
| New code (per PR) | ≥ 80% | `docs/quality/quality-gates.md` — New Code Gate |
| Service layer | ≥ 80% | `docs/15-implementation-guidelines.md` |
| Handler layer | ≥ 60% | `docs/15-implementation-guidelines.md` |
| Critical paths (auth, RBAC, campus isolation) | ≥ 90% | Security requirements |
| Overall codebase (Sprint 1-2) | ≥ 70% | `docs/quality/quality-gates.md` — Overall Code Gate |
| Overall codebase (Sprint 3+) | ≥ 80% | `docs/quality/quality-gates.md` — Overall Code Gate |

## Outputs / Deliverables

1. **Test case table**: Organized by layer (service, repository, handler, component, offline).
2. **Test file paths**: Exact locations for each test file.
3. **Mock definitions**: Interfaces to mock and mock setup patterns.
4. **Test data fixtures**: Reusable test data factories.
5. **Coverage targets**: Per quality gate thresholds above.
6. **Test quality assessment**: Pass/fail on test quality criteria (independence, determinism, meaningful assertions).

## References

| Document | Path | Usage |
|----------|------|-------|
| Security test strategy | `docs/17-security-test-strategy.md` | Security test patterns and examples |
| Implementation guidelines | `docs/15-implementation-guidelines.md` | Testing framework choices |
| API design | `docs/11-api-design.md` | Response schemas for handler tests |
| Offline sync | `docs/12-offline-sync-strategy.md` | Sync behavior for offline tests |
| Quality gates | `docs/quality/quality-gates.md` | Coverage thresholds |

## Constraints / Quality Bar

- All service layer methods must have table-driven tests.
- Repository tests must run against real PostgreSQL (not mocked SQL).
- Handler tests must verify status codes AND response body structure.
- Security tests (RBAC matrix, campus isolation) are mandatory for every endpoint.
- No test may depend on external services (Keycloak mocked in tests).
- Frontend form tests must verify pt-BR validation messages.

## Interaction with Other Artifacts

- **Invoked by agents**: backend-engineer, frontend-engineer, tech-lead.
- **Depends on skills**: design-backend-feature, design-frontend-feature (feature designs provide test targets).
- **Feeds into skills**: assess-release-readiness (tests must pass for release).
