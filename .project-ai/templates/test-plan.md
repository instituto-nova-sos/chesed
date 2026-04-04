# Test Plan: [Feature Name]

## Metadata

| Field | Value |
|-------|-------|
| **Story Reference** | S0x.x — [Story title from docs/09-backlog.md] |
| **Author** | [Name] |
| **Date** | [YYYY-MM-DD] |
| **Status** | Draft / In Progress / Complete |

---

## Coverage Targets

| Layer | Target | Tool |
|-------|--------|------|
| Service (business logic) | 80% | `go test` + `testify` |
| Handler (HTTP layer) | 60% | `go test` + `httptest` |
| Repository (data access) | Integration tests | `go test` + test PostgreSQL |
| Frontend hooks | 70% | Vitest + React Testing Library |
| Frontend forms/components | Validation + submission paths | Vitest + React Testing Library |
| Offline operations | Manual + automated | Vitest (Dexie) + manual browser test |

---

## Unit Tests: Service Layer

File: `backend/internal/service/[entity]_service_test.go`

Use table-driven tests with `testing` + `testify`. Mock the repository interface.

### Test Cases

| # | Test Name | Input | Expected | Validates |
|---|-----------|-------|----------|-----------|
| 1 | [e.g., Create_ValidRequest_ReturnsEntity] | Valid CreatePersonRequest | Person created, no error | Happy path |
| 2 | [e.g., Create_MissingFullName_ReturnsError] | Request with empty full_name | Validation error | Required field validation |
| 3 | [e.g., Create_InvalidDocumentType_ReturnsError] | Request with document_type="INVALID" | Validation error | Enum constraint |
| 4 | [e.g., Create_RepositoryError_ReturnsError] | Valid request, repo returns error | Wrapped error propagated | Error handling |
| 5 | [e.g., Create_AuditLogCreated] | Valid request | Audit log service called with correct params | Audit logging |
| 6 | [e.g., GetByID_DifferentCampus_ReturnsNotFound] | Valid ID, different campus_id | Not found error | Campus isolation |
| 7 | [e.g., List_ReturnsWithPagination] | Filter with page/per_page | Correct slice + total count | Pagination |
| 8 | [e.g., Update_InactiveRecord_ReturnsError] | ID of soft-deleted record | Not found error | Soft delete |

### Template

```go
func TestPersonService_Create(t *testing.T) {
    tests := []struct {
        name        string
        req         domain.CreatePersonRequest
        campusID    uuid.UUID
        userID      uuid.UUID
        mockSetup   func(repo *mocks.PersonRepository)
        wantErr     bool
        errContains string
    }{
        {
            name:     "valid request creates person",
            req:      domain.CreatePersonRequest{FullName: "Maria Santos", DocumentType: "CPF"},
            campusID: uuid.New(),
            userID:   uuid.New(),
            mockSetup: func(repo *mocks.PersonRepository) {
                repo.On("Create", mock.Anything, mock.Anything).Return(nil)
            },
            wantErr: false,
        },
        {
            name:        "missing full_name returns validation error",
            req:         domain.CreatePersonRequest{DocumentType: "CPF"},
            campusID:    uuid.New(),
            userID:      uuid.New(),
            wantErr:     true,
            errContains: "validation failed",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := new(mocks.PersonRepository)
            if tt.mockSetup != nil {
                tt.mockSetup(repo)
            }
            svc := service.NewPersonService(repo, auditSvc)

            result, err := svc.Create(context.Background(), tt.req, tt.campusID, tt.userID)

            if tt.wantErr {
                assert.Error(t, err)
                if tt.errContains != "" {
                    assert.Contains(t, err.Error(), tt.errContains)
                }
                assert.Nil(t, result)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, result)
                assert.Equal(t, tt.req.FullName, result.FullName)
                assert.Equal(t, tt.campusID, result.CampusID)
            }
        })
    }
}
```

---

## Integration Tests: Repository Layer

File: `backend/internal/repository/[entity]_repository_test.go`

Run against a real PostgreSQL test database. Use test fixtures for setup/teardown.

### Test Cases

| # | Test Name | Setup | Action | Expected | Validates |
|---|-----------|-------|--------|----------|-----------|
| 1 | Create_InsertsRecord | Clean table | Create person | Row exists in DB | Basic insert |
| 2 | GetByID_CorrectCampus_ReturnsRecord | Insert person in campus A | GetByID with campus A | Person returned | Read |
| 3 | GetByID_WrongCampus_ReturnsNil | Insert person in campus A | GetByID with campus B | Nil result, no error | Campus isolation |
| 4 | List_FiltersByCampus | Insert 3 in campus A, 2 in campus B | List with campus A | 3 results | Campus-scoped list |
| 5 | List_Pagination | Insert 25 records | List page=2, per_page=10 | 10 results, total=25 | Pagination |
| 6 | Update_ModifiesFields | Insert person | Update full_name | New name persisted | Update |
| 7 | Update_WrongCampus_NoEffect | Insert in campus A | Update with campus B | 0 rows affected | Campus isolation on write |
| 8 | SoftDelete_SetsInactive | Insert person | SoftDelete | is_active = false | Soft delete |
| 9 | GetByID_AfterSoftDelete_ReturnsNil | Soft-delete person | GetByID | Nil result | Soft delete visibility |
| 10 | Create_DuplicateID_ReturnsError | Insert person | Insert same ID | Constraint violation error | Uniqueness |

---

## Frontend Tests: Hooks

File: `frontend/src/hooks/__tests__/use[Entity].test.ts`

### Test Cases

| # | Test Name | Setup | Expected | Validates |
|---|-----------|-------|----------|-----------|
| 1 | Fetches data on mount | Mock API returns list | Data populated, loading=false | Initial fetch |
| 2 | Handles API error | Mock API rejects | Error message set, data empty | Error handling |
| 3 | Refetch updates data | Mock API returns new data | Data updated after refetch | Manual refresh |
| 4 | Pagination works | Mock API returns paginated response | Pagination meta correct | Pagination state |

### Template

```typescript
import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';

describe('usePersons', () => {
  it('fetches persons on mount', async () => {
    vi.mocked(personApi.list).mockResolvedValue({
      data: [{ id: '1', full_name: 'Maria Santos', is_active: true }],
      pagination: { page: 1, per_page: 20, total: 1, total_pages: 1 },
    });

    const { result } = renderHook(() => usePersons());

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.data).toHaveLength(1);
    expect(result.current.error).toBeNull();
  });
});
```

---

## Frontend Tests: Forms and Components

File: `frontend/src/components/[entity]/__tests__/[Component].test.tsx`

### Form Validation Tests

| # | Test Name | Action | Expected | Validates |
|---|-----------|--------|----------|-----------|
| 1 | Shows error for empty required fields | Submit empty form | Error messages shown for required fields | Required field validation |
| 2 | Shows error for invalid email | Enter invalid email, submit | "Email invalido" error shown | Email format validation |
| 3 | Accepts valid input | Fill all required fields, submit | onSubmit called with correct data | Successful submission |
| 4 | Shows loading state during submit | Submit form | Button shows "Salvando..." and is disabled | Loading state |
| 5 | Populates initial data for edit | Pass initialData prop | Form fields pre-filled | Edit mode |

### Template

```typescript
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import { PersonForm } from '../PersonForm';

describe('PersonForm', () => {
  it('shows validation error for empty full_name', async () => {
    const onSubmit = vi.fn();
    render(<PersonForm onSubmit={onSubmit} />);

    await userEvent.click(screen.getByRole('button', { name: /salvar/i }));

    expect(await screen.findByText(/nome completo e obrigatorio/i)).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('submits with valid data', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    render(<PersonForm onSubmit={onSubmit} />);

    await userEvent.type(screen.getByLabelText(/nome completo/i), 'Maria Santos');
    await userEvent.selectOptions(screen.getByLabelText(/tipo de documento/i), 'CPF');
    await userEvent.click(screen.getByRole('button', { name: /salvar/i }));

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith(
        expect.objectContaining({ full_name: 'Maria Santos', document_type: 'CPF' })
      );
    });
  });
});
```

---

## Offline Tests

### Automated (Vitest with Dexie)

| # | Test Name | Action | Expected |
|---|-----------|--------|----------|
| 1 | Creates record in IndexedDB | Call createPersonOffline() | Record in Dexie with syncStatus='pending' |
| 2 | Adds sync queue entry | Call createPersonOffline() | SyncQueue entry with entityType and syncId |
| 3 | Encrypts sensitive fields | Create with document_number | encryptedFields is not plaintext |
| 4 | Lists pending records | Create 3 offline records | All 3 returned with syncStatus='pending' |

### Manual Test Procedure

| # | Step | Expected Result |
|---|------|-----------------|
| 1 | Log in and navigate to the feature page | Page loads with server data |
| 2 | Open DevTools > Network > set Offline | "Offline" indicator appears |
| 3 | Create a new record | Record appears in local list with "Pendente" badge |
| 4 | Check IndexedDB (DevTools > Application > IndexedDB > chesed) | Record stored, PII encrypted |
| 5 | Set Network back to Online | Sync starts automatically |
| 6 | Wait for sync to complete | Badge changes to "Sincronizado" |
| 7 | Refresh page | Record persists with server-generated timestamps |
| 8 | Check server (API or database) | Record exists on server |

---

## Edge Cases

| # | Scenario | Expected Behavior | Test Type |
|---|----------|-------------------|-----------|
| 1 | Missing required fields in request | 400 with VALIDATION_ERROR | Service unit test |
| 2 | Duplicate document_number + document_type | 409 with CONFLICT (or business rule) | Service unit test |
| 3 | Unauthorized user (no token) | 401 UNAUTHORIZED | Handler/integration test |
| 4 | Wrong role (e.g., VOLUNTEER tries admin action) | 403 FORBIDDEN | Handler/integration test |
| 5 | Cross-campus access (user from campus A queries campus B data) | 404 NOT_FOUND (data invisible) | Repository integration test |
| 6 | Soft-deleted record accessed by ID | 404 NOT_FOUND | Repository integration test |
| 7 | Invalid UUID in path parameter | 400 BAD_REQUEST | Handler test |
| 8 | Very long string input (exceeds VARCHAR limit) | 400 VALIDATION_ERROR | Service unit test |
| 9 | Concurrent update (optimistic locking) | Last-write-wins in MVP | Manual test |
| 10 | Offline creation with same sync_id pushed twice | Server returns idempotent result | Sync integration test |

---

## Test Results

| Category | Total | Passed | Failed | Skipped |
|----------|-------|--------|--------|---------|
| Service unit tests | | | | |
| Repository integration tests | | | | |
| Frontend hook tests | | | | |
| Frontend form tests | | | | |
| Offline tests (automated) | | | | |
| Offline tests (manual) | | | | |
| Edge case tests | | | | |

**Overall verdict**: PASS / FAIL

**Notes**: [Any observations, known issues, or deferred test coverage]
