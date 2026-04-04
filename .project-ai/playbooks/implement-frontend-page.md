# Playbook: Implement Frontend Page

## Purpose

End-to-end guide for implementing a new page in the Chesed React frontend. Covers TypeScript types through offline support and testing.

---

## Prerequisites

- The backing API endpoint(s) are documented in `docs/11-api-design.md`
- The feature is confirmed in Phase 1 scope (`docs/07-mvp-scope.md`)
- The feature's RBAC requirements are known from `docs/16-iam-and-access-control.md`

---

## Steps

### Step 1: Review API Contract

Open `docs/11-api-design.md` and locate all endpoints the page will consume. Note:

- Request/response schemas (field names, types, required/optional)
- Pagination format (`{ data: [], pagination: { page, per_page, total, total_pages } }`)
- Error response format (`{ error: { code, message } }`)
- Role requirements (determines route guard)
- Whether the endpoint supports `sync_id` for offline creation

### Step 2: Create TypeScript Interfaces

File: `frontend/src/types/<entity>.ts`

- Define interfaces matching the API response schemas exactly
- Define separate request interfaces for POST/PUT/PATCH payloads
- Use strict types (never `any`)
- Export all interfaces

```typescript
// frontend/src/types/person.ts

export interface Person {
  id: string;
  full_name: string;
  birth_date?: string;
  document_type: 'CPF' | 'SSN' | 'EU_ID' | 'PASSPORT' | 'OTHER';
  document_number?: string;
  gender?: 'M' | 'F' | 'OTHER' | 'PREFER_NOT_TO_SAY';
  email?: string;
  phone?: string;
  campus_id: string;
  roles?: string[];
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreatePersonRequest {
  full_name: string;
  birth_date?: string;
  document_type: string;
  document_number?: string;
  gender?: string;
  email?: string;
  phone?: string;
  referral_source?: string;
  address?: CreateAddressRequest;
  sync_id?: string;
}

export interface PersonListResponse {
  data: Person[];
  pagination: PaginationMeta;
}

export interface PaginationMeta {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
}
```

### Step 3: Create API Client

File: `frontend/src/api/<entity>Api.ts`

One file per domain entity. All API calls go through a shared HTTP client that attaches the Keycloak bearer token automatically.

```typescript
// frontend/src/api/personApi.ts

import { httpClient } from './httpClient';
import type { Person, CreatePersonRequest, PersonListResponse } from '../types/person';

const BASE_PATH = '/api/v1/persons';

export const personApi = {
  create: (data: CreatePersonRequest): Promise<Person> =>
    httpClient.post(BASE_PATH, data),

  list: (params: { q?: string; page?: number; per_page?: number }): Promise<PersonListResponse> =>
    httpClient.get(BASE_PATH, { params }),

  getById: (id: string): Promise<Person> =>
    httpClient.get(`${BASE_PATH}/${id}`),

  update: (id: string, data: Partial<CreatePersonRequest>): Promise<Person> =>
    httpClient.put(`${BASE_PATH}/${id}`, data),

  checkDuplicate: (documentNumber: string, documentType: string) =>
    httpClient.get(`${BASE_PATH}/check-duplicate`, {
      params: { document_number: documentNumber, document_type: documentType },
    }),
};
```

The shared HTTP client (`frontend/src/api/httpClient.ts`) must:
- Attach `Authorization: Bearer <token>` from keycloak-js on every request
- Handle 401 responses by triggering token refresh
- Handle network errors gracefully for offline detection

### Step 4: Create Custom Hooks

File: `frontend/src/hooks/use<Entity>.ts`

- One hook per data-fetching concern
- Handle loading, error, and data states
- Integrate with offline storage if applicable
- Return typed data (never `any`)

```typescript
// frontend/src/hooks/usePersons.ts

import { useState, useEffect, useCallback } from 'react';
import { personApi } from '../api/personApi';
import type { Person, PersonListResponse } from '../types/person';

export function usePersons(initialPage = 1, perPage = 20) {
  const [data, setData] = useState<Person[]>([]);
  const [pagination, setPagination] = useState({ page: initialPage, per_page: perPage, total: 0, total_pages: 0 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchPersons = useCallback(async (page: number, query?: string) => {
    setLoading(true);
    setError(null);
    try {
      const response = await personApi.list({ page, per_page: perPage, q: query });
      setData(response.data);
      setPagination(response.pagination);
    } catch (err) {
      setError('Erro ao carregar pessoas. Tente novamente.');
      // Fallback to offline data if network error
    } finally {
      setLoading(false);
    }
  }, [perPage]);

  useEffect(() => {
    fetchPersons(initialPage);
  }, [fetchPersons, initialPage]);

  return { data, pagination, loading, error, refetch: fetchPersons };
}
```

### Step 5: Build Reusable Components

Directory: `frontend/src/components/<entity>/`

Design rules:
- **Mobile-first**: Design for 320px width first, then scale up
- **Tailwind CSS only**: No CSS modules, no styled-components, no inline style objects
- **No direct API calls**: Components receive data and callbacks via props
- **Portuguese (Brazilian) for all user-visible text**: labels, placeholders, error messages, buttons

```typescript
// frontend/src/components/person/PersonCard.tsx

interface PersonCardProps {
  person: Person;
  onSelect: (id: string) => void;
}

export function PersonCard({ person, onSelect }: PersonCardProps) {
  return (
    <div
      className="rounded-lg border border-gray-200 p-4 shadow-sm hover:shadow-md transition-shadow cursor-pointer"
      onClick={() => onSelect(person.id)}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => e.key === 'Enter' && onSelect(person.id)}
    >
      <h3 className="text-base font-semibold text-gray-900 truncate">
        {person.full_name}
      </h3>
      <p className="mt-1 text-sm text-gray-500">
        {person.document_type}: {person.document_number ?? 'Nao informado'}
      </p>
      {person.phone && (
        <p className="mt-1 text-sm text-gray-500">{person.phone}</p>
      )}
    </div>
  );
}
```

For forms, use React Hook Form + Zod:

```typescript
// frontend/src/components/person/PersonForm.tsx

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';

const personSchema = z.object({
  full_name: z.string().min(1, 'Nome completo e obrigatorio').max(200),
  document_type: z.enum(['CPF', 'SSN', 'EU_ID', 'PASSPORT', 'OTHER']),
  document_number: z.string().optional(),
  birth_date: z.string().optional(),
  gender: z.enum(['M', 'F', 'OTHER', 'PREFER_NOT_TO_SAY']).optional(),
  email: z.string().email('Email invalido').optional().or(z.literal('')),
  phone: z.string().optional(),
});

type PersonFormData = z.infer<typeof personSchema>;

interface PersonFormProps {
  onSubmit: (data: PersonFormData) => Promise<void>;
  initialData?: Partial<PersonFormData>;
  isLoading?: boolean;
}

export function PersonForm({ onSubmit, initialData, isLoading }: PersonFormProps) {
  const { register, handleSubmit, formState: { errors } } = useForm<PersonFormData>({
    resolver: zodResolver(personSchema),
    defaultValues: initialData,
  });

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div>
        <label htmlFor="full_name" className="block text-sm font-medium text-gray-700">
          Nome completo *
        </label>
        <input
          id="full_name"
          type="text"
          {...register('full_name')}
          className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 text-sm"
        />
        {errors.full_name && (
          <p className="mt-1 text-sm text-red-600">{errors.full_name.message}</p>
        )}
      </div>
      {/* Additional fields... */}
      <button
        type="submit"
        disabled={isLoading}
        className="w-full rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
      >
        {isLoading ? 'Salvando...' : 'Salvar'}
      </button>
    </form>
  );
}
```

### Step 6: Create Page Component

File: `frontend/src/pages/<Entity>/<PageName>.tsx`

Pages are route-level components that compose hooks + components. They orchestrate data fetching, handle navigation, and manage page-level state.

```typescript
// frontend/src/pages/Person/PersonListPage.tsx

import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { usePersons } from '../../hooks/usePersons';
import { PersonCard } from '../../components/person/PersonCard';
import { SearchInput } from '../../components/common/SearchInput';
import { Pagination } from '../../components/common/Pagination';
import { SyncStatusBadge } from '../../components/common/SyncStatusBadge';

export function PersonListPage() {
  const navigate = useNavigate();
  const [searchQuery, setSearchQuery] = useState('');
  const { data, pagination, loading, error, refetch } = usePersons();

  const handleSearch = (query: string) => {
    setSearchQuery(query);
    refetch(1, query);
  };

  return (
    <div className="px-4 py-6 sm:px-6 lg:px-8">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold text-gray-900">Pessoas</h1>
        <button
          onClick={() => navigate('/persons/new')}
          className="rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white"
        >
          Nova Pessoa
        </button>
      </div>

      <div className="mt-4">
        <SearchInput
          placeholder="Buscar por nome ou CPF..."
          onSearch={handleSearch}
        />
      </div>

      {error && <p className="mt-4 text-sm text-red-600">{error}</p>}

      <div className="mt-4 grid gap-3">
        {loading ? (
          <p className="text-sm text-gray-500">Carregando...</p>
        ) : (
          data.map((person) => (
            <PersonCard
              key={person.id}
              person={person}
              onSelect={(id) => navigate(`/persons/${id}`)}
            />
          ))
        )}
      </div>

      <Pagination
        pagination={pagination}
        onPageChange={(page) => refetch(page, searchQuery)}
      />
    </div>
  );
}
```

### Step 7: Add Route with Auth Guard

File: `frontend/src/App.tsx`

Add the route wrapped in the appropriate auth guard. The guard checks:
1. User is authenticated via Keycloak
2. User has the required role (from token claims)

```typescript
import { Route, Routes } from 'react-router-dom';
import { RequireAuth } from './components/auth/RequireAuth';
import { RequireRole } from './components/auth/RequireRole';
import { PersonListPage } from './pages/Person/PersonListPage';
import { PersonCreatePage } from './pages/Person/PersonCreatePage';

function App() {
  return (
    <Routes>
      <Route element={<RequireAuth />}>
        {/* All authenticated users */}
        <Route path="/persons" element={<PersonListPage />} />
        <Route path="/persons/new" element={<PersonCreatePage />} />

        {/* Secretary+ only */}
        <Route element={<RequireRole roles={['SECRETARY', 'PROFESSIONAL', 'COORDINATOR', 'ADMIN']} />}>
          <Route path="/persons/:id/edit" element={<PersonEditPage />} />
        </Route>
      </Route>
    </Routes>
  );
}
```

### Step 8: Add Offline Support (If Applicable)

Reference: `docs/12-offline-sync-strategy.md`

Determine offline classification:
- **Fully offline-capable**: person creation, triage creation (must work without network)
- **Read-only offline**: person list, attendance list (cached data viewable, no creation)
- **Online-only**: reports, user management, audit logs

If fully offline-capable:

**8a.** Add Dexie table in `frontend/src/offline/db.ts`:

```typescript
import Dexie, { Table } from 'dexie';

interface LocalPerson {
  id: string;
  syncId: string;
  data: PersonData;
  syncStatus: 'pending' | 'synced' | 'conflict';
  localCreatedAt: string;
  serverUpdatedAt?: string;
}

class ChesedDatabase extends Dexie {
  persons!: Table<LocalPerson, string>;
  syncQueue!: Table<SyncQueueItem, number>;

  constructor() {
    super('chesed');
    this.version(1).stores({
      persons: 'id, syncStatus, localCreatedAt',
      syncQueue: '++id, entityType, syncStatus',
    });
  }
}
```

**8b.** Generate UUID client-side for offline records:

```typescript
const offlineId = crypto.randomUUID();
```

**8c.** Add sync queue entry on local save:

```typescript
await db.syncQueue.add({
  entityType: 'person',
  action: 'create',
  data: personData,
  syncId: offlineId,
  timestamp: new Date().toISOString(),
  syncStatus: 'pending',
});
```

**8d.** Show sync status indicator using the `SyncStatusBadge` component.

**8e.** Encrypt sensitive fields (document_number, health data) using Web Crypto API (AES-256-GCM) before storing in IndexedDB. Reference: `docs/12-offline-sync-strategy.md`.

### Step 9: User-Visible Text Language

All user-visible text must be in **Portuguese (Brazilian)**:
- Labels: "Nome completo", "Data de nascimento", "Tipo de documento"
- Buttons: "Salvar", "Cancelar", "Nova Pessoa", "Buscar"
- Error messages: "Campo obrigatorio", "Email invalido", "Erro ao salvar"
- Empty states: "Nenhuma pessoa encontrada"
- Loading states: "Carregando..."

All code (variable names, comments, function names) must be in **English**.

### Step 10: Write Tests

**Hook tests** (file: `frontend/src/hooks/__tests__/use<Entity>.test.ts`):

```typescript
import { renderHook, waitFor } from '@testing-library/react';
import { usePersons } from '../usePersons';
import { personApi } from '../../api/personApi';

vi.mock('../../api/personApi');

describe('usePersons', () => {
  it('fetches persons on mount', async () => {
    const mockResponse = { data: [{ id: '1', full_name: 'Test' }], pagination: { page: 1, per_page: 20, total: 1, total_pages: 1 } };
    vi.mocked(personApi.list).mockResolvedValue(mockResponse);

    const { result } = renderHook(() => usePersons());

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.data).toHaveLength(1);
  });
});
```

**Form/component tests** (file: `frontend/src/components/<entity>/__tests__/<Component>.test.tsx`):
- Use React Testing Library
- Test form validation (submit with empty required fields)
- Test form submission (fill fields, submit, verify callback)
- Test accessibility (labels, roles, keyboard navigation)

### Step 11: Run Pre-Review Checks

```bash
# Run TypeScript type check
cd frontend && npx tsc --noEmit

# Run linter
cd frontend && npx eslint src/

# Run tests
cd frontend && npx vitest run

# Test at 320px width
# Open browser DevTools > Toggle device toolbar > Select 320px width
# Navigate to the new page and verify layout
```

---

## Checklist

- [ ] TypeScript interfaces match API contract in `docs/11-api-design.md`
- [ ] API client created (one file per domain)
- [ ] Custom hooks handle loading, error, and data states
- [ ] Components use Tailwind CSS, mobile-first (test at 320px)
- [ ] Page composes hooks + components correctly
- [ ] Route added with auth guard and role check
- [ ] Offline support added if feature is offline-capable
- [ ] All user-visible text in Portuguese (Brazilian)
- [ ] All code in English
- [ ] Hook tests written with Vitest
- [ ] Component/form tests written with React Testing Library
- [ ] No `any` types in TypeScript
- [ ] No lint warnings
- [ ] All existing tests pass
